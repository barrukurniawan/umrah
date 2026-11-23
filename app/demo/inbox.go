package demo

import (
	"fmt"
	"time"
)

// Notifikasi: semua "WA" dan "email" berakhir di sini, tidak pernah dikirim keluar.

func UserAudience(userID uint) string { return fmt.Sprintf("user:%d", userID) }

// Notify menyimpan satu pesan simulasi. channel: "wa" | "email".
func Notify(audience, channel, to, subject, body, link string) {
	if DB == nil || audience == "" {
		return
	}
	DB.Create(&InboxMessage{Audience: audience, Channel: channel, To: to, Subject: subject, Body: body, Link: link, CreatedAt: time.Now()})
}

func Inbox(audience string, limit int) []InboxMessage {
	var m []InboxMessage
	if DB != nil && audience != "" {
		DB.Where("audience = ?", audience).Order("created_at DESC, id DESC").Limit(limit).Find(&m)
	}
	return m
}

func UnreadCount(audience string) int {
	var n int64
	if DB != nil && audience != "" {
		DB.Model(&InboxMessage{}).Where("audience = ? AND read = ?", audience, false).Count(&n)
	}
	return int(n)
}

func MarkRead(audience string) {
	if DB != nil && audience != "" {
		DB.Model(&InboxMessage{}).Where("audience = ? AND read = ?", audience, false).Update("read", true)
	}
}
