// Package qris membaca payload QR pembayaran berstandar EMVCo Merchant-Presented Mode (QRIS).
//
// Urutan validasi mengikuti alur decode issuer pada umumnya (lihat MobileDecode di
// core-payment-ultimate): bersihkan baris baru → cek bentuk EMV → urai TLV → cek Payload Format
// Indicator & CRC → ambil Merchant Account Information (tag 26–51; tag 51 = NMID QRIS) → statis
// atau dinamis (tag 01) → nominal, tip/biaya layanan, mata uang, data tambahan.
//
// Paket ini hanya MEMBACA QR. Tidak ada panggilan ke switching/acquirer dan tidak ada uang yang
// berpindah; pembayaran di demo disimulasikan terpisah.
package qris

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	ErrNotEMV       = errors.New("bukan QR pembayaran EMVCo/QRIS")
	ErrMalformed    = errors.New("format QR rusak")
	ErrCRC          = errors.New("CRC QR tidak cocok")
	ErrNoMerchant   = errors.New("QR tidak berisi informasi merchant (tag 26–51)")
	ErrPayloadFmt   = errors.New("payload format indicator (tag 00) harus 01")
	isEMVRe         = regexp.MustCompile(`^(\d{2}\d{2}){2,}`)
	lineBreakRemove = strings.NewReplacer("\r", "", "\n", "")
)

// TLV satu data object EMV: tag 2 digit, panjang 2 digit, nilai.
type TLV struct {
	Tag   string
	Value string
}

// Account: Merchant Account Information (tag 26–51).
type Account struct {
	Tag      string // "26".."51"
	GUID     string // sub-tag 00, mis. "ID.CO.QRIS.WWW", "COM.GO-JEK.WWW"
	PAN      string // sub-tag 01: Merchant PAN (diawali NNS 8 digit)
	ID       string // sub-tag 02: merchant ID / NMID (tag 51)
	Criteria string // sub-tag 03: UMI/UKE/UME/UBE/URE
}

type QR struct {
	Payload      string
	TLVs         []TLV
	Dynamic      bool // tag 01 = "12"
	Accounts     []Account
	MCC          string
	Currency     string // ISO 4217 numerik, "360" = IDR
	Amount       int64  // tag 54 (rupiah, tanpa desimal); 0 = diisi pembayar (statis)
	TipIndicator string // tag 55: "" | 01 prompt | 02 fixed | 03 persen
	TipFixed     int64  // tag 56
	TipPercent   float64
	Country      string
	MerchantName string
	MerchantCity string
	PostalCode   string
	BillNumber   string // tag 62.01
	StoreLabel   string // tag 62.03
	Reference    string // tag 62.05
	TerminalID   string // tag 62.07
	CRC          string
}

// Clean menghapus baris baru (beberapa pemindai/cetakan menyisipkan \n).
func Clean(s string) string { return strings.TrimSpace(lineBreakRemove.Replace(s)) }

// IsEMV: bentuk awal payload EMV (pasangan tag+panjang dua digit).
func IsEMV(s string) bool { return isEMVRe.MatchString(s) }

// ParseTLV mengurai deret TLV EMV. Panjang dihitung per karakter (rune).
func ParseTLV(s string) ([]TLV, error) {
	r := []rune(s)
	var out []TLV
	for i := 0; i < len(r); {
		if i+4 > len(r) {
			return nil, fmt.Errorf("%w: sisa data tidak lengkap di posisi %d", ErrMalformed, i)
		}
		tag := string(r[i : i+2])
		n, err := strconv.Atoi(string(r[i+2 : i+4]))
		if err != nil || !isDigits(tag) {
			return nil, fmt.Errorf("%w: tag/panjang tidak valid di posisi %d", ErrMalformed, i)
		}
		if i+4+n > len(r) {
			return nil, fmt.Errorf("%w: nilai tag %s melebihi panjang data", ErrMalformed, tag)
		}
		out = append(out, TLV{Tag: tag, Value: string(r[i+4 : i+4+n])})
		i += 4 + n
	}
	return out, nil
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return s != ""
}

// CRC16 CCITT-FALSE (poly 0x1021, init 0xFFFF), seperti disyaratkan EMVCo untuk tag 63.
func CRC16(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// WithCRC menambahkan "6304"+CRC ke payload tanpa CRC (dipakai untuk membuat QR contoh & tes).
func WithCRC(payloadNoCRC string) string {
	s := payloadNoCRC + "6304"
	return s + fmt.Sprintf("%04X", CRC16([]byte(s)))
}

// Decode memvalidasi dan mengurai payload QRIS/EMV MPM.
func Decode(raw string) (*QR, error) {
	s := Clean(raw)
	if !IsEMV(s) {
		return nil, ErrNotEMV
	}
	tlvs, err := ParseTLV(s)
	if err != nil {
		return nil, err
	}
	q := &QR{Payload: s, TLVs: tlvs}
	if len(tlvs) == 0 || tlvs[0].Tag != "00" || tlvs[0].Value != "01" {
		return nil, ErrPayloadFmt
	}
	last := tlvs[len(tlvs)-1]
	if last.Tag != "63" || len(last.Value) != 4 {
		return nil, fmt.Errorf("%w: tag 63 harus menjadi data terakhir", ErrCRC)
	}
	body := s[:len(s)-4]
	if want := fmt.Sprintf("%04X", CRC16([]byte(body))); !strings.EqualFold(want, last.Value) {
		return nil, fmt.Errorf("%w (tertulis %s, dihitung %s)", ErrCRC, strings.ToUpper(last.Value), want)
	}
	q.CRC = strings.ToUpper(last.Value)

	for _, t := range tlvs {
		n, _ := strconv.Atoi(t.Tag)
		switch {
		case t.Tag == "01":
			q.Dynamic = t.Value == "12"
		case n >= 26 && n <= 51:
			sub, err := ParseTLV(t.Value)
			if err != nil {
				return nil, fmt.Errorf("merchant account tag %s: %w", t.Tag, err)
			}
			a := Account{Tag: t.Tag}
			for _, st := range sub {
				switch st.Tag {
				case "00":
					a.GUID = st.Value
				case "01":
					a.PAN = st.Value
				case "02":
					a.ID = st.Value
				case "03":
					a.Criteria = st.Value
				}
			}
			q.Accounts = append(q.Accounts, a)
		case t.Tag == "52":
			q.MCC = t.Value
		case t.Tag == "53":
			q.Currency = t.Value
		case t.Tag == "54":
			q.Amount = parseAmount(t.Value)
		case t.Tag == "55":
			q.TipIndicator = t.Value
		case t.Tag == "56":
			q.TipFixed = parseAmount(t.Value)
		case t.Tag == "57":
			q.TipPercent, _ = strconv.ParseFloat(strings.Replace(t.Value, ",", ".", 1), 64)
		case t.Tag == "58":
			q.Country = t.Value
		case t.Tag == "59":
			q.MerchantName = strings.TrimSpace(t.Value)
		case t.Tag == "60":
			q.MerchantCity = strings.TrimSpace(t.Value)
		case t.Tag == "61":
			q.PostalCode = t.Value
		case t.Tag == "62":
			if sub, err := ParseTLV(t.Value); err == nil {
				for _, st := range sub {
					switch st.Tag {
					case "01":
						q.BillNumber = st.Value
					case "03":
						q.StoreLabel = st.Value
					case "05":
						q.Reference = st.Value
					case "07":
						q.TerminalID = st.Value
					}
				}
			}
		}
	}
	if len(q.Accounts) == 0 {
		return nil, ErrNoMerchant
	}
	sort.SliceStable(q.Accounts, func(i, j int) bool { return q.Accounts[i].Tag < q.Accounts[j].Tag })
	return q, nil
}

// parseAmount: "25000", "25000.00" -> 25000 (rupiah tidak punya sen dalam praktik QRIS).
func parseAmount(v string) int64 {
	f, err := strconv.ParseFloat(strings.Replace(v, ",", ".", 1), 64)
	if err != nil || f < 0 {
		return 0
	}
	return int64(f + 0.5)
}

// ---- turunan untuk tampilan ----

// Acquirer: akun pertama (tag 26..) adalah penyelenggara (acquirer) QR.
func (q *QR) Acquirer() Account { return q.Accounts[0] }

// NMID: National Merchant ID dari tag 51 (QRIS), bila ada.
func (q *QR) NMID() string {
	for _, a := range q.Accounts {
		if a.Tag == "51" {
			return a.ID
		}
	}
	return ""
}

func (q *QR) Criteria() string {
	for _, a := range q.Accounts {
		if a.Tag == "51" && a.Criteria != "" {
			return a.Criteria
		}
	}
	return q.Acquirer().Criteria
}

func (q *QR) IsIDR() bool { return q.Currency == "" || q.Currency == "360" }

func (q *QR) IsQRIS() bool {
	for _, a := range q.Accounts {
		if strings.Contains(strings.ToUpper(a.GUID), "QRIS") {
			return true
		}
	}
	return false
}

// Fee: biaya layanan/tip untuk nominal tertentu (prompt = diisi pembayar, dikembalikan 0).
func (q *QR) Fee(amount int64) int64 {
	switch q.TipIndicator {
	case "02":
		return q.TipFixed
	case "03":
		return int64(float64(amount)*q.TipPercent/100 + 0.5)
	}
	return 0
}

// AcquirerName: nama penyelenggara dari GUID (reverse domain) atau NNS di PAN.
func (q *QR) AcquirerName() string {
	a := q.Acquirer()
	g := strings.ToUpper(a.GUID)
	for _, k := range guidNames {
		if strings.Contains(g, k[0]) {
			return k[1]
		}
	}
	if len(a.PAN) >= 8 {
		if n, ok := nnsNames[a.PAN[:8]]; ok {
			return n
		}
	}
	// "ID.CO.BANKXYZ.WWW" -> "BANKXYZ"
	parts := strings.Split(g, ".")
	for _, p := range parts {
		if p != "ID" && p != "CO" && p != "COM" && p != "WWW" && p != "" {
			return p
		}
	}
	return "Penyelenggara QRIS"
}

// guidNames: potongan GUID tag 26 yang dikenal (urutan: yang lebih spesifik dulu).
var guidNames = [][2]string{
	{"BANKSYARIAHCONTOH", "Bank Syariah Contoh"}, {"GO-JEK", "GoPay"}, {"GOPAY", "GoPay"}, {"DANA", "DANA"}, {"SHOPEE", "ShopeePay"}, {"OVO", "OVO"},
	{"LINKAJA", "LinkAja"}, {"BANKMANDIRI", "Bank Mandiri"}, {"MANDIRI", "Bank Mandiri"}, {"BCA", "BCA"},
	{"BRI", "BRI"}, {"BNI", "BNI"}, {"BSI", "Bank Syariah Indonesia"}, {"SYARIAHINDONESIA", "Bank Syariah Indonesia"},
	{"CIMB", "CIMB Niaga"}, {"PERMATA", "Bank Permata"}, {"DANAMON", "Bank Danamon"}, {"BTN", "BTN"},
	{"MUAMALAT", "Bank Muamalat"}, {"NOBU", "Nobu Bank"}, {"MAYBANK", "Maybank Indonesia"}, {"MEGA", "Bank Mega"},
	{"ASTRAPAY", "AstraPay"}, {"DOKU", "DOKU"}, {"XENDIT", "Xendit"}, {"MIDTRANS", "Midtrans"}, {"BLU", "blu by BCA Digital"},
	{"JAGO", "Bank Jago"}, {"SEABANK", "SeaBank"}, {"MOTIONPAY", "MotionPay"}, {"ESPAY", "Espay"}, {"FLIP", "Flip"},
}

// nnsNames: kode NNS (8 digit awal PAN) bank yang umum.
var nnsNames = map[string]string{
	"93600002": "BRI", "93600008": "Bank Mandiri", "93600009": "BNI", "93600011": "Bank Danamon", "93600013": "Bank Permata",
	"93600014": "BCA", "93600016": "Maybank Indonesia", "93600022": "CIMB Niaga", "93600147": "Bank Muamalat",
	"93600200": "BTN", "93600426": "Bank Mega", "93600451": "Bank Syariah Indonesia", "93600503": "Nobu Bank",
}

// CriteriaLabel: kriteria merchant QRIS (tag 51.03).
func CriteriaLabel(c string) string {
	return map[string]string{"UMI": "Usaha mikro", "UKE": "Usaha kecil", "UME": "Usaha menengah", "UBE": "Usaha besar", "URE": "Usaha reguler"}[c]
}

// MCCLabel: kategori merchant (ISO 18245) yang sering muncul.
func MCCLabel(mcc string) string {
	return map[string]string{
		"5411": "Supermarket", "5499": "Toko makanan", "5812": "Restoran", "5813": "Kafe & minuman", "5814": "Makanan cepat saji",
		"5311": "Toserba", "5331": "Toko serba ada", "5399": "Toko umum", "5912": "Apotek", "5999": "Toko ritel", "5541": "SPBU",
		"4111": "Transportasi", "4121": "Taksi & ojek", "7011": "Hotel", "4722": "Agen perjalanan", "8398": "Donasi & sosial",
		"8661": "Rumah ibadah", "5651": "Pakaian", "5732": "Elektronik", "7230": "Salon", "5942": "Toko buku", "0000": "",
	}[mcc]
}

// MaskPAN: "9360091430012345678" -> "936009••••5678".
func MaskPAN(p string) string {
	if len(p) <= 10 {
		return p
	}
	return p[:6] + "••••" + p[len(p)-4:]
}
