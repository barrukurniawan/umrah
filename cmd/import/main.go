package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"umrah/app/models"
	"umrah/app/repositories"

	"gorm.io/gorm"
)

type CrawlResult struct {
	Timestamp string                  `json:"timestamp"`
	Site      string                  `json:"site"`
	URL       string                  `json:"url"`
	Packages  []CrawledPackage        `json:"packages"`
	Error     string                  `json:"error,omitempty"`
}

type CrawledPackage struct {
	TravelName     string   `json:"travel_name"`
	PackageName    string   `json:"package_name"`
	Price          int      `json:"price"`
	Duration       int      `json:"duration"`
	Airline        string   `json:"airline"`
	HotelMakkah    string   `json:"hotel_makkah"`
	HotelMadinah   string   `json:"hotel_madinah"`
	DepartureDates []string `json:"departure_dates"`
	Seats          int      `json:"seats"`
	Airport        string   `json:"airport"`
	URL            string   `json:"url"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: import <json_file> [json_file2 ...]")
	}

	repositories.InitDB()
	db := repositories.DB

	for _, path := range os.Args[1:] {
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("skip %s: %v", path, err)
			continue
		}

		var results []CrawlResult
		if err := json.Unmarshal(data, &results); err != nil {
			var single CrawlResult
			if err := json.Unmarshal(data, &single); err != nil {
				log.Printf("skip %s: invalid format", path)
				continue
			}
			results = []CrawlResult{single}
		}

		for _, r := range results {
			log.Printf("importing %d packages from %s", len(r.Packages), r.Site)
			for _, p := range r.Packages {
				importPackage(db, r.Site, p)
			}
		}
	}

	log.Println("import selesai")
}

func importPackage(db *gorm.DB, site string, cp CrawledPackage) {
	travelName := cp.TravelName
	if travelName == "" {
		travelName = site
	}

	if cp.Price > 0 && cp.Price < 5000000 {
		log.Printf("[skip] %s - %s: harga Rp %d terlalu kecil, kemungkinan data error", travelName, cp.PackageName, cp.Price)
		return
	}

	cp = enrichPackage(travelName, cp)

	var travel models.Travel
	db.Where("name = ?", travelName).FirstOrCreate(&travel, models.Travel{
		Name:   travelName,
		Rating: estimateRating(travelName),
	})

	dp := cp.Price / 5
	if travelName == "Khasanah Travel" {
		dp = 14000000
	} else if travelName == "PT Labbaika Cipta Imani" {
		dp = 15000000
	} else if travelName == "Rahmah Travel" {
		dp = 7500000
	} else if travelName == "Umrah Bisa" {
		dp = 5000000
	} else if travelName == "PT Wisata Hati Universal" {
		dp = 5000000
	} else if travelName == "Marwa Mustajab" {
		dp = 5000000
	} else if travelName == "Taiba Medina" {
		dp = 2500000
	} else if travelName == "UMI Tour & Travel" {
		dp = 5000000
	} else if travelName == "Uhud Tour" {
		dp = 9000000
	} else if travelName == "Lafaya Travel" {
		dp = 10000000
	} else if travelName == "Al Hijaz" {
		dp = 5000000
	} else if travelName == "Hamdan Tour" {
		dp = 5000000
	} else if travelName == "Jejak Imani" {
		if strings.Contains(cp.PackageName, "Ruby") || strings.Contains(cp.PackageName, "Sapphire") {
			dp = 10000000
		} else {
			dp = 5000000
		}
	}

	pkg := models.Package{
		TravelID:      travel.ID,
		Name:          cp.PackageName,
		Price:         cp.Price,
		Duration:      cp.Duration,
		Airline:       cp.Airline,
		DownPayment:   dp,
		GroupSize:     cp.Seats,
		Guide:         "Ustadz/Ustadzah",
		HotelDistance: estimateDistance(cp.Price),
		IsDirect:      isDirectAirline(cp.Airline),
		IsNearHaram:   cp.Price > 35000000,
		IsFamily:      strings.Contains(strings.ToLower(cp.PackageName), "keluarga"),
		IsKajian:      strings.Contains(strings.ToLower(cp.PackageName), "kajian"),
		SunnahScore:   estimateSunnah(cp.Price, cp.Airline),
		Facilities:    "[]",
	}

	db.Where("travel_id = ? AND name = ?", travel.ID, cp.PackageName).FirstOrCreate(&pkg)

	if pkg.Airline == "" {
		db.Model(&pkg).Update("airline", cp.Airline)
	}
	if pkg.Duration == 0 && cp.Duration > 0 {
		db.Model(&pkg).Update("duration", cp.Duration)
	}
	db.Model(&pkg).Update("price", cp.Price)
	db.Model(&pkg).Update("down_payment", dp)
	db.Model(&pkg).Update("group_size", cp.Seats)

	cleanHotel := strings.TrimSuffix(cp.HotelMakkah, " (Makkah)")
	cleanHotel = strings.TrimSuffix(cleanHotel, " Makkah")
	cleanHotel = strings.TrimSpace(cleanHotel)

	cleanHotelMd := strings.TrimSuffix(cp.HotelMadinah, " (Madinah)")
	cleanHotelMd = strings.TrimSuffix(cleanHotelMd, " Madinah")
	cleanHotelMd = strings.TrimSpace(cleanHotelMd)

	for _, dateStr := range cp.DepartureDates {
		dates := splitDates(dateStr)
		for _, d := range dates {
			depDate := parseDeparture(d)
			if depDate == "" {
				continue
			}

			var existing models.DetailPackage
			res := db.Where("package_id = ? AND departure_date = ?", pkg.ID, depDate).First(&existing)
			if res.Error == nil {
				continue
			}

			hMakkah := cleanHotel
			hMadinah := cleanHotelMd
			starsMakkah := 3
			starsMadinah := 3

			enriched := enrichHotel(travelName, cp.PackageName, depDate)
			if enriched.Makkah != "" {
				hMakkah = enriched.Makkah
			}
			if enriched.Madinah != "" {
				hMadinah = enriched.Madinah
			}
			if enriched.StarsMakkah > 0 {
				starsMakkah = enriched.StarsMakkah
			}
			if enriched.StarsMadinah > 0 {
				starsMadinah = enriched.StarsMadinah
			}

			detail := models.DetailPackage{
				PackageID:         pkg.ID,
				DepartureDate:     depDate,
				ReturnDate:        computeReturn(depDate, cp.Duration),
				HotelMakkah:       hMakkah,
				HotelMadinah:      hMadinah,
				StarsMakkah:       starsMakkah,
				StarsMadinah:      starsMadinah,
				RoomType:          "Quad",
				TotalQuota:        cp.Seats,
				AvailableQuota:    cp.Seats,
				DepartureLocation: cp.Airport,
				Guide:             pkg.Guide,
			}
			db.Create(&detail)
		}
	}
}

func splitDates(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "+") {
			continue
		}
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return []string{s}
	}
	return result
}

var monthsID = map[string]string{
	"januari": "01", "februari": "02", "maret": "03", "april": "04",
	"mei": "05", "juni": "06", "juli": "07", "agustus": "08",
	"september": "09", "oktober": "10", "november": "11", "desember": "12",
	"jan": "01", "feb": "02", "mar": "03", "apr": "04",
	"may": "05", "jun": "06", "jul": "07", "aug": "08",
	"agu": "08", "agt": "08", "sep": "09", "okt": "10", "oct": "10",
	"nov": "11", "des": "12", "dec": "12",
}

func parseDeparture(s string) string {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "-") {
		return s
	}

	parts := strings.Fields(s)
	if len(parts) < 2 {
		return ""
	}

	day := parts[0]
	monthName := strings.ToLower(parts[1])
	year := "2026"
	if len(parts) >= 3 {
		y := parts[len(parts)-1]
		if len(y) == 4 {
			year = y
		}
	}

	m, ok := monthsID[monthName]
	if !ok {
		return ""
	}

	return fmt.Sprintf("%s-%s-%02s", year, m, day)
}

func computeReturn(depDate string, duration int) string {
	t, err := time.Parse("2006-01-02", depDate)
	if err != nil {
		return depDate
	}
	if duration <= 0 {
		duration = 9
	}
	return t.AddDate(0, 0, duration).Format("2006-01-02")
}

func estimateDistance(price int) int {
	switch {
	case price >= 40000000:
		return 150
	case price >= 35000000:
		return 250
	case price >= 30000000:
		return 400
	case price >= 27000000:
		return 800
	case price >= 25000000:
		return 800
	default:
		return 1200
	}
}

func isDirectAirline(airline string) bool {
	direct := []string{"Garuda Indonesia", "Saudia", "Saudi Airlines"}
	for _, a := range direct {
		if strings.Contains(airline, a) {
			return true
		}
	}
	return false
}

func estimateSunnah(price int, airline string) int {
	score := 5
	if price >= 40000000 {
		score += 4
	} else if price >= 35000000 {
		score += 3
	} else if price >= 30000000 {
		score += 2
	} else if price >= 27000000 {
		score += 1
	}
	if isDirectAirline(airline) {
		score += 1
	}
	if strings.Contains(strings.ToLower(airline), "qatar") || strings.Contains(strings.ToLower(airline), "emirates") {
		score += 1
	}
	if score > 10 {
		score = 10
	}
	return score
}

func estimateRating(travelName string) float64 {
	ratings := map[string]float64{
		"Hamdan Tour":       4.5,
		"Taiba Medina":      4.3,
		"Al Hijaz":          4.7,
		"Marwa Mustajab":    4.2,
		"Rabbani Tour":      4.6,
		"UMI Tour & Travel": 4.1,
		"Namira Travel":     4.2,
		"Umrah Bisa":        4.5,
		"Lafaya Travel":     4.4,
	}
	if r, ok := ratings[travelName]; ok {
		return r
	}
	return 4.0
}

type hotelInfo struct {
	Makkah      string
	Madinah     string
	StarsMakkah int
	StarsMadinah int
}

func enrichPackage(travelName string, cp CrawledPackage) CrawledPackage {
	switch travelName {
	case "Marwa Mustajab":
		if cp.Airline == "" {
			switch {
			case strings.Contains(cp.PackageName, "HEART 03 JULI"):
				cp.Airline = "Garuda Indonesia"
			case strings.Contains(cp.PackageName, "HEART 16 SEPTEMBER"):
				cp.Airline = "Saudia Airlines"
			case strings.Contains(cp.PackageName, "HEMAYA 06 OKTOBER"):
				cp.Airline = "Oman Airlines"
			case strings.Contains(cp.PackageName, "HEMAYA 25 OKTOBER"):
				cp.Airline = "Oman Airlines"
			case strings.Contains(cp.PackageName, "HEMAYA 05 NOVEMBER"):
				cp.Airline = "Oman Airlines"
			case strings.Contains(cp.PackageName, "AKHIR TAHUN 22 DESEMBER"):
				cp.Airline = "Garuda Indonesia"
			}
		}

	case "Taiba Medina":
		if cp.Airline == "" || cp.Airline == "-" || cp.Airline == "Request" {
			switch {
			case strings.Contains(cp.PackageName, "PLUS SPECIAL THAIF DIRECT"):
				cp.Airline = "Saudia Airlines"
			case strings.Contains(cp.PackageName, "MENGINAP THAIF 10D") && strings.Contains(cp.PackageName, "11 AGUSTUS"):
				cp.Airline = "Qatar Airways"
			case strings.Contains(cp.PackageName, "PLUS THAIF 10D") && strings.Contains(cp.PackageName, "14 AGUSTUS"):
				cp.Airline = "Qatar Airways"
			case strings.Contains(cp.PackageName, "MENGINAP DI THAIF 10D") && strings.Contains(cp.PackageName, "1 SEPTEMBER"):
				cp.Airline = "Garuda Indonesia"
			case strings.Contains(cp.PackageName, "PLUS SPECIAL DUBAI"):
				cp.Airline = "Emirates"
			case strings.Contains(cp.PackageName, "PLUS THAIF 9D") && strings.Contains(cp.PackageName, "4 OKTOBER"):
				cp.Airline = "Garuda Indonesia"
			case strings.Contains(cp.PackageName, "MENGINAP THAIF 10D") && strings.Contains(cp.PackageName, "27 OKTOBER"):
				cp.Airline = "Garuda Indonesia"
			case strings.Contains(cp.PackageName, "PLUS THAIF 9D") && strings.Contains(cp.PackageName, "1 NOVEMBER"):
				cp.Airline = "Garuda Indonesia"
			case strings.Contains(cp.PackageName, "MENGINAP DI THAIF 10D") && strings.Contains(cp.PackageName, "24 NOVEMBER"):
				cp.Airline = "Qatar Airways"
			case strings.Contains(cp.PackageName, "PLUS THAIF 9D") && strings.Contains(cp.PackageName, "1 DESEMBER"):
				cp.Airline = "Saudia Airlines"
			case strings.Contains(cp.PackageName, "PLUS THAIF 9D") && strings.Contains(cp.PackageName, "24 DESEMBER"):
				cp.Airline = "Oman Air"
			case strings.Contains(cp.PackageName, "MENGINAP DI THAIF 10D") && strings.Contains(cp.PackageName, "29 DESEMBER"):
				cp.Airline = "Saudia Airlines"
			}
		}

	case "UMI Tour & Travel":
		if cp.Airline == "" || cp.Airline == "N/A" || cp.Airline == "-" {
			switch {
			case strings.Contains(cp.PackageName, "Qonaah Salwa") || strings.Contains(cp.PackageName, "Muhasabah Salwa"):
				cp.Airline = "Etihad Airways"
			case strings.Contains(cp.PackageName, "2x Jumat Plus Thaif 24 Sep"):
				cp.Airline = "Saudia Airlines"
			case strings.Contains(cp.PackageName, "Sirah Nabawiyah Plus Thaif"):
				cp.Airline = "Qatar Airways"
			}
		}

	case "Umrah Bisa":
		if cp.Airline == "" {
			cp.Airline = "Oman Air"
		}
	}

	return cp
}

func enrichHotel(travelName, packageName, depDate string) hotelInfo {
	switch travelName {
	case "Taiba Medina":
		switch {
		case strings.Contains(packageName, "PLUS SPECIAL THAIF DIRECT"):
			return hotelInfo{Makkah: "Royal Majestic", Madinah: "Shaza Regency", StarsMakkah: 5, StarsMadinah: 5}
		case strings.Contains(packageName, "MENGINAP THAIF 10D") && strings.Contains(packageName, "11 AGUSTUS"):
			return hotelInfo{Makkah: "Nada Ajyad", Madinah: "Plaza Inn Ohud", StarsMakkah: 4, StarsMadinah: 3}
		case strings.Contains(packageName, "PLUS THAIF 10D") && strings.Contains(packageName, "14 AGUSTUS"):
			return hotelInfo{Makkah: "Grand Al Masa", Madinah: "Arkan Al Manar", StarsMakkah: 4, StarsMadinah: 3}
		case strings.Contains(packageName, "MENGINAP DI THAIF 10D") && strings.Contains(packageName, "1 SEPTEMBER"):
			return hotelInfo{Makkah: "Nada Ajyad", Madinah: "Plaza Inn Ohud", StarsMakkah: 4, StarsMadinah: 3}
		case strings.Contains(packageName, "PLUS SPECIAL DUBAI"):
			return hotelInfo{Makkah: "Nada Ajyad", Madinah: "Plaza Inn Ohud", StarsMakkah: 4, StarsMadinah: 3}
		case strings.Contains(packageName, "PLUS THAIF 9D") && strings.Contains(packageName, "4 OKTOBER"):
			return hotelInfo{Makkah: "Nada Ajyad", Madinah: "Plaza Inn Ohud", StarsMakkah: 4, StarsMadinah: 3}
		case strings.Contains(packageName, "MENGINAP THAIF 10D") && strings.Contains(packageName, "27 OKTOBER"):
			return hotelInfo{Makkah: "Nada Ajyad", Madinah: "Plaza Inn Ohud", StarsMakkah: 4, StarsMadinah: 3}
		case strings.Contains(packageName, "PLUS THAIF 9D") && strings.Contains(packageName, "1 NOVEMBER"):
			return hotelInfo{Makkah: "Nada Ajyad", Madinah: "Plaza Inn Ohud", StarsMakkah: 4, StarsMadinah: 3}
		case strings.Contains(packageName, "MENGINAP DI THAIF 10D") && strings.Contains(packageName, "24 NOVEMBER"):
			return hotelInfo{Makkah: "Nada Ajyad", Madinah: "Plaza Inn Ohud", StarsMakkah: 4, StarsMadinah: 3}
		case strings.Contains(packageName, "PLUS THAIF 9D") && strings.Contains(packageName, "1 DESEMBER"):
			return hotelInfo{Makkah: "Nada Ajyad", Madinah: "Plaza Inn Ohud", StarsMakkah: 4, StarsMadinah: 3}
		case strings.Contains(packageName, "PLUS THAIF 9D") && strings.Contains(packageName, "24 DESEMBER"):
			return hotelInfo{Makkah: "Nada Ajyad", Madinah: "Plaza Inn Ohud", StarsMakkah: 4, StarsMadinah: 3}
		case strings.Contains(packageName, "MENGINAP DI THAIF 10D") && strings.Contains(packageName, "29 DESEMBER"):
			return hotelInfo{Makkah: "Nada Ajyad", Madinah: "Plaza Inn Ohud", StarsMakkah: 4, StarsMadinah: 3}
		}

	case "UMI Tour & Travel":
		switch {
		case strings.Contains(packageName, "Qonaah Salwa") || strings.Contains(packageName, "Muhasabah Salwa"):
			return hotelInfo{Makkah: "Al Massa Dar Fayzeen", Madinah: "Dar Al-Naeem Hotel", StarsMakkah: 4, StarsMadinah: 3}
		case strings.Contains(packageName, "2x Jumat Plus Thaif 24 Sep"):
			return hotelInfo{Makkah: "Elaf Kinda Hotel", Madinah: "Hotel Ritz Madinah", StarsMakkah: 5, StarsMadinah: 4}
		case strings.Contains(packageName, "Sirah Nabawiyah Plus Thaif"):
			return hotelInfo{Makkah: "Prestige Al Mashaer Hotel", Madinah: "Hotel Ritz Madinah", StarsMakkah: 5, StarsMadinah: 4}
		}

	case "Umrah Bisa":
		return hotelInfo{Makkah: "Grand Al Massa", Madinah: "Jawharat Ar Rasheed", StarsMakkah: 4, StarsMadinah: 3}

	case "Uhud Tour":
		return hotelInfo{Makkah: "Shohada Hotel", Madinah: "Concorde Dar Al Khair", StarsMakkah: 5, StarsMadinah: 4}

	case "PT Wisata Hati Universal":
		switch {
		case strings.Contains(packageName, "Hemat"):
			return hotelInfo{Makkah: "Le Meridien Tower", Madinah: "One Inn Hotel", StarsMakkah: 4, StarsMadinah: 3}
		case strings.Contains(packageName, "Reguler"):
			return hotelInfo{Makkah: "Grand Al Massa", Madinah: "One Inn Hotel", StarsMakkah: 4, StarsMadinah: 3}
		case strings.Contains(packageName, "Gold"):
			return hotelInfo{Makkah: "Prestige Hotel", Madinah: "One Inn Hotel", StarsMakkah: 5, StarsMadinah: 3}
		case strings.Contains(packageName, "Diamond"):
			return hotelInfo{Makkah: "Pullman Hotel", Madinah: "Al Haram", StarsMakkah: 5, StarsMadinah: 5}
		}

	case "Marwa Mustajab":
		switch {
		case strings.Contains(packageName, "HEART"):
			return hotelInfo{Makkah: "Movenpick Hajar Tower", Madinah: "Frontel Al Harithia", StarsMakkah: 5, StarsMadinah: 5}
		case strings.Contains(packageName, "HEMAYA"):
			return hotelInfo{Makkah: "Nada Ajyad", Madinah: "ODST Al Madinah", StarsMakkah: 3, StarsMadinah: 3}
		case strings.Contains(packageName, "AKHIR TAHUN"):
			return hotelInfo{Makkah: "Movenpick Hajar Tower", Madinah: "Frontel Al Harithia", StarsMakkah: 5, StarsMadinah: 5}
		case strings.Contains(packageName, "ITTIKAF"):
			return hotelInfo{Makkah: "Nada Ajyad", Madinah: "ODST Al Madinah", StarsMakkah: 3, StarsMadinah: 3}
		}
	}

	return hotelInfo{}
}
