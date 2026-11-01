package accounts

import (
	"strings"
	"time"
)

// Statistik akun untuk tab "Pengguna" di /admin. Hanya data yang pengguna isi sendiri di akun
// (tidak dihubungkan dengan statistik kunjungan anonim, sesuai halaman Privasi).

type UserDay struct {
	Label string
	Count int64
	Pct   int
}

type UserRow struct {
	ID          uint
	Name        string
	Email       string
	EmailMasked string
	Picture     string
	CreatedAt   time.Time
	LastLoginAt time.Time
	Saved       int64
	Goals       int64
	Messages    int64
}

type TopSaved struct {
	Travel, Package string
	Count           int64
}

type UserStats struct {
	Total, NewInRange, Active7, Active30 int64
	Pending                              int64 // login tapi belum menyetujui kebijakan
	Days                                 []UserDay
	Users                                []UserRow
	TopSaved                             []TopSaved
	Goals                                int64
	AvgTarget                            int64 // rata-rata target tabungan per tujuan
	AvgPeople                            float64
	Achieved                             int64
}

// MaskEmail: "barru@motionpay.id" -> "bar***@motionpay.id".
func MaskEmail(e string) string {
	at := strings.LastIndex(e, "@")
	if at <= 0 {
		return e
	}
	n := 3
	if at < n {
		n = at
	}
	return e[:n] + "***" + e[at:]
}

// AdminUserStats: ringkasan akun untuk rentang [from, to) (WIB, per hari).
func AdminUserStats(from, to time.Time) UserStats {
	var st UserStats
	if DB == nil {
		return st
	}
	now := time.Now()
	DB.Model(&User{}).Where("consent_at IS NOT NULL").Count(&st.Total)
	DB.Model(&User{}).Where("consent_at IS NULL").Count(&st.Pending)
	DB.Model(&User{}).Where("consent_at IS NOT NULL AND created_at >= ? AND created_at < ?", from, to).Count(&st.NewInRange)
	DB.Model(&User{}).Where("consent_at IS NOT NULL AND last_login_at >= ?", now.AddDate(0, 0, -7)).Count(&st.Active7)
	DB.Model(&User{}).Where("consent_at IS NOT NULL AND last_login_at >= ?", now.AddDate(0, 0, -30)).Count(&st.Active30)

	// Pendaftar baru per hari.
	var created []time.Time
	DB.Model(&User{}).Where("consent_at IS NOT NULL AND created_at >= ? AND created_at < ?", from, to).Pluck("created_at", &created)
	perDay := map[string]int64{}
	for _, t := range created {
		perDay[t.In(from.Location()).Format("2006-01-02")]++
	}
	var max int64
	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		c := perDay[d.Format("2006-01-02")]
		st.Days = append(st.Days, UserDay{Label: d.Format("2/1"), Count: c})
		if c > max {
			max = c
		}
	}
	for i := range st.Days {
		if max > 0 {
			st.Days[i].Pct = int(st.Days[i].Count * 100 / max)
		}
	}

	// Daftar akun (terbaru dulu) dengan jumlah tersimpan, tabungan, pesan.
	DB.Raw(`
		SELECT u.id, u.name, u.email, u.picture, u.created_at, u.last_login_at,
		       (SELECT COUNT(*) FROM saved_packages s WHERE s.user_id = u.id) AS saved,
		       (SELECT COUNT(*) FROM savings_goals g WHERE g.user_id = u.id) AS goals,
		       (SELECT COUNT(*) FROM support_messages m WHERE m.user_id = u.id AND m.sender = 'user') AS messages
		FROM users u WHERE u.consent_at IS NOT NULL
		ORDER BY u.created_at DESC LIMIT 500`).Scan(&st.Users)
	for i := range st.Users {
		st.Users[i].EmailMasked = MaskEmail(st.Users[i].Email)
	}

	DB.Raw(`SELECT travel, package, COUNT(*) AS count FROM saved_packages
		GROUP BY travel, package ORDER BY count DESC, travel LIMIT 10`).Scan(&st.TopSaved)

	var g struct {
		N         int64
		AvgTarget float64
		AvgPeople float64
		Achieved  int64
	}
	DB.Raw(`SELECT COUNT(*) AS n, COALESCE(AVG(target), 0) AS avg_target, COALESCE(AVG(people), 0) AS avg_people,
		SUM(CASE WHEN achieved_at IS NOT NULL THEN 1 ELSE 0 END) AS achieved FROM savings_goals`).Scan(&g)
	st.Goals, st.AvgTarget, st.AvgPeople, st.Achieved = g.N, int64(g.AvgTarget), g.AvgPeople, g.Achieved
	return st
}
