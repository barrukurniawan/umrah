package crawlers

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

// FandiegoParser scrapes PT Wisata Hati Universal (brand Fandiego Travel) at
// https://fandiegotravel.com/. The start page links to program pages
// (/umroh-reguler/<slug>/, /umroh-plus/<slug>/), each with a "Package Information"
// box: Category, Airlines, Program ("9, 10, 11 Hari"), Package Price
// ("23,9 Juta - 30,9 Juta"), plus include details with hotels.
//
// As of 2026-11-01 the program pages (and sitemap.xml) publish NO departure
// dates: schedules exist only inside flyer images, and the commented-out
// /umroh-reguler/umroh-9D/ and umroh-12D/ pages are 404. So this parser emits
// nothing until a page contains a text date ("15 Januari 2027") on/after
// today. It never invents dates. Fields that are ambiguous on a program page
// ("Program: 9, 10, 11 Hari", several Makkah hotel tiers) are left empty.
type FandiegoParser struct {
	URL string
}

var (
	fandiegoProgramPathRe = regexp.MustCompile(`^/umroh-(reguler|plus)/[^/]+/?$`)
	fandiegoJutaRe        = regexp.MustCompile(`(\d+(?:[.,']\d+)?)\s*Juta`)
	fandiegoDateRe        = regexp.MustCompile(`(?i)\b(\d{1,2})\s+(januari|january|jan|februari|february|feb|maret|march|mar|april|apr|mei|may|juni|june|jun|juli|july|jul|agustus|august|agu|agt|aug|september|sept|sep|oktober|october|okt|oct|november|nov|desember|december|des|dec)\s+(20\d{2})\b`)
	fandiegoNumRe         = regexp.MustCompile(`\d+`)
	fandiegoDPRe          = regexp.MustCompile(`(?i)\bDP\s*(?:Rp\.?\s*)?(\d+(?:[.,']\d+)?\s*Juta)`)
	fandiegoParenRe       = regexp.MustCompile(`\([^)]*\)`)
	fandiegoTierRe        = regexp.MustCompile(`(?i)Paket\s+\w+\s*:\s*(.+?)(?:\s+Paket\s+\w+\s*:|$)`)
)

var fandiegoMonthIdx = map[string]time.Month{
	"jan": 1, "feb": 2, "mar": 3, "apr": 4, "mei": 5, "may": 5, "jun": 6, "jul": 7,
	"agu": 8, "agt": 8, "aug": 8, "sep": 9, "okt": 10, "oct": 10, "nov": 11, "des": 12, "dec": 12,
}

// fandiegoJuta converts "23,9 Juta" / "23'9 Juta" to 23900000.
func fandiegoJuta(s string) int {
	m := fandiegoJutaRe.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	num := strings.NewReplacer(",", ".", "'", ".").Replace(m[1])
	f, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0
	}
	return int(f*10+0.5) * 100000
}

func (f *FandiegoParser) Crawl() ([]CrawledPackage, error) {
	var packages []CrawledPackage
	seen := make(map[string]bool)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	base, err := url.Parse(f.URL)
	if err != nil {
		return nil, err
	}

	c := NewCollector(base.Hostname())

	// Discover program pages from the start page (and the category index pages).
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		abs := e.Request.AbsoluteURL(e.Attr("href"))
		u, err := url.Parse(abs)
		if err != nil || u.Hostname() != base.Hostname() {
			return
		}
		if fandiegoProgramPathRe.MatchString(u.Path) || u.Path == "/umroh-reguler/" || u.Path == "/umroh-plus/" {
			e.Request.Visit(abs)
		}
	})

	c.OnHTML("div.portfolio-info", func(e *colly.HTMLElement) {
		if !fandiegoProgramPathRe.MatchString(e.Request.URL.Path) {
			return
		}
		pageURL := e.Request.URL.String()
		info := e.DOM

		pkg := CrawledPackage{
			TravelName: "PT Wisata Hati Universal",
			URL:        pageURL,
		}

		var airlines []string
		info.Find("li").Each(func(_ int, li *goquery.Selection) {
			label := strings.ToLower(CleanText(li.Find("strong").Text()))
			val := strings.Join(strings.Fields(li.Text()), " ")
			if i := strings.Index(val, ":"); i >= 0 {
				val = strings.TrimSpace(val[i+1:])
			}
			switch {
			case strings.HasPrefix(label, "category"):
				pkg.PackageName = val
			case strings.HasPrefix(label, "program"):
				// Only a single stated duration is usable ("9, 10, 11 Hari" is not).
				if ns := fandiegoNumRe.FindAllString(val, -1); len(ns) == 1 {
					pkg.Duration, _ = strconv.Atoi(ns[0])
				}
			case strings.HasPrefix(label, "package price"):
				pkg.Price = fandiegoJuta(val) // lowest price of the range
			}
		})
		for _, line := range strings.Split(info.Find("p").Text(), "\n") {
			if a := KnownAirline(line); a != "" {
				airlines = append(airlines, a)
			}
		}
		if len(airlines) == 1 {
			pkg.Airline = airlines[0]
		}

		// Hotels from the include list, dates from the whole page text.
		doc := e.DOM.Closest("body")
		var lines []string
		for _, l := range strings.Split(doc.Text(), "\n") {
			if l = CleanText(l); l != "" {
				lines = append(lines, l)
			}
		}
		doc.Find("li").Each(func(_ int, li *goquery.Selection) {
			label := strings.ToLower(li.Find("b").First().Text())
			text := strings.Join(strings.Fields(li.Text()), " ")
			switch {
			case strings.Contains(label, "hotel madinah") && pkg.HotelMadinah == "":
				if i := strings.Index(text, ":"); i >= 0 {
					pkg.HotelMadinah = fandiegoParenRe.ReplaceAllString(text[i+1:], "")
					pkg.HotelMadinah = strings.TrimSpace(pkg.HotelMadinah)
				}
			case (strings.Contains(label, "hotel mekkah") || strings.Contains(label, "hotel makkah")) && pkg.HotelMakkah == "":
				// Several tiers ("Paket Silver : ... Paket Gold : ...") without a
				// per-tier price: only use the hotel when there is exactly one.
				if strings.Count(strings.ToLower(text), "paket ") == 1 {
					if m := fandiegoTierRe.FindStringSubmatch(text); m != nil {
						pkg.HotelMakkah = strings.TrimSpace(fandiegoParenRe.ReplaceAllString(m[1], ""))
					}
				} else if i := strings.Index(text, ":"); i >= 0 && !strings.Contains(strings.ToLower(text), "paket ") {
					pkg.HotelMakkah = strings.TrimSpace(fandiegoParenRe.ReplaceAllString(text[i+1:], ""))
				}
			}
		})

		// Include / exclude lists and the DP note.
		doc.Find("h4").Each(func(_ int, h *goquery.Selection) {
			head := strings.ToLower(h.Text())
			var dst *[]string
			switch {
			case strings.Contains(head, "include"):
				dst = &pkg.Facilities
			case strings.Contains(head, "exclude"):
				dst = &pkg.Excludes
			default:
				return
			}
			h.NextFiltered("ul").Children().Each(func(_ int, li *goquery.Selection) {
				if li.Find("b").Length() > 0 && dst == &pkg.Facilities {
					return // hotel blocks
				}
				t := strings.Join(strings.Fields(li.Text()), " ")
				if t != "" {
					*dst = append(*dst, t)
				}
			})
		})
		if m := fandiegoDPRe.FindStringSubmatch(doc.Text()); m != nil {
			pkg.DownPayment = fandiegoJuta(m[1])
		}

		dateSeen := make(map[string]bool)
		for _, m := range fandiegoDateRe.FindAllStringSubmatch(strings.Join(lines, " "), -1) {
			mon := fandiegoMonthIdx[strings.ToLower(m[2])[:3]]
			d, _ := strconv.Atoi(m[1])
			y, _ := strconv.Atoi(m[3])
			if mon == 0 || d < 1 || d > 31 {
				continue
			}
			t := time.Date(y, mon, d, 0, 0, 0, 0, time.UTC)
			ds := t.Format("2 Jan 2006")
			if !t.Before(today) && !dateSeen[ds] {
				dateSeen[ds] = true
				pkg.DepartureDates = append(pkg.DepartureDates, ds)
				pkg.Departures = append(pkg.Departures, CrawledDeparture{Date: t.Format("2006-01-02")})
			}
		}

		if len(pkg.DepartureDates) == 0 {
			log.Printf("[fandiego] %s: no upcoming departure dates on page, skipped", pageURL)
			return
		}
		key := fmt.Sprintf("%s|%d", pageURL, pkg.Price)
		if pkg.PackageName != "" && pkg.Price > 0 && !seen[key] {
			seen[key] = true
			packages = append(packages, pkg)
		}
	})

	if err := c.Visit(f.URL); err != nil {
		return packages, err
	}
	c.Wait()
	return packages, nil
}
