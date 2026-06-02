package main

import (
	"fmt"
	"log"
	"strings"

	"umrah/app/handlers"
	"umrah/app/repositories"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

func main() {
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

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Static("/static", "./web/static")

	repositories.InitDB()

	app.Get("/", handlers.HomePage)
	app.Post("/recommendations", handlers.GetRecommendations)

	log.Fatal(app.Listen(":3000"))
}
