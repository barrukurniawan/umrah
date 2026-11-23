package handlers

import (
	"umrah/app/services"

	"github.com/gofiber/fiber/v2"
)

// GET /perlengkapan: daftar perlengkapan umrah dengan tombol ke Shopee (link affiliate).
func PerlengkapanPage(c *fiber.Ctx) error {
	return c.Render("perlengkapan", viewer(c, fiber.Map{
		"Title":    "Perlengkapan Umrah",
		"Products": services.Products(),
	}))
}
