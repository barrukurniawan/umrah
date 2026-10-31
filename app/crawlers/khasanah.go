package crawlers

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

// KhasanahParser scrapes Khasanah Travel (Next.js storefront, server-rendered).
// The /trip listing contains every trip card (name, date range, lowest price,
// seats); each trip detail page is visited for the Makkah/Madinah hotels and
// the departure airport. URL may be the homepage or the /trip listing.
type KhasanahParser struct {
	URL string
}

var (
	khasanahDateRe  = regexp.MustCompile(`(\d{1,2})\s+([A-Za-z]+)\s+(\d{4})`)
	khasanahSeatsRe = regexp.MustCompile(`(?i)sisa\s*(\d+)\s*seat`)
)

func (k *KhasanahParser) Crawl() ([]CrawledPackage, error) {
	// Always use the canonical host (www redirects here; the collector is restricted to it).
	const base = "https://khasanahtravel.com"
	listURL := base + "/trip"

	var packages []CrawledPackage
	seen := make(map[string]bool)
	cutoff := khasanahCutoff()

	c := NewCollector("khasanahtravel.com")
	detail := c.Clone()

	c.OnHTML(`a[href^="/trip/"]`, func(e *colly.HTMLElement) {
		href := e.Attr("href")
		name := CleanText(e.DOM.Find("h3").First().Text())
		if name == "" || seen[href] {
			return
		}

		// Date range: "19 Jan 2027 – 27 Jan 2027"
		var dateText string
		e.DOM.Find("p").EachWithBreak(func(_ int, s *goquery.Selection) bool {
			t := CleanText(s.Text())
			if khasanahDateRe.MatchString(t) && !strings.Contains(t, "Rp") {
				dateText = t
				return false
			}
			return true
		})
		dates := khasanahDateRe.FindAllString(dateText, -1)
		if len(dates) == 0 {
			return
		}
		start, ok := khasanahParseDate(dates[0])
		if !ok || start.Before(cutoff) {
			return
		}

		// Lowest price (quad); skip the "Bisa DP" line.
		price := 0
		e.DOM.Find("p").EachWithBreak(func(_ int, s *goquery.Selection) bool {
			t := CleanText(s.Text())
			if strings.HasPrefix(t, "Rp") {
				price = ParsePrice(t)
				return false
			}
			return true
		})
		if price == 0 {
			return
		}

		seen[href] = true
		pkg := CrawledPackage{
			TravelName:     "Khasanah Travel",
			PackageName:    name,
			Price:          price,
			DepartureDates: []string{start.Format("2 Jan 2006")},
			URL:            base + href,
		}
		if len(dates) > 1 {
			if end, ok := khasanahParseDate(dates[len(dates)-1]); ok && !end.Before(start) {
				pkg.Duration = int(end.Sub(start).Hours()/24) + 1
			}
		}
		if m := khasanahSeatsRe.FindStringSubmatch(e.Text); len(m) > 1 {
			pkg.Seats, _ = strconv.Atoi(m[1])
		}
		packages = append(packages, pkg)
	})

	if err := c.Visit(listURL); err != nil {
		return nil, err
	}

	for i := range packages {
		pkg := &packages[i]
		detail.OnHTMLDetach("body")
		detail.OnHTML("body", func(e *colly.HTMLElement) {
			khasanahFillDetail(e.DOM, pkg)
		})
		_ = detail.Visit(pkg.URL)
	}

	return packages, nil
}

func khasanahFillDetail(body *goquery.Selection, pkg *CrawledPackage) {
	// Hotels: cards under the "Hotel" h2, name in div.p-4 > p.font-bold
	var unlabeled []string
	body.Find("h2").Each(func(_ int, h *goquery.Selection) {
		if !strings.HasSuffix(CleanText(h.Text()), "Hotel") {
			return
		}
		h.Parent().Find("div.p-4 > p.font-bold").Each(func(_ int, p *goquery.Selection) {
			name := CleanText(p.Text())
			lower := strings.ToLower(name)
			switch {
			case strings.Contains(lower, "makkah") || strings.Contains(lower, "mekkah"):
				if pkg.HotelMakkah == "" {
					pkg.HotelMakkah = khasanahCleanHotel(name)
				}
			case strings.Contains(lower, "madinah"):
				if pkg.HotelMadinah == "" {
					pkg.HotelMadinah = khasanahCleanHotel(name)
				}
			case strings.Contains(lower, "turki") || strings.Contains(lower, "eropa") ||
				strings.Contains(lower, "dubai") || strings.Contains(lower, "aqsa") ||
				strings.Contains(lower, "jordan") || strings.Contains(lower, "mesir"):
				// hotel outside the Holy Cities
			default:
				unlabeled = append(unlabeled, khasanahCleanHotel(name))
			}
		})
	})
	// Unlabeled hotels are listed Makkah first, then Madinah.
	for _, h := range unlabeled {
		if pkg.HotelMakkah == "" {
			pkg.HotelMakkah = h
		} else if pkg.HotelMadinah == "" {
			pkg.HotelMadinah = h
		}
	}

	text := body.Text()
	idxCGK := strings.Index(text, "Soekarno")
	idxSIN := strings.Index(text, "Changi")
	switch {
	case idxSIN >= 0 && (idxCGK < 0 || idxSIN < idxCGK):
		pkg.Airport = "SIN"
	case idxCGK >= 0:
		pkg.Airport = "CGK"
	}

	if pkg.Airline == "" {
		lower := strings.ToLower(text)
		for _, kw := range []string{"saudia airlines", "saudi arabian airlines", "garuda indonesia", "qatar airways", "emirates airline", "etihad airways", "turkish airlines", "oman air", "lion air", "batik air"} {
			if strings.Contains(lower, kw) {
				pkg.Airline = KnownAirline(kw)
				break
			}
		}
	}
}

func khasanahCleanHotel(s string) string {
	s = strings.TrimSpace(s)
	for _, p := range []string{"Makkah - ", "Mekkah - ", "Madinah - "} {
		s = strings.TrimPrefix(s, p)
	}
	return strings.TrimSpace(s)
}

func khasanahCutoff() time.Time {
	min := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if today.After(min) {
		return today
	}
	return min
}

var khasanahMonths = map[string]time.Month{
	"jan": time.January, "januari": time.January, "january": time.January,
	"feb": time.February, "februari": time.February, "february": time.February,
	"mar": time.March, "maret": time.March, "march": time.March,
	"apr": time.April, "april": time.April,
	"mei": time.May, "may": time.May,
	"jun": time.June, "juni": time.June, "june": time.June,
	"jul": time.July, "juli": time.July, "july": time.July,
	"agu": time.August, "agt": time.August, "aug": time.August, "agustus": time.August, "august": time.August,
	"sep": time.September, "sept": time.September, "september": time.September,
	"okt": time.October, "oct": time.October, "oktober": time.October, "october": time.October,
	"nov": time.November, "november": time.November,
	"des": time.December, "dec": time.December, "desember": time.December, "december": time.December,
}

// khasanahParseDate parses "19 Jan 2027" / "19 Januari 2027".
func khasanahParseDate(s string) (time.Time, bool) {
	m := khasanahDateRe.FindStringSubmatch(s)
	if len(m) < 4 {
		return time.Time{}, false
	}
	mon, ok := khasanahMonths[strings.ToLower(m[2])]
	if !ok {
		return time.Time{}, false
	}
	d, _ := strconv.Atoi(m[1])
	y, _ := strconv.Atoi(m[3])
	return time.Date(y, mon, d, 0, 0, 0, 0, time.UTC), true
}
