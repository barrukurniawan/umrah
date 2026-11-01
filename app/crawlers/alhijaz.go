package crawlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// AlhijazParser reads the JSON API that the alhijaz.co single-page app itself
// calls (GET /api/schedules/<hijri-year>, e.g. 1448). Each record ("jadwal") is
// one departure with its own share link https://alhijaz.co/<jadwal_id>, and has:
// name ("REGULER 9HR ..."), seat_total / seat_sisa, maskapai, berangkat/pulang
// date + route + flight code, paket_harga per tier (Quard/Triple/Double) and
// paket_hotel per tier ("ANJUM (★5)").
//
// Hotel distances come from /api/hotels/public (distance_label "±450m"), the
// travel's own hotel database; the app bundle also has an older built-in
// table, so a distance is only used when the two do not contradict each other.
type AlhijazParser struct {
	URL string
}

type alhijazSchedule struct {
	JadwalID     string                       `json:"jadwal_id"`
	Nama         string                       `json:"jadwal_nama"`
	SeatTotal    json.RawMessage              `json:"seat_total"`
	SeatSisa     json.RawMessage              `json:"seat_sisa"`
	Maskapai     string                       `json:"maskapai"`
	BerangkatTgl string                       `json:"berangkat_tgl"`
	BerangkatRut string                       `json:"berangkat_rute"`
	BerangkatKod string                       `json:"berangkat_kode_penerbangan"`
	PulangTgl    string                       `json:"pulang_tgl"`
	PulangRute   string                       `json:"pulang_rute"`
	HargaRaw     json.RawMessage              `json:"paket_harga"`
	HotelRaw     json.RawMessage              `json:"paket_hotel"`
	Harga        map[string]map[string]string `json:"-"`
	Hotel        map[string]map[string]string `json:"-"`
}

// alhijazInt reads a JSON number or numeric string; ok=false when absent.
func alhijazInt(raw json.RawMessage) (int, bool) {
	s := strings.Trim(strings.TrimSpace(string(raw)), `"`)
	n, err := strconv.Atoi(strings.TrimSpace(s))
	return n, err == nil
}

// alhijazTierMap decodes {"TIER":{"k":"v"}}; arrays/empty values give nil.
func alhijazTierMap(raw json.RawMessage) map[string]map[string]string {
	var generic map[string]map[string]interface{}
	if json.Unmarshal(raw, &generic) != nil {
		return nil
	}
	out := map[string]map[string]string{}
	for t, kv := range generic {
		out[t] = map[string]string{}
		for k, v := range kv {
			if v != nil {
				out[t][k] = strings.TrimSpace(fmt.Sprint(v))
			}
		}
	}
	return out
}

type alhijazHotelInfo struct {
	Name     string   `json:"name"`
	Aliases  []string `json:"aliases"`
	City     string   `json:"city"`
	Stars    int      `json:"stars"`
	Distance string   `json:"distance_label"`
}

var (
	alhijazStarRe     = regexp.MustCompile(`\s*\(\s*[★⭐]\s*(\d)\s*\)\s*$`)
	alhijazSetarafRe  = regexp.MustCompile(`(?i)\s*/\s*SETARAF.*$`)
	alhijazDistRe     = regexp.MustCompile(`(?i)±?\s*([\d.,]+)\s*(km|m)\b`)
	alhijazScriptRe   = regexp.MustCompile(`src="(/assets/index-[^"]+\.js)"`)
	alhijazBundleHtRe = regexp.MustCompile(`(?:"([^"]+)"|([A-Z][A-Z0-9 ]*)):\{stars:"\d"(?:,distance:"([^"]*)")?\}`)
	alhijazIATARe     = regexp.MustCompile(`[A-Z]{3}`)
	alhijazShortDurRe = regexp.MustCompile(`(?i)\b(\d{1,2})\s*H\b`)
)

var alhijazCity = map[string]string{
	"CGK": "Jakarta", "SUB": "Surabaya", "KNO": "Medan", "UPG": "Makassar", "BTH": "Batam",
	"SOC": "Solo", "JOG": "Yogyakarta", "YIA": "Yogyakarta", "BPN": "Balikpapan", "PDG": "Padang",
	"PLM": "Palembang", "BDJ": "Banjarmasin", "PKU": "Pekanbaru", "DPS": "Denpasar", "KJT": "Majalengka",
	"DXB": "Dubai", "AUH": "Abu Dhabi", "DOH": "Doha", "MCT": "Muscat", "IST": "Istanbul",
	"CAI": "Kairo", "HAK": "Haikou", "KUL": "Kuala Lumpur", "SIN": "Singapura", "BAH": "Bahrain",
	"JED": "Jeddah", "MED": "Madinah", "TIF": "Taif", "AMM": "Amman", "CMB": "Kolombo", "BOM": "Mumbai",
}

func alhijazGetJSON(u string, v interface{}) error {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", lstUserAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: status %d", u, resp.StatusCode)
	}
	return json.Unmarshal(body, v)
}

// alhijazNorm mirrors the site's own Ca(): upper-case, drop "/SETARAF...", squeeze spaces.
func alhijazNorm(s string) string {
	s = alhijazStarRe.ReplaceAllString(s, "")
	s = strings.ToUpper(s)
	s = alhijazSetarafRe.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(s), " ")
}

// alhijazMeters converts "±450m" / "±2.5km" to meters.
func alhijazMeters(s string) int {
	m := alhijazDistRe.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	f, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64)
	if err != nil {
		return 0
	}
	if strings.EqualFold(m[2], "km") {
		f *= 1000
	}
	return int(f + 0.5)
}

// alhijazHotel splits "AL MASSA GRAND/SETARAF (★4)" into ("AL MASSA GRAND", 4).
func alhijazHotel(s string) (string, int) {
	stars := 0
	if m := alhijazStarRe.FindStringSubmatch(s); m != nil {
		stars, _ = strconv.Atoi(m[1])
	}
	name := alhijazStarRe.ReplaceAllString(s, "")
	name = alhijazSetarafRe.ReplaceAllString(name, "")
	name = strings.Join(strings.Fields(name), " ")
	if name == "-" {
		name = ""
	}
	return name, stars
}

// alhijazDistances builds normalized-hotel-name -> meters for Makkah/Madinah hotels.
func (a *AlhijazParser) alhijazDistances() map[string]int {
	out := map[string]int{}
	var hres struct {
		Success bool               `json:"success"`
		Data    []alhijazHotelInfo `json:"data"`
	}
	if err := alhijazGetJSON(a.base()+"/api/hotels/public", &hres); err != nil {
		log.Printf("[alhijaz] hotels: %v", err)
		return out
	}
	// Built-in table of the front-end bundle (may be older); used only to veto.
	bundle := map[string]int{}
	if home, err := lstFetch(a.base() + "/"); err == nil {
		if m := alhijazScriptRe.FindStringSubmatch(home); m != nil {
			if js, err := lstFetch(a.base() + m[1]); err == nil {
				if i := strings.Index(js, "const En={"); i >= 0 {
					seg := js[i:]
					if j := strings.Index(seg, "}}"); j > 0 {
						seg = seg[:j+2]
					}
					for _, mm := range alhijazBundleHtRe.FindAllStringSubmatch(seg, -1) {
						if d := alhijazMeters(mm[3]); d > 0 {
							bundle[alhijazNorm(mm[1]+mm[2])] = d
						}
					}
				}
			}
		}
	}
	for _, h := range hres.Data {
		city := strings.ToLower(h.City)
		if city != "mekkah" && city != "makkah" && city != "madinah" {
			continue
		}
		d := alhijazMeters(h.Distance)
		if d <= 0 {
			continue
		}
		for _, n := range append([]string{h.Name}, h.Aliases...) {
			key := alhijazNorm(n)
			if bd, ok := bundle[key]; ok && bd != d {
				log.Printf("[alhijaz] jarak %s tidak konsisten (API %dm, situs %dm), tidak dipakai", key, d, bd)
				continue
			}
			out[key] = d
		}
	}
	return out
}

func alhijazLookupDistance(dist map[string]int, hotel string) int {
	key := alhijazNorm(hotel)
	if key == "" {
		return 0
	}
	if d, ok := dist[key]; ok {
		return d
	}
	return 0
}

func (a *AlhijazParser) base() string {
	u, err := url.Parse(a.URL)
	if err != nil || u.Host == "" {
		return "https://alhijaz.co"
	}
	return u.Scheme + "://" + u.Host
}

func (a *AlhijazParser) Crawl() ([]CrawledPackage, error) {
	cutoff := lstCutoff()

	// Hijri year codes: the app defaults to the current one ("1448"); also try
	// neighbours so a newly opened season is picked up.
	var all []alhijazSchedule
	var lastErr error
	for _, year := range []string{"1447", "1448", "1449", "1450"} {
		var res struct {
			Status string            `json:"status"`
			Data   []alhijazSchedule `json:"aaData"`
		}
		u := a.base() + "/api/schedules/" + year
		log.Println("[crawler] visiting:", u)
		if err := alhijazGetJSON(u, &res); err != nil {
			if year == "1448" {
				log.Printf("[alhijaz] %s: %v", u, err)
			}
			lastErr = err
			continue
		}
		if res.Status == "ok" {
			all = append(all, res.Data...)
		}
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("alhijaz: no schedules (%v)", lastErr)
	}
	dist := a.alhijazDistances()

	var packages []CrawledPackage
	seen := map[string]bool{}
	for _, s := range all {
		if s.JadwalID == "" || seen[s.JadwalID] {
			continue
		}
		dep, err := time.Parse("2006-01-02", strings.TrimSpace(s.BerangkatTgl))
		if err != nil || dep.Before(cutoff) {
			continue
		}
		seen[s.JadwalID] = true
		s.Harga, s.Hotel = alhijazTierMap(s.HargaRaw), alhijazTierMap(s.HotelRaw)
		if p, ok := a.toPackage(s, dep, dist); ok {
			packages = append(packages, p)
		}
	}
	sort.SliceStable(packages, func(i, j int) bool {
		return packages[i].Departures[0].Date < packages[j].Departures[0].Date
	})
	return packages, nil
}

func (a *AlhijazParser) toPackage(s alhijazSchedule, dep time.Time, dist map[string]int) (CrawledPackage, bool) {
	pkg := CrawledPackage{
		TravelName:  "Al Hijaz",
		PackageName: strings.Join(strings.Fields(s.Nama), " "),
		URL:         a.base() + "/" + url.PathEscape(s.JadwalID),
		Duration:    ParseDuration(s.Nama), // "9HR" / "12 HR" / "9 HARI"
	}
	if pkg.Duration == 0 {
		// "UMRAH HEMAT PLUS TAIF 9H"
		if m := alhijazShortDurRe.FindStringSubmatch(s.Nama); m != nil {
			pkg.Duration, _ = strconv.Atoi(m[1])
		}
	}

	// Cheapest tier (by Quad) is the package-level price/hotels.
	tiers := make([]string, 0, len(s.Harga))
	for t := range s.Harga {
		tiers = append(tiers, t)
	}
	sort.Strings(tiers)
	best := ""
	bestQuad := 0
	for _, t := range tiers {
		q := ParsePrice(s.Harga[t]["Quard"] + s.Harga[t]["Quad"])
		if q <= 0 {
			continue
		}
		if best == "" || q < bestQuad {
			best, bestQuad = t, q
		}
	}
	if best == "" {
		log.Printf("[alhijaz] %s %s: tanpa harga, dilewati", s.JadwalID, pkg.PackageName)
		return pkg, false
	}
	pkg.Price = bestQuad
	pkg.PriceTriple = ParsePrice(s.Harga[best]["Triple"])
	pkg.PriceDouble = ParsePrice(s.Harga[best]["Double"])
	if h, ok := s.Hotel[best]; ok {
		pkg.HotelMakkah, pkg.StarsMakkah = alhijazHotel(h["mekkah"])
		pkg.HotelMadinah, pkg.StarsMadinah = alhijazHotel(h["madinah"])
		pkg.DistanceMakkah = alhijazLookupDistance(dist, pkg.HotelMakkah)
		pkg.DistanceMadinah = alhijazLookupDistance(dist, pkg.HotelMadinah)
	}

	// Airline.
	if al := KnownAirline(s.Maskapai); al != "" {
		pkg.Airline = al
	} else if m := strings.TrimSpace(s.Maskapai); m != "" {
		switch strings.ToUpper(m) {
		case "EGYPTAIR":
			pkg.Airline = "EgyptAir"
		case "HAINAN":
			pkg.Airline = "Hainan Airlines"
		default:
			pkg.Airline = strings.Title(strings.ToLower(m))
		}
	}

	// Route: "CGK - JED" (non-stop) vs "CGK-DXB / DXB-JED" (via Dubai).
	codes := alhijazIATARe.FindAllString(strings.ToUpper(s.BerangkatRut), -1)
	if len(codes) >= 2 {
		pkg.Airport = alhijazCity[codes[0]]
		if pkg.Airport == "" {
			pkg.Airport = codes[0]
		}
		first := codes[1]
		direct := first == "JED" || first == "MED" || first == "TIF"
		pkg.Direct = &direct
		if !direct {
			pkg.TransitVia = alhijazCity[first]
			if pkg.TransitVia == "" {
				pkg.TransitVia = first
			}
		}
	}

	d := CrawledDeparture{Date: dep.Format("2006-01-02")}
	if r, err := time.Parse("2006-01-02", strings.TrimSpace(s.PulangTgl)); err == nil && !r.Before(dep) {
		d.ReturnDate = r.Format("2006-01-02")
	}
	if total, ok := alhijazInt(s.SeatTotal); ok && total > 0 {
		d.SeatsTotal = total
		pkg.SeatsTotal = total
	}
	if left, ok := alhijazInt(s.SeatSisa); ok {
		if left <= 0 {
			d.Full, pkg.Full = true, true
		} else {
			d.Seats, pkg.Seats = left, left
		}
	}
	pkg.Departures = []CrawledDeparture{d}
	pkg.DepartureDates = []string{lstFormatDate(dep)}
	return pkg, true
}

func parseJutaStr(s string) int {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "rp")
	s = strings.Join(strings.Fields(s), "")

	parts := strings.Split(s, "jt")
	if len(parts) == 0 {
		return 0
	}
	numPart := strings.ReplaceAll(parts[0], ",", ".")
	var num float64
	fmt.Sscanf(numPart, "%f", &num)
	return int(num * 1000000)
}

func findChrome() string {
	for _, name := range []string{"chromium-browser", "chromium", "google-chrome", "google-chrome-stable"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	if _, err := os.Stat("/snap/bin/chromium"); err == nil {
		return "/snap/bin/chromium"
	}
	return ""
}
