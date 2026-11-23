package handlers

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"umrah/app/accounts"
	"umrah/app/analytics"
	"umrah/app/demo"
	"umrah/app/services"

	"github.com/gofiber/fiber/v2"
)

// Sisi jamaah: "Mau cicil?" + simulasi, booking + DP (mock), pelunasan, pembiayaan (rujukan),
// undian, ulasan, dan penghubung Tabungan ke rekening bank syariah mitra.

func registerDemoJamaah(app *fiber.App) {
	app.Get("/cicil", CicilSheet)
	app.Post("/api/cicil/intent", CicilIntent)
	app.Post("/cicil/tabungan", CicilToSavings)

	app.Get("/booking/baru", BookingForm)
	app.Post("/booking", BookingCreate)
	app.Get("/booking/:id<int>", BookingDetail)
	app.Get("/booking/:id<int>/kuitansi", BookingReceipt)
	app.Post("/booking/:id<int>/pelunasan", BookingPelunasan)
	app.Post("/booking/:id<int>/batal", BookingCancel)
	app.Post("/bayar/:pid<int>/simulasi", PaySimulate)
	app.Get("/booking/:id<int>/ulasan", ReviewForm)
	app.Post("/booking/:id<int>/ulasan", ReviewSubmit)
	app.Get("/akun/booking", MyBookings)

	app.Get("/pembiayaan/ajukan", FinancingForm)
	app.Post("/pembiayaan", FinancingSubmit)
	app.Get("/akun/pembiayaan", MyFinancing)
	app.Post("/akun/pembiayaan/:id<int>/cabut", FinancingRevoke)

	app.Get("/undian", DrawList)
	app.Get("/undian/:id<int>", DrawDetail)
	app.Get("/akun/undian", MyDrawEntries)

	app.Post("/tabungan/:id<int>/hubungkan-bank", SavingsLinkBank)
	app.Post("/tabungan/:id<int>/autodebet", SavingsAutodebit)
}

func urlq(s string) string { return url.QueryEscape(s) }

// jamaahOrLogin: jamaah harus login (di demo: lewat pengalih peran / halaman masuk demo).
func jamaahOrLogin(c *fiber.Ctx) (*accounts.User, error) {
	if u := accounts.Current(c); u != nil {
		return u, nil
	}
	return nil, c.Redirect("/auth/google/login?next="+urlq(c.OriginalURL()), fiber.StatusSeeOther)
}

func idParam(c *fiber.Ctx, name string) uint { return uint(atoiDef(c.Params(name), 0)) }

// ---------- #7 Mau cicil? ----------

// GET /cicil?key=...: lembar bawah (htmx) berisi pilihan jalur + simulasi.
func CicilSheet(c *fiber.Ctx) error {
	key := c.Query("key")
	cur := services.CurrentByKeys(map[string]bool{key: true})[key]
	if cur == nil {
		return c.Status(404).SendString(`<p class="p-6 text-sm text-gray-500">Paket tidak ditemukan.</p>`)
	}
	people := max(1, min(9, atoiDef(c.Query("orang"), 1)))
	price := int64(cur.MinPrice)
	total := price * int64(people)
	path := c.Query("jalur")
	tenor := atoiDef(c.Query("tenor"), 12)
	if tenor != 6 && tenor != 12 && tenor != 24 {
		tenor = 12
	}
	initial := rupiahInput(c.Query("awal"))
	if initial > total {
		initial = total
	}
	depart := c.Query("tgl")
	if depart == "" && len(cur.Departures) > 0 {
		depart = cur.Departures[0]
	}
	hminus := 45
	t := demo.TravelByName(cur.Travel)
	var terms *demo.TravelTerms
	if t != nil && t.Partner {
		terms = demo.TermsFor(t.Slug)
		if terms != nil {
			hminus = terms.PelunasanHMinus
		}
	}
	// Cicil sebelum berangkat masuk akal untuk keberangkatan yang masih jauh: bila jadwal terdekat
	// kurang dari 6 bulan lagi, usulkan berangkat ±12 bulan lagi (pengguna bisa memilih lain).
	if d := c.Query("target"); len(d) == 10 && d > demo.Today() {
		depart = d
	} else if path == "sebelum" && demo.DaysUntil(depart) < 180 {
		depart = demo.AddDays(demo.Today(), 365)
		for _, d := range cur.Departures {
			if demo.DaysUntil(d) >= 180 {
				depart = d
				break
			}
		}
	}
	deadline := demo.AddDays(depart, -hminus)
	if demo.DaysUntil(deadline) < 30 {
		deadline = demo.AddDays(demo.Today(), 30)
	}
	m := fiber.Map{"Cur": cur, "Key": key, "People": people, "Total": total, "Path": path, "Tenor": tenor, "Initial": initial,
		"Depart": depart, "HMinus": hminus, "Travel": t, "Terms": terms, "User": accounts.Current(c),
		"Saving": demo.PlanSaving(total, initial, deadline), "Estimates": demo.Estimates(total, tenor)}
	if path != "" {
		recordIntent(c, cur, path, tenor)
		ev := "simulasi_cicil"
		analytics.Record(analytics.Event{Type: ev, Travel: cur.Travel, Package: cur.Name, Price: cur.MinPrice, Meta: fmt.Sprintf(`{"jalur":%q,"tenor":%d}`, path, tenor)})
	}
	return demoRender(c, "demo/cicil", m)
}

func recordIntent(c *fiber.Ctx, cur *services.CurrentPackage, path string, tenor int) {
	if path != "sebelum" && path != "pembiayaan" {
		return
	}
	visitor, _, _, bot := analytics.FromRequest(c)
	if bot {
		return
	}
	var uid uint
	if u := accounts.Current(c); u != nil {
		uid = u.ID
	}
	// satu catatan per pengunjung+paket+jalur per hari
	var n int64
	demo.DB.Model(&demo.CicilIntent{}).Where("visitor = ? AND package = ? AND path = ? AND created_at > ?", visitor, cur.Name, path, time.Now().Add(-24*time.Hour)).Count(&n)
	if n == 0 {
		demo.DB.Create(&demo.CicilIntent{Visitor: visitor, UserID: uid, Travel: cur.Travel, Package: cur.Name, Price: int64(cur.MinPrice), Path: path, Tenor: tenor, CreatedAt: time.Now()})
	}
}

// POST /api/cicil/intent {key, jalur}: dipanggil tombol "Cuma lihat-lihat".
func CicilIntent(c *fiber.Ctx) error {
	key := c.FormValue("key")
	if cur := services.CurrentByKeys(map[string]bool{key: true})[key]; cur != nil {
		recordIntent(c, cur, c.FormValue("jalur"), 0)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// POST /cicil/tabungan: buat rencana tabungan dari simulasi "cicil sebelum berangkat".
func CicilToSavings(c *fiber.Ctx) error {
	u, err := jamaahOrLogin(c)
	if u == nil {
		return err
	}
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	key := c.FormValue("key")
	cur := services.CurrentByKeys(map[string]bool{key: true})[key]
	if cur == nil {
		return c.Redirect("/", fiber.StatusSeeOther)
	}
	if accounts.CountGoals(u.ID) >= accounts.MaxGoals {
		return c.Redirect("/tabungan", fiber.StatusSeeOther)
	}
	people := max(1, min(9, atoiDef(c.FormValue("orang"), 1)))
	dp := int64(0)
	if t := demo.TravelByName(cur.Travel); t != nil {
		if tm := demo.TermsFor(t.Slug); tm != nil {
			dp = tm.DPMinUmrohku
		}
	}
	g := accounts.SavingsGoal{UserID: u.ID, Title: "Umroh " + cur.Travel, Target: int64(cur.MinPrice) * int64(people), TargetDate: demo.AddDays(c.FormValue("tgl"), -45),
		People: people, PackageKey: key, PackageLabel: cur.Travel + " · " + cur.Name, PackagePrice: int64(cur.MinPrice), DownPayment: dp}
	if len(g.TargetDate) != 10 {
		g.TargetDate = ""
	}
	if err := accounts.CreateGoal(&g); err != nil {
		return c.Redirect("/tabungan", fiber.StatusSeeOther)
	}
	if initial := rupiahInput(c.FormValue("awal")); initial > 0 && initial <= g.Target {
		accounts.DB.Create(&accounts.SavingsEntry{GoalID: g.ID, UserID: u.ID, Amount: initial, Date: demo.Today(), Note: "Setoran awal", CreatedAt: time.Now()})
	}
	return c.Redirect(fmt.Sprintf("/tabungan/%d?baru=1", g.ID), fiber.StatusSeeOther)
}

// ---------- #8 Booking + DP ----------

func bookingInput(c *fiber.Ctx, uid uint) demo.BookingInput {
	in := demo.BookingInput{UserID: uid, Key: c.FormValue("key", c.Query("key")), Date: c.FormValue("tgl", c.Query("tgl")),
		Room: c.FormValue("kamar", c.Query("kamar")), ContactName: c.FormValue("kontak"), ContactPhone: c.FormValue("hp"),
		City: c.FormValue("kota"), PromoCode: c.FormValue("kode", c.Query("kode")), Method: c.FormValue("metode", "va")}
	n := max(1, min(9, atoiDef(c.FormValue("orang", c.Query("orang")), 1)))
	for i := 1; i <= n; i++ {
		in.Pilgrims = append(in.Pilgrims, c.FormValue(fmt.Sprintf("jamaah%d", i), "-"))
	}
	in.DPPerPerson = rupiahInput(c.FormValue("dp", c.Query("dp")))
	return in
}

func BookingForm(c *fiber.Ctx) error {
	u, err := jamaahOrLogin(c)
	if u == nil {
		return err
	}
	in := bookingInput(c, u.ID)
	q, qerr := demo.BuildQuote(in)
	if qerr != nil {
		return demoRender(c, "demo/booking_form", viewer(c, fiber.Map{"Title": "Pesan paket", "Error": qerr.Error()}))
	}
	if in.DPPerPerson == 0 {
		in.DPPerPerson = q.Opt.Terms.DPMinUmrohku
	}
	m := fiber.Map{"Title": "Pesan & bayar DP", "Q": q, "In": in, "People": q.People, "Partial": c.Get("HX-Request") == "true"}
	if c.Get("HX-Request") == "true" {
		return demoRender(c, "demo/booking_summary", m)
	}
	return demoRender(c, "demo/booking_form", viewer(c, m))
}

func BookingCreate(c *fiber.Ctx) error {
	u, err := jamaahOrLogin(c)
	if u == nil {
		return err
	}
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	in := bookingInput(c, u.ID)
	if c.FormValue("setuju") != "1" {
		return renderBookingErr(c, in, "Centang persetujuan syarat pembayaran & pembatalan dulu.")
	}
	b, p, berr := demo.CreateBooking(in)
	if berr != nil {
		return renderBookingErr(c, in, berr.Error())
	}
	_ = p
	return c.Redirect(fmt.Sprintf("/booking/%d", b.ID), fiber.StatusSeeOther)
}

func renderBookingErr(c *fiber.Ctx, in demo.BookingInput, msg string) error {
	q, err := demo.BuildQuote(in)
	if err != nil {
		return demoRender(c, "demo/booking_form", viewer(c, fiber.Map{"Title": "Pesan paket", "Error": err.Error()}))
	}
	return demoRender(c, "demo/booking_form", viewer(c, fiber.Map{"Title": "Pesan & bayar DP", "Q": q, "In": in, "People": q.People, "FormError": msg}))
}

// ownBooking: hanya pemilik booking yang boleh melihat (travel lewat portal).
func ownBooking(c *fiber.Ctx) (*demo.Booking, *accounts.User, error) {
	u, err := jamaahOrLogin(c)
	if u == nil {
		return nil, nil, err
	}
	b, gerr := demo.GetBooking(idParam(c, "id"))
	if gerr != nil || b.UserID != u.ID {
		return nil, u, c.Redirect("/akun/booking", fiber.StatusSeeOther)
	}
	return b, u, nil
}

func BookingDetail(c *fiber.Ctx) error {
	b, _, err := ownBooking(c)
	if b == nil {
		return err
	}
	pays := demo.BookingPayments(b.ID)
	var open *demo.Payment
	for i := range pays {
		if pays[i].Status == "menunggu" {
			open = &pays[i]
		}
	}
	var entries []demo.DrawEntry
	demo.DB.Where("booking_id = ?", b.ID).Find(&entries)
	var review demo.Review
	hasReview := demo.DB.Where("booking_id = ?", b.ID).Take(&review).Error == nil
	return demoRender(c, "demo/booking_detail", viewer(c, fiber.Map{"Title": "Booking " + b.Code, "B": b, "T": demo.TravelBySlug(b.TravelSlug),
		"Terms": demo.TermsFor(b.TravelSlug), "Payments": pays, "Open": open, "Promos": demo.BookingPromos(b.ID), "Entries": entries,
		"HasReview": hasReview, "Paid": c.Query("dibayar") == "1"}))
}

func BookingReceipt(c *fiber.Ctx) error {
	b, _, err := ownBooking(c)
	if b == nil {
		return err
	}
	return demoRender(c, "demo/kuitansi", fiber.Map{"Title": "Kuitansi " + b.Code, "B": b, "T": demo.TravelBySlug(b.TravelSlug), "Payments": demo.BookingPayments(b.ID)})
}

func BookingPelunasan(c *fiber.Ctx) error {
	b, _, err := ownBooking(c)
	if b == nil {
		return err
	}
	if accounts.SameOrigin(c) {
		demo.CreatePelunasan(b, c.FormValue("metode", "va"))
	}
	return c.Redirect(fmt.Sprintf("/booking/%d#bayar", b.ID), fiber.StatusSeeOther)
}

func BookingCancel(c *fiber.Ctx) error {
	b, _, err := ownBooking(c)
	if b == nil {
		return err
	}
	if accounts.SameOrigin(c) && b.Status == "menunggu_dp" {
		demo.CancelBooking(b, "Dibatalkan jamaah sebelum membayar DP")
	}
	return c.Redirect(fmt.Sprintf("/booking/%d", b.ID), fiber.StatusSeeOther)
}

// POST /bayar/:pid/simulasi: pengganti notifikasi gateway pembayaran.
func PaySimulate(c *fiber.Ctx) error {
	u, err := jamaahOrLogin(c)
	if u == nil {
		return err
	}
	p, perr := demo.GetPayment(idParam(c, "pid"))
	if perr != nil || !accounts.SameOrigin(c) {
		return c.Redirect("/akun/booking", fiber.StatusSeeOther)
	}
	b, berr := demo.GetBooking(p.BookingID)
	if berr != nil || b.UserID != u.ID {
		return c.Redirect("/akun/booking", fiber.StatusSeeOther)
	}
	if _, err := demo.SimulatePaid(p.ID); err != nil {
		return c.Redirect(fmt.Sprintf("/booking/%d", b.ID), fiber.StatusSeeOther)
	}
	return c.Redirect(fmt.Sprintf("/booking/%d?dibayar=1", b.ID), fiber.StatusSeeOther)
}

func MyBookings(c *fiber.Ctx) error {
	u, err := jamaahOrLogin(c)
	if u == nil {
		return err
	}
	return demoRender(c, "demo/booking_list", viewer(c, fiber.Map{"Title": "Booking saya", "Bookings": demo.UserBookings(u.ID)}))
}

// ---------- Ulasan terverifikasi ----------

func ReviewForm(c *fiber.Ctx) error {
	b, _, err := ownBooking(c)
	if b == nil {
		return err
	}
	return demoRender(c, "demo/ulasan", viewer(c, fiber.Map{"Title": "Tulis ulasan", "B": b, "Error": c.Query("err")}))
}

func ReviewSubmit(c *fiber.Ctx) error {
	b, u, err := ownBooking(c)
	if b == nil {
		return err
	}
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	if rerr := demo.AddReview(b, atoiDef(c.FormValue("rating"), 0), c.FormValue("text"), u.Name); rerr != nil {
		return c.Redirect(fmt.Sprintf("/booking/%d/ulasan?err=%s", b.ID, urlq(rerr.Error())), fiber.StatusSeeOther)
	}
	return c.Redirect("/travel/"+b.TravelSlug+"#ulasan", fiber.StatusSeeOther)
}

// ---------- #13 Rujukan pembiayaan syariah ----------

func FinancingForm(c *fiber.Ctx) error {
	u, err := jamaahOrLogin(c)
	if u == nil {
		return err
	}
	key := c.Query("key")
	cur := services.CurrentByKeys(map[string]bool{key: true})[key]
	m := fiber.Map{"Title": "Ajukan pembiayaan syariah", "Cur": cur, "Key": key, "Tenor": atoiDef(c.Query("tenor"), 12), "Partner": c.Query("mitra"),
		"Name": u.Name, "Error": c.Query("err"), "BookingID": atoiDef(c.Query("booking"), 0)}
	amount := rupiahInput(c.Query("nominal"))
	if amount == 0 && cur != nil {
		amount = int64(cur.MinPrice) * int64(max(1, atoiDef(c.Query("orang"), 1)))
	}
	if bid := atoiDef(c.Query("booking"), 0); bid > 0 {
		if b, err := demo.GetBooking(uint(bid)); err == nil && b.UserID == u.ID {
			amount = b.Remaining()
			m["B"] = b
		}
	}
	m["Amount"] = amount
	m["Estimates"] = demo.Estimates(amount, m["Tenor"].(int))
	m["Partners"] = demo.Cfg.Partners
	return demoRender(c, "demo/pembiayaan_form", viewer(c, m))
}

func FinancingSubmit(c *fiber.Ctx) error {
	u, err := jamaahOrLogin(c)
	if u == nil {
		return err
	}
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	key := c.FormValue("key")
	in := demo.FinancingInput{UserID: u.ID, PartnerSlug: c.FormValue("mitra"), PackageKey: key, Amount: rupiahInput(c.FormValue("nominal")),
		Tenor: atoiDef(c.FormValue("tenor"), 12), Name: trunc(c.FormValue("nama"), 80), Phone: trunc(c.FormValue("hp"), 30), Job: trunc(c.FormValue("pekerjaan"), 60),
		Income: rupiahInput(c.FormValue("penghasilan")), City: trunc(c.FormValue("kota"), 60), Consent: c.FormValue("setuju_data") == "1"}
	if cur := services.CurrentByKeys(map[string]bool{key: true})[key]; cur != nil {
		in.TravelName, in.PackageName = cur.Travel, cur.Name
	}
	if bid := uint(atoiDef(c.FormValue("booking"), 0)); bid > 0 {
		if b, err := demo.GetBooking(bid); err == nil && b.UserID == u.ID {
			in.BookingID, in.TravelName, in.PackageName, in.PackageKey = b.ID, b.TravelName, b.PackageName, b.PackageKey
		}
	}
	if _, ferr := demo.ApplyFinancing(in); ferr != nil {
		q := fmt.Sprintf("/pembiayaan/ajukan?key=%s&tenor=%d&mitra=%s&nominal=%d&err=%s", urlq(key), in.Tenor, urlq(in.PartnerSlug), in.Amount, urlq(ferr.Error()))
		if in.BookingID > 0 {
			q += fmt.Sprintf("&booking=%d", in.BookingID)
		}
		return c.Redirect(q, fiber.StatusSeeOther)
	}
	return c.Redirect("/akun/pembiayaan?terkirim=1", fiber.StatusSeeOther)
}

func MyFinancing(c *fiber.Ctx) error {
	u, err := jamaahOrLogin(c)
	if u == nil {
		return err
	}
	return demoRender(c, "demo/pembiayaan_list", viewer(c, fiber.Map{"Title": "Pengajuan pembiayaan", "Apps": demo.UserFinancing(u.ID), "Sent": c.Query("terkirim") == "1", "Error": c.Query("err")}))
}

func FinancingRevoke(c *fiber.Ctx) error {
	u, err := jamaahOrLogin(c)
	if u == nil {
		return err
	}
	if accounts.SameOrigin(c) {
		if rerr := demo.RevokeConsent(u.ID, idParam(c, "id")); rerr != nil {
			return c.Redirect("/akun/pembiayaan?err="+urlq(rerr.Error()), fiber.StatusSeeOther)
		}
	}
	return c.Redirect("/akun/pembiayaan", fiber.StatusSeeOther)
}

// ---------- Undian ----------

func DrawList(c *fiber.Ctx) error {
	return demoRender(c, "demo/undian_list", viewer(c, fiber.Map{"Title": "Undian umroh gratis", "Draws": demo.PublicDraws()}))
}

func DrawDetail(c *fiber.Ctx) error {
	d, err := demo.GetDraw(idParam(c, "id"))
	if err != nil || d.Status == "draft" {
		return c.Redirect("/undian")
	}
	active, void := d.Count()
	m := fiber.Map{"Title": d.Title, "D": d, "Active": active, "Void": void, "Winners": d.WinnersList()}
	if d.Status == "ditutup" || d.Status == "diundi" {
		var codes []string
		for _, e := range demo.DrawEntries(d.ID) {
			if !e.Void {
				codes = append(codes, e.Code)
			}
		}
		m["Codes"] = strings.Join(codes, " ")
	}
	if u := accounts.Current(c); u != nil {
		var mine []demo.DrawEntry
		demo.DB.Where("draw_id = ? AND user_id = ?", d.ID, u.ID).Find(&mine)
		m["Mine"] = mine
	}
	return demoRender(c, "demo/undian_detail", viewer(c, m))
}

func MyDrawEntries(c *fiber.Ctx) error {
	u, err := jamaahOrLogin(c)
	if u == nil {
		return err
	}
	entries := demo.UserEntries(u.ID)
	draws := map[uint]*demo.Draw{}
	for _, e := range entries {
		if draws[e.DrawID] == nil {
			if d, err := demo.GetDraw(e.DrawID); err == nil {
				draws[e.DrawID] = d
			}
		}
	}
	return demoRender(c, "demo/akun_undian", viewer(c, fiber.Map{"Title": "Kode undian saya", "Entries": entries, "Draws": draws}))
}

// ---------- #12 Tabungan terhubung bank syariah mitra ----------

func SavingsLinkBank(c *fiber.Ctx) error {
	u, err := jamaahOrLogin(c)
	if u == nil {
		return err
	}
	g, gerr := accounts.GetGoal(u.ID, idParam(c, "id"))
	if gerr != nil || !accounts.SameOrigin(c) {
		return c.Redirect("/tabungan", fiber.StatusSeeOther)
	}
	amt := rupiahInput(c.FormValue("autodebet"))
	if amt < 100_000 || amt > 100_000_000 {
		amt = 1_000_000
	}
	day := max(1, min(28, atoiDef(c.FormValue("tanggal"), 25)))
	var l demo.SavingsLink
	if demo.DB.Where("goal_id = ?", g.ID).Take(&l).Error != nil {
		l = demo.SavingsLink{UserID: u.ID, GoalID: g.ID, Bank: "Bank Syariah Contoh", AccountNo: fmt.Sprintf("7101 %04d %04d", u.ID%10000, g.ID%10000), LinkedAt: time.Now()}
	}
	l.Autodebit, l.Day = amt, day
	demo.DB.Save(&l)
	demo.Notify(demo.UserAudience(u.ID), "email", u.Name, "Rekening tabungan umroh aktif",
		fmt.Sprintf("Tabungan \"%s\" kini terhubung ke rekening tabungan umroh (akad wadiah) di Bank Syariah Contoh no. %s. Autodebet Rp %s setiap tanggal %d. Dana tersimpan atas nama Anda, bukan di umrohku.", g.Title, l.AccountNo, demo.Rp(amt), day), fmt.Sprintf("/tabungan/%d", g.ID))
	return c.Redirect(fmt.Sprintf("/tabungan/%d", g.ID), fiber.StatusSeeOther)
}

// POST /tabungan/:id/autodebet: simulasikan autodebet bulan ini (saldo "tersinkron" dari bank).
func SavingsAutodebit(c *fiber.Ctx) error {
	u, err := jamaahOrLogin(c)
	if u == nil {
		return err
	}
	g, gerr := accounts.GetGoal(u.ID, idParam(c, "id"))
	var l demo.SavingsLink
	if gerr != nil || !accounts.SameOrigin(c) || demo.DB.Where("goal_id = ? AND user_id = ?", g.ID, u.ID).Take(&l).Error != nil {
		return c.Redirect("/tabungan", fiber.StatusSeeOther)
	}
	accounts.DB.Create(&accounts.SavingsEntry{GoalID: g.ID, UserID: u.ID, Amount: l.Autodebit, Date: demo.Today(), Note: "Autodebet " + l.Bank, CreatedAt: time.Now()})
	return c.Redirect(fmt.Sprintf("/tabungan/%d", g.ID), fiber.StatusSeeOther)
}

// SavingsLinkFor dipakai template tabungan (lewat handler SavingsDetail).
func savingsLinkFor(goalID uint) *demo.SavingsLink {
	if !demo.Enabled() {
		return nil
	}
	var l demo.SavingsLink
	if demo.DB.Where("goal_id = ?", goalID).Take(&l).Error != nil {
		return nil
	}
	return &l
}
