package handlers

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"umrah/app/accounts"
	"umrah/app/demo"

	"github.com/gofiber/fiber/v2"
)

// Portal travel mitra (#2, #3, #5, #6): dashboard, paket & sisa seat, lead, booking, tagihan,
// langganan, profil. Akses lewat peran demo "travel:<slug>".

func registerDemoPortal(app *fiber.App) {
	g := app.Group("/portal", portalAuth)
	g.Get("/", PortalDashboard)
	g.Get("/paket", PortalPackages)
	g.Get("/paket/baru", PortalPackageForm)
	g.Get("/paket/:id<int>", PortalPackageForm)
	g.Post("/paket", PortalPackageSave)
	g.Post("/paket/:id<int>/seat", PortalSeats)
	g.Post("/paket/:id<int>/aktif", PortalPackageToggle)
	g.Get("/lead", PortalLeads)
	g.Get("/lead.csv", PortalLeadsCSV)
	g.Post("/lead/:id<int>", PortalLeadStatus)
	g.Get("/booking", PortalBookings)
	g.Post("/booking/:id<int>/status", PortalBookingStatus)
	g.Get("/tagihan", PortalInvoice)
	g.Post("/tagihan/bayar", PortalInvoicePay)
	g.Get("/langganan", PortalPlans)
	g.Post("/langganan", PortalSubscribe)
	g.Get("/promo", PortalPromo)
	g.Get("/profil", PortalProfile)
}

func portalAuth(c *fiber.Ctx) error {
	r, ok := requireRole(c, "travel")
	if !ok || demo.TravelBySlug(r.Slug) == nil {
		return demoRender(c, "demo/role_needed", viewer(c, fiber.Map{"Title": "Portal travel mitra", "Need": "travel", "Roles": demo.RoleOptions()}))
	}
	c.Locals("travel", demo.TravelBySlug(r.Slug))
	return c.Next()
}

func curTravel(c *fiber.Ctx) *demo.Travel { t, _ := c.Locals("travel").(*demo.Travel); return t }

func portalTabs(active string, t *demo.Travel) []Tab {
	ts := tabs(active, "/portal", "Ringkasan", "/portal/paket", "Paket & seat", "/portal/lead", "Lead", "/portal/booking", "Booking",
		"/portal/tagihan", "Tagihan", "/portal/langganan", "Langganan", "/portal/promo", "Promo Spesial", "/portal/profil", "Profil & PPIU")
	var n int64
	demo.DB.Model(&demo.Lead{}).Where("travel_slug = ? AND status = ? AND billable = ?", t.Slug, "baru", true).Count(&n)
	ts[2].Count = int(n)
	return ts
}

func portalRender(c *fiber.Ctx, tpl, active, title string, m fiber.Map) error {
	t := curTravel(c)
	m["T"], m["Tabs"], m["Title"], m["Plan"] = t, portalTabs(active, t), title, demo.PlanByKey(t.Plan)
	m["Badge"], m["Sub"] = "Portal travel mitra", t.Name+" · paket "+demo.PlanByKey(t.Plan).Name
	if t.PPIUStatus == "terverifikasi" {
		m["Sub"] = m["Sub"].(string) + " · PPIU terverifikasi"
	}
	return demoRender(c, "demo/"+tpl, viewer(c, m))
}

func PortalDashboard(c *fiber.Ctx) error {
	t := curTravel(c)
	month := demo.CurrentPeriod()
	inv := demo.BuildInvoice(*t, month)
	var leads30, leadsPrev, bookings30, closing int64
	since := time.Now().AddDate(0, 0, -30)
	demo.DB.Model(&demo.Lead{}).Where("travel_slug = ? AND billable = ? AND created_at >= ?", t.Slug, true, since).Count(&leads30)
	demo.DB.Model(&demo.Lead{}).Where("travel_slug = ? AND billable = ? AND created_at >= ? AND created_at < ?", t.Slug, true, since.AddDate(0, 0, -30), since).Count(&leadsPrev)
	demo.DB.Model(&demo.Booking{}).Where("travel_slug = ? AND paid_at >= ? AND status <> ?", t.Slug, since, "batal").Count(&bookings30)
	demo.DB.Model(&demo.Lead{}).Where("travel_slug = ? AND status = ? AND created_at >= ?", t.Slug, "closing", since).Count(&closing)
	var gmv int64
	demo.DB.Model(&demo.Booking{}).Where("travel_slug = ? AND paid_at >= ? AND status <> ?", t.Slug, since, "batal").Select("COALESCE(SUM(total),0)").Scan(&gmv)
	growth := 0
	if leadsPrev > 0 {
		growth = int((leads30 - leadsPrev) * 100 / leadsPrev)
	}
	avg, nrev := demo.ReviewStats(t.Slug)
	return portalRender(c, "portal_dashboard", "/portal", "Ringkasan", fiber.Map{
		"Leads30": leads30, "Growth": growth, "Bookings30": bookings30, "Closing": closing, "GMV": gmv, "Inv": inv,
		"Daily": demo.LeadDaily(t.Slug, 30), "Stats": demo.PackageStats(t.Slug, 30), "Recent": demo.Leads(demo.LeadFilter{Slug: t.Slug, Limit: 6}),
		"RecentBookings": firstN(demo.TravelBookings(t.Slug, ""), 5), "Avg": avg, "NRev": nrev, "Placement": demo.ActivePlacement(t.Name),
		"Inbox": demo.Inbox("travel:"+t.Slug, 4),
	})
}

func firstN[T any](s []T, n int) []T {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// ---- paket & seat ----

func PortalPackages(c *fiber.Ctx) error {
	t := curTravel(c)
	var pkgs []demo.PortalPackage
	demo.DB.Preload("Departures", func(db *gormDB) *gormDB { return db.Order("date") }).Where("travel_slug = ?", t.Slug).Order("active DESC, price").Find(&pkgs)
	return portalRender(c, "portal_paket", "/portal/paket", "Paket & sisa seat", fiber.Map{"Pkgs": pkgs, "Today": demo.Today(), "Saved": c.Query("tersimpan") == "1"})
}

func PortalPackageForm(c *fiber.Ctx) error {
	t := curTravel(c)
	p := demo.PortalPackage{TravelSlug: t.Slug, Duration: 9, StarsMakkah: 4, StarsMadinah: 4, Direct: true, Active: true,
		Facilities: "Tiket pesawat PP\nVisa umroh\nHotel sesuai paket\nMakan 3x sehari\nMuthawwif & pembimbing\nManasik", Excludes: "Paspor\nVaksin meningitis"}
	if id := idParam(c, "id"); id > 0 {
		if demo.DB.Preload("Departures", func(db *gormDB) *gormDB { return db.Order("date") }).Where("id = ? AND travel_slug = ?", id, t.Slug).Take(&p).Error != nil {
			return c.Redirect("/portal/paket")
		}
	}
	return portalRender(c, "portal_paket_form", "/portal/paket", "Ubah paket", fiber.Map{"P": p, "Error": c.Query("err")})
}

func PortalPackageSave(c *fiber.Ctx) error {
	t := curTravel(c)
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	var p demo.PortalPackage
	id := uint(atoiDef(c.FormValue("id"), 0))
	if id > 0 && demo.DB.Where("id = ? AND travel_slug = ?", id, t.Slug).Take(&p).Error != nil {
		return c.Redirect("/portal/paket", fiber.StatusSeeOther)
	}
	p.TravelSlug = t.Slug
	p.Name = trunc(c.FormValue("name"), 200)
	p.Price, p.PriceTriple, p.PriceDouble = rupiahInput(c.FormValue("price")), rupiahInput(c.FormValue("triple")), rupiahInput(c.FormValue("double"))
	p.Duration = max(1, min(45, atoiDef(c.FormValue("duration"), 9)))
	p.Airline, p.TransitVia = trunc(c.FormValue("airline"), 80), trunc(c.FormValue("via"), 60)
	p.Direct = c.FormValue("direct") == "1"
	p.HotelMakkah, p.HotelMadinah = trunc(c.FormValue("hm"), 120), trunc(c.FormValue("hd"), 120)
	p.StarsMakkah, p.StarsMadinah = max(0, min(5, atoiDef(c.FormValue("sm"), 0))), max(0, min(5, atoiDef(c.FormValue("sd"), 0)))
	p.DistMakkah, p.DistMadinah = max(0, min(20000, atoiDef(c.FormValue("dm"), 0))), max(0, min(20000, atoiDef(c.FormValue("dd"), 0)))
	p.Facilities, p.Excludes, p.Guide = trunc(c.FormValue("facilities"), 2000), trunc(c.FormValue("excludes"), 1000), trunc(c.FormValue("guide"), 120)
	p.Active, p.UpdatedAt = true, time.Now()
	back := "/portal/paket/baru"
	if id > 0 {
		back = fmt.Sprintf("/portal/paket/%d", id)
	}
	if len(p.Name) < 5 || p.Price < 5_000_000 || p.Price > 500_000_000 || p.Airline == "" {
		return c.Redirect(back+"?err="+urlq("Isi nama paket, maskapai, dan harga Rp 5 jt – 500 jt."), fiber.StatusSeeOther)
	}
	demo.DB.Save(&p)
	// Tanggal baru: "2027-01-15" + total seat.
	if d := strings.TrimSpace(c.FormValue("new_date")); len(d) == 10 && d > demo.Today() {
		seats := max(1, min(500, atoiDef(c.FormValue("new_seats"), 45)))
		demo.DB.Create(&demo.PortalDeparture{PackageID: p.ID, Date: d, ReturnDate: demo.AddDays(d, p.Duration-1), SeatsTotal: seats, SeatsLeft: seats})
	}
	demo.ReloadPackages()
	return c.Redirect("/portal/paket?tersimpan=1#paket-"+fmt.Sprint(p.ID), fiber.StatusSeeOther)
}

// POST /portal/paket/:id/seat: ubah sisa seat (dan harga khusus) per tanggal sekaligus.
func PortalSeats(c *fiber.Ctx) error {
	t := curTravel(c)
	var p demo.PortalPackage
	if !accounts.SameOrigin(c) || demo.DB.Preload("Departures").Where("id = ? AND travel_slug = ?", idParam(c, "id"), t.Slug).Take(&p).Error != nil {
		return c.Redirect("/portal/paket", fiber.StatusSeeOther)
	}
	for _, d := range p.Departures {
		if v := c.FormValue(fmt.Sprintf("seat_%d", d.ID)); v != "" {
			d.SeatsLeft = max(0, min(d.SeatsTotal, atoiDef(v, d.SeatsLeft)))
		}
		if c.FormValue(fmt.Sprintf("hapus_%d", d.ID)) == "1" {
			demo.DB.Delete(&d)
			continue
		}
		demo.DB.Save(&d)
	}
	demo.ReloadPackages()
	return c.Redirect(fmt.Sprintf("/portal/paket?tersimpan=1#paket-%d", p.ID), fiber.StatusSeeOther)
}

func PortalPackageToggle(c *fiber.Ctx) error {
	t := curTravel(c)
	if accounts.SameOrigin(c) {
		demo.DB.Model(&demo.PortalPackage{}).Where("id = ? AND travel_slug = ?", idParam(c, "id"), t.Slug).Update("active", c.FormValue("aktif") == "1")
		demo.ReloadPackages()
	}
	return c.Redirect("/portal/paket", fiber.StatusSeeOther)
}

// ---- lead ----

func PortalLeads(c *fiber.Ctx) error {
	t := curTravel(c)
	status, period := c.Query("status"), c.Query("bulan", demo.CurrentPeriod())
	leads := demo.Leads(demo.LeadFilter{Slug: t.Slug, Status: status, Period: period, Limit: 300})
	counts := map[string]int{}
	var all []demo.Lead
	demo.DB.Select("status, billable, kind").Where("travel_slug = ?", t.Slug).Find(&all)
	for _, l := range all {
		counts[l.Status]++
		counts["kind_"+l.Kind]++
	}
	return portalRender(c, "portal_lead", "/portal/lead", "Lead", fiber.Map{"Leads": leads, "Status": status, "Period": period,
		"PeriodLabel": demo.PeriodLabel(period), "Periods": []string{demo.CurrentPeriod(), demo.PrevPeriod(demo.CurrentPeriod()), demo.PrevPeriod(demo.PrevPeriod(demo.CurrentPeriod()))},
		"Counts": counts, "CPL": demo.Cfg.CPL})
}

func PortalLeadsCSV(c *fiber.Ctx) error {
	t := curTravel(c)
	if t.Plan == "gratis" {
		return c.Redirect("/portal/langganan?fitur=csv")
	}
	period := c.Query("bulan", demo.CurrentPeriod())
	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="lead-%s-%s.csv"`, t.Slug, period))
	var b strings.Builder
	b.WriteString("waktu,jenis,paket,nama,no_wa,kota,sumber,status,ditagih,alasan\n")
	for _, l := range demo.Leads(demo.LeadFilter{Slug: t.Slug, Period: period, Limit: 5000}) {
		fmt.Fprintf(&b, "%s,%s,%q,%q,%q,%q,%s,%s,%v,%q\n", l.CreatedAt.In(demo.WIB).Format("2006-01-02 15:04"), l.Kind, l.Package, l.Name, l.Phone, l.City, l.Source, l.Status, l.Billable, l.Reason)
	}
	return c.SendString(b.String())
}

func PortalLeadStatus(c *fiber.Ctx) error {
	t := curTravel(c)
	if accounts.SameOrigin(c) {
		demo.SetLeadStatus(t.Slug, idParam(c, "id"), c.FormValue("status"))
	}
	return c.Redirect("/portal/lead?"+string(c.Request().URI().QueryString()), fiber.StatusSeeOther)
}

// ---- booking ----

func PortalBookings(c *fiber.Ctx) error {
	t := curTravel(c)
	status := c.Query("status")
	bs := demo.TravelBookings(t.Slug, status)
	counts := map[string]int{}
	for _, b := range demo.TravelBookings(t.Slug, "") {
		counts[b.Status]++
	}
	// Manifest per tanggal berangkat (booking aktif).
	type group struct {
		Date   string
		People int
		Count  int
	}
	gm := map[string]*group{}
	for _, b := range demo.TravelBookings(t.Slug, "") {
		if b.Status == "batal" || b.Status == "menunggu_dp" || b.DepartureDate < demo.Today() {
			continue
		}
		if gm[b.DepartureDate] == nil {
			gm[b.DepartureDate] = &group{Date: b.DepartureDate}
		}
		gm[b.DepartureDate].People += b.People
		gm[b.DepartureDate].Count++
	}
	var groups []group
	for _, g := range gm {
		groups = append(groups, *g)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Date < groups[j].Date })
	return portalRender(c, "portal_booking", "/portal/booking", "Booking", fiber.Map{"Bookings": bs, "Status": status, "Counts": counts,
		"Groups": firstN(groups, 6), "Err": c.Query("err")})
}

func PortalBookingStatus(c *fiber.Ctx) error {
	t := curTravel(c)
	if accounts.SameOrigin(c) {
		if err := demo.TravelSetStatus(t.Slug, idParam(c, "id"), c.FormValue("status")); err != nil {
			return c.Redirect("/portal/booking?err="+urlq(err.Error()), fiber.StatusSeeOther)
		}
	}
	return c.Redirect("/portal/booking", fiber.StatusSeeOther)
}

// ---- tagihan & langganan ----

func PortalInvoice(c *fiber.Ctx) error {
	t := curTravel(c)
	period := c.Query("bulan", demo.CurrentPeriod())
	inv := demo.BuildInvoice(*t, period)
	m := fiber.Map{"Inv": inv, "Period": period, "Periods": []string{demo.CurrentPeriod(), demo.PrevPeriod(demo.CurrentPeriod()), demo.PrevPeriod(demo.PrevPeriod(demo.CurrentPeriod()))},
		"CPL": demo.Cfg.CPL, "Final": period < demo.CurrentPeriod()}
	if c.Query("cetak") == "1" {
		m["T"] = t
		return demoRender(c, "demo/invoice_print", m)
	}
	return portalRender(c, "portal_tagihan", "/portal/tagihan", "Tagihan", m)
}

func PortalInvoicePay(c *fiber.Ctx) error {
	t := curTravel(c)
	p := c.FormValue("bulan")
	if accounts.SameOrigin(c) && len(p) == 7 && p < demo.CurrentPeriod() {
		demo.MarkInvoicePaid(t.Slug, p)
		inv := demo.BuildInvoice(*t, p)
		demo.Notify("admin", "email", "finance", "Tagihan dibayar: "+t.Name, fmt.Sprintf("%s membayar tagihan %s sebesar Rp %s (simulasi).", t.Name, inv.Label, demo.Rp(inv.Total)), "/admin/bisnis")
	}
	return c.Redirect("/portal/tagihan?bulan="+p, fiber.StatusSeeOther)
}

func PortalPlans(c *fiber.Ctx) error {
	t := curTravel(c)
	var logs []demo.SubscriptionLog
	demo.DB.Where("travel_slug = ?", t.Slug).Order("created_at DESC").Find(&logs)
	return portalRender(c, "portal_langganan", "/portal/langganan", "Langganan portal", fiber.Map{"Plans": demo.Cfg.Plans, "Logs": logs,
		"Feature": c.Query("fitur"), "Done": c.Query("ok") == "1"})
}

func PortalSubscribe(c *fiber.Ctx) error {
	t := curTravel(c)
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	plan := demo.PlanByKey(c.FormValue("plan"))
	months := atoiDef(c.FormValue("bulan"), 1)
	if months != 1 && months != 3 && months != 12 {
		months = 1
	}
	from := demo.Today()
	if t.PlanUntil > from && t.Plan == plan.Key {
		from = demo.AddDays(t.PlanUntil, 1)
	}
	fromT, _ := time.ParseInLocation("2006-01-02", from, demo.WIB)
	until := fromT.AddDate(0, months, -1).Format("2006-01-02")
	amount := plan.Price * int64(months)
	if months == 12 {
		amount = amount * 10 / 12 // bayar tahunan: 2 bulan gratis
	}
	if plan.Price > 0 {
		demo.DB.Create(&demo.SubscriptionLog{TravelSlug: t.Slug, Plan: plan.Key, Months: months, Amount: amount, From: from, Until: until, CreatedAt: time.Now()})
	}
	demo.DB.Model(&demo.Travel{}).Where("slug = ?", t.Slug).Updates(map[string]any{"plan": plan.Key, "plan_until": until})
	demo.WritePackagesFile()
	demo.Notify("travel:"+t.Slug, "email", t.PIC, "Langganan "+plan.Name+" aktif", fmt.Sprintf("Langganan portal %s aktif sampai %s. Tagihan Rp %s masuk ke tagihan bulan ini (simulasi).", plan.Name, demo.Tgl(until), demo.Rp(amount)), "/portal/tagihan")
	return c.Redirect("/portal/langganan?ok=1", fiber.StatusSeeOther)
}

func PortalPromo(c *fiber.Ctx) error {
	t := curTravel(c)
	var ps []demo.Placement
	demo.DB.Where("travel_name = ?", t.Name).Order("ends_at DESC").Find(&ps)
	days, cost := demo.PlacementCost(t.Name, demo.CurrentPeriod())
	return portalRender(c, "portal_promo", "/portal/promo", "Promo Spesial", fiber.Map{"Placements": ps, "Active": demo.ActivePlacement(t.Name),
		"Days": days, "Cost": cost, "Discount": t.Plan == "premium"})
}

func PortalProfile(c *fiber.Ctx) error {
	t := curTravel(c)
	avg, n := demo.ReviewStats(t.Slug)
	return portalRender(c, "portal_profil", "/portal/profil", "Profil & PPIU", fiber.Map{"Terms": demo.TermsFor(t.Slug), "Avg": avg, "N": n})
}
