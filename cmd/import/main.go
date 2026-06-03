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

	var travel models.Travel
	db.Where("name = ?", travelName).FirstOrCreate(&travel, models.Travel{
		Name:   travelName,
		Rating: estimateRating(travelName),
	})

	pkg := models.Package{
		TravelID:      travel.ID,
		Name:          cp.PackageName,
		Price:         cp.Price,
		Duration:      cp.Duration,
		Airline:       cp.Airline,
		DownPayment:   cp.Price / 5,
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

			detail := models.DetailPackage{
				PackageID:         pkg.ID,
				DepartureDate:     depDate,
				ReturnDate:        computeReturn(depDate, cp.Duration),
				HotelMakkah:       cleanHotel,
				HotelMadinah:      cleanHotelMd,
				StarsMakkah:       3,
				StarsMadinah:      3,
				RoomType:          "Quad",
				TotalQuota:        cp.Seats,
				AvailableQuota:    cp.Seats,
				DepartureLocation: cp.Airport,
				Guide:             pkg.Guide,
			}
			db.Create(&detail)
		}
	}

	if len(cp.DepartureDates) == 0 {
		placeholder := time.Now().AddDate(0, 6, 0).Format("2006-01-02")
		db.Create(&models.DetailPackage{
			PackageID:         pkg.ID,
			DepartureDate:     placeholder,
			ReturnDate:        computeReturn(placeholder, cp.Duration),
			HotelMakkah:       cleanHotel,
			HotelMadinah:      cleanHotelMd,
			StarsMakkah:       3,
			StarsMadinah:      3,
			RoomType:          "Quad",
			TotalQuota:        cp.Seats,
			AvailableQuota:    cp.Seats,
			DepartureLocation: cp.Airport,
			Guide:             pkg.Guide,
		})
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
		return 600
	case price >= 25000000:
		return 800
	default:
		return 1200
	}
}

func isDirectAirline(airline string) bool {
	direct := []string{"Garuda Indonesia", "Saudia", "Emirates"}
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
		"Hamdan Tour":    4.5,
		"Taiba Medina":   4.3,
		"Al Hijaz":       4.7,
		"Marwa Mustajab": 4.2,
		"Rabbani Tour":   4.6,
		"UMI Tour & Travel": 4.1,
		"Namira Travel":  4.2,
	}
	if r, ok := ratings[travelName]; ok {
		return r
	}
	return 4.0
}
