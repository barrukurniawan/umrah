// Package demo: mode demo startup.umrohku.web.id (DEMO_MODE=1).
//
// Semua fitur proposal (portal travel, lead, booking + DP, cicilan, promo, undian, umrah mandiri,
// layanan Tanah Suci) hidup di sini dengan data dummy di data/demo.db. Produksi umrohku.web.id
// menjalankan binary yang sama tanpa DEMO_MODE: route demo tidak terdaftar, fungsi template demo
// mengembalikan nilai kosong, dan tidak ada tabel demo yang dibuat.
//
// Prinsip (lihat PLAN_STARTUP_DEMO.md & PLAN_CICILAN_PROMO_UNDIAN.md):
//   - tidak ada uang sungguhan yang bergerak; semua pembayaran lewat payments.MockGateway
//   - dana jamaah selalu diarahkan ke rekening penampungan travel, tidak pernah ke umrohku
//   - umrohku tidak memberi pinjaman (pembiayaan hanya rujukan) dan tidak punya "saldo umrohku"
//   - travel & lembaga keuangan di alur bisnis semuanya fiktif; tidak ada nomor izin karangan
//     atas nama pihak sungguhan
//   - WA/email tidak dikirim keluar; semuanya masuk Kotak masuk demo
package demo

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	DB      *gorm.DB
	enabled bool
	demoModels []any
	modelMu sync.Mutex
)

// WIB dipakai untuk semua tanggal demo (seed relatif terhadap "hari ini" WIB).
var WIB = time.FixedZone("WIB", 7*3600)

// Enabled: true hanya bila DEMO_MODE=1 dan database demo berhasil dibuka.
func Enabled() bool { return enabled }

// FlagSet: DEMO_MODE=1 di env (dipakai main.go sebelum Init, mis. untuk menulis paket fiktif).
func FlagSet() bool { return os.Getenv("DEMO_MODE") == "1" }

// RegisterModels dipanggil dari init() setiap berkas model supaya AutoMigrate satu tempat.
func RegisterModels(m ...any) {
	modelMu.Lock()
	defer modelMu.Unlock()
	demoModels = append(demoModels, m...)
}

// Init membuka data/demo.db, migrasi, menulis paket travel fiktif ke output/, dan mengisi seed
// bila database masih kosong. Harus dipanggil SEBELUM repositories.InitDB supaya paket fiktif
// ikut termuat saat data paket dibangun.
func Init(path string) {
	if !FlagSet() {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		log.Println("[demo] nonaktif:", err)
		return
	}
	db, err := gorm.Open(sqlite.Open(path+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err == nil {
		err = db.AutoMigrate(demoModels...)
	}
	if err != nil {
		log.Println("[demo] nonaktif:", err)
		return
	}
	DB = db
	enabled = true
	loadConfig()
	var n int64
	DB.Model(&Travel{}).Count(&n)
	if n == 0 {
		seedBusiness()
	}
	if err := WritePackagesFile(); err != nil {
		log.Println("[demo] gagal menulis paket fiktif:", err)
	}
	startJobs()
	log.Println("[demo] MODE DEMO aktif:", path)
}

// ---- Konfigurasi tarif & teks (config/startup_demo.json, boleh diubah pemilik) ----

type Plan struct {
	Key      string   `json:"key"`
	Name     string   `json:"name"`
	Price    int64    `json:"price"` // per bulan
	Features []string `json:"features"`
}

type Partner struct {
	Slug   string  `json:"slug"`
	Name   string  `json:"name"`
	Kind   string  `json:"kind"` // "pembiayaan" | "bank"
	Note   string  `json:"note"`
	Ujrah  map[string]float64 `json:"ujrah"` // tenor (bulan) -> total ujrah/margin % dari pokok (contoh)
	Admin  int64   `json:"admin"`           // biaya administrasi (contoh)
	FeePct float64 `json:"fee_pct"`         // fee rujukan untuk umrohku dari mitra (% pokok cair)
}

type Config struct {
	CPL            int64     `json:"cpl"`             // biaya per lead valid
	DefaultKomisi  float64   `json:"default_komisi"`  // % komisi booking default
	Plans          []Plan    `json:"plans"`
	Partners       []Partner `json:"partners"`
	SAR            int64     `json:"sar"` // kurs rupiah per 1 SAR untuk voucher Tanah Suci (contoh)
	AccessCode     string    `json:"access_code"`     // opsional: kode akses demo ("" = terbuka)
}

var Cfg = Config{
	CPL:           25000,
	DefaultKomisi: 3,
	SAR:           4300,
	Plans: []Plan{
		{Key: "gratis", Name: "Gratis", Price: 0, Features: []string{"Profil travel & badge PPIU", "Paket tampil di pencarian", "Laporan klik bulanan ringkas"}},
		{Key: "pro", Name: "Pro", Price: 499000, Features: []string{"Kelola paket & sisa seat sendiri", "Laporan lead harian + ekspor CSV", "Terima booking & DP lewat umrohku", "Kotak masuk lead terpadu"}},
		{Key: "premium", Name: "Premium", Price: 1490000, Features: []string{"Semua fitur Pro", "Diskon 20% posisi Promo Spesial", "Laporan pesaing & tren harga", "Manajer akun khusus"}},
	},
	Partners: []Partner{
		{Slug: "amanah", Name: "Mitra Syariah Amanah (contoh)", Kind: "pembiayaan", Note: "Akad murabahah, contoh perusahaan pembiayaan syariah", Ujrah: map[string]float64{"6": 6, "12": 11, "24": 20}, Admin: 250000, FeePct: 1.5},
		{Slug: "barokah", Name: "Mitra Syariah Barokah (contoh)", Kind: "pembiayaan", Note: "Akad ijarah multijasa, contoh", Ujrah: map[string]float64{"6": 5.5, "12": 10.5, "24": 21}, Admin: 300000, FeePct: 1.25},
		{Slug: "bank-contoh", Name: "Bank Syariah Contoh", Kind: "bank", Note: "Tabungan umroh (wadiah) & pembiayaan, contoh", Ujrah: map[string]float64{"12": 9.5, "24": 18}, Admin: 200000, FeePct: 1},
	},
}

func loadConfig() {
	b, err := os.ReadFile("config/startup_demo.json")
	if err != nil {
		return
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		log.Println("[demo] config/startup_demo.json tidak valid:", err)
		return
	}
	if c.CPL > 0 {
		Cfg.CPL = c.CPL
	}
	if c.DefaultKomisi > 0 {
		Cfg.DefaultKomisi = c.DefaultKomisi
	}
	if len(c.Plans) > 0 {
		Cfg.Plans = c.Plans
	}
	if len(c.Partners) > 0 {
		Cfg.Partners = c.Partners
	}
	if c.SAR > 0 {
		Cfg.SAR = c.SAR
	}
	Cfg.AccessCode = strings.TrimSpace(c.AccessCode)
}

func PlanByKey(k string) Plan {
	for _, p := range Cfg.Plans {
		if p.Key == k {
			return p
		}
	}
	return Cfg.Plans[0]
}

func PartnerBySlug(s string) *Partner {
	for i := range Cfg.Partners {
		if Cfg.Partners[i].Slug == s {
			return &Cfg.Partners[i]
		}
	}
	return nil
}

// Today: tanggal WIB "2006-01-02".
func Today() string { return time.Now().In(WIB).Format("2006-01-02") }
