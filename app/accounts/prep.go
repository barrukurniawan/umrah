package accounts

import (
	"strings"
	"time"
)

// PrepState: progres checklist Persiapan Umrah milik satu pengguna (satu baris per pengguna).
type PrepState struct {
	UserID     uint   `gorm:"primaryKey"`
	Checks     string `gorm:"size:4000"` // key dicentang, dipisah koma
	DepartDate string `gorm:"size:10"`   // "2006-01-02" atau kosong
	Gender     string `gorm:"size:1"`    // m | f | kosong
	UpdatedAt  time.Time
}

func GetPrep(userID uint) PrepState {
	st := PrepState{UserID: userID}
	if DB != nil {
		DB.Where("user_id = ?", userID).Take(&st)
	}
	return st
}

func SavePrep(st PrepState) error {
	st.UpdatedAt = time.Now()
	return DB.Save(&st).Error
}

func (p PrepState) CheckList() []string {
	if p.Checks == "" {
		return []string{}
	}
	return strings.Split(p.Checks, ",")
}

// PrepUsage: jumlah pengguna yang memakai checklist (untuk statistik admin nanti).
func PrepUsage() int64 {
	var n int64
	if DB != nil {
		DB.Model(&PrepState{}).Where("checks <> ''").Count(&n)
	}
	return n
}
