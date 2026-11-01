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
)

// TaibaParser scrapes https://taibamedina.com/paket-umroh (server-rendered,
// paginated with ?page=N). Every listing card is one departure with its own
// detail page (/paket-umroh/<slug>):
//
//   - card:   "<N> Hari", "Sisa <N> kursi", "Mulai dari Rp ..."
//   - detail: "Keberangkatan: <hari>, <d> <bulan> <yyyy>", Maskapai, Bandara,
//     optional description ("Durasi: 9 hari | ..."), and "Pilihan Tingkatan
//     Paket" with per-tier hotels (sometimes "*4" stars) and Quad/Triple/Double.
//
// The site never shows a return date, so ReturnDate stays empty.
type TaibaParser struct {
	URL string
}

var (
	taibaDurRe     = regexp.MustCompile(`(?i)^\s*(\d{1,2})\s*Hari\s*$`)
	taibaSeatRe    = regexp.MustCompile(`(?i)Sisa\s+(\d+)\s+kursi`)
	taibaDescDurRe = regexp.MustCompile(`(?i)Durasi\s*:?\s*(\d{1,2})\s*hari`)
	taibaDateRe    = regexp.MustCompile(`(?i)(\d{1,2})\s+(` + lstMonthAlt + `)\.?\s+(20\d{2})`)
	taibaStarRe    = regexp.MustCompile(`\*\s*([1-5])\b`)
	taibaPageRe    = regexp.MustCompile(`[?&]page=(\d+)`)
	taibaRoomRe    = regexp.MustCompile(`(?i)^(Quad|Triple|Double)\s*:\s*(.+)$`)
	taibaDirectRe  = regexp.MustCompile(`(?i)\bdirect\b`)
)

type taibaCard struct {
	url      string
	duration int
	seats    int
	hasSeats bool
	price    int
}

type taibaTier struct {
	makkah, madinah           string
	starsMakkah, starsMadinah int
	quad, triple, double      int
}

func (t *TaibaParser) Crawl() ([]CrawledPackage, error) {
	cutoff := lstCutoff()
	base, err := url.Parse(t.URL)
	if err != nil {
		return nil, err
	}

	// 1) Listing pages.
	var cards []taibaCard
	seenURL := map[string]bool{}
	maxPage := 1
	for page := 1; page <= maxPage && page <= 20; page++ {
		pu := t.URL
		if page > 1 {
			pu = fmt.Sprintf("%s?page=%d", t.URL, page)
		}
		log.Println("[crawler] visiting:", pu)
		html, err := lstFetch(pu)
		if err != nil {
			if page == 1 {
				return nil, err
			}
			log.Printf("[taiba] %s: %v", pu, err)
			continue
		}
		for _, m := range taibaPageRe.FindAllStringSubmatch(html, -1) {
			if n, _ := strconv.Atoi(m[1]); n > maxPage {
				maxPage = n
			}
		}
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			continue
		}
		doc.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
			href, _ := a.Attr("href")
			u, err := base.Parse(href)
			if err != nil || u.Hostname() != base.Hostname() || !strings.HasPrefix(u.Path, "/paket-umroh/") {
				return
			}
			if a.Find("h3").Length() == 0 {
				return
			}
			u.RawQuery, u.Fragment = "", ""
			c := taibaCard{url: u.String()}
			if seenURL[c.url] {
				return
			}
			a.Find("span").Each(func(_ int, sp *goquery.Selection) {
				txt := strings.TrimSpace(sp.Text())
				if m := taibaDurRe.FindStringSubmatch(txt); m != nil && c.duration == 0 {
					c.duration, _ = strconv.Atoi(m[1])
				}
				if m := taibaSeatRe.FindStringSubmatch(txt); m != nil {
					c.seats, _ = strconv.Atoi(m[1])
					c.hasSeats = true
				}
			})
			c.price = ParsePrice(a.Find("p.text-2xl").First().Text())
			seenURL[c.url] = true
			cards = append(cards, c)
		})
		time.Sleep(lstPoliteDelay)
	}

	// 2) Detail pages.
	var packages []CrawledPackage
	for _, c := range cards {
		log.Println("[crawler] visiting:", c.url)
		html, err := lstFetch(c.url)
		if err != nil {
			log.Printf("[taiba] %s: %v", c.url, err)
			continue
		}
		time.Sleep(lstPoliteDelay)
		if p, ok := t.parseDetail(html, c, cutoff); ok {
			packages = append(packages, p)
		}
	}
	return packages, nil
}

func (t *TaibaParser) parseDetail(html string, c taibaCard, cutoff time.Time) (CrawledPackage, bool) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return CrawledPackage{}, false
	}
	pkg := CrawledPackage{
		TravelName:  "Taiba Medina",
		URL:         c.url,
		PackageName: strings.Join(strings.Fields(doc.Find("h1").First().Text()), " "),
	}

	// Departure date.
	var dep time.Time
	doc.Find("p").EachWithBreak(func(_ int, p *goquery.Selection) bool {
		txt := strings.Join(strings.Fields(p.Text()), " ")
		if !strings.HasPrefix(strings.ToLower(txt), "keberangkatan") {
			return true
		}
		if m := taibaDateRe.FindStringSubmatch(txt); m != nil {
			d, _ := strconv.Atoi(m[1])
			y, _ := strconv.Atoi(m[3])
			dep, _ = lstMakeDate(y, lstMonthNum(m[2]), d)
		}
		return false
	})
	if dep.IsZero() || dep.Before(cutoff) {
		return pkg, false
	}

	// Info boxes: Maskapai / Bandara / Transportasi.
	doc.Find("div.rounded-lg > span.block").Each(func(_ int, sp *goquery.Selection) {
		label := strings.ToLower(strings.TrimSpace(sp.Text()))
		val := strings.TrimSpace(sp.Parent().Find("strong").First().Text())
		if val == "" || val == "-" || strings.EqualFold(val, "request") {
			return
		}
		switch label {
		case "maskapai":
			if a := KnownAirline(val); a != "" {
				pkg.Airline = a
			} else {
				pkg.Airline = val
			}
		case "bandara":
			pkg.Airport = taibaAirport(val)
		}
	})

	// Duration: description "Durasi: N hari" first, else the card badge.
	desc := strings.Join(strings.Fields(doc.Find("div.prose").First().Text()), " ")
	if m := taibaDescDurRe.FindStringSubmatch(desc); m != nil {
		pkg.Duration, _ = strconv.Atoi(m[1])
		if c.duration > 0 && c.duration != pkg.Duration {
			log.Printf("[taiba] %s: durasi deskripsi %d hari vs kartu %d hari (dipakai deskripsi)", c.url, pkg.Duration, c.duration)
		}
	} else {
		pkg.Duration = c.duration
	}

	// Direct: only when the package name says so.
	// "SLOW UMROH DIRECT" is partly a product-line name: it also appears on
	// Emirates/Qatar/Oman departures, which always connect in their hub, so
	// the claim is only trusted for carriers that fly CGK-KSA non-stop.
	if taibaDirectRe.MatchString(pkg.PackageName) && !taibaHubCarrier(pkg.Airline) {
		v := true
		pkg.Direct = &v
	}

	// Tiers. Package level = cheapest tier by Quad (matches "Mulai dari").
	var tiers []taibaTier
	doc.Find("h3").Each(func(_ int, h *goquery.Selection) {
		if !strings.Contains(strings.ToLower(h.Text()), "tingkatan paket") {
			return
		}
		h.Next().Find("div.border").Each(func(_ int, box *goquery.Selection) {
			var tr taibaTier
			box.Find("p").Each(func(_ int, p *goquery.Selection) {
				txt := strings.TrimSpace(p.Text())
				lower := strings.ToLower(txt)
				if i := strings.Index(txt, ":"); i >= 0 {
					val := strings.TrimSpace(txt[i+1:])
					switch {
					case strings.HasPrefix(lower, "hotel makkah"), strings.HasPrefix(lower, "hotel mekkah"):
						tr.makkah, tr.starsMakkah = taibaHotel(val)
					case strings.HasPrefix(lower, "hotel madinah"):
						tr.madinah, tr.starsMadinah = taibaHotel(val)
					}
				}
			})
			box.Find("li").Each(func(_ int, li *goquery.Selection) {
				if m := taibaRoomRe.FindStringSubmatch(strings.TrimSpace(li.Text())); m != nil {
					v := ParsePrice(m[2])
					switch strings.ToLower(m[1]) {
					case "quad":
						tr.quad = v
					case "triple":
						tr.triple = v
					case "double":
						tr.double = v
					}
				}
			})
			if tr.quad > 0 || tr.makkah != "" {
				tiers = append(tiers, tr)
			}
		})
	})
	var best *taibaTier
	for i := range tiers {
		tr := &tiers[i]
		if tr.quad <= 0 {
			continue
		}
		if best == nil || tr.quad < best.quad || (tr.quad == best.quad && tr.triple > 0 && tr.triple < best.triple) {
			best = tr
		}
	}
	if best != nil {
		pkg.Price = best.quad
		pkg.PriceTriple = best.triple
		pkg.PriceDouble = best.double
		pkg.HotelMakkah, pkg.StarsMakkah = best.makkah, best.starsMakkah
		pkg.HotelMadinah, pkg.StarsMadinah = best.madinah, best.starsMadinah
	}
	if pkg.Price == 0 {
		pkg.Price = c.price
	}
	if pkg.Price < lstMinPrice || pkg.Price > lstMaxPrice {
		return pkg, false
	}

	dd := CrawledDeparture{Date: dep.Format("2006-01-02")}
	if c.hasSeats {
		pkg.Seats = c.seats
		dd.Seats = c.seats
		if c.seats == 0 {
			pkg.Full, dd.Full = true, true
		}
	}
	pkg.Departures = []CrawledDeparture{dd}
	pkg.DepartureDates = []string{lstFormatDate(dep)}
	return pkg, true
}

// taibaHotel turns "Grand Al-Massa *4 / Setaraf" into ("Grand Al-Massa", 4).
func taibaHotel(s string) (string, int) {
	stars := 0
	if m := taibaStarRe.FindStringSubmatch(s); m != nil {
		stars, _ = strconv.Atoi(m[1])
		s = taibaStarRe.ReplaceAllString(s, "")
	}
	if i := strings.Index(s, "/"); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	if s == "-" {
		s = ""
	}
	return s, stars
}

func taibaAirport(s string) string {
	l := strings.ToLower(s)
	if strings.Contains(l, "soekarno") || strings.Contains(l, "cgk") || strings.Contains(l, "cengkareng") {
		return "Jakarta"
	}
	return s
}

// taibaHubCarrier reports airlines whose Indonesia-Saudi flights always
// connect through their home hub (so they can never be "direct").
func taibaHubCarrier(airline string) bool {
	switch airline {
	case "Emirates", "Qatar Airways", "Oman Air", "Etihad Airways", "Turkish Airlines", "EgyptAir", "Scoot", "IndiGo":
		return true
	}
	return false
}
