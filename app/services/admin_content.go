package services

import (
	"fmt"
	"sort"
	"time"

	"umrah/app/repositories"
)

// Data untuk generator gambar postingan (admin, tab Konten). Semua angka diambil dari data
// crawl terbaru; gambar menuliskan waktu data supaya pembaca tahu harga bisa berubah.

type ContentDep struct {
	Travel   string `json:"travel"`
	Slug     string `json:"slug"`
	Initials string `json:"initials"`
	Package  string `json:"package"`
	Date     string `json:"date"`
	Price    int    `json:"price"`
	Duration int    `json:"duration"`
	Airline  string `json:"airline"`
	Direct   bool   `json:"direct"`
	Hotel    string `json:"hotel"`
	Stars    int    `json:"stars"`
	Seats    int    `json:"seats"`
	From     string `json:"from"`
}

type ContentMonth struct {
	Key      string       `json:"key"`
	Title    string       `json:"title"`
	Count    int          `json:"count"`   // keberangkatan yang masih bisa dipesan
	Travels  int          `json:"travels"` // jumlah travel
	Min      int          `json:"min"`
	Median   int          `json:"median"`
	Cheapest []ContentDep `json:"cheapest"`
}

type ContentData struct {
	Updated  string         `json:"updated"`
	Months   []ContentMonth `json:"months"`
	LowSeats []ContentDep   `json:"lowSeats"`
	Ramadhan []ContentDep   `json:"ramadhan"`
	RamLabel string         `json:"ramLabel"`
	Travels  int            `json:"travels"`
	Deps     int            `json:"deps"`
}

// pickCheapest: termurah dulu, satu entri per paket, maksimal 2 per travel supaya beragam.
func pickCheapest(deps []ContentDep, n int) []ContentDep {
	sort.SliceStable(deps, func(i, j int) bool {
		if deps[i].Price != deps[j].Price {
			return deps[i].Price < deps[j].Price
		}
		return deps[i].Date < deps[j].Date
	})
	seenPkg := map[string]bool{}
	perTravel := map[string]int{}
	var out []ContentDep
	for _, d := range deps {
		// Travel kadang memasang paket yang sama dua kali (nama/tanggal/harga sama): ambil sekali.
		k := d.Travel + "|" + d.Package
		k2 := fmt.Sprintf("%s|%s|%d", d.Travel, d.Date, d.Price)
		if seenPkg[k] || seenPkg[k2] || perTravel[d.Travel] >= 2 {
			continue
		}
		seenPkg[k], seenPkg[k2] = true, true
		perTravel[d.Travel]++
		out = append(out, d)
		if len(out) == n {
			break
		}
	}
	return out
}

func AdminContent(updated time.Time) ContentData {
	wib := time.FixedZone("WIB", 7*3600)
	now := time.Now().In(wib)
	today := now.Format("2006-01-02")
	// Ramadhan berikutnya (atau yang sedang berjalan) dari daftar momen.
	var ramFrom, ramTo string
	for _, e := range events {
		if e.Kind == "ramadhan" && e.To >= today {
			ramFrom, ramTo = e.From, e.To
			break
		}
	}

	byMonth := map[string][]ContentDep{}
	var low, ram []ContentDep
	travelSet := map[string]bool{}
	total := 0
	soon := now.AddDate(0, 0, 120).Format("2006-01-02")
	for _, p := range repositories.AllPackages() {
		for _, d := range p.Details {
			if d.DepartureDate <= today || d.IsFull {
				continue
			}
			price := d.Price
			if price == 0 {
				price = p.Price
			}
			if price <= 0 {
				continue
			}
			cd := ContentDep{
				Travel: TravelDisplayName(p.Travel.Name), Slug: TravelSlug(p.Travel.Name), Initials: TravelInitials(p.Travel.Name),
				Package: StripEmoji(p.Name), Date: d.DepartureDate, Price: price, Duration: p.Duration,
				Airline: p.Airline, Direct: p.FlightType == "direct", Hotel: d.HotelMakkah, Stars: d.StarsMakkah,
				Seats: d.AvailableQuota, From: d.DepartureLocation,
			}
			if d.StarsMakkahAssumed {
				cd.Stars = 0
			}
			travelSet[cd.Travel] = true
			total++
			byMonth[d.DepartureDate[:7]] = append(byMonth[d.DepartureDate[:7]], cd)
			if d.AvailableQuota > 0 && d.AvailableQuota <= 9 && d.DepartureDate <= soon {
				low = append(low, cd)
			}
			if ramFrom != "" && d.DepartureDate >= ramFrom && d.DepartureDate <= ramTo {
				ram = append(ram, cd)
			}
		}
	}

	v := ContentData{Updated: updated.In(wib).Format("02 Jan 2006 15:04"), Travels: len(travelSet), Deps: total}
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, wib)
	for i := 0; i < CalendarMonths; i++ {
		m := first.AddDate(0, i, 0)
		key := m.Format("2006-01")
		deps := byMonth[key]
		if len(deps) == 0 {
			continue
		}
		ts := map[string]bool{}
		prices := make([]int, 0, len(deps))
		for _, d := range deps {
			ts[d.Travel] = true
			prices = append(prices, d.Price)
		}
		sort.Ints(prices)
		v.Months = append(v.Months, ContentMonth{
			Key: key, Title: fmt.Sprintf("%s %d", idMonths[m.Month()], m.Year()),
			Count: len(deps), Travels: len(ts), Min: prices[0], Median: prices[len(prices)/2],
			Cheapest: pickCheapest(deps, 5),
		})
	}
	// Seat menipis: paling sedikit dulu, lalu tanggal terdekat; satu per paket.
	sort.SliceStable(low, func(i, j int) bool {
		if low[i].Seats != low[j].Seats {
			return low[i].Seats < low[j].Seats
		}
		return low[i].Date < low[j].Date
	})
	seen := map[string]bool{}
	for _, d := range low {
		k := d.Travel + "|" + d.Package
		k2 := fmt.Sprintf("%s|%s|%d", d.Travel, d.Date, d.Price)
		if seen[k] || seen[k2] {
			continue
		}
		seen[k], seen[k2] = true, true
		v.LowSeats = append(v.LowSeats, d)
		if len(v.LowSeats) == 5 {
			break
		}
	}
	v.Ramadhan = pickCheapest(ram, 5)
	if ramFrom != "" {
		f, _ := time.Parse("2006-01-02", ramFrom)
		t, _ := time.Parse("2006-01-02", ramTo)
		v.RamLabel = fmt.Sprintf("%d %s – %d %s %d", f.Day(), idMonths[f.Month()][:3], t.Day(), idMonths[t.Month()][:3], t.Year())
	}
	return v
}
