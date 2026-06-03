package handlers

import (
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

	input := services.FilterInput{
		Budget:   budget,
		Priority: priority,
		Advanced: strings.Split(advanced, ","),
	}

	results := services.GetRecommendations(input)

	return c.Render("recommendations", fiber.Map{
		"Results": results,
		"Budget":  budget,
	})
}
