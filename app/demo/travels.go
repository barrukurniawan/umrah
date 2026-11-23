package demo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// PackagesFile: paket travel fiktif ditulis sebagai satu "hasil crawl" sehingga ikut dibangun ke
// database paket seperti travel lain (pencarian, kalender, ♡ simpan). Diawali "all_zz" supaya
// selalu dibaca terakhir.
const PackagesFile = "output/all_zz_demo_portal.json"

// ---- cache travel per nama (dipakai fungsi template di setiap kartu paket) ----

var (
	cacheMu    sync.RWMutex
	travelByNm = map[string]*Travel{}
	termsBySl  = map[string]*TravelTerms{}
	portalPkgs = map[string]bool{} // "travel|nama paket" yang datang dari portal
)

func refreshCache() {
	if DB == nil {
		return
	}
	var ts []Travel
	DB.Find(&ts)
	var terms []TravelTerms
	DB.Find(&terms)
	var pp []PortalPackage
	DB.Where("active = ?", true).Find(&pp)
	byName := map[string]*Travel{}
	for i := range ts {
		byName[ts[i].Name] = &ts[i]
	}
	bySlug := map[string]*TravelTerms{}
	for i := range terms {
		bySlug[terms[i].TravelSlug] = &terms[i]
	}
	slugName := map[string]string{}
	for _, t := range ts {
		slugName[t.Slug] = t.Name
	}
	pk := map[string]bool{}
	for _, p := range pp {
		pk[slugName[p.TravelSlug]+"|"+p.Name] = true
	}
	cacheMu.Lock()
	travelByNm, termsBySl, portalPkgs = byName, bySlug, pk
	cacheMu.Unlock()
	refreshPlacements()
}

// TravelByName: travel mitra/terdaftar untuk nama travel di data paket (nil bila tidak ada).
func TravelByName(name string) *Travel {
	if !enabled {
		return nil
	}
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	return travelByNm[name]
}

func TravelBySlug(slug string) *Travel {
	if !enabled {
		return nil
	}
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	for _, t := range travelByNm {
		if t.Slug == slug {
			return t
		}
	}
	return nil
}

func TermsFor(slug string) *TravelTerms {
	if !enabled {
		return nil
	}
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	return termsBySl[slug]
}

// IsPortalPackage: paket ini diisi travel lewat portal (bukan hasil scraping).
func IsPortalPackage(travel, pkg string) bool {
	if !enabled {
		return false
	}
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	return portalPkgs[travel+"|"+pkg]
}

func AllTravels() []Travel {
	var ts []Travel
	if DB != nil {
		DB.Order("fictional DESC, partner DESC, name").Find(&ts)
	}
	return ts
}

func PartnerTravels() []Travel {
	var ts []Travel
	if DB != nil {
		DB.Where("partner = ?", true).Order("name").Find(&ts)
	}
	return ts
}

// ---- paket portal -> berkas crawl ----

type crawlDeparture struct {
	Date        string `json:"date"`
	ReturnDate  string `json:"return_date,omitempty"`
	Price       int64  `json:"price,omitempty"`
	PriceTriple int64  `json:"price_triple,omitempty"`
	PriceDouble int64  `json:"price_double,omitempty"`
	Seats       int    `json:"seats,omitempty"`
	SeatsTotal  int    `json:"seats_total,omitempty"`
	Full        bool   `json:"full,omitempty"`
}

type crawlPackage struct {
	TravelName      string           `json:"travel_name"`
	PackageName     string           `json:"package_name"`
	Price           int64            `json:"price"`
	Duration        int              `json:"duration"`
	Airline         string           `json:"airline"`
	HotelMakkah     string           `json:"hotel_makkah"`
	HotelMadinah    string           `json:"hotel_madinah"`
	DepartureDates  []string         `json:"departure_dates"`
	Seats           int              `json:"seats"`
	Airport         string           `json:"airport"`
	URL             string           `json:"url"`
	Departures      []crawlDeparture `json:"departures,omitempty"`
	StarsMakkah     int              `json:"stars_makkah,omitempty"`
	StarsMadinah    int              `json:"stars_madinah,omitempty"`
	DistanceMakkah  int              `json:"distance_makkah,omitempty"`
	DistanceMadinah int              `json:"distance_madinah,omitempty"`
	Direct          *bool            `json:"direct,omitempty"`
	TransitVia      string           `json:"transit_via,omitempty"`
	Facilities      []string         `json:"facilities,omitempty"`
	Excludes        []string         `json:"excludes,omitempty"`
	PriceTriple     int64            `json:"price_triple,omitempty"`
	PriceDouble     int64            `json:"price_double,omitempty"`
	DownPayment     int64            `json:"down_payment,omitempty"`
	Guide           string           `json:"guide,omitempty"`
}

type crawlResult struct {
	Timestamp string         `json:"timestamp"`
	Site      string         `json:"site"`
	URL       string         `json:"url"`
	Packages  []crawlPackage `json:"packages"`
}

func lines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// WritePackagesFile menulis semua paket portal aktif ke output/ (satu hasil per travel) dan
// memperbarui cache. Pemanggil yang mengubah paket di portal lalu memanggil repositories.Reload().
func WritePackagesFile() error {
	if DB == nil {
		return nil
	}
	var ts []Travel
	DB.Find(&ts)
	var results []crawlResult
	now := time.Now().In(WIB).Format(time.RFC3339)
	for _, t := range ts {
		var pkgs []PortalPackage
		DB.Preload("Departures").Where("travel_slug = ? AND active = ?", t.Slug, true).Order("price").Find(&pkgs)
		if len(pkgs) == 0 {
			continue
		}
		dp := int64(0)
		if tm := TermsFor(t.Slug); tm != nil {
			dp = tm.DPNormalTravel
		} else {
			var tt TravelTerms
			if DB.Where("travel_slug = ?", t.Slug).Take(&tt).Error == nil {
				dp = tt.DPNormalTravel
			}
		}
		r := crawlResult{Timestamp: now, Site: t.Name + " (portal mitra)", URL: "portal:" + t.Slug}
		for _, p := range pkgs {
			direct := p.Direct
			cp := crawlPackage{
				TravelName: t.Name, PackageName: p.Name, Price: p.Price, Duration: p.Duration, Airline: p.Airline,
				HotelMakkah: p.HotelMakkah, HotelMadinah: p.HotelMadinah, DepartureDates: []string{},
				URL: fmt.Sprintf("/travel/%s#paket-%d", t.Slug, p.ID), StarsMakkah: p.StarsMakkah, StarsMadinah: p.StarsMadinah,
				DistanceMakkah: p.DistMakkah, DistanceMadinah: p.DistMadinah, Direct: &direct, TransitVia: p.TransitVia,
				Facilities: lines(p.Facilities), Excludes: lines(p.Excludes), PriceTriple: p.PriceTriple, PriceDouble: p.PriceDouble,
				DownPayment: dp, Guide: p.Guide,
			}
			sort.Slice(p.Departures, func(i, j int) bool { return p.Departures[i].Date < p.Departures[j].Date })
			for _, d := range p.Departures {
				price := d.Price
				if price == 0 {
					price = p.Price
				}
				cd := crawlDeparture{Date: d.Date, ReturnDate: d.ReturnDate, Price: price, Seats: d.SeatsLeft, SeatsTotal: d.SeatsTotal, Full: d.SeatsLeft <= 0}
				if p.PriceTriple > 0 {
					cd.PriceTriple = price + (p.PriceTriple - p.Price)
				}
				if p.PriceDouble > 0 {
					cd.PriceDouble = price + (p.PriceDouble - p.Price)
				}
				cp.Departures = append(cp.Departures, cd)
			}
			r.Packages = append(r.Packages, cp)
		}
		results = append(results, r)
	}
	b, err := json.MarshalIndent(results, "", " ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(PackagesFile), 0755); err != nil {
		return err
	}
	tmp := PackagesFile + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, PackagesFile); err != nil {
		return err
	}
	refreshCache()
	return nil
}

// AddDays untuk tanggal "2006-01-02".
func AddDays(date string, n int) string {
	t, err := time.ParseInLocation("2006-01-02", date, WIB)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, n).Format("2006-01-02")
}

// DaysUntil: selisih hari dari hari ini (WIB) ke tanggal.
func DaysUntil(date string) int {
	t, err := time.ParseInLocation("2006-01-02", date, WIB)
	if err != nil {
		return 0
	}
	today, _ := time.ParseInLocation("2006-01-02", Today(), WIB)
	return int(t.Sub(today).Hours() / 24)
}
