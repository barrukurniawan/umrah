package crawlers

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// UhudTourParser scrapes a single Uhud Tour package detail page.
// Price is extracted from JSON-LD <script type="application/ld+json"> in <head>.
// Hotel, airline, and departure date are extracted from the rendered HTML text.
type UhudTourParser struct {
	URL string
}

func (u *UhudTourParser) Crawl() ([]CrawledPackage, error) {
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
		TravelName:     "Uhud Tour",
		URL:            u.URL,
		DepartureDates: []string{},
	}

	// --- Extract price from JSON-LD ---
	jsonLDRe := regexp.MustCompile(`(?s)<script[^>]+application/ld\+json[^>]*>(.*?)</script>`)
	if m := jsonLDRe.FindStringSubmatch(html); len(m) > 1 {
		var schema struct {
			Offers struct {
				Price string `json:"price"`
			} `json:"offers"`
		}
		if err := json.Unmarshal([]byte(m[1]), &schema); err == nil && schema.Offers.Price != "" {
			pkg.Price = ParsePrice(schema.Offers.Price)
		}
	}

	// --- Extract package name from <h1 id="page-title"> ---
	h1Re := regexp.MustCompile(`<h1[^>]*id="page-title"[^>]*>(.*?)</h1>`)
	if m := h1Re.FindStringSubmatch(html); len(m) > 1 {
		pkg.PackageName = CleanText(stripTags(m[1]))
	}

	// --- Extract departure date: parse from page title pattern "DD BULAN YYYY" ---
	if pkg.PackageName != "" && isDateLine(pkg.PackageName) {
		// e.g. "30 JULI 2026 | BRONZE" → extract date part
		dateRe := regexp.MustCompile(`(?i)(\d{1,2}\s+(?:januari|februari|maret|april|mei|juni|juli|agustus|september|oktober|november|desember)\s+20\d{2})`)
		if dm := dateRe.FindStringSubmatch(pkg.PackageName); len(dm) > 1 {
			pkg.DepartureDates = append(pkg.DepartureDates, strings.Title(strings.ToLower(dm[1])))
		}
	}

	// --- Strip all tags for plain text parsing of hotel, airline, duration ---
	scriptRe := regexp.MustCompile(`(?s)<script[^>]*>.*?</script>`)
	styleRe := regexp.MustCompile(`(?s)<style[^>]*>.*?</style>`)
	cleaned := scriptRe.ReplaceAllString(html, "")
	cleaned = styleRe.ReplaceAllString(cleaned, "")

	tagRe := regexp.MustCompile(`<[^>]+>`)
	text := tagRe.ReplaceAllString(cleaned, "\n")
	text = regexp.MustCompile(`\n+`).ReplaceAllString(text, "\n")

	lines := strings.Split(text, "\n")
	inHotelMakkah := false
	inHotelMadinah := false
	inMaskapai := false

	for _, line := range lines {
		line = CleanText(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)

		switch {
		case lower == "hotel makkah":
			inHotelMakkah = true
			inHotelMadinah = false
			inMaskapai = false
		case lower == "hotel madinah":
			inHotelMadinah = true
			inHotelMakkah = false
			inMaskapai = false
		case lower == "maskapai":
			inMaskapai = true
			inHotelMakkah = false
			inHotelMadinah = false
		case inHotelMakkah && pkg.HotelMakkah == "" && len(line) > 3 && !strings.HasPrefix(lower, "hotel"):
			pkg.HotelMakkah = line
			inHotelMakkah = false
		case inHotelMadinah && pkg.HotelMadinah == "" && len(line) > 3 && !strings.HasPrefix(lower, "hotel"):
			pkg.HotelMadinah = line
			inHotelMadinah = false
		case inMaskapai && pkg.Airline == "" && KnownAirline(line) != "":
			pkg.Airline = KnownAirline(line)
			inMaskapai = false
		}

		// Fallback airline detection
		if pkg.Airline == "" && KnownAirline(line) != "" {
			pkg.Airline = KnownAirline(line)
		}

		// Duration from package name
		if pkg.Duration == 0 && strings.Contains(lower, "hari") {
			if d := ParseDuration(line); d > 0 && d < 60 {
				pkg.Duration = d
			}
		}
	}

	if pkg.Duration == 0 && pkg.PackageName != "" {
		pkg.Duration = ParseDuration(pkg.PackageName)
	}

	// Default airport
	pkg.Airport = "CGK"

	var packages []CrawledPackage
	if pkg.Price > 0 && pkg.PackageName != "" {
		packages = append(packages, pkg)
	}
	return packages, nil
}

func stripTags(s string) string {
	return regexp.MustCompile(`<[^>]+>`).ReplaceAllString(s, "")
}
