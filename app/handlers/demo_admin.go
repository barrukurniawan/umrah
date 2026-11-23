package handlers

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"umrah/app/accounts"
	"umrah/app/demo"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type gormDB = gorm.DB

// Konsol bisnis admin umrohku (/admin/bisnis): pendapatan per sumber, PPIU & mitra, Promo
// Spesial, promo diskon/cashback, undian, minat cicil, pembiayaan, booking.

func registerDemoAdmin(app *fiber.App) {
	g := app.Group("/admin/bisnis", func(c *fiber.Ctx) error {
		if !isAdmin(c) {
			return demoRender(c, "demo/role_needed", viewer(c, fiber.Map{"Title": "Konsol admin", "Need": "admin", "Roles": demo.RoleOptions()}))
		}
		return c.Next()
	})
	g.Get("/", AdminBizSummary)
	g.Get("/mitra", AdminBizPartners)
	g.Post("/mitra/:slug/ppiu", AdminBizVerifyPPIU)
	g.Post("/mitra/:slug/terms", AdminBizTerms)
	g.Post("/pendaftar/:id<int>", AdminBizSignupStatus)
	g.Get("/promo", AdminBizPromo)
	g.Post("/promo/posisi", AdminBizPlacementSave)
	g.Post("/promo/posisi/:id<int>/hapus", AdminBizPlacementDelete)
	g.Post("/promo/aturan", AdminBizPromoSave)
	g.Post("/promo/aturan/:id<int>/aktif", AdminBizPromoToggle)
	g.Get("/undian", AdminBizDraws)
	g.Post("/undian", AdminBizDrawCreate)
	g.Post("/undian/:id<int>/status", AdminBizDrawStatus)
	g.Post("/undian/:id<int>/undi", AdminBizDrawRun)
	g.Get("/undian/:id<int>/berita-acara.csv", AdminBizDrawCSV)
	g.Get("/cicil", AdminBizCicil)
	g.Get("/pembiayaan", AdminBizFinancing)
	g.Get("/booking", AdminBizBookings)
}

func adminTabsBiz(active string) []Tab {
	t := tabs(active, "/admin/bisnis", "Pendapatan", "/admin/bisnis/mitra", "Mitra & PPIU", "/admin/bisnis/promo", "Promo", "/admin/bisnis/undian", "Undian",
		"/admin/bisnis/cicil", "Minat cicil", "/admin/bisnis/pembiayaan", "Pembiayaan", "/admin/bisnis/booking", "Booking", "/admin", "Analitik situs ↗")
	var n int64
	demo.DB.Model(&demo.PartnerSignup{}).Where("status = ?", "baru").Count(&n)
	t[1].Count = int(n)
	return t
}

func adminRender(c *fiber.Ctx, tpl, active, title string, m fiber.Map) error {
	m["Tabs"], m["Title"], m["Badge"], m["Sub"] = adminTabsBiz(active), title, "Konsol admin umrohku", "Data contoh · "+demo.PeriodLabel(demo.CurrentPeriod())
	return demoRender(c, "demo/"+tpl, viewer(c, m))
}

func AdminBizSummary(c *fiber.Ctx) error {
	cur := demo.CurrentPeriod()
	prev := demo.PrevPeriod(cur)
	var months []demo.Revenue
	p := cur
	for i := 0; i < 3; i++ {
		months = append([]demo.Revenue{demo.BuildRevenue(p)}, months...)
		p = demo.PrevPeriod(p)
	}
	var mx int64 = 1
	for _, r := range months {
		mx = max(mx, r.Total)
	}
	var bookings, users int64
	demo.DB.Model(&demo.Booking{}).Where("status <> ?", "batal").Count(&bookings)
	accounts.DB.Model(&accounts.User{}).Count(&users)
	return adminRender(c, "admin_ringkasan", "/admin/bisnis", "Konsol bisnis", fiber.Map{
		"Rev": demo.BuildRevenue(cur), "Prev": demo.BuildRevenue(prev), "Months": months, "Max": mx, "Bookings": bookings, "Users": users,
		"Inbox": demo.Inbox("admin", 6), "Reset": c.Query("reset") == "1", "Cfg": demo.Cfg,
	})
}

func AdminBizPartners(c *fiber.Ctx) error {
	var signups []demo.PartnerSignup
	demo.DB.Order("created_at DESC").Find(&signups)
	terms := map[string]*demo.TravelTerms{}
	for _, t := range demo.AllTravels() {
		terms[t.Slug] = demo.TermsFor(t.Slug)
	}
	// Travel asli hasil crawl: belum ada data PPIU yang dicek (tidak ada badge).
	known := map[string]bool{}
	for _, t := range demo.AllTravels() {
		known[t.Name] = true
	}
	var real []string
	for _, n := range crawledTravelNames() {
		if !known[n] {
			real = append(real, n)
		}
	}
	return adminRender(c, "admin_mitra", "/admin/bisnis/mitra", "Mitra & verifikasi PPIU", fiber.Map{"Travels": demo.AllTravels(), "Terms": terms,
		"Signups": signups, "Real": real, "Msg": c.Query("msg")})
}

func AdminBizVerifyPPIU(c *fiber.Ctx) error {
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	now := time.Now()
	slug := c.Params("slug")
	st := c.FormValue("status", "terverifikasi")
	if st != "terverifikasi" && st != "belum" && st != "kedaluwarsa" {
		st = "belum"
	}
	demo.DB.Model(&demo.Travel{}).Where("slug = ?", slug).Updates(map[string]any{"ppiu_status": st, "ppiu_checked_at": &now})
	demo.WritePackagesFile()
	return c.Redirect("/admin/bisnis/mitra?msg="+urlq("Status PPIU diperbarui · dicek "+demo.TglJam(now)), fiber.StatusSeeOther)
}

func AdminBizTerms(c *fiber.Ctx) error {
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	slug := c.Params("slug")
	var tm demo.TravelTerms
	if demo.DB.Where("travel_slug = ?", slug).Take(&tm).Error != nil {
		return c.Redirect("/admin/bisnis/mitra", fiber.StatusSeeOther)
	}
	if v := rupiahInput(c.FormValue("dp_min")); v >= 100_000 && v <= 50_000_000 {
		tm.DPMinUmrohku = v
	}
	if v := rupiahInput(c.FormValue("dp_normal")); v >= 100_000 && v <= 50_000_000 {
		tm.DPNormalTravel = v
	}
	if v := atoiDef(c.FormValue("h"), 0); v >= 14 && v <= 120 {
		tm.PelunasanHMinus = v
	}
	if v := c.FormValue("komisi"); v != "" {
		var f float64
		if _, err := fmt.Sscanf(strings.Replace(v, ",", ".", 1), "%g", &f); err == nil && f >= 0 && f <= 15 {
			tm.KomisiPersen = f
		}
	}
	demo.DB.Save(&tm)
	demo.WritePackagesFile()
	demo.ReloadPackages()
	return c.Redirect("/admin/bisnis/mitra?msg="+urlq("Ketentuan perjanjian diperbarui"), fiber.StatusSeeOther)
}

func AdminBizSignupStatus(c *fiber.Ctx) error {
	if accounts.SameOrigin(c) {
		st := c.FormValue("status")
		if st == "dihubungi" || st == "diverifikasi" || st == "baru" {
			demo.DB.Model(&demo.PartnerSignup{}).Where("id = ?", idParam(c, "id")).Update("status", st)
		}
	}
	return c.Redirect("/admin/bisnis/mitra#pendaftar", fiber.StatusSeeOther)
}

// ---- promo ----

func AdminBizPromo(c *fiber.Ctx) error {
	var travels []string
	for _, t := range demo.AllTravels() {
		travels = append(travels, t.Name)
	}
	travels = append(travels, crawledTravelNames()...)
	travels = uniq(travels)
	var dpUsed, cbUsed int64
	demo.DB.Model(&demo.PromoUse{}).Where("kind = ? AND status <> ?", "diskon_dp", "batal").Select("COALESCE(SUM(amount),0)").Scan(&dpUsed)
	demo.DB.Model(&demo.PromoUse{}).Where("kind = ? AND status <> ?", "cashback_pelunasan", "batal").Select("COALESCE(SUM(amount),0)").Scan(&cbUsed)
	return adminRender(c, "admin_promo", "/admin/bisnis/promo", "Promo", fiber.Map{"Placements": demo.Placements(), "Rules": demo.PromoRules(),
		"Monthly": demo.PromoMonthly(6), "TravelNames": travels, "Partners": demo.PartnerTravels(), "Today": demo.Today(),
		"DPUsed": dpUsed, "CBUsed": cbUsed, "Err": c.Query("err"), "Msg": c.Query("msg")})
}

func AdminBizPlacementSave(c *fiber.Ctx) error {
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	p := demo.Placement{TravelName: trunc(c.FormValue("travel"), 120), MinBudget: int(rupiahInput(c.FormValue("min"))), MaxBudget: int(rupiahInput(c.FormValue("max"))),
		StartsAt: c.FormValue("mulai"), EndsAt: c.FormValue("selesai"), PricePerDay: rupiahInput(c.FormValue("harga")), Active: true}
	if p.TravelName == "" || p.MinBudget < 1_000_000 || p.MaxBudget < p.MinBudget || len(p.StartsAt) != 10 || len(p.EndsAt) != 10 || p.EndsAt < p.StartsAt {
		return c.Redirect("/admin/bisnis/promo?err="+urlq("Lengkapi travel, rentang budget, dan tanggal (selesai ≥ mulai)."), fiber.StatusSeeOther)
	}
	demo.SavePlacement(&p)
	if t := demo.TravelByName(p.TravelName); t != nil {
		demo.Notify("travel:"+t.Slug, "email", t.PIC, "Posisi Promo Spesial aktif",
			fmt.Sprintf("Paket Anda tampil paling atas dengan label \"Promo Spesial\" untuk budget Rp %s – %s, %s s.d. %s. Biaya Rp %s/hari masuk tagihan bulanan.",
				demo.RpShort(int64(p.MinBudget)), demo.RpShort(int64(p.MaxBudget)), demo.Tgl(p.StartsAt), demo.Tgl(p.EndsAt), demo.Rp(p.PricePerDay)), "/portal/promo")
	}
	return c.Redirect("/admin/bisnis/promo?msg="+urlq("Posisi unggulan disimpan dan langsung berlaku di pencarian"), fiber.StatusSeeOther)
}

func AdminBizPlacementDelete(c *fiber.Ctx) error {
	if accounts.SameOrigin(c) {
		demo.DeletePlacement(idParam(c, "id"))
	}
	return c.Redirect("/admin/bisnis/promo", fiber.StatusSeeOther)
}

func AdminBizPromoSave(c *fiber.Ctx) error {
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	start, err1 := time.ParseInLocation("2006-01-02", c.FormValue("mulai"), demo.WIB)
	end, err2 := time.ParseInLocation("2006-01-02", c.FormValue("selesai"), demo.WIB)
	r := demo.PromoRule{Code: strings.ToUpper(trunc(c.FormValue("kode"), 40)), Title: trunc(c.FormValue("judul"), 120), Kind: c.FormValue("jenis"),
		Amount: rupiahInput(c.FormValue("nominal")), Cap: rupiahInput(c.FormValue("maks")), TravelSlug: c.FormValue("travel"),
		MinPrice: rupiahInput(c.FormValue("min")), Quota: atoiDef(c.FormValue("kuota"), 0), StartsAt: start, EndsAt: end.Add(24*time.Hour - time.Second),
		FundedBy: "umrohku", Active: true}
	if pct := c.FormValue("persen"); pct != "" {
		fmt.Sscanf(strings.Replace(pct, ",", ".", 1), "%g", &r.Percent)
	}
	switch {
	case r.Kind != "diskon_dp" && r.Kind != "cashback_pelunasan":
		return c.Redirect("/admin/bisnis/promo?err="+urlq("Pilih jenis promo"), fiber.StatusSeeOther)
	case err1 != nil || err2 != nil || !end.After(start):
		return c.Redirect("/admin/bisnis/promo?err="+urlq("Tanggal promo tidak valid"), fiber.StatusSeeOther)
	case (r.Amount <= 0 && r.Percent <= 0) || r.Percent > 10 || r.Amount > 5_000_000:
		return c.Redirect("/admin/bisnis/promo?err="+urlq("Isi nominal (≤ Rp 5 jt) atau persen (≤ 10%)"), fiber.StatusSeeOther)
	case len(r.Title) < 5:
		return c.Redirect("/admin/bisnis/promo?err="+urlq("Isi judul promo"), fiber.StatusSeeOther)
	}
	if r.Code != "" {
		var n int64
		demo.DB.Model(&demo.PromoRule{}).Where("code = ?", r.Code).Count(&n)
		if n > 0 {
			return c.Redirect("/admin/bisnis/promo?err="+urlq("Kode "+r.Code+" sudah dipakai"), fiber.StatusSeeOther)
		}
	}
	demo.DB.Create(&r)
	return c.Redirect("/admin/bisnis/promo?msg="+urlq("Promo dibuat"), fiber.StatusSeeOther)
}

func AdminBizPromoToggle(c *fiber.Ctx) error {
	if accounts.SameOrigin(c) {
		demo.DB.Model(&demo.PromoRule{}).Where("id = ?", idParam(c, "id")).Update("active", c.FormValue("aktif") == "1")
	}
	return c.Redirect("/admin/bisnis/promo", fiber.StatusSeeOther)
}

// ---- undian ----

func AdminBizDraws(c *fiber.Ctx) error {
	return adminRender(c, "admin_undian", "/admin/bisnis/undian", "Undian", fiber.Map{"Draws": demo.Draws(), "Err": c.Query("err"), "Msg": c.Query("msg"),
		"DefStart": demo.Today(), "DefEnd": demo.AddDays(demo.Today(), 90), "DefDraw": demo.AddDays(demo.Today(), 97)})
}

func AdminBizDrawCreate(c *fiber.Ctx) error {
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	s, _ := time.ParseInLocation("2006-01-02", c.FormValue("mulai"), demo.WIB)
	e, _ := time.ParseInLocation("2006-01-02", c.FormValue("selesai"), demo.WIB)
	dr, _ := time.ParseInLocation("2006-01-02", c.FormValue("undi"), demo.WIB)
	d := demo.Draw{Title: trunc(c.FormValue("judul"), 150), PrizeDesc: trunc(c.FormValue("hadiah"), 300), Winners: atoiDef(c.FormValue("pemenang"), 1),
		StartsAt: s, EndsAt: e.Add(24*time.Hour - time.Second), DrawAt: dr.Add(10 * time.Hour), PermitNo: "dalam proses", Terms: demoDrawTerms(), Status: "draft"}
	if len(d.Title) < 5 || len(d.PrizeDesc) < 5 {
		return c.Redirect("/admin/bisnis/undian?err="+urlq("Isi judul dan hadiah"), fiber.StatusSeeOther)
	}
	if err := demo.SaveDraw(&d); err != nil {
		return c.Redirect("/admin/bisnis/undian?err="+urlq(err.Error()), fiber.StatusSeeOther)
	}
	return c.Redirect("/admin/bisnis/undian?msg="+urlq("Undian dibuat sebagai draf"), fiber.StatusSeeOther)
}

func demoDrawTerms() string {
	for _, d := range demo.Draws() {
		if d.Terms != "" {
			return d.Terms
		}
	}
	return ""
}

func AdminBizDrawStatus(c *fiber.Ctx) error {
	if accounts.SameOrigin(c) {
		if err := demo.SetDrawStatus(idParam(c, "id"), c.FormValue("status")); err != nil {
			return c.Redirect("/admin/bisnis/undian?err="+urlq(err.Error()), fiber.StatusSeeOther)
		}
	}
	return c.Redirect("/admin/bisnis/undian", fiber.StatusSeeOther)
}

func AdminBizDrawRun(c *fiber.Ctx) error {
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	id := idParam(c, "id")
	if err := demo.RunDraw(id, c.FormValue("seed"), c.FormValue("saksi")); err != nil {
		return c.Redirect("/admin/bisnis/undian?err="+urlq(err.Error()), fiber.StatusSeeOther)
	}
	return c.Redirect(fmt.Sprintf("/undian/%d#hasil", id), fiber.StatusSeeOther)
}

func AdminBizDrawCSV(c *fiber.Ctx) error {
	d, err := demo.GetDraw(idParam(c, "id"))
	if err != nil {
		return c.SendStatus(404)
	}
	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="berita-acara-undian-%d.csv"`, d.ID))
	demo.WriteMinutesCSV(c.Response().BodyWriter(), d)
	return nil
}

// ---- minat cicil, pembiayaan, booking ----

func AdminBizCicil(c *fiber.Ctx) error {
	var rows []demo.CicilIntent
	demo.DB.Where("created_at >= ?", time.Now().AddDate(0, 0, -30)).Find(&rows)
	paths := map[string]int{}
	tenors := map[int]int{}
	byTravel := map[string]int{}
	var sumPrice int64
	for _, r := range rows {
		paths[r.Path]++
		if r.Tenor > 0 {
			tenors[r.Tenor]++
		}
		byTravel[r.Travel]++
		sumPrice += r.Price
	}
	type kv struct {
		K   string
		N   int
		Pct int
	}
	total := max(1, len(rows))
	var pathRows, travelRows []kv
	for _, k := range []string{"sebelum", "pembiayaan"} {
		pathRows = append(pathRows, kv{k, paths[k], paths[k] * 100 / total})
	}
	for k, n := range byTravel {
		travelRows = append(travelRows, kv{k, n, n * 100 / total})
	}
	sort.Slice(travelRows, func(i, j int) bool { return travelRows[i].N > travelRows[j].N })
	var tenorRows []kv
	tt := max(1, tenors[6]+tenors[12]+tenors[24])
	for _, t := range []int{6, 12, 24} {
		tenorRows = append(tenorRows, kv{fmt.Sprintf("%d bulan", t), tenors[t], tenors[t] * 100 / tt})
	}
	avg := int64(0)
	if len(rows) > 0 {
		avg = sumPrice / int64(len(rows))
	}
	var recent []demo.CicilIntent
	demo.DB.Order("created_at DESC").Limit(15).Find(&recent)
	return adminRender(c, "admin_cicil", "/admin/bisnis/cicil", "Minat cicilan", fiber.Map{"Total": len(rows), "Paths": pathRows, "Tenors": tenorRows,
		"Travels": travelRows, "Avg": avg, "Recent": recent})
}

func AdminBizFinancing(c *fiber.Ctx) error {
	apps := demo.AllApplications()
	counts := map[string]int{}
	var fee, disbursed int64
	for _, a := range apps {
		counts[a.Status]++
		fee += a.ReferralFee
		if a.DisbursedAt != nil {
			disbursed += a.Amount
		}
	}
	return adminRender(c, "admin_pembiayaan", "/admin/bisnis/pembiayaan", "Rujukan pembiayaan", fiber.Map{"Apps": apps, "Counts": counts, "Fee": fee,
		"Disbursed": disbursed, "Partners": demo.Cfg.Partners})
}

func AdminBizBookings(c *fiber.Ctx) error {
	var bs []demo.Booking
	q := demo.DB.Order("created_at DESC").Limit(200)
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	q.Find(&bs)
	return adminRender(c, "admin_booking", "/admin/bisnis/booking", "Semua booking", fiber.Map{"Bookings": bs, "Status": c.Query("status")})
}

// ---- util ----

func crawledTravelNames() []string {
	var out []string
	seen := map[string]bool{}
	for _, t := range repoTravelNames() {
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	sort.Strings(out)
	return out
}

func uniq(s []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range s {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}
