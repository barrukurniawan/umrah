package accounts

import (
	"sync"
	"time"
)

// SupportMessage: chat pengguna <-> admin (kritik, saran, pertanyaan). Satu percakapan per
// akun, teks saja. ReadAt = kapan pesan dibaca pihak lawan (admin membaca pesan "user",
// pengguna membaca pesan "admin").
type SupportMessage struct {
	ID        uint       `gorm:"primaryKey"`
	UserID    uint       `gorm:"index:idx_support_user_id"`
	Sender    string     `gorm:"size:8;index:idx_support_unread"` // "user" | "admin"
	Body      string     `gorm:"size:2000"`
	Context   string     `gorm:"size:255"` // halaman saat pesan dikirim
	ReadAt    *time.Time `gorm:"index:idx_support_unread"`
	CreatedAt time.Time
}

const SupportMaxLen = 2000

// SupportMessages: pesan percakapan userID dengan id > after (maks 200 terakhir).
func SupportMessages(userID, after uint) []SupportMessage {
	var out []SupportMessage
	if DB == nil {
		return out
	}
	q := DB.Where("user_id = ?", userID)
	if after > 0 {
		q = q.Where("id > ?", after)
	}
	q.Order("id desc").Limit(200).Find(&out)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// SupportMarkRead menandai pesan dari `from` di percakapan userID sudah dibaca.
func SupportMarkRead(userID uint, from string) {
	if DB != nil {
		DB.Model(&SupportMessage{}).Where("user_id = ? AND sender = ? AND read_at IS NULL", userID, from).Update("read_at", time.Now())
	}
}

// SupportUnread: jumlah pesan dari `from` yang belum dibaca di percakapan userID.
func SupportUnread(userID uint, from string) int64 {
	var n int64
	if DB != nil {
		DB.Model(&SupportMessage{}).Where("user_id = ? AND sender = ? AND read_at IS NULL", userID, from).Count(&n)
	}
	return n
}

func SupportSend(userID uint, sender, body, context string) (SupportMessage, error) {
	m := SupportMessage{UserID: userID, Sender: sender, Body: body, Context: context, CreatedAt: time.Now()}
	err := DB.Create(&m).Error
	return m, err
}

// SupportThread: ringkasan satu percakapan untuk daftar di admin.
type SupportThread struct {
	UserID   uint
	Name     string
	Email    string
	Picture  string
	LastBody string
	LastFrom string
	LastAt   time.Time
	Unread   int64
}

func SupportThreads() []SupportThread {
	var out []SupportThread
	if DB == nil {
		return out
	}
	DB.Raw(`
		SELECT m.user_id, u.name, u.email, u.picture,
		       m.body AS last_body, m.sender AS last_from, m.created_at AS last_at,
		       (SELECT COUNT(*) FROM support_messages x WHERE x.user_id = m.user_id AND x.sender = 'user' AND x.read_at IS NULL) AS unread
		FROM support_messages m
		JOIN users u ON u.id = m.user_id
		WHERE m.id = (SELECT MAX(id) FROM support_messages y WHERE y.user_id = m.user_id)
		ORDER BY m.id DESC LIMIT 200`).Scan(&out)
	return out
}

func SupportUnreadForAdmin() int64 {
	var n int64
	if DB != nil {
		DB.Model(&SupportMessage{}).Where("sender = 'user' AND read_at IS NULL").Count(&n)
	}
	return n
}

func UserByID(id uint) *User {
	var u User
	if DB == nil || DB.Take(&u, id).Error != nil {
		return nil
	}
	return &u
}

var (
	supportRateMu sync.Mutex
	supportRate   = map[uint][]time.Time{}
)

// SupportAllow: maks 10 pesan per menit per akun.
func SupportAllow(userID uint) bool {
	supportRateMu.Lock()
	defer supportRateMu.Unlock()
	cutoff := time.Now().Add(-time.Minute)
	kept := supportRate[userID][:0]
	for _, t := range supportRate[userID] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= 10 {
		supportRate[userID] = kept
		return false
	}
	supportRate[userID] = append(kept, time.Now())
	return true
}
