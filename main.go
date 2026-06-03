package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"umrah/app/handlers"
	"umrah/app/repositories"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	engine := html.New("./web/templates", ".html")

	engine.AddFunc("formatCurrency", func(price int) string {
		parts := []string{}
		s := fmt.Sprintf("%d", price)
		for len(s) > 3 {
			parts = append([]string{s[len(s)-3:]}, parts...)
			s = s[:len(s)-3]
		}
		if len(s) > 0 {
			parts = append([]string{s}, parts...)
		}
		return strings.Join(parts, ".")
	})

	engine.AddFunc("formatDate", func(dateStr string) string {
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return dateStr
		}
		return fmt.Sprintf("%d %s %d", t.Day(), t.Month().String()[:3], t.Year())
	})

	engine.AddFunc("formatDateID", func(dateStr string) string {
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return dateStr
		}
		months := map[string]string{
			"Jan": "Jan", "Feb": "Feb", "Mar": "Mar", "Apr": "Apr",
			"May": "Mei", "Jun": "Jun", "Jul": "Jul", "Aug": "Agu",
			"Sep": "Sep", "Oct": "Okt", "Nov": "Nov", "Dec": "Des",
		}
		return fmt.Sprintf("%d %s %d", t.Day(), months[t.Month().String()[:3]], t.Year())
	})

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Static("/static", "./web/static")

	repositories.InitDB()

	app.Get("/", handlers.HomePage)
	app.Post("/recommendations", handlers.GetRecommendations)

	log.Fatal(app.Listen(":3000"))
}
