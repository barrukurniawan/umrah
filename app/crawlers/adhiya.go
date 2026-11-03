package crawlers

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// AdhiyaParser: Adhiya Travel (PT Adhiya Wisata Utama), https://adhiyatravel.co.id/v2/.
// Daftar paket per bulan ada di /v2/bulan/<bulan>?page=N; tiap halaman detail /v2/paket/...
// memuat label berpasangan: Maskapai, Hotel Mekkah, Hotel Madinah, Keberangkatan
// ("05-Nov-2026"), Total Seat, Available Seat, Harga Jual.
type AdhiyaParser struct {
	URL, TravelName string
}

var (
	adhLinkRe = regexp.MustCompile(`https://adhiyatravel\.co\.id/v2/paket/[a-z0-9-]+`)
	adhCodeRe = regexp.MustCompile(`\s+\d{6}(?:\s+RE\s+\d{6})?\s*$`)
	adhDurRe  = regexp.MustCompile(`(?i)\b(\d{1,2})\s*D\b`)
	adhNumRe  = regexp.MustCompile(`\d+`)
	adhDateRe = regexp.MustCompile(`^(\d{1,2})-([A-Za-z]{3})-(\d{4})$`)
)

var adhMonths = []string{"januari", "februari", "maret", "april", "mei", "juni", "juli", "agustus", "september", "oktober", "november", "desember"}

func (a *AdhiyaParser) Crawl() ([]CrawledPackage, error) {
	travel := a.TravelName
	if travel == "" {
		travel = "Adhiya Travel"
	}
	// Kumpulkan semua halaman detail dari daftar per bulan (beberapa halaman per bulan).
	seen := map[string]bool{}
	var links []string
	for _, m := range adhMonths {
		for page := 1; page <= 10; page++ {
			html, err := lstFetchPolite(fmt.Sprintf("https://adhiyatravel.co.id/v2/bulan/%s?page=%d", m, page))
			if err != nil {
				break
			}
			added := 0
			for _, l := range adhLinkRe.FindAllString(html, -1) {
				if !seen[l] {
					seen[l] = true
					links = append(links, l)
					added++
				}
			}
			if added == 0 || !strings.Contains(html, fmt.Sprintf("page=%d", page+1)) {
				break
			}
		}
	}
	if len(links) == 0 {
		return nil, fmt.Errorf("adhiya: tidak ada tautan paket")
	}
	cutoff := lstCutoff()
	var out []CrawledPackage
	for _, u := range links {
		html, err := lstFetchPolite(u)
		if err != nil {
			continue
		}
		p, ok := adhParse(html, travel, u)
		if ok && lstFinalize(&p, cutoff) {
			out = append(out, p)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Departures[0].Date < out[j].Departures[0].Date })
	return out, nil
}

func adhParse(html, travel, url string) (CrawledPackage, bool) {
	L := lstHTMLLines(html)
	p := CrawledPackage{TravelName: travel, URL: url, Airport: ""}
	val := func(label string) string {
		for i := 0; i+1 < len(L); i++ {
			if strings.EqualFold(strings.TrimSpace(L[i]), label) {
				return strings.TrimSpace(L[i+1])
			}
		}
		return ""
	}
	for i := 2; i < len(L); i++ {
		if strings.TrimSpace(L[i]) == "Kode Produk" {
			p.PackageName = strings.TrimSpace(adhCodeRe.ReplaceAllString(L[i-1], ""))
			break
		}
	}
	if p.PackageName == "" {
		return p, false
	}
	if m := adhDurRe.FindStringSubmatch(p.PackageName); m != nil {
		p.Duration, _ = strconv.Atoi(m[1])
		// "Umroh Reguler 9D" -> "Umroh Reguler 9 Hari"
		p.PackageName = strings.TrimSpace(adhDurRe.ReplaceAllString(p.PackageName, m[1]+" Hari"))
	}
	al := strings.TrimSpace(strings.TrimPrefix(val("Maskapai"), "Fly "))
	switch {
	case strings.Contains(strings.ToLower(al), "saudi"):
		p.Airline = "Saudia"
	default:
		if p.Airline = lstAirlineName(al); p.Airline == "" {
			p.Airline = al
		}
	}
	p.HotelMakkah = strings.TrimSpace(strings.Split(val("Hotel Mekkah"), "/")[0])
	p.HotelMadinah = strings.TrimSpace(strings.Split(val("Hotel Madinah"), "/")[0])
	p.Price = plainMoney(val("Harga Jual"))

	m := adhDateRe.FindStringSubmatch(val("Keberangkatan"))
	if m == nil {
		return p, false
	}
	t, err := time.Parse("2-Jan-2006", m[1]+"-"+strings.ToUpper(m[2][:1])+strings.ToLower(m[2][1:])+"-"+m[3])
	if err != nil {
		return p, false
	}
	d := CrawledDeparture{Date: lstISODate(t)}
	if p.Duration > 0 {
		d.ReturnDate = lstISODate(t.AddDate(0, 0, p.Duration-1))
	}
	total, _ := strconv.Atoi(adhNumRe.FindString(val("Total Seat")))
	left, _ := strconv.Atoi(adhNumRe.FindString(val("Available Seat")))
	d.SeatsTotal = total
	if total > 0 && left <= 0 {
		d.Full = true
	} else {
		d.Seats = left
	}
	p.Departures = []CrawledDeparture{d}
	return p, p.Price > 0
}
