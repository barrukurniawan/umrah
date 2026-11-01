package repositories

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"umrah/app/crawlers"
)

// hotelOverride mengisi hotel untuk paket yang websitenya tidak mencantumkan hotel
// (hasil riset manual: flyer, halaman lain travel). Disimpan di
// config/hotel_overrides.json dan diterapkan setiap import saat boot, jadi tidak
// hilang saat crawl ulang. Field match yang diisi harus cocok semua
// (substring, case-insensitive; departure_date persis YYYY-MM-DD).
// Override hanya mengisi hotel yang kosong, tidak menimpa data crawler.
type hotelOverride struct {
	TravelName   string `json:"travel_name"`
	URLContains  string `json:"url_contains,omitempty"`
	NameContains string `json:"name_contains,omitempty"`
	Airline      string `json:"airline,omitempty"`
	Departure    string `json:"departure_date,omitempty"`
	HotelMakkah  string `json:"hotel_makkah"`
	StarsMakkah  int    `json:"stars_makkah,omitempty"`
	HotelMadinah string `json:"hotel_madinah"`
	StarsMadinah int    `json:"stars_madinah,omitempty"`
	Source       string `json:"source"`
	Note         string `json:"note,omitempty"`
}

func loadHotelOverrides() []hotelOverride {
	path := os.Getenv("HOTEL_OVERRIDES")
	if path == "" {
		path = "config/hotel_overrides.json"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []hotelOverride
	if err := json.Unmarshal(data, &out); err != nil {
		log.Printf("[import] hotel overrides %s tidak valid: %v", path, err)
		return nil
	}
	return out
}

func containsFold(s, sub string) bool {
	return sub == "" || strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

// findHotelOverride mengembalikan override pertama yang cocok (urutan file = prioritas).
func findHotelOverride(list []hotelOverride, travelName string, cp crawlers.CrawledPackage, depDate string) (hotelOverride, bool) {
	for _, o := range list {
		if !strings.EqualFold(o.TravelName, travelName) {
			continue
		}
		if o.URLContains == "" && o.NameContains == "" && o.Departure == "" {
			continue // terlalu luas, abaikan
		}
		if containsFold(cp.URL, o.URLContains) && containsFold(cp.PackageName, o.NameContains) &&
			containsFold(cp.Airline, o.Airline) && (o.Departure == "" || o.Departure == depDate) {
			return o, true
		}
	}
	return hotelOverride{}, false
}
