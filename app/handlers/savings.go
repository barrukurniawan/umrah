package handlers

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"umrah/app/accounts"
	"umrah/app/services"

	"github.com/gofiber/fiber/v2"
)

var wib = time.FixedZone("WIB", 7*3600)

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// parseRupiah menerima "1.500.000", "Rp 1,5 jt", "500rb", "-250.000" (pengambilan).
func parseRupiah(s string) (int64, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	neg := strings.HasPrefix(s, "-")
	mult := 1.0
	switch {
	case strings.Contains(s, "jt") || strings.Contains(s, "juta"):
		mult = 1_000_000
	case strings.Contains(s, "rb") || strings.Contains(s, "ribu") || strings.HasSuffix(s, "k"):
		mult = 1_000
	}
	var n float64
	if mult > 1 {
		num := regexp.MustCompile(`[0-9]+([.,][0-9]+)?`).FindString(s)
		f, err := strconv.ParseFloat(strings.Replace(num, ",", ".", 1), 64)
		if err != nil {
			return 0, false
		}
		n = f * mult
	} else {
		digits := regexp.MustCompile(`\D`).ReplaceAllString(s, "")
		v, err := strconv.ParseInt(digits, 10, 64)
		if err != nil {
			return 0, false
		}
		n = float64(v)
	}
	v := int64(n + 0.5)
	if v <= 0 || v > accounts.MaxAmount {
		return 0, false
	}
	if neg {
		v = -v
	}
	return v, true
}

func needUser(c *fiber.Ctx) *accounts.User {
	return accounts.Current(c)
}

func goalID(c *fiber.Ctx) uint {
	id, _ := strconv.ParseUint(c.Params("id"), 10, 32)
	return uint(id)
}

func todayWIB() string { return time.Now().In(wib).Format("2006-01-02") }

type goalCard struct {
	accounts.SavingsGoal
	P accounts.Progress
}

func SavingsList(c *fiber.Ctx) error {
	u := needUser(c)
	m := fiber.Map{}
	if u != nil {
		var cards []goalCard
		for _, g := range accounts.ListGoals(u.ID) {
			g := g
			cards = append(cards, goalCard{SavingsGoal: g, P: accounts.ComputeProgress(&g, accounts.ListEntries(g.ID), time.Now().In(wib))})
		}
		m["Goals"] = cards
		m["CanAdd"] = len(cards) < accounts.MaxGoals
	}
	return c.Render("savings_list", viewer(c, m))
}

// savedOptions: paket tersimpan + harga/DP terkini, untuk mengisi target otomatis.
type pkgOption struct {
	Key, Label string
	Price, DP  int
}

func savedOptions(userID uint) []pkgOption {
	saved := accounts.ListSaved(userID)
	keys := map[string]bool{}
	for _, s := range saved {
		keys[s.Key] = true
	}
	cur := services.CurrentByKeys(keys)
	var out []pkgOption
	for _, s := range saved {
		o := pkgOption{Key: s.Key, Label: s.Travel + " · " + s.Package, Price: s.Price}
		if cp := cur[s.Key]; cp != nil {
			o.Price, o.DP = cp.MinPrice, cp.DownPayment
		}
		out = append(out, o)
	}
	return out
}

type goalForm struct {
	Title, TargetDate, PackageKey string
	Price, Extra, People          string
	Error                         string
}

func SavingsNew(c *fiber.Ctx) error {
	u := needUser(c)
	if u == nil {
		return c.Redirect("/auth/google/login?next=" + c.OriginalURL())
	}
	if accounts.CountGoals(u.ID) >= accounts.MaxGoals {
		return c.Redirect("/tabungan")
	}
	opts := savedOptions(u.ID)
	f := goalForm{Title: "Umroh bersama keluarga", People: "1", Extra: "3.000.000", PackageKey: c.Query("paket")}
	for _, o := range opts {
		if o.Key == f.PackageKey {
			f.Price = strconv.Itoa(o.Price)
			f.Title = "Umroh " + strings.SplitN(o.Label, " · ", 2)[0]
		}
	}
	return c.Render("savings_form", viewer(c, fiber.Map{"Form": f, "Options": opts, "Editing": false}))
}

// fillGoal memvalidasi form dan menghitung target = (harga paket + tambahan) × jumlah orang.
func fillGoal(c *fiber.Ctx, userID uint, g *accounts.SavingsGoal) (goalForm, bool) {
	f := goalForm{
		Title: strings.TrimSpace(c.FormValue("title")), TargetDate: c.FormValue("target_date"), PackageKey: c.FormValue("package"),
		Price: c.FormValue("price"), Extra: c.FormValue("extra"), People: c.FormValue("people"),
	}
	people, err := strconv.Atoi(f.People)
	if err != nil || people < 1 || people > 20 {
		f.Error = "Jumlah orang 1–20."
		return f, false
	}
	price, ok := parseRupiah(f.Price)
	if !ok || price < 0 {
		f.Error = "Isi harga paket per orang (contoh: 28.500.000)."
		return f, false
	}
	extra := int64(0)
	if strings.TrimSpace(f.Extra) != "" {
		if extra, ok = parseRupiah(f.Extra); !ok || extra < 0 {
			f.Error = "Tambahan per orang tidak valid."
			return f, false
		}
	}
	if f.Title == "" {
		f.Title = "Tabungan umroh"
	}
	if len([]rune(f.Title)) > 100 {
		f.Title = string([]rune(f.Title)[:100])
	}
	if f.TargetDate != "" && (!dateRe.MatchString(f.TargetDate) || f.TargetDate < todayWIB()) {
		f.Error = "Tanggal target harus hari ini atau setelahnya."
		return f, false
	}

	g.Title, g.TargetDate, g.People = f.Title, f.TargetDate, people
	g.PackagePrice = price
	g.Target = (price + extra) * int64(people)
	g.PackageKey, g.PackageLabel, g.DownPayment = "", "", 0
	for _, o := range savedOptions(userID) {
		if o.Key == f.PackageKey {
			g.PackageKey, g.PackageLabel, g.DownPayment = o.Key, o.Label, int64(o.DP)
		}
	}
	return f, true
}

func SavingsCreate(c *fiber.Ctx) error {
	u := needUser(c)
	if u == nil || !accounts.SameOrigin(c) {
		return c.Redirect("/tabungan", fiber.StatusSeeOther)
	}
	if accounts.CountGoals(u.ID) >= accounts.MaxGoals {
		return c.Redirect("/tabungan", fiber.StatusSeeOther)
	}
	g := accounts.SavingsGoal{UserID: u.ID}
	f, ok := fillGoal(c, u.ID, &g)
	if !ok {
		return c.Render("savings_form", viewer(c, fiber.Map{"Form": f, "Options": savedOptions(u.ID), "Editing": false}))
	}
	if err := accounts.CreateGoal(&g); err != nil {
		f.Error = "Gagal menyimpan, coba lagi."
		return c.Render("savings_form", viewer(c, fiber.Map{"Form": f, "Options": savedOptions(u.ID), "Editing": false}))
	}
	return c.Redirect(fmt.Sprintf("/tabungan/%d?baru=1", g.ID), fiber.StatusSeeOther)
}

func SavingsEdit(c *fiber.Ctx) error {
	u := needUser(c)
	if u == nil {
		return c.Redirect("/auth/google/login?next=/tabungan")
	}
	g, err := accounts.GetGoal(u.ID, goalID(c))
	if err != nil {
		return c.Redirect("/tabungan")
	}
	people := int64(max(g.People, 1))
	f := goalForm{Title: g.Title, TargetDate: g.TargetDate, PackageKey: g.PackageKey, People: strconv.Itoa(g.People),
		Price: strconv.FormatInt(g.PackagePrice, 10), Extra: strconv.FormatInt(max(g.Target/people-g.PackagePrice, 0), 10)}
	return c.Render("savings_form", viewer(c, fiber.Map{"Form": f, "Options": savedOptions(u.ID), "Editing": true, "GoalID": g.ID}))
}

func SavingsUpdate(c *fiber.Ctx) error {
	u := needUser(c)
	if u == nil || !accounts.SameOrigin(c) {
		return c.Redirect("/tabungan", fiber.StatusSeeOther)
	}
	g, err := accounts.GetGoal(u.ID, goalID(c))
	if err != nil {
		return c.Redirect("/tabungan", fiber.StatusSeeOther)
	}
	f, ok := fillGoal(c, u.ID, g)
	if !ok {
		return c.Render("savings_form", viewer(c, fiber.Map{"Form": f, "Options": savedOptions(u.ID), "Editing": true, "GoalID": g.ID}))
	}
	accounts.UpdateGoal(g)
	accounts.MarkAchieved(g, accounts.GoalTotal(g.ID))
	return c.Redirect(fmt.Sprintf("/tabungan/%d", g.ID), fiber.StatusSeeOther)
}

func SavingsDelete(c *fiber.Ctx) error {
	u := needUser(c)
	if u != nil && accounts.SameOrigin(c) {
		accounts.DeleteGoal(u.ID, goalID(c))
	}
	return c.Redirect("/tabungan", fiber.StatusSeeOther)
}

func SavingsDetail(c *fiber.Ctx) error {
	u := needUser(c)
	if u == nil {
		return c.Redirect("/auth/google/login?next=" + c.OriginalURL())
	}
	g, err := accounts.GetGoal(u.ID, goalID(c))
	if err != nil {
		return c.Redirect("/tabungan")
	}
	entries := accounts.ListEntries(g.ID)
	p := accounts.ComputeProgress(g, entries, time.Now().In(wib))
	var current *services.CurrentPackage
	if g.PackageKey != "" {
		current = services.CurrentByKeys(map[string]bool{g.PackageKey: true})[g.PackageKey]
	}
	travel := ""
	if g.PackageLabel != "" {
		travel = strings.SplitN(g.PackageLabel, " · ", 2)[0]
	}
	return c.Render("savings_detail", viewer(c, fiber.Map{
		"Goal": g, "P": p, "Entries": entries, "Today": todayWIB(), "Current": current, "Travel": travel,
		"Celebrate": c.Query("tercapai") == "1", "Milestone": c.Query("tonggak"), "IsNew": c.Query("baru") == "1",
		"EntryError": c.Query("err"),
	}))
}

func EntryAdd(c *fiber.Ctx) error {
	u := needUser(c)
	if u == nil || !accounts.SameOrigin(c) {
		return c.Redirect("/tabungan", fiber.StatusSeeOther)
	}
	g, err := accounts.GetGoal(u.ID, goalID(c))
	if err != nil {
		return c.Redirect("/tabungan", fiber.StatusSeeOther)
	}
	back := fmt.Sprintf("/tabungan/%d", g.ID)
	amount, ok := parseRupiah(c.FormValue("amount"))
	if !ok {
		return c.Redirect(back+"?err=jumlah#setor", fiber.StatusSeeOther)
	}
	if c.FormValue("kind") == "ambil" && amount > 0 {
		amount = -amount
	}
	date := c.FormValue("date")
	if !dateRe.MatchString(date) || date > todayWIB() {
		date = todayWIB()
	}
	if accounts.CountEntries(g.ID) >= accounts.MaxEntriesPerGoal {
		return c.Redirect(back+"?err=penuh#setor", fiber.StatusSeeOther)
	}
	note := strings.TrimSpace(c.FormValue("note"))
	if len([]rune(note)) > 120 {
		note = string([]rune(note)[:120])
	}

	before := accounts.GoalTotal(g.ID)
	if err := accounts.AddEntry(&accounts.SavingsEntry{GoalID: g.ID, UserID: u.ID, Amount: amount, Date: date, Note: note}); err != nil {
		return c.Redirect(back+"?err=gagal#setor", fiber.StatusSeeOther)
	}
	after := before + amount
	if accounts.MarkAchieved(g, after) {
		return c.Redirect(back+"?tercapai=1", fiber.StatusSeeOther)
	}
	// Tonggak 25/50/75% yang baru terlewati.
	if g.Target > 0 && amount > 0 {
		for _, m := range []int64{75, 50, 25} {
			if before*100 < g.Target*m && after*100 >= g.Target*m {
				return c.Redirect(fmt.Sprintf("%s?tonggak=%d", back, m), fiber.StatusSeeOther)
			}
		}
	}
	return c.Redirect(back+"#riwayat", fiber.StatusSeeOther)
}

func EntryDelete(c *fiber.Ctx) error {
	u := needUser(c)
	if u == nil || !accounts.SameOrigin(c) {
		return c.Redirect("/tabungan", fiber.StatusSeeOther)
	}
	g, err := accounts.GetGoal(u.ID, goalID(c))
	if err != nil {
		return c.Redirect("/tabungan", fiber.StatusSeeOther)
	}
	eid, _ := strconv.ParseUint(c.Params("eid"), 10, 32)
	accounts.DeleteEntry(u.ID, g.ID, uint(eid))
	accounts.MarkAchieved(g, accounts.GoalTotal(g.ID))
	return c.Redirect(fmt.Sprintf("/tabungan/%d#riwayat", g.ID), fiber.StatusSeeOther)
}
