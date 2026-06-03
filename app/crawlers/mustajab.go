package crawlers

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

type MustajabParser struct {
	URL        string
	TravelName string
}

func (m *MustajabParser) Crawl() ([]CrawledPackage, error) {
	var packages []CrawledPackage
	seen := make(map[string]bool)

	c := NewCollector("umrohmustajab.com")

	c.OnHTML("body", func(e *colly.HTMLElement) {
		var leaves []*goquery.Selection
		e.DOM.Find(":contains('Available Pax'), :contains('FULLY BOOKED')").Each(func(_ int, s *goquery.Selection) {
			text := strings.ToLower(s.Text())
			if !strings.Contains(text, "available pax") && !strings.Contains(text, "fully booked") {
				return
			}
			hasChild := false
			s.Children().Each(func(_ int, child *goquery.Selection) {
				ct := strings.ToLower(child.Text())
				if strings.Contains(ct, "available pax") || strings.Contains(ct, "fully booked") {
					hasChild = true
				}
			})
			if !hasChild {
				leaves = append(leaves, s)
			}
		})

		for _, leaf := range leaves {
			card := leaf.Closest("[class*='col-md'], [class*='col-lg'], [class*='col-sm'], [class*='card']")
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
				TravelName: m.TravelName,
				URL:        m.URL,
			}

			fullText := strings.Join(lines, " ")

			for i, line := range lines {
				lower := strings.ToLower(line)

				switch {
				case strings.HasPrefix(lower, "available pax"):
					seatsNum := CleanText(strings.TrimPrefix(line, "Available Pax"))
					pkg.Seats = ParseSeats(seatsNum)
				case strings.HasPrefix(lower, "fully booked"):
					pkg.Seats = 0

				case strings.HasPrefix(lower, "harga mulai"):
					priceText := CleanText(strings.TrimPrefix(line, "Harga Mulai"))
					if priceText == "" && i+1 < len(lines) {
						priceText = lines[i+1]
					}
					if priceText == "" && i+2 < len(lines) {
						priceText = lines[i+2]
					}
					pkg.Price = ParsePrice(priceText)

				case strings.HasPrefix(lower, "berangkat dari"):
					pkg.Airport = CleanText(strings.TrimPrefix(line, "Berangkat dari"))

				case strings.HasPrefix(lower, "maskapai") && pkg.Airline == "":
					al := CleanText(strings.TrimPrefix(line, "Maskapai"))
					if ka := KnownAirline(al); ka != "" {
						pkg.Airline = ka
					} else {
						pkg.Airline = al
					}

				case strings.Contains(lower, "hari") && pkg.Duration == 0:
					if d := ParseDuration(line); d > 0 && d < 100 {
						pkg.Duration = d
					}

				case pkg.PackageName == "" && len(line) > 10 &&
					!strings.HasPrefix(lower, "available") && !strings.HasPrefix(lower, "fully") &&
					!strings.HasPrefix(lower, "total") && !strings.HasPrefix(lower, "berangkat") &&
					!strings.HasPrefix(lower, "maskapai") && !strings.HasPrefix(lower, "kelas") &&
					!strings.HasPrefix(lower, "harga") && !strings.HasPrefix(lower, "lihat"):
					if !isDateLine(line) || (i > 0 && isDateLine(lines[i-1])) {
						pkg.PackageName = line
					}
				}
			}

			if pkg.Price == 0 {
				if idx := strings.Index(strings.ToLower(fullText), "harga mulai"); idx >= 0 {
					sub := fullText[idx+len("harga mulai"):]
					pkg.Price = ParsePrice(strings.TrimSpace(sub))
				}
			}

			if pkg.DepartureDates == nil {
				pkg.DepartureDates = []string{}
			}
			for _, line := range lines {
				lower := strings.ToLower(line)
				if isDateLine(line) && !strings.HasPrefix(lower, "umroh") && len(line) < 25 {
					pkg.DepartureDates = append(pkg.DepartureDates, line)
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

	c.Visit(m.URL)
	c.Wait()
	return packages, nil
}
