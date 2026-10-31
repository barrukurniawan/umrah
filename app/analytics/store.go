// Package analytics mencatat kunjungan, pencarian, dan klik ke SQLite terpisah (data/analytics.db).
// Tanpa pihak ketiga; IP tidak disimpan, pengunjung unik = hash harian ber-secret.
package analytics

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Event struct {
	ID       uint   `gorm:"primaryKey"`
	TS       int64  `gorm:"index"`
	Type     string `gorm:"size:16;index"`
	Visitor  string `gorm:"size:16"`
	Source   string `gorm:"size:16"`
	Device   string `gorm:"size:8"`
	Path     string `gorm:"size:200"`
	Travel   string `gorm:"size:80;index"`
	Package  string `gorm:"size:200"`
	Price    int
	Position int
	Budget   int
	Meta     string `gorm:"size:500"`
	// VID: ID anonim dari cookie (acak, 1 tahun) untuk frekuensi kunjungan lintas hari.
	VID string `gorm:"column:vid;size:32;index"`
	// Lokasi hasil lookup GeoIP saat kunjungan; IP sendiri tidak disimpan.
	City    string `gorm:"size:60"`
	Region  string `gorm:"size:60"`
	Country string `gorm:"size:60"`
}

// Data mentah disimpan ±13 bulan.
const retention = 395 * 24 * time.Hour

var (
	DB     *gorm.DB
	secret []byte
	queue  = make(chan Event, 2000)
	WIB    = time.FixedZone("WIB", 7*3600)
)

// Init membuka DB analitik dan menyalakan penulis latar belakang. Gagal = analitik nonaktif, web tetap jalan.
func Init(path string) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		log.Println("[analytics] nonaktif:", err)
		return
	}
	db, err := gorm.Open(sqlite.Open(path+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		log.Println("[analytics] nonaktif:", err)
		return
	}
	if err := db.AutoMigrate(&Event{}); err != nil {
		log.Println("[analytics] nonaktif:", err)
		return
	}
	DB = db
	secret = loadSecret(filepath.Join(filepath.Dir(path), "analytics.secret"))
	go writer()
	go pruner()
	log.Println("[analytics] aktif:", path)
}

func loadSecret(path string) []byte {
	if b, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(b))) >= 32 {
		return []byte(strings.TrimSpace(string(b)))
	}
	buf := make([]byte, 32)
	rand.Read(buf)
	s := hex.EncodeToString(buf)
	if err := os.WriteFile(path, []byte(s), 0600); err != nil {
		log.Println("[analytics] secret tidak tersimpan, pengunjung unik akan reset saat restart:", err)
	}
	return []byte(s)
}

// Secret dipakai juga untuk menandatangani cookie admin.
func Secret() []byte { return secret }

// Record mengantrikan event; tidak pernah memblokir request (antrean penuh = event dibuang).
func Record(e Event) {
	if DB == nil {
		return
	}
	if e.TS == 0 {
		e.TS = time.Now().Unix()
	}
	select {
	case queue <- e:
	default:
	}
}

func writer() {
	batch := make([]Event, 0, 100)
	tick := time.NewTicker(2 * time.Second)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := DB.CreateInBatches(batch, 100).Error; err != nil {
			log.Println("[analytics] gagal simpan:", err)
		}
		batch = batch[:0]
	}
	for {
		select {
		case e := <-queue:
			batch = append(batch, e)
			if len(batch) >= 100 {
				flush()
			}
		case <-tick.C:
			flush()
		}
	}
}

func pruner() {
	for {
		DB.Where("ts < ?", time.Now().Add(-retention).Unix()).Delete(&Event{})
		time.Sleep(24 * time.Hour)
	}
}

// VisitorID: hash harian dari IP + User-Agent. Berganti tiap hari WIB, tidak bisa dibalik tanpa secret.
func VisitorID(ip, ua string) string {
	day := time.Now().In(WIB).Format("2006-01-02")
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(day + "|" + ip + "|" + ua))
	return hex.EncodeToString(m.Sum(nil))[:16]
}
