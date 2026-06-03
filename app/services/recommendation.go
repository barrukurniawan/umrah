package services

import (
	"encoding/json"
	"sort"
	"strings"

	"umrah/app/models"
	"umrah/app/repositories"
)

type FilterInput struct {
	Budget   int
	Priority string
	Advanced []string
}

type ScoredPackage struct {
	models.Package
	Score        int
	FacilityList []string
}

func GetRecommendations(input FilterInput) []ScoredPackage {
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

		if input.Priority == "advanced" {
			if advanceFilter["direct"] && !pkg.IsDirect {
				continue
			}
			if advanceFilter["transit"] && pkg.IsDirect {
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

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	if len(scored) > 10 {
		scored = scored[:10]
	}

	return scored
}

func parseFacilities(raw string) []string {
	var list []string
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return []string{}
	}
	return list
}
