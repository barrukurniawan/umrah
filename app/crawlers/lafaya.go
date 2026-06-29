package crawlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type LafayaParser struct {
	PackageIDs []string
}

type supabasePackage struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Price          int    `json:"price"`
	Duration       int    `json:"duration"`
	DepartureDate  string `json:"departure_date"`
	AvailableSeats int    `json:"available_seats"`
	TotalSeats     int    `json:"total_seats"`
	Description    string `json:"description"`
}

const (
	lafayaSupabaseURL = "https://yisgtmdzwklfupkfsvtb.supabase.co/rest/v1/packages"
	lafayaAPIKey      = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6Inlpc2d0bWR6d2tsZnVwa2ZzdnRiIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NTMyNTc5NjIsImV4cCI6MjA2ODgzMzk2Mn0.a6djeye_zYwVIsTvhVsU8W1r72NCfR6p2jWcNUV62F0"
)

func (p *LafayaParser) Crawl() ([]CrawledPackage, error) {
	if len(p.PackageIDs) == 0 {
		return nil, fmt.Errorf("no package IDs provided")
	}

	idFilter := strings.Join(p.PackageIDs, ",")
	url := fmt.Sprintf("%s?id=in.(%s)&select=*", lafayaSupabaseURL, idFilter)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("apikey", lafayaAPIKey)
	req.Header.Set("Authorization", "Bearer "+lafayaAPIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
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

	var packages []supabasePackage
	if err := json.Unmarshal(body, &packages); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	var result []CrawledPackage
	for _, pkg := range packages {
		if pkg.Type != "umrah" {
			continue
		}
		cp := p.parsePackage(pkg)
		if cp.Price > 0 {
			result = append(result, cp)
		}
	}

	return result, nil
}

func (p *LafayaParser) parsePackage(pkg supabasePackage) CrawledPackage {
	desc := pkg.Description

	airline := extractLafayaAirline(desc)
	hMakkah, hMadinah := extractLafayaHotels(desc)
	dates := extractLafayaDates(desc)

	seats := pkg.AvailableSeats
	if seats <= 0 {
		seats = pkg.TotalSeats
	}
	if seats <= 0 {
		seats = 90
	}

	price := pkg.Price
	if price < 1000000 {
		price = extractLafayaPrice(desc)
	}

	return CrawledPackage{
		TravelName:     "Lafaya Travel",
		PackageName:    pkg.Name,
		Price:          price,
		Duration:       pkg.Duration,
		Airline:        airline,
		HotelMakkah:    hMakkah,
		HotelMadinah:   hMadinah,
		DepartureDates: dates,
		Seats:          seats,
		Airport:        "Jakarta",
		URL:            fmt.Sprintf("https://lafayatravel.com/paket/%s", pkg.ID),
	}
}

func extractLafayaAirline(desc string) string {
	lower := strings.ToLower(desc)

	patterns := []struct {
		key  string
		name string
	}{
		{"egyptair", "EgyptAir"},
		{"qatar airways", "Qatar Airways"},
		{"emirates", "Emirates"},
		{"oman air", "Oman Air"},
		{"garuda indonesia", "Garuda Indonesia"},
		{"saudia", "Saudia"},
		{"etihad", "Etihad Airways"},
		{"turkish airlines", "Turkish Airlines"},
	}

	for _, p := range patterns {
		if strings.Contains(lower, p.key) {
			return p.name
		}
	}

	return ""
}

func extractLafayaHotels(desc string) (makkah, madinah string) {
	lines := strings.Split(desc, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) < 5 {
			continue
		}

		lower := strings.ToLower(line)

		if makkah == "" && (isMakkahLine(lower) || strings.Contains(lower, "mekk")) {
			name := extractHotelFromLine(line, "makkah")
			if name == "" {
				name = extractHotelFromLine(line, "mekkah")
			}
			if name == "" {
				name = extractHotelFromLine(line, "mekah")
			}
			if name != "" {
				makkah = name
			}
		}

		if madinah == "" && isMadinahLine(lower) {
			name := extractHotelFromLine(line, "madinah")
			if name == "" {
				name = extractHotelFromLine(line, "madina")
			}
			if name != "" {
				madinah = name
			}
		}
	}

	return makkah, madinah
}

func extractHotelFromLine(line, cityKeyword string) string {
	re := regexp.MustCompile(`(?i)\d+\s*N\s*` + cityKeyword + `\s*⭐+\s*([^/\n]+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		name := strings.TrimSpace(matches[1])
		if len(name) > 2 {
			return name
		}
	}

	re2 := regexp.MustCompile(`(?i)` + cityKeyword + `\s*[:\-]?\s*⭐+\s*([^/\n]+)`)
	matches2 := re2.FindStringSubmatch(line)
	if len(matches2) > 1 {
		name := strings.TrimSpace(matches2[1])
		if len(name) > 2 {
			return name
		}
	}

	re3 := regexp.MustCompile(`(?i)` + cityKeyword + `\s*[:\-]?\s*([A-Z][a-zA-Z\s]+?)(?:\s*/|\s*⭐|\s*$)`)
	matches3 := re3.FindStringSubmatch(line)
	if len(matches3) > 1 {
		name := strings.TrimSpace(matches3[1])
		if len(name) > 3 {
			return name
		}
	}

	return ""
}

func isMakkahLine(s string) bool {
	return strings.Contains(s, "makkah") || strings.Contains(s, "mekkah") || strings.Contains(s, "mekah")
}

func isMadinahLine(s string) bool {
	return strings.Contains(s, "madinah") || strings.Contains(s, "madina") || strings.Contains(s, "medina")
}

func extractLafayaDates(desc string) []string {
	var dates []string

	lines := strings.Split(desc, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) < 6 {
			continue
		}

		re := regexp.MustCompile(`(\d{1,2})\s+(Januari|Februari|Maret|April|Mei|Juni|Juli|Agustus|September|Oktober|November|Desember|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Agu|Sep|Okt|Oct|Nov|Des|Dec)\s+(\d{4})`)
		matches := re.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			if len(m) >= 4 {
				dateStr := fmt.Sprintf("%s %s %s", m[1], m[2], m[3])
				dates = append(dates, dateStr)
			}
		}

		re2 := regexp.MustCompile(`(\d{1,2})\s+(Januari|Februari|Maret|April|Mei|Juni|Juli|Agustus|September|Oktober|November|Desember|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Agu|Sep|Okt|Oct|Nov|Des|Dec)`)
		matches2 := re2.FindAllStringSubmatch(line, -1)
		for _, m := range matches2 {
			if len(m) >= 3 {
				hasYear := false
				for _, existing := range dates {
					if strings.Contains(existing, m[1]) && strings.Contains(existing, m[2]) {
						hasYear = true
						break
					}
				}
				if !hasYear {
					dateStr := fmt.Sprintf("%s %s 2026", m[1], m[2])
					dates = append(dates, dateStr)
				}
			}
		}
	}

	return dates
}

func extractLafayaPrice(desc string) int {
	re := regexp.MustCompile(`(?i)quad[:\s]*rp\.?\s*([\d.,]+)`)
	matches := re.FindStringSubmatch(desc)
	if len(matches) > 1 {
		return ParsePrice(matches[1])
	}

	re2 := regexp.MustCompile(`(?i)start\s*from[:\s]*rp\.?\s*([\d.,]+)`)
	matches2 := re2.FindStringSubmatch(desc)
	if len(matches2) > 1 {
		return ParsePrice(matches2[1])
	}

	return 0
}
