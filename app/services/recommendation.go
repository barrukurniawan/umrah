package services

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"umrah/app/models"
	"umrah/app/repositories"
)

type FilterInput struct {
	Budget   int
	Priority string
	Advanced []string
	Page     int
	Sort     string
	Month    string
	Travels  string
}

type ScoredPackage struct {
	models.Package
	Score             int
	FacilityList      []string
	DepartureDatesStr string
	DistanceMakkah    string
	DistanceMadinah   string
}

func GetRecommendations(input FilterInput) ([]ScoredPackage, int) {
	var allPackages []models.Package
	repositories.DB.Preload("Travel").Preload("Details").Find(&allPackages)

	advanceFilter := make(map[string]bool)
	for _, a := range input.Advanced {
		if a != "" {
			advanceFilter[a] = true
		}
	}

	travelFilter := make(map[string]bool)
	hasTravelFilter := false
	if input.Travels != "" {
		for _, t := range strings.Split(input.Travels, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				travelFilter[t] = true
				hasTravelFilter = true
			}
		}
	}

	var scored []ScoredPackage
	for _, pkg := range allPackages {
		// Skip packages with no valid departure dates
		hasValidDate := false
		for _, d := range pkg.Details {
			if d.DepartureDate != "" {
				hasValidDate = true
				break
			}
		}
		if !hasValidDate {
			continue
		}

		if pkg.Price > input.Budget {
			continue
		}

		if hasTravelFilter && !travelFilter[pkg.Travel.Name] {
			continue
		}

		distMakkah := "700-1000m"
		distMadinah := "200-400m"
		if len(pkg.Details) > 0 {
			distMakkah = getDistanceMakkah(pkg.Details[0].HotelMakkah)
			distMadinah = getDistanceMadinah(pkg.Details[0].HotelMadinah)
		}

		// Override DB values dynamically
		pkg.HotelDistance = parseMinDistance(distMakkah)
		pkg.IsNearHaram = pkg.HotelDistance <= 550

		// Apply advanced filters always if set (regardless of priority)
		if advanceFilter["direct"] && !pkg.IsDirect {
			continue
		}
		if advanceFilter["transit"] && pkg.IsDirect {
			continue
		}
		if advanceFilter["near_haram"] && !pkg.IsNearHaram {
			continue
		}
		if advanceFilter["family_friendly"] && !pkg.IsKidFriendly && !pkg.IsSeniorFriendly {
			continue
		}
		if advanceFilter["quad"] || advanceFilter["triple"] || advanceFilter["double"] {
			hasRoom := false
			for _, d := range pkg.Details {
				rt := strings.ToLower(d.RoomType)
				if (advanceFilter["quad"] && rt == "quad") ||
					(advanceFilter["triple"] && rt == "triple") ||
					(advanceFilter["double"] && rt == "double") {
					hasRoom = true
					break
				}
			}
			if !hasRoom {
				continue
			}
		}

		score := 0

		if advanceFilter["near_haram"] {
			if pkg.IsNearHaram {
				score += 20
			}
			if pkg.HotelDistance <= 300 {
				score += 10
			}
		}
		if advanceFilter["family_friendly"] {
			if pkg.Travel.Name == "Taiba Medina" {
				score += 25
			}
			if pkg.IsKidFriendly || pkg.IsSeniorFriendly {
				score += 15
			}
			if pkg.IsDirect {
				score += 10
			}
			if pkg.IsFamily || pkg.IsSenior {
				score += 10
			}
			if pkg.HotelDistance <= 400 {
				score += 10
			} else {
				score += 5
			}
		}

		if pkg.SunnahScore >= 8 {
			score += 5
		}
		if pkg.HotelDistance <= 500 {
			score += 5
		}
		if pkg.Travel.Rating >= 4.5 {
			score += 5
		}

		scored = append(scored, ScoredPackage{
			Package:           pkg,
			Score:             score,
			FacilityList:      parseFacilities(pkg.Facilities),
			DepartureDatesStr: formatDepartureDatesStr(pkg.Details),
			DistanceMakkah:    distMakkah,
			DistanceMadinah:   distMadinah,
		})
	}

	// Server-side sort
	sort.Slice(scored, func(i, j int) bool {
		switch input.Sort {
		case "price_asc":
			return scored[i].Price < scored[j].Price
		case "price_desc":
			return scored[i].Price > scored[j].Price
		case "distance_asc":
			return scored[i].HotelDistance < scored[j].HotelDistance
		case "duration_asc":
			return scored[i].Duration < scored[j].Duration
		case "duration_desc":
			return scored[i].Duration > scored[j].Duration
		case "dp_asc":
			return scored[i].DownPayment < scored[j].DownPayment
		case "direct_only":
			return scored[i].Price < scored[j].Price
		default:
			return scored[i].Score > scored[j].Score
		}
	})

	// Filter direct only
	if input.Sort == "direct_only" {
		filtered := make([]ScoredPackage, 0)
		for _, sp := range scored {
			if sp.IsDirect {
				filtered = append(filtered, sp)
			}
		}
		scored = filtered
	}

	// Server-side month filter
	if input.Month != "" {
		filtered := make([]ScoredPackage, 0)
		now := time.Now()
		currentMonth := int(now.Month())
		selMonth, _ := strconv.Atoi(input.Month)
		targetYear := now.Year()
		if selMonth < currentMonth {
			targetYear++
		}
		targetPrefix := fmt.Sprintf("%d-%02d", targetYear, selMonth)

		for _, sp := range scored {
			for _, d := range sp.Details {
				if strings.HasPrefix(d.DepartureDate, targetPrefix) {
					filtered = append(filtered, sp)
					break
				}
			}
		}
		scored = filtered
	}

	total := len(scored)

	if input.Page < 1 {
		input.Page = 1
	}
	start := (input.Page - 1) * 5
	if start >= len(scored) {
		return []ScoredPackage{}, total
	}
	end := start + 5
	if end > len(scored) {
		end = len(scored)
	}

	return scored[start:end], total
}

func parseFacilities(raw string) []string {
	var list []string
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return []string{}
	}
	return list
}

func formatDepartureDatesStr(details []models.DetailPackage) string {
	if len(details) == 0 {
		return "-"
	}

	var dates []time.Time
	seen := make(map[string]bool)
	for _, d := range details {
		if d.DepartureDate == "" {
			continue
		}
		if !seen[d.DepartureDate] {
			seen[d.DepartureDate] = true
			if t, err := time.Parse("2006-01-02", d.DepartureDate); err == nil {
				dates = append(dates, t)
			}
		}
	}

	if len(dates) == 0 {
		return "-"
	}

	sort.Slice(dates, func(i, j int) bool {
		return dates[i].Before(dates[j])
	})

	months := map[time.Month]string{
		time.January: "Jan", time.February: "Feb", time.March: "Mar", time.April: "Apr",
		time.May: "Mei", time.June: "Jun", time.July: "Jul", time.August: "Agu",
		time.September: "Sep", time.October: "Okt", time.November: "Nov", time.December: "Des",
	}

	var parts []string
	type group struct {
		Month time.Month
		Year  int
		Days  []int
	}
	var groups []group

	for _, d := range dates {
		if len(groups) > 0 {
			last := &groups[len(groups)-1]
			if last.Month == d.Month() && last.Year == d.Year() {
				last.Days = append(last.Days, d.Day())
				continue
			}
		}
		groups = append(groups, group{Month: d.Month(), Year: d.Year(), Days: []int{d.Day()}})
	}

	for i, g := range groups {
		var daysStr []string
		for _, day := range g.Days {
			daysStr = append(daysStr, strconv.Itoa(day))
		}
		daysJoined := strings.Join(daysStr, ", ")

		str := fmt.Sprintf("%s %s", daysJoined, months[g.Month])
		if i == len(groups)-1 {
			str += fmt.Sprintf(" %d", g.Year)
		} else if groups[i].Year != groups[len(groups)-1].Year {
			str += fmt.Sprintf(" %d", g.Year)
		}
		parts = append(parts, str)
	}

	return strings.Join(parts, ", ")
}

func getDistanceMakkah(hotel string) string {
	h := strings.ToLower(hotel)
	if strings.Contains(h, "marwa rotana") {
		return "250m"
	}
	if strings.Contains(h, "zamzam") {
		return "150-250m"
	}
	if strings.Contains(h, "azka") || strings.Contains(h, "safa") {
		return "250-350m"
	}
	if strings.Contains(h, "anjum") {
		return "450-550m"
	}
	if strings.Contains(h, "shohada") {
		return "600-700m"
	}
	if strings.Contains(h, "almassa grand") || strings.Contains(h, "grand al masa") || strings.Contains(h, "al massa grand") || strings.Contains(h, "ramada dar") || strings.Contains(h, "bader al massa") {
		return "700m"
	}
	if strings.Contains(h, "al massa dar") || strings.Contains(h, "faiezeen") || strings.Contains(h, "fayzeen") || strings.Contains(h, "dar faizin") {
		return "750m"
	}
	if strings.Contains(h, "jada ajyad") {
		return "750-900m"
	}
	if strings.Contains(h, "nada ajyad") {
		return "800-950m"
	}
	if strings.Contains(h, "mashaer") {
		return "850-1000m"
	}
	if strings.Contains(h, "majestic") {
		return "900-1100m"
	}
	if strings.Contains(h, "le meridien tower") || strings.Contains(h, "meridien tower") {
		return "1,5 km"
	}
	if strings.Contains(h, "ibis styles") || strings.Contains(h, "ibis style") {
		return "2,5 km"
	}
	if strings.Contains(h, "fajr badea") {
		return "2 km"
	}
	return "700-1000m"
}

func getDistanceMadinah(hotel string) string {
	h := strings.ToLower(hotel)
	if strings.Contains(h, "maden taibah") || strings.Contains(h, "madinah taibah") {
		return "200m"
	}
	if strings.Contains(h, "concorde") || strings.Contains(h, "dar al khair") {
		return "120m"
	}
	if strings.Contains(h, "shaza") {
		return "180m"
	}
	if strings.Contains(h, "plaza inn") {
		return "250m"
	}
	if strings.Contains(h, "ritz") {
		return "300m"
	}
	if strings.Contains(h, "odst") {
		return "350m"
	}
	if strings.Contains(h, "jauharat") || strings.Contains(h, "jawharat") || strings.Contains(h, "rasheed") || strings.Contains(h, "rashed") {
		return "400m"
	}
	if strings.Contains(h, "arkan") || strings.Contains(h, "manar") {
		return "500m"
	}
	if strings.Contains(h, "al ansar golden") || strings.Contains(h, "golden tulip") {
		return "600m"
	}
	if strings.Contains(h, "almukhtaro") || strings.Contains(h, "al mukhtara") || strings.Contains(h, "alghorbi") || strings.Contains(h, "al gharbi") {
		return "1,2 km"
	}
	if strings.Contains(h, "sham province") {
		return "650m"
	}
	if strings.Contains(h, "sky view") {
		return "1,5 km"
	}
	return "200-400m"
}

func parseMinDistance(dist string) int {
	if strings.Contains(dist, "km") {
		d := strings.ReplaceAll(dist, "km", "")
		d = strings.ReplaceAll(d, ",", ".")
		d = strings.TrimSpace(d)
		val, _ := strconv.ParseFloat(d, 64)
		return int(val * 1000)
	}
	d := strings.Split(dist, "-")[0]
	d = strings.ReplaceAll(d, "m", "")
	d = strings.TrimSpace(d)
	val, _ := strconv.Atoi(d)
	if val == 0 {
		return 1000
	}
	return val
}
