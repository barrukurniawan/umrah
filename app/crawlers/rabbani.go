package crawlers

import (
	"fmt"
	"strings"

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
		lines := make([]string, 0)
		for _, l := range strings.Split(e.Text, "\n") {
			l = CleanText(l)
			if l != "" {
				lines = append(lines, l)
			}
		}

		var pkg *CrawledPackage
		for i, line := range lines {
			lower := strings.ToLower(line)

			if isDateLine(line) && len(line) < 20 && !strings.Contains(lower, "hari") {
				if pkg != nil && pkg.PackageName != "" {
					key := fmt.Sprintf("%s|%d|%s", pkg.PackageName, pkg.Price, line)
					if !seen[key] {
						seen[key] = true
						packages = append(packages, *pkg)
					}
				}
				pkg = &CrawledPackage{
					TravelName:     "Rabbani Tour",
					URL:            r.URL,
					DepartureDates: []string{line},
				}
				continue
			}

			if pkg == nil {
				continue
			}

			switch {
			case strings.HasPrefix(lower, "harga mulai"):
				if i+1 < len(lines) {
					pkg.Price = parseJutaStr(lines[i+1])
				}

			case strings.Contains(lower, "juta") && pkg.Price == 0:
				pkg.Price = parseJutaStr(line)

			case strings.Contains(lower, "hari") && pkg.Duration == 0:
				if d := ParseDuration(line); d > 0 && d < 100 {
					pkg.Duration = d
				}

			case strings.HasPrefix(lower, "hotel madinah"):
				pkg.HotelMadinah = strings.TrimPrefix(line, "Hotel Madinah")
				pkg.HotelMadinah = strings.TrimSpace(pkg.HotelMadinah)
				if pkg.HotelMadinah == "" && i+1 < len(lines) {
					pkg.HotelMadinah = lines[i+1]
				}

			case strings.HasPrefix(lower, "hotel mekkah"):
				pkg.HotelMakkah = strings.TrimPrefix(line, "Hotel Mekkah")
				pkg.HotelMakkah = strings.TrimSpace(pkg.HotelMakkah)
				if pkg.HotelMakkah == "" && i+1 < len(lines) {
					pkg.HotelMakkah = lines[i+1]
				}

			case pkg.PackageName == "" && len(line) > 3 &&
				!isDateLine(line) &&
				!strings.HasPrefix(lower, "harga") &&
				!strings.HasPrefix(lower, "hotel") &&
				!strings.HasPrefix(lower, "booking") &&
				!strings.HasPrefix(lower, "terbatas") &&
				!strings.HasPrefix(lower, "segera") &&
				!strings.Contains(lower, "juta") &&
				!strings.HasPrefix(lower, "all in") &&
				!strings.HasPrefix(lower, "transit") &&
				!strings.HasPrefix(lower, "thaif") &&
				!strings.HasPrefix(lower, "direct") &&
				!strings.HasPrefix(lower, "et /") &&
				!strings.HasPrefix(lower, "ey /") &&
				!strings.HasPrefix(lower, "wy /"):
				pkg.PackageName = line
			}
		}

		if pkg != nil && pkg.PackageName != "" {
			key := fmt.Sprintf("%s|%d", pkg.PackageName, pkg.Price)
			if !seen[key] {
				seen[key] = true
				packages = append(packages, *pkg)
			}
		}

		merged := make(map[string]*CrawledPackage)
		for i := range packages {
			p := &packages[i]
			if len(p.DepartureDates) == 0 {
				continue
			}
			dateKey := p.DepartureDates[0]
			if existing, ok := merged[dateKey]; ok {
				if p.Price > existing.Price { existing.Price = p.Price }
				if p.Duration > existing.Duration { existing.Duration = p.Duration }
				if p.HotelMakkah != "" && existing.HotelMakkah == "" { existing.HotelMakkah = p.HotelMakkah }
				if p.HotelMadinah != "" && existing.HotelMadinah == "" { existing.HotelMadinah = p.HotelMadinah }
				if p.Airline != "" && existing.Airline == "" { existing.Airline = p.Airline }
				if p.PackageName != "" && (existing.PackageName == "" || strings.Contains(strings.ToLower(existing.PackageName), "/setaraf")) {
					existing.PackageName = p.PackageName
				}
			} else {
				merged[dateKey] = p
			}
		}

		result := make([]CrawledPackage, 0)
		for _, p := range merged {
			if p.Price > 1000000 {
				result = append(result, *p)
			}
		}
		packages = result
	})

	c.Visit(r.URL)
	c.Wait()
	return packages, nil
}
