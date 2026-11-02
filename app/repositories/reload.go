package repositories

import (
	"errors"
	"log"
	"sync"
	"time"

	"umrah/app/models"

	"gorm.io/gorm"
)

// Muat ulang data crawl tanpa restart (dipicu SIGHUP dari cron setelah crawl).
// Seluruh import berjalan dalam satu transaksi: pengunjung tetap melihat data lama
// sampai commit, dan bila tidak ada data, transaksi dibatalkan (data lama tetap).

type ImportStat struct {
	Travel string
	Count  int64
}

type ImportInfo struct {
	At       time.Time
	Packages int64
	Details  int64
	Travels  []ImportStat
}

var (
	importMu   sync.RWMutex
	lastImport ImportInfo
	reloadMu   sync.Mutex
)

func recordImport(db *gorm.DB) {
	info := ImportInfo{At: time.Now()}
	db.Table("packages").Count(&info.Packages)
	db.Table("detail_packages").Count(&info.Details)
	db.Raw(`SELECT t.name AS travel, COUNT(p.id) AS count FROM travels t
		LEFT JOIN packages p ON p.travel_id = t.id GROUP BY t.name ORDER BY count DESC`).Scan(&info.Travels)
	importMu.Lock()
	lastImport = info
	importMu.Unlock()
}

// LastImport: kapan data paket terakhir dimuat, untuk ditampilkan di admin.
func LastImport() ImportInfo {
	importMu.RLock()
	defer importMu.RUnlock()
	return lastImport
}

var errNoCrawlData = errors.New("tidak ada data crawl")

// Reload membaca ulang output/all_*.json ke database dalam satu transaksi.
func Reload() error {
	reloadMu.Lock()
	defer reloadMu.Unlock()
	start := time.Now()
	err := DB.Transaction(func(tx *gorm.DB) error {
		if !importCrawlerData(tx) {
			return errNoCrawlData
		}
		return nil
	})
	if err != nil {
		log.Printf("[reload] dibatalkan, data lama tetap dipakai: %v", err)
		return err
	}
	recordImport(DB)
	li := LastImport()
	log.Printf("[reload] selesai dalam %s: %d paket, %d keberangkatan", time.Since(start).Round(time.Millisecond), li.Packages, li.Details)
	return nil
}

// AllPackages memuat semua paket beserta travel & keberangkatan dalam SATU transaksi baca,
// sehingga query paket dan query keberangkatan melihat snapshot yang sama walau Reload
// sedang berjalan (tanpa ini, hasil bisa kosong sesaat karena ID paket berganti).
func AllPackages() []models.Package {
	var out []models.Package
	DB.Transaction(func(tx *gorm.DB) error {
		return tx.Preload("Travel").Preload("Details").Find(&out).Error
	})
	return out
}
