package crawlers

// Shared engine for travel sites built on the erahajj.co.id platform
// ("/transaksi/paket-umrah"): Hamdan Tour, Uhud Tour and PT Labbaika Cipta Imani.
//
// The listing page only shows a summary card per package. All per-departure data
// lives behind the AJAX calls the detail page itself makes:
//
//	POST /transaksi/paket-umrah/load-bandara-keberangkatan   (program_hari)            -> <option> airports
//	POST /transaksi/paket-umrah/load-tanggal-keberangkatan   (+ bandara_id)            -> <option> "09 Desember 2026 (36 Seat Tersedia)" / "(Habis)"
//	POST /transaksi/paket-umrah/load-harga-paket             (+ keberangkatan_id)      -> room prices (jenis-kamar 2/3/4 = double/triple/quad)
//	POST /transaksi/paket-umrah/get-informasi-maskapai-hotel (+ keberangkatan_id)      -> hotels + stars, airline, flight legs with dates/airports
//
// "program_hari" is the marketed duration ("Umrah 9 Hari"). The return date is
// taken from the published return flight (landing date of the last "Kembali"
// leg) when the travel filled it in; otherwise it is departure + program - 1
// (verified on uhudtour.com/.../detail/289: 9 hari, depart 09 Dec, return flight
// lands CGK 17 Dec).

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type hamdanEHSite struct {
	Base       string // e.g. "https://hamdantour.id"
	ListURL    string // e.g. "https://hamdantour.id/transaksi/paket-umrah"
	TravelName string
	Workers    int
}

const (
	hamdanEHUA       = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"
	hamdanEHMaxPages = 40
)

var (
	hamdanEHDetailRe  = regexp.MustCompile(`/transaksi/paket-umrah/detail/(\d+)/`)
	hamdanEHDateRe    = regexp.MustCompile(`(?i)(\d{1,2})\s+([a-z]+)\s+(20\d{2})`)
	hamdanEHShortDate = regexp.MustCompile(`(?i)(\d{1,2})\s+([a-z]{3,})`)
	hamdanEHSeatRe    = regexp.MustCompile(`(?i)(\d+)\s*seat\s*tersedia`)
	hamdanEHAirportRe = regexp.MustCompile(`\(([A-Z]{3})\)`)
	hamdanEHCodeRe    = regexp.MustCompile(`\b([A-Z]{3})\b`)
	hamdanEHDPRe      = regexp.MustCompile(`(?i)\b(?:DP|down\s*payment|uang\s+muka)\b[^0-9\n]{0,40}?(?:Rp\.?|IDR)?\s*([0-9][0-9.,]*)\s*(jt|juta|rb|ribu)?`)
	hamdanEHGuideRe   = regexp.MustCompile(`(?i)\bbersama\s+((?:ustadz|ustadzah|ustaz|ust\.|syekh|syaikh|kh\.?|habib|buya)\s+[^|,()\-–]+)`)
	hamdanEHMonths    = map[string]time.Month{
		"januari": 1, "jan": 1, "january": 1,
		"februari": 2, "feb": 2, "february": 2,
		"maret": 3, "mar": 3, "march": 3,
		"april": 4, "apr": 4,
		"mei": 5, "may": 5,
		"juni": 6, "jun": 6, "june": 6,
		"juli": 7, "jul": 7, "july": 7,
		"agustus": 8, "agu": 8, "agt": 8, "aug": 8, "august": 8,
		"september": 9, "sep": 9, "sept": 9,
		"oktober": 10, "okt": 10, "oct": 10, "october": 10,
		"november": 11, "nov": 11, "nop": 11,
		"desember": 12, "des": 12, "dec": 12, "december": 12,
	}
	// Saudi arrival airports: a single published leg from Indonesia to one of
	// these means a direct flight.
	hamdanEHSaudi = map[string]bool{"JED": true, "MED": true, "TIF": true, "RUH": true, "ULH": true, "YNB": true, "DMM": true, "AHB": true, "TUU": true}
	// Transit airport code -> city, for TransitVia.
	hamdanEHCities = map[string]string{
		"MCT": "Muscat", "DOH": "Doha", "AUH": "Abu Dhabi", "DXB": "Dubai", "SHJ": "Sharjah",
		"KUL": "Kuala Lumpur", "SIN": "Singapura", "IST": "Istanbul", "SAW": "Istanbul",
		"BAH": "Bahrain", "CMB": "Kolombo", "BOM": "Mumbai", "DEL": "Delhi", "KWI": "Kuwait",
		"CAI": "Kairo", "AMM": "Amman", "BKK": "Bangkok", "MAA": "Chennai", "HYD": "Hyderabad",
		"JED": "Jeddah", "MED": "Madinah", "RUH": "Riyadh", "CGK": "Jakarta", "SUB": "Surabaya",
	}
)

func hamdanEHToday() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func hamdanEHParseDate(s string) (time.Time, bool) {
	m := hamdanEHDateRe.FindStringSubmatch(s)
	if len(m) < 4 {
		return time.Time{}, false
	}
	mon, ok := hamdanEHMonths[strings.ToLower(m[2])]
	if !ok {
		return time.Time{}, false
	}
	d, _ := strconv.Atoi(m[1])
	y, _ := strconv.Atoi(m[3])
	if d < 1 || d > 31 {
		return time.Time{}, false
	}
	return time.Date(y, mon, d, 0, 0, 0, 0, time.UTC), true
}

// hamdanEHFlightDate parses "17 Dec, 09:25" (no year) relative to the departure date.
func hamdanEHFlightDate(s string, dep time.Time) (time.Time, bool) {
	m := hamdanEHShortDate.FindStringSubmatch(s)
	if len(m) < 3 {
		return time.Time{}, false
	}
	mon, ok := hamdanEHMonths[strings.ToLower(m[2])]
	if !ok {
		return time.Time{}, false
	}
	d, _ := strconv.Atoi(m[1])
	for _, y := range []int{dep.Year(), dep.Year() + 1} {
		t := time.Date(y, mon, d, 0, 0, 0, 0, time.UTC)
		if !t.Before(dep.AddDate(0, 0, -1)) && t.Sub(dep) < 70*24*time.Hour {
			return t, true
		}
	}
	return time.Time{}, false
}

func hamdanEHClean(s string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

// ---------------------------------------------------------------- HTTP

type hamdanEHSession struct {
	client  *http.Client
	site    hamdanEHSite
	referer string
	token   string
	id      string
}

func hamdanEHNewClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Timeout: 40 * time.Second, Jar: jar}
}

func hamdanEHGet(client *http.Client, u string) (string, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}
		req, _ := http.NewRequest("GET", u, nil)
		req.Header.Set("User-Agent", hamdanEHUA)
		req.Header.Set("Accept", "text/html,application/xhtml+xml")
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		b, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("%s: HTTP %d", u, resp.StatusCode)
			continue
		}
		if err != nil {
			lastErr = err
			continue
		}
		return string(b), nil
	}
	return "", lastErr
}

func (s *hamdanEHSession) post(endpoint string, extra [][2]string) (string, error) {
	fields := [][2]string{
		{"_token", s.token},
		{"id", s.id},
		{"jenis_transaksi", "umrah"},
		{"kurs_harga_paket", "idr"},
		{"is_quotation", "0"},
	}
	fields = append(fields, extra...)
	fields = append(fields, [2]string{"browser_url", s.referer}, [2]string{"client_timestamp", strconv.FormatInt(time.Now().Unix(), 10)})

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		for _, f := range fields {
			_ = w.WriteField(f[0], f[1])
		}
		w.Close()
		req, _ := http.NewRequest("POST", s.site.Base+"/transaksi/paket-umrah/"+endpoint, &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("User-Agent", hamdanEHUA)
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		req.Header.Set("Referer", s.referer)
		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		b, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("%s: HTTP %d", endpoint, resp.StatusCode)
			continue
		}
		body := string(b)
		if strings.Contains(body, "404 Error! Page not found") {
			lastErr = fmt.Errorf("%s: platform 404", endpoint)
			continue
		}
		return body, nil
	}
	return "", lastErr
}

// ---------------------------------------------------------------- listing

type hamdanEHCard struct {
	URL  string
	Name string
}

func hamdanEHListCards(site hamdanEHSite) ([]hamdanEHCard, error) {
	client := hamdanEHNewClient()
	var cards []hamdanEHCard
	seen := map[string]bool{}
	for page := 1; page <= hamdanEHMaxPages; page++ {
		pageURL := site.ListURL
		if page > 1 {
			pu, err := url.Parse(site.ListURL)
			if err != nil {
				break
			}
			q := pu.Query()
			q.Set("page", strconv.Itoa(page))
			pu.RawQuery = q.Encode()
			pageURL = pu.String()
		}
		log.Println("[crawler] visiting:", pageURL)
		html, err := hamdanEHGet(client, pageURL)
		if err != nil {
			if page == 1 {
				return nil, err
			}
			log.Printf("[crawler] %s list page %d: %v\n", site.TravelName, page, err)
			break
		}
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			return nil, err
		}
		newLinks := 0
		doc.Find("div.list-paket-umrah").Each(func(_ int, card *goquery.Selection) {
			title := card.Find("a.paket-title").First()
			href, _ := title.Attr("href")
			if !hamdanEHDetailRe.MatchString(href) {
				return
			}
			if strings.HasPrefix(href, "/") {
				href = site.Base + href
			}
			if seen[href] {
				return
			}
			seen[href] = true
			newLinks++
			// Cards without any published date ("Tanggal Belum Tersedia") only
			// carry past/sold-out departures; skip them to save requests.
			dateCell := ""
			card.Find("table tr").Each(func(_ int, tr *goquery.Selection) {
				if icon, _ := tr.Find("i").Attr("class"); strings.Contains(icon, "fa-calendar") {
					dateCell = tr.Find("td").Last().Text()
				}
			})
			if !hamdanEHDateRe.MatchString(dateCell) {
				return
			}
			cards = append(cards, hamdanEHCard{URL: href, Name: hamdanEHClean(title.Text())})
		})
		if newLinks == 0 {
			break
		}
	}
	return cards, nil
}

// ---------------------------------------------------------------- per departure

type hamdanEHDep struct {
	program      int
	kid          string
	date         time.Time
	ret          time.Time
	retFromFlt   bool
	airport      string
	seats        int
	full         bool
	quad         int
	triple       int
	double       int
	minRoom      int
	hotelMakkah  string
	hotelMadinah string
	starsMakkah  int
	starsMadinah int
	airline      string
	direct       *bool
	transitVia   string
}

func (d *hamdanEHDep) price() int {
	if d.quad > 0 {
		return d.quad
	}
	return d.minRoom
}

func hamdanEHParseOptions(html string) [][2]string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<select>" + html + "</select>"))
	if err != nil {
		return nil
	}
	var out [][2]string
	doc.Find("option").Each(func(_ int, o *goquery.Selection) {
		v, _ := o.Attr("value")
		if v = strings.TrimSpace(v); v != "" && v != "0" {
			out = append(out, [2]string{v, hamdanEHClean(o.Text())})
		}
	})
	return out
}

func hamdanEHParsePrices(html string, d *hamdanEHDep) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return
	}
	doc.Find(".m-jenis-kamar-input").Each(func(_ int, s *goquery.Selection) {
		hv, _ := s.Find("input.harga-kamar").Attr("value")
		jv, _ := s.Find("input.jenis-kamar").Attr("value")
		f, err := strconv.ParseFloat(strings.TrimSpace(hv), 64)
		if err != nil || f <= 0 {
			return
		}
		p := int(f)
		switch strings.TrimSpace(jv) {
		case "4":
			d.quad = p
		case "3":
			d.triple = p
		case "2":
			d.double = p
		}
		if d.minRoom == 0 || p < d.minRoom {
			d.minRoom = p
		}
	})
}

func hamdanEHStars(s *goquery.Selection) int {
	return s.Find(".star-rating i.fa-star.full").Length()
}

func hamdanEHAirlineName(raw string) string {
	var parts []string
	seen := map[string]bool{}
	for _, p := range strings.Split(raw, ",") {
		p = hamdanEHClean(p)
		if p == "" || p == "-" {
			continue
		}
		if a := KnownAirline(p); a != "" {
			p = a
		}
		if !seen[p] {
			seen[p] = true
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, ", ")
}

type hamdanEHPoint struct {
	date    string
	airport string
	airline string
}

// Carriers whose Indonesia<->Saudi service always stops at their hub; a single
// published leg on them is incomplete data, not a direct flight.
var hamdanEHHubCarriers = []string{"qatar", "emirates", "etihad", "oman", "turkish", "flydubai", "air arabia", "gulf air", "kuwait", "srilankan", "malaysia", "airasia", "scoot", "singapore", "egyptair", "royal brunei", "indigo", "air india"}

// hamdanEHLegs reads one direction's flight points (take-off/landing pairs).
// ok=false when any airport is missing. For inbound legs the order is reversed
// (Saudi -> Indonesia).
func hamdanEHLegs(pts []hamdanEHPoint, inbound bool) (direct bool, via string, ok bool) {
	n := len(pts)
	if n < 2 || n%2 != 0 {
		return false, "", false
	}
	for _, p := range pts {
		if p.airport == "" {
			return false, "", false
		}
	}
	if n == 2 {
		from, to := pts[0].airport, pts[1].airport
		if inbound {
			from, to = to, from
		}
		if !hamdanEHSaudi[to] || hamdanEHSaudi[from] {
			return false, "", false
		}
		lower := strings.ToLower(pts[0].airline)
		for _, h := range hamdanEHHubCarriers {
			if strings.Contains(lower, h) {
				return false, "", false
			}
		}
		return true, "", true
	}
	for i := 1; i < n-1; i += 2 {
		if ap := pts[i].airport; !hamdanEHSaudi[ap] {
			if city, found := hamdanEHCities[ap]; found {
				return false, city, true
			}
			return false, ap, true
		}
	}
	return false, "", false
}

func hamdanEHParseInfo(html string, d *hamdanEHDep) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return
	}
	doc.Find(".info-paket-min").Each(func(_ int, s *goquery.Selection) {
		title := strings.ToLower(hamdanEHClean(s.Find(".title").First().Text()))
		content := s.Find(".content").First()
		name := hamdanEHClean(content.Find("a").First().Text())
		if name == "" {
			name = hamdanEHClean(content.Text())
		}
		switch {
		case strings.Contains(title, "makkah") || strings.Contains(title, "mekkah"):
			if name != "" && name != "-" {
				d.hotelMakkah = name
				d.starsMakkah = hamdanEHStars(content)
			}
		case strings.Contains(title, "madinah"):
			if name != "" && name != "-" {
				d.hotelMadinah = name
				d.starsMadinah = hamdanEHStars(content)
			}
		case strings.Contains(title, "maskapai"):
			d.airline = hamdanEHAirlineName(hamdanEHClean(content.Text()))
		}
	})

	var outbound, inbound []hamdanEHPoint
	doc.Find(".list-info-maskapai").Each(func(_ int, s *goquery.Selection) {
		dir := strings.ToLower(hamdanEHClean(s.Find(".lipd-arah").First().Text()))
		airline := hamdanEHClean(s.Find(".lipdi-title-1").First().Text())
		var pts []hamdanEHPoint
		s.Find(".lipd-sektor-penerbangan").Each(func(_ int, p *goquery.Selection) {
			ap := hamdanEHClean(p.Find(".lipds-bandara").Text())
			code := ""
			if m := hamdanEHCodeRe.FindStringSubmatch(ap); len(m) > 1 {
				code = m[1]
			}
			pts = append(pts, hamdanEHPoint{date: hamdanEHClean(p.Find(".lipds-tanggal").Text()), airport: code, airline: airline})
		})
		switch {
		case strings.Contains(dir, "berangkat"):
			outbound = append(outbound, pts...)
		case strings.Contains(dir, "kembali") || strings.Contains(dir, "pulang"):
			inbound = append(inbound, pts...)
		}
	})

	// Direct vs transit, only from fully published flight legs.
	outDirect, outVia, outOK := hamdanEHLegs(outbound, false)
	if outOK {
		inDirect, _, inOK := hamdanEHLegs(inbound, true)
		switch {
		case !outDirect:
			f := false
			d.direct, d.transitVia = &f, outVia
		case len(inbound) == 0 || (inOK && inDirect):
			t := true
			d.direct = &t
		case inOK && !inDirect:
			f := false // direct out, transit back
			d.direct = &f
		}
	}

	// Return date: landing date of the last inbound leg.
	if n := len(inbound); n > 0 {
		if t, ok := hamdanEHFlightDate(inbound[n-1].date, d.date); ok {
			d.ret = t
			d.retFromFlt = true
		}
	}
}

// ---------------------------------------------------------------- per package

type hamdanEHDetail struct {
	name       string
	facilities []string
	excludes   []string
	dp         int
	guide      string
	deps       []*hamdanEHDep
}

func hamdanEHUniqueAppend(list []string, seen map[string]bool, v string) []string {
	v = hamdanEHClean(v)
	if v == "" || seen[strings.ToLower(v)] {
		return list
	}
	seen[strings.ToLower(v)] = true
	return append(list, v)
}

func hamdanEHParseDP(text string) int {
	for _, m := range hamdanEHDPRe.FindAllStringSubmatch(text, -1) {
		num := m[1]
		unit := strings.ToLower(m[2])
		var v int
		switch unit {
		case "jt", "juta":
			f, err := strconv.ParseFloat(strings.ReplaceAll(num, ",", "."), 64)
			if err != nil {
				continue
			}
			v = int(f * 1000000)
		case "rb", "ribu":
			f, err := strconv.ParseFloat(strings.ReplaceAll(num, ",", "."), 64)
			if err != nil {
				continue
			}
			v = int(f * 1000)
		default:
			// "10.000.000" / "10.000.000,00"
			if i := strings.LastIndex(num, ","); i > 0 && len(num)-i == 3 {
				num = num[:i]
			}
			v, _ = strconv.Atoi(strings.NewReplacer(".", "", ",", "").Replace(num))
		}
		if v >= 1000000 && v <= 100000000 {
			return v
		}
	}
	return 0
}

func hamdanEHFetchDetail(site hamdanEHSite, card hamdanEHCard) (*hamdanEHDetail, error) {
	client := hamdanEHNewClient()
	log.Println("[crawler] visiting:", card.URL)
	html, err := hamdanEHGet(client, card.URL)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}
	form := doc.Find("#form-pesan-paket-umrah-haji").First()
	sess := &hamdanEHSession{client: client, site: site, referer: card.URL}
	sess.token, _ = form.Find("input[name=_token]").Attr("value")
	if sess.token == "" {
		sess.token, _ = doc.Find("meta[name=csrf-token]").Attr("content")
	}
	sess.id, _ = form.Find("input[name=id]").Attr("value")
	if sess.id == "" {
		if m := hamdanEHDetailRe.FindStringSubmatch(card.URL); len(m) > 1 {
			sess.id = m[1]
		}
	}

	det := &hamdanEHDetail{name: card.Name}
	if det.name == "" {
		det.name = hamdanEHClean(doc.Find("title").First().Text())
	}

	// Facilities (the detail page renders the accordion twice: desktop + mobile).
	seenF, seenX := map[string]bool{}, map[string]bool{}
	doc.Find(".block-list-fasilitas").Each(func(_ int, b *goquery.Selection) {
		head := strings.ToUpper(hamdanEHClean(b.Find("b").First().Text()))
		exclude := strings.Contains(head, "TIDAK") || strings.Contains(head, "TAMBAHAN") || strings.Contains(head, "DILUAR") || strings.Contains(head, "DI LUAR") || strings.Contains(head, "EXCLUDE")
		b.Find("li").Each(func(_ int, li *goquery.Selection) {
			if exclude {
				det.excludes = hamdanEHUniqueAppend(det.excludes, seenX, li.Text())
			} else {
				det.facilities = hamdanEHUniqueAppend(det.facilities, seenF, li.Text())
			}
		})
	})

	// Down payment: stated in "Syarat & Ketentuan" / "Persyaratan Peserta".
	var panelText strings.Builder
	doc.Find(".front-accordion .collapse-inner").Each(func(_ int, s *goquery.Selection) {
		panelText.WriteString(s.Text())
		panelText.WriteString("\n")
	})
	det.dp = hamdanEHParseDP(panelText.String())

	if m := hamdanEHGuideRe.FindStringSubmatch(det.name); len(m) > 1 {
		det.guide = strings.TrimSpace(m[1])
	}

	// Programs ("Umrah 9 Hari").
	var programs []int
	form.Find("select[name=program_hari] option").Each(func(_ int, o *goquery.Selection) {
		v, _ := o.Attr("value")
		if p, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && p > 0 && p < 60 {
			programs = append(programs, p)
		}
	})

	today := hamdanEHToday()
	for _, prog := range programs {
		pf := [2]string{"program_hari", strconv.Itoa(prog)}
		bHTML, err := sess.post("load-bandara-keberangkatan", [][2]string{pf})
		if err != nil {
			log.Printf("[crawler] %s %s bandara: %v\n", site.TravelName, card.URL, err)
			continue
		}
		for _, b := range hamdanEHParseOptions(bHTML) {
			airport := ""
			if m := hamdanEHAirportRe.FindStringSubmatch(b[1]); len(m) > 1 {
				airport = m[1]
			}
			bf := [2]string{"bandara_id", b[0]}
			tHTML, err := sess.post("load-tanggal-keberangkatan", [][2]string{pf, bf})
			if err != nil {
				log.Printf("[crawler] %s %s tanggal: %v\n", site.TravelName, card.URL, err)
				continue
			}
			for _, o := range hamdanEHParseOptions(tHTML) {
				dt, ok := hamdanEHParseDate(o[1])
				if !ok || dt.Before(today) {
					continue
				}
				d := &hamdanEHDep{program: prog, kid: o[0], date: dt, airport: airport}
				lower := strings.ToLower(o[1])
				if strings.Contains(lower, "(habis)") || strings.Contains(lower, "penuh") || strings.Contains(lower, "sold") {
					d.full = true
				} else if m := hamdanEHSeatRe.FindStringSubmatch(o[1]); len(m) > 1 {
					d.seats, _ = strconv.Atoi(m[1])
					if d.seats == 0 {
						d.full = true
					}
				}
				kf := [2]string{"keberangkatan_id", o[0]}
				if h, err := sess.post("load-harga-paket", [][2]string{pf, bf, kf}); err == nil {
					hamdanEHParsePrices(h, d)
				} else {
					log.Printf("[crawler] %s %s harga %s: %v\n", site.TravelName, card.URL, o[0], err)
				}
				if h, err := sess.post("get-informasi-maskapai-hotel", [][2]string{pf, bf, kf}); err == nil {
					hamdanEHParseInfo(h, d)
				} else {
					log.Printf("[crawler] %s %s info %s: %v\n", site.TravelName, card.URL, o[0], err)
				}
				if d.ret.IsZero() {
					d.ret = dt.AddDate(0, 0, prog-1)
				}
				det.deps = append(det.deps, d)
			}
		}
	}
	return det, nil
}

// ---------------------------------------------------------------- build packages

// hamdanEHMerge combines several platform departures that share a date (the
// travel sometimes opens a second group on the same day): remaining seats are
// added up, the cheapest group supplies price/hotels.
func hamdanEHMerge(group []*hamdanEHDep) *hamdanEHDep {
	sort.SliceStable(group, func(i, j int) bool {
		pi, pj := group[i].price(), group[j].price()
		if group[i].full != group[j].full {
			return !group[i].full
		}
		if pi == 0 || pj == 0 {
			return pi != 0
		}
		return pi < pj
	})
	m := *group[0]
	m.seats, m.full = 0, true
	for _, g := range group {
		m.seats += g.seats
		if !g.full {
			m.full = false
		}
	}
	if m.seats > 0 {
		m.full = false
	}
	return &m
}

func hamdanEHBuild(site hamdanEHSite, card hamdanEHCard, det *hamdanEHDetail) []CrawledPackage {
	if len(det.deps) == 0 {
		return nil
	}
	// Group by program: a detail page normally has one program ("Umrah 9 Hari"),
	// but some offer several; each becomes its own package so Duration is exact.
	byProg := map[int][]*hamdanEHDep{}
	var progs []int
	for _, d := range det.deps {
		if _, ok := byProg[d.program]; !ok {
			progs = append(progs, d.program)
		}
		byProg[d.program] = append(byProg[d.program], d)
	}
	// Main program: the one named in the title, else the one with most departures.
	titleDur := ParseDuration(det.name)
	sort.SliceStable(progs, func(i, j int) bool {
		a, b := progs[i], progs[j]
		if (a == titleDur) != (b == titleDur) {
			return a == titleDur
		}
		return len(byProg[a]) > len(byProg[b])
	})

	var out []CrawledPackage
	for idx, prog := range progs {
		// Merge same-date entries (same date + airport).
		groups := map[string][]*hamdanEHDep{}
		var keys []string
		for _, d := range byProg[prog] {
			k := d.date.Format("2006-01-02") + "|" + d.airport
			if _, ok := groups[k]; !ok {
				keys = append(keys, k)
			}
			groups[k] = append(groups[k], d)
		}
		var deps []*hamdanEHDep
		for _, k := range keys {
			deps = append(deps, hamdanEHMerge(groups[k]))
		}
		// The import keys departures by (package, date), so if the same date is
		// offered from two airports only the cheapest available one is kept.
		perDate := map[string][]*hamdanEHDep{}
		var dates []string
		for _, d := range deps {
			k := d.date.Format("2006-01-02")
			if _, ok := perDate[k]; !ok {
				dates = append(dates, k)
			}
			perDate[k] = append(perDate[k], d)
		}
		sort.Strings(dates)
		deps = deps[:0]
		for _, k := range dates {
			g := perDate[k]
			if len(g) > 1 {
				sort.SliceStable(g, func(i, j int) bool {
					if g[i].full != g[j].full {
						return !g[i].full
					}
					pi, pj := g[i].price(), g[j].price()
					if pi == 0 || pj == 0 {
						return pi != 0
					}
					return pi < pj
				})
			}
			deps = append(deps, g[0])
		}

		pkg := CrawledPackage{
			TravelName:  site.TravelName,
			PackageName: det.name,
			Duration:    prog,
			URL:         card.URL,
			Facilities:  det.facilities,
			Excludes:    det.excludes,
			DownPayment: det.dp,
			Guide:       det.guide,
		}
		if idx > 0 {
			// Secondary program on the same detail page.
			pkg.URL = fmt.Sprintf("%s#program-%d-hari", card.URL, prog)
			pkg.PackageName = fmt.Sprintf("%s (Program %d Hari)", det.name, prog)
		}

		// Package-level values come from the cheapest available departure.
		var best *hamdanEHDep
		for _, d := range deps {
			if d.price() == 0 {
				continue
			}
			if best == nil || (best.full && !d.full) || (best.full == d.full && d.price() < best.price()) {
				best = d
			}
		}
		if best == nil {
			best = deps[0]
		}
		pkg.Price = best.price()
		pkg.PriceTriple = best.triple
		pkg.PriceDouble = best.double
		pkg.HotelMakkah, pkg.HotelMadinah = best.hotelMakkah, best.hotelMadinah
		pkg.StarsMakkah, pkg.StarsMadinah = best.starsMakkah, best.starsMadinah
		pkg.Airline = best.airline
		pkg.Airport = best.airport
		pkg.Direct, pkg.TransitVia = best.direct, best.transitVia

		allFull := true
		for _, d := range deps {
			cd := CrawledDeparture{
				Date:       d.date.Format("2006-01-02"),
				ReturnDate: d.ret.Format("2006-01-02"),
				Seats:      d.seats,
				Full:       d.full,
			}
			if d.price() != pkg.Price || d.triple != pkg.PriceTriple || d.double != pkg.PriceDouble {
				cd.Price, cd.PriceTriple, cd.PriceDouble = d.price(), d.triple, d.double
			}
			if d.hotelMakkah != pkg.HotelMakkah {
				cd.HotelMakkah = d.hotelMakkah
			}
			if d.hotelMadinah != pkg.HotelMadinah {
				cd.HotelMadinah = d.hotelMadinah
			}
			if d.airline != pkg.Airline {
				cd.Airline = d.airline
			}
			if d.airport != pkg.Airport {
				cd.Airport = d.airport
			}
			if !d.full {
				allFull = false
				pkg.Seats += d.seats
			}
			pkg.Departures = append(pkg.Departures, cd)
			pkg.DepartureDates = append(pkg.DepartureDates, d.date.Format("02 Jan 2006"))
		}
		pkg.Full = allFull
		// Package-level direct/transit only if every departure agrees.
		for _, d := range deps {
			if (d.direct == nil) != (pkg.Direct == nil) || (d.direct != nil && *d.direct != *pkg.Direct) {
				pkg.Direct, pkg.TransitVia = nil, ""
				break
			}
		}
		out = append(out, pkg)
	}
	return out
}

// hamdanEHCrawl runs the whole platform crawl for one site.
func hamdanEHCrawl(site hamdanEHSite) ([]CrawledPackage, error) {
	cards, err := hamdanEHListCards(site)
	if err != nil {
		return nil, err
	}
	return hamdanEHCrawlCards(site, cards), nil
}

func hamdanEHCrawlCards(site hamdanEHSite, cards []hamdanEHCard) []CrawledPackage {
	workers := site.Workers
	if workers <= 0 {
		workers = 4
	}
	results := make([][]CrawledPackage, len(cards))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				det, err := hamdanEHFetchDetail(site, cards[i])
				if err != nil {
					log.Printf("[crawler] %s detail %s: %v\n", site.TravelName, cards[i].URL, err)
					continue
				}
				results[i] = hamdanEHBuild(site, cards[i], det)
			}
		}()
	}
	for i := range cards {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	var out []CrawledPackage
	seen := map[string]bool{}
	same := map[string]int{} // identical offer -> index in out
	for _, r := range results {
		for _, p := range r {
			if p.PackageName == "" || seen[p.URL] {
				continue
			}
			seen[p.URL] = true
			// Some travels (Labbaika) open a new detail page per batch of dates
			// for the very same offer. When everything the user sees is identical
			// except the dates, show it once with all dates.
			key := strings.Join([]string{p.PackageName, strconv.Itoa(p.Duration), p.HotelMakkah, p.HotelMadinah,
				p.Airline, p.Airport, strconv.Itoa(p.Price), strconv.Itoa(p.PriceTriple), strconv.Itoa(p.PriceDouble),
				strings.Join(p.Facilities, ";"), strings.Join(p.Excludes, ";"), strconv.Itoa(p.DownPayment)}, "|")
			if i, ok := same[key]; ok {
				out[i] = hamdanEHMergePackages(out[i], p)
				continue
			}
			same[key] = len(out)
			out = append(out, p)
		}
	}
	return out
}

func hamdanEHMergePackages(a, b CrawledPackage) CrawledPackage {
	have := map[string]bool{}
	for _, d := range a.Departures {
		have[d.Date] = true
	}
	for _, d := range b.Departures {
		if !have[d.Date] {
			have[d.Date] = true
			a.Departures = append(a.Departures, d)
		}
	}
	sort.SliceStable(a.Departures, func(i, j int) bool { return a.Departures[i].Date < a.Departures[j].Date })
	a.DepartureDates = nil
	a.Seats, a.Full = 0, true
	for _, d := range a.Departures {
		if t, err := time.Parse("2006-01-02", d.Date); err == nil {
			a.DepartureDates = append(a.DepartureDates, t.Format("02 Jan 2006"))
		}
		if !d.Full {
			a.Full = false
			a.Seats += d.Seats
		}
	}
	if (a.Direct == nil) != (b.Direct == nil) || (a.Direct != nil && *a.Direct != *b.Direct) || a.TransitVia != b.TransitVia {
		a.Direct, a.TransitVia = nil, ""
	}
	return a
}
