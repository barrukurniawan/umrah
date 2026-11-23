package demo

import (
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func shopTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:shoptest?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Voucher{}, &VoucherRedemption{}, &InboxMessage{}, &Pickup{}, &CartItem{}); err != nil {
		t.Fatal(err)
	}
	old := DB
	DB = db
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
		DB = old
	})
}

func TestNormalizeVoucherCode(t *testing.T) {
	for in, want := range map[string]string{
		"UMV-AB2C-DE3F":                "UMV-AB2C-DE3F",
		"umv ab2c de3f":                "UMV-AB2C-DE3F",
		"ab2cde3f":                     "UMV-AB2C-DE3F",
		"UMV-AB2C":                     "",
		"":                             "",
		"UMV-AB2C-DE3F-XXXX-YYYY-ZZZZ": "",
	} {
		if got := NormalizeVoucherCode(in); got != want {
			t.Errorf("NormalizeVoucherCode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestVoucherRedemptionRules(t *testing.T) {
	shopTestDB(t)
	Cfg.SAR = 4300

	if _, err := BuyVoucher(1, 100_000); err == nil {
		t.Fatal("nominal di bawah minimum harus ditolak")
	}
	v, err := BuyVoucher(1, 1_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if v.ValueSAR != 232 || v.AmountIDR != 232*4300 || v.BalanceSAR != 232 {
		t.Fatalf("nilai voucher salah: %+v", v)
	}
	m := Merchants[0].Slug

	// belum dibayar
	if _, _, err := RedeemVoucher(v.Code, m, 10, ""); !errors.Is(err, ErrVoucherUnpaid) {
		t.Fatalf("voucher belum bayar: err = %v", err)
	}
	if err := PayVoucher(v, "x"); err != nil {
		t.Fatal(err)
	}
	if err := PayVoucher(v, "x"); err == nil {
		t.Fatal("bayar dua kali harus ditolak")
	}

	// validasi input
	if _, _, err := RedeemVoucher("UMV-ZZZZ-ZZZZ", m, 10, ""); !errors.Is(err, ErrVoucherNotFound) {
		t.Fatalf("kode tak dikenal: err = %v", err)
	}
	if _, _, err := RedeemVoucher(v.Code, "toko-palsu", 10, ""); !errors.Is(err, ErrBadMerchant) {
		t.Fatalf("toko tak dikenal: err = %v", err)
	}
	if _, _, err := RedeemVoucher(v.Code, m, 0, ""); !errors.Is(err, ErrBadAmount) {
		t.Fatalf("nominal 0: err = %v", err)
	}
	if _, _, err := RedeemVoucher(v.Code, m, 233, ""); !errors.Is(err, ErrOverBalance) {
		t.Fatalf("melebihi saldo: err = %v", err)
	}

	// tukar sebagian, kode boleh diketik tanpa tanda hubung/huruf kecil
	got, red, err := RedeemVoucher(" "+lower(v.Code[4:8]+v.Code[9:]), m, 200, "kurma")
	if err != nil {
		t.Fatal(err)
	}
	if got.BalanceSAR != 32 || got.Status != "aktif" || red.BalanceAfter != 32 {
		t.Fatalf("saldo setelah tukar sebagian salah: %+v %+v", got, red)
	}
	if _, _, err := RedeemVoucher(v.Code, m, 33, ""); !errors.Is(err, ErrOverBalance) {
		t.Fatalf("sisa saldo terlampaui: err = %v", err)
	}
	// habiskan saldo
	got, _, err = RedeemVoucher(v.Code, Merchants[2].Slug, 32, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.BalanceSAR != 0 || got.Status != "habis" {
		t.Fatalf("voucher harus habis: %+v", got)
	}
	if _, _, err := RedeemVoucher(v.Code, m, 1, ""); !errors.Is(err, ErrVoucherEmpty) {
		t.Fatalf("voucher habis: err = %v", err)
	}
	if n := len(Redemptions(v.ID)); n != 2 {
		t.Fatalf("riwayat penukaran = %d, want 2", n)
	}
	// kepemilikan
	if VoucherFor(2, v.Code) != nil {
		t.Fatal("pengguna lain tidak boleh melihat voucher ini")
	}
	if VoucherFor(1, v.Code) == nil {
		t.Fatal("pemilik harus bisa melihat vouchernya")
	}
}

func TestPickupOwnership(t *testing.T) {
	shopTestDB(t)
	slot := PickupSlots()[0].Value
	p, err := CreatePickup(1, "Siti", Merchants[0].Slug, slot, map[string]int{"kurma-ajwa": 2})
	if err != nil {
		t.Fatal(err)
	}
	if p.TotalSAR != 140 {
		t.Fatalf("total = %d", p.TotalSAR)
	}
	if _, err := CreatePickup(1, "Siti", Merchants[0].Slug, "2000-01-01 10:00", map[string]int{"kurma-ajwa": 1}); err == nil {
		t.Fatal("slot tidak valid harus ditolak")
	}
	if _, err := AdvancePickup(Merchants[1].Slug, p.ID, "siap"); err == nil {
		t.Fatal("toko lain tidak boleh mengubah pickup ini")
	}
	if _, err := AdvancePickup(Merchants[0].Slug, p.ID, "diambil"); err == nil {
		t.Fatal("belum siap tidak boleh langsung diambil")
	}
	if q, err := AdvancePickup(Merchants[0].Slug, p.ID, "siap"); err != nil || q.Status != "siap" {
		t.Fatalf("siap: %v", err)
	}
	if q, err := AdvancePickup(Merchants[0].Slug, p.ID, "diambil"); err != nil || q.Status != "diambil" {
		t.Fatalf("diambil: %v", err)
	}
	if PickupFor(2, p.Code) != nil {
		t.Fatal("pengguna lain tidak boleh melihat pickup ini")
	}
}

func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
