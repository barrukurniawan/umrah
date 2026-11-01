package crawlers

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Shared helpers for the listing parsers (umitravellist.go, umrahbisalist.go,
// rabbanilist.go). All identifiers are prefixed with "lst" to avoid clashes
// with helpers in other parser files.

const (
	lstUserAgent   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
	lstMinPrice    = 5000000
	lstMaxPrice    = 200000000
	lstPoliteDelay = 700 * time.Millisecond
)

// lstFetch downloads a page and returns its body as a string.
func lstFetch(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", lstUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// lstCutoff returns today's date (Asia/Jakarta, 00:00). Departures before it are dropped.
func lstCutoff() time.Time {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	now := time.Now().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// lstMonthAlt is a regexp alternation of Indonesian/English month names
// (longest first, so RE2 leftmost-first matching picks the full name).
const lstMonthAlt = `januari|februari|maret|april|mei|juni|juli|agustus|september|oktober|november|desember|` +
	`january|february|march|may|june|july|august|october|december|` +
	`sept|jan|feb|mar|apr|jun|jul|agu|agt|aug|sep|okt|oct|nov|des|dec`

// lstMonthNum maps a month token (any case) to its number, 0 if unknown.
func lstMonthNum(s string) int {
	s = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(s), "."))
	switch {
	case strings.HasPrefix(s, "jan"):
		return 1
	case strings.HasPrefix(s, "feb"):
		return 2
	case strings.HasPrefix(s, "mar"):
		return 3
	case strings.HasPrefix(s, "apr"):
		return 4
	case s == "mei" || s == "may":
		return 5
	case strings.HasPrefix(s, "jun"):
		return 6
	case strings.HasPrefix(s, "jul"):
		return 7
	case strings.HasPrefix(s, "agu") || strings.HasPrefix(s, "agt") || strings.HasPrefix(s, "aug"):
		return 8
	case strings.HasPrefix(s, "sep"):
		return 9
	case strings.HasPrefix(s, "okt") || strings.HasPrefix(s, "oct"):
		return 10
	case strings.HasPrefix(s, "nov"):
		return 11
	case strings.HasPrefix(s, "des") || strings.HasPrefix(s, "dec"):
		return 12
	}
	return 0
}

var lstSingleDateRe = regexp.MustCompile(`(?i)^\s*(\d{1,2})\s+(` + lstMonthAlt + `)\.?\s+(20\d{2})\s*$`)

// lstParseDate parses "15 Jan 2027", "15 Januari 2027", "13 agustus 2026", ...
func lstParseDate(s string) (time.Time, bool) {
	m := lstSingleDateRe.FindStringSubmatch(s)
	if m == nil {
		return time.Time{}, false
	}
	var d, y int
	fmt.Sscanf(m[1], "%d", &d)
	fmt.Sscanf(m[3], "%d", &y)
	return lstMakeDate(y, lstMonthNum(m[2]), d)
}

func lstMakeDate(y, mo, d int) (time.Time, bool) {
	if mo < 1 || mo > 12 || d < 1 || d > 31 || y < 2000 {
		return time.Time{}, false
	}
	t := time.Date(y, time.Month(mo), d, 0, 0, 0, 0, time.UTC)
	if t.Day() != d {
		return time.Time{}, false
	}
	return t, true
}

// lstFormatDate renders a date as "2 Jan 2006" (understood by the importer).
func lstFormatDate(t time.Time) string {
	return t.Format("2 Jan 2006")
}

// lstKeepPackage drops past departure dates (normalising the remaining ones to
// "2 Jan 2006") and reports whether the package is still worth emitting.
func lstKeepPackage(p *CrawledPackage, cutoff time.Time) bool {
	if p.Price < lstMinPrice || p.Price > lstMaxPrice {
		return false
	}
	var kept []string
	seen := map[string]bool{}
	for _, ds := range p.DepartureDates {
		t, ok := lstParseDate(ds)
		if !ok || t.Before(cutoff) {
			continue
		}
		f := lstFormatDate(t)
		if !seen[f] {
			seen[f] = true
			kept = append(kept, f)
		}
	}
	p.DepartureDates = kept
	return len(kept) > 0
}

var (
	lstScriptRe   = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	lstStyleRe    = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	lstNoscriptRe = regexp.MustCompile(`(?is)<noscript[^>]*>.*?</noscript>`)
	lstTagRe      = regexp.MustCompile(`<[^>]+>`)
	lstSpaceRe    = regexp.MustCompile(`[ \t\r\f\v\x{00a0}]+`)
)

// lstHTMLLines strips scripts/styles/tags and returns the non-empty text lines.
func lstHTMLLines(html string) []string {
	s := lstScriptRe.ReplaceAllString(html, "")
	s = lstStyleRe.ReplaceAllString(s, "")
	s = lstNoscriptRe.ReplaceAllString(s, "")
	s = lstTagRe.ReplaceAllString(s, "\n")
	s = lstUnescape(s)
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(lstSpaceRe.ReplaceAllString(l, " "))
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

var lstEntityReplacer = strings.NewReplacer(
	"&amp;", "&", "&#038;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`,
	"&#039;", "'", "&#39;", "'", "&apos;", "'", "&nbsp;", " ", "&#8211;", "–",
	"&#8212;", "—", "&ndash;", "–", "&mdash;", "—", "&#8217;", "’",
)

func lstUnescape(s string) string { return lstEntityReplacer.Replace(s) }

var lstOgTitleRe = regexp.MustCompile(`<meta[^>]+property=["']og:title["'][^>]+content=(?:"([^"]+)"|'([^']+)')`)

func lstOgTitle(html string) string {
	if m := lstOgTitleRe.FindStringSubmatch(html); m != nil {
		return strings.TrimSpace(lstUnescape(m[1] + m[2]))
	}
	return ""
}

// ---------------------------------------------------------------------------
// Helpers shared by the rabbani_list / umi_travel_list / umrahbisa_list parsers
// for filling the richer CrawledPackage / CrawledDeparture fields.
// ---------------------------------------------------------------------------

const lstISO = "2006-01-02"

// lstFetchPolite fetches a URL, retrying once after a short pause on failure.
func lstFetchPolite(u string) (string, error) {
	body, err := lstFetch(u)
	if err != nil {
		time.Sleep(2 * time.Second)
		body, err = lstFetch(u)
	}
	return body, err
}

// lstAirlineCodes maps IATA codes as travels write them ("EY / WY / QR") to names.
var lstAirlineCodes = map[string]string{
	"SV": "Saudia", "GA": "Garuda Indonesia", "QR": "Qatar Airways", "EY": "Etihad Airways",
	"WY": "Oman Air", "TK": "Turkish Airlines", "EK": "Emirates", "MS": "EgyptAir",
	"6E": "IndiGo", "HU": "Hainan Airlines", "ID": "Batik Air", "JT": "Lion Air",
	"MH": "Malaysia Airlines", "SQ": "Singapore Airlines", "FZ": "flydubai", "XY": "flynas",
	"KU": "Kuwait Airways", "GF": "Gulf Air", "UL": "SriLankan Airlines", "AI": "Air India",
	"J9": "Jazeera Airways", "BI": "Royal Brunei Airlines", "TR": "Scoot", "OD": "Batik Air Malaysia",
	"RJ": "Royal Jordanian", "PK": "Pakistan International Airlines", "CZ": "China Southern Airlines",
}

// lstAirlineName normalises one airline mention ("Saudia Airlines", "Qatar
// Airlines", "Egypt Air", "SV") to a canonical name; "" when not an airline.
func lstAirlineName(s string) string {
	t := strings.TrimSpace(strings.Trim(s, " .,;:-()"))
	if t == "" {
		return ""
	}
	if n, ok := lstAirlineCodes[strings.ToUpper(t)]; ok && len(t) == 2 {
		return n
	}
	l := strings.ToLower(t)
	switch {
	case strings.Contains(l, "egypt"):
		return "EgyptAir"
	case strings.Contains(l, "hainan"):
		return "Hainan Airlines"
	case strings.Contains(l, "indigo"):
		return "IndiGo"
	case strings.Contains(l, "flynas"):
		return "flynas"
	case strings.Contains(l, "flydubai") || strings.Contains(l, "fly dubai"):
		return "flydubai"
	case strings.Contains(l, "malaysia"):
		return "Malaysia Airlines"
	case strings.Contains(l, "singapore"):
		return "Singapore Airlines"
	case strings.Contains(l, "kuwait"):
		return "Kuwait Airways"
	case strings.Contains(l, "jazeera"):
		return "Jazeera Airways"
	case strings.Contains(l, "sriwijaya"):
		return "Sriwijaya Air"
	}
	return KnownAirline(t)
}

var lstAirlineSplitRe = regexp.MustCompile(`\s*(?:/|,|&|\bdan\b|\bor\b|\batau\b)\s*`)

// lstAirlines turns a published airline field ("EY / WY / QR", "SV / GA",
// "Saudia Airlines") into "Etihad Airways / Oman Air / Qatar Airways".
// Unknown tokens are dropped; "" when nothing is recognised.
func lstAirlines(s string) string {
	var out []string
	seen := map[string]bool{}
	for _, tok := range lstAirlineSplitRe.Split(s, -1) {
		if n := lstAirlineName(tok); n != "" && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return strings.Join(out, " / ")
}

var lstJutaRe = regexp.MustCompile(`(?i)(\d{1,3}(?:[.,]\d{1,3})?)\s*(?:juta|jt)`)
var lstRupiahRe = regexp.MustCompile(`(?i)(?:rp\.?|idr)\s*(\d{1,3}(?:[.,]\d{3})+|\d{4,})`)

// lstMoney parses "Rp. 28,9 Juta", "28,9 Juta", "Rp41.400.000,-", "5 Jt".
func lstMoney(s string) int {
	if m := lstRupiahRe.FindStringSubmatch(s); m != nil {
		v := 0
		for _, c := range m[1] {
			if c >= '0' && c <= '9' {
				v = v*10 + int(c-'0')
			}
		}
		if v >= 100000 {
			return v
		}
	}
	if m := lstJutaRe.FindStringSubmatch(s); m != nil {
		num := strings.ReplaceAll(m[1], ",", ".")
		var f float64
		fmt.Sscanf(num, "%g", &f)
		return int(f*1000000 + 0.5)
	}
	return 0
}

var lstStarsSuffixRe = regexp.MustCompile(`\s*(?:\(?\s*(?:\*\s*(\d)|(\d)\s*\*|⭐️?\s*(\d)|((?:⭐️?)+))\s*\)?)\s*$`)

// lstSplitStars removes a trailing star rating ("Dar Al Naeem *3", "Grand Al
// Massa (⭐⭐⭐⭐)", "Owais ⭐️4") from a hotel name and returns it separately.
func lstSplitStars(name string) (string, int) {
	m := lstStarsSuffixRe.FindStringSubmatchIndex(name)
	if m == nil {
		return strings.TrimSpace(name), 0
	}
	stars := 0
	for g := 1; g <= 3; g++ {
		if m[2*g] >= 0 {
			fmt.Sscanf(name[m[2*g]:m[2*g+1]], "%d", &stars)
		}
	}
	if m[8] >= 0 {
		stars = strings.Count(name[m[8]:m[9]], "⭐")
	}
	if stars < 1 || stars > 5 {
		return strings.TrimSpace(name), 0
	}
	return strings.TrimSpace(name[:m[0]]), stars
}

var lstCityNames = map[string]string{
	"jakarta": "Jakarta", "cgk": "Jakarta", "jkt": "Jakarta", "surabaya": "Surabaya", "sub": "Surabaya",
	"denpasar": "Denpasar", "bali": "Denpasar", "dps": "Denpasar", "makassar": "Makassar", "upg": "Makassar",
	"medan": "Medan", "kno": "Medan", "solo": "Solo", "soc": "Solo", "batam": "Batam", "bth": "Batam",
	"bandung": "Bandung", "bdo": "Bandung", "yogyakarta": "Yogyakarta", "yia": "Yogyakarta",
	"semarang": "Semarang", "srg": "Semarang", "balikpapan": "Balikpapan", "bpn": "Balikpapan",
	"padang": "Padang", "pdg": "Padang", "palembang": "Palembang", "plm": "Palembang",
	"pekanbaru": "Pekanbaru", "pku": "Pekanbaru", "banjarmasin": "Banjarmasin", "bdj": "Banjarmasin",
	"lombok": "Lombok", "lop": "Lombok", "aceh": "Banda Aceh", "btj": "Banda Aceh",
}

// lstCity normalises a departure city / airport code; "" when unknown.
func lstCity(s string) string {
	return lstCityNames[strings.ToLower(strings.TrimSpace(s))]
}

// lstFinalize normalises a package built from Departures: drops past and
// duplicate dates, sorts them, fills the package-level summary fields and the
// legacy DepartureDates/Seats/Hotel* fields, and reports whether the package
// should be emitted.
func lstFinalize(p *CrawledPackage, cutoff time.Time) bool {
	byDate := map[string]int{}
	var deps []CrawledDeparture
	for _, d := range p.Departures {
		t, err := time.Parse(lstISO, d.Date)
		if err != nil || t.Before(cutoff) {
			continue
		}
		if d.ReturnDate != "" {
			if r, err := time.Parse(lstISO, d.ReturnDate); err != nil || !r.After(t) {
				d.ReturnDate = ""
			}
		}
		if i, ok := byDate[d.Date]; ok {
			lstMergeDeparture(&deps[i], d)
			continue
		}
		byDate[d.Date] = len(deps)
		deps = append(deps, d)
	}
	sort.SliceStable(deps, func(i, j int) bool { return deps[i].Date < deps[j].Date })
	if len(deps) == 0 {
		return false
	}

	// Package price = cheapest departure price when not set.
	if p.Price == 0 {
		for _, d := range deps {
			if d.Price > 0 && (p.Price == 0 || d.Price < p.Price) {
				p.Price = d.Price
			}
		}
	}
	if p.Price < lstMinPrice || p.Price > lstMaxPrice {
		return false
	}
	// Per-departure values equal to the package value are redundant.
	allFull := true
	for i := range deps {
		d := &deps[i]
		if d.Price == p.Price {
			d.Price = 0
		}
		if d.PriceTriple == p.PriceTriple {
			d.PriceTriple = 0
		}
		if d.PriceDouble == p.PriceDouble {
			d.PriceDouble = 0
		}
		if d.HotelMakkah == p.HotelMakkah {
			d.HotelMakkah = ""
		}
		if d.HotelMadinah == p.HotelMadinah {
			d.HotelMadinah = ""
		}
		if d.Airline == p.Airline {
			d.Airline = ""
		}
		if d.Airport == p.Airport {
			d.Airport = ""
		}
		if !d.Full {
			allFull = false
		}
	}
	p.Departures = deps

	p.DepartureDates = nil
	for _, d := range deps {
		t, _ := time.Parse(lstISO, d.Date)
		p.DepartureDates = append(p.DepartureDates, lstFormatDate(t))
	}
	if len(deps) == 1 {
		d := deps[0]
		if d.Seats > 0 {
			p.Seats = d.Seats
		}
		if d.SeatsTotal > 0 {
			p.SeatsTotal = d.SeatsTotal
		}
	}
	if allFull {
		p.Full = true
	}
	return true
}

func lstMergeDeparture(dst *CrawledDeparture, src CrawledDeparture) {
	if dst.ReturnDate == "" {
		dst.ReturnDate = src.ReturnDate
	}
	if dst.Price == 0 || (src.Price > 0 && src.Price < dst.Price) {
		dst.Price = src.Price
	}
	if dst.PriceTriple == 0 {
		dst.PriceTriple = src.PriceTriple
	}
	if dst.PriceDouble == 0 {
		dst.PriceDouble = src.PriceDouble
	}
	if dst.Seats == 0 {
		dst.Seats = src.Seats
	}
	if dst.SeatsTotal == 0 {
		dst.SeatsTotal = src.SeatsTotal
	}
	dst.Full = dst.Full && src.Full
	if dst.HotelMakkah == "" {
		dst.HotelMakkah = src.HotelMakkah
	}
	if dst.HotelMadinah == "" {
		dst.HotelMadinah = src.HotelMadinah
	}
	if dst.Airline == "" {
		dst.Airline = src.Airline
	}
	if dst.Airport == "" {
		dst.Airport = src.Airport
	}
}

// lstListItems collects the list items that follow lines[start] until a line
// for which stop returns true. Separator-only lines ("|", "*", "•") are skipped.
func lstListItems(lines []string, start int, stop func(string) bool) []string {
	var out []string
	seen := map[string]bool{}
	for i := start; i < len(lines); i++ {
		l := strings.TrimSpace(lines[i])
		if stop(l) {
			break
		}
		l = strings.TrimSpace(strings.TrimLeft(l, "•*-|·✓✔✅❌ \t"))
		if l == "" || len(l) < 2 || seen[strings.ToLower(l)] {
			continue
		}
		seen[strings.ToLower(l)] = true
		out = append(out, l)
	}
	return out
}

var lstSetaraRe = regexp.MustCompile(`(?i)\s*(?:/\s*|\batau\s+|\bor\s+)?\b(?:setaraf|setara|setaraf?nya|equivalent)\b\.?\s*$`)

// lstCleanHotel drops a trailing "/ Setara(f)" / "atau setara" marker.
func lstCleanHotel(s string) string {
	s = strings.TrimSpace(lstSetaraRe.ReplaceAllString(strings.TrimSpace(s), ""))
	return strings.TrimSpace(strings.TrimRight(s, "/-–,"))
}
