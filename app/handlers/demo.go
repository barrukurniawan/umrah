package handlers

import (
	"strconv"
	"strings"
	"time"

	"umrah/app/accounts"
	"umrah/app/demo"

	"github.com/gofiber/fiber/v2"
)

// Mode demo startup.umrohku.web.id: semua route di sini hanya didaftarkan bila demo.Enabled().

type DemoInfo struct {
	Role   string
	Label  string
	Unread int
	Path   string
}

// Tab untuk partial "biz_header".
type Tab struct {
	Href, Label string
	Active      bool
	Count       int
}

func demoView(c *fiber.Ctx) DemoInfo {
	r := demo.CurrentRole(c)
	return DemoInfo{Role: r.String(), Label: r.Label(), Unread: demo.UnreadCount(inboxAudience(c)), Path: c.OriginalURL()}
}

// inboxAudience: notifikasi peran aktif; jamaah = akun yang login.
func inboxAudience(c *fiber.Ctx) string {
	r := demo.CurrentRole(c)
	if r.Kind == "" || r.Kind == "jamaah" {
		if u := accounts.Current(c); u != nil {
			return demo.UserAudience(u.ID)
		}
		return ""
	}
	return r.Audience()
}

func tabs(active string, items ...string) []Tab {
	var t []Tab
	for i := 0; i+1 < len(items); i += 2 {
		t = append(t, Tab{Href: items[i], Label: items[i+1], Active: items[i] == active})
	}
	return t
}

func RegisterDemo(app *fiber.App) {
	app.Get("/robots.txt", func(c *fiber.Ctx) error { return c.SendString("User-agent: *\nDisallow: /\n") })
	app.Get("/demo", DemoTour)
	// Proposal bisnis (statis, khusus subdomain demo)
	app.Get("/proposal", func(c *fiber.Ctx) error { return c.SendFile("./web/proposal/index.html") })
	app.Get("/proposal/ksa", func(c *fiber.Ctx) error { return c.SendFile("./web/proposal/ksa.html") })
	app.Get("/proposal/ksa.pdf", func(c *fiber.Ctx) error {
		c.Set("Content-Disposition", `inline; filename="Umrohku-KSA-Proposal-PUAS-Nov2026.pdf"`)
		return c.SendFile("./web/proposal/ksa.pdf")
	})
	app.Static("/proposal/img", "./web/proposal/img", fiber.Static{MaxAge: 86400})
	app.Post("/demo/peran", DemoSwitchRole)
	app.Get("/inbox", DemoInbox)
	app.Post("/admin/demo/reset", DemoReset)

	registerDemoPublic(app)   // keuntungan, investor, mitra, S&K, tentang, profil travel
	registerDemoJamaah(app)   // cicil, booking + DP, pembiayaan, undian, ulasan
	registerDemoPortal(app)   // portal travel
	registerDemoAdmin(app)    // konsol bisnis admin
	registerDemoPartner(app)  // mitra pembiayaan
	registerDemoMandiri(app)  // Fase 3
	registerDemoTanahSuci(app) // Fase 4 + toko mitra
	registerDemoQRIS(app)      // bayar QRIS + navigasi bawah
	registerDemoMarket(app)    // market Tanah Suci ala GoFood
}

func DemoTour(c *fiber.Ctx) error {
	return demoRender(c, "demo/tour", viewer(c, fiber.Map{"Title": "Tur demo", "Roles": demo.RoleOptions(), "Draw": demo.OpenDraw()}))
}

// POST /demo/peran: ganti peran sekali klik (jamaah, travel:<slug>, admin, pembiayaan:<slug>, merchant:<slug>).
func DemoSwitchRole(c *fiber.Ctx) error {
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	v := c.FormValue("peran")
	kind, slug, _ := strings.Cut(v, ":")
	r := demo.Role{Kind: kind, Slug: slug}
	next := safeNext(c.FormValue("next", "/"))
	switch kind {
	case "jamaah":
		demo.LoginAs(c, demo.EnsureDemoUser())
		if strings.HasPrefix(next, "/portal") || strings.HasPrefix(next, "/admin") || strings.HasPrefix(next, "/mitra-pembiayaan") || strings.HasPrefix(next, "/toko") {
			next = "/"
		}
	case "travel":
		if demo.TravelBySlug(slug) == nil {
			return c.Redirect("/demo", fiber.StatusSeeOther)
		}
		next = "/portal"
	case "admin":
		grantAdmin(c)
		next = "/admin/bisnis"
	case "pembiayaan":
		if demo.PartnerBySlug(slug) == nil {
			return c.Redirect("/demo", fiber.StatusSeeOther)
		}
		next = "/mitra-pembiayaan"
	case "merchant":
		if demo.MerchantBySlug(slug) == nil {
			return c.Redirect("/demo", fiber.StatusSeeOther)
		}
		next = "/toko"
	default:
		r = demo.Role{}
	}
	demo.SetRole(c, r)
	// Peran bisnis tidak membawa sesi jamaah, supaya kepala halaman tidak menampilkan "Siti".
	if r.Kind != "jamaah" && demo.IsDemoUser(accounts.Current(c)) {
		accounts.DeleteSession(c.Cookies(accounts.SessionCookie))
		c.ClearCookie(accounts.SessionCookie)
	}
	if c.FormValue("goto") != "" {
		next = safeNext(c.FormValue("goto"))
	}
	return c.Redirect(next, fiber.StatusSeeOther)
}

// grantAdmin: sesi admin tanpa password (HANYA mode demo).
func grantAdmin(c *fiber.Ctx) {
	if !demo.Enabled() {
		return
	}
	exp := time.Now().Add(adminSession).Unix()
	c.Cookie(&fiber.Cookie{Name: adminCookie, Value: strconv.FormatInt(exp, 10) + "." + sign(exp), Path: "/", HTTPOnly: true,
		SameSite: "Lax", Secure: c.Protocol() == "https" || c.Get("X-Forwarded-Proto") == "https", Expires: time.Unix(exp, 0)})
}

func DemoInbox(c *fiber.Ctx) error {
	aud := inboxAudience(c)
	msgs := demo.Inbox(aud, 100)
	unread := map[uint]bool{}
	for _, m := range msgs {
		if !m.Read {
			unread[m.ID] = true
		}
	}
	demo.MarkRead(aud)
	return demoRender(c, "demo/inbox", viewer(c, fiber.Map{"Title": "Notifikasi", "Messages": msgs, "Unread": unread,
		"Audience": aud, "RoleLabel": demo.CurrentRole(c).Label()}))
}

func DemoReset(c *fiber.Ctx) error {
	if !isAdmin(c) || !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	demo.Reset()
	return c.Redirect("/admin/bisnis?reset=1", fiber.StatusSeeOther)
}

// requireRole: peran demo aktif harus kind (dan mengembalikan slug-nya).
func requireRole(c *fiber.Ctx, kind string) (demo.Role, bool) {
	r := demo.CurrentRole(c)
	return r, r.Kind == kind
}

// demoUser: jamaah yang login (akun Google sungguhan atau akun demo).
func demoUser(c *fiber.Ctx) *accounts.User { return accounts.Current(c) }

func atoiDef(s string, def int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return n
	}
	return def
}

// rupiahInput: angka dari input bebas ("27.500.000", "Rp 500rb" -> hanya digit); 0 bila kosong/terlalu besar.
func rupiahInput(s string) int64 {
	var n int64
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n = n*10 + int64(r-'0')
			if n > 100_000_000_000 {
				return 0
			}
		}
	}
	return n
}

// demoRender memilih layout: halaman bisnis (portal/admin/mitra) memakai demo/layout_biz, halaman
// jamaah & publik memakai demo/layout, sedangkan potongan htmx & halaman cetak tanpa layout.
func demoRender(c *fiber.Ctx, name string, m fiber.Map) error {
	base := strings.TrimPrefix(name, "demo/")
	switch {
	case base == "kuitansi" || base == "invoice_print" || base == "cicil" || base == "booking_summary":
		return c.Render(name, m)
	case strings.HasPrefix(base, "portal_") || strings.HasPrefix(base, "admin_") || base == "partner":
		return c.Render(name, m, "demo/layout_biz")
	}
	return c.Render(name, m, "demo/layout")
}
