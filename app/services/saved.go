package services

import (
	"strings"
	"time"

	"umrah/app/models"
	"umrah/app/repositories"
)

// PackageKey: identitas paket yang tetap sama walau data crawl dimuat ulang (ID database berubah).
// Paket dengan nama sama di travel yang sama dianggap satu "lini paket" (jadwalnya bisa banyak).
func PackageKey(travel, name string) string {
	norm := func(s string) string { return strings.Join(strings.Fields(strings.ToLower(s)), " ") }
	return norm(travel) + "|" + norm(name)
}

// CurrentPackage: kondisi terkini satu lini paket (harga termurah & jadwal yang belum lewat).
type CurrentPackage struct {
	Key          string
	Travel       string
	Name         string
	MinPrice     int
	DownPayment  int // DP per orang (dari data paket)
	Duration     int
	Airline      string
	HotelMakkah  string
	HotelMadinah string
	Departures   []string // "2006-01-02", terurut
	DatesLabel   string
}

func today() string { return time.Now().In(time.FixedZone("WIB", 7*3600)).Format("2006-01-02") }

// CurrentByKeys mengembalikan paket yang masih punya jadwal mendatang, dipetakan per key.
// keys nil = semua paket.
func CurrentByKeys(keys map[string]bool) map[string]*CurrentPackage {
	var all []models.Package
	repositories.DB.Preload("Travel").Preload("Details").Find(&all)
	t := today()
	out := map[string]*CurrentPackage{}
	for _, p := range all {
		key := PackageKey(p.Travel.Name, p.Name)
		if keys != nil && !keys[key] {
			continue
		}
		var upcoming []models.DetailPackage
		for _, d := range p.Details {
			if d.DepartureDate != "" && d.DepartureDate >= t {
				upcoming = append(upcoming, d)
			}
		}
		if len(upcoming) == 0 {
			continue
		}
		cp, ok := out[key]
		if !ok {
			cp = &CurrentPackage{Key: key, Travel: p.Travel.Name, Name: p.Name, MinPrice: p.Price, DownPayment: p.DownPayment, Duration: p.Duration, Airline: p.Airline,
				HotelMakkah: upcoming[0].HotelMakkah, HotelMadinah: upcoming[0].HotelMadinah}
			out[key] = cp
		}
		if p.Price < cp.MinPrice {
			cp.MinPrice = p.Price
		}
		for _, d := range upcoming {
			cp.Departures = append(cp.Departures, d.DepartureDate)
		}
	}
	for _, cp := range out {
		cp.Departures = uniqueSorted(cp.Departures)
		var details []models.DetailPackage
		for _, d := range cp.Departures {
			details = append(details, models.DetailPackage{DepartureDate: d})
		}
		cp.DatesLabel = formatDepartureDatesStr(details)
	}
	return out
}

func uniqueSorted(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
