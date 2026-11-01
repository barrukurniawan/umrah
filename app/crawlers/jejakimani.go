package crawlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// JejakImaniParser fetches Jejak Imani umroh departures from the public JSON
// API used by their Angular site (www.jejakimani.com):
//
//	GET https://api.jejakimanistage.com/public/catalog/umroh-departures?page=N
//
// Each record is a single departure (date, price, total_days, airline, hotels
// with city/star/check-in/check-out, available seats, sold-out flag, ustadz,
// day-by-day itinerary). Each departure also has a server-rendered page
// (/umroh-products/<sub>/keberangkatan/<slug>) whose "Fasilitas yang
// didapatkan" / "Biaya Tidak Termasuk" lists are read for facilities.
//
// Departures sharing sub-category, package, duration, airline and hotels are
// merged into one CrawledPackage with one Departures entry per date (price,
// seats and sold-out per date).
type JejakImaniParser struct {
	// APIBase defaults to https://api.jejakimanistage.com/public/ when empty.
	APIBase string
}

func NewJejakImaniParser() *JejakImaniParser {
	return &JejakImaniParser{}
}

const (
	jejakImaniDefaultAPI = "https://api.jejakimanistage.com/public/"
	jejakImaniSiteURL    = "https://www.jejakimani.com"
	jejakImaniMaxPages   = 50
	jejakImaniUA         = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
)

var (
	// "Hotel Makkah bintang 5/setaraf (jarak ± 650m)"
	jejakImaniHotelFacRe = regexp.MustCompile(`(?i)^hotel\s+(makkah|mekkah|madinah)\s+bintang\s*(\d)[^(]*\(\s*jarak\s*[±~]?\s*([\d.,]+)\s*(km|m)\b`)
	jejakImaniSetarafRe  = regexp.MustCompile(`(?i)\s*/\s*setaraf.*$`)
	jejakImaniTagRe      = regexp.MustCompile(`<[^>]+>`)
)

type jejakImaniHotel struct {
	CityName    string          `json:"city_name"`
	HotelName   string          `json:"hotel_name"`
	Star        json.RawMessage `json:"star"`
	CheckIn     string          `json:"check_in"`
	CheckOut    string          `json:"check_out"`
	PackageName string          `json:"package_name"`
}

type jejakImaniItinerary struct {
	DayTitle    string `json:"day_title"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type jejakImaniDeparture struct {
	ID              int                   `json:"id"`
	Slug            string                `json:"slug"`
	Title           string                `json:"title"`
	DepartureDate   string                `json:"departure_date"`
	PackageName     string                `json:"package_name"`
	TotalDays       json.RawMessage       `json:"total_days"`
	AvailableSeat   json.RawMessage       `json:"available_seat"`
	IsSoldOut       bool                  `json:"is_sold_out"`
	SubCategorySlug string                `json:"sub_category_slug"`
	SubCategoryName string                `json:"sub_category_name"`
	Airlines        string                `json:"airlines"`
	PriceStartFrom  json.RawMessage       `json:"price_start_from"`
	Ustadz          string                `json:"ustadz"`
	Hotels          []jejakImaniHotel     `json:"hotels"`
	Itineraries     []jejakImaniItinerary `json:"itineraries"`
}

type jejakImaniPage struct {
	Products struct {
		CurrentPage int                   `json:"current_page"`
		LastPage    int                   `json:"last_page"`
		Data        []jejakImaniDeparture `json:"data"`
	} `json:"products"`
}

// rawInt decodes a JSON number or numeric string ("12", "44900000.00") to int.
func jejakImaniRawInt(raw json.RawMessage) int {
	s := strings.Trim(strings.TrimSpace(string(raw)), `"`)
	if s == "" || s == "null" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return int(f)
}

func jejakImaniGet(url string, accept string) ([]byte, error) {
	log.Println("[crawler] visiting:", url)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", jejakImaniUA)
	req.Header.Set("Accept", accept)
	req.Header.Set("Origin", jejakImaniSiteURL)
	req.Header.Set("Referer", jejakImaniSiteURL+"/")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jejakimani: %s returned HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (j *JejakImaniParser) fetchPage(base string, page int) (*jejakImaniPage, error) {
	body, err := jejakImaniGet(fmt.Sprintf("%scatalog/umroh-departures?page=%d", base, page), "application/json")
	if err != nil {
		return nil, err
	}
	var p jejakImaniPage
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, fmt.Errorf("jejakimani: decode page %d: %w", page, err)
	}
	return &p, nil
}

func jejakImaniTitle(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// jejakImaniHotelName: "AL ANSHAR GOLDEN TULIP" -> "Al Anshar Golden Tulip",
// "Anjum / Setaraf" -> "Anjum".
func jejakImaniHotelName(s string) string {
	s = strings.TrimSpace(jejakImaniSetarafRe.ReplaceAllString(s, ""))
	if s == strings.ToUpper(s) {
		s = jejakImaniTitle(s)
	}
	return strings.Join(strings.Fields(s), " ")
}

// jejakImaniFacilities reads the included/excluded lists of a departure page.
type jejakImaniFacilities struct {
	inc, exc []string
}

func jejakImaniFetchFacilities(url string) (jejakImaniFacilities, error) {
	var f jejakImaniFacilities
	body, err := jejakImaniGet(url, "text/html")
	if err != nil {
		return f, err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return f, err
	}
	read := func(sel string) []string {
		var out []string
		seen := map[string]bool{}
		doc.Find(sel + " li").Each(func(_ int, li *goquery.Selection) {
			t := strings.Join(strings.Fields(li.Text()), " ")
			if t != "" && !seen[t] {
				seen[t] = true
				out = append(out, t)
			}
		})
		return out
	}
	f.inc = read("ul.departure-facility-list.included")
	f.exc = read("ul.departure-facility-list.excluded")
	return f, nil
}

func jejakImaniRupiah(n int) string {
	s := strconv.Itoa(n)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return "Rp" + b.String()
}

func jejakImaniCity(city string) string {
	c := strings.ToLower(city)
	switch {
	case strings.Contains(c, "makkah") || strings.Contains(c, "mekkah") || strings.Contains(c, "mecca"):
		return "makkah"
	case strings.Contains(c, "madinah") || strings.Contains(c, "medina"):
		return "madinah"
	}
	return ""
}

// jejakImaniReturn computes the return date from the stated total_days. The
// itineraries number "Hari 1" as the departure day, so the last day is
// dep + total_days - 1. Used only when the hotel stays agree (the last check-out
// is 0-2 days before that date and no stay starts before departure); otherwise
// the hotel and duration data contradict each other and "" is returned.
func jejakImaniReturn(dep time.Time, days int, hotels []jejakImaniHotel) string {
	if days <= 1 {
		return ""
	}
	ret := dep.AddDate(0, 0, days-1)
	var last time.Time
	for _, h := range hotels {
		if ci, err := time.Parse("2006-01-02", h.CheckIn); err == nil && ci.Before(dep.AddDate(0, 0, -1)) {
			return ""
		}
		if co, err := time.Parse("2006-01-02", h.CheckOut); err == nil && co.After(last) {
			last = co
		}
	}
	if !last.IsZero() && (last.After(ret) || last.Before(ret.AddDate(0, 0, -2))) {
		return ""
	}
	return ret.Format("2006-01-02")
}

func (j *JejakImaniParser) Crawl() ([]CrawledPackage, error) {
	base := j.APIBase
	if base == "" {
		base = jejakImaniDefaultAPI
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}

	cutoff := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	if today := time.Now().UTC().Truncate(24 * time.Hour); today.After(cutoff) {
		cutoff = today
	}

	var all []jejakImaniDeparture
	lastPage := 1
	for page := 1; page <= lastPage && page <= jejakImaniMaxPages; page++ {
		p, err := j.fetchPage(base, page)
		if err != nil {
			if page == 1 {
				return nil, err
			}
			log.Printf("[crawler] jejakimani page %d: %v\n", page, err)
			break
		}
		if p.Products.LastPage > lastPage {
			lastPage = p.Products.LastPage
		}
		if len(p.Products.Data) == 0 {
			break
		}
		all = append(all, p.Products.Data...)
	}

	type group struct {
		pkg      CrawledPackage
		base     string
		seen     map[string]bool
		guides   map[string]bool
		urls     map[string]string // date -> departure page
		minPrice int
	}
	groups := map[string]*group{}
	var order []string
	seenSlug := map[string]bool{}

	for _, d := range all {
		dep, err := time.Parse("2006-01-02", strings.Fields(d.DepartureDate + " ")[0])
		if err != nil || dep.Before(cutoff) || seenSlug[d.Slug] {
			continue
		}
		seenSlug[d.Slug] = true
		price := jejakImaniRawInt(d.PriceStartFrom)
		if price <= 0 {
			continue
		}
		days := jejakImaniRawInt(d.TotalDays)

		var makkah, madinah string
		var starMakkah, starMadinah int
		for _, h := range d.Hotels {
			if h.PackageName != "" && d.PackageName != "" && !strings.EqualFold(h.PackageName, d.PackageName) {
				continue
			}
			switch jejakImaniCity(h.CityName) {
			case "makkah":
				if makkah == "" {
					makkah, starMakkah = jejakImaniHotelName(h.HotelName), jejakImaniRawInt(h.Star)
				}
			case "madinah":
				if madinah == "" {
					madinah, starMadinah = jejakImaniHotelName(h.HotelName), jejakImaniRawInt(h.Star)
				}
			}
		}

		airline := KnownAirline(d.Airlines)
		if airline == "" {
			airline = CleanText(d.Airlines)
		}

		baseName := CleanText(d.SubCategoryName) + " - " + CleanText(d.PackageName)
		if days > 0 {
			baseName += fmt.Sprintf(" %d Hari", days)
		}

		url := jejakImaniSiteURL + "/umroh"
		if d.SubCategorySlug != "" && d.Slug != "" {
			url = fmt.Sprintf("%s/umroh-products/%s/keberangkatan/%s", jejakImaniSiteURL, d.SubCategorySlug, d.Slug)
		}

		key := strings.Join([]string{baseName, airline, makkah, madinah}, "|")
		g, ok := groups[key]
		if !ok {
			g = &group{
				pkg: CrawledPackage{
					TravelName:     "Jejak Imani",
					PackageName:    baseName,
					Duration:       days,
					Airline:        airline,
					HotelMakkah:    makkah,
					HotelMadinah:   madinah,
					StarsMakkah:    starMakkah,
					StarsMadinah:   starMadinah,
					DepartureDates: []string{},
					URL:            url,
				},
				base:   baseName,
				seen:   map[string]bool{},
				guides: map[string]bool{},
				urls:   map[string]string{},
			}
			groups[key] = g
			order = append(order, key)
		}
		ds := dep.Format("2006-01-02")
		if g.seen[ds] {
			continue
		}
		g.seen[ds] = true

		seats := jejakImaniRawInt(d.AvailableSeat)
		cd := CrawledDeparture{
			Date:       ds,
			ReturnDate: jejakImaniReturn(dep, days, d.Hotels),
			Price:      price,
			Seats:      seats,
			Full:       d.IsSoldOut,
		}
		if d.IsSoldOut {
			cd.Seats = 0
		}
		// Departure airport from the travel's day-1 itinerary.
		if len(d.Itineraries) > 0 {
			d1 := d.Itineraries[0].Description
			if strings.Contains(d1, "(CGK)") || strings.Contains(strings.ToLower(d1), "soekarno") {
				cd.Airport = "Jakarta"
			}
		}
		g.pkg.Departures = append(g.pkg.Departures, cd)
		if d.IsSoldOut {
			// Sold-out departures have no page (API/site answer 404).
			url = fmt.Sprintf("%s/umroh/%s#%s", jejakImaniSiteURL, d.SubCategorySlug, d.Slug)
		}
		g.urls[ds] = url
		if g.minPrice == 0 || price < g.minPrice {
			g.minPrice = price
		}
		guide := strings.TrimSpace(jejakImaniTagRe.ReplaceAllString(d.Ustadz, ""))
		if guide != "" && !strings.Contains(strings.ToLower(guide), "assatidz") {
			g.guides[guide] = true
		} else {
			g.guides["?"] = true
		}
	}

	var packages []CrawledPackage
	for _, k := range order {
		g := groups[k]
		p := &g.pkg
		sort.Slice(p.Departures, func(a, b int) bool { return p.Departures[a].Date < p.Departures[b].Date })
		// Earliest bookable departure's own page (sold-out ones have none).
		p.URL = g.urls[p.Departures[0].Date]
		for _, dd := range p.Departures {
			if !dd.Full {
				p.URL = g.urls[dd.Date]
				break
			}
		}
		p.Price = g.minPrice
		airports := map[string]bool{}
		for i := range p.Departures {
			dd := &p.Departures[i]
			t, _ := time.Parse("2006-01-02", dd.Date)
			p.DepartureDates = append(p.DepartureDates, t.Format("02 Jan 2006"))
			if !dd.Full {
				p.Seats += dd.Seats
			}
			airports[dd.Airport] = true
		}
		if len(airports) == 1 {
			for a := range airports {
				p.Airport = a
			}
		}
		allFull := true
		for _, dd := range p.Departures {
			allFull = allFull && dd.Full
		}
		p.Full = allFull
		if len(g.guides) == 1 && !g.guides["?"] {
			for gname := range g.guides {
				p.Guide = gname
			}
		}
		packages = append(packages, *p)
	}

	// Facilities from the earliest departure page of each package.
	for i := range packages {
		p := &packages[i]
		if strings.Contains(p.URL, "#") {
			continue // all departures sold out: no departure page
		}
		f, err := jejakImaniFetchFacilities(p.URL)
		if err != nil {
			log.Printf("[crawler] jejakimani facilities %s: %v\n", p.URL, err)
			continue
		}
		for _, line := range f.inc {
			if m := jejakImaniHotelFacRe.FindStringSubmatch(line); m != nil {
				star, _ := strconv.Atoi(m[2])
				dist := ParsePrice(m[3])
				if strings.EqualFold(m[4], "km") {
					if v, err := strconv.ParseFloat(strings.ReplaceAll(m[3], ",", "."), 64); err == nil {
						dist = int(v * 1000)
					}
				}
				// The list is written per product; use the distance only when its
				// star rating matches this departure's hotel.
				if strings.EqualFold(m[1], "madinah") {
					if star == p.StarsMadinah && p.HotelMadinah != "" {
						p.DistanceMadinah = dist
					}
				} else if star == p.StarsMakkah && p.HotelMakkah != "" {
					p.DistanceMakkah = dist
				}
			}
			lower := strings.ToLower(line)
			if strings.HasPrefix(lower, "hotel makkah") || strings.HasPrefix(lower, "hotel mekkah") || strings.HasPrefix(lower, "hotel madinah") {
				continue // shown through the hotel fields
			}
			p.Facilities = append(p.Facilities, line)
		}
		p.Excludes = f.exc
		time.Sleep(200 * time.Millisecond)
	}

	// Names must be unique per travel: same base name with different hotels
	// gets the Makkah (then Madinah) hotel as suffix.
	nameCount := map[string]int{}
	for _, p := range packages {
		nameCount[p.PackageName]++
	}
	used := map[string]bool{}
	for i := range packages {
		p := &packages[i]
		if nameCount[p.PackageName] > 1 {
			base := p.PackageName
			name := base
			if p.HotelMakkah != "" {
				name = base + " (" + p.HotelMakkah + ")"
			}
			if used[name] && p.HotelMadinah != "" {
				name = base + " (" + strings.Trim(p.HotelMakkah+" / "+p.HotelMadinah, " /") + ")"
			}
			if used[name] {
				name = base + " (" + jejakImaniRupiah(p.Price) + ")"
			}
			p.PackageName = name
		}
		used[p.PackageName] = true
	}

	sort.SliceStable(packages, func(a, b int) bool {
		if packages[a].PackageName != packages[b].PackageName {
			return packages[a].PackageName < packages[b].PackageName
		}
		return packages[a].Departures[0].Date < packages[b].Departures[0].Date
	})
	return packages, nil
}
