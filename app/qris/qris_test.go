package qris

import (
	"errors"
	"fmt"
	"testing"
)

func tlv(tag, v string) string { return fmt.Sprintf("%s%02d%s", tag, len([]rune(v)), v) }

// Contoh QRIS statis buatan (bukan merchant sungguhan).
func sampleStatic() string {
	acq := tlv("00", "COM.GO-JEK.WWW") + tlv("01", "936009140123456789") + tlv("02", "G123456789") + tlv("03", "UMI")
	nmid := tlv("00", "ID.CO.QRIS.WWW") + tlv("02", "ID1020012345678") + tlv("03", "UMI")
	return WithCRC(tlv("00", "01") + tlv("01", "11") + tlv("26", acq) + tlv("51", nmid) + tlv("52", "5812") + tlv("53", "360") +
		tlv("58", "ID") + tlv("59", "KEDAI KOPI CONTOH") + tlv("60", "BANDUNG") + tlv("61", "40115") + tlv("62", tlv("07", "T01")))
}

func TestCRCKnownVector(t *testing.T) {
	// CRC-16/CCITT-FALSE("123456789") = 0x29B1
	if got := CRC16([]byte("123456789")); got != 0x29B1 {
		t.Fatalf("CRC16 = %04X, mau 29B1", got)
	}
}

func TestDecodeStatic(t *testing.T) {
	q, err := Decode(sampleStatic())
	if err != nil {
		t.Fatal(err)
	}
	if q.Dynamic || q.Amount != 0 || q.MerchantName != "KEDAI KOPI CONTOH" || q.MerchantCity != "BANDUNG" {
		t.Fatalf("hasil salah: %+v", q)
	}
	if q.NMID() != "ID1020012345678" || q.Criteria() != "UMI" || q.AcquirerName() != "GoPay" || !q.IsQRIS() || !q.IsIDR() {
		t.Fatalf("akun salah: nmid=%s crit=%s acq=%s", q.NMID(), q.Criteria(), q.AcquirerName())
	}
	if q.TerminalID != "T01" || MCCLabel(q.MCC) != "Restoran" {
		t.Fatalf("tambahan salah: %+v", q)
	}
}

func TestDecodeDynamicWithFee(t *testing.T) {
	acq := tlv("00", "ID.CO.BANKMANDIRI.WWW") + tlv("01", "936000080000123456") + tlv("02", "M001")
	p := WithCRC(tlv("00", "01") + tlv("01", "12") + tlv("26", acq) + tlv("52", "5411") + tlv("53", "360") + tlv("54", "25000.00") +
		tlv("55", "02") + tlv("56", "1000") + tlv("58", "ID") + tlv("59", "TOKO") + tlv("60", "JAKARTA"))
	q, err := Decode(p + "\n")
	if err != nil {
		t.Fatal(err)
	}
	if !q.Dynamic || q.Amount != 25000 || q.Fee(q.Amount) != 1000 || q.AcquirerName() != "Bank Mandiri" {
		t.Fatalf("hasil salah: dyn=%v amt=%d fee=%d acq=%s", q.Dynamic, q.Amount, q.Fee(q.Amount), q.AcquirerName())
	}
}

func TestRejects(t *testing.T) {
	good := sampleStatic()
	bad := good[:len(good)-1] + map[bool]string{true: "0", false: "1"}[good[len(good)-1] != '0']
	if _, err := Decode(bad); !errors.Is(err, ErrCRC) {
		t.Fatalf("CRC rusak harus ditolak, dapat %v", err)
	}
	if _, err := Decode("https://umrohku.web.id"); !errors.Is(err, ErrNotEMV) {
		t.Fatalf("URL harus ErrNotEMV, dapat %v", err)
	}
	if _, err := Decode("000201010211"); err == nil {
		t.Fatal("tanpa CRC harus ditolak")
	}
	noMerchant := WithCRC(tlv("00", "01") + tlv("01", "11") + tlv("53", "360") + tlv("59", "X"))
	if _, err := Decode(noMerchant); !errors.Is(err, ErrNoMerchant) {
		t.Fatalf("tanpa merchant harus ditolak, dapat %v", err)
	}
	if _, err := Decode("00020101021226"); err == nil {
		t.Fatal("TLV terpotong harus ditolak")
	}
}

// Contoh QR merchant Saudi buatan (negara SA, mata uang 682 = SAR), format EMVCo MPM.
func sampleSaudi(amount string) string {
	acq := tlv("00", "SA.COM.PENYELENGGARACONTOH") + tlv("01", "682099990000054321") + tlv("02", "MKH000321")
	body := tlv("00", "01") + tlv("01", "11") + tlv("26", acq) + tlv("52", "5499") + tlv("53", "682")
	if amount != "" {
		body = tlv("00", "01") + tlv("01", "12") + tlv("26", acq) + tlv("52", "5499") + tlv("53", "682") + tlv("54", amount)
	}
	return WithCRC(body + tlv("58", "SA") + tlv("59", "Toko Kurma Contoh") + tlv("60", "Makkah"))
}

func TestDecodeSaudiDynamic(t *testing.T) {
	q, err := Decode(sampleSaudi("35.50"))
	if err != nil {
		t.Fatal(err)
	}
	if !q.IsSAR() || !q.IsCrossBorder() || q.IsIDR() || q.IsQRIS() || q.CurrencyCode() != "SAR" || q.Country != "SA" {
		t.Fatalf("mata uang/negara salah: cur=%s negara=%s", q.Currency, q.Country)
	}
	if !q.Dynamic || q.AmountMinor != 3550 {
		t.Fatalf("nominal halalah = %d, mau 3550", q.AmountMinor)
	}
	if q.AcquirerName() != "PENYELENGGARACONTOH" || q.NMID() != "" {
		t.Fatalf("penyelenggara salah: %q nmid=%q", q.AcquirerName(), q.NMID())
	}
}

func TestDecodeSaudiStatic(t *testing.T) {
	q, err := Decode(sampleSaudi(""))
	if err != nil {
		t.Fatal(err)
	}
	if q.Dynamic || q.AmountMinor != 0 || !q.IsSAR() {
		t.Fatalf("hasil salah: %+v", q)
	}
}

func TestRupiahTetapBukanLintasNegara(t *testing.T) {
	q, err := Decode(sampleStatic())
	if err != nil {
		t.Fatal(err)
	}
	if q.IsCrossBorder() || q.IsSAR() || q.CurrencyCode() != "IDR" {
		t.Fatalf("QRIS rupiah terbaca lintas negara: %s", q.CurrencyCode())
	}
}
