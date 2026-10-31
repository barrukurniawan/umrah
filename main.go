package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"umrah/app/accounts"
	"umrah/app/analytics"
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

	// Untuk template admin: {{template "kpi" (dict "Label" "..." "S" .X)}} dan nomor urut {{inc $i}}.
	engine.AddFunc("dict", func(kv ...any) map[string]any {
		m := make(map[string]any, len(kv)/2)
		for i := 0; i+1 < len(kv); i += 2 {
			m[fmt.Sprint(kv[i])] = kv[i+1]
		}
		return m
	})
	engine.AddFunc("inc", func(i int) int { return i + 1 })
	// Halaman tabungan: lingkar progres, tonggak, persen DP, tombol nominal cepat.
	engine.AddFunc("ringOffset", func(pct int) string { return fmt.Sprintf("%.1f", 326.7*(1-float64(min(max(pct, 0), 100))/100)) })
	engine.AddFunc("milestones", func() []int { return []int{25, 50, 75, 100} })
	engine.AddFunc("pct", func(a, b int64) int {
		if b <= 0 {
			return 0
		}
		return int(min(100, a*100/b))
	})
	engine.AddFunc("int", func(n int64) int { return int(n) })
	engine.AddFunc("quickAmounts", func() []struct {
		Value int
		Label string
	} {
		return []struct {
			Value int
			Label string
		}{{100000, "100rb"}, {250000, "250rb"}, {500000, "500rb"}, {1000000, "1 jt"}, {5000000, "5 jt"}}
	})
	// Rupiah untuk int64 (tabungan) dan tanggal "2006-01-02" → "15 Jan 2027".
	engine.AddFunc("rupiah", func(n int64) string {
		neg := n < 0
		if neg {
			n = -n
		}
		s := fmt.Sprintf("%d", n)
		var parts []string
		for len(s) > 3 {
			parts = append([]string{s[len(s)-3:]}, parts...)
			s = s[:len(s)-3]
		}
		out := strings.Join(append([]string{s}, parts...), ".")
		if neg {
			return "-" + out
		}
		return out
	})
	// Header akun: huruf awal (avatar tanpa foto) dan nama depan.
	engine.AddFunc("initial", func(name string) string {
		for _, r := range strings.TrimSpace(name) {
			return strings.ToUpper(string(r))
		}
		return "?"
	})
	engine.AddFunc("firstName", func(name string) string {
		if f := strings.Fields(name); len(f) > 0 {
			return f[0]
		}
		return "Akun"
	})
	// Nomor urut kartu di seluruh hasil (5 per halaman), untuk analitik posisi klik.
	engine.AddFunc("cardPos", func(page, i int) int { return (max(page, 1)-1)*5 + i + 1 })

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
			"Hamdan Tour":               "https://hamdantour.id",
			"Taiba Medina":              "https://taibamedina.com",
			"Al Hijaz":                  "https://alhijaz.co",
			"Marwa Mustajab":            "https://umrohmustajab.com",
			"UMI Tour & Travel":         "https://umi.travel",
			"Umrah Bisa":                "https://umrahbisa.com",
			"Khasanah Travel":           "https://www.khasanahtravel.com",
			"PT Labbaika Cipta Imani":   "https://labbaika.co.id/",
			"Rahmah Travel":             "https://rahmahtravel.id",
			"PT Wisata Hati Universal":  "https://fandiegotravel.com/",
			"Uhud Tour":                 "https://uhudtour.com",
			"Lafaya Travel":             "https://www.lafayatravel.com",
			"Jejak Imani":               "https://www.jejakimani.com",
			"Alviandra Travel":          "https://www.instagram.com/alviandratravel",
			"Rabbani Tour":              "https://rabbanitour.com",
		}
		if u, ok := urls[name]; ok {
			return u
		}
		return "#"
	})

	engine.AddFunc("whatsappURL", func(name string) string {
		urls := map[string]string{
			"Hamdan Tour":               "https://api.whatsapp.com/send/?phone=6282120009897&text=Assalamualaikum%2C+mohon+informasi+umrah+dari+HAMDAN+TOUR",
			"Taiba Medina":              "https://api.whatsapp.com/send/?phone=6285380000883&text=Assalamu%27alaikum%2C+saya+ingin+bertanya+mengenai+layanan+Umroh+Taiba+Medina+Tour",
			"Al Hijaz":                  "https://api.whatsapp.com/send/?text=%2AALHIJAZ+INDOWISATA%2A%0AAssalamualaikum%2C+saya+ingin+informasi+paket+umroh",
			"Marwa Mustajab":            "https://api.whatsapp.com/send/?phone=6288214793139&text=Assalamualaikum%2C+saya+ingin+informasi+paket+umroh",
			"Umrah Bisa":                "https://api.whatsapp.com/send/?phone=628159888154&text=Assalamualaikum%2C+Boleh+info+lebih+lanjut+untuk+paket+umrahnya%3F",
			"Khasanah Travel":           "https://api.whatsapp.com/send/?phone=6282124144331&text=halo%20saya%20ingin%20tanya%20jadwal%20umroh&type=phone_number&app_absent=0",
			"PT Labbaika Cipta Imani":   "https://api.whatsapp.com/send/?phone=6281313557780&text=%5BK-44JWD%5D%0AAssalamu%27alaikum%2C+saya+melihat+informasi+Umroh+di+website+dan+tertarik+dengan+paket+yang+tersedia.+Mohon+informasinya.&type=phone_number&app_absent=0",
			"Rahmah Travel":             "https://api.whatsapp.com/send/?phone=6282246386908&text=%5BK-44JWD%5D%0AAssalamu%27alaikum%2C+saya+melihat+informasi+Umroh+di+website+dan+tertarik+dengan+paket+yang+tersedia.+Mohon+informasinya.&type=phone_number&app_absent=0",
			"PT Wisata Hati Universal":  "https://api.whatsapp.com/send/?phone=628111718567&text=halo%20saya%20ingin%20tanya%20jadwal%20umroh&type=phone_number&app_absent=0",
			"Uhud Tour":                 "https://api.whatsapp.com/send/?phone=6281113304343&text=Assalamualaikum%2C+saya+ingin+informasi+paket+umroh+Uhud+Tour",
			"Lafaya Travel":             "https://api.whatsapp.com/send?phone=6281977382091&text=Contact%20CS%20Lafaya%20Travel",
			"UMI Tour & Travel":         "https://api.whatsapp.com/send/?phone=628159888154&text=Assalamualaikum%2C+saya+ingin+informasi+paket+umroh+UMI+Tour",
			"Jejak Imani":               "https://api.whatsapp.com/send/?phone=6281519898880&text=Assalamualaikum%20Susi%2C%20saya%20mau%20tanya%20info%20paket%20umroh%20di%20Jejak%20Imani.%20Saya%20lihat%20dari%20website.",
			"Rabbani Tour":              "https://api.whatsapp.com/send/?phone=6282280001799&text=Assalamualaikum%2C+saya+ingin+informasi+paket+umroh+Rabbani+Tour",
			"Alviandra Travel":          "https://api.whatsapp.com/send/?phone=6285183032548&text=Assalamualaikum%2C+saya+ingin+informasi+paket+Umroh+Plus+City+Tour+Oman+%2815+Januari+2027%29.+Saya+lihat+dari+Umrohku.",
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
	analyticsPath := os.Getenv("ANALYTICS_DB_PATH")
	if analyticsPath == "" {
		analyticsPath = "data/analytics.db"
	}
	analytics.Init(analyticsPath)
	geoPath := os.Getenv("GEOIP_DB_PATH")
	if geoPath == "" {
		geoPath = "data/dbip-city-lite.mmdb"
	}
	analytics.InitGeo(geoPath)

	usersPath := os.Getenv("USERS_DB_PATH")
	if usersPath == "" {
		usersPath = "data/users.db"
	}
	accounts.Init(usersPath)

	app.Use(analytics.ViewMiddleware)
	app.Use(accounts.Middleware)
	app.Get("/", handlers.HomePage)
	app.Post("/recommendations", handlers.GetRecommendations)
	app.Post("/chat", handlers.HandleChat)
	app.Post("/t", analytics.BeaconHandler)
	app.Get("/privasi", handlers.PrivacyPage)

	// Akun (opsional): login Google, persetujuan, paket tersimpan, unduh/hapus data.
	app.Get("/auth/google/login", accounts.GoogleLogin)
	app.Get("/auth/google/callback", accounts.GoogleCallback)
	app.Post("/auth/logout", accounts.Logout)
	app.Get("/akun/persetujuan", handlers.ConsentPage)
	app.Post("/akun/persetujuan", handlers.ConsentAccept)
	app.Post("/akun/persetujuan/tolak", handlers.ConsentDecline)
	app.Get("/akun", handlers.AccountPage)
	app.Get("/akun/data.json", handlers.AccountExport)
	app.Post("/akun/hapus", handlers.AccountDelete)
	app.Get("/tersimpan", handlers.SavedPage)
	app.Post("/api/saved/toggle", handlers.SavedToggle)
	app.Post("/api/saved/import", handlers.SavedImport)

	// Tabungan umroh (perlu login).
	app.Get("/tabungan", handlers.SavingsList)
	app.Get("/tabungan/baru", handlers.SavingsNew)
	app.Post("/tabungan", handlers.SavingsCreate)
	app.Get("/tabungan/:id<int>", handlers.SavingsDetail)
	app.Get("/tabungan/:id<int>/ubah", handlers.SavingsEdit)
	app.Post("/tabungan/:id<int>/ubah", handlers.SavingsUpdate)
	app.Post("/tabungan/:id<int>/hapus", handlers.SavingsDelete)
	app.Post("/tabungan/:id<int>/setoran", handlers.EntryAdd)
	app.Post("/tabungan/:id<int>/setoran/:eid<int>/hapus", handlers.EntryDelete)
	app.Post("/privasi/opt-out", handlers.PrivacyOptOut)
	app.Post("/privasi/opt-in", handlers.PrivacyOptIn)

	app.Get("/admin", handlers.AdminDashboard)
	app.Get("/admin/login", handlers.AdminLoginPage)
	app.Post("/admin/login", handlers.AdminLogin)
	app.Post("/admin/logout", handlers.AdminLogout)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "3000"
	}
	log.Fatal(app.Listen(":" + port))
}
