package crawlers

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

// LabbaikaParser scrapes PT Labbaika Cipta Imani's paginated umrah listing
// (https://labbaika.co.id/transaksi/paket-umrah?page=N). Each card is one
// departure and already carries date, hotels, airline, airport, seats and the
// starting price, so detail pages are not needed. The listing is sorted by
// departure date; undated ("Tanggal Belum Tersedia") cards come last and are skipped.
type LabbaikaParser struct {
	URL string
}

var (
	labbaikaAirportRe = regexp.MustCompile(`\(([A-Z]{3})\)`)
	labbaikaSeatRe    = regexp.MustCompile(`(?i)sisa\s*seat\s*:?\s*(\d+)`)
	labbaikaCityRe    = regexp.MustCompile(`\s*\((?i:makkah|mekkah|madinah)\)\s*$`)
	labbaikaDateRe    = regexp.MustCompile(`(\d{1,2})\s+([A-Za-z]+)\s+(\d{4})`)
)

var labbaikaMonths = map[string]time.Month{
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

func labbaikaParseDate(s string) (time.Time, bool) {
	m := labbaikaDateRe.FindStringSubmatch(s)
	if len(m) < 4 {
		return time.Time{}, false
	}
	mon, ok := labbaikaMonths[strings.ToLower(m[2])]
	if !ok {
		return time.Time{}, false
	}
	d, _ := strconv.Atoi(m[1])
	y, _ := strconv.Atoi(m[3])
	return time.Date(y, mon, d, 0, 0, 0, 0, time.UTC), true
}

// labbaikaCutoff: departures before today (and never before 2026-10-31) are dropped.
func labbaikaCutoff() time.Time {
	min := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if today.After(min) {
		return today
	}
	return min
}

const labbaikaMaxPages = 40

func (l *LabbaikaParser) Crawl() ([]CrawledPackage, error) {
	listURL := "https://labbaika.co.id/transaksi/paket-umrah"
	if strings.Contains(l.URL, "/transaksi/paket-umrah") {
		listURL = strings.SplitN(l.URL, "?", 2)[0]
	}

	var packages []CrawledPackage
	seen := make(map[string]bool)
	cutoff := labbaikaCutoff()

	c := NewCollector("labbaika.co.id")
	cardsOnPage := 0

	c.OnHTML("div.list-paket-umrah", func(e *colly.HTMLElement) {
		cardsOnPage++
		title := e.DOM.Find("a.paket-title").First()
		name := CleanText(title.Text())
		href, _ := title.Attr("href")
		if name == "" {
			return
		}

		pkg := CrawledPackage{
			TravelName:  "PT Labbaika Cipta Imani",
			PackageName: name,
			URL:         href,
			Duration:    ParseDuration(name),
		}

		var dateCell string
		var hotels []string
		e.DOM.Find("table tr").Each(func(_ int, tr *goquery.Selection) {
			icon, _ := tr.Find("i").Attr("class")
			val := CleanText(tr.Find("td").Last().Text())
			switch {
			case strings.Contains(icon, "fa-calendar"):
				// "02 Nov 2026, 22 Nov 2026 (+3 Tanggal Lainnya)": extra dates live in a.title-tip[title].
				dateCell = val
				if extra, ok := tr.Find("a.title-tip").Attr("title"); ok {
					dateCell += ", " + extra
				}
			case strings.Contains(icon, "fa-building"):
				hotels = append(hotels, val)
			case strings.Contains(icon, "fa-plane"):
				if a := KnownAirline(val); a != "" {
					pkg.Airline = a
				} else {
					pkg.Airline = val
				}
			case strings.Contains(icon, "fa-map-marker"):
				if m := labbaikaAirportRe.FindStringSubmatch(val); len(m) > 1 {
					pkg.Airport = m[1]
				} else {
					pkg.Airport = val
				}
			}
		})

		for _, h := range hotels {
			lower := strings.ToLower(h)
			clean := strings.TrimSpace(labbaikaCityRe.ReplaceAllString(h, ""))
			switch {
			case strings.Contains(lower, "(makkah)") || strings.Contains(lower, "(mekkah)"):
				pkg.HotelMakkah = clean
			case strings.Contains(lower, "(madinah)"):
				pkg.HotelMadinah = clean
			}
		}

		for _, raw := range labbaikaDateRe.FindAllString(dateCell, -1) {
			if t, ok := labbaikaParseDate(raw); ok && !t.Before(cutoff) {
				pkg.DepartureDates = append(pkg.DepartureDates, t.Format("2 Jan 2006"))
			}
		}
		if len(pkg.DepartureDates) == 0 {
			return
		}

		// "IDR 38.900.000,00" -> drop decimals before parsing.
		priceText := CleanText(e.DOM.Find(".harga-paket").First().Text())
		if i := strings.LastIndex(priceText, ","); i > 0 {
			priceText = priceText[:i]
		}
		pkg.Price = ParsePrice(priceText)
		if pkg.Price == 0 {
			return
		}

		if m := labbaikaSeatRe.FindStringSubmatch(e.DOM.Text()); len(m) > 1 {
			pkg.Seats, _ = strconv.Atoi(m[1])
		}

		key := href + "|" + strings.Join(pkg.DepartureDates, ",")
		if seen[key] {
			return
		}
		seen[key] = true
		packages = append(packages, pkg)
	})

	for page := 1; page <= labbaikaMaxPages; page++ {
		cardsOnPage = 0
		u := listURL
		if page > 1 {
			u = fmt.Sprintf("%s?page=%d", listURL, page)
		}
		if err := c.Visit(u); err != nil {
			if page == 1 {
				return nil, err
			}
			break
		}
		if cardsOnPage == 0 {
			break
		}
	}

	return packages, nil
}
