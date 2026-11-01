package crawlers

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

// KhasanahParser scrapes Khasanah Travel (Next.js storefront, server-rendered).
// The /trip listing contains every trip card (name, date range, lowest price,
// DP, seats); each trip is one departure with its own detail page, which is
// visited for hotels (name, stars, distance as written in the hotel card),
// room prices, itinerary (departure airport, transit/direct, day count) and
// the "Termasuk" / "Tidak Termasuk" lists. URL may be the homepage or /trip.
type KhasanahParser struct {
	URL string
}

var (
	khasanahDateRe   = regexp.MustCompile(`(\d{1,2})\s+([A-Za-z]+)\s+(\d{4})`)
	khasanahSeatsRe  = regexp.MustCompile(`(?i)sisa\s*(\d+)\s*seat`)
	khasanahDPRe     = regexp.MustCompile(`(?i)\bDP\s*(Rp\s*[\d.]+)`)
	khasanahStarRe   = regexp.MustCompile(`(?i)bintang\s*(\d)`)
	khasanahDistRe   = regexp.MustCompile(`(?i)(?:jarak|sekitar)[^.\d]{0,20}?(\d[\d.]*)(?:\s*[-–]\s*(\d[\d.]*))?\s*(km|meter|m)\b`)
	khasanahDaysRe   = regexp.MustCompile(`(?i)\b(\d{1,2})\s*(?:hari|days?)\b`)
	khasanahSetaraf  = regexp.MustCompile(`(?i)\s*/\s*setaraf.*$|\s+setaraf\s+bintang.*$`)
	khasanahViaRe    = regexp.MustCompile(`(?i)(?:melalui|menuju)\s+([A-Z][A-Za-z]+(?:\s*(?:&|dan)\s*[A-Z][A-Za-z]+)*)\s*\(transit\)`)
	khasanahFacStars = regexp.MustCompile(`(?i)^hotel\b.*\*\s*\d`)
)

var khasanahAirlines = []struct {
	re   *regexp.Regexp
	name string
}{
	{regexp.MustCompile(`(?i)\bsaudia\b|\bsaudi arabian airlines\b`), "Saudia"},
	{regexp.MustCompile(`(?i)\bgaruda indonesia\b`), "Garuda Indonesia"},
	{regexp.MustCompile(`(?i)\bqatar airways\b`), "Qatar Airways"},
	{regexp.MustCompile(`(?i)\bemirates\b`), "Emirates"},
	{regexp.MustCompile(`(?i)\betihad\b`), "Etihad Airways"},
	{regexp.MustCompile(`(?i)\bturkish airlines\b`), "Turkish Airlines"},
	{regexp.MustCompile(`(?i)\boman air\b`), "Oman Air"},
	{regexp.MustCompile(`(?i)\bsingapore airlines\b`), "Singapore Airlines"},
	{regexp.MustCompile(`(?i)\bscoot\b`), "Scoot"},
	{regexp.MustCompile(`(?i)\blion air\b`), "Lion Air"},
	{regexp.MustCompile(`(?i)\bbatik air\b`), "Batik Air"},
	{regexp.MustCompile(`(?i)\bflynas\b`), "Flynas"},
}

var khasanahTrailCity = regexp.MustCompile(`\s+(?:makkah|mekkah|madinah)$`)

// khasanahHotel is one hotel card on a trip page.
type khasanahHotel struct {
	label, name, desc string
	city              string // "makkah" / "madinah" / "other" / ""
}

func (k *KhasanahParser) Crawl() ([]CrawledPackage, error) {
	// Always use the canonical host (www redirects here; the collector is restricted to it).
	const base = "https://khasanahtravel.com"
	listURL := base + "/trip"

	var packages []CrawledPackage
	seen := make(map[string]bool)
	cutoff := khasanahCutoff()

	c := NewCollector("khasanahtravel.com")
	detail := c.Clone()
	_ = detail.Limit(&colly.LimitRule{DomainGlob: "*", Delay: 300 * time.Millisecond})

	c.OnHTML(`a[href^="/trip/"]`, func(e *colly.HTMLElement) {
		href := e.Attr("href")
		name := CleanText(e.DOM.Find("h3").First().Text())
		if name == "" || seen[href] {
			return
		}

		// Date range: "19 Jan 2027 – 27 Jan 2027"
		var dateText string
		e.DOM.Find("p").EachWithBreak(func(_ int, s *goquery.Selection) bool {
			t := CleanText(s.Text())
			if khasanahDateRe.MatchString(t) && !strings.Contains(t, "Rp") {
				dateText = t
				return false
			}
			return true
		})
		dates := khasanahDateRe.FindAllString(dateText, -1)
		if len(dates) == 0 {
			return
		}
		start, ok := khasanahParseDate(dates[0])
		if !ok || start.Before(cutoff) {
			return
		}

		// Lowest price (quad); the "Bisa DP" line is the down payment.
		price := 0
		e.DOM.Find("p").EachWithBreak(func(_ int, s *goquery.Selection) bool {
			t := CleanText(s.Text())
			if strings.HasPrefix(t, "Rp") {
				price = ParsePrice(t)
				return false
			}
			return true
		})
		if price == 0 {
			return
		}

		seen[href] = true
		pkg := CrawledPackage{
			TravelName:     "Khasanah Travel",
			PackageName:    name,
			Price:          price,
			DepartureDates: []string{start.Format("2 Jan 2006")},
			URL:            base + href,
		}
		dep := CrawledDeparture{Date: start.Format("2006-01-02"), Price: price}
		if len(dates) > 1 {
			if end, ok := khasanahParseDate(dates[len(dates)-1]); ok && end.After(start) {
				dep.ReturnDate = end.Format("2006-01-02")
			}
		}
		text := strings.Join(strings.Fields(e.Text), " ")
		if m := khasanahSeatsRe.FindStringSubmatch(text); len(m) > 1 {
			pkg.Seats, _ = strconv.Atoi(m[1])
		}
		lower := strings.ToLower(text)
		if strings.Contains(lower, "sold out") || strings.Contains(lower, "penuh") || strings.Contains(lower, "habis") {
			pkg.Full, pkg.Seats = true, 0
		}
		if m := khasanahDPRe.FindStringSubmatch(text); m != nil {
			pkg.DownPayment = ParsePrice(m[1])
		}
		dep.Seats, dep.Full = pkg.Seats, pkg.Full
		pkg.Departures = []CrawledDeparture{dep}
		packages = append(packages, pkg)
	})

	if err := c.Visit(listURL); err != nil {
		return nil, err
	}

	for i := range packages {
		pkg := &packages[i]
		detail.OnHTMLDetach("body")
		detail.OnHTML("body", func(e *colly.HTMLElement) {
			khasanahFillDetail(e.DOM, pkg)
		})
		_ = detail.Visit(pkg.URL)
	}

	// Mirror package-level values into the single departure entry.
	for i := range packages {
		p := &packages[i]
		d := &p.Departures[0]
		d.PriceTriple, d.PriceDouble = p.PriceTriple, p.PriceDouble
		d.HotelMakkah, d.HotelMadinah = p.HotelMakkah, p.HotelMadinah
		d.Airline, d.Airport = p.Airline, p.Airport
	}
	return packages, nil
}

// khasanahSection returns the content block that follows the h2 whose text
// ends with title (e.g. "Hotel", "Itinerary").
func khasanahSection(body *goquery.Selection, title string) *goquery.Selection {
	var out *goquery.Selection
	body.Find("h2").EachWithBreak(func(_ int, h *goquery.Selection) bool {
		if strings.HasSuffix(CleanText(h.Text()), title) {
			out = h.Parent()
			return false
		}
		return true
	})
	return out
}

func khasanahFillDetail(body *goquery.Selection, pkg *CrawledPackage) {
	// ---- hotels ----
	var hotels []khasanahHotel
	if sec := khasanahSection(body, "Hotel"); sec != nil {
		sec.Find("div.p-4").Each(func(_ int, card *goquery.Selection) {
			name := CleanText(card.Find("p.font-bold").First().Text())
			if name == "" {
				return
			}
			h := khasanahHotel{name: name, desc: CleanText(card.Find("p").Eq(1).Text())}
			if i := strings.Index(name, " - "); i > 0 {
				h.label, h.name = strings.TrimSpace(name[:i]), strings.TrimSpace(name[i+3:])
			}
			h.city = khasanahHotelCity(h)
			h.name = strings.TrimSpace(khasanahSetaraf.ReplaceAllString(h.name, ""))
			// "Grand AL Massa makkah" -> "Grand AL Massa" (lower-case city suffix only)
			h.name = khasanahTrailCity.ReplaceAllString(h.name, "")
			hotels = append(hotels, h)
		})
	}
	// Two cards with one city identified: the other card is the other Holy City.
	if len(hotels) == 2 {
		a, b := &hotels[0], &hotels[1]
		if a.city == "" && (b.city == "makkah" || b.city == "madinah") {
			a.city = khasanahOtherCity(b.city)
		} else if b.city == "" && (a.city == "makkah" || a.city == "madinah") {
			b.city = khasanahOtherCity(a.city)
		}
	}
	for _, h := range hotels {
		star := 0
		if m := khasanahStarRe.FindStringSubmatch(h.desc); m != nil {
			star, _ = strconv.Atoi(m[1])
		}
		dist := khasanahDistance(h.desc)
		switch h.city {
		case "makkah":
			if pkg.HotelMakkah == "" {
				pkg.HotelMakkah, pkg.StarsMakkah, pkg.DistanceMakkah = h.name, star, dist
			}
		case "madinah":
			if pkg.HotelMadinah == "" {
				pkg.HotelMadinah, pkg.StarsMadinah, pkg.DistanceMadinah = h.name, star, dist
			}
		}
	}

	// ---- room prices ----
	body.Find("p").Each(func(_ int, p *goquery.Selection) {
		label := strings.ToLower(CleanText(p.Text()))
		if label != "triple room" && label != "double room" && label != "quad room" {
			return
		}
		price := 0
		p.Parent().Find("p").EachWithBreak(func(_ int, q *goquery.Selection) bool {
			if t := CleanText(q.Text()); strings.HasPrefix(t, "Rp") {
				price = ParsePrice(t)
				return false
			}
			return true
		})
		switch label {
		case "triple room":
			if pkg.PriceTriple == 0 {
				pkg.PriceTriple = price
			}
		case "double room":
			if pkg.PriceDouble == 0 {
				pkg.PriceDouble = price
			}
		}
	})

	// ---- itinerary ----
	var itin []string
	days := 0
	if sec := khasanahSection(body, "Itinerary"); sec != nil {
		sec.Find("ol > li").Each(func(_ int, li *goquery.Selection) {
			if n, err := strconv.Atoi(CleanText(li.Find("span.rounded-full").First().Text())); err == nil && n > days {
				days = n
			}
			li.Find("p").Each(func(_ int, p *goquery.Selection) {
				for _, l := range strings.Split(p.Text(), "\n") {
					if l = strings.Join(strings.Fields(l), " "); l != "" {
						itin = append(itin, l)
					}
				}
			})
		})
	}
	itinText := strings.Join(itin, "\n")
	itinLower := strings.ToLower(itinText)

	// Tagline under "Tentang Trip", e.g. "Umroh Hemat 9 Hari Spesial Ramadhan, Direct Flight!"
	tagline := ""
	if h := body.Find("h2").FilterFunction(func(_ int, s *goquery.Selection) bool {
		return strings.Contains(s.Text(), "Tentang Trip")
	}).First(); h.Length() > 0 {
		tagline = CleanText(h.Parent().Text())
		tagline = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(tagline, "🕋"), " Tentang Trip"))
		tagline = strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(tagline, "Tentang Trip")), "✓")
	}

	// Duration: the marketed number of days (name, then tagline), else the
	// number of days in the travel's own itinerary.
	for _, s := range []string{pkg.PackageName, tagline} {
		if m := khasanahDaysRe.FindStringSubmatch(s); m != nil && pkg.Duration == 0 {
			pkg.Duration, _ = strconv.Atoi(m[1])
		}
	}
	if pkg.Duration == 0 {
		if m := regexp.MustCompile(`(?i)(\d{1,2})hari`).FindStringSubmatch(pkg.URL); m != nil {
			pkg.Duration, _ = strconv.Atoi(m[1])
		}
	}
	if pkg.Duration == 0 && days >= 5 {
		pkg.Duration = days
	}

	// Departure airport: the first airport named in the itinerary.
	idxCGK := strings.Index(itinLower, "soekarno")
	idxSIN := strings.Index(itinLower, "changi")
	switch {
	case idxSIN >= 0 && (idxCGK < 0 || idxSIN < idxCGK):
		pkg.Airport = "Singapura"
	case idxCGK >= 0:
		pkg.Airport = "Jakarta"
	}

	// Direct / transit as written in the name, tagline or itinerary.
	allLower := strings.ToLower(pkg.PackageName + "\n" + tagline + "\n" + itinText)
	switch {
	case strings.Contains(allLower, "direct flight") || strings.Contains(allLower, "penerbangan langsung"):
		t := true
		pkg.Direct = &t
	case strings.Contains(itinLower, "transit"):
		f := false
		pkg.Direct = &f
		if m := khasanahViaRe.FindStringSubmatch(itinText); m != nil {
			pkg.TransitVia = strings.TrimSpace(m[1])
		}
	}

	// Airline: only when the trip content itself names the airline (not
	// reviews/T&C). Full names only: "Turkish Delight" / "Roman" must not match.
	if pkg.Airline == "" {
		for _, a := range khasanahAirlines {
			if a.re.MatchString(pkg.PackageName + " " + tagline + " " + itinText) {
				pkg.Airline = a.name
				break
			}
		}
	}

	// ---- included / excluded ----
	body.Find("p").Each(func(_ int, p *goquery.Selection) {
		t := CleanText(p.Text())
		var target *[]string
		switch {
		case strings.HasSuffix(t, "Tidak Termasuk"):
			target = &pkg.Excludes
		case strings.HasSuffix(t, "Termasuk"):
			target = &pkg.Facilities
		default:
			return
		}
		if len(*target) > 0 {
			return
		}
		*target = khasanahList(p.Parent().Find("ul").First())
	})
	// The hotel line ("Hotel Madinah *3 Makkah *4") is shown via the hotel fields.
	var facs []string
	for _, f := range pkg.Facilities {
		if !khasanahFacStars.MatchString(f) {
			facs = append(facs, f)
		}
	}
	pkg.Facilities = facs
}

// khasanahList reads a ✓/✕ list; "–" sub-items are appended to their parent and
// an item broken in two ("(jika diperbolehkan oleh" + "maskapai/pihak Saudi)") is
// joined back.
func khasanahList(ul *goquery.Selection) []string {
	var out []string
	var subs []string
	flush := func() {
		if len(subs) > 0 && len(out) > 0 {
			out[len(out)-1] = strings.TrimSuffix(out[len(out)-1], ":") + ": " + strings.Join(subs, ", ")
		}
		subs = nil
	}
	ul.Find("li").Each(func(_ int, li *goquery.Selection) {
		t := CleanText(li.Find("span").Last().Text())
		t = strings.Join(strings.Fields(t), " ")
		if t == "" {
			return
		}
		if li.HasClass("pl-7") {
			subs = append(subs, t)
			return
		}
		flush()
		if n := len(out); n > 0 && (strings.Count(out[n-1], "(") > strings.Count(out[n-1], ")") || strings.HasPrefix(t, "Rp")) {
			out[n-1] += " " + t
			return
		}
		out = append(out, t)
	})
	flush()
	// "Buku panduan doa dan, dzikir umroh" -> "Buku panduan doa dan dzikir umroh"
	for i := range out {
		out[i] = strings.ReplaceAll(out[i], " dan, ", " dan ")
	}
	return out
}

func khasanahHotelCity(h khasanahHotel) string {
	for _, s := range []string{strings.ToLower(h.label), strings.ToLower(h.name)} {
		switch {
		case strings.Contains(s, "makkah") || strings.Contains(s, "mekkah"):
			return "makkah"
		case strings.Contains(s, "madinah"):
			return "madinah"
		}
	}
	if h.label != "" {
		return "other" // "Turki - ...", "Eropa - ..."
	}
	d := strings.ToLower(h.desc)
	switch {
	case strings.Contains(d, "nabawi"):
		return "madinah"
	case strings.Contains(d, "haram"):
		return "makkah"
	}
	return ""
}

func khasanahOtherCity(c string) string {
	if c == "makkah" {
		return "madinah"
	}
	return "makkah"
}

// khasanahDistance returns the distance in metres written in a hotel
// description ("jarak sekitar 300-400 meter" -> 400, the upper bound).
func khasanahDistance(desc string) int {
	m := khasanahDistRe.FindStringSubmatch(desc)
	if m == nil {
		return 0
	}
	num := m[1]
	if m[2] != "" {
		num = m[2]
	}
	if strings.EqualFold(m[3], "km") {
		v, err := strconv.ParseFloat(strings.ReplaceAll(num, ",", "."), 64)
		if err != nil {
			return 0
		}
		return int(v * 1000)
	}
	n, _ := strconv.Atoi(strings.ReplaceAll(num, ".", ""))
	if n <= 0 || n > 5000 {
		return 0
	}
	return n
}

func khasanahCutoff() time.Time {
	min := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if today.After(min) {
		return today
	}
	return min
}

var khasanahMonths = map[string]time.Month{
	"jan": time.January, "januari": time.January, "january": time.January,
	"feb": time.February, "februari": time.February, "february": time.February,
	"mar": time.March, "maret": time.March, "march": time.March,
	"apr": time.April, "april": time.April,
	"mei": time.May, "may": time.May,
	"jun": time.June, "juni": time.June, "june": time.June,
	"jul": time.July, "juli": time.July, "july": time.July,
	"agu": time.August, "agt": time.August, "aug": time.August, "agustus": time.August, "august": time.August,
	"sep": time.September, "sept": time.September, "september": time.September,
	"okt": time.October, "oct": time.October, "oktober": time.October, "october": time.October,
	"nov": time.November, "november": time.November,
	"des": time.December, "dec": time.December, "desember": time.December, "december": time.December,
}

// khasanahParseDate parses "19 Jan 2027" / "19 Januari 2027".
func khasanahParseDate(s string) (time.Time, bool) {
	m := khasanahDateRe.FindStringSubmatch(s)
	if len(m) < 4 {
		return time.Time{}, false
	}
	mon, ok := khasanahMonths[strings.ToLower(m[2])]
	if !ok {
		return time.Time{}, false
	}
	d, _ := strconv.Atoi(m[1])
	y, _ := strconv.Atoi(m[3])
	return time.Date(y, mon, d, 0, 0, 0, 0, time.UTC), true
}
