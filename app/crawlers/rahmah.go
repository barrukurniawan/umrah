package crawlers

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

// RahmahParser scrapes Rahmah Travel (server-rendered Astro site).
//
//  1. The schedule table at https://rahmahtravel.id/jadwal-biaya-umroh-2026-2027
//     (tr.schedule-row) lists every open departure with its own detail page:
//     berangkat | pulang | durasi ("12 hari") | maskapai | harga ("Rp 23.900.000") | sisa seat | link
//  2. Each detail page /paket/<slug> is visited for the package name (breadcrumb
//     JSON-LD), the stated departure/return dates (JSON-LD departureTime /
//     arrivalTime), the room-price tabs (data-category-panel, one tab per hotel
//     category, rooms in [data-room-type][data-room-price]) and the included
//     city tour.
//  3. https://rahmahtravel.id/paket-umroh-direct lists the packages the travel
//     itself labels as direct flights.
//  4. The travel's itinerary site https://itinerary.rahmahtravel.id/ lists one card
//     per batch (name + departure date). Matching batches are visited for the
//     hotel names written in the itinerary, the transit airport and the
//     departure airport.
//
// Every detail page is one departure, so one CrawledPackage (with exactly one
// Departures entry) is emitted per page. Hotels: category tabs are named after
// the Makkah hotel ("NADA AJYAD / SETARAF", "Reguler - Rawdah As Sharia"); the
// tab that holds the listed ("Harga Mulai") price is the one used. Tabs named
// after the package ("Tanpa Kategori", "Umroh 12D By Garuda 3 Juli") give no hotel.
type RahmahParser struct {
	URL string
}

const (
	rahmahBase         = "https://rahmahtravel.id"
	rahmahItineraryURL = "https://itinerary.rahmahtravel.id/"
)

var (
	rahmahDateRe     = regexp.MustCompile(`(\d{1,2})\s+([A-Za-z]+)\s+(20\d{2})`)
	rahmahWANameRe   = regexp.MustCompile(`(?i)Paket\s+(.+?)\s+yang saya`)
	rahmahTrailDate  = regexp.MustCompile(`\s+\d{1,2}\s+[A-Za-z]+$`)
	rahmahNameDayRe  = regexp.MustCompile(`(?i)\b(\d{1,2})\s*(?:D|Hari)\b`)
	rahmahFirstNumRe = regexp.MustCompile(`\d+`)
	rahmahLDRe       = regexp.MustCompile(`(?s)<script type="application/ld\+json"[^>]*>(.*?)</script>`)
	rahmahSisaRe     = regexp.MustCompile(`(?i)sisa\s+(\d+)\s*seat`)

	// Category-name cleanup (the remainder is the Makkah hotel).
	rahmahCatPrefixRe = regexp.MustCompile(`(?i)^(?:reguler|semi mandiri\s*\(room only\)|plus\s*\([^)]*\)|combo\s*\([^)]*\))\s*-?\s*`)
	rahmahCatSuffixRe = regexp.MustCompile(`(?i)\s*(?:/\s*setaraf|\(\s*setaraf\s*\)|\+\s*tha?o?if\b.*|\bnew\b|\breschedule\b)\s*$`)

	// Itinerary patterns.
	rahmahItCardRe    = regexp.MustCompile(`(?s)<a href="/paket\?batch=(\d+)"[^>]*data-name="([^"]*)"[^>]*>(.*?)</a>`)
	rahmahItCardDate  = regexp.MustCompile(`calendar"[^>]*></i>\s*<span>([^<]*)</span>`)
	rahmahItMadinahRe = regexp.MustCompile(`(?i)\bHotel\s+([A-Za-z][^()&/\n]*?)\s*&\s*Masjid\s+Nabawi`)
	rahmahItMakkahRe  = regexp.MustCompile(`(?i)\bHotel\s+([A-Za-z][^()&/\n]*?)\s*&\s*Masjid(?:il|ill)?\s+Haram`)
	rahmahItMakkahP   = regexp.MustCompile(`(?i)Check\s*In\s+Hotel\s+Makkah[^(\n]*\(\s*Hotel\s+([^)&/]+?)\s*\)`)
	rahmahTagRe       = regexp.MustCompile(`<[^>]+>`)
	rahmahScriptRe    = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(?:script|style)>`)
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

// Transit hubs that appear in Rahmah itineraries ("Transit di Doha", "BANDARA MUSCAT").
var rahmahTransitHubs = []struct{ key, city string }{
	{"doha", "Doha"}, {"hamad international", "Doha"}, {"abu dhabi", "Abu Dhabi"},
	{"muscat", "Muscat"}, {"mumbai", "Mumbai"}, {"dubai", "Dubai"},
	{"kuala lumpur", "Kuala Lumpur"}, {"colombo", "Colombo"}, {"istanbul", "Istanbul"},
	{"bahrain", "Bahrain"}, {"delhi", "Delhi"}, {"bengaluru", "Bengaluru"},
	{"bangalore", "Bengaluru"}, {"kochi", "Kochi"}, {"hyderabad", "Hyderabad"},
}

// Indonesian departure airports named in itineraries.
var rahmahDepAirports = []struct{ key, city string }{
	{"soekarno", "Jakarta"}, {"juanda", "Surabaya"}, {"kualanamu", "Medan"},
	{"kertajati", "Majalengka"}, {"sultan hasanuddin", "Makassar"}, {"adi soemarmo", "Solo"},
	{"yogyakarta international", "Yogyakarta"}, {"sultan iskandar muda", "Banda Aceh"},
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

// rahmahRow is one row of the schedule table.
type rahmahRow struct {
	url      string
	name     string // from ItemList / WhatsApp CTA (fallback)
	dep, ret time.Time
	durasi   int
	airline  string
	price    int
	seats    int
	seatText string
}

// rahmahRoom is one room price card on a detail page.
type rahmahRoom struct {
	kind  string // quad / triple / double / single / infant / ""
	price int
}

type rahmahCategory struct {
	name  string
	rooms []rahmahRoom
}

// rahmahDetail is what a detail page publishes.
type rahmahDetail struct {
	name       string
	dep, ret   string // YYYY-MM-DD from JSON-LD
	durasi     int
	cityTour   string
	categories []rahmahCategory
	soldOut    bool
	sisa       int
	sisaFound  bool
	direct     bool // badge "Direct (Tanpa Transit)" / FAQ "Direct, tanpa transit"
}

// rahmahItin is what a batch itinerary publishes.
type rahmahItin struct {
	makkah, madinah string
	transit         string
	airport         string
}

func rahmahNorm(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func rahmahSpace(s string) string { return strings.Join(strings.Fields(s), " ") }

// rahmahTextLines turns an HTML page into its visible text lines.
func rahmahTextLines(body []byte) []string {
	s := rahmahScriptRe.ReplaceAllString(string(body), "")
	s = rahmahTagRe.ReplaceAllString(s, "\n")
	s = html.UnescapeString(s)
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = rahmahSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func rahmahRoomKind(t string) string {
	first := strings.ToLower(strings.TrimSpace(strings.SplitN(t, "-", 2)[0]))
	for _, k := range []string{"quad", "triple", "double", "single", "infant"} {
		if strings.HasPrefix(first, k) {
			return k
		}
	}
	return ""
}

// rahmahCategoryHotel returns the hotel a room category is named after, or "" when
// the category is named after the package itself.
func rahmahCategoryHotel(cat string) string {
	s := rahmahSpace(cat)
	lower := strings.ToLower(s)
	if lower == "" || strings.HasPrefix(lower, "tanpa kategori") || strings.HasPrefix(lower, "umroh") ||
		strings.HasPrefix(lower, "paket") || strings.Contains(lower, "room only)") && !strings.Contains(s, "-") {
		return ""
	}
	s = rahmahCatPrefixRe.ReplaceAllString(s, "")
	for i := 0; i < 3; i++ {
		s = strings.TrimSpace(rahmahCatSuffixRe.ReplaceAllString(s, ""))
	}
	s = strings.Trim(s, " -/")
	if s == "" || strings.HasPrefix(strings.ToLower(s), "umroh") {
		return ""
	}
	// "NADA AJYAD" -> "Nada Ajyad"; keep mixed-case names as written.
	if s == strings.ToUpper(s) || s == strings.ToLower(s) {
		words := strings.Fields(strings.ToLower(s))
		for i, w := range words {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
		s = strings.Join(words, " ")
	}
	return s
}

func rahmahParseDetail(body []byte) rahmahDetail {
	var d rahmahDetail
	for _, m := range rahmahLDRe.FindAllSubmatch(body, -1) {
		var obj map[string]json.RawMessage
		if json.Unmarshal(m[1], &obj) != nil {
			continue
		}
		var typ interface{}
		_ = json.Unmarshal(obj["@type"], &typ)
		types := fmt.Sprint(typ)
		switch {
		case strings.Contains(types, "BreadcrumbList"):
			var bl struct {
				Items []struct {
					Name string `json:"name"`
				} `json:"itemListElement"`
			}
			if json.Unmarshal(m[1], &bl) == nil && len(bl.Items) >= 3 {
				d.name = rahmahSpace(bl.Items[len(bl.Items)-1].Name)
			}
		case strings.Contains(types, "Product") || strings.Contains(types, "TouristTrip"):
			var p struct {
				Description string `json:"description"`
				Departure   string `json:"departureTime"`
				Arrival     string `json:"arrivalTime"`
				Props       []struct {
					Name  string          `json:"name"`
					Value json.RawMessage `json:"value"`
				} `json:"additionalProperty"`
				Offers []struct {
					Availability string `json:"availability"`
				} `json:"offers"`
			}
			if json.Unmarshal(m[1], &p) != nil {
				continue
			}
			if _, err := time.Parse("2006-01-02", p.Departure); err == nil {
				d.dep = p.Departure
			}
			if _, err := time.Parse("2006-01-02", p.Arrival); err == nil {
				d.ret = p.Arrival
			}
			for _, pr := range p.Props {
				switch strings.ToLower(pr.Name) {
				case "durasi":
					d.durasi = jejakImaniRawInt(pr.Value)
				case "city tour tambahan":
					var s string
					if json.Unmarshal(pr.Value, &s) == nil {
						d.cityTour = rahmahSpace(s)
					}
				}
			}
			if len(p.Offers) > 0 {
				all := true
				for _, o := range p.Offers {
					if !strings.Contains(strings.ToLower(o.Availability), "soldout") {
						all = false
					}
				}
				d.soldOut = all
			}
			if m := rahmahSisaRe.FindStringSubmatch(p.Description); m != nil {
				d.sisa, _ = strconv.Atoi(m[1])
				d.sisaFound = true
			}
		}
	}

	d.direct = strings.Contains(string(body), "Direct (Tanpa Transit)") ||
		strings.Contains(string(body), "Direct, tanpa transit")

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return d
	}
	// The page renders the room tabs twice (inline + modal); the modal copy
	// (div.tab-panel) carries data-room-type/data-room-price.
	seen := map[string]bool{}
	doc.Find("div[data-category-panel]").Each(func(_ int, s *goquery.Selection) {
		name := rahmahSpace(s.AttrOr("data-category-panel", ""))
		rooms := s.Find("[data-room-type][data-room-price]")
		if rooms.Length() == 0 || seen[name] {
			return
		}
		seen[name] = true
		cat := rahmahCategory{name: name}
		rooms.Each(func(_ int, r *goquery.Selection) {
			p, _ := strconv.Atoi(strings.TrimSpace(r.AttrOr("data-room-price", "")))
			if p <= 0 {
				return
			}
			cat.rooms = append(cat.rooms, rahmahRoom{kind: rahmahRoomKind(r.AttrOr("data-room-type", "")), price: p})
		})
		if len(cat.rooms) > 0 {
			d.categories = append(d.categories, cat)
		}
	})
	return d
}

func rahmahParseItinerary(body []byte) rahmahItin {
	var it rahmahItin
	text := strings.Join(rahmahTextLines(body), "\n")
	uniq := func(re *regexp.Regexp) string {
		found := ""
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			h := strings.TrimSpace(m[1])
			h = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(h, "Madinah"), "Makkah"))
			if h == "" {
				continue
			}
			if found != "" && !strings.EqualFold(found, h) {
				return "" // several hotels named: ambiguous
			}
			found = h
		}
		return found
	}
	it.madinah = uniq(rahmahItMadinahRe)
	if mk := uniq(rahmahItMakkahP); mk != "" {
		it.makkah = mk
	} else {
		it.makkah = uniq(rahmahItMakkahRe)
	}
	lower := strings.ToLower(text)
	for _, l := range strings.Split(lower, "\n") {
		if !strings.Contains(l, "transit") && !strings.HasPrefix(l, "bandara ") && !strings.HasPrefix(l, "landing") {
			continue
		}
		best, bestIdx := "", -1
		for _, h := range rahmahTransitHubs {
			if i := strings.Index(l, h.key); i >= 0 && (bestIdx < 0 || i < bestIdx) {
				best, bestIdx = h.city, i
			}
		}
		if best != "" {
			it.transit = best
			break
		}
	}
	bestIdx := -1
	for _, a := range rahmahDepAirports {
		if i := strings.Index(lower, a.key); i >= 0 && (bestIdx < 0 || i < bestIdx) {
			it.airport, bestIdx = a.city, i
		}
	}
	return it
}

func (r *RahmahParser) Crawl() ([]CrawledPackage, error) {
	listURL := r.URL
	if listURL == "" {
		listURL = rahmahBase + "/jadwal-biaya-umroh-2026-2027"
	}
	base, err := url.Parse(listURL)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	c := NewCollector(base.Hostname())
	_ = c.Limit(&colly.LimitRule{DomainGlob: "*", Delay: 300 * time.Millisecond})

	// ---- 1. schedule table ----
	ldNames := make(map[string]string)
	c.OnResponse(func(resp *colly.Response) {
		if resp.Request.URL.String() != listURL {
			return
		}
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

	var rows []*rahmahRow
	rowByURL := map[string]*rahmahRow{}
	c.OnHTML("tr.schedule-row", func(e *colly.HTMLElement) {
		var cells []string
		e.ForEach("td", func(_ int, td *colly.HTMLElement) {
			cells = append(cells, rahmahSpace(td.Text))
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
		if detail == "" {
			return
		}
		pkgURL := e.Request.AbsoluteURL(detail)
		if i := strings.IndexAny(pkgURL, "?#"); i >= 0 {
			pkgURL = pkgURL[:i]
		}
		if rowByURL[pkgURL] != nil {
			return // same departure listed twice
		}
		row := &rahmahRow{url: pkgURL, dep: dep, price: ParsePrice(cells[4]), seatText: cells[5]}
		if ret, ok := rahmahParseDate(cells[1]); ok && ret.After(dep) {
			row.ret = ret
		}
		if n := rahmahFirstNumRe.FindString(cells[2]); n != "" {
			row.durasi, _ = strconv.Atoi(n)
		}
		row.airline = KnownAirline(cells[3])
		if row.airline == "" && !strings.Contains(strings.ToLower(cells[3]), "menyusul") {
			row.airline = cells[3] // "Maskapai Menyusul" = not decided yet
		}
		if n := rahmahFirstNumRe.FindString(cells[5]); n != "" && strings.Contains(strings.ToLower(cells[5]), "seat") {
			row.seats, _ = strconv.Atoi(n)
		}
		row.name = ldNames[pkgURL]
		if row.name == "" {
			if wa, err := url.Parse(e.ChildAttr("a.schedule-row-cta", "href")); err == nil {
				if m := rahmahWANameRe.FindStringSubmatch(wa.Query().Get("text")); m != nil {
					row.name = rahmahTrailDate.ReplaceAllString(m[1], "")
				}
			}
		}
		row.name = rahmahSpace(strings.ReplaceAll(row.name, "()", ""))
		if row.price <= 0 {
			return
		}
		rowByURL[pkgURL] = row
		rows = append(rows, row)
	})

	// ---- 3. direct-flight listing ----
	direct := map[string]bool{}
	c.OnHTML(`a[href^="/paket/"]`, func(e *colly.HTMLElement) {
		if !strings.Contains(e.Request.URL.Path, "paket-umroh-direct") {
			return
		}
		u := e.Request.AbsoluteURL(e.Attr("href"))
		if i := strings.IndexAny(u, "?#"); i >= 0 {
			u = u[:i]
		}
		direct[u] = true
	})

	// ---- 2. detail pages ----
	details := map[string]rahmahDetail{}
	c.OnResponse(func(resp *colly.Response) {
		if strings.HasPrefix(resp.Request.URL.Path, "/paket/") {
			u := rahmahBase + resp.Request.URL.Path
			details[u] = rahmahParseDetail(resp.Body)
		}
	})

	if err := c.Visit(listURL); err != nil {
		return nil, err
	}
	if err := c.Visit(rahmahBase + "/paket-umroh-direct"); err != nil {
		log.Printf("[crawler] rahmah direct page: %v\n", err)
	}
	for _, row := range rows {
		_ = c.Visit(row.url)
	}
	c.Wait()

	// ---- 4. itinerary site ----
	itins := r.crawlItineraries(rows, details)

	var packages []CrawledPackage
	for _, row := range rows {
		d := details[row.url]
		isDirect := direct[row.url] || d.direct
		packages = append(packages, rahmahBuild(row, d, isDirect, itins[row.url]))
	}

	// The travel sometimes opens a second group for the same departure (two pages,
	// e.g. sku 650/651, identical in every published field but the seat pool).
	// Show it once, with the seats of both groups.
	{
		type ident struct {
			name, date, ret, hm, hmd, air, cats string
			price, tri, dbl                     int
		}
		at := map[ident]int{}
		var merged []CrawledPackage
		for _, p := range packages {
			d := p.Departures[0]
			var cats []string
			for _, c := range details[p.URL].categories {
				cats = append(cats, c.name)
			}
			k := ident{p.PackageName, d.Date, d.ReturnDate, p.HotelMakkah, p.HotelMadinah, p.Airline,
				strings.Join(cats, "|"), p.Price, p.PriceTriple, p.PriceDouble}
			if i, ok := at[k]; ok {
				m := &merged[i]
				m.Seats += p.Seats
				m.Departures[0].Seats = m.Seats
				m.Full = m.Full && p.Full
				m.Departures[0].Full = m.Full
				continue
			}
			at[k] = len(merged)
			merged = append(merged, p)
		}
		packages = merged
	}

	// Same name + same date on two pages (e.g. "Kepulangan via Jeddah" vs "via
	// Madinah"): add the category label the travel uses to tell them apart.
	type nk struct{ name, date string }
	count := map[nk]int{}
	for _, p := range packages {
		count[nk{p.PackageName, p.Departures[0].Date}]++
	}
	for i := range packages {
		p := &packages[i]
		if count[nk{p.PackageName, p.Departures[0].Date}] < 2 {
			continue
		}
		cats := details[p.URL].categories
		if len(cats) == 1 {
			if i := strings.Index(cats[0].name, "("); i >= 0 && strings.HasSuffix(cats[0].name, ")") {
				p.PackageName += " " + cats[0].name[i:]
			}
		}
	}

	sort.SliceStable(packages, func(a, b int) bool {
		return packages[a].Departures[0].Date < packages[b].Departures[0].Date
	})
	return packages, nil
}

func rahmahBuild(row *rahmahRow, d rahmahDetail, isDirect bool, it rahmahItin) CrawledPackage {
	name := d.name
	if name == "" {
		name = row.name
	}
	if name == "" {
		slug := strings.TrimPrefix(row.url, rahmahBase+"/paket/")
		name = strings.Title(strings.ReplaceAll(slug, "-", " ")) //nolint:staticcheck
	}

	depISO := row.dep.Format("2006-01-02")
	retISO := ""
	if !row.ret.IsZero() {
		retISO = row.ret.Format("2006-01-02")
	}
	// The detail page's JSON-LD is authoritative when it agrees on the departure.
	if d.dep == depISO && d.ret > depISO {
		retISO = d.ret
	}

	pkg := CrawledPackage{
		TravelName:     "Rahmah Travel",
		PackageName:    name,
		Price:          row.price,
		Airline:        row.airline,
		DepartureDates: []string{row.dep.Format("2 Jan 2006")},
		Seats:          row.seats,
		URL:            row.url,
	}
	if d.sisaFound {
		pkg.Seats = d.sisa
	}
	lowerSeat := strings.ToLower(row.seatText)
	if d.soldOut || strings.Contains(lowerSeat, "sold out") || strings.Contains(lowerSeat, "penuh") {
		pkg.Full = true
		pkg.Seats = 0
	}

	// Marketed duration from the name ("9D", "12 D", "9 Hari"); the table/JSON-LD
	// "durasi" is computed from the dates and only used when the name has none.
	if m := rahmahNameDayRe.FindStringSubmatch(name); m != nil {
		pkg.Duration, _ = strconv.Atoi(m[1])
	}
	if pkg.Duration == 0 {
		pkg.Duration = d.durasi
	}
	if pkg.Duration == 0 {
		pkg.Duration = row.durasi
	}

	// Room category holding the listed price.
	var chosen *rahmahCategory
	hotelAmbiguous := false
	for i := range d.categories {
		cat := &d.categories[i]
		has, quad := false, false
		for _, rm := range cat.rooms {
			if rm.price == row.price {
				has = true
				quad = quad || rm.kind == "quad"
			}
		}
		if !has {
			continue
		}
		if chosen == nil {
			chosen = cat
			continue
		}
		// Prefer the tab where the listed price is the Quad price.
		chosenQuad := false
		for _, rm := range chosen.rooms {
			if rm.price == row.price && rm.kind == "quad" {
				chosenQuad = true
			}
		}
		if quad && !chosenQuad {
			chosen = cat
		} else if quad == chosenQuad && rahmahCategoryHotel(cat.name) != rahmahCategoryHotel(chosen.name) {
			hotelAmbiguous = true
		}
	}
	// If the listed price is in no tab, the page contradicts itself (e.g. "Harga
	// mulai Rp 36.900.000" with a Quad card of Rp 29.900.000): keep the listed
	// price and take no room prices/hotel from the tabs.
	if chosen != nil {
		for _, rm := range chosen.rooms {
			switch rm.kind {
			case "triple":
				if pkg.PriceTriple == 0 {
					pkg.PriceTriple = rm.price
				}
			case "double":
				if pkg.PriceDouble == 0 {
					pkg.PriceDouble = rm.price
				}
			}
		}
		if !hotelAmbiguous {
			pkg.HotelMakkah = rahmahCategoryHotel(chosen.name)
		}
	}
	if pkg.HotelMakkah == "" && it.makkah != "" {
		pkg.HotelMakkah = it.makkah
	}
	pkg.HotelMadinah = it.madinah
	pkg.Airport = it.airport

	switch {
	case isDirect && it.transit == "":
		t := true
		pkg.Direct = &t
	case it.transit != "" && !isDirect:
		f := false
		pkg.Direct = &f
		pkg.TransitVia = it.transit
	}

	if d.cityTour != "" {
		pkg.Facilities = append(pkg.Facilities, "City tour "+d.cityTour)
	}

	pkg.Departures = []CrawledDeparture{{
		Date:         depISO,
		ReturnDate:   retISO,
		Price:        pkg.Price,
		PriceTriple:  pkg.PriceTriple,
		PriceDouble:  pkg.PriceDouble,
		Seats:        pkg.Seats,
		Full:         pkg.Full,
		HotelMakkah:  pkg.HotelMakkah,
		HotelMadinah: pkg.HotelMadinah,
		Airline:      pkg.Airline,
		Airport:      pkg.Airport,
	}}
	return pkg
}

// crawlItineraries matches packages to batches on itinerary.rahmahtravel.id by
// (package name, departure date) and parses the matched itinerary pages.
func (r *RahmahParser) crawlItineraries(rows []*rahmahRow, details map[string]rahmahDetail) map[string]rahmahItin {
	out := map[string]rahmahItin{}
	ic := NewCollector("itinerary.rahmahtravel.id")
	_ = ic.Limit(&colly.LimitRule{DomainGlob: "*", Delay: 300 * time.Millisecond})

	type key struct{ name, date string }
	batches := map[key][]string{}
	pages := map[string]rahmahItin{}
	ic.OnResponse(func(resp *colly.Response) {
		if b := resp.Request.URL.Query().Get("batch"); b != "" {
			pages[b] = rahmahParseItinerary(resp.Body)
			return
		}
		for _, m := range rahmahItCardRe.FindAllSubmatch(resp.Body, -1) {
			dm := rahmahItCardDate.FindSubmatch(m[3])
			if dm == nil {
				continue
			}
			t, ok := rahmahParseDate(string(dm[1]))
			if !ok {
				continue
			}
			k := key{rahmahNorm(html.UnescapeString(string(m[2]))), t.Format("2006-01-02")}
			batches[k] = append(batches[k], string(m[1]))
		}
	})
	if err := ic.Visit(rahmahItineraryURL); err != nil {
		log.Printf("[crawler] rahmah itinerary index: %v\n", err)
		return out
	}

	want := map[string][]string{}
	for _, row := range rows {
		name := details[row.url].name
		if name == "" {
			name = row.name
		}
		bs := batches[key{rahmahNorm(name), row.dep.Format("2006-01-02")}]
		if len(bs) == 0 {
			continue
		}
		want[row.url] = bs
		for _, b := range bs {
			if _, done := pages[b]; !done {
				pages[b] = rahmahItin{}
				_ = ic.Visit(rahmahItineraryURL + "paket?batch=" + b)
			}
		}
	}
	ic.Wait()

	// When several batches share the name and date, keep only what they agree on.
	for u, bs := range want {
		it := pages[bs[0]]
		for _, b := range bs[1:] {
			o := pages[b]
			if o.makkah != it.makkah {
				it.makkah = ""
			}
			if o.madinah != it.madinah {
				it.madinah = ""
			}
			if o.transit != it.transit {
				it.transit = ""
			}
			if o.airport != it.airport {
				it.airport = ""
			}
		}
		out[u] = it
	}
	return out
}
