package services

import (
	"sort"

	"umrah/app/models"
	"umrah/app/repositories"
)

type FilterInput struct {
	Budget    int
	Who       string
	Priority  string
}

type ScoredPackage struct {
	models.Package
	Score int
}

func GetRecommendations(input FilterInput) []ScoredPackage {
	var allPackages []models.Package
	repositories.DB.Preload("Travel").Find(&allPackages)

	var scored []ScoredPackage
	for _, pkg := range allPackages {
		score := 0

		if pkg.Price <= input.Budget {
			score += 40
		} else if pkg.Price <= input.Budget+5000000 {
			score += 20
		}

		switch input.Who {
		case "family":
			if pkg.IsFamily {
				score += 20
			}
			if pkg.IsKidFriendly {
				score += 10
			}
		case "senior":
			if pkg.IsSenior {
				score += 20
			}
			if pkg.IsSeniorFriendly {
				score += 10
			}
		case "couple":
			if pkg.GroupSize <= 30 {
				score += 10
			}
		case "alone":
			score += 0
		}

		switch input.Priority {
		case "near_haram":
			if pkg.IsNearHaram {
				score += 20
			}
			if pkg.HotelDistance <= 300 {
				score += 10
			}
		case "family_friendly":
			if pkg.IsKidFriendly || pkg.IsSeniorFriendly {
				score += 20
			}
			if pkg.IsFamily || pkg.IsSenior {
				score += 10
			}
		case "full_activity":
			if pkg.IsKajian || pkg.IsSunnah {
				score += 20
			}
			if pkg.SunnahScore >= 8 {
				score += 10
			}
		}

		if pkg.SunnahScore >= 8 {
			score += 10
		}
		if pkg.HotelDistance <= 500 {
			score += 5
		}
		if pkg.Travel.Rating >= 4.5 {
			score += 5
		}

		scored = append(scored, ScoredPackage{
			Package: pkg,
			Score:   score,
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
