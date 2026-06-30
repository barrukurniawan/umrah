package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"umrah/app/handlers"
	"umrah/app/repositories"
	"umrah/app/services"

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
		months := map[string]string{
			"Jan": "Jan", "Feb": "Feb", "Mar": "Mar", "Apr": "Apr",
			"May": "Mei", "Jun": "Jun", "Jul": "Jul", "Aug": "Agu",
			"Sep": "Sep", "Oct": "Okt", "Nov": "Nov", "Dec": "Des",
		}
		return fmt.Sprintf("%d %s %d", t.Day(), months[t.Month().String()[:3]], t.Year())
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

	engine.AddFunc("travelURL", func(name string) string {
		urls := map[string]string{
			"Hamdan Tour":       "https://hamdantour.id",
			"Taiba Medina":      "https://taibamedina.com",
			"Al Hijaz":          "https://alhijaz.co",
			"Marwa Mustajab":    "https://umrohmustajab.com",
			"Namira Travel":     "https://namira.travel",
			"UMI Tour & Travel":       "https://umi.travel",
			"Rabbani Tour":            "https://rabbanitour.com",
			"Umrah Bisa":              "https://umrahbisa.com",
			"Khasanah Travel":         "https://www.khasanahtravel.com",
			"Labbaika Umrah":          "https://labbaika.co.id/",
			"Rahmah Travel":           "https://rahmahtravel.id",
			"Fandiego Tours & Travel": "https://fandiegotravel.com/",
			"Uhud Tour":               "https://uhudtour.com",
			"Lafaya Travel":           "https://www.lafayatravel.com",
		}
		if u, ok := urls[name]; ok {
			return u
		}
		return "#"
	})

	engine.AddFunc("whatsappURL", func(name string) string {
		urls := map[string]string{
			"Hamdan Tour":    "https://api.whatsapp.com/send/?phone=6282120009897&text=Assalamualaikum%2C+mohon+informasi+umrah+dari+HAMDAN+TOUR",
			"Taiba Medina":   "https://api.whatsapp.com/send/?phone=6285380000883&text=Assalamu%27alaikum%2C+saya+ingin+bertanya+mengenai+layanan+Umroh+Taiba+Medina+Tour",
			"Al Hijaz":       "https://api.whatsapp.com/send/?text=%2AALHIJAZ+INDOWISATA%2A%0AAssalamualaikum%2C+saya+ingin+informasi+paket+umroh",
			"Marwa Mustajab":          "https://api.whatsapp.com/send/?phone=6288214793139&text=Assalamualaikum%2C+saya+ingin+informasi+paket+umroh",
			"Rabbani Tour":            "https://api.whatsapp.com/send/?phone=6281297505402&text=Assalamu%27alaikum%2C+saya+ingin+konsultasi+umroh",
			"Umrah Bisa":              "https://api.whatsapp.com/send/?phone=628159888154&text=Assalamualaikum%2C+Boleh+info+lebih+lanjut+untuk+paket+umrahnya%3F",
			"Khasanah Travel":         "https://api.whatsapp.com/send/?phone=6282124144331&text=halo%20saya%20ingin%20tanya%20jadwal%20umroh&type=phone_number&app_absent=0",
			"Labbaika Umrah":          "https://api.whatsapp.com/send/?phone=6281313557780&text=%5BK-44JWD%5D%0AAssalamu%27alaikum%2C+saya+melihat+informasi+Umroh+di+website+dan+tertarik+dengan+paket+yang+tersedia.+Mohon+informasinya.&type=phone_number&app_absent=0",
			"Rahmah Travel":           "https://api.whatsapp.com/send/?phone=6282246386908&text=%5BK-44JWD%5D%0AAssalamu%27alaikum%2C+saya+melihat+informasi+Umroh+di+website+dan+tertarik+dengan+paket+yang+tersedia.+Mohon+informasinya.&type=phone_number&app_absent=0",
			"Fandiego Tours & Travel": "https://api.whatsapp.com/send/?phone=628111718567&text=halo%20saya%20ingin%20tanya%20jadwal%20umroh&type=phone_number&app_absent=0",
			"Uhud Tour":               "https://api.whatsapp.com/send/?phone=6281113304343&text=Assalamualaikum%2C+saya+ingin+informasi+paket+umroh+Uhud+Tour",
			"Lafaya Travel":           "https://api.whatsapp.com/send?phone=6281977382091&text=Contact%20CS%20Lafaya%20Travel",
		}
		if u, ok := urls[name]; ok {
			return u
		}
		return "#"
	})

	engine.AddFunc("gmapsMakkahURL", func(hotel string) string {
		if hotel == "" {
			return "#"
		}
		return "https://www.google.com/maps/dir/Kaaba,+Mecca,+Saudi+Arabia/" + strings.ReplaceAll(hotel, " ", "+") + ",+Mecca,+Saudi+Arabia"
	})

	engine.AddFunc("gmapsMadinahURL", func(hotel string) string {
		if hotel == "" {
			return "#"
		}
		return "https://www.google.com/maps/dir/Masjid+Nabawi,+Madinah,+Saudi+Arabia/" + strings.ReplaceAll(hotel, " ", "+") + ",+Madinah,+Saudi+Arabia"
	})

	engine.AddFunc("airlineURL", func(name string) string {
		urls := map[string]string{
			"Garuda Indonesia":    "https://www.garuda-indonesia.com",
			"Saudia":              "https://www.saudia.com",
			"Saudi Airlines":      "https://www.saudia.com",
			"Saudia Airlines":     "https://www.saudia.com",
			"Qatar Airways":       "https://www.qatarairways.com",
			"Emirates":            "https://www.emirates.com",
			"Etihad Airways":      "https://www.etihad.com",
			"Oman Air":            "https://www.omanair.com",
			"Oman Airlines":       "https://www.omanair.com",
			"Turkish Airlines":    "https://www.turkishairlines.com",
			"Royal Brunei Airlines": "https://www.flyroyalbrunei.com",
			"Scoot":               "https://www.flyscoot.com",
			"IndiGo":              "https://www.goindigo.in",
			"EgyptAir":            "https://www.egyptair.com",
			"AirAsia":             "https://www.airasia.com",
		}
		for k, u := range urls {
			if strings.Contains(name, k) {
				return u
			}
		}
		return "#"
	})

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Static("/static", "./web/static")

	repositories.InitDB()
	services.InitAI()

	app.Get("/", handlers.HomePage)
	app.Post("/recommendations", handlers.GetRecommendations)
	app.Post("/chat", handlers.HandleChat)

	log.Fatal(app.Listen(":3000"))
}
