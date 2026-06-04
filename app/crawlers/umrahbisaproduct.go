package crawlers

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type UmrahBisaProductParser struct {
	URL string
}

func (u *UmrahBisaProductParser) Crawl() ([]CrawledPackage, error) {
	req, _ := http.NewRequest("GET", u.URL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	html = regexp.MustCompile(`<script[^>]*>[\s\S]*?</script>`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`<style[^>]*>[\s\S]*?</style>`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`<noscript[^>]*>[\s\S]*?</noscript>`).ReplaceAllString(html, "")

	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, "\n")
	text = regexp.MustCompile(`\n+`).ReplaceAllString(text, "\n")
	text = regexp.MustCompile(`\{[^}]*\}`).ReplaceAllString(text, "")
	text = strings.ReplaceAll(text, "\\", "")
	text = strings.ReplaceAll(text, `"`, "")
	text = strings.ReplaceAll(text, "`", "")

	lower := strings.ToLower(text)

	pkg := CrawledPackage{
		TravelName:     "Umrah Bisa",
		URL:            u.URL,
		DepartureDates: []string{},
	}

	if idx := strings.Index(lower, "jadwal:"); idx >= 0 {
		sub := strings.TrimSpace(lower[idx+7:])
		end := strings.Index(sub, "\n")
		if end < 0 {
			end = strings.Index(sub, "pesawat:")
		}
		if end < 0 {
			end = 40
		}
		raw := strings.TrimSpace(sub[:end])
		raw = CleanText(strings.SplitN(raw, "pesawat", 2)[0])
		raw = CleanText(strings.SplitN(raw, "hotel", 2)[0])
		if strings.Contains(raw, "-") {
			parts := strings.Fields(raw)
			if len(parts) >= 4 {
				var sd, ed int
				fmt.Sscanf(parts[0], "%d", &sd)
				fmt.Sscanf(parts[2], "%d", &ed)
				if ed > sd {
					pkg.Duration = ed - sd + 1
				}
			}
			if dash := strings.Index(raw, "-"); dash > 0 {
				raw = strings.TrimSpace(raw[:dash]) + " " + parts[len(parts)-2] + " " + parts[len(parts)-1]
			}
		}
		if raw != "" {
			pkg.DepartureDates = append(pkg.DepartureDates, raw)
		}
	}

	if idx := strings.Index(lower, "pesawat:"); idx >= 0 {
		sub := strings.TrimSpace(lower[idx+8:])
		end := strings.Index(sub, "hotel:")
		if end < 0 {
			end = 80
		}
		al := CleanText(strings.TrimRight(sub[:end], ",.! "))
		if ka := KnownAirline(al); ka != "" {
			pkg.Airline = ka
		}
	}

	if pkg.Airline == "" {
		for _, line := range strings.Split(text, "\n") {
			line = CleanText(line)
			if ka := KnownAirline(line); ka != "" {
				pkg.Airline = ka
				break
			}
		}
	}

	if idx := strings.Index(lower, "hotel:"); idx >= 0 {
		sub := lower[idx+6:]
		end := strings.Index(sub, "harga spesial:")
		if end < 0 {
			end = 80
		}
		pkg.HotelMakkah = CleanText(strings.TrimRight(sub[:end], ",.! "))
		pkg.HotelMadinah = pkg.HotelMakkah
	}

	if idx := strings.Index(lower, "harga spesial:"); idx >= 0 {
		sub := lower[idx+14:]
		end := strings.Index(sub, "plus:")
		if end < 0 {
			end = strings.Index(sub, "kuota:")
		}
		if end < 0 {
			end = 40
		}
		pkg.Price = parseJutaStr(CleanText(sub[:end]))
	}

	for _, line := range strings.Split(text, "\n") {
		line = CleanText(line)
		lowerLine := strings.ToLower(line)
		if pkg.PackageName == "" && len(line) > 15 &&
			(strings.HasPrefix(lowerLine, "dp umrah") || strings.HasPrefix(lowerLine, "umrah hemat")) {
			if idx := strings.Index(line, ","); idx > 0 {
				line = CleanText(line[:idx])
			}
			if idx := strings.Index(line, "sku"); idx > 0 {
				line = CleanText(line[:idx])
			}
			pkg.PackageName = line
		}
	}

	if pkg.Duration == 0 && pkg.PackageName != "" {
		pkg.Duration = ParseDuration(pkg.PackageName)
	}
	if pkg.Duration == 0 && len(pkg.DepartureDates) > 0 {
		dateStr := pkg.DepartureDates[0]
		parts := strings.Fields(dateStr)
		if len(parts) >= 3 {
			startDay := parts[0]
			var endDay string
			if strings.Contains(dateStr, "-") {
				dashParts := strings.Split(dateStr, "-")
				if len(dashParts) >= 2 {
					endDay = strings.TrimSpace(dashParts[1])
				}
			}
			if endDay != "" {
				var sd, ed int
				fmt.Sscanf(startDay, "%d", &sd)
				fmt.Sscanf(endDay, "%d", &ed)
				if ed > sd {
					pkg.Duration = ed - sd + 1
				}
			}
		}
	}

	if pkg.PackageName == "" {
		pkg.PackageName = "Umrah Hemat"
	}

	var packages []CrawledPackage
	if pkg.Price > 0 {
		packages = append(packages, pkg)
	}
	return packages, nil
}
