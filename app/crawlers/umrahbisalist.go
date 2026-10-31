package crawlers

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// UmrahBisaListParser discovers all current Umrah Bisa products (Berdu shop at
// daftar.umrahbisa.com) from the sitemap, the home page and the "all products"
// page, then scrapes every product page.
//
// Products whose description still uses the old "Jadwal: / Harga Spesial:"
// template are parsed with the existing UmrahBisaProductParser. Newer products
// have free-form descriptions, so they are parsed here: departure date ranges
// ("26 Desember 2026 – 3 Januari 2027", "16-24 Maret" + "Maret 2027", ...) and
// the package price ("Rp 25,9 Juta", "Rp 25.900.000"), ignoring DP / add-on
// amounts. The shop's listed product price (Rp 5.000.000) is the DP and is
// never used.
type UmrahBisaListParser struct {
	URL string // shop root, e.g. https://daftar.umrahbisa.com/
}

func NewUmrahBisaListParser(shopURL string) *UmrahBisaListParser {
	return &UmrahBisaListParser{URL: shopURL}
}

var ubListProductRe = regexp.MustCompile(`https://daftar\.umrahbisa\.com/product/([A-Za-z0-9][A-Za-z0-9_-]*)`)

func (u *UmrahBisaListParser) Crawl() ([]CrawledPackage, error) {
	root := u.URL
	if root == "" {
		root = "https://daftar.umrahbisa.com/"
	}
	base, err := url.Parse(root)
	if err != nil {
		return nil, err
	}
	abs := func(p string) string { r, _ := base.Parse(p); return r.String() }

	// 1. Discover product URLs.
	seen := map[string]bool{}
	var productURLs []string
	sources := []string{abs("/sitemap.xml"), root, abs("/product/category/")}
	okSources := 0
	for i, src := range sources {
		if i > 0 {
			time.Sleep(lstPoliteDelay)
		}
		log.Println("[crawler] visiting:", src)
		body, err := lstFetch(src)
		if err != nil {
			log.Printf("[crawler] error on %s: %v\n", src, err)
			continue
		}
		okSources++
		for _, m := range ubListProductRe.FindAllStringSubmatch(body, -1) {
			if m[1] == "category" {
				continue
			}
			pu := "https://daftar.umrahbisa.com/product/" + m[1]
			if !seen[pu] {
				seen[pu] = true
				productURLs = append(productURLs, pu)
			}
		}
	}
	if okSources == 0 {
		return nil, fmt.Errorf("umrahbisa: no listing source reachable")
	}
	sort.Strings(productURLs)
	log.Printf("[crawler] daftar.umrahbisa.com: %d product links discovered\n", len(productURLs))

	// 2. Scrape each product.
	cutoff := lstCutoff()
	var result []CrawledPackage
	for i, pu := range productURLs {
		if i > 0 {
			time.Sleep(lstPoliteDelay)
		}
		html, err := lstFetch(pu)
		if err != nil {
			log.Printf("[crawler] error on %s: %v\n", pu, err)
			continue
		}
		title := lstOgTitle(html)
		lt := strings.ToLower(title + " " + pu)
		if (!strings.Contains(lt, "umrah") && !strings.Contains(lt, "umroh")) ||
			strings.Contains(lt, "haji") || strings.Contains(lt, "pelatihan") {
			continue
		}

		var pkgs []CrawledPackage
		lowerHTML := strings.ToLower(html)
		if strings.Contains(lowerHTML, "jadwal:") && strings.Contains(lowerHTML, "harga spesial:") {
			// Old template: reuse the existing per-product parser.
			old, err := (&UmrahBisaProductParser{URL: pu}).Crawl()
			if err == nil {
				for _, p := range old {
					if lstKeepPackage(&p, cutoff) {
						pkgs = append(pkgs, p)
					}
				}
			}
		}
		if len(pkgs) == 0 {
			for _, p := range parseUmrahBisaFreeform(html, title, pu) {
				if lstKeepPackage(&p, cutoff) {
					pkgs = append(pkgs, p)
				}
			}
		}
		for _, p := range pkgs {
			p.TravelName = "Umrah Bisa"
			p.URL = pu
			result = append(result, p)
		}
	}
	return result, nil
}

// ---- free-form product page parsing ----

var (
	ubRangeRe = regexp.MustCompile(`(?i)\b(\d{1,2})\s*(?:(` + lstMonthAlt + `)\.?\s*(20\d{2})?\s*)?` +
		`(?:-|–|—|s/d|sd|sampai|hingga)\s*(\d{1,2})\s*(` + lstMonthAlt + `)\b\.?\s*(20\d{2})?`)
	ubMonthYearRe = regexp.MustCompile(`(?i)\b(` + lstMonthAlt + `)\b\.?\s+(20\d{2})`)
	ubYearRe      = regexp.MustCompile(`20(2[6-9])`)
	ubJutaRe      = regexp.MustCompile(`(?i)(rp\.?|idr)?\s*(\d{1,3}(?:[.,]\d{1,3})?)\s*(juta|jt)(an)?\b`)
	ubFullRe      = regexp.MustCompile(`(?i)(?:rp\.?|idr)\s*(\d{1,3}(?:\.\d{3}){2,})`)
	ubExcludeRe   = regexp.MustCompile(`(?i)\bdp\b|uang muka|down payment|\+|tambah|double|triple|quad|normal|dipotong|potongan|pembatalan|upgrade|cashback|diskon|admin`)
	ubKeywordRe   = regexp.MustCompile(`(?i)harga|biaya|mulai|hanya|cuma|promo|spesial|total|paket`)
	ubHariRe      = regexp.MustCompile(`(?i)(\d{1,2})\s*(?:hari|days?|d\b)`)
	ubSeatRe      = regexp.MustCompile(`(?i)tersedia\s+(\d{1,4})\s*seat`)
	ubHotelCityRe = regexp.MustCompile(`[Hh]otel\s+([A-Z][\w'’&. ]{2,40}?)\s*(?:atau setara\s*|/\s*setara\s*)?\(((?i)makkah|mekkah|mekah|madinah)\)`)
	ubAndCityRe   = regexp.MustCompile(`(?:&|\bdan)\s+([A-Z][\w'’&. ]{2,40}?)\s*(?:atau setara\s*|/\s*setara\s*)?\(((?i)makkah|mekkah|mekah|madinah)\)`)
	ubCityLineRe  = regexp.MustCompile(`(?i)^(?:hotel\s+)?(makkah|mekkah|mekah|madinah)\s*:\s*(.*)$`)
)

type ubPriceHit struct {
	pos, end int
	value    int
	keyword  bool
}

// ubFindPrices returns the plausible package prices in s, skipping DP,
// room-upgrade and cancellation amounts.
func ubFindPrices(s string) []ubPriceHit {
	var hits []ubPriceHit
	add := func(start, end, value int) {
		if value <= lstMinPrice || value > lstMaxPrice {
			return
		}
		pre := s[max(0, start-22):start]
		// Only look back to the end of the previous clause / amount.
		if k := strings.LastIndexAny(pre, ")!|•;"); k >= 0 {
			pre = pre[k+1:]
		}
		if k := strings.LastIndex(strings.ToLower(pre), "juta"); k >= 0 {
			pre = pre[k+4:]
		}
		if ubExcludeRe.MatchString(pre) {
			return
		}
		kw := ubKeywordRe.MatchString(s[max(0, start-45):start])
		hits = append(hits, ubPriceHit{pos: start, end: end, value: value, keyword: kw})
	}
	for _, m := range ubJutaRe.FindAllStringSubmatchIndex(s, -1) {
		hasRp := m[2] >= 0
		isJutaan := m[8] >= 0
		if isJutaan && !hasRp {
			continue // "Umrah 26 Jutaan" is marketing copy, not a price
		}
		num := strings.ReplaceAll(s[m[4]:m[5]], ",", ".")
		f, err := strconv.ParseFloat(num, 64)
		if err != nil {
			continue
		}
		add(m[0], m[1], int(f*1000000+0.5))
	}
	for _, m := range ubFullRe.FindAllStringSubmatchIndex(s, -1) {
		v, _ := strconv.Atoi(strings.ReplaceAll(s[m[2]:m[3]], ".", ""))
		add(m[0], m[1], v)
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].pos < hits[j].pos })
	return hits
}

type ubRange struct {
	start    time.Time
	duration int
	pos, end int
}

// ubFindRanges extracts departure date ranges. A missing year is taken from
// the nearest preceding "<Month> <Year>" mention, else from the title.
func ubFindRanges(s, title string) []ubRange {
	titleYear := 0
	if m := ubYearRe.FindString(title); m != "" {
		titleYear, _ = strconv.Atoi(m)
	}
	var out []ubRange
	for _, m := range ubRangeRe.FindAllStringSubmatchIndex(s, -1) {
		get := func(i int) string {
			if m[2*i] < 0 {
				return ""
			}
			return s[m[2*i]:m[2*i+1]]
		}
		sd, _ := strconv.Atoi(get(1))
		ed, _ := strconv.Atoi(get(4))
		em := lstMonthNum(get(5))
		sm := em
		if get(2) != "" {
			sm = lstMonthNum(get(2))
		}
		ey, _ := strconv.Atoi(get(6))
		sy, _ := strconv.Atoi(get(3))
		if ey == 0 {
			ey = sy
		}
		if ey == 0 {
			// nearest preceding "<Month> <Year>" with the same month, else any year mention
			for _, my := range ubMonthYearRe.FindAllStringSubmatchIndex(s[:m[0]], -1) {
				if lstMonthNum(s[my[2]:my[3]]) == em {
					ey, _ = strconv.Atoi(s[my[4]:my[5]])
				}
			}
			if ey == 0 {
				ey = titleYear
			}
		}
		if ey == 0 {
			continue
		}
		if sy == 0 {
			sy = ey
			if sm > em {
				sy = ey - 1
			}
		}
		st, ok1 := lstMakeDate(sy, sm, sd)
		en, ok2 := lstMakeDate(ey, em, ed)
		if !ok1 || !ok2 {
			continue
		}
		dur := int(en.Sub(st).Hours()/24) + 1
		if dur < 5 || dur > 45 {
			continue // not a trip (e.g. a registration window)
		}
		out = append(out, ubRange{start: st, duration: dur, pos: m[0], end: m[1]})
	}
	return out
}

func ubCleanHotel(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(strings.TrimPrefix(v, "Hotel "), "hotel ")
	for _, sep := range []string{" / ", "/setara", "/ setara", "/Setara", "(", " — ", " – ", " - ", " atau ", ",", "⭐"} {
		if i := strings.Index(v, sep); i > 0 {
			v = v[:i]
		}
	}
	v = strings.TrimSpace(strings.Trim(v, " -:|•"))
	lv := strings.ToLower(v)
	if len(v) < 3 || len(v) > 50 || strings.Contains(lv, "malam") || strings.Contains(lv, "km") {
		return ""
	}
	return v
}

func parseUmrahBisaFreeform(html, title, pageURL string) []CrawledPackage {
	lines := lstHTMLLines(html)
	// The description sits between the cart button and the "Produk Lainnya" block.
	from, to := 0, len(lines)
	for i, l := range lines {
		if from == 0 && strings.EqualFold(l, "Tambah ke Keranjang") {
			from = i + 1
		}
		if strings.EqualFold(l, "Produk Lainnya") && i > from {
			to = i
			break
		}
	}
	desc := lines[from:to]
	flat := strings.Join(desc, " ")

	ranges := ubFindRanges(flat, title)
	if len(ranges) == 0 && title != "" {
		ranges = ubFindRanges(title, title)
	}
	if len(ranges) == 0 {
		return nil
	}

	prices := ubFindPrices(flat)
	pagePrice := 0
	for _, h := range prices {
		if h.keyword {
			pagePrice = h.value
			break
		}
	}
	if pagePrice == 0 && len(prices) > 0 {
		pagePrice = prices[0].value
	}

	// Price written next to a specific date range (multi-schedule pages).
	rangePrice := make([]int, len(ranges))
	anyRangePrice := false
	for i, r := range ranges {
		limit := min(len(flat), r.end+150)
		if i+1 < len(ranges) && ranges[i+1].pos < limit {
			limit = ranges[i+1].pos
		}
		for _, h := range prices {
			if h.pos >= r.end && h.end <= limit {
				rangePrice[i] = h.value
				anyRangePrice = true
				break
			}
		}
	}

	titleDur := 0
	if m := ubHariRe.FindStringSubmatch(title); m != nil {
		titleDur, _ = strconv.Atoi(m[1])
	}

	// Shared fields.
	tmpl := CrawledPackage{PackageName: title, URL: pageURL}
	if tmpl.PackageName == "" && len(lines) > 0 {
		tmpl.PackageName = lines[0]
	}
	if a := KnownAirline(title); a != "" {
		tmpl.Airline = a
	} else {
		for _, l := range desc {
			if a := KnownAirline(l); a != "" {
				tmpl.Airline = a
				break
			}
		}
	}
	if tmpl.Airline == "" && regexp.MustCompile(`(?i)hainan\s+airlines?|by\s+hainan`).MatchString(title+" "+flat) {
		tmpl.Airline = "Hainan Airlines"
	}
	lowerFlat := strings.ToLower(flat)
	switch {
	case regexp.MustCompile(`(?i)(dari|keberangkatan|berangkat|start(ing)?)\s+(kota\s+)?surabaya`).MatchString(flat):
		tmpl.Airport = "SUB"
	case strings.Contains(lowerFlat, "jakarta"):
		tmpl.Airport = "CGK"
	}
	// "Tersedia N seat" (but not "Total tersedia N seat", which spans all departures).
	seatText := title + " " + flat
	if m := ubSeatRe.FindStringSubmatchIndex(seatText); m != nil &&
		!strings.Contains(strings.ToLower(seatText[max(0, m[0]-8):m[0]]), "total") {
		tmpl.Seats, _ = strconv.Atoi(seatText[m[2]:m[3]])
	}
	setHotel := func(city, name string) {
		name = ubCleanHotel(name)
		if name == "" {
			return
		}
		if strings.EqualFold(city, "madinah") {
			if tmpl.HotelMadinah == "" {
				tmpl.HotelMadinah = name
			}
		} else if tmpl.HotelMakkah == "" {
			tmpl.HotelMakkah = name
		}
	}
	for _, re := range []*regexp.Regexp{ubHotelCityRe, ubAndCityRe} {
		for _, m := range re.FindAllStringSubmatch(flat, -1) {
			setHotel(m[2], m[1])
		}
	}
	for i, l := range desc {
		if m := ubCityLineRe.FindStringSubmatch(l); m != nil {
			v := m[2]
			if strings.TrimSpace(v) == "" && i+1 < len(desc) {
				v = desc[i+1]
			}
			setHotel(m[1], v)
		}
	}

	// Airline written right before a specific range ("21 Hari — Hainan | 23 Feb–15 Mar 2027").
	rangeAirline := make([]string, len(ranges))
	for i, r := range ranges {
		from := max(0, r.pos-40)
		if i > 0 && ranges[i-1].end > from {
			from = ranges[i-1].end
		}
		seg := flat[from:r.pos]
		if a := KnownAirline(seg); a != "" {
			rangeAirline[i] = a
		} else if strings.Contains(strings.ToLower(seg), "hainan") {
			rangeAirline[i] = "Hainan Airlines"
		}
	}

	// Group ranges by (price, duration, airline).
	type group struct {
		price, dur int
		airline    string
		dates      []string
	}
	var groups []*group
	index := map[string]*group{}
	for i, r := range ranges {
		price := pagePrice
		if anyRangePrice && rangePrice[i] > 0 {
			price = rangePrice[i]
		}
		dur := r.duration
		if !anyRangePrice && titleDur > 0 {
			dur = titleDur
		}
		airline := tmpl.Airline
		if rangeAirline[i] != "" {
			airline = rangeAirline[i]
		}
		key := fmt.Sprintf("%d|%d|%s", price, dur, airline)
		g, ok := index[key]
		if !ok {
			g = &group{price: price, dur: dur, airline: airline}
			index[key] = g
			groups = append(groups, g)
		}
		g.dates = append(g.dates, lstFormatDate(r.start))
	}

	var out []CrawledPackage
	for _, g := range groups {
		if g.price <= lstMinPrice {
			continue
		}
		p := tmpl
		p.Price = g.price
		p.Duration = g.dur
		p.Airline = g.airline
		p.DepartureDates = g.dates
		out = append(out, p)
	}
	return out
}
