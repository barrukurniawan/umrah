package handlers

import (
	"math"
	"strconv"
	"strings"

	"umrah/app/analytics"
	"umrah/app/services"

	"github.com/gofiber/fiber/v2"
)

func HomePage(c *fiber.Ctx) error {
	return c.Render("home", fiber.Map{})
}

func PrivacyPage(c *fiber.Ctx) error {
	return c.Render("privacy", fiber.Map{"OptedOut": analytics.OptedOut(c)})
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

	input := services.FilterInput{
		Budget:   budget,
		Priority: priority,
		Advanced: strings.Split(advanced, ","),
		Page:     page,
		Sort:     sort,
		Month:    month,
		Travels:  travels,
	}

	results, total := services.GetRecommendations(input)
	totalPages := int(math.Ceil(float64(total) / 5))

	shown := make([]analytics.Impression, 0, len(results))
	for i, r := range results {
		shown = append(shown, analytics.Impression{Travel: r.Travel.Name, Package: r.Name, Price: r.Price, Position: (max(page, 1)-1)*5 + i + 1})
	}
	analytics.RecordSearch(c, budget, page, analytics.SearchMeta{Travels: travels, Month: month, Sort: sort, Total: total}, shown)

	return c.Render("recommendations", fiber.Map{
		"Results":    results,
		"Budget":     budget,
		"Page":       page,
		"TotalPages": totalPages,
		"HasPrev":    page > 1,
		"HasNext":    page < totalPages,
		"PrevPage":   page - 1,
		"NextPage":   page + 1,
		"Sort":       sort,
		"Month":      month,
		"Travels":    travels,
	})
}
