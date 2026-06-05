package handlers

import (
	"math"
	"strconv"
	"strings"

	"umrah/app/services"

	"github.com/gofiber/fiber/v2"
)

func HomePage(c *fiber.Ctx) error {
	return c.Render("home", fiber.Map{})
}

func GetRecommendations(c *fiber.Ctx) error {
	budget, _ := strconv.Atoi(c.FormValue("budget", "25000000"))
	priority := c.FormValue("priority", "all")
	advanced := c.FormValue("advanced", "")
	page, _ := strconv.Atoi(c.FormValue("page", "1"))
	sort := c.FormValue("sort", "score")
	month := c.FormValue("month", "")

	input := services.FilterInput{
		Budget:   budget,
		Priority: priority,
		Advanced: strings.Split(advanced, ","),
		Page:     page,
		Sort:     sort,
		Month:    month,
	}

	results, total := services.GetRecommendations(input)
	totalPages := int(math.Ceil(float64(total) / 5))

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
	})
}
