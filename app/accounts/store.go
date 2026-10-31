// Package accounts: login Google (opsional), sesi, dan paket tersimpan. Data di data/users.db,
// terpisah dari data paket (dibangun ulang tiap restart) dan dari analitik anonim.
package accounts

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type User struct {
	ID          uint   `gorm:"primaryKey"`
	GoogleSub   string `gorm:"size:64;uniqueIndex"`
	Email       string `gorm:"size:200"`
	Name        string `gorm:"size:120"`
	Picture     string `gorm:"size:500"`
	CreatedAt   time.Time
	LastLoginAt time.Time
	ConsentAt   *time.Time // nil = belum menyetujui kebijakan privasi (login pertama)
}

type Session struct {
	TokenHash string `gorm:"primaryKey;size:64"` // sha256 token; token mentah hanya ada di cookie
	UserID    uint   `gorm:"index"`
	ExpiresAt time.Time
	CreatedAt time.Time
}

// SavedPackage: kunci stabil (travel|nama paket) + salinan data saat disimpan, karena ID paket
// berubah setiap data crawl dimuat ulang.
type SavedPackage struct {
	ID           uint   `gorm:"primaryKey"`
	UserID       uint   `gorm:"uniqueIndex:idx_user_key"`
	Key          string `gorm:"size:300;uniqueIndex:idx_user_key"`
	Travel       string `gorm:"size:80"`
	Package      string `gorm:"size:200"`
	Price        int
	Duration     int
	Airline      string `gorm:"size:80"`
	HotelMakkah  string `gorm:"size:120"`
	HotelMadinah string `gorm:"size:120"`
	Departures   string `gorm:"size:300"`
	SavedAt      time.Time
}

var DB *gorm.DB

const sessionTTL = 30 * 24 * time.Hour

func Init(path string) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		log.Println("[akun] nonaktif:", err)
		return
	}
	db, err := gorm.Open(sqlite.Open(path+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err == nil {
		err = db.AutoMigrate(&User{}, &Session{}, &SavedPackage{}, &SavingsGoal{}, &SavingsEntry{})
	}
	if err != nil {
		log.Println("[akun] nonaktif:", err)
		return
	}
	DB = db
	go func() {
		for {
			DB.Where("expires_at < ?", time.Now()).Delete(&Session{})
			time.Sleep(6 * time.Hour)
		}
	}()
	log.Println("[akun] aktif:", path)
}

func hashToken(t string) string {
	h := sha256.Sum256([]byte(t))
	return hex.EncodeToString(h[:])
}

// NewSession membuat token acak; yang disimpan di DB hanya hash-nya.
func NewSession(userID uint) (token string, expires time.Time, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return
	}
	token = hex.EncodeToString(buf)
	expires = time.Now().Add(sessionTTL)
	err = DB.Create(&Session{TokenHash: hashToken(token), UserID: userID, ExpiresAt: expires, CreatedAt: time.Now()}).Error
	return
}

func UserBySession(token string) *User {
	if DB == nil || len(token) != 64 {
		return nil
	}
	var s Session
	if DB.Where("token_hash = ? AND expires_at > ?", hashToken(token), time.Now()).Take(&s).Error != nil {
		return nil
	}
	var u User
	if DB.Take(&u, s.UserID).Error != nil {
		return nil
	}
	return &u
}

func DeleteSession(token string) {
	if DB != nil && token != "" {
		DB.Delete(&Session{}, "token_hash = ?", hashToken(token))
	}
}

// UpsertGoogleUser membuat atau memperbarui pengguna berdasarkan "sub" Google (ID permanen akun).
func UpsertGoogleUser(sub, email, name, picture string) (*User, error) {
	var u User
	err := DB.Where("google_sub = ?", sub).Take(&u).Error
	now := time.Now()
	if err != nil {
		u = User{GoogleSub: sub, Email: email, Name: name, Picture: picture, CreatedAt: now, LastLoginAt: now}
		return &u, DB.Create(&u).Error
	}
	u.Email, u.Name, u.Picture, u.LastLoginAt = email, name, picture, now
	return &u, DB.Save(&u).Error
}

func SetConsent(userID uint) error {
	now := time.Now()
	return DB.Model(&User{}).Where("id = ?", userID).Update("consent_at", &now).Error
}

// DeleteUser menghapus akun beserta semua datanya secara permanen.
func DeleteUser(userID uint) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&SavedPackage{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&SavingsEntry{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&SavingsGoal{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&Session{}).Error; err != nil {
			return err
		}
		return tx.Delete(&User{}, userID).Error
	})
}

func SavedKeys(userID uint) map[string]bool {
	keys := map[string]bool{}
	if DB == nil {
		return keys
	}
	var rows []SavedPackage
	DB.Select("key").Where("user_id = ?", userID).Find(&rows)
	for _, r := range rows {
		keys[r.Key] = true
	}
	return keys
}

func ListSaved(userID uint) []SavedPackage {
	var rows []SavedPackage
	DB.Where("user_id = ?", userID).Order("saved_at DESC").Find(&rows)
	return rows
}

// Save menyimpan (atau memperbarui salinan) paket; Unsave menghapusnya.
func Save(p SavedPackage) error {
	var existing SavedPackage
	if DB.Where("user_id = ? AND key = ?", p.UserID, p.Key).Take(&existing).Error == nil {
		return nil
	}
	if p.SavedAt.IsZero() {
		p.SavedAt = time.Now()
	}
	return DB.Create(&p).Error
}

func Unsave(userID uint, key string) error {
	return DB.Where("user_id = ? AND key = ?", userID, key).Delete(&SavedPackage{}).Error
}

func CountSaved(userID uint) int {
	var n int64
	DB.Model(&SavedPackage{}).Where("user_id = ?", userID).Count(&n)
	return int(n)
}
