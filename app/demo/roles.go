package demo

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"umrah/app/accounts"
	"umrah/app/analytics"

	"github.com/gofiber/fiber/v2"
)

// Peran demo: satu klik dari pengalih peran, tanpa Google login/password, supaya alur demo lancar
// di depan juri. Cookie ditandatangani HMAC (kunci = secret analitik instance demo).
const (
	RoleCookie   = "uk_demo_role"
	accessCookie = "uk_demo_access"
)

type Role struct {
	Kind string // "" (tamu) | jamaah | travel | admin | pembiayaan | merchant
	Slug string // travel/mitra/toko untuk peran travel, pembiayaan, merchant
}

func (r Role) String() string {
	if r.Slug != "" {
		return r.Kind + ":" + r.Slug
	}
	return r.Kind
}

// Audience: kunci Notifikasi untuk peran ini (jamaah memakai user:<id>, lihat InboxAudience).
func (r Role) Audience() string {
	switch r.Kind {
	case "admin":
		return "admin"
	case "travel", "pembiayaan", "merchant":
		if r.Kind == "pembiayaan" {
			return "partner:" + r.Slug
		}
		return r.Kind + ":" + r.Slug
	}
	return ""
}

func (r Role) Label() string {
	switch r.Kind {
	case "jamaah":
		return "Jamaah"
	case "travel":
		if t := TravelBySlug(r.Slug); t != nil {
			return "Travel · " + t.Name
		}
		return "Travel"
	case "admin":
		return "Admin umrohku"
	case "pembiayaan":
		if p := PartnerBySlug(r.Slug); p != nil {
			return "Mitra · " + p.Name
		}
		return "Mitra pembiayaan"
	case "merchant":
		if m := MerchantBySlug(r.Slug); m != nil {
			return "Toko · " + m.Name
		}
		return "Toko mitra"
	}
	return "Tamu"
}

func sign(v string) string {
	m := hmac.New(sha256.New, analytics.Secret())
	m.Write([]byte("demo-role|" + v))
	return hex.EncodeToString(m.Sum(nil))[:32]
}

func CurrentRole(c *fiber.Ctx) Role {
	if r, ok := c.Locals("demo_role").(Role); ok {
		return r
	}
	return Role{}
}

func parseRole(v string) Role {
	kind, slug, _ := strings.Cut(v, ":")
	switch kind {
	case "jamaah", "admin":
		return Role{Kind: kind}
	case "travel":
		if TravelBySlug(slug) != nil {
			return Role{Kind: kind, Slug: slug}
		}
	case "pembiayaan":
		if PartnerBySlug(slug) != nil {
			return Role{Kind: kind, Slug: slug}
		}
	case "merchant":
		if MerchantBySlug(slug) != nil {
			return Role{Kind: kind, Slug: slug}
		}
	}
	return Role{}
}

func SetRole(c *fiber.Ctx, r Role) {
	v := r.String()
	c.Cookie(&fiber.Cookie{Name: RoleCookie, Value: v + "." + sign(v), Path: "/", HTTPOnly: true, SameSite: "Lax",
		Secure: secure(c), Expires: time.Now().Add(30 * 24 * time.Hour)})
	c.Locals("demo_role", r)
}

func secure(c *fiber.Ctx) bool { return c.Protocol() == "https" || c.Get("X-Forwarded-Proto") == "https" }

// Middleware: noindex, kode akses (opsional), dan peran demo aktif. Dipasang setelah
// accounts.Middleware.
func Middleware(c *fiber.Ctx) error {
	c.Set("X-Robots-Tag", "noindex, nofollow")
	if Cfg.AccessCode != "" && !strings.HasPrefix(c.Path(), "/static/") && c.Path() != "/robots.txt" {
		if k := c.Query("kode"); k != "" && hmac.Equal([]byte(k), []byte(Cfg.AccessCode)) {
			c.Cookie(&fiber.Cookie{Name: accessCookie, Value: sign("access|" + Cfg.AccessCode), Path: "/", HTTPOnly: true,
				SameSite: "Lax", Secure: secure(c), Expires: time.Now().Add(30 * 24 * time.Hour)})
		} else if c.Cookies(accessCookie) != sign("access|"+Cfg.AccessCode) {
			c.Status(fiber.StatusUnauthorized)
			return c.Render("demo/access", fiber.Map{"Title": "Kode akses demo"})
		}
	}
	if raw := c.Cookies(RoleCookie); raw != "" {
		if i := strings.LastIndex(raw, "."); i > 0 && hmac.Equal([]byte(raw[i+1:]), []byte(sign(raw[:i]))) {
			c.Locals("demo_role", parseRole(raw[:i]))
		}
	}
	return c.Next()
}

// ---- akun jamaah demo (memakai tabel users yang sama dengan login Google) ----

const DemoUserSub = "demo:jamaah"

// EnsureDemoUser membuat (sekali) akun jamaah demo yang sudah menyetujui kebijakan privasi.
func EnsureDemoUser() *accounts.User {
	return ensureUser(DemoUserSub, "Siti Rahmawati", "siti.rahmawati@jamaah.contoh")
}

func ensureUser(sub, name, email string) *accounts.User {
	if accounts.DB == nil {
		return nil
	}
	var u accounts.User
	if accounts.DB.Where("google_sub = ?", sub).Take(&u).Error == nil {
		return &u
	}
	now := time.Now()
	u = accounts.User{GoogleSub: sub, Email: email, Name: name, CreatedAt: now, LastLoginAt: now, ConsentAt: &now}
	if accounts.DB.Create(&u).Error != nil {
		return nil
	}
	return &u
}

// LoginAs membuat sesi login untuk akun demo dan memasang cookie sesi biasa.
func LoginAs(c *fiber.Ctx, u *accounts.User) {
	if u == nil {
		return
	}
	tok, exp, err := accounts.NewSession(u.ID)
	if err != nil {
		return
	}
	c.Cookie(&fiber.Cookie{Name: accounts.SessionCookie, Value: tok, Path: "/", HTTPOnly: true, SameSite: "Lax",
		Secure: secure(c), Expires: exp})
	c.Locals("user", u)
}

// IsDemoUser: akun dibuat oleh seed/pengalih peran (dihapus saat reset).
func IsDemoUser(u *accounts.User) bool { return u != nil && strings.HasPrefix(u.GoogleSub, "demo:") }
