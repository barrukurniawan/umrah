package crawlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// UmrahBisaListParser crawls Umrah Bisa (PT Alzam Mandiri Wisata), whose
// packages are products of a Berdu shop at daftar.umrahbisa.com. Product URLs
// come from the sitemap, the home page and the "all products" page.
//
// The shop price of every product (Rp 5.000.000) is the DP, not the package
// price. Everything else is free-form description text written by the travel
// (read from the product JSON embedded in the page, which keeps the line
// structure). From it the parser reads, only where written:
//   - departure ranges ("4 - 13 Januari 2027", "16-24 Maret | Pembimbing: ... |
//     Harga: Rp 26,9 Jt", "21 Hari — Hainan | 23 Feb–15 Mar 2027"): the range
//     end is the stated return date; price / guide / airline / "N Hari" written
//     on the same line belong to that departure
//   - package price, "N Hari", airline, direct flight, departure city
//   - hotels ("Makkah: Owais / Setara (Jalan kaki ±500m)", "Hotel Abraj Al
//     Qaswa (Makkah)", "Hotel Makkah (Grand Al Massa ⭐️4)") with stars/distance
//   - room upgrades ("Double: +Rp 3 Juta") -> double/triple price = price + upgrade
//   - DP, guide ("Pembimbing: ..."), include / exclude lists, seat quota.
//
// One package is emitted per product page (URL = product page) with one
// Departures entry per date. Products whose text names no date (the schedule
// is only in the flyer image) are skipped.
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
		body, err := lstFetchPolite(src)
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
		html, err := lstFetchPolite(pu)
		if err != nil {
			log.Printf("[crawler] error on %s: %v\n", pu, err)
			continue
		}
		prod := ubProductData(html)
		title := prod.name
		if title == "" {
			title = lstOgTitle(html)
		}
		lt := strings.ToLower(title + " " + pu)
		if (!strings.Contains(lt, "umrah") && !strings.Contains(lt, "umroh")) ||
			strings.Contains(lt, "haji") || strings.Contains(lt, "pelatihan") {
			continue
		}
		lines := prod.lines
		if len(lines) == 0 {
			lines = ubHTMLDescLines(html)
		}

		p, ok := parseUmrahBisaProduct(title, prod.category, lines, prod.shopPrice, pu)
		if !ok {
			// Old "Jadwal: / Harga Spesial:" template: reuse the old parser.
			lowerHTML := strings.ToLower(html)
			if strings.Contains(lowerHTML, "jadwal:") && strings.Contains(lowerHTML, "harga spesial:") {
				if old, err := (&UmrahBisaProductParser{URL: pu}).Crawl(); err == nil && len(old) > 0 {
					p, ok = ubFromLegacy(old[0])
				}
			}
		}
		if !ok {
			continue
		}
		p.TravelName = "Umrah Bisa"
		p.URL = pu
		if lstFinalize(&p, cutoff) {
			result = append(result, p)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Departures[0].Date != result[j].Departures[0].Date {
			return result[i].Departures[0].Date < result[j].Departures[0].Date
		}
		return result[i].URL < result[j].URL
	})
	return result, nil
}

// ubFromLegacy converts an old-template package (DepartureDates only).
func ubFromLegacy(old CrawledPackage) (CrawledPackage, bool) {
	p := old
	p.Departures = nil
	for _, ds := range old.DepartureDates {
		t, ok := lstParseDate(ds)
		if !ok {
			continue
		}
		d := CrawledDeparture{Date: t.Format(lstISO)}
		if p.Duration > 1 {
			d.ReturnDate = t.AddDate(0, 0, p.Duration-1).Format(lstISO)
		}
		p.Departures = append(p.Departures, d)
	}
	// The old parser copies one "Hotel:" line into both cities; that is not
	// a per-city statement.
	if p.HotelMakkah == p.HotelMadinah {
		p.HotelMakkah, p.HotelMadinah = "", ""
	}
	return p, len(p.Departures) > 0
}

// ---- product data (Berdu initialStore JSON) ----

type ubProduct struct {
	name, category string
	shopPrice      int
	lines          []string
}

var (
	ubStoreRe     = regexp.MustCompile(`(?s)var initialStore=JSON\.parse\(("(?:[^"\\]|\\.)*")\)`)
	ubLdPriceRe   = regexp.MustCompile(`"offers":\{"@type":"Offer","price":(\d+)`)
	ubHTMLStartRe = regexp.MustCompile(`(?i)^tambah ke keranjang$`)
)

func ubProductData(html string) ubProduct {
	var out ubProduct
	if m := ubLdPriceRe.FindStringSubmatch(html); m != nil {
		out.shopPrice, _ = strconv.Atoi(m[1])
	}
	m := ubStoreRe.FindStringSubmatch(html)
	if m == nil {
		return out
	}
	var raw string
	if json.Unmarshal([]byte(m[1]), &raw) != nil {
		return out
	}
	var store []struct {
		Result struct {
			Camp map[string]json.RawMessage `json:"camp"`
		} `json:"result"`
	}
	if json.Unmarshal([]byte(raw), &store) != nil {
		return out
	}
	for _, q := range store {
		for k, v := range q.Result.Camp {
			if !strings.HasPrefix(k, "_productBySlug") {
				continue
			}
			var prod struct {
				Name     string `json:"name"`
				Category *struct {
					Name string `json:"name"`
				} `json:"category"`
				Description *struct {
					Blocks []struct {
						Text string `json:"text"`
						Type string `json:"type"`
					} `json:"blocks"`
				} `json:"description"`
			}
			if json.Unmarshal(v, &prod) != nil || prod.Description == nil {
				continue
			}
			out.name = strings.TrimSpace(prod.Name)
			if prod.Category != nil {
				out.category = prod.Category.Name
			}
			out.lines = nil
			for _, b := range prod.Description.Blocks {
				for _, l := range strings.Split(b.Text, "\n") {
					l = strings.TrimSpace(lstSpaceRe.ReplaceAllString(l, " "))
					if strings.Contains(b.Type, "list-item") && l != "" {
						l = "• " + l
					}
					out.lines = append(out.lines, l)
				}
			}
		}
	}
	return out
}

// ubHTMLDescLines is the fallback when the product JSON is missing: the
// description text between the cart button and "Produk Lainnya".
func ubHTMLDescLines(html string) []string {
	lines := lstHTMLLines(html)
	from, to := 0, len(lines)
	for i, l := range lines {
		if from == 0 && ubHTMLStartRe.MatchString(l) {
			from = i + 1
		}
		if strings.EqualFold(l, "Produk Lainnya") && i > from {
			to = i
			break
		}
	}
	return lines[from:to]
}

// ---- free-form text helpers ----

var (
	ubRangeRe = regexp.MustCompile(`(?i)\b(\d{1,2})\s*(?:(` + lstMonthAlt + `)\.?\s*(20\d{2})?\s*)?` +
		`(?:-|–|—|s/d|sd|sampai|hingga)\s*(\d{1,2})\s*(` + lstMonthAlt + `)\b\.?\s*(20\d{2})?`)
	ubMonthYearRe = regexp.MustCompile(`(?i)\b(` + lstMonthAlt + `)\b\.?\s+(20\d{2})`)
	ubYearRe      = regexp.MustCompile(`20(2[6-9])`)
	ubJutaRe      = regexp.MustCompile(`(?i)(rp\.?|idr)?\s*(\d{1,3}(?:[.,]\d{1,3})?)\s*(juta|jt)(an)?\b`)
	ubFullRe      = regexp.MustCompile(`(?i)(?:rp\.?|idr)\s*(\d{1,3}(?:\.\d{3}){2,})`)
	ubExcludeRe   = regexp.MustCompile(`(?i)\bdp\b|uang muka|down payment|\+|tambah|double|triple|quad|normal|dipotong|potongan|pembatalan|upgrade|cashback|diskon|admin|saku`)
	ubKeywordRe   = regexp.MustCompile(`(?i)harga|biaya|mulai|hanya|cuma|promo|spesial|total|paket`)
	ubTitleHariRe = regexp.MustCompile(`(?i)(?:^|[^\d\-–])(\d{1,2})\s*(?:hari|d)\b`)
	ubTextHariRe  = regexp.MustCompile(`(?i)(?:^|[^\d\-–])(\d{1,2})\s*(?:hari|d)\b`)
	ubPaketHariRe = regexp.MustCompile(`(?i)paket\s+(\d{1,2})\s*hari`)
	ubWindowRe    = regexp.MustCompile(`(?i)(daftar|pendaftaran|periode|berlaku|flash sale)\W*(?:\w+\W+){0,2}$`)
	ubAirlineRe   = regexp.MustCompile(`(?i)\b(saudia(?:\s+airlines)?|saudi arabian airlines|garuda(?:\s+indonesia)?|qatar(?:\s+airways)?|emirates|etihad|turkish(?:\s+airlines)?|oman\s+air(?:lines)?|lion\s+air|batik\s+air|citilink|scoot|indigo(?:\s+airlines)?|hainan(?:\s+airlines)?|egypt\s*air|flynas|flydubai)\b`)
	ubGuideRe     = regexp.MustCompile(`(?i)(?:pembimbing|bersama)\s*:\s*([^|()\n]+)`)
	ubGuideTextRe = regexp.MustCompile(`(?i)(?:dibimbing\s+(?:langsung\s+)?oleh|bersama\s+pembimbing|founder umrahbisa:)\s+((?:ustadz|ust\.?|kak|kang|dr\.?)\s+[^.!\n(]+?)(?:[.!\n(]|\s+dengan\b|\s+yang\b|$)`)
	ubGuideTitle  = regexp.MustCompile(`(?i)\b(?:bersama|bersam)\s+(.+)$|\b((?:ustadz|ust\.?|kak|kang|dr\.?)\s+[A-Z].*)$`)
	ubCityHotelRe = regexp.MustCompile(`(?i)(?:hotel\s+)?\b(makkah|mekkah|mekah|madinah)\s*:\s*([^|🏨✅📍•*\n]+)`)
	ubHotelParRe  = regexp.MustCompile(`(?i)(?:hotel\s+|&\s*|\bdan\s+)(makkah|mekkah|mekah|madinah)\s*\(([^)]+)\)`)
	ubHotelCityRe = regexp.MustCompile(`[Hh]otel\s+([A-Z][\w'’&. -]{2,40}?)\s*(?:atau setara\s*|/\s*setara\s*)?\(((?i)makkah|mekkah|mekah|madinah)\)`)
	ubAndCityRe   = regexp.MustCompile(`(?:&|\bdan)\s+([A-Z][\w'’&. -]{2,40}?)\s*(?:atau setara\s*|/\s*setara\s*)?\(((?i)makkah|mekkah|mekah|madinah)\)`)
	ubDistRe      = regexp.MustCompile(`(?i)(\d{1,4}(?:[.,]\d+)?)\s*(km|m|meter)\b`)
	ubUpgradeRe   = regexp.MustCompile(`(?i)\b(double|triple)\b[^+\d\n]{0,30}?(?:\+|tambahan)\s*(rp\.?\s*[\d.,]+\s*(?:juta|jt)?|[\d.,]+\s*(?:juta|jt))`)
	ubDPRe        = regexp.MustCompile(`(?i)\bdp\b[^\d\n]{0,25}?((?:rp\.?\s*)?\d[\d.,]*\s*(?:juta|jt)?)`)
	ubQuotaEachRe = regexp.MustCompile(`(?i)(?:kuota masing-masing keberangkatan\s*:?\s*(\d{2,3})\s*seat|(\d{2,3})\s*(?:seat|pax|orang)?\s+per\s+keberangkatan)`)
	ubQuotaRe     = regexp.MustCompile(`(?i)kuota\s+(?:terbatas|hanya)\s+(\d{2,3})\s*seat`)
	ubSeatRe      = regexp.MustCompile(`(?i)tersedia\s+(\d{1,4})\s*seat`)
	ubDirectRe    = regexp.MustCompile(`(?i)\bdirect\b|penerbangan langsung|tanpa transit`)
	ubTransitRe   = regexp.MustCompile(`(?i)\btransit\s+di\s+([A-Z][a-z]+)`)
	ubAirportRe   = regexp.MustCompile(`(?i)(?:keberangkatan|berangkat|terbang|start(?:ing)?)\s+(?:dari\s+)?(jakarta|surabaya|denpasar|bali|makassar|medan|solo|batam|bandung|yogyakarta|semarang|balikpapan|padang|palembang|pekanbaru)\b`)
	ubRouteRe     = regexp.MustCompile(`\b(CGK|JKT|SUB|DPS|UPG|KNO|SOC|BTH)\s*-\s*(?:JED|MED)\b`)
	ubIncHdrRe    = regexp.MustCompile(`(?i)fasilitas|sudah termasuk|harga termasuk|\binclude\b|all[- ]in\b|yang termasuk`)
	ubExcHdrRe    = regexp.MustCompile(`(?i)belum termasuk|tidak termasuk|\bexclude\b`)
	ubExcInlineRe = regexp.MustCompile(`(?i)belum termasuk\s*(?:\(exclude\))?\s*:?\s*([^.)\n]+)`)
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
		if k := strings.LastIndexAny(pre, ")!|•;\n"); k >= 0 {
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
	start, end time.Time
	pos, stop  int
}

// ubFindRanges extracts departure date ranges ("4 - 13 Januari 2027",
// "22 Des - 1 Jan 2027", "16-24 Maret"). A missing year is taken from the
// nearest preceding "<Month> <Year>" mention, else from the title.
// Registration windows ("daftar tanggal 20-24 Agustus") are skipped.
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
		if ubWindowRe.MatchString(s[max(0, m[0]-30):m[0]]) {
			continue
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
			continue // not a trip
		}
		out = append(out, ubRange{start: st, end: en, pos: m[0], stop: m[1]})
	}
	return out
}

// ubCleanHotel turns "Owais / Setara (Jalan kaki ±500m)" into "Owais".
func ubCleanHotel(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(strings.TrimPrefix(v, "Hotel "), "hotel ")
	for _, sep := range []string{" / ", "/setara", "/ setara", "/Setara", "(", " — ", " – ", " - ", " atau ", ",", "⭐", "±", " sekitar ", " jalan kaki"} {
		if i := strings.Index(strings.ToLower(v), strings.ToLower(sep)); i > 0 {
			v = v[:i]
		}
	}
	v = strings.TrimSpace(strings.Trim(v, " -:|•/"))
	lv := strings.ToLower(v)
	if len(v) < 3 || len(v) > 50 || strings.Contains(lv, "malam") || strings.Contains(lv, "km") ||
		strings.Contains(lv, "dekat") || strings.Contains(lv, "nyaman") || strings.Contains(lv, "bintang") ||
		!unicode.IsLetter([]rune(v)[0]) {
		return ""
	}
	return v
}

// ubStripBullet removes leading emoji / bullet characters.
func ubStripBullet(s string) string {
	return strings.TrimSpace(strings.TrimLeftFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '(' && r != '"'
	}))
}

func ubIsBulletLine(s string) bool {
	for _, r := range s {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}
	return false
}

// ubSplitTopLevel splits "a, b (c, d), dan e" on commas / " dan " outside parentheses.
func ubSplitTopLevel(s string) []string {
	var out []string
	depth, last := 0, 0
	flush := func(end int) {
		item := strings.TrimSpace(s[last:end])
		item = strings.TrimPrefix(strings.TrimPrefix(item, "dan "), "hingga ")
		item = strings.TrimSpace(strings.Trim(item, " .&"))
		if len(item) >= 2 {
			out = append(out, item)
		}
	}
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				flush(i)
				last = i + 1
			}
		}
	}
	flush(len(s))
	return out
}

func ubCleanGuide(g string) string {
	g = strings.TrimSpace(g)
	for _, sep := range []string{" - ", " — ", " | ", "(", "!"} {
		if i := strings.Index(g, sep); i > 0 {
			g = g[:i]
		}
	}
	g = strings.TrimSpace(strings.TrimRight(strings.TrimSpace(g), ".,;:"))
	lg := strings.ToLower(g)
	if len(g) < 4 || len(g) > 70 || strings.Contains(lg, "asatid") || strings.Contains(lg, "jakarta") ||
		strings.Contains(lg, "berpengalaman") || strings.Contains(lg, "umrahbisa") || strings.Contains(lg, "umrah bisa") ||
		strings.Contains(lg, "keluarga") || strings.Contains(lg, "owner") || strings.Contains(lg, "umrah") {
		return ""
	}
	return g
}

func ubAirlineIn(s string) string {
	if m := ubAirlineRe.FindString(s); m != "" {
		return lstAirlineName(m)
	}
	return ""
}

// ubDur returns the "N Hari" stated in s (not part of a "9-13 hari" range).
func ubDur(re *regexp.Regexp, s string) int {
	if m := re.FindStringSubmatch(s); m != nil {
		d, _ := strconv.Atoi(m[1])
		if d >= 5 && d <= 45 {
			return d
		}
	}
	return 0
}

// parseUmrahBisaProduct builds one package from a product's description lines.
func parseUmrahBisaProduct(title, category string, lines []string, shopPrice int, pageURL string) (CrawledPackage, bool) {
	p := CrawledPackage{TravelName: "Umrah Bisa", URL: pageURL}
	p.PackageName = strings.TrimSpace(title)
	if strings.HasPrefix(p.PackageName, "DP ") {
		p.PackageName = strings.TrimSpace(p.PackageName[3:])
	}
	text := strings.Join(lines, "\n")
	lineStart := make([]int, len(lines)+1)
	for i, l := range lines {
		lineStart[i+1] = lineStart[i] + len(l) + 1
	}
	lineOf := func(pos int) int {
		return sort.Search(len(lines), func(i int) bool { return lineStart[i+1] > pos })
	}

	ranges := ubFindRanges(text, title)
	if len(ranges) == 0 {
		ranges = ubFindRanges(title, title)
		for i := range ranges {
			ranges[i].pos, ranges[i].stop = -1, -1
		}
	}
	if len(ranges) == 0 {
		return p, false
	}
	multi := len(ranges) > 1
	prices := ubFindPrices(text)

	// --- package-level values ---
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

	pkgDur := ubDur(ubTitleHariRe, title)
	if pkgDur == 0 && !multi {
		pkgDur = ubDur(ubTextHariRe, text)
	}

	pkgAirline := ""
	for _, l := range lines {
		ll := strings.ToLower(l)
		if strings.Contains(ll, "maskapai") || strings.Contains(ll, "penerbangan") || strings.Contains(ll, "terbang") {
			if a := ubAirlineIn(l); a != "" {
				pkgAirline = a
				break
			}
		}
	}
	if pkgAirline == "" {
		pkgAirline = ubAirlineIn(title)
	}
	if pkgAirline == "" {
		pkgAirline = ubAirlineIn(text)
	}

	pkgGuide := ""
	if !multi {
		for _, l := range lines {
			if m := ubGuideRe.FindStringSubmatch(l); m != nil && strings.Contains(strings.ToLower(l), "pembimbing") {
				if g := ubCleanGuide(m[1]); g != "" {
					pkgGuide = g
					break
				}
			}
		}
		if pkgGuide == "" {
			if m := ubGuideTextRe.FindStringSubmatch(text); m != nil {
				pkgGuide = ubCleanGuide(m[1])
			}
		}
		if pkgGuide == "" {
			if m := ubGuideTitle.FindStringSubmatch(title); m != nil {
				pkgGuide = ubCleanGuide(m[1] + m[2])
			}
		}
	}
	p.Guide = pkgGuide

	// Hotels.
	setHotel := func(city, raw string) {
		name := ubCleanHotel(lstCleanHotel(raw))
		if name == "" {
			return
		}
		_, stars := lstSplitStars(strings.TrimSpace(strings.Trim(raw, "() ")))
		if stars == 0 {
			if n := strings.Count(raw, "⭐"); n >= 1 && n <= 5 {
				stars = n
			}
		}
		dist := 0
		if m := ubDistRe.FindStringSubmatch(raw); m != nil {
			f, _ := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64)
			if strings.EqualFold(m[2], "km") {
				f *= 1000
			}
			dist = int(f + 0.5)
		}
		if strings.EqualFold(city, "madinah") {
			if p.HotelMadinah == "" {
				p.HotelMadinah, p.StarsMadinah, p.DistanceMadinah = name, stars, dist
			}
		} else if p.HotelMakkah == "" {
			p.HotelMakkah, p.StarsMakkah, p.DistanceMakkah = name, stars, dist
		}
	}
	for _, l := range lines {
		for _, m := range ubCityHotelRe.FindAllStringSubmatch(l, -1) {
			setHotel(m[1], m[2])
		}
		for _, m := range ubHotelParRe.FindAllStringSubmatch(l, -1) {
			setHotel(m[1], m[2])
		}
		for _, re := range []*regexp.Regexp{ubHotelCityRe, ubAndCityRe} {
			for _, m := range re.FindAllStringSubmatch(l, -1) {
				setHotel(m[2], m[1])
			}
		}
	}

	// Facilities / excludes.
	stopList := func(l string) bool {
		if strings.TrimSpace(l) == "" {
			return true
		}
		s := ubStripBullet(l)
		if strings.HasSuffix(s, ":") && !strings.Contains(strings.ToLower(s), "hotel") {
			return true
		}
		return ubExcHdrRe.MatchString(s) || len(s) > 110
	}
	collect := func(start int) []string {
		var out []string
		for j := start; j < len(lines); j++ {
			l := lines[j]
			if stopList(l) || !ubIsBulletLine(l) {
				break
			}
			if s := ubStripBullet(l); len(s) >= 2 && !strings.HasSuffix(s, ":") {
				out = append(out, s)
			}
		}
		return out
	}
	for i, l := range lines {
		s := ubStripBullet(l)
		head, rest, hasColon := strings.Cut(s, ":")
		switch {
		case len(p.Excludes) == 0 && ubExcHdrRe.MatchString(head) && len(head) < 50:
			if hasColon && strings.TrimSpace(rest) != "" {
				r := rest
				if k := strings.Index(r, "."); k > 0 {
					r = r[:k]
				}
				p.Excludes = ubSplitTopLevel(r)
			} else {
				p.Excludes = collect(i + 1)
			}
		case ubIncHdrRe.MatchString(head) && len(head) < 64 && !ubExcHdrRe.MatchString(head) &&
			!strings.Contains(strings.ToLower(head), "hotel"):
			if hasColon && strings.TrimSpace(rest) != "" {
				r := strings.TrimSpace(rest)
				if k := strings.Index(r, ". "); k > 0 {
					r = r[:k]
				}
				p.Facilities = append(p.Facilities, ubSplitTopLevel(r)...)
			} else if !hasColon && len(s) > 40 {
				// A sentence that merely mentions "fasilitas" - unless it is
				// followed by a bullet list ("... sudah termasuk dalam harga! ❤️").
				if strings.Contains(strings.ToLower(s), "termasuk") && i+1 < len(lines) && ubIsBulletLine(lines[i+1]) {
					p.Facilities = append(p.Facilities, collect(i+1)...)
				}
			} else {
				p.Facilities = append(p.Facilities, collect(i+1)...)
			}
		}
	}
	if len(p.Excludes) == 0 {
		if m := ubExcInlineRe.FindStringSubmatch(text); m != nil {
			p.Excludes = ubSplitTopLevel(m[1])
		}
	}
	p.Facilities = lstDedupStrings(p.Facilities)
	p.Excludes = lstDedupStrings(p.Excludes)

	// Room upgrades (per person, on top of the departure price).
	addDouble, addTriple := 0, 0
	for _, m := range ubUpgradeRe.FindAllStringSubmatch(text, -1) {
		v := lstMoney(m[2])
		if v <= 0 || v >= lstMinPrice*4 {
			continue
		}
		if strings.EqualFold(m[1], "double") && addDouble == 0 {
			addDouble = v
		} else if strings.EqualFold(m[1], "triple") && addTriple == 0 {
			addTriple = v
		}
	}

	// DP: written in the text; else the shop price of a "DP ..." product.
	for _, l := range lines {
		ll := strings.ToLower(l)
		if strings.Contains(ll, "flash") || strings.Contains(ll, "normal") || strings.Contains(ll, "khusus") {
			continue
		}
		if m := ubDPRe.FindStringSubmatch(l); m != nil {
			if v := lstMoney(m[1]); v >= 500000 && v < lstMinPrice*2 {
				p.DownPayment = v
				break
			}
		}
	}
	if p.DownPayment == 0 && strings.HasPrefix(strings.TrimSpace(title), "DP ") && shopPrice > 0 {
		p.DownPayment = shopPrice
	}

	// Seats / quota.
	quotaEach := 0
	if m := ubQuotaEachRe.FindStringSubmatch(text); m != nil {
		quotaEach, _ = strconv.Atoi(m[1] + m[2])
	}
	quotaOne, seatsOne := 0, 0
	if !multi {
		if m := ubQuotaRe.FindStringSubmatch(text); m != nil {
			quotaOne, _ = strconv.Atoi(m[1])
		}
		st := title + "\n" + text
		if m := ubSeatRe.FindStringSubmatchIndex(st); m != nil &&
			!strings.Contains(strings.ToLower(st[max(0, m[0]-8):m[0]]), "total") {
			seatsOne, _ = strconv.Atoi(st[m[2]:m[3]])
		}
	}

	// Flight.
	if ubDirectRe.MatchString(text) {
		v := true
		p.Direct = &v
	} else if m := ubTransitRe.FindStringSubmatch(text); m != nil {
		v := false
		p.Direct = &v
		p.TransitVia = m[1]
	}
	if m := ubAirportRe.FindStringSubmatch(text); m != nil {
		p.Airport = lstCity(m[1])
	} else if m := ubRouteRe.FindStringSubmatch(text); m != nil {
		p.Airport = lstCity(m[1])
	} else if strings.Contains(strings.ToLower(category), "surabaya") {
		p.Airport = "Surabaya"
	}

	// --- departures ---
	durs := map[int]bool{}
	airlines := map[string]bool{}
	for i, r := range ranges {
		d := CrawledDeparture{Date: r.start.Format(lstISO), ReturnDate: r.end.Format(lstISO)}
		dur, price, airline, guide := 0, 0, "", ""
		if r.pos >= 0 {
			li := lineOf(r.pos)
			lineEnd := lineStart[li+1] - 1
			segEnd := lineEnd
			if i+1 < len(ranges) && ranges[i+1].pos < segEnd {
				segEnd = ranges[i+1].pos
			}
			segStart := lineStart[li]
			if i > 0 && ranges[i-1].stop > segStart {
				segStart = ranges[i-1].stop
			}
			pre, seg := text[segStart:r.pos], text[r.stop:segEnd]
			ctx := pre + " " + seg
			dur = ubDur(ubTextHariRe, ctx)
			airline = ubAirlineIn(ctx)
			if m := ubGuideRe.FindStringSubmatch(seg); m != nil {
				guide = ubCleanGuide(m[1])
			}
			if multi {
				for _, h := range prices {
					if h.pos >= r.stop && h.end <= segEnd {
						price = h.value
						break
					}
				}
				if price == 0 {
					// Price on the following lines ("💰 27,9 JUTA"), before the next range.
					limit := len(text)
					if i+1 < len(ranges) {
						limit = ranges[i+1].pos
					}
					if l3 := lineStart[min(len(lines), li+4)]; l3 < limit {
						limit = l3
					}
					for _, h := range prices {
						if h.pos >= segEnd && h.end <= limit {
							price = h.value
							break
						}
					}
				}
				if dur == 0 {
					// Section header "Paket 9 Hari (9D)" above the range.
					for k := li - 1; k >= 0; k-- {
						if m := ubPaketHariRe.FindStringSubmatch(lines[k]); m != nil {
							dur, _ = strconv.Atoi(m[1])
							break
						}
						if k < li-12 {
							break
						}
					}
				}
			}
		}
		if dur == 0 && !multi {
			dur = pkgDur
		}
		if dur == 0 && multi && len(ranges) > 0 && pkgDur > 0 {
			// Title states one duration for the whole product ("Umrah Ceria 10 Hari ...").
			dur = pkgDur
		}
		if price == 0 {
			price = pagePrice
		}
		if airline == "" {
			airline = pkgAirline
		}
		d.Price = price
		if price > 0 && addDouble > 0 {
			d.PriceDouble = price + addDouble
		}
		if price > 0 && addTriple > 0 {
			d.PriceTriple = price + addTriple
		}
		d.Airline = airline
		if quotaEach > 0 {
			d.SeatsTotal = quotaEach
		} else if quotaOne > 0 {
			d.SeatsTotal = quotaOne
		}
		if seatsOne > 0 {
			d.Seats = seatsOne
		}
		if guide != "" && multi {
			// Guides differ per departure; keep them in the package guide list.
			entry := guide + " (" + r.start.Format("2 Jan") + ")"
			if p.Guide == "" {
				p.Guide = entry
			} else {
				p.Guide += "; " + entry
			}
		} else if guide != "" && p.Guide == "" {
			p.Guide = guide
		}
		durs[dur] = true
		airlines[airline] = true
		p.Departures = append(p.Departures, d)
	}
	if len(durs) == 1 {
		for k := range durs {
			p.Duration = k
		}
	}
	if len(airlines) == 1 {
		for k := range airlines {
			p.Airline = k
		}
	}
	// Package-level prices: the cheapest departure's.
	best := -1
	for i, d := range p.Departures {
		if d.Price > 0 && (best < 0 || d.Price < p.Departures[best].Price) {
			best = i
		}
	}
	if best < 0 {
		return p, false
	}
	p.Price = p.Departures[best].Price
	p.PriceTriple = p.Departures[best].PriceTriple
	p.PriceDouble = p.Departures[best].PriceDouble
	if p.Airline == "" && pkgAirline != "" && len(airlines) > 1 {
		p.Airline = ""
	}
	return p, true
}
