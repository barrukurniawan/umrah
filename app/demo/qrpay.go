package demo

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"umrah/app/qris"

	"gorm.io/gorm"
)

// Bayar QRIS (demo). Scan & decode QR sungguhan (app/qris); pembayarannya simulasi.
//
// Prinsip: umrohku tidak menyimpan saldo. Sumber dana adalah rekening jamaah di bank syariah mitra
// (contoh), dan transaksi QRIS diproses oleh mitra PJP berizin BI. Di demo, saldo rekening hanya
// angka contoh yang berkurang saat "membayar"; tidak ada dana yang sampai ke merchant.

const QRISLimit int64 = 10_000_000 // batas per transaksi QRIS (ketentuan umum BI)

// SARRateDemo: kurs contoh 1 SAR = Rp 4.300 untuk SIMULASI QR lintas negara.
// Bukan kurs resmi. Pembayaran sungguhan butuh mitra penerbit berizin BI yang terhubung ke switching
// lintas negara; umrohku sendiri tidak bisa (dan tidak boleh) memprosesnya.
const SARRateDemo int64 = 4300

// SARToIDR mengonversi halalah (1/100 SAR) ke rupiah dengan kurs contoh.
func SARToIDR(minor int64) int64 { return (minor*SARRateDemo + 50) / 100 }

// FormatMinor: 3500 -> "35", 3550 -> "35,50".
func FormatMinor(minor int64) string {
	if minor%100 == 0 {
		return Rp(minor / 100)
	}
	return fmt.Sprintf("%s,%02d", Rp(minor/100), minor%100)
}

type FundSource struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"uniqueIndex"`
	Bank      string `gorm:"size:80"`
	AccountNo string `gorm:"size:40"`
	Holder    string `gorm:"size:80"`
	Balance   int64
	UpdatedAt time.Time
}

func (f FundSource) Masked() string {
	n := strings.ReplaceAll(f.AccountNo, " ", "")
	if len(n) > 4 {
		return "••" + n[len(n)-4:]
	}
	return n
}

type QRPayment struct {
	ID           uint   `gorm:"primaryKey"`
	Code         string `gorm:"size:20;uniqueIndex"`
	UserID       uint   `gorm:"index"`
	MerchantName string `gorm:"size:80"`
	MerchantCity string `gorm:"size:60"`
	NMID         string `gorm:"size:40"`
	Acquirer     string `gorm:"size:60"`
	MCC          string `gorm:"size:4"`
	QRType       string `gorm:"size:10"` // STATIC | DYNAMIC
	Amount       int64
	Fee          int64
	Total        int64
	Source       string    `gorm:"size:80"`
	Reference    string    `gorm:"size:40"`
	Payload      string    `gorm:"type:text"`
	Status       string    `gorm:"size:20"` // berhasil
	Currency     string    `gorm:"size:3"`  // "" = IDR; "SAR" = simulasi lintas negara
	ForeignMinor int64     // nominal asli dalam satuan terkecil (halalah)
	Rate         int64     // kurs contoh rupiah per 1 unit mata uang asing
	CreatedAt    time.Time `gorm:"index"`
}

func init() { RegisterModels(&FundSource{}, &QRPayment{}) }

func (p QRPayment) Cross() bool     { return p.Currency != "" && p.Currency != "IDR" }
func (p QRPayment) Foreign() string { return FormatMinor(p.ForeignMinor) }

// Fund: sumber dana jamaah (dibuat otomatis untuk akun demo).
func Fund(userID uint, holder string) *FundSource {
	var f FundSource
	if DB.Where("user_id = ?", userID).Take(&f).Error == nil {
		return &f
	}
	f = FundSource{UserID: userID, Bank: "Bank Syariah Contoh", AccountNo: fmt.Sprintf("7102 %04d 4821", userID%10000), Holder: holder, Balance: 3_250_000, UpdatedAt: time.Now()}
	DB.Create(&f)
	return &f
}

func TopUpFund(userID uint, amount int64) {
	DB.Model(&FundSource{}).Where("user_id = ?", userID).Updates(map[string]any{"balance": gorm.Expr("balance + ?", amount), "updated_at": time.Now()})
}

type PayInput struct {
	UserID       uint
	Holder       string
	QR           *qris.QR
	Amount       int64 // untuk QR statis; QR dinamis memakai nominal di QR
	ForeignMinor int64 // QR lintas negara statis: nominal dalam halalah yang diisi pembayar
	Fee          int64 // tip yang diisi pembayar (indikator 01)
	PIN          string
}

func PayQR(in PayInput) (*QRPayment, error) {
	q := in.QR
	if q.IsCrossBorder() && !q.IsSAR() {
		return nil, errors.New("simulasi lintas negara baru mendukung QR riyal Saudi (SAR)")
	}
	var amount, fee, foreign int64
	if q.IsSAR() {
		// Simulasi lintas negara: nominal riyal dikonversi ke rupiah dengan kurs contoh. Tip tidak didukung.
		foreign = in.ForeignMinor
		if q.Dynamic && q.AmountMinor > 0 {
			foreign = q.AmountMinor
		}
		if foreign < 1 {
			return nil, errors.New("isi nominal pembayaran dalam riyal")
		}
		amount = SARToIDR(foreign)
	} else {
		amount = in.Amount
		if q.Dynamic && q.Amount > 0 {
			amount = q.Amount
		}
		if amount < 1 {
			return nil, errors.New("isi nominal pembayaran")
		}
		fee = q.Fee(amount)
		if q.TipIndicator == "01" {
			fee = max(0, min(in.Fee, amount))
		}
	}
	total := amount + fee
	if total > QRISLimit {
		return nil, fmt.Errorf("melebihi batas QRIS Rp %s per transaksi", Rp(QRISLimit))
	}
	if len(in.PIN) != 6 || strings.Trim(in.PIN, "0123456789") != "" {
		return nil, errors.New("PIN harus 6 angka")
	}
	f := Fund(in.UserID, in.Holder)
	var p *QRPayment
	err := DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&FundSource{}).Where("id = ? AND balance >= ?", f.ID, total).
			Updates(map[string]any{"balance": gorm.Expr("balance - ?", total), "updated_at": time.Now()})
		if res.RowsAffected == 0 {
			return fmt.Errorf("saldo %s tidak cukup", f.Bank)
		}
		qrType := "STATIC"
		if q.Dynamic {
			qrType = "DYNAMIC"
		}
		name := q.MerchantName
		if name == "" {
			name = "Merchant QRIS"
		}
		cur, rate := "", int64(0)
		if q.IsSAR() {
			cur, rate = "SAR", SARRateDemo
		}
		p = &QRPayment{Code: "QR" + time.Now().In(WIB).Format("060102") + randCode(6), UserID: in.UserID, MerchantName: truncate(name, 80),
			MerchantCity: truncate(q.MerchantCity, 60), NMID: q.NMID(), Acquirer: q.AcquirerName(), MCC: q.MCC, QRType: qrType,
			Amount: amount, Fee: fee, Total: total, Source: f.Bank + " " + f.Masked(), Reference: truncate(q.Reference, 40),
			Payload: q.Payload, Status: "berhasil", Currency: cur, ForeignMinor: foreign, Rate: rate, CreatedAt: time.Now()}
		return tx.Create(p).Error
	})
	if err != nil {
		return nil, err
	}
	marketPaidByQR(q, p)
	if p.Cross() {
		Notify(UserAudience(in.UserID), "wa", in.Holder, "Simulasi bayar QR Saudi berhasil",
			fmt.Sprintf("SAR %s (Rp %s, kurs contoh Rp %s) ke %s (%s) dari %s. Ref %s. Simulasi demo, tidak ada dana yang berpindah ke merchant.",
				p.Foreign(), Rp(total), Rp(p.Rate), p.MerchantName, p.MerchantCity, p.Source, p.Code), "/riwayat/qris/"+p.Code)
		return p, nil
	}
	Notify(UserAudience(in.UserID), "wa", in.Holder, "Pembayaran QRIS berhasil",
		fmt.Sprintf("Rp %s ke %s (%s) berhasil dibayar dari %s. Ref %s. Simulasi demo, tidak ada dana yang berpindah ke merchant.",
			Rp(total), p.MerchantName, p.MerchantCity, p.Source, p.Code), "/riwayat/qris/"+p.Code)
	return p, nil
}

func QRPaymentFor(userID uint, code string) *QRPayment {
	var p QRPayment
	if DB.Where("user_id = ? AND code = ?", userID, code).Take(&p).Error != nil {
		return nil
	}
	return &p
}

// ---- Riwayat gabungan ----

type HistoryItem struct {
	At     time.Time
	Kind   string // qris | booking | oleh | voucher
	Title  string
	Sub    string
	Amount int64
	Out    bool // uang keluar
	Status string
	Link   string
}

func History(userID uint, kind string, limit int) []HistoryItem {
	var out []HistoryItem
	if kind == "" || kind == "qris" {
		var ps []QRPayment
		DB.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&ps)
		for _, p := range ps {
			sub := "QRIS · " + p.MerchantCity
			if p.Cross() {
				sub = fmt.Sprintf("QR Saudi (simulasi) · %s · SAR %s", p.MerchantCity, p.Foreign())
			}
			out = append(out, HistoryItem{At: p.CreatedAt, Kind: "qris", Title: p.MerchantName, Sub: sub, Amount: p.Total, Out: true, Status: p.Status, Link: "/riwayat/qris/" + p.Code})
		}
	}
	if kind == "" || kind == "umroh" {
		var bs []Booking
		DB.Where("user_id = ?", userID).Find(&bs)
		ids := map[uint]Booking{}
		var bid []uint
		for _, b := range bs {
			ids[b.ID] = b
			bid = append(bid, b.ID)
		}
		if len(bid) > 0 {
			var pays []Payment
			DB.Where("booking_id IN ? AND status = ?", bid, "berhasil").Find(&pays)
			for _, p := range pays {
				b := ids[p.BookingID]
				t := "DP umroh"
				if p.Kind == "pelunasan" {
					t = "Pelunasan umroh"
				}
				at := p.CreatedAt
				if p.PaidAt != nil {
					at = *p.PaidAt
				}
				out = append(out, HistoryItem{At: at, Kind: "umroh", Title: t + " · " + b.TravelName, Sub: b.PackageName, Amount: p.Amount, Out: true, Status: "berhasil", Link: fmt.Sprintf("/booking/%d", b.ID)})
			}
		}
	}
	if kind == "" || kind == "belanja" {
		var os []ShopOrder
		DB.Where("user_id = ? AND paid_at IS NOT NULL", userID).Find(&os)
		for _, o := range os {
			out = append(out, HistoryItem{At: *o.PaidAt, Kind: "belanja", Title: "Oleh-oleh " + o.Code, Sub: "Dikirim ke " + o.City, Amount: o.Total, Out: true, Status: o.Status, Link: "/tanah-suci/oleh-oleh/" + o.Code})
		}
		var ms []MarketOrder
		DB.Where("user_id = ? AND paid_at IS NOT NULL", userID).Find(&ms)
		for _, o := range ms {
			name := o.MerchantSlug
			if m := o.Merchant(); m != nil {
				name = m.Name
			}
			out = append(out, HistoryItem{At: *o.PaidAt, Kind: "belanja", Title: name, Sub: fmt.Sprintf("Ambil di toko · %d barang · SAR %d", o.ItemCount(), o.TotalSAR), Amount: o.TotalIDR, Out: true, Status: o.Status, Link: "/market/pesanan/" + o.Code})
		}
		var vs []Voucher
		DB.Where("user_id = ? AND paid_at IS NOT NULL", userID).Find(&vs)
		for _, v := range vs {
			out = append(out, HistoryItem{At: *v.PaidAt, Kind: "belanja", Title: "Voucher Tanah Suci", Sub: fmt.Sprintf("%d SAR · %s", v.ValueSAR, MaskVoucher(v.Code)), Amount: v.AmountIDR, Out: true, Status: v.Status, Link: "/tanah-suci/voucher/" + v.Code})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// SampleQR: QRIS contoh (merchant fiktif) untuk mencoba alur bayar tanpa kamera.
func SampleQR(dynamic bool) string {
	t := func(tag, v string) string { return fmt.Sprintf("%s%02d%s", tag, len([]rune(v)), v) }
	acq := t("00", "ID.CO.BANKSYARIAHCONTOH.WWW") + t("01", "936099990000012345") + t("02", "UMK000123") + t("03", "UMI")
	nmid := t("00", "ID.CO.QRIS.WWW") + t("02", "ID0000DEMO00001") + t("03", "UMI")
	body := t("00", "01") + t("01", "11") + t("26", acq) + t("51", nmid) + t("52", "5812") + t("53", "360")
	if dynamic {
		body = t("00", "01") + t("01", "12") + t("26", acq) + t("51", nmid) + t("52", "5812") + t("53", "360") + t("54", "35000")
	}
	body += t("58", "ID") + t("59", "Kedai Kurma Contoh") + t("60", "Bandung") + t("61", "40115") + t("62", t("07", "KASIR01"))
	return qris.WithCRC(body)
}

// SampleQRSaudi: QR merchant fiktif di Makkah (negara SA, mata uang 682) dalam format EMVCo MPM,
// untuk mencoba simulasi lintas negara. Bukan QR penyelenggara sungguhan.
func SampleQRSaudi(dynamic bool) string {
	t := func(tag, v string) string { return fmt.Sprintf("%s%02d%s", tag, len([]rune(v)), v) }
	acq := t("00", "SA.COM.PENYELENGGARACONTOH") + t("01", "682099990000054321") + t("02", "MKH000321")
	body := t("00", "01") + t("01", "11") + t("26", acq) + t("52", "5499") + t("53", "682")
	if dynamic {
		body = t("00", "01") + t("01", "12") + t("26", acq) + t("52", "5499") + t("53", "682") + t("54", "35.50")
	}
	body += t("58", "SA") + t("59", "Toko Kurma Contoh") + t("60", "Makkah") + t("62", t("07", "KASIR01"))
	return qris.WithCRC(body)
}

func seedQR(userID uint, holder string) {
	f := Fund(userID, holder)
	now := time.Now()
	samples := []struct {
		name, city, mcc string
		amt             int64
		ago             time.Duration
	}{{"Kedai Kurma Contoh", "Bandung", "5812", 35000, 26 * time.Hour}, {"Toko Perlengkapan Ihram Contoh", "Bandung", "5651", 185000, 74 * time.Hour}, {"Apotek Sehat Contoh", "Bandung", "5912", 62500, 140 * time.Hour}}
	for i, s := range samples {
		DB.Create(&QRPayment{Code: fmt.Sprintf("QR%s%s", now.Add(-s.ago).In(WIB).Format("060102"), randCode(6)), UserID: userID, MerchantName: s.name,
			MerchantCity: s.city, NMID: fmt.Sprintf("ID0000DEMO0000%d", i+1), Acquirer: "Bank Syariah Contoh", MCC: s.mcc, QRType: "STATIC",
			Amount: s.amt, Total: s.amt, Source: f.Bank + " " + f.Masked(), Status: "berhasil", CreatedAt: now.Add(-s.ago)})
	}
}
