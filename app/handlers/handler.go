package handlers

import (
	"strconv"

	"umrah/app/services"

	"github.com/gofiber/fiber/v2"
)

func HomePage(c *fiber.Ctx) error {
	return c.Render("home", fiber.Map{})
}

func GetRecommendations(c *fiber.Ctx) error {
	budget, _ := strconv.Atoi(c.FormValue("budget", "25000000"))
	who := c.FormValue("who", "alone")
	priority := c.FormValue("priority", "near_haram")

	input := services.FilterInput{
		Budget:   budget,
		Who:      who,
		Priority: priority,
	}

	results := services.GetRecommendations(input)

	return c.Render("recommendations", fiber.Map{
		"Results": results,
		"Budget":  budget,
	})
}
