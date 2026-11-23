package handlers

import (
	"strings"
	"time"

	"umrah/app/accounts"
	"umrah/app/analytics"
	"umrah/app/demo"
	"umrah/app/repositories"
	"umrah/app/services"

	"github.com/gofiber/fiber/v2"
)

// Halaman publik demo: keuntungan, roadmap/investor, pendaftaran mitra, S&K, tentang, profil travel,
// simulasi chat WA travel fiktif, dan pengganti login Google.

func registerDemoPublic(app *fiber.App) {
	// Login Google tidak dipakai di demo: tawarkan masuk sebagai jamaah demo.
	app.Get("/auth/google/login", DemoLoginPage)
	app.Get("/keuntungan", DemoPage("keuntungan", "Keuntungan pesan lewat umrohku"))
	app.Get("/investor", DemoInvestor)
	app.Get("/syarat-ketentuan", DemoPage("syarat", "Syarat & Ketentuan"))
	app.Get("/tentang", DemoPage("tentang", "Tentang umrohku"))
	app.Get("/mitra", DemoPartnerForm)
	app.Post("/mitra", DemoPartnerSubmit)
	app.Get("/travel/:slug", DemoTravelProfile)
	app.Get("/demo/wa", DemoWAChat)
	app.Post("/demo/wa", DemoWASend)
}

func DemoLoginPage(c *fiber.Ctx) error {
	return demoRender(c, "demo/login", viewer(c, fiber.Map{"Title": "Masuk", "Next": safeNext(c.Query("next", "/"))}))
}

func DemoPage(tpl, title string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return demoRender(c, "demo/"+tpl, viewer(c, fiber.Map{"Title": title, "Draw": demo.OpenDraw(), "Travels": demo.PartnerTravels()}))
	}
}

func DemoInvestor(c *fiber.Ctx) error {
	r := demo.BuildRevenue(demo.CurrentPeriod())
	prev := demo.BuildRevenue(demo.PrevPeriod(demo.CurrentPeriod()))
	return demoRender(c, "demo/investor", viewer(c, fiber.Map{"Title": "Roadmap & demo", "Rev": r, "Prev": prev,
		"Import": repositories.LastImport(), "Travels": len(demo.AllTravels())}))
}

func DemoPartnerForm(c *fiber.Ctx) error {
	return demoRender(c, "demo/mitra", viewer(c, fiber.Map{"Title": "Daftar jadi travel mitra", "Sent": c.Query("terkirim") == "1", "Plans": demo.Cfg.Plans}))
}

func DemoPartnerSubmit(c *fiber.Ctx) error {
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	s := demo.PartnerSignup{Company: trunc(c.FormValue("company"), 160), Brand: trunc(c.FormValue("brand"), 120), PPIUNo: trunc(c.FormValue("ppiu"), 80),
		City: trunc(c.FormValue("city"), 80), PIC: trunc(c.FormValue("pic"), 80), Phone: trunc(c.FormValue("phone"), 30),
		Email: trunc(c.FormValue("email"), 120), Note: trunc(c.FormValue("note"), 500), Status: "baru", CreatedAt: time.Now()}
	if s.Company == "" || s.PIC == "" || len(s.Phone) < 8 {
		return demoRender(c, "demo/mitra", viewer(c, fiber.Map{"Title": "Daftar jadi travel mitra", "Error": "Nama PT, PIC, dan nomor WhatsApp wajib diisi.", "Form": s, "Plans": demo.Cfg.Plans}))
	}
	demo.DB.Create(&s)
	demo.Notify("admin", "email", "admin", "Pendaftaran mitra: "+s.Company, s.PIC+" ("+s.City+") mendaftar sebagai travel mitra. No. izin PPIU yang diisi: "+s.PPIUNo+". Cek ke data resmi Kemenhaj sebelum diaktifkan.", "/admin/bisnis/mitra")
	return c.Redirect("/mitra?terkirim=1", fiber.StatusSeeOther)
}

func DemoTravelProfile(c *fiber.Ctx) error {
	t := demo.TravelBySlug(c.Params("slug"))
	if t == nil {
		return c.Redirect("/")
	}
	avg, n := demo.ReviewStats(t.Slug)
	today := demo.Today()
	type pkgView struct {
		Name, Key, Airline string
		Price, Duration   int
		Dates             []string
		Hotel             string
	}
	var pkgs []pkgView
	for _, p := range repositories.AllPackages() {
		if p.Travel.Name != t.Name {
			continue
		}
		v := pkgView{Name: p.Name, Key: services.PackageKey(p.Travel.Name, p.Name), Price: p.Price, Duration: p.Duration, Airline: p.Airline}
		for _, d := range p.Details {
			if d.DepartureDate >= today && !d.IsFull && len(v.Dates) < 6 {
				v.Dates = append(v.Dates, d.DepartureDate)
			}
			if v.Hotel == "" && d.HotelMakkah != "" {
				v.Hotel = d.HotelMakkah + " · " + d.HotelMadinah
			}
		}
		if len(v.Dates) > 0 {
			pkgs = append(pkgs, v)
		}
	}
	return demoRender(c, "demo/travel", viewer(c, fiber.Map{"Title": t.Name, "T": t, "Terms": demo.TermsFor(t.Slug), "Reviews": demo.Reviews(t.Slug),
		"Avg": avg, "N": n, "Packages": pkgs}))
}

// Simulasi chat WhatsApp ke travel fiktif: pesan masuk ke Notifikasi travel (bukan WA sungguhan).
func DemoWAChat(c *fiber.Ctx) error {
	t := demo.TravelBySlug(c.Query("travel"))
	if t == nil {
		return c.Redirect("/")
	}
	m := fiber.Map{"Title": "Chat " + t.Name, "T": t, "Pkg": trunc(c.Query("pkg"), 200), "Sent": c.Query("terkirim") == "1"}
	if u := accounts.Current(c); u != nil {
		m["Name"] = u.Name
	}
	return demoRender(c, "demo/wa", viewer(c, m))
}

func DemoWASend(c *fiber.Ctx) error {
	t := demo.TravelBySlug(c.FormValue("travel"))
	if t == nil || !accounts.SameOrigin(c) {
		return c.Redirect("/", fiber.StatusSeeOther)
	}
	pkg := trunc(c.FormValue("pkg"), 200)
	back := "/demo/wa?travel=" + t.Slug + "&pkg=" + urlq(pkg)
	name, phone := trunc(c.FormValue("name"), 80), trunc(c.FormValue("phone"), 30)
	if name == "" || len(phone) < 8 {
		return c.Redirect(back, fiber.StatusSeeOther)
	}
	visitor, _, device, _ := analytics.FromRequest(c)
	demo.RecordLead(demo.Lead{TravelSlug: t.Slug, Package: pkg, Kind: "form", Visitor: visitor, Device: device, Name: name, Phone: phone,
		City: trunc(c.FormValue("city"), 60), Source: "umrohku"})
	return c.Redirect(back+"&terkirim=1", fiber.StatusSeeOther)
}

func trunc(s string, n int) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
