package crawlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// JejakImaniParser fetches Jejak Imani umroh departures from the public JSON
// API used by their Angular site (www.jejakimani.com):
//
//	GET https://api.jejakimanistage.com/public/catalog/umroh-departures?page=N
//
// Each record is a single departure (date, price, days, airline, hotels,
// available seats). Departures sharing the same sub-category, package,
// duration, airline, hotels and price are merged into one CrawledPackage with
// multiple DepartureDates.
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
)

type jejakImaniHotel struct {
	CityName  string `json:"city_name"`
	HotelName string `json:"hotel_name"`
}

type jejakImaniDeparture struct {
	ID              int               `json:"id"`
	Slug            string            `json:"slug"`
	DepartureDate   string            `json:"departure_date"`
	PackageName     string            `json:"package_name"`
	TotalDays       json.RawMessage   `json:"total_days"`
	AvailableSeat   json.RawMessage   `json:"available_seat"`
	SubCategorySlug string            `json:"sub_category_slug"`
	SubCategoryName string            `json:"sub_category_name"`
	Airlines        string            `json:"airlines"`
	PriceStartFrom  json.RawMessage   `json:"price_start_from"`
	Hotels          []jejakImaniHotel `json:"hotels"`
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

func (j *JejakImaniParser) fetchPage(base string, page int) (*jejakImaniPage, error) {
	url := fmt.Sprintf("%scatalog/umroh-departures?page=%d", base, page)
	log.Println("[crawler] visiting:", url)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")
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
	body, err := io.ReadAll(resp.Body)
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
		pkg   CrawledPackage
		base  string
		first time.Time
		seen  map[string]bool
	}
	groups := map[string]*group{}
	var order []string

	for _, d := range all {
		dep, err := time.Parse("2006-01-02", strings.Fields(d.DepartureDate + " ")[0])
		if err != nil || dep.Before(cutoff) {
			continue
		}
		price := jejakImaniRawInt(d.PriceStartFrom)
		if price <= 0 {
			continue
		}
		days := jejakImaniRawInt(d.TotalDays)

		var makkah, madinah string
		for _, h := range d.Hotels {
			city := strings.ToLower(h.CityName)
			switch {
			case strings.Contains(city, "makkah") || strings.Contains(city, "mekkah") || strings.Contains(city, "mecca"):
				if makkah == "" {
					makkah = jejakImaniTitle(h.HotelName)
				}
			case strings.Contains(city, "madinah") || strings.Contains(city, "medina"):
				if madinah == "" {
					madinah = jejakImaniTitle(h.HotelName)
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

		key := strings.Join([]string{baseName, airline, makkah, madinah, strconv.Itoa(price)}, "|")
		g, ok := groups[key]
		if !ok {
			url := jejakImaniSiteURL + "/umroh"
			if d.SubCategorySlug != "" && d.Slug != "" {
				url = fmt.Sprintf("%s/umroh-products/%s/keberangkatan/%s", jejakImaniSiteURL, d.SubCategorySlug, d.Slug)
			}
			g = &group{
				pkg: CrawledPackage{
					TravelName:     "Jejak Imani",
					PackageName:    baseName,
					Price:          price,
					Duration:       days,
					Airline:        airline,
					HotelMakkah:    makkah,
					HotelMadinah:   madinah,
					DepartureDates: []string{},
					Airport:        "Jakarta",
					URL:            url,
				},
				base:  baseName,
				first: dep,
				seen:  map[string]bool{},
			}
			groups[key] = g
			order = append(order, key)
		}
		ds := dep.Format("02 Jan 2006")
		if !g.seen[ds] {
			g.seen[ds] = true
			g.pkg.DepartureDates = append(g.pkg.DepartureDates, ds)
			g.pkg.Seats += jejakImaniRawInt(d.AvailableSeat)
		}
		if dep.Before(g.first) {
			g.first = dep
		}
	}

	// The importer dedupes by (travel, package name), so groups with the same
	// base name but different price/hotels get a distinguishing suffix.
	nameCount := map[string]int{}
	for _, k := range order {
		nameCount[groups[k].base]++
	}

	sort.SliceStable(order, func(a, b int) bool {
		ga, gb := groups[order[a]], groups[order[b]]
		if ga.base != gb.base {
			return ga.base < gb.base
		}
		return ga.pkg.Price < gb.pkg.Price
	})

	usedNames := map[string]bool{}
	var packages []CrawledPackage
	for _, k := range order {
		g := groups[k]
		if nameCount[g.base] > 1 {
			g.pkg.PackageName = g.base + " (" + jejakImaniRupiah(g.pkg.Price) + ")"
			if usedNames[g.pkg.PackageName] && g.pkg.HotelMakkah != "" {
				g.pkg.PackageName = g.base + " (" + jejakImaniRupiah(g.pkg.Price) + ", " + g.pkg.HotelMakkah + ")"
			}
		}
		usedNames[g.pkg.PackageName] = true
		sort.Slice(g.pkg.DepartureDates, func(a, b int) bool {
			ta, _ := time.Parse("02 Jan 2006", g.pkg.DepartureDates[a])
			tb, _ := time.Parse("02 Jan 2006", g.pkg.DepartureDates[b])
			return ta.Before(tb)
		})
		packages = append(packages, g.pkg)
	}
	return packages, nil
}
