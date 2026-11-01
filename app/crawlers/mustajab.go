package crawlers

import (
	"encoding/json"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// MustajabParser scrapes PT Marwa Mustajab Wisata (El Marwa) on the
// muslimpergi platform: https://umrohmustajab.com/listings.
//
// Every listing card is one departure. Cards that are FULLY BOOKED have a
// disabled "Lihat Detail" button, but their detail page is still public and
// listed in /sitemap.xml, so it is found there by matching the package name.
// The detail page provides: JSON-LD offers ("[QUAD] SILVER", ...), Total /
// Tersisa Pax, structured hotels (name, star icons, Lokasi Makkah/Madinah),
// flight segments (route + dates incl. the arrival back in Jakarta), guide
// ("Pembimbing Jamaah"), includes and excludes.
type MustajabParser struct {
	URL        string
	TravelName string
}

type mustajabCard struct {
	name     string
	date     time.Time
	duration int
	total    int
	avail    int
	hasAvail bool
	full     bool
	airport  string
	airline  string
	price    int
	href     string
}

type mustajabHotel struct {
	name  string
	city  string // "makkah" / "madinah"
	stars int
}

type mustajabOffer struct {
	kind  string // QUAD / TRIPLE / DOUBLE / ...
	label string // tier label, normalised
	price int
}

var (
	mustajabCardDateRe = regexp.MustCompile(`(?i)^(\d{1,2})\s+(` + lstMonthAlt + `)\s+(20\d{2})$`)
	mustajabDurRe      = regexp.MustCompile(`(?i)(\d{1,2})\s*Hari`)
	mustajabNumRe      = regexp.MustCompile(`\d+`)
	mustajabOfferRe    = regexp.MustCompile(`^\s*\[([A-Za-z ]+)\]\s*(.*)$`)
	mustajabBerRe      = regexp.MustCompile(`(?i)\bber\s*\d+\b|\bnew\b`)
	mustajabFltDateRe  = regexp.MustCompile(`(?i)^(\d{1,2})\s+(` + lstMonthAlt + `)\s+(20\d{2}),\s*\d{1,2}[:.]\d{2}$`)
	mustajabIATARe     = regexp.MustCompile(`^[A-Z]{3}$`)
	mustajabCityRe     = regexp.MustCompile(`^\(([^,]+),\s*[A-Z]{2}\)$`)
	mustajabLocRe      = regexp.MustCompile(`<loc>([^<]+/listings/[^<]+)</loc>`)
	mustajabEmojiRe    = regexp.MustCompile(`^[\x{2705}\x{274C}\x{2714}\x{2716}\x{FE0F}\s]+`)
	mustajabDPRe       = regexp.MustCompile(`(?i)\b(?:DP|uang muka)\s*:?\s*Rp\.?\s*([\d.,]+)`)
)

func mustajabNorm(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

func (m *MustajabParser) Crawl() ([]CrawledPackage, error) {
	cutoff := lstCutoff()
	base, err := url.Parse(m.URL)
	if err != nil {
		return nil, err
	}
	travel := m.TravelName
	if travel == "" {
		travel = "Marwa Mustajab"
	}

	log.Println("[crawler] visiting:", m.URL)
	listHTML, err := lstFetch(m.URL)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(listHTML))
	if err != nil {
		return nil, err
	}

	var cards []mustajabCard
	doc.Find("div.box_grid").Each(func(_ int, box *goquery.Selection) {
		c := mustajabCard{name: strings.Join(strings.Fields(box.Find("h3").First().Text()), " ")}
		box.Find("p.h6").Each(func(_ int, p *goquery.Selection) {
			t := strings.Join(strings.Fields(p.Text()), " ")
			if mm := mustajabCardDateRe.FindStringSubmatch(t); mm != nil {
				d, _ := strconv.Atoi(mm[1])
				y, _ := strconv.Atoi(mm[3])
				c.date, _ = lstMakeDate(y, lstMonthNum(mm[2]), d)
			} else if mm := mustajabDurRe.FindStringSubmatch(t); mm != nil && c.duration == 0 {
				c.duration, _ = strconv.Atoi(mm[1])
			}
		})
		box.Find("table tr").Each(func(_ int, tr *goquery.Selection) {
			tds := tr.Find("td")
			if tds.Length() < 3 {
				return
			}
			label := strings.ToLower(strings.TrimSpace(tds.Eq(1).Text()))
			val := strings.Join(strings.Fields(tds.Eq(2).Text()), " ")
			switch {
			case strings.HasPrefix(label, "total pax"):
				c.total, _ = strconv.Atoi(mustajabNumRe.FindString(val))
			case strings.HasPrefix(label, "available pax"):
				if strings.Contains(strings.ToLower(val), "fully booked") || strings.Contains(strings.ToLower(val), "penuh") {
					c.full = true
				} else if n := mustajabNumRe.FindString(val); n != "" {
					c.avail, _ = strconv.Atoi(n)
					c.hasAvail = true
				}
			case strings.HasPrefix(label, "berangkat dari"):
				c.airport = val
			case strings.HasPrefix(label, "maskapai"):
				c.airline = val
			}
		})
		c.price = ParsePrice(box.Find("div.price p.font-bold").First().Text())
		if href, ok := box.Find("a.btn_1").Attr("href"); ok && strings.Contains(href, "/listings/") {
			if u, err := base.Parse(href); err == nil {
				u.RawQuery, u.Fragment = "", ""
				c.href = u.String()
			}
		}
		if c.name != "" && !c.date.IsZero() {
			cards = append(cards, c)
		}
	})

	// Detail pages: linked ones first; then sitemap pages for linkless cards.
	details := map[string]string{} // url -> html
	byName := map[string]string{}  // normalised JSON-LD name -> url
	fetchDetail := func(u string) {
		if _, ok := details[u]; ok {
			return
		}
		log.Println("[crawler] visiting:", u)
		h, err := lstFetch(u)
		time.Sleep(lstPoliteDelay)
		if err != nil {
			log.Printf("[mustajab] %s: %v", u, err)
			details[u] = ""
			return
		}
		details[u] = h
		if n := mustajabProductName(h); n != "" {
			byName[mustajabNorm(n)] = u
		}
	}
	missing := 0
	for _, c := range cards {
		if c.date.Before(cutoff) {
			continue
		}
		if c.href != "" {
			fetchDetail(c.href)
		} else {
			missing++
		}
	}
	if missing > 0 {
		sm, err := lstFetch(base.Scheme + "://" + base.Host + "/sitemap.xml")
		if err == nil {
			for _, mm := range mustajabLocRe.FindAllStringSubmatch(sm, -1) {
				u := strings.TrimSpace(mm[1])
				if strings.Contains(u, "haji") || strings.Contains(u, "-haj-") {
					continue
				}
				fetchDetail(u)
			}
		} else {
			log.Printf("[mustajab] sitemap: %v", err)
		}
	}

	var packages []CrawledPackage
	seen := map[string]bool{}
	for _, c := range cards {
		if c.date.Before(cutoff) {
			continue
		}
		u := c.href
		if u == "" {
			u = byName[mustajabNorm(c.name)]
		}
		if u == "" {
			log.Printf("[mustajab] %q: halaman detail tidak ditemukan, dilewati", c.name)
			continue
		}
		key := u + "|" + c.date.Format("2006-01-02")
		if seen[key] {
			continue
		}
		seen[key] = true
		pkg := m.buildPackage(c, u, details[u], travel)
		if pkg.Price < lstMinPrice || pkg.Price > lstMaxPrice {
			continue
		}
		packages = append(packages, pkg)
	}
	return packages, nil
}

func mustajabLDProduct(html string) map[string]interface{} {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil
	}
	var product map[string]interface{}
	doc.Find(`script[type="application/ld+json"]`).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		var root map[string]interface{}
		if json.Unmarshal([]byte(s.Text()), &root) != nil {
			return true
		}
		nodes := []interface{}{root}
		if g, ok := root["@graph"].([]interface{}); ok {
			nodes = g
		}
		for _, n := range nodes {
			if mm, ok := n.(map[string]interface{}); ok && mm["@type"] == "Product" {
				product = mm
				return false
			}
		}
		return true
	})
	return product
}

func mustajabProductName(html string) string {
	if p := mustajabLDProduct(html); p != nil {
		if n, ok := p["name"].(string); ok {
			return n
		}
	}
	return ""
}

func (m *MustajabParser) buildPackage(c mustajabCard, u, html, travel string) CrawledPackage {
	pkg := CrawledPackage{
		TravelName:  travel,
		PackageName: c.name,
		URL:         u,
		Duration:    c.duration,
		Price:       c.price,
		Airport:     mustajabTitle(c.airport),
	}
	if a := KnownAirline(c.airline); a != "" {
		pkg.Airline = a
	} else if c.airline != "" {
		pkg.Airline = strings.TrimSpace(strings.TrimSuffix(c.airline, " Airlines"))
		if strings.EqualFold(pkg.Airline, "oman") {
			pkg.Airline = "Oman Air"
		}
	}
	dep := CrawledDeparture{Date: c.date.Format("2006-01-02")}
	if c.total > 0 {
		pkg.SeatsTotal, dep.SeatsTotal = c.total, c.total
	}
	if c.full {
		pkg.Full, dep.Full = true, true
	} else if c.hasAvail {
		if c.avail <= 0 {
			pkg.Full, dep.Full = true, true
		} else {
			pkg.Seats, dep.Seats = c.avail, c.avail
		}
	}

	if html != "" {
		m.fillFromDetail(&pkg, &dep, c, html)
	}
	pkg.Departures = []CrawledDeparture{dep}
	pkg.DepartureDates = []string{lstFormatDate(c.date)}
	return pkg
}

func (m *MustajabParser) fillFromDetail(pkg *CrawledPackage, dep *CrawledDeparture, c mustajabCard, html string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return
	}

	// Prices from JSON-LD offers.
	var offers []mustajabOffer
	if p := mustajabLDProduct(html); p != nil {
		if list, ok := p["offers"].([]interface{}); ok {
			for _, o := range list {
				om, ok := o.(map[string]interface{})
				if !ok {
					continue
				}
				name, _ := om["name"].(string)
				mm := mustajabOfferRe.FindStringSubmatch(name)
				if mm == nil {
					continue
				}
				price := 0
				switch v := om["price"].(type) {
				case string:
					price, _ = strconv.Atoi(strings.Split(v, ".")[0])
				case float64:
					price = int(v)
				}
				label := mustajabBerRe.ReplaceAllString(mm[2], "")
				offers = append(offers, mustajabOffer{
					kind:  strings.ToUpper(strings.TrimSpace(mm[1])),
					label: mustajabNorm(label),
					price: price,
				})
			}
		}
	}
	tier := ""
	quad := 0
	for _, o := range offers {
		if o.kind == "QUAD" && o.price > 0 && !strings.Contains(o.label, "upgrade") && (quad == 0 || o.price < quad) {
			quad, tier = o.price, o.label
		}
	}
	pick := func(kind string) int {
		var same, plain []int
		for _, o := range offers {
			if o.kind != kind || o.price <= 0 || strings.Contains(o.label, "upgrade") {
				continue
			}
			if o.label == tier {
				same = append(same, o.price)
			}
			plain = append(plain, o.price)
		}
		if len(same) == 1 {
			return same[0]
		}
		if len(plain) == 1 {
			return plain[0]
		}
		return 0
	}
	if quad > 0 {
		pkg.Price = quad
		pkg.PriceTriple = pick("TRIPLE")
		pkg.PriceDouble = pick("DOUBLE")
	}

	// Seats from the detail page (card values win when present).
	doc.Find("div.box_content").Each(func(_ int, b *goquery.Selection) {
		label := strings.ToLower(strings.TrimSpace(b.Find("h3").Text()))
		n, err := strconv.Atoi(strings.TrimSpace(b.Find("p").First().Text()))
		if err != nil {
			return
		}
		switch label {
		case "total pax":
			if pkg.SeatsTotal == 0 && n > 0 {
				pkg.SeatsTotal, dep.SeatsTotal = n, n
			}
		case "tersisa pax":
			if !pkg.Full && pkg.Seats == 0 {
				if n <= 0 {
					pkg.Full, dep.Full = true, true
				} else {
					pkg.Seats, dep.Seats = n, n
				}
			}
		}
	})

	// Hotels.
	var hotels []mustajabHotel
	doc.Find("div.hotel-info-item").Each(func(_ int, it *goquery.Selection) {
		h := mustajabHotel{name: strings.Join(strings.Fields(it.Find("h4").First().Text()), " ")}
		h.stars = it.Find(".hotel-info-rate i.fa-star").Not(".empty").Length()
		if h.stars > 5 {
			h.stars = 0
		}
		it.Find("div.address h5").Each(func(_ int, h5 *goquery.Selection) {
			if strings.Contains(strings.ToLower(h5.Text()), "lokasi") {
				loc := strings.ToLower(strings.TrimSpace(h5.Parent().Find("p").First().Text()))
				switch {
				case strings.Contains(loc, "mak") || strings.Contains(loc, "mek"):
					h.city = "makkah"
				case strings.Contains(loc, "madin"):
					h.city = "madinah"
				}
			}
		})
		if h.name != "" && h.city != "" {
			hotels = append(hotels, h)
		}
	})
	descLines := lstHTMLLines(mustajabOuter(doc.Find("div.st-main-description").First()))
	for _, city := range []string{"makkah", "madinah"} {
		var cand []mustajabHotel
		for _, h := range hotels {
			if h.city == city {
				cand = append(cand, h)
			}
		}
		chosen := cand
		if len(cand) > 1 {
			chosen = mustajabTierHotels(cand, descLines, tier, city, offers)
		}
		if len(cand) == 0 {
			// No structured hotel: use the description's own
			// "Hotel Makkah Silver :\n<name>/Setaraf***" line for the tier.
			if h, ok := mustajabDescHotel(descLines, city, tier); ok {
				chosen = []mustajabHotel{h}
			}
		}
		if len(chosen) == 0 {
			continue
		}
		var names []string
		stars := chosen[0].stars
		for _, h := range chosen {
			names = append(names, h.name)
			if h.stars != stars {
				stars = 0
			}
		}
		name := strings.Join(names, " / ")
		if city == "makkah" {
			pkg.HotelMakkah, pkg.StarsMakkah = name, stars
		} else {
			pkg.HotelMadinah, pkg.StarsMadinah = name, stars
		}
	}

	// Guide, includes, excludes.
	var guides []string
	doc.Find("ul.st-guide-list li").Each(func(_ int, li *goquery.Selection) {
		li.Find("span").Remove()
		if t := strings.Join(strings.Fields(li.Text()), " "); t != "" {
			guides = append(guides, t)
		}
	})
	pkg.Guide = strings.Join(guides, ", ")
	collect := func(sel string) []string {
		var out []string
		doc.Find(sel).Each(func(_ int, li *goquery.Selection) {
			li.Find("span").Remove()
			t := strings.Join(strings.Fields(li.Text()), " ")
			t = strings.TrimSpace(mustajabEmojiRe.ReplaceAllString(t, ""))
			if t != "" {
				out = append(out, t)
			}
		})
		return out
	}
	pkg.Facilities = collect("ul.st-include-list li")
	pkg.Excludes = collect("ul.st-exclude-list li")
	if mm := mustajabDPRe.FindStringSubmatch(strings.Join(descLines, " ")); mm != nil {
		pkg.DownPayment = ParsePrice(mm[1])
	}

	// Flights: "Departure"/"Arrival" segments, each with two (IATA, city, datetime) points.
	lines := lstHTMLLines(html)
	type point struct {
		code, city string
		date       time.Time
	}
	type segment struct {
		kind   string
		points []point
	}
	var segs []segment
	for i := 0; i < len(lines); i++ {
		if lines[i] != "Departure" && lines[i] != "Arrival" {
			continue
		}
		seg := segment{kind: lines[i]}
		for j := i + 1; j < len(lines) && j < i+25 && len(seg.points) < 2; j++ {
			if lines[j] == "Departure" || lines[j] == "Arrival" || lines[j] == "Transportasi" {
				break
			}
			if mustajabIATARe.MatchString(lines[j]) && j+3 < len(lines) {
				p := point{code: lines[j]}
				for k := j + 1; k <= j+3 && k < len(lines); k++ {
					if mm := mustajabCityRe.FindStringSubmatch(lines[k]); mm != nil {
						p.city = strings.TrimSpace(mm[1])
					}
					if mm := mustajabFltDateRe.FindStringSubmatch(lines[k]); mm != nil {
						d, _ := strconv.Atoi(mm[1])
						y, _ := strconv.Atoi(mm[3])
						p.date, _ = lstMakeDate(y, lstMonthNum(mm[2]), d)
					}
				}
				if !p.date.IsZero() {
					seg.points = append(seg.points, p)
				}
			}
		}
		if len(seg.points) == 2 {
			segs = append(segs, seg)
		}
	}
	if len(segs) > 0 {
		first := segs[0]
		if first.kind == "Departure" {
			to := first.points[1].code
			direct := to == "JED" || to == "MED"
			pkg.Direct = &direct
			if !direct {
				pkg.TransitVia = first.points[1].city
				if pkg.TransitVia == "" {
					pkg.TransitVia = to
				}
			}
		}
		last := segs[len(segs)-1]
		if last.kind == "Arrival" {
			r := last.points[1].date
			if !r.Before(c.date) {
				dep.ReturnDate = r.Format("2006-01-02")
			}
		}
	}
}

// mustajabTierHotels picks, among several structured hotels in one city, the
// one(s) named in the description block of the cheapest tier (e.g. "SILVER").
// Returns nil when that cannot be decided from the text.
func mustajabTierHotels(cand []mustajabHotel, desc []string, tier, city string, offers []mustajabOffer) []mustajabHotel {
	if tier == "" || len(desc) == 0 {
		return nil
	}
	labels := map[string]bool{}
	for _, o := range offers {
		if o.label != "" {
			labels[o.label] = true
		}
	}
	isLabel := func(l string) (string, bool) {
		n := mustajabNorm(strings.TrimSuffix(strings.TrimSpace(l), ":"))
		n = strings.TrimSpace(strings.TrimSuffix(n, ":"))
		return n, labels[n]
	}
	// Start at the hotel section.
	start := -1
	for i, l := range desc {
		if strings.HasPrefix(strings.ToLower(l), "hotel") {
			start = i
			break
		}
	}
	if start < 0 {
		return nil
	}
	var seg []string
	in := false
	for _, l := range desc[start:] {
		if n, ok := isLabel(l); ok {
			in = n == tier
			continue
		}
		if strings.HasPrefix(strings.ToLower(l), "pembimbing") || strings.HasPrefix(strings.ToLower(l), "harga") {
			if in {
				break
			}
		}
		if in {
			seg = append(seg, l)
		}
	}
	text := strings.ToLower(strings.Join(seg, " "))
	// Keep only the part about this city.
	mk := strings.Index(text, "makkah")
	if mk < 0 {
		mk = strings.Index(text, "mekkah")
	}
	md := strings.Index(text, "madinah")
	if mk < 0 || md < 0 {
		return nil
	}
	if city == "makkah" {
		if md > mk {
			text = text[mk:md]
		} else {
			text = text[mk:]
		}
	} else {
		if mk > md {
			text = text[md:mk]
		} else {
			text = text[md:]
		}
	}
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	var out []mustajabHotel
	for _, h := range cand {
		if mustajabNameInText(h.name, words) {
			out = append(out, h)
		}
	}
	return out
}

var mustajabStop = map[string]bool{
	"hotel": true, "hotels": true, "tower": true, "towers": true, "makkah": true, "madinah": true,
	"madina": true, "grand": true, "golden": true, "setaraf": true, "the": true, "al": true, "and": true,
}

// mustajabNameInText: a significant word of the hotel name is (nearly) present in the text.
func mustajabNameInText(name string, words []string) bool {
	for _, w := range strings.Fields(strings.ToLower(name)) {
		w = strings.Trim(w, "()-.,/")
		if len(w) < 4 || mustajabStop[w] {
			continue
		}
		for _, t := range words {
			if len(t) >= 4 && !mustajabStop[t] && mustajabEditLE1(w, t) {
				return true
			}
		}
	}
	return false
}

func mustajabEditLE1(a, b string) bool {
	if a == b {
		return true
	}
	la, lb := len(a), len(b)
	if la-lb > 1 || lb-la > 1 {
		return false
	}
	i, j, diff := 0, 0, 0
	for i < la && j < lb {
		if a[i] == b[j] {
			i++
			j++
			continue
		}
		diff++
		if diff > 1 {
			return false
		}
		switch {
		case la > lb:
			i++
		case lb > la:
			j++
		default:
			i++
			j++
		}
	}
	return diff+(la-i)+(lb-j) <= 1
}

func mustajabOuter(s *goquery.Selection) string {
	h, err := goquery.OuterHtml(s)
	if err != nil {
		return ""
	}
	return h
}

func mustajabTitle(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

var (
	mustajabDescHotelRe = regexp.MustCompile(`(?i)^hotel\s+(makkah|mekkah|madinah)\s*([a-z ]*?)\s*:\s*(.*)$`)
	mustajabDescDistRe  = regexp.MustCompile(`(?i)^[\d\s.,±-]+(m|km)$`)
	mustajabStarsTail   = regexp.MustCompile(`\*+\s*$`)
)

// mustajabDescHotel reads "Hotel Makkah Silver : 350-500m" followed by
// "Manazel Ajyad/Tara Ajyad/Setaraf***" from the description, for the given
// city and (cheapest) tier. Stars only when written as "***".
func mustajabDescHotel(desc []string, city, tier string) (mustajabHotel, bool) {
	for i, l := range desc {
		m := mustajabDescHotelRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		c := strings.ToLower(m[1])
		if c == "mekkah" {
			c = "makkah"
		}
		if c != city || (m[2] != "" && mustajabNorm(m[2]) != tier) || (m[2] == "" && tier != "") {
			continue
		}
		val := strings.TrimSpace(m[3])
		if (val == "" || mustajabDescDistRe.MatchString(val)) && i+1 < len(desc) {
			val = strings.TrimSpace(desc[i+1])
		}
		h := mustajabHotel{city: city}
		if st := mustajabStarsTail.FindString(val); st != "" {
			h.stars = strings.Count(st, "*")
			if h.stars > 5 {
				h.stars = 0
			}
			val = mustajabStarsTail.ReplaceAllString(val, "")
		}
		val = regexp.MustCompile(`(?i)\s*/?\s*setaraf\s*$`).ReplaceAllString(strings.TrimSpace(val), "")
		h.name = strings.Trim(strings.TrimSpace(val), "/ ")
		if h.name == "" || strings.HasPrefix(strings.ToLower(h.name), "hotel ") {
			return mustajabHotel{}, false
		}
		return h, true
	}
	return mustajabHotel{}, false
}
