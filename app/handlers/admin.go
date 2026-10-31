package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"umrah/app/analytics"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

// /admin: satu password (bcrypt di env ADMIN_PASSWORD_HASH), cookie sesi bertanda tangan 7 hari.
const (
	adminCookie   = "umrohku_admin"
	adminSession  = 7 * 24 * time.Hour
	loginWindow   = 15 * time.Minute
	loginAttempts = 5
)

var adminTabs = []struct{ Key, Label string }{
	{"ringkasan", "Ringkasan"},
	{"sumber", "Sumber Trafik"},
	{"travel", "Travel & Paket"},
	{"pencarian", "Pencarian"},
	{"live", "Live"},
}

var (
	loginMu    sync.Mutex
	loginTries = map[string][]time.Time{}
)

func adminHash() string { return strings.TrimSpace(os.Getenv("ADMIN_PASSWORD_HASH")) }

func sign(exp int64) string {
	m := hmac.New(sha256.New, analytics.Secret())
	m.Write([]byte("admin|" + strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(m.Sum(nil))
}

func isAdmin(c *fiber.Ctx) bool {
	parts := strings.SplitN(c.Cookies(adminCookie), ".", 2)
	if len(parts) != 2 || analytics.Secret() == nil {
		return false
	}
	exp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	return hmac.Equal([]byte(parts[1]), []byte(sign(exp)))
}

// tooManyAttempts mencatat percobaan gagal per IP dan menolak setelah 5x dalam 15 menit.
func tooManyAttempts(ip string, failed bool) bool {
	loginMu.Lock()
	defer loginMu.Unlock()
	cutoff := time.Now().Add(-loginWindow)
	kept := loginTries[ip][:0]
	for _, t := range loginTries[ip] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if failed {
		kept = append(kept, time.Now())
	}
	loginTries[ip] = kept
	return len(kept) >= loginAttempts
}

func AdminLoginPage(c *fiber.Ctx) error {
	if isAdmin(c) {
		return c.Redirect("/admin")
	}
	return c.Render("admin/login", fiber.Map{"Configured": adminHash() != ""})
}

func AdminLogin(c *fiber.Ctx) error {
	ip := analytics.ClientIP(c)
	if tooManyAttempts(ip, false) {
		return c.Status(fiber.StatusTooManyRequests).Render("admin/login", fiber.Map{
			"Configured": true, "Error": "Terlalu banyak percobaan. Coba lagi 15 menit lagi."})
	}
	hash := adminHash()
	if hash == "" || bcrypt.CompareHashAndPassword([]byte(hash), []byte(c.FormValue("password"))) != nil {
		tooManyAttempts(ip, true)
		return c.Status(fiber.StatusUnauthorized).Render("admin/login", fiber.Map{
			"Configured": hash != "", "Error": "Password salah."})
	}
	exp := time.Now().Add(adminSession).Unix()
	c.Cookie(&fiber.Cookie{
		Name: adminCookie, Value: fmt.Sprintf("%d.%s", exp, sign(exp)), Path: "/admin",
		Expires: time.Unix(exp, 0), HTTPOnly: true, Secure: c.Protocol() == "https" || c.Get("X-Forwarded-Proto") == "https",
		SameSite: "Lax",
	})
	return c.Redirect("/admin")
}

func AdminLogout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{Name: adminCookie, Value: "", Path: "/admin", Expires: time.Unix(0, 0), HTTPOnly: true})
	return c.Redirect("/admin/login")
}

func AdminDashboard(c *fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Redirect("/admin/login")
	}
	tab := c.Query("tab", "ringkasan")
	valid := false
	for _, t := range adminTabs {
		valid = valid || t.Key == tab
	}
	if !valid {
		tab = "ringkasan"
	}
	days, _ := strconv.Atoi(c.Query("days", "7"))
	if days != 1 && days != 7 && days != 30 && days != 90 {
		days = 7
	}
	c.Set("Cache-Control", "no-store")
	return c.Render("admin/index", fiber.Map{
		"Tab":     tab,
		"Tabs":    adminTabs,
		"DaysOpt": []int{1, 7, 30, 90},
		"Days":    days,
		"R":       analytics.Build(analytics.RangeFor(days)),
		"Now":     time.Now().In(analytics.WIB).Format("02 Jan 2006 15:04"),
	})
}
