package handlers

import (
	"math"
	"strconv"
	"strings"

	"umrah/app/accounts"
	"umrah/app/analytics"
	"umrah/app/services"

	"github.com/gofiber/fiber/v2"
)

func HomePage(c *fiber.Ctx) error {
	return c.Render("home", viewer(c, fiber.Map{}))
}

func SimulasiPage(c *fiber.Ctx) error {
	part, title := "tawaf", "Simulasi Tawaf"
	if c.Query("bagian") == "sai" {
		part, title = "sai", "Simulasi Sa'i"
	}
	return c.Render("simulasi", viewer(c, fiber.Map{"Title": title, "Part": part, "HideChat": true, "WideNav": true}))
}

func PrivacyPage(c *fiber.Ctx) error {
	return c.Render("privacy", viewer(c, fiber.Map{"OptedOut": analytics.OptedOut(c)}))
}

func PrivacyOptOut(c *fiber.Ctx) error {
	analytics.SetOptOut(c, true)
	return c.Redirect("/privasi#opt-out", fiber.StatusSeeOther)
}

func PrivacyOptIn(c *fiber.Ctx) error {
	analytics.SetOptOut(c, false)
	return c.Redirect("/privasi#opt-out", fiber.StatusSeeOther)
}

func GetRecommendations(c *fiber.Ctx) error {
	budget, _ := strconv.Atoi(c.FormValue("budget", "26000000"))
	priority := c.FormValue("priority", "all")
	advanced := c.FormValue("advanced", "")
	page, _ := strconv.Atoi(c.FormValue("page", "1"))
	sort := c.FormValue("sort", "price_desc")
	month := c.FormValue("month", "")
	travels := c.FormValue("travels", "")
	flight := c.FormValue("flight", "")
	if sort == "direct_only" { // nilai lama (dulu Direct ada di daftar urutan)
		sort, flight = "price_desc", "direct"
	}
	if _, ok := sortLabels[sort]; !ok {
		sort = "price_desc"
	}
	if flight != "direct" && flight != "transit" {
		flight = ""
	}

	input := services.FilterInput{
		Budget:   budget,
		Priority: priority,
		Advanced: strings.Split(advanced, ","),
		Page:     page,
		Sort:     sort,
		Month:    month,
		Travels:  travels,
		Flight:   flight,
	}

	results, total := services.GetRecommendations(input)
	nearest, nearestTravel := 0, ""
	if total == 0 {
		nearest, nearestTravel = services.NearestPrice(input)
	}
	totalPages := int(math.Ceil(float64(total) / 5))

	shown := make([]analytics.Impression, 0, len(results))
	for i, r := range results {
		shown = append(shown, analytics.Impression{Travel: r.Travel.Name, Package: r.Name, Price: r.Price, Position: (max(page, 1)-1)*5 + i + 1})
	}
	analytics.RecordSearch(c, budget, page, analytics.SearchMeta{Travels: travels, Month: month, Sort: sort, Flight: flight, Total: total}, shown)

	saved := map[string]bool{}
	if u := accounts.Current(c); u != nil {
		saved = accounts.SavedKeys(u.ID)
	}

	return c.Render("recommendations", fiber.Map{
		"Searched":      true, // tampilkan dropdown urutan/bulan walau hasilnya kosong
		"Saved":         saved,
		"LoggedIn":      accounts.Current(c) != nil,
		"Results":       results,
		"Budget":        budget,
		"Page":          page,
		"TotalPages":    totalPages,
		"HasPrev":       page > 1,
		"HasNext":       page < totalPages,
		"PrevPage":      page - 1,
		"NextPage":      page + 1,
		"Sort":          sort,
		"Month":         month,
		"Travels":       travels,
		"Flight":        flight,
		"Nearest":       nearest,
		"NearestTravel": nearestTravel,
		"NearestBudget": nearest / 1000000 * 1000000,
		"SortLabel":     sortLabels[sort],
		"MonthLabel":    monthLabels[month],
		"FilterCount":   boolCount(flight != "", sort != "price_desc", month != ""),
	})
}

var sortLabels = map[string]string{
	"price_desc": "Harga Tertinggi", "price_asc": "Harga Terendah", "distance_asc": "Hotel Terdekat",
	"duration_asc": "Durasi Tersingkat", "duration_desc": "Durasi Terlama", "dp_asc": "DP Terendah",
}

var monthLabels = map[string]string{
	"01": "Januari", "02": "Februari", "03": "Maret", "04": "April", "05": "Mei", "06": "Juni",
	"07": "Juli", "08": "Agustus", "09": "September", "10": "Oktober", "11": "November", "12": "Desember",
}

func boolCount(bs ...bool) int {
	n := 0
	for _, b := range bs {
		if b {
			n++
		}
	}
	return n
}
