package demo

import (
	"errors"
	"testing"
	"time"

	"umrah/app/qris"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func marketTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:markettest?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&MarketOrder{}, &InboxMessage{}, &FundSource{}, &QRPayment{}); err != nil {
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

func TestMarketCatalogConsistent(t *testing.T) {
	for slug, items := range StoreItems {
		m := MerchantBySlug(slug)
		if m == nil {
			t.Fatalf("toko %s tidak ada di Merchants", slug)
		}
		if len(m.Short) == 0 || len(m.Short) > 25 || m.Lat == 0 || m.MCC == "" {
			t.Errorf("%s: data market belum lengkap", slug)
		}
		seen := map[string]bool{}
		for _, i := range items {
			if seen[i.Key] || MarketCatByKey(i.Cat) == nil || i.SAR <= 0 {
				t.Errorf("%s: barang %q duplikat/kategori salah/harga nol", slug, i.Key)
			}
			seen[i.Key] = true
		}
	}
	for _, c := range Cities {
		if len(MarketStores(c, "", "")) < 4 {
			t.Errorf("%s: toko terlalu sedikit", c.Name)
		}
	}
	if got := MarketStores(CityByKey("madinah"), "abaya", ""); len(got) != 1 || got[0].Slug != "madinah-abaya" {
		t.Errorf("filter kategori abaya Madinah: %v", got)
	}
	if got := MarketStores(CityByKey("makkah"), "", "sajadah"); len(got) == 0 {
		t.Error("pencarian sajadah di Makkah kosong")
	}
}

func TestMarketOrderQRPayPickup(t *testing.T) {
	marketTestDB(t)
	slug := "madinah-babsalam"
	if _, err := CreateMarketOrder(1, slug, "Siti", "segera", "", map[string]int{"tidak-ada": 2}); !errors.Is(err, ErrMarketEmpty) {
		t.Fatalf("barang tak dikenal harus ditolak, dapat %v", err)
	}
	if _, err := CreateMarketOrder(1, slug, " ", "segera", "", map[string]int{"sajadah-travel": 1}); !errors.Is(err, ErrMarketName) {
		t.Fatalf("nama kosong harus ditolak, dapat %v", err)
	}
	o, err := CreateMarketOrder(1, slug, "Siti", "segera", "", map[string]int{"sajadah-travel": 2, "tasbih-kayu": 1, "x": 9})
	if err != nil {
		t.Fatal(err)
	}
	if o.TotalSAR != 60 || o.TotalIDR != 60*SARRateDemo || len(o.Lines()) != 2 || len(o.PickupCode) != 6 {
		t.Fatalf("total/baris salah: %+v", o)
	}

	// QR dinamis harus terbaca parser QRIS: CRC valid, nominal rupiah, kode pesanan di tag 62.
	q, err := qris.Decode(o.QRPayload)
	if err != nil {
		t.Fatalf("QR pesanan tidak valid: %v", err)
	}
	if !q.Dynamic || q.Amount != o.TotalIDR || q.Reference != o.Code || q.BillNumber != o.Code || !q.IsIDR() {
		t.Fatalf("isi QR salah: dyn=%v amount=%d ref=%q", q.Dynamic, q.Amount, q.Reference)
	}
	if q.AcquirerName() != "PJP Contoh (berizin BI)" {
		t.Errorf("penyelenggara: %q", q.AcquirerName())
	}

	// Bayar lewat QRIS umrohku -> pesanan otomatis dibayar.
	Fund(1, "Siti")
	p, err := PayQR(PayInput{UserID: 1, Holder: "Siti", QR: q, PIN: "123456"})
	if err != nil {
		t.Fatal(err)
	}
	got := MarketOrderFor(1, o.Code)
	if got.Status != "dibayar" || got.QRRef != p.Code || MarketOrderByQRRef(p.Code) == nil {
		t.Fatalf("pesanan belum dibayar setelah QRIS: %+v", got)
	}
	if err := PayMarketOrder(got, "DANA (simulasi)", ""); !errors.Is(err, ErrMarketState) {
		t.Fatalf("bayar dua kali harus ditolak, dapat %v", err)
	}

	// Toko: siap, lalu serahkan dengan kode ambil.
	if _, err := AdvanceMarketOrder("madinah-abaya", o.Code, "siap", ""); err == nil {
		t.Fatal("toko lain tidak boleh mengubah pesanan")
	}
	if _, err := AdvanceMarketOrder(slug, o.Code, "siap", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := AdvanceMarketOrder(slug, o.Code, "diambil", "000000x"); !errors.Is(err, ErrMarketCode) {
		t.Fatalf("kode ambil salah harus ditolak, dapat %v", err)
	}
	done, err := AdvanceMarketOrder(slug, o.Code, "diambil", got.PickupCode)
	if err != nil || done.Status != "diambil" || done.DoneAt == nil {
		t.Fatalf("serah terima gagal: %v %+v", err, done)
	}
}

func TestMarketQRExpiryRenew(t *testing.T) {
	marketTestDB(t)
	o, err := CreateMarketOrder(1, "makkah-ajyad", "Siti", "isya", "", map[string]int{"ajwa-500": 1})
	if err != nil {
		t.Fatal(err)
	}
	DB.Model(&MarketOrder{}).Where("id = ?", o.ID).Update("expires_at", time.Now().Add(-time.Minute))
	got := MarketOrderFor(1, o.Code)
	if got.Status != "kedaluwarsa" {
		t.Fatalf("harus kedaluwarsa, dapat %s", got.Status)
	}
	if err := PayMarketOrder(got, "DANA (simulasi)", ""); !errors.Is(err, ErrMarketExpired) {
		t.Fatalf("QR kedaluwarsa tidak boleh dibayar, dapat %v", err)
	}
	if err := RenewMarketQR(got); err != nil || got.Status != "menunggu_bayar" || got.SecondsLeft() < 800 {
		t.Fatalf("QR baru gagal: %v %+v", err, got)
	}
	if err := PayMarketOrder(got, "ShopeePay (simulasi)", ""); err != nil {
		t.Fatal(err)
	}
}

func TestDistLabel(t *testing.T) {
	for m, want := range map[float64]string{300: "300 m · ±6 mnt jalan", 3400: "3,4 km · ±12 mnt naik mobil"} {
		if got := DistLabel(m); got != want {
			t.Errorf("DistLabel(%v) = %q, want %q", m, got, want)
		}
	}
}
