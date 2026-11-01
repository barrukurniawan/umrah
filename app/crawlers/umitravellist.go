package crawlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

// UmiTravelListParser crawls UMI Tour & Travel (umi.travel, a MuslimPergi
// site). The paginated listing (https://umi.travel/listings?page=N) links to
// one product page per departure. A product page states:
//   - header: "Tipe Paket", "Durasi N Hari", "Maskapai <name|N/A>", <CITY>, <date>
//   - "Harga Paket" per room type (also in JSON-LD offers: [QUAD]/[TRIPLE]/[DOUBLE])
//   - "Total Pax" / "Tersisa Pax" (seats), "Pembimbing Jamaah:", DP
//     ("Uang muka minimum paket ini adalah Rp..."), "Harga Termasuk:" /
//     "Harga Tidak Termasuk:" lists, "Transportasi"
//   - a dated day-by-day itinerary (flight numbers, route) and hotel cards
//     (name, star icons, city, check-in/out).
//
// The return date is the last itinerary day when the itinerary starts on the
// departure date. Without an itinerary it is departure + Durasi - 1: on every
// UMI page with both, the last itinerary day is exactly that (e.g. 24 Nov 2026,
// 9 Hari -> 2 Dec 2026). When the itinerary starts on another day than the
// header date the page contradicts itself and the return date is left empty.
type UmiTravelListParser struct {
	URL string // listing URL, e.g. https://umi.travel/listings
}

func NewUmiTravelListParser(listURL string) *UmiTravelListParser {
	return &UmiTravelListParser{URL: listURL}
}

var umiListLinkRe = regexp.MustCompile(`href="((?:https?://(?:www\.)?umi\.travel)?/listings/[a-z0-9][a-z0-9-]*)"`)

const umiListMaxPages = 30

func (u *UmiTravelListParser) Crawl() ([]CrawledPackage, error) {
	listURL := u.URL
	if listURL == "" {
		listURL = "https://umi.travel/listings"
	}
	base, err := url.Parse(listURL)
	if err != nil {
		return nil, err
	}

	// 1. Discover product URLs, following ?page=N until a page adds nothing new.
	seen := map[string]bool{}
	var productURLs []string
	for page := 1; page <= umiListMaxPages; page++ {
		pageURL := listURL
		if page > 1 {
			q := base.Query()
			q.Set("page", fmt.Sprintf("%d", page))
			pu := *base
			pu.RawQuery = q.Encode()
			pageURL = pu.String()
		}
		log.Println("[crawler] visiting:", pageURL)
		html, err := lstFetchPolite(pageURL)
		if err != nil {
			if page == 1 {
				return nil, err
			}
			log.Printf("[crawler] error on %s: %v\n", pageURL, err)
			break
		}
		added := 0
		for _, m := range umiListLinkRe.FindAllStringSubmatch(html, -1) {
			ref, err := base.Parse(m[1])
			if err != nil {
				continue
			}
			abs := ref.String()
			if seen[abs] {
				continue
			}
			seen[abs] = true
			if !umiIsUmrahSlug(abs) {
				continue
			}
			productURLs = append(productURLs, abs)
			added++
		}
		if added == 0 {
			break
		}
		time.Sleep(lstPoliteDelay)
	}
	log.Printf("[crawler] umi.travel: %d product links discovered\n", len(productURLs))

	// 2. Scrape each product page.
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
		p, tipe, ok := parseUmiProduct(html, pu)
		lt := strings.ToLower(tipe)
		if tipe != "" && !strings.Contains(lt, "umro") && !strings.Contains(lt, "umra") {
			continue
		}
		if !ok {
			log.Printf("[crawler] umi.travel: no package data on %s\n", pu)
			continue
		}
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

// umiIsUmrahSlug keeps umrah/umroh products and skips hajj / halal-tour listings.
func umiIsUmrahSlug(u string) bool {
	slug := strings.ToLower(u[strings.LastIndex(u, "/")+1:])
	if !strings.Contains(slug, "umro") && !strings.Contains(slug, "umra") {
		return false
	}
	return !strings.Contains(slug, "haji") && !strings.Contains(slug, "hajj")
}

var (
	umiLdRe      = regexp.MustCompile(`(?is)<script[^>]*application/ld\+json[^>]*>(.*?)</script>`)
	umiH4Re      = regexp.MustCompile(`(?is)<h4[^>]*>\s*(.*?)\s*</h4>`)
	umiStarRe    = regexp.MustCompile(`class="fa fa-star"`)
	umiDurRe     = regexp.MustCompile(`(?i)^(\d{1,2})\s*hari$`)
	umiPaxRe     = regexp.MustCompile(`^\d{1,4}$`)
	umiDayNumRe  = regexp.MustCompile(`^\d{1,2}$`)
	umiFlightRe  = regexp.MustCompile(`\b(SV|GA|QR|EK|EY|TK|WY|MS|6E|HU|FZ|XY)\s?(\d{2,4})\b`)
	umiTakeOffRe = regexp.MustCompile(`(?i)take\s*off\b.*?\b(?:menuju|ke|to)\s+([A-Za-z]+(?:\s+[A-Z][a-z]+)?)`)
	umiDPRe      = regexp.MustCompile(`(?i)^\W*(?:pembayaran\s+)?dp\s+((?:rp\.?\s*)?\d[\d.,]*\s*(?:juta|jt)?)`)
	umiArabicRe  = regexp.MustCompile(`[\x{0600}-\x{06FF}]+`)
)

type umiLdProduct struct {
	Type   string `json:"@type"`
	Name   string `json:"name"`
	Offers []struct {
		Price json.Number `json:"price"`
		Name  string      `json:"name"`
	} `json:"offers"`
}

// parseUmiProduct parses one product page. It returns the package, the
// "Tipe Paket" value and whether a usable package (date + price) was found.
func parseUmiProduct(html, pageURL string) (CrawledPackage, string, bool) {
	p := CrawledPackage{TravelName: "UMI Tour & Travel", URL: pageURL}
	var dep CrawledDeparture
	lines := lstHTMLLines(html)

	// --- JSON-LD: name + room prices ---
	for _, m := range umiLdRe.FindAllStringSubmatch(html, -1) {
		var g struct {
			Graph []json.RawMessage `json:"@graph"`
		}
		var nodes []json.RawMessage
		if json.Unmarshal([]byte(m[1]), &g) == nil && len(g.Graph) > 0 {
			nodes = g.Graph
		} else {
			nodes = []json.RawMessage{json.RawMessage(m[1])}
		}
		for _, n := range nodes {
			var prod umiLdProduct
			if json.Unmarshal(n, &prod) != nil || prod.Type != "Product" {
				continue
			}
			if p.PackageName == "" {
				p.PackageName = strings.TrimSpace(prod.Name)
			}
			for _, o := range prod.Offers {
				v, _ := o.Price.Int64()
				switch strings.ToUpper(strings.TrimSpace(o.Name)) {
				case "[QUAD]", "QUAD":
					p.Price = int(v)
				case "[TRIPLE]", "TRIPLE":
					p.PriceTriple = int(v)
				case "[DOUBLE]", "DOUBLE":
					p.PriceDouble = int(v)
				}
			}
		}
	}
	if p.PackageName == "" {
		p.PackageName = lstOgTitle(html)
	}

	tipe := ""
	dpText, dpFAQ := 0, 0
	itStart, itEnd := -1, -1
	for i, l := range lines {
		next := ""
		if i+1 < len(lines) {
			next = lines[i+1]
		}
		switch l {
		case "Tipe Paket":
			if tipe == "" {
				tipe = next
			}
		case "Durasi":
			if m := umiDurRe.FindStringSubmatch(next); m != nil && p.Duration == 0 {
				fmt.Sscanf(m[1], "%d", &p.Duration)
			}
		case "Maskapai":
			// Maskapai / <airline|N/A> / <CITY> / <date>
			if dep.Date == "" && i+3 < len(lines) {
				if t, ok := lstParseDate(lines[i+3]); ok {
					dep.Date = t.Format(lstISO)
					p.Airport = lstCity(lines[i+2])
					if p.Airport == "" {
						p.Airport = strings.Title(strings.ToLower(lines[i+2]))
					}
					if a := lstAirlines(next); a != "" {
						p.Airline = a
					}
				}
			}
		case "Mulai dari":
			if p.Price == 0 {
				p.Price = lstMoney(next)
			}
		case "Total Pax":
			if umiPaxRe.MatchString(next) {
				fmt.Sscanf(next, "%d", &dep.SeatsTotal)
			}
		case "Tersisa Pax":
			if umiPaxRe.MatchString(next) {
				var n int
				fmt.Sscanf(next, "%d", &n)
				dep.Seats = n
				if n == 0 {
					dep.Full = true
				}
			}
		case "Pembimbing Jamaah:":
			var names []string
			for _, g := range lstListItems(lines, i+1, func(s string) bool { return strings.HasSuffix(s, ":") }) {
				g = strings.TrimSpace(umiArabicRe.ReplaceAllString(g, ""))
				if g != "" {
					names = append(names, g)
				}
			}
			p.Guide = strings.Join(names, " / ")
		case "Harga Termasuk:":
			p.Facilities = lstListItems(lines, i+1, func(s string) bool { return strings.HasSuffix(s, ":") })
		case "Harga Tidak Termasuk:":
			p.Excludes = lstListItems(lines, i+1, func(s string) bool { return strings.HasSuffix(s, ":") })
		case "Transportasi":
			// "Transportasi" / <train or bus description> (pax-info box)
			if next != "" && next != "Informasi Paket" && !strings.HasSuffix(next, ":") && len(next) < 80 {
				p.Facilities = append(p.Facilities, "Transportasi: "+next)
			}
		case "Itinerari":
			itStart = i + 1
		}
		// DP: the travel's own "PENDAFTARAN & PEMBAYARAN" text ("DP 5 Juta /
		// orang") wins over the platform FAQ ("Uang muka minimum ... Rp1.000.000").
		if m := umiDPRe.FindStringSubmatch(l); m != nil && dpText == 0 {
			dpText = lstMoney(m[1])
		}
		if strings.HasPrefix(l, "Uang muka minimum paket ini adalah") && dpFAQ == 0 {
			dpFAQ = lstMoney(l + " " + next)
		}
		if strings.Contains(l, "kursi telah habis") {
			dep.Full = true
		}
		if itStart >= 0 && itEnd < 0 && i > itStart && (l == "Hotel" || l == "Informasi Paket") {
			itEnd = i
		}
	}
	if dep.Seats > 0 {
		dep.Full = false
	}
	p.DownPayment = dpText
	if p.DownPayment == 0 {
		p.DownPayment = dpFAQ
	}
	p.Facilities = lstDedupStrings(p.Facilities)

	// --- Itinerary: numbered days with dates, flights, route ---
	var itDates []time.Time
	var itText []string
	if itStart >= 0 {
		if itEnd < 0 {
			itEnd = len(lines)
		}
		for i := itStart; i < itEnd; i++ {
			itText = append(itText, lines[i])
			if umiDayNumRe.MatchString(lines[i]) && i+1 < itEnd {
				if t, ok := lstParseDate(lines[i+1]); ok {
					itDates = append(itDates, t)
				}
			}
		}
	}
	depT, depOK := time.Parse(lstISO, dep.Date)
	switch {
	case depOK != nil:
	case len(itDates) >= 2:
		if itDates[0].Equal(depT) {
			dep.ReturnDate = itDates[len(itDates)-1].Format(lstISO)
		}
	case p.Duration > 1:
		dep.ReturnDate = depT.AddDate(0, 0, p.Duration-1).Format(lstISO)
	}
	if p.Airline == "" {
		// "Maskapai: N/A" but the itinerary names the flights ("by SV 819").
		var names []string
		seen := map[string]bool{}
		for _, l := range itText {
			for _, m := range umiFlightRe.FindAllStringSubmatch(l, -1) {
				if n := lstAirlineCodes[m[1]]; n != "" && !seen[n] {
					seen[n] = true
					names = append(names, n)
				}
			}
		}
		p.Airline = strings.Join(names, " / ")
	}

	// --- Hotel cards ---
	plusTour := false
	for _, blk := range strings.Split(html, `<div class="hotel-info-item">`)[1:] {
		if k := strings.Index(blk, "Check Out"); k > 0 {
			blk = blk[:k]
		}
		h := umiH4Re.FindStringSubmatch(blk)
		if h == nil {
			continue
		}
		name := strings.TrimSpace(lstUnescape(lstTagRe.ReplaceAllString(h[1], "")))
		rateEnd := strings.Index(blk, "Lokasi")
		if rateEnd < 0 {
			continue
		}
		stars := len(umiStarRe.FindAllString(blk[:rateEnd], -1))
		bl := lstHTMLLines(blk[rateEnd:])
		city := ""
		if len(bl) > 1 {
			city = strings.ToLower(strings.Trim(bl[1], " :"))
			if strings.HasPrefix(strings.ToLower(bl[0]), "lokasi :") && len(bl[0]) > 9 {
				city = strings.ToLower(strings.TrimSpace(bl[0][9:]))
			}
		}
		if stars > 5 {
			stars = 0
		}
		switch city {
		case "makkah", "mekkah", "mekah":
			if p.HotelMakkah == "" {
				p.HotelMakkah, p.StarsMakkah = name, stars
			}
		case "madinah":
			if p.HotelMadinah == "" {
				p.HotelMadinah, p.StarsMadinah = name, stars
			}
		case "":
		default:
			plusTour = true // Dubai, Istanbul, Bursa, ...
		}
	}

	// --- Direct / transit, only for plain umrah (no tour stop) ---
	if !plusTour {
		for _, l := range itText {
			m := umiTakeOffRe.FindStringSubmatch(l)
			if m == nil {
				continue
			}
			dest := strings.ToLower(m[1])
			switch {
			case strings.HasPrefix(dest, "jeddah"), strings.HasPrefix(dest, "madinah"), dest == "jed", dest == "med":
				v := true
				p.Direct = &v
			default:
				v := false
				p.Direct = &v
				p.TransitVia = strings.Title(strings.ToLower(strings.Fields(m[1])[0]))
			}
			break // the first take-off is the outbound flight
		}
	}

	if dep.Date == "" || p.Price == 0 {
		return p, tipe, false
	}
	p.Departures = []CrawledDeparture{dep}
	return p, tipe, true
}
