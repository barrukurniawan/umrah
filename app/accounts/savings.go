package accounts

import (
	"errors"
	"math"
	"time"

	"gorm.io/gorm"
)

// Tabungan umroh: hanya pencatatan (tidak terhubung ke bank, Umrohku tidak memegang uang).

type SavingsGoal struct {
	ID           uint   `gorm:"primaryKey"`
	UserID       uint   `gorm:"index"`
	Title        string `gorm:"size:100"`
	Target       int64
	TargetDate   string `gorm:"size:10"` // "2006-01-02", boleh kosong
	People       int
	PackageKey   string `gorm:"size:300"`
	PackageLabel string `gorm:"size:300"`
	PackagePrice int64
	DownPayment  int64 // DP per orang dari paket (0 = tidak diketahui)
	CreatedAt    time.Time
	AchievedAt   *time.Time
}

type SavingsEntry struct {
	ID        uint  `gorm:"primaryKey"`
	GoalID    uint  `gorm:"index"`
	UserID    uint  `gorm:"index"`
	Amount    int64 // negatif = pengambilan
	Date      string `gorm:"size:10"`
	Note      string `gorm:"size:120"`
	CreatedAt time.Time
}

const (
	MaxGoals          = 10
	MaxEntriesPerGoal = 1000
	MaxAmount         = 10_000_000_000 // Rp 10 miliar, batas wajar input
)

var ErrNotFound = errors.New("tidak ditemukan")

func ListGoals(userID uint) []SavingsGoal {
	var g []SavingsGoal
	DB.Where("user_id = ?", userID).Order("achieved_at IS NOT NULL, created_at DESC").Find(&g)
	return g
}

func CountGoals(userID uint) int {
	var n int64
	DB.Model(&SavingsGoal{}).Where("user_id = ?", userID).Count(&n)
	return int(n)
}

// GetGoal selalu memeriksa pemilik supaya pengguna tidak bisa membuka tabungan orang lain.
func GetGoal(userID, id uint) (*SavingsGoal, error) {
	var g SavingsGoal
	if err := DB.Where("id = ? AND user_id = ?", id, userID).Take(&g).Error; err != nil {
		return nil, ErrNotFound
	}
	return &g, nil
}

func CreateGoal(g *SavingsGoal) error {
	g.CreatedAt = time.Now()
	return DB.Create(g).Error
}

func UpdateGoal(g *SavingsGoal) error { return DB.Save(g).Error }

func DeleteGoal(userID, id uint) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("goal_id = ? AND user_id = ?", id, userID).Delete(&SavingsEntry{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND user_id = ?", id, userID).Delete(&SavingsGoal{}).Error
	})
}

func ListEntries(goalID uint) []SavingsEntry {
	var e []SavingsEntry
	DB.Where("goal_id = ?", goalID).Order("date DESC, id DESC").Find(&e)
	return e
}

func CountEntries(goalID uint) int {
	var n int64
	DB.Model(&SavingsEntry{}).Where("goal_id = ?", goalID).Count(&n)
	return int(n)
}

func AddEntry(e *SavingsEntry) error {
	e.CreatedAt = time.Now()
	return DB.Create(e).Error
}

func DeleteEntry(userID, goalID, entryID uint) error {
	return DB.Where("id = ? AND goal_id = ? AND user_id = ?", entryID, goalID, userID).Delete(&SavingsEntry{}).Error
}

func GoalTotal(goalID uint) int64 {
	var total int64
	DB.Model(&SavingsEntry{}).Where("goal_id = ?", goalID).Select("COALESCE(SUM(amount), 0)").Scan(&total)
	return total
}

// Progress: ringkasan angka untuk kartu & halaman detail.
type Progress struct {
	Saved, Remaining int64
	Percent          int // 0-100
	PerMonth         int64
	MonthsLeft       int
	DaysLeft         int
	Overdue          bool
	EstimateDate     string // perkiraan tercapai dari rata-rata setoran ("" bila belum bisa dihitung)
	DPTotal          int64
	DPReady          bool
}

func ComputeProgress(g *SavingsGoal, entries []SavingsEntry, now time.Time) Progress {
	var p Progress
	for _, e := range entries {
		p.Saved += e.Amount
	}
	if p.Saved < 0 {
		p.Saved = 0
	}
	p.Remaining = max(g.Target-p.Saved, 0)
	if g.Target > 0 {
		p.Percent = int(math.Min(100, math.Floor(float64(p.Saved)*100/float64(g.Target))))
	}
	people := int64(max(g.People, 1))
	if g.DownPayment > 0 {
		p.DPTotal = g.DownPayment * people
		p.DPReady = p.Saved >= p.DPTotal
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if t, err := time.ParseInLocation("2006-01-02", g.TargetDate, now.Location()); err == nil {
		p.DaysLeft = int(t.Sub(today).Hours() / 24)
		if p.DaysLeft < 0 {
			p.Overdue = true
		} else {
			// Bulan tersisa dibulatkan ke atas, minimal 1, supaya "per bulan" tidak pernah tak terhingga.
			p.MonthsLeft = max(1, int(math.Ceil(float64(p.DaysLeft)/30.44)))
			p.PerMonth = int64(math.Ceil(float64(p.Remaining) / float64(p.MonthsLeft)))
		}
	}

	// Perkiraan dari rata-rata setoran bulanan sejak setoran pertama.
	if p.Remaining > 0 && len(entries) > 0 {
		first := entries[len(entries)-1].Date
		for _, e := range entries {
			if e.Date < first {
				first = e.Date
			}
		}
		if t, err := time.ParseInLocation("2006-01-02", first, now.Location()); err == nil {
			months := math.Max(1, today.Sub(t).Hours()/24/30.44)
			avg := float64(p.Saved) / months
			if avg > 0 {
				need := float64(p.Remaining) / avg
				p.EstimateDate = today.AddDate(0, 0, int(math.Ceil(need*30.44))).Format("2006-01-02")
			}
		}
	}
	return p
}

// MarkAchieved mencatat waktu tujuan tercapai (sekali); mengembalikan true bila baru saja tercapai.
func MarkAchieved(g *SavingsGoal, saved int64) bool {
	if g.AchievedAt == nil && g.Target > 0 && saved >= g.Target {
		now := time.Now()
		g.AchievedAt = &now
		DB.Model(g).Update("achieved_at", &now)
		return true
	}
	if g.AchievedAt != nil && saved < g.Target {
		g.AchievedAt = nil
		DB.Model(g).Update("achieved_at", nil)
	}
	return false
}
