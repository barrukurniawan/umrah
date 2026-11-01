package handlers

import (
	"strconv"
	"strings"

	"umrah/app/accounts"
	"umrah/app/services"

	"github.com/gofiber/fiber/v2"
)

// Mode Kalender (HTMX). Parameter sama dengan form pencarian: budget (50 jt = semua), travels.

func calFilters(c *fiber.Ctx) (int, []string) {
	budget, _ := strconv.Atoi(c.Query("budget", "50000000"))
	var travels []string
	if t := strings.TrimSpace(c.Query("travels")); t != "" {
		travels = strings.Split(t, ",")
	}
	return budget, travels
}

// GET /kalender?bulan=2026-11&budget=&travels=
func CalendarMonth(c *fiber.Ctx) error {
	budget, travels := calFilters(c)
	c.Set("Cache-Control", "no-store")
	return c.Render("calendar", fiber.Map{"V": services.CalendarMonth(c.Query("bulan"), budget, travels)})
}

// GET /kalender/hari?tanggal=2026-12-30&budget=&travels=
func CalendarDay(c *fiber.Ctx) error {
	budget, travels := calFilters(c)
	d, ok := services.CalendarDay(c.Query("tanggal"), budget, travels)
	if !ok {
		return c.Status(fiber.StatusBadRequest).SendString("tanggal tidak valid")
	}
	saved := map[string]bool{}
	if u := accounts.Current(c); u != nil {
		saved = accounts.SavedKeys(u.ID)
	}
	c.Set("Cache-Control", "no-store")
	return c.Render("calendar_day", fiber.Map{"D": d, "Saved": saved})
}
