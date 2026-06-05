package crawlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
)

type RabbaniParser struct {
	URL string
}

func (r *RabbaniParser) Crawl() ([]CrawledPackage, error) {
	var allPackages []CrawledPackage
	seen := make(map[string]bool)

	c := NewCollector("rabbanitour.com")

	maxPage := 1

	c.OnHTML(".e-load-more-anchor", func(e *colly.HTMLElement) {
		mp := e.Attr("data-max-page")
		if mp != "" {
			if n, err := strconv.Atoi(mp); err == nil && n > maxPage {
				maxPage = n
			}
		}
	})

	c.OnHTML("body", func(e *colly.HTMLElement) {
		pkgs := parseRabbaniBody(e.Text, r.URL, seen)
		allPackages = append(allPackages, pkgs...)
	})

	c.Visit(r.URL)
	c.Wait()

	// Scrape additional pages if any
	if maxPage > 1 {
		c2 := NewCollector("rabbanitour.com")
		c2.OnHTML("body", func(e *colly.HTMLElement) {
			pkgs := parseRabbaniBody(e.Text, r.URL, seen)
			allPackages = append(allPackages, pkgs...)
		})

		for page := 2; page <= maxPage; page++ {
			nextURL := fmt.Sprintf("%spage/%d/", r.URL, page)
			if !strings.HasSuffix(r.URL, "/") {
				nextURL = fmt.Sprintf("%s/page/%d/", r.URL, page)
			}
			c2.Visit(nextURL)
		}
		c2.Wait()
	}

	merged := make(map[string]*CrawledPackage)
	for i := range allPackages {
		p := &allPackages[i]
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

	return result, nil
}

func parseRabbaniBody(bodyText, url string, seen map[string]bool) []CrawledPackage {
	var packages []CrawledPackage
	lines := make([]string, 0)
	for _, l := range strings.Split(bodyText, "\n") {
		l = CleanText(l)
		if l != "" {
			lines = append(lines, l)
		}
	}

	var pkg *CrawledPackage
	defaultAirline := ""
	for i, line := range lines {
		lower := strings.ToLower(line)

		if pkg == nil && KnownAirline(line) != "" {
			defaultAirline = KnownAirline(line)
			continue
		}

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
				URL:            url,
				DepartureDates: []string{line},
				Airline:        defaultAirline,
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

		case KnownAirline(line) != "" && pkg.Airline == "":
			pkg.Airline = KnownAirline(line)

		case pkg.PackageName == "" && len(line) > 3 &&
			!isDateLine(line) &&
			!strings.HasPrefix(lower, "harga") &&
			!strings.HasPrefix(lower, "hotel") &&
			!strings.HasPrefix(lower, "booking") &&
			!strings.HasPrefix(lower, "terbatas") &&
			!strings.HasPrefix(lower, "segera") &&
			!strings.HasPrefix(lower, "push") &&
			!strings.HasPrefix(lower, "promo") &&
			!strings.HasPrefix(lower, "diskon") &&
			!strings.HasPrefix(lower, "dapatkan") &&
			!strings.Contains(lower, "juta") &&
			!strings.Contains(lower, "/setaraf") &&
			!strings.HasPrefix(lower, "all in") &&
			!strings.HasPrefix(lower, "transit") &&
			!strings.HasPrefix(lower, "thaif") &&
			!strings.HasPrefix(lower, "direct") &&
			!strings.HasPrefix(lower, "kereta") &&
			!strings.HasPrefix(lower, "al ula"):
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

	return packages
}
