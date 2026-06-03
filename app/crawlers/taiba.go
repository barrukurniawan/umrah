package crawlers

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

type TaibaParser struct {
	URL string
}

func (t *TaibaParser) Crawl() ([]CrawledPackage, error) {
	var packages []CrawledPackage
	seen := make(map[string]bool)

	c := NewCollector("taibamedina.com")

	c.OnHTML("body", func(e *colly.HTMLElement) {
		var leaves []*goquery.Selection
		e.DOM.Find(":contains('Keberangkatan')").Each(func(_ int, s *goquery.Selection) {
			text := s.Text()
			if !strings.HasPrefix(CleanText(text), "Keberangkatan") {
				return
			}
			hasChild := false
			s.Children().Each(func(_ int, child *goquery.Selection) {
				if strings.HasPrefix(CleanText(child.Text()), "Keberangkatan") {
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
				TravelName: "Taiba Medina",
				URL:        t.URL,
			}

			fullText := strings.Join(lines, " ")

			for _, line := range lines {
				line = CleanText(line)
				if line == "" {
					continue
				}
				lower := strings.ToLower(line)

				switch {
				case strings.HasPrefix(lower, "keberangkatan:"):
					ds := strings.TrimPrefix(line, "Keberangkatan:")
					ds = strings.TrimPrefix(ds, "keberangkatan:")
					pkg.DepartureDates = append(pkg.DepartureDates, CleanText(ds))

				case strings.HasPrefix(lower, "mulai dari"):
					pkg.Price = ParsePrice(line)

				case strings.HasPrefix(lower, "makkah:"):
					pkg.HotelMakkah = CleanText(strings.TrimPrefix(line, "Makkah:"))
					pkg.HotelMakkah = CleanText(strings.TrimPrefix(pkg.HotelMakkah, "makkah:"))

				case strings.HasPrefix(lower, "madinah:"):
					pkg.HotelMadinah = CleanText(strings.TrimPrefix(line, "Madinah:"))
					pkg.HotelMadinah = CleanText(strings.TrimPrefix(pkg.HotelMadinah, "madinah:"))

				case strings.HasPrefix(lower, "maskapai:"):
					al := CleanText(strings.TrimPrefix(line, "Maskapai:"))
					al = CleanText(strings.TrimPrefix(al, "maskapai:"))
					if ka := KnownAirline(al); ka != "" {
						pkg.Airline = ka
					} else if al != "Request" {
						pkg.Airline = al
					}

				case strings.Contains(lower, "hari") && pkg.Duration == 0:
					if d := ParseDuration(line); d > 0 && d < 100 {
						pkg.Duration = d
					}

				case strings.Contains(lower, "sisa") && strings.Contains(lower, "kursi"):
					pkg.Seats = ParseSeats(line)
				}
			}

			if pkg.Price == 0 {
				if idx := strings.Index(strings.ToLower(fullText), "mulai dari"); idx != -1 {
					sub := fullText[idx+len("mulai dari"):]
					sub = CleanText(sub)
					pkg.Price = ParsePrice(sub)
				}
			}

			if pkg.PackageName == "" {
				for _, line := range lines {
					line = CleanText(line)
					lower := strings.ToLower(line)
					if len(line) > 5 &&
						!strings.HasPrefix(lower, "sisa") &&
						!strings.HasPrefix(lower, "mulai") &&
						!strings.HasPrefix(lower, "makkah") &&
						!strings.HasPrefix(lower, "madinah") &&
						!strings.HasPrefix(lower, "maskapai") &&
						!strings.HasPrefix(lower, "keberangkatan") &&
						!strings.HasPrefix(lower, "rp") &&
						!strings.HasPrefix(lower, "pesan") &&
						!strings.Contains(lower, "bintang") {
						if d := ParseDuration(line); d == 0 || d > 100 {
							pkg.PackageName = line
							break
						}
					}
				}
			}

			key := fmt.Sprintf("%s|%d", pkg.PackageName, pkg.Price)
			if pkg.PackageName != "" && !seen[key] {
				seen[key] = true
				packages = append(packages, pkg)
			}
		}
	})

	err := c.Visit(t.URL)
	if err != nil {
		return packages, err
	}
	c.Wait()
	return packages, nil
}
