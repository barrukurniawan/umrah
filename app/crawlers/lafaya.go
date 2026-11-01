package crawlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// LafayaParser reads Lafaya Travel's packages from the public Supabase REST
// endpoint that the lafayatravel.com single-page app itself queries
// (from("packages").select("*").is("is_archived", false), anon key from the
// site bundle). Detail page: https://lafayatravel.com/paket/<id>.
//
// The row's own departure_date is a single (often stale) date; the real
// schedule is in the free-text description, in blocks like
//
//	🗓️ 7 NOVEMBER 2026
//	💠 QUAD: Rp. 32,500,000 / TRIPLE / DOUBLE
//
// so every dated line (with an explicit year) becomes a departure carrying the
// price block that follows it. Hotels come from "🏨 4 N Makkah ⭐⭐⭐⭐ <name> /
// Sekitar" lines (stars = number of ⭐). Seats are only package-wide in the
// data (and default to 90/90), so they are not attached to any date.
type LafayaParser struct {
	PackageIDs []string
}

type lafayaRow struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	Price          float64           `json:"price"`
	Duration       int               `json:"duration"`
	DepartureCity  *string           `json:"departure_city"`
	DepartureDate  *string           `json:"departure_date"`
	AvailableSeats *int              `json:"available_seats"`
	TotalSeats     *int              `json:"total_seats"`
	Description    string            `json:"description"`
	Includes       []string          `json:"includes"`
	Excludes       []string          `json:"excludes"`
	Itinerary      []json.RawMessage `json:"itinerary"`
	IsArchived     bool              `json:"is_archived"`
}

const (
	lafayaSupabaseURL = "https://yisgtmdzwklfupkfsvtb.supabase.co/rest/v1/packages"
	lafayaAPIKey      = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6Inlpc2d0bWR6d2tsZnVwa2ZzdnRiIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NTMyNTc5NjIsImV4cCI6MjA2ODgzMzk2Mn0.a6djeye_zYwVIsTvhVsU8W1r72NCfR6p2jWcNUV62F0"
)

var (
	lafayaDateRe     = regexp.MustCompile(`(?i)\b(\d{1,2})\s+(` + lstMonthAlt + `)\.?\s+(20\d{2})\b`)
	lafayaRoomRe     = regexp.MustCompile(`(?i)\b(quad|triple|double)\b`)
	lafayaRpRe       = regexp.MustCompile(`(?i)Rp\.?\s*([\d][\d.,]*\d)`)
	lafayaDurRe      = regexp.MustCompile(`(?i)\b(\d{1,2})\s*Hari\b`)
	lafayaHotelRe    = regexp.MustCompile(`(?i)\b(makkah|mekkah|mekah|madinah)\b\s*[:\-]?\s*((?:⭐\x{FE0F}?\s*)+)(.+)$`)
	lafayaSekitarRe  = regexp.MustCompile(`(?i)\s*/\s*(sekitar|setaraf)\b.*$`)
	lafayaParenRe    = regexp.MustCompile(`\([^)]*\)`)
	lafayaJarakRe    = regexp.MustCompile(`(?i)jarak[^\d]{0,15}(\d+(?:[.,]\d+)?)\s*(km|m)\b`)
	lafayaPesawatRe  = regexp.MustCompile(`(?i)^\W*(?:pesawat|maskapai)\s*:?\s*(.+)$`)
	lafayaDPRe       = regexp.MustCompile(`(?i)\bDP\s*Rp\.?\s*([\d.,]+)`)
	lafayaGuideRe    = regexp.MustCompile(`(?i)\bbersama\s+(ustadz?\b.+)$`)
	lafayaLeadEmoji  = regexp.MustCompile(`^[^\p{L}\p{N}]+`)
	lafayaTrailEmoji = regexp.MustCompile(`[^\p{L}\p{N}.)]+$`)
	lafayaDanRe      = regexp.MustCompile(`(?i)\s+dan\s+`)
)

func (p *LafayaParser) Crawl() ([]CrawledPackage, error) {
	url := lafayaSupabaseURL + "?is_archived=eq.false&select=*"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("apikey", lafayaAPIKey)
	req.Header.Set("Authorization", "Bearer "+lafayaAPIKey)
	log.Println("[crawler] visiting:", url)
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch packages: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("supabase error %d: %s", resp.StatusCode, string(body))
	}
	var rows []lafayaRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	cutoff := lstCutoff()
	var result []CrawledPackage
	for _, r := range rows {
		if r.Type != "umrah" || r.IsArchived {
			continue
		}
		if len(p.PackageIDs) > 0 && !lafayaContains(p.PackageIDs, r.ID) {
			continue
		}
		cp, ok := p.parsePackage(r, cutoff)
		if !ok {
			log.Printf("[lafaya] %s (%s): tidak ada tanggal keberangkatan mendatang, dilewati", r.Name, r.ID)
			continue
		}
		result = append(result, cp)
	}
	return result, nil
}

func lafayaContains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

type lafayaDep struct {
	date                 time.Time
	quad, triple, double int
}

func (p *LafayaParser) parsePackage(r lafayaRow, cutoff time.Time) (CrawledPackage, bool) {
	desc := strings.ReplaceAll(r.Description, "\r", "")
	lines := strings.Split(desc, "\n")

	cp := CrawledPackage{
		TravelName:  "Lafaya Travel",
		PackageName: strings.Join(strings.Fields(r.Name), " "),
		URL:         "https://lafayatravel.com/paket/" + r.ID,
	}

	// Duration: the row's duration, unless the description states another one.
	descDur := 0
	if m := lafayaDurRe.FindStringSubmatch(desc); m != nil {
		descDur, _ = strconv.Atoi(m[1])
	}
	switch {
	case r.Duration > 1 && (descDur == 0 || descDur == r.Duration):
		cp.Duration = r.Duration
	case r.Duration > 1 && descDur > 0:
		log.Printf("[lafaya] %s: durasi kolom %d vs deskripsi %d hari, durasi dikosongkan", r.Name, r.Duration, descDur)
	}

	// Departures + price blocks.
	var deps []lafayaDep
	groupStart := 0 // index in deps of the current date group
	priced := false // current group already got its prices
	frozen := false // an "ALL IN" estimate block started before prices
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if ms := lafayaDateRe.FindAllStringSubmatch(line, -1); ms != nil && !lafayaRoomRe.MatchString(line) {
			if priced || frozen {
				groupStart = len(deps)
				priced, frozen = false, false
			}
			for _, m := range ms {
				d, _ := strconv.Atoi(m[1])
				y, _ := strconv.Atoi(m[3])
				if t, ok := lstMakeDate(y, lstMonthNum(m[2]), d); ok {
					deps = append(deps, lafayaDep{date: t})
				}
			}
			continue
		}
		if strings.Contains(strings.ToLower(line), "all in") && !priced {
			frozen = true
		}
		if frozen || len(deps) == groupStart {
			continue
		}
		room := lafayaRoomRe.FindString(line)
		rp := lafayaRpRe.FindStringSubmatch(line)
		if room == "" || rp == nil {
			continue
		}
		v := ParsePrice(rp[1])
		if v < lstMinPrice || v > lstMaxPrice {
			continue
		}
		for i := groupStart; i < len(deps); i++ {
			switch strings.ToLower(room) {
			case "quad":
				if deps[i].quad == 0 {
					deps[i].quad = v
				}
			case "triple":
				if deps[i].triple == 0 {
					deps[i].triple = v
				}
			case "double":
				if deps[i].double == 0 {
					deps[i].double = v
				}
			}
		}
		if strings.EqualFold(room, "double") {
			priced = true
		}
	}

	// Keep upcoming, unique dates.
	sort.SliceStable(deps, func(i, j int) bool { return deps[i].date.Before(deps[j].date) })
	seen := map[string]bool{}
	var future []lafayaDep
	for _, d := range deps {
		k := d.date.Format("2006-01-02")
		if d.date.Before(cutoff) || seen[k] {
			continue
		}
		seen[k] = true
		future = append(future, d)
	}
	if len(future) == 0 {
		return cp, false
	}

	// Package price: cheapest dated Quad, else the price shown on the site card.
	best := -1
	for i, d := range future {
		if d.quad > 0 && (best < 0 || d.quad < future[best].quad) {
			best = i
		}
	}
	if best >= 0 {
		cp.Price = future[best].quad
		cp.PriceTriple = future[best].triple
		cp.PriceDouble = future[best].double
	} else {
		cp.Price = int(r.Price)
	}
	if cp.Price < lstMinPrice || cp.Price > lstMaxPrice {
		return cp, false
	}

	// Return date: on this site day 1 = leaving Jakarta and day N = arriving
	// in Jakarta (see the itineraries "Hari 1: JAKARTA – JEDDAH ...",
	// "Hari 9: JAKARTA"), so return = departure + (N-1).
	for _, d := range future {
		dd := CrawledDeparture{Date: d.date.Format("2006-01-02")}
		if cp.Duration > 1 {
			dd.ReturnDate = d.date.AddDate(0, 0, cp.Duration-1).Format("2006-01-02")
		}
		if d.quad > 0 && d.quad != cp.Price {
			dd.Price = d.quad
		}
		if d.triple > 0 && d.triple != cp.PriceTriple {
			dd.PriceTriple = d.triple
		}
		if d.double > 0 && d.double != cp.PriceDouble {
			dd.PriceDouble = d.double
		}
		cp.Departures = append(cp.Departures, dd)
		cp.DepartureDates = append(cp.DepartureDates, lstFormatDate(d.date))
	}

	// Airline, hotels, DP, guide from the description.
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if cp.Airline == "" {
			if m := lafayaPesawatRe.FindStringSubmatch(lafayaLeadEmoji.ReplaceAllString(line, "")); m != nil {
				cp.Airline = lafayaAirline(m[1])
			} else if strings.HasPrefix(line, "✈") {
				// "✈️ EGYPT AIR" line under the title
				rest := strings.TrimSpace(lafayaLeadEmoji.ReplaceAllString(line, ""))
				if strings.Contains(strings.ToLower(strings.ReplaceAll(rest, " ", "")), "egyptair") || KnownAirline(rest) != "" {
					cp.Airline = lafayaAirline(strings.ReplaceAll(rest, "EGYPT AIR", "EgyptAir"))
				}
			}
		}
		if m := lafayaHotelRe.FindStringSubmatch(line); m != nil {
			city := strings.ToLower(m[1])
			stars := strings.Count(m[2], "⭐")
			if stars > 5 {
				stars = 0
			}
			rest := m[3]
			dist := 0
			if j := lafayaJarakRe.FindStringSubmatch(rest); j != nil {
				f, _ := strconv.ParseFloat(strings.ReplaceAll(j[1], ",", "."), 64)
				if strings.EqualFold(j[2], "km") {
					f *= 1000
				}
				dist = int(f + 0.5)
			}
			name := lafayaParenRe.ReplaceAllString(rest, "")
			name = lafayaSekitarRe.ReplaceAllString(name, "")
			name = strings.Trim(strings.Join(strings.Fields(name), " "), "/ ")
			if name == "" {
				continue
			}
			if city == "madinah" {
				if cp.HotelMadinah == "" {
					cp.HotelMadinah, cp.StarsMadinah, cp.DistanceMadinah = name, stars, dist
				}
			} else if cp.HotelMakkah == "" {
				cp.HotelMakkah, cp.StarsMakkah, cp.DistanceMakkah = name, stars, dist
			}
		}
		if cp.DownPayment == 0 {
			if m := lafayaDPRe.FindStringSubmatch(line); m != nil {
				cp.DownPayment = ParsePrice(m[1])
			}
		}
		if cp.Guide == "" {
			if m := lafayaGuideRe.FindStringSubmatch(lafayaLeadEmoji.ReplaceAllString(line, "")); m != nil {
				g := lafayaTrailEmoji.ReplaceAllString(strings.TrimSpace(m[1]), "")
				g = lafayaDanRe.ReplaceAllString(g, ", ")
				cp.Guide = strings.TrimSpace(g)
			}
		}
	}

	// Includes / excludes: structured columns first, else the ✅ / ❌ lists
	// under "TERMASUK" / "TIDAK TERMASUK".
	cp.Facilities = lafayaClean(r.Includes)
	cp.Excludes = lafayaClean(r.Excludes)
	if len(cp.Facilities) == 0 || len(cp.Excludes) == 0 {
		inc, exc := lafayaSections(lines)
		if len(cp.Facilities) == 0 {
			cp.Facilities = inc
		}
		if len(cp.Excludes) == 0 {
			cp.Excludes = exc
		}
	}

	// Departure city only when the data says so.
	if r.DepartureCity != nil && strings.TrimSpace(*r.DepartureCity) != "" {
		cp.Airport = strings.TrimSpace(*r.DepartureCity)
	} else if len(r.Itinerary) > 0 {
		var day struct {
			Activity string `json:"activity"`
		}
		if json.Unmarshal(r.Itinerary[0], &day) == nil && strings.HasPrefix(strings.ToUpper(strings.TrimSpace(day.Activity)), "JAKARTA") {
			cp.Airport = "Jakarta"
		}
	}
	return cp, true
}

// lafayaAirline keeps the travel's wording ("QATAR AIRWAYS/middle east
// carrier" stays as such) but normalises a single known carrier name.
func lafayaAirline(s string) string {
	s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "."))
	s = lafayaParenRe.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, "/") {
		low := strings.ToLower(s)
		if strings.Contains(low, "egypt") {
			return "EgyptAir"
		}
		if a := KnownAirline(s); a != "" {
			return a
		}
		return s
	}
	parts := strings.Split(s, "/")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if a := KnownAirline(part); a != "" {
			parts[i] = a
		} else {
			parts[i] = part
		}
	}
	return strings.Join(parts, " / ")
}

func lafayaClean(in []string) []string {
	var out []string
	for _, s := range in {
		s = strings.Join(strings.Fields(s), " ")
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func lafayaSections(lines []string) (inc, exc []string) {
	mode := ""
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		up := strings.ToUpper(line)
		if i == len(lines)-1 && mode != "" {
			// Some descriptions are cut off mid-item ("Snack + Air M").
			break
		}
		switch {
		case strings.Contains(up, "TIDAK TERMASUK") || strings.Contains(up, "BELUM TERMASUK"):
			mode = "exc"
			continue
		case strings.Contains(up, "TERMASUK") && strings.HasSuffix(strings.TrimSpace(strings.TrimRight(up, " :")), "TERMASUK"):
			mode = "inc"
			continue
		}
		if mode == "" {
			continue
		}
		mark := strings.HasPrefix(line, "✅") || strings.HasPrefix(line, "❌")
		if !mark {
			if line == "" {
				continue
			}
			mode = ""
			continue
		}
		item := strings.TrimSpace(lafayaLeadEmoji.ReplaceAllString(line, ""))
		if item == "" {
			continue
		}
		if mode == "inc" {
			inc = append(inc, item)
		} else {
			exc = append(exc, item)
		}
	}
	return inc, exc
}
