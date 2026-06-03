package crawlers

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

type HamdanParser struct {
	URL string
}

func (h *HamdanParser) Crawl() ([]CrawledPackage, error) {
	var packages []CrawledPackage
	seen := make(map[string]bool)

	c := NewCollector("hamdantour.id")

	c.OnHTML("body", func(e *colly.HTMLElement) {
		var leaves []*goquery.Selection
		e.DOM.Find(":contains('Sisa Seat')").Each(func(_ int, s *goquery.Selection) {
			hasChild := false
			s.Children().Each(func(_ int, child *goquery.Selection) {
				if strings.Contains(child.Text(), "Sisa Seat") {
					hasChild = true
				}
			})
			if !hasChild {
				leaves = append(leaves, s)
			}
		})

		for _, leaf := range leaves {
			card := leaf.Closest("[class*='col-md'], [class*='col-lg'], [class*='col-sm'], [class*='card'], div.card")
			if card.Length() == 0 {
				card = leaf.ParentsFiltered("div").First()
			}

			text := card.Text()
			lines := strings.Split(text, "\n")

			pkg := CrawledPackage{
				TravelName: "Hamdan Tour",
				URL:        h.URL,
			}

			for _, line := range lines {
				line = CleanText(line)
				if line == "" {
					continue
				}
				lower := strings.ToLower(line)

				switch {
				case strings.HasPrefix(lower, "umrah") || strings.Contains(lower, "✈"):
					if pkg.PackageName == "" {
						pkg.PackageName = line
					}

				case strings.HasPrefix(lower, "sisa seat"):
					pkg.Seats = ParseSeats(line)

				case strings.HasPrefix(lower, "harga mulai"):
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						pkg.Price = ParsePrice(parts[1])
					}

				case strings.Contains(lower, "makkah") && pkg.HotelMakkah == "":
					pkg.HotelMakkah = line

				case strings.Contains(lower, "madinah") && pkg.HotelMadinah == "":
					pkg.HotelMadinah = line

				case strings.Contains(lower, "hari") && pkg.Duration == 0:
					if d := ParseDuration(line); d > 0 && d < 100 {
						pkg.Duration = d
					}

				case strings.Contains(lower, "soekarno") || strings.Contains(lower, "cgk"):
					pkg.Airport = "CGK"

				case pkg.Airline == "" && KnownAirline(line) != "":
					pkg.Airline = KnownAirline(line)
				}
			}

			if pkg.Price == 0 {
				fullText := strings.Join(lines, " ")
				if idx := strings.Index(strings.ToLower(fullText), "harga mulai"); idx != -1 {
					sub := fullText[idx:]
					parts := strings.SplitN(sub, ":", 2)
					if len(parts) == 2 {
						end := strings.Index(parts[1], "DETAIL")
						priceText := parts[1]
						if end > 0 {
							priceText = parts[1][:end]
						}
						pkg.Price = ParsePrice(priceText)
					}
				}
			}

			if pkg.PackageName == "" {
				for _, line := range lines {
					line = CleanText(line)
					lower := strings.ToLower(line)
					if len(line) > 15 &&
						!strings.HasPrefix(lower, "sisa") &&
						!strings.HasPrefix(lower, "harga") &&
						!strings.HasPrefix(lower, "detail") &&
						!strings.HasPrefix(lower, "promo") &&
						!strings.Contains(lower, "makkah") &&
						!strings.Contains(lower, "madinah") &&
						!strings.Contains(lower, "soekarno") &&
						!strings.Contains(lower, "cgk") &&
						!strings.HasPrefix(lower, "idr") &&
						KnownAirline(line) == "" &&
						!isDateLine(line) {
						if d := ParseDuration(line); d == 0 || d > 100 {
							pkg.PackageName = line
							break
						}
					}
				}
			}

			for _, line := range lines {
				line = CleanText(line)
				lower := strings.ToLower(line)
				if len(line) < 10 {
					continue
				}
				if strings.HasPrefix(lower, "harga") || strings.HasPrefix(lower, "sisa") ||
					strings.HasPrefix(lower, "detail") || strings.HasPrefix(lower, "idr") {
					continue
				}
				if isDateLine(line) &&
					!strings.Contains(lower, "makkah") && !strings.Contains(lower, "madinah") {
					pkg.DepartureDates = append(pkg.DepartureDates, line)
				}
			}

			key := fmt.Sprintf("%s|%d", pkg.PackageName, pkg.Price)
			if pkg.Duration == 0 && pkg.PackageName != "" {
				pkg.Duration = ParseDuration(pkg.PackageName)
			}

			if pkg.PackageName != "" && !seen[key] {
				seen[key] = true
				packages = append(packages, pkg)
			}
		}
	})

	err := c.Visit(h.URL)
	if err != nil {
		return packages, err
	}
	c.Wait()
	return packages, nil
}
