package crawlers

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

type RabbaniParser struct {
	URL string
}

func (r *RabbaniParser) Crawl() ([]CrawledPackage, error) {
	var packages []CrawledPackage
	seen := make(map[string]bool)

	c := NewCollector("rabbanitour.com")

	c.OnHTML("body", func(e *colly.HTMLElement) {
		var leaves []*goquery.Selection
		e.DOM.Find(":contains('Harga Mulai'), :contains('Juta')").Each(func(_ int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if !strings.HasPrefix(text, "Harga Mulai") {
				return
			}
			hasChild := false
			s.Children().Each(func(_ int, child *goquery.Selection) {
				if strings.HasPrefix(strings.TrimSpace(child.Text()), "Harga Mulai") {
					hasChild = true
				}
			})
			if !hasChild {
				leaves = append(leaves, s)
			}
		})

		for _, leaf := range leaves {
			card := leaf.Closest("[class*='col-md'], [class*='col-lg'], [class*='col-sm'], [class*='card'], [class*='jadwal']")
			if card.Length() == 0 {
				card = leaf.ParentsFiltered("div").First()
			}

			text := card.Text()
			lines := make([]string, 0)
			for _, line := range strings.Split(text, "\n") {
				line = CleanText(line)
				if line != "" {
					lines = append(lines, line)
				}
			}

			pkg := CrawledPackage{
				TravelName:     "Rabbani Tour",
				URL:            r.URL,
				DepartureDates: []string{},
			}

			var hotelMode string
			for i, line := range lines {
				lower := strings.ToLower(line)

				switch {
				case strings.HasPrefix(lower, "hotel madinah") && pkg.HotelMadinah == "":
					pkg.HotelMadinah = CleanText(strings.TrimPrefix(line, "Hotel Madinah"))
				case strings.HasPrefix(lower, "hotel mekkah") && pkg.HotelMakkah == "":
					pkg.HotelMakkah = CleanText(strings.TrimPrefix(line, "Hotel Mekkah"))

				case strings.HasPrefix(lower, "harga mulai"):
					if i+1 < len(lines) {
						pkg.Price = parseJutaStr(lines[i+1])
					} else {
						pkg.Price = parseJutaStr(CleanText(strings.TrimPrefix(line, "Harga Mulai")))
					}

				case strings.Contains(lower, "hari") && pkg.Duration == 0:
					if d := ParseDuration(line); d > 0 && d < 100 {
						pkg.Duration = d
					}

				case strings.Contains(lower, "juta") && pkg.Price == 0:
					pkg.Price = parseJutaStr(line)

				case strings.HasPrefix(lower, "hotel mekkah"):
					hotelMode = "mekkah"
				case strings.HasPrefix(lower, "hotel madinah"):
					hotelMode = "madinah"

				case isDateLine(line) && len(line) < 20:
					pkg.DepartureDates = append(pkg.DepartureDates, line)

				default:
					if (hotelMode == "mekkah" && pkg.HotelMakkah == "" && len(line) > 3 &&
						!strings.HasPrefix(lower, "booking") && !strings.HasPrefix(lower, "terbatas") &&
						!strings.HasPrefix(lower, "segera") && !isDateLine(line)) {
						pkg.HotelMakkah = line
					}
					if (hotelMode == "madinah" && pkg.HotelMadinah == "" && len(line) > 3 &&
						!strings.HasPrefix(lower, "booking") && !strings.HasPrefix(lower, "terbatas") &&
						!strings.HasPrefix(lower, "segera") && !isDateLine(line)) {
						pkg.HotelMadinah = line
					}

					if pkg.PackageName == "" && len(line) > 5 &&
						!strings.HasPrefix(lower, "harga") && !strings.HasPrefix(lower, "hotel") &&
						!strings.HasPrefix(lower, "booking") && !strings.HasPrefix(lower, "terbatas") &&
						!strings.HasPrefix(lower, "segera") && !isDateLine(line) &&
						!strings.Contains(lower, "juta") && !strings.HasPrefix(lower, "all in") &&
						!strings.HasPrefix(lower, "transit") && !strings.HasPrefix(lower, "thaif") &&
						!strings.HasPrefix(lower, "direct") && !strings.HasPrefix(lower, "reguler") {
						pkg.PackageName = line
					}
				}
			}

			if pkg.Price == 0 {
				fullText := strings.Join(lines, " ")
				idx := strings.Index(fullText, "Harga Mulai")
				if idx >= 0 {
					pkg.Price = parseJutaStr(strings.TrimSpace(fullText[idx+len("Harga Mulai"):]))
				}
			}

			if pkg.PackageName != "" {
				key := fmt.Sprintf("%s|%d", pkg.PackageName, pkg.Price)
				if !seen[key] {
					seen[key] = true
					packages = append(packages, pkg)
				}
			}
		}
	})

	c.Visit(r.URL)
	c.Wait()
	return packages, nil
}
