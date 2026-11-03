package crawlers

import (
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// VentourParser: Ventour Travel (PT Ventura Semesta Wisata). Beranda ventour.co.id memuat
// tautan ke halaman detail publik venom.ventour.co.id/jamaah/detail_paket?id=N; tiap halaman
// berisi nama paket, harga (normal & promo), DP, 2 hotel, durasi, tanggal berangkat/pulang,
// dan jadwal penerbangan (kode maskapai + rute). Sisa seat tidak dipublikasikan.
type VentourParser struct {
	URL        string
	TravelName string
}

var (
	ventIDRe     = regexp.MustCompile(`detail_paket\?id=(\d+)`)
	ventIDRRe    = regexp.MustCompile(`IDR\s*([\d.]+)`)
	ventDPRe     = regexp.MustCompile(`(?i)^Rp\s*([\d.,]+)\s*(Jt|Juta)?`)
	ventDateRe   = regexp.MustCompile(`^(?:Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday),\s*(\d{1,2} \w+ \d{4})$`)
	ventStartRe  = regexp.MustCompile(`\bSTART\s+(JAKARTA|BANDUNG|SURABAYA|PADANG|MEDAN|MAKASSAR|SOLO|SEMARANG|YOGYAKARTA|PALEMBANG|PEKANBARU|BATAM|BALIKPAPAN|BANJARMASIN|LOMBOK|BALI|ACEH|KERTAJATI)\b`)
	ventFlightRe = regexp.MustCompile(`^([A-Z0-9]{2})\s?\d{2,4}\s+\d{2}[A-Z]{3}\s+([A-Z]{3})([A-Z]{3})\b`)
)

func (v *VentourParser) Crawl() ([]CrawledPackage, error) {
	home := v.URL
	if home == "" {
		home = "https://ventour.co.id/"
	}
	travel := v.TravelName
	if travel == "" {
		travel = "Ventour Travel"
	}
	html, err := lstFetch(home)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var ids []string
	for _, m := range ventIDRe.FindAllStringSubmatch(html, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			ids = append(ids, m[1])
		}
	}
	cutoff := lstCutoff()
	var out []CrawledPackage
	for _, id := range ids {
		u := "https://venom.ventour.co.id/jamaah/detail_paket?id=" + id
		page, err := lstFetchPolite(u)
		if err != nil {
			continue
		}
		p, ok := ventParse(page, travel, u)
		if ok && lstFinalize(&p, cutoff) {
			out = append(out, p)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Departures[0].Date < out[j].Departures[0].Date })
	return out, nil
}

func ventParse(html, travel, url string) (CrawledPackage, bool) {
	L := lstHTMLLines(html)
	p := CrawledPackage{TravelName: travel, URL: url}
	start := -1
	for i, l := range L {
		if l == "Terpercaya, Terbukti, Recommended" && i+1 < len(L) {
			p.PackageName = strings.TrimSpace(L[i+1])
			start = i
			break
		}
	}
	if start < 0 || p.PackageName == "" {
		return p, false
	}
	var prices, dates []string
	var hotels []string
	var flights [][]string
	section := ""
	for i := start + 1; i < len(L); i++ {
		l := strings.TrimSpace(L[i])
		switch {
		case l == "Daftar Sekarang" && section == "flight":
			i = len(L)
			continue
		case strings.HasPrefix(l, "Harga mulai"):
			if section == "flight" {
				i = len(L)
				continue
			}
			section = "price"
			continue
		case l == "DP mulai dari":
			section = "dp"
			continue
		case l == "Hotel":
			section = "hotel"
			continue
		case l == "Trip Schedule":
			section = "dates"
			continue
		case l == "Maskapai" || l == "Flight Schedule":
			section = "flight"
			continue
		case l == "Detail Promo":
			section = "promo"
			continue
		}
		switch section {
		case "price":
			if m := ventIDRRe.FindStringSubmatch(l); m != nil {
				prices = append(prices, m[1])
			}
		case "dp":
			if m := ventDPRe.FindStringSubmatch(l); m != nil {
				n := plainMoney(m[1])
				if m[2] != "" && n == 0 {
					f, _ := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64)
					n = int(f * 1000000)
				}
				p.DownPayment = n
				section = ""
			}
		case "hotel":
			if n, err := strconv.Atoi(l); err == nil {
				p.Duration = n
				section = ""
			} else if l != "" && l != "Hari" {
				hotels = append(hotels, l)
			}
		case "dates":
			if m := ventDateRe.FindStringSubmatch(l); m != nil {
				dates = append(dates, m[1])
			}
		case "flight":
			if m := ventFlightRe.FindStringSubmatch(l); m != nil {
				flights = append(flights, m)
			}
		}
	}
	// Harga: yang termurah (harga promo bila ada).
	for _, s := range prices {
		if n := plainMoney(s); n > 0 && (p.Price == 0 || n < p.Price) {
			p.Price = n
		}
	}
	if len(dates) == 0 || p.Duration <= 0 || p.Duration > 45 {
		return p, false // mis. "program rencana" dengan rentang 2030
	}
	dep, ok1 := lstParseDate(dates[0])
	if !ok1 {
		return p, false
	}
	d := CrawledDeparture{Date: lstISODate(dep)}
	if len(dates) > 1 {
		if ret, ok := lstParseDate(dates[1]); ok {
			d.ReturnDate = lstISODate(ret)
		}
	}
	p.Departures = []CrawledDeparture{d}

	// Penerbangan berangkat = penerbangan sebelum tiba di JED/MED pertama kali.
	landMED := false
	if len(flights) > 0 {
		p.Airline = lstAirlineName(flights[0][1])
		p.Airport = airportCity[flights[0][2]]
		var via string
		for _, f := range flights {
			if f[3] == "JED" || f[3] == "MED" {
				landMED = f[3] == "MED"
				break
			}
			via = f[3]
		}
		direct := via == ""
		p.Direct = &direct
		if via != "" {
			p.TransitVia = map[string]string{"MCT": "Muscat", "DXB": "Dubai", "DOH": "Doha", "AUH": "Abu Dhabi", "KUL": "Kuala Lumpur", "IST": "Istanbul", "BAH": "Bahrain", "CAI": "Kairo", "KWI": "Kuwait", "DEL": "Delhi", "BOM": "Mumbai", "CMB": "Kolombo"}[via]
		}
	}
	nameU := strings.ToUpper(p.PackageName)
	// "START BANDUNG/SURABAYA/PADANG" di nama paket = kota keberangkatan jamaah.
	if m := ventStartRe.FindStringSubmatch(nameU); m != nil {
		p.Airport = lstTitle(m[1])
	}
	if strings.Contains(nameU, "LANDING MADINAH") || strings.Contains(nameU, "MADINAH FIRST") {
		landMED = true
	}
	p.HotelMakkah, p.HotelMadinah = ventHotels(hotels, landMED)
	return p, p.Price > 0
}

// ventHotels menentukan hotel Makkah vs Madinah dari 2 nama hotel: pakai daftar hotel yang
// sudah dikenal (config/hotel_geo.json), lalu kata kunci; bila tetap tidak jelas, urutan
// mengikuti itinerary (landing Madinah -> hotel pertama di Madinah).
func ventHotels(h []string, landMED bool) (makkah, madinah string) {
	if len(h) == 0 {
		return "", ""
	}
	city := func(n string) string {
		if c := knownHotelCity(n); c != "" {
			return c
		}
		u := strings.ToUpper(n)
		for _, k := range []string{"ODST", "NABAWI", "MADINAH", "MADINA", "TAIBA", "DALLAH", "ANSAR", "AL HARAM", "SHAZA", "MUNAWARAH", "ANDALUS", "PULLMAN ZAMZAM MADINAH"} {
			if strings.Contains(u, k) {
				return "madinah"
			}
		}
		for _, k := range []string{"AJYAD", "AYJAD", "MAQAM", "MASSA", "MISFALAH", "AZIZ", "KUDAI", "SAFWA", "HAJAR", "ZAMZAM", "SWISS", "HILTON", "ELAF", "MAKKAH", "MAKAH", "MEKKAH", "QASWA", "ABRAJ"} {
			if strings.Contains(u, k) {
				return "makkah"
			}
		}
		return ""
	}
	if len(h) == 1 {
		if city(h[0]) == "madinah" {
			return "", h[0]
		}
		return h[0], ""
	}
	a, b := h[0], h[1]
	ca, cb := city(a), city(b)
	switch {
	case ca == "makkah" || cb == "madinah":
		return a, b
	case ca == "madinah" || cb == "makkah":
		return b, a
	case landMED:
		return b, a
	}
	return a, b
}

var (
	hotelCityOnce sync.Once
	hotelCityIdx  map[string]string
)

func knownHotelCity(name string) string {
	hotelCityOnce.Do(func() {
		hotelCityIdx = map[string]string{}
		b, err := os.ReadFile("config/hotel_geo.json")
		if err != nil {
			return
		}
		var doc struct {
			Hotels []struct {
				City    string   `json:"city"`
				Name    string   `json:"name"`
				Aliases []string `json:"aliases"`
			} `json:"hotels"`
		}
		if json.Unmarshal(b, &doc) != nil {
			return
		}
		for _, h := range doc.Hotels {
			for _, n := range append([]string{h.Name}, h.Aliases...) {
				hotelCityIdx[strings.ToUpper(strings.TrimSpace(n))] = h.City
			}
		}
	})
	return hotelCityIdx[strings.ToUpper(strings.TrimSpace(name))]
}
