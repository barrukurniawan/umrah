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
}

type ScoredPackage struct {
	models.Package
	Score        int
	FacilityList []string
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

	var scored []ScoredPackage
	for _, pkg := range allPackages {
		if pkg.Price > input.Budget {
			continue
		}

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

		if input.Priority == "all" {
			score += 10
		}

		switch input.Priority {
		case "near_haram":
			if pkg.IsNearHaram {
				score += 30
			}
			if pkg.HotelDistance <= 300 {
				score += 10
			}
		case "family_friendly":
			if pkg.Travel.Name == "Taiba Medina" {
				score += 35
			}
			if pkg.IsKidFriendly || pkg.IsSeniorFriendly {
				score += 20
			}
			if pkg.IsDirect {
				score += 15
			}
			if pkg.IsFamily || pkg.IsSenior {
				score += 10
			}
			if pkg.HotelDistance <= 400 {
				score += 10
			} else if pkg.IsKidFriendly || pkg.IsSeniorFriendly {
				score += 5
			}
		case "advanced":
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
				}
			}
			if len(advanceFilter) == 0 {
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
			Package:      pkg,
			Score:        score,
			FacilityList: parseFacilities(pkg.Facilities),
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
		default:
			return scored[i].Score > scored[j].Score
		}
	})

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
