package crawlers

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

// RahmahParser scrapes the Rahmah Travel schedule table at
// https://rahmahtravel.id/jadwal-biaya-umroh-2026-2027 (server-rendered Astro page).
// Every table row (tr.schedule-row) is one departure with its own detail page, so
// one CrawledPackage is emitted per row:
//
//	berangkat | pulang | durasi ("12 hari") | maskapai | harga ("Rp 23.900.000") | sisa seat ("6 seat") | link
//
// Package names come from the JSON-LD ItemList (first ~30 rows) or, as fallback,
// from the WhatsApp CTA text "...dengan Paket <name> <d Month> yang saya lihat...".
// Hotels are not exposed reliably (detail pages only show a free-form room
// category), so HotelMakkah/HotelMadinah stay empty.
type RahmahParser struct {
	URL string
}

var (
	rahmahDateRe     = regexp.MustCompile(`(\d{1,2})\s+([A-Za-z]+)\s+(20\d{2})`)
	rahmahWANameRe   = regexp.MustCompile(`(?i)Paket\s+(.+?)\s+yang saya`)
	rahmahTrailDate  = regexp.MustCompile(`\s+\d{1,2}\s+[A-Za-z]+$`)
	rahmahNameDayRe  = regexp.MustCompile(`(?i)\b(\d{1,2})\s*(?:D|Hari)\b`)
	rahmahFirstNumRe = regexp.MustCompile(`\d+`)
	rahmahLDRe       = regexp.MustCompile(`(?s)<script type="application/ld\+json"[^>]*>(.*?)</script>`)
)

var rahmahMonths = map[string]time.Month{
	"januari": time.January, "january": time.January, "jan": time.January,
	"februari": time.February, "february": time.February, "feb": time.February,
	"maret": time.March, "march": time.March, "mar": time.March,
	"april": time.April, "apr": time.April,
	"mei": time.May, "may": time.May,
	"juni": time.June, "june": time.June, "jun": time.June,
	"juli": time.July, "july": time.July, "jul": time.July,
	"agustus": time.August, "august": time.August, "agu": time.August, "agt": time.August, "aug": time.August,
	"september": time.September, "sep": time.September, "sept": time.September,
	"oktober": time.October, "october": time.October, "okt": time.October, "oct": time.October,
	"november": time.November, "nov": time.November,
	"desember": time.December, "december": time.December, "des": time.December, "dec": time.December,
}

// rahmahParseDate finds the first "d Month yyyy" date (Indonesian or English) in s.
func rahmahParseDate(s string) (time.Time, bool) {
	m := rahmahDateRe.FindStringSubmatch(s)
	if m == nil {
		return time.Time{}, false
	}
	mon, ok := rahmahMonths[strings.ToLower(m[2])]
	if !ok {
		return time.Time{}, false
	}
	d, _ := strconv.Atoi(m[1])
	y, _ := strconv.Atoi(m[3])
	return time.Date(y, mon, d, 0, 0, 0, 0, time.UTC), true
}

func (r *RahmahParser) Crawl() ([]CrawledPackage, error) {
	var packages []CrawledPackage
	seen := make(map[string]bool)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	base, err := url.Parse(r.URL)
	if err != nil {
		return nil, err
	}

	c := NewCollector(base.Hostname())

	// url -> product name from the JSON-LD ItemList (has nicer names, e.g. "(Grup 2)").
	ldNames := make(map[string]string)
	c.OnResponse(func(resp *colly.Response) {
		for _, m := range rahmahLDRe.FindAllSubmatch(resp.Body, -1) {
			var list struct {
				Type  string `json:"@type"`
				Items []struct {
					Item struct {
						Name string `json:"name"`
						URL  string `json:"url"`
					} `json:"item"`
				} `json:"itemListElement"`
			}
			if json.Unmarshal(m[1], &list) != nil || list.Type != "ItemList" {
				continue
			}
			for _, it := range list.Items {
				if it.Item.URL != "" && it.Item.Name != "" {
					ldNames[it.Item.URL] = CleanText(it.Item.Name)
				}
			}
		}
	})

	c.OnHTML("tr.schedule-row", func(e *colly.HTMLElement) {
		var cells []string
		e.ForEach("td", func(_ int, td *colly.HTMLElement) {
			cells = append(cells, strings.Join(strings.Fields(td.Text), " "))
		})
		if len(cells) < 6 {
			return
		}

		dep, ok := rahmahParseDate(cells[0])
		if !ok || dep.Before(today) {
			return
		}

		detail := e.ChildAttr("a.schedule-row-link", "href")
		if detail == "" {
			detail = e.ChildAttr("a[href^='/paket/']", "href")
		}
		pkgURL := e.Request.AbsoluteURL(detail)
		if pkgURL == "" {
			pkgURL = r.URL
		}

		name := ldNames[pkgURL]
		if name == "" {
			if wa, err := url.Parse(e.ChildAttr("a.schedule-row-cta", "href")); err == nil {
				if m := rahmahWANameRe.FindStringSubmatch(wa.Query().Get("text")); m != nil {
					name = rahmahTrailDate.ReplaceAllString(m[1], "")
				}
			}
		}
		if name == "" && detail != "" {
			slug := strings.TrimPrefix(detail, "/paket/")
			name = strings.Title(strings.ReplaceAll(slug, "-", " ")) //nolint:staticcheck
		}
		name = strings.Join(strings.Fields(strings.ReplaceAll(name, "()", "")), " ")
		if name == "" {
			return
		}

		pkg := CrawledPackage{
			TravelName:     "Rahmah Travel",
			PackageName:    name,
			Price:          ParsePrice(cells[4]),
			Airline:        KnownAirline(cells[3]),
			DepartureDates: []string{dep.Format("2 Jan 2006")},
			URL:            pkgURL,
		}
		if pkg.Airline == "" {
			pkg.Airline = cells[3]
		}
		if n := rahmahFirstNumRe.FindString(cells[5]); n != "" && strings.Contains(strings.ToLower(cells[5]), "seat") {
			pkg.Seats, _ = strconv.Atoi(n)
		}
		// Prefer the marketed duration in the name ("9D", "12 D", "9 Hari"); the
		// table's "durasi" is computed from the dates and is often one day shorter.
		if m := rahmahNameDayRe.FindStringSubmatch(name); m != nil {
			pkg.Duration, _ = strconv.Atoi(m[1])
		}
		if pkg.Duration == 0 {
			if n := rahmahFirstNumRe.FindString(cells[2]); n != "" {
				pkg.Duration, _ = strconv.Atoi(n)
			}
		}

		key := fmt.Sprintf("%s|%s|%d", pkg.URL, pkg.DepartureDates[0], pkg.Price)
		if pkg.Price > 0 && !seen[key] {
			seen[key] = true
			packages = append(packages, pkg)
		}
	})

	if err := c.Visit(r.URL); err != nil {
		return packages, err
	}
	c.Wait()
	return packages, nil
}
