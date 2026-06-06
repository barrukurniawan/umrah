package crawlers

import (
	"io"
	"net/http"
	"regexp"
	"strings"
)

// UmiTravelParser scrapes a single UMI Travel listing page.
// Price is extracted from the WhatsApp share URL embedded in the HTML.
// Hotel, airline, seats and departure date are extracted from plain-text sections.
type UmiTravelParser struct {
	URL string
}

// englishMonthToID converts English month names to Indonesian for consistency
func englishMonthToID(s string) string {
	r := strings.NewReplacer(
		"January", "Januari", "January", "Januari",
		"February", "Februari",
		"March", "Maret",
		"April", "April",
		"May", "Mei",
		"June", "Juni",
		"July", "Juli",
		"August", "Agustus",
		"September", "September",
		"October", "Oktober",
		"November", "November",
		"December", "Desember",
	)
	return r.Replace(s)
}

func (u *UmiTravelParser) Crawl() ([]CrawledPackage, error) {
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

	pkg := CrawledPackage{
		TravelName:     "UMI Tour & Travel",
		URL:            u.URL,
		DepartureDates: []string{},
		Airport:        "CGK",
	}

	// --- Price: from WhatsApp share URL (most reliable) ---
	// Pattern: "harga+mulai+dari+Rp36.900.000,-"
	waRe := regexp.MustCompile(`(?i)harga[+%20 ]+mulai[+%20 ]+dari[+%20 ,]*Rp(\d[\d.,]*)`)
	if m := waRe.FindStringSubmatch(html); len(m) > 1 {
		raw := strings.ReplaceAll(m[1], "%2C", ",")
		raw = strings.ReplaceAll(raw, "%2E", ".")
		pkg.Price = ParsePrice(raw)
	}
	// Fallback: meta / og
	if pkg.Price == 0 {
		ogPriceRe := regexp.MustCompile(`(?i)(?:mulai dari|harga)\D*(Rp[\d.,]+)`)
		if m := ogPriceRe.FindStringSubmatch(html); len(m) > 1 {
			pkg.Price = ParsePrice(m[1])
		}
	}

	// --- Package name: og:title or <title> ---
	ogTitleRe := regexp.MustCompile(`<meta[^>]+property="og:title"[^>]+content="([^"]+)"`)
	if m := ogTitleRe.FindStringSubmatch(html); len(m) > 1 {
		pkg.PackageName = CleanText(m[1])
	}
	if pkg.PackageName == "" {
		titleRe := regexp.MustCompile(`<title>([^<]+)</title>`)
		if m := titleRe.FindStringSubmatch(html); len(m) > 1 {
			pkg.PackageName = CleanText(m[1])
		}
	}

	// --- Strip script/style, flatten to plain lines ---
	scriptRe := regexp.MustCompile(`(?s)<script[^>]*>.*?</script>`)
	styleRe := regexp.MustCompile(`(?s)<style[^>]*>.*?</style>`)
	text := scriptRe.ReplaceAllString(html, "")
	text = styleRe.ReplaceAllString(text, "")
	tagRe := regexp.MustCompile(`<[^>]+>`)
	text = tagRe.ReplaceAllString(text, "\n")
	text = regexp.MustCompile(`[ \t]+`).ReplaceAllString(text, " ")
	text = regexp.MustCompile(`\n+`).ReplaceAllString(text, "\n")

	lines := strings.Split(text, "\n")

	// --- Seats: line "Tersisa Pax" followed by number ---
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), "tersisa") && strings.Contains(strings.ToLower(line), "pax") {
			// Number might be on the same line or next non-empty line
			numRe := regexp.MustCompile(`\d+`)
			if nums := numRe.FindString(line); nums != "" {
				var s int
				for _, c := range nums {
					s = s*10 + int(c-'0')
				}
				pkg.Seats = s
			} else {
				for j := i + 1; j < len(lines) && j < i+4; j++ {
					next := CleanText(lines[j])
					if numRe.MatchString(next) {
						var s int
						for _, c := range numRe.FindString(next) {
							s = s*10 + int(c-'0')
						}
						pkg.Seats = s
						break
					}
				}
			}
			break
		}
	}

	// --- Departure date ---
	// Strategy 1: WA URL "Keberangkatan: JAKARTA, DD Month YYYY"
	deptRe := regexp.MustCompile(`(?i)Keberangkatan[%20+: ,]+[A-Z ,]+(\d{1,2}[+%20 ]+\w+[+%20 ]+20\d{2})`)
	if m := deptRe.FindStringSubmatch(html); len(m) > 1 {
		dateStr := strings.ReplaceAll(m[1], "+", " ")
		dateStr = strings.ReplaceAll(dateStr, "%20", " ")
		dateStr = strings.TrimSpace(dateStr)
		dateStr = englishMonthToID(dateStr)
		if dateStr != "" {
			pkg.DepartureDates = append(pkg.DepartureDates, dateStr)
		}
	}

	// Strategy 2: plain text — English months "13 August 2026" or Indonesian "13 Agustus 2026"
	if len(pkg.DepartureDates) == 0 {
		dateRe := regexp.MustCompile(`(?i)(\d{1,2}\s+(?:january|february|march|april|may|june|july|august|september|october|november|december|januari|februari|maret|mei|juni|juli|agustus|oktober|desember)\s+20\d{2})`)
		// Scan near "JAKARTA" city marker first, then full doc
		jakartaIdx := -1
		for i, line := range lines {
			if strings.TrimSpace(strings.ToUpper(line)) == "JAKARTA" {
				jakartaIdx = i
				break
			}
		}
		searchFrom := 0
		if jakartaIdx >= 0 {
			searchFrom = jakartaIdx
		}
		for i := searchFrom; i < len(lines); i++ {
			line := CleanText(lines[i])
			if dm := dateRe.FindStringSubmatch(line); len(dm) > 1 {
				pkg.DepartureDates = append(pkg.DepartureDates, englishMonthToID(dm[1]))
				break
			}
		}
	}

	// --- Airline ---
	// "Departure : Saudia Airlines" pattern
	deptAirlineRe := regexp.MustCompile(`(?i)departure\s*:\s*([^\n]+)`)
	if m := deptAirlineRe.FindStringSubmatch(text); len(m) > 1 {
		if ka := KnownAirline(m[1]); ka != "" {
			pkg.Airline = ka
		}
	}
	if pkg.Airline == "" {
		for _, line := range lines {
			if ka := KnownAirline(line); ka != "" {
				pkg.Airline = ka
				break
			}
		}
	}

	// --- Hotels ---
	// Pattern A (most pages): "HOTEL_NAME\nHotel Rate\nLokasi :\nMakkah/Madinah"
	// Pattern B (some pages): "#### HOTEL_NAME\n...\n##### Lokasi :\nMakkah/Madinah"
	for i, line := range lines {
		line = CleanText(line)
		lower := strings.ToLower(line)

		// Strip markdown heading markers
		stripped := strings.TrimLeft(lower, "#")
		stripped = strings.TrimSpace(stripped)

		if strings.Contains(stripped, "lokasi") {
			// Find city in next few lines
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				city := strings.TrimSpace(strings.ToLower(lines[j]))
				city = strings.Trim(city, " :-")
				// Strip markdown
				city = strings.TrimLeft(city, "#")
				city = strings.TrimSpace(city)

				if city == "makkah" && pkg.HotelMakkah == "" {
					// Hotel name: look backwards for a non-empty, non-keyword line
					for k := i - 1; k >= 0 && k >= i-6; k-- {
						candidate := CleanText(lines[k])
						cl := strings.ToLower(candidate)
						cl = strings.TrimLeft(cl, "#")
						cl = strings.TrimSpace(cl)
						if candidate == "" || cl == "hotel" || cl == "hotel rate" ||
							cl == "lokasi" || strings.HasPrefix(cl, "info detail") ||
							strings.Contains(cl, "check in") || strings.Contains(cl, "check out") {
							continue
						}
						// Remove markdown prefix from hotel name
						candidate = strings.TrimLeft(candidate, "# ")
						candidate = CleanText(candidate)
						if len(candidate) > 3 {
							pkg.HotelMakkah = candidate
						}
						break
					}
					break
				}
				if city == "madinah" && pkg.HotelMadinah == "" {
					for k := i - 1; k >= 0 && k >= i-6; k-- {
						candidate := CleanText(lines[k])
						cl := strings.ToLower(candidate)
						cl = strings.TrimLeft(cl, "#")
						cl = strings.TrimSpace(cl)
						if candidate == "" || cl == "hotel" || cl == "hotel rate" ||
							cl == "lokasi" || strings.HasPrefix(cl, "info detail") ||
							strings.Contains(cl, "check in") || strings.Contains(cl, "check out") {
							continue
						}
						candidate = strings.TrimLeft(candidate, "# ")
						candidate = CleanText(candidate)
						if len(candidate) > 3 {
							pkg.HotelMadinah = candidate
						}
						break
					}
					break
				}
				if city != "" {
					break
				}
			}
		}
	}

	// --- Duration ---
	// "Durasi\n12 Hari" in plain text or WA URL
	durasiRe := regexp.MustCompile(`(?i)durasi[%20+: ]+(\d+)[%20+ ]+hari`)
	if m := durasiRe.FindStringSubmatch(html); len(m) > 1 {
		pkg.Duration = ParseDuration(m[1] + " hari")
	}
	if pkg.Duration == 0 {
		for i, line := range lines {
			if strings.ToLower(strings.TrimSpace(line)) == "durasi" {
				for j := i + 1; j < len(lines) && j < i+4; j++ {
					next := CleanText(lines[j])
					if d := ParseDuration(next); d > 0 && d < 60 {
						pkg.Duration = d
						break
					}
				}
				break
			}
		}
	}
	if pkg.Duration == 0 && pkg.PackageName != "" {
		pkg.Duration = ParseDuration(pkg.PackageName)
	}

	var packages []CrawledPackage
	if pkg.Price > 0 && pkg.PackageName != "" {
		packages = append(packages, pkg)
	}
	return packages, nil
}
