package repositories

import (
	"encoding/json"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"umrah/app/crawlers"
	"umrah/app/models"

	"gorm.io/gorm"
)

// crawlResultLite adalah satu hasil crawl per site (isi file output/all_*.json).
type crawlResultLite struct {
	Site     string
	URL      string
	Packages []crawlers.CrawledPackage
	Error    string
}

// Durasi dari nama paket: "9 Hari", "12D", "13 Days". Rentang ("16-21 Hari") sengaja
// tidak dicocokkan karena durasinya tidak pasti.
var nameDurationRe = regexp.MustCompile(`(?i)(?:^|[^\d-])(\d{1,2})\s*(?:hari|days?|d)\b`)

func durationFromName(name string) int {
	m := nameDurationRe.FindStringSubmatch(name)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	if n < 5 || n > 45 {
		return 0
	}
	return n
}

func isoDate(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if _, err := time.Parse("2006-01-02", s); err == nil {
		return s, true
	}
	return parseCrawlerDate(s)
}

func daysBetween(a, b string) int {
	ta, err1 := time.Parse("2006-01-02", a)
	tb, err2 := time.Parse("2006-01-02", b)
	if err1 != nil || err2 != nil {
		return 0
	}
	return int(tb.Sub(ta).Hours() / 24)
}

// normalizeDepartures mengembalikan keberangkatan unik (per tanggal, urut) dari
// field baru Departures, atau dari DepartureDates lama.
func normalizeDepartures(cp crawlers.CrawledPackage) []crawlers.CrawledDeparture {
	byDate := map[string]crawlers.CrawledDeparture{}
	add := func(d crawlers.CrawledDeparture) {
		date, ok := isoDate(d.Date)
		if !ok {
			return
		}
		d.Date = date
		if d.ReturnDate != "" {
			if r, ok := isoDate(d.ReturnDate); ok && r > date {
				d.ReturnDate = r
			} else {
				d.ReturnDate = ""
			}
		}
		if prev, ok := byDate[date]; ok {
			byDate[date] = mergeDeparture(prev, d)
			return
		}
		byDate[date] = d
	}
	if len(cp.Departures) > 0 {
		for _, d := range cp.Departures {
			add(d)
		}
	} else {
		for _, s := range cp.DepartureDates {
			add(crawlers.CrawledDeparture{Date: s})
		}
	}
	out := make([]crawlers.CrawledDeparture, 0, len(byDate))
	for _, d := range byDate {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out
}

// mergeDeparture mengisi field kosong a dari b (data duplikat untuk tanggal sama).
func mergeDeparture(a, b crawlers.CrawledDeparture) crawlers.CrawledDeparture {
	if a.ReturnDate == "" {
		a.ReturnDate = b.ReturnDate
	}
	if a.Price == 0 {
		a.Price = b.Price
	}
	if a.PriceTriple == 0 {
		a.PriceTriple = b.PriceTriple
	}
	if a.PriceDouble == 0 {
		a.PriceDouble = b.PriceDouble
	}
	if a.Seats == 0 {
		a.Seats = b.Seats
	}
	if a.SeatsTotal == 0 {
		a.SeatsTotal = b.SeatsTotal
	}
	a.Full = a.Full || b.Full
	if a.HotelMakkah == "" {
		a.HotelMakkah = b.HotelMakkah
	}
	if a.HotelMadinah == "" {
		a.HotelMadinah = b.HotelMadinah
	}
	if a.Airline == "" {
		a.Airline = b.Airline
	}
	if a.Airport == "" {
		a.Airport = b.Airport
	}
	return a
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func firstPositive(vals ...int) int {
	for _, v := range vals {
		if v > 0 {
			return v
		}
	}
	return 0
}

type pkgAgg struct {
	travel string
	cp     crawlers.CrawledPackage
	deps   []crawlers.CrawledDeparture
}

// buildFromCrawl menulis travel/paket/detail ke DB dari hasil crawl.
//
// Prinsip: hanya data yang diberikan travel yang disimpan. Field yang tidak
// diketahui dibiarkan nol/kosong supaya UI menyembunyikannya (bukan diisi angka
// karangan seperti kuota 30/40 atau lokasi Jakarta).
//
// Duplikat dibuang di dua level: paket yang sama (URL detail sama, atau nama+harga
// sama jika URL tidak unik) digabung, dan keberangkatan dengan travel+nama+tanggal+
// harga yang sama hanya disimpan sekali.
func buildFromCrawl(db *gorm.DB, results []crawlResultLite, dpOverrides map[string]int, ratingOverrides map[string]float64) (pkgCount, detailCount, travelCount int) {
	overrides := loadHotelOverrides()
	overrideCount, dupDeps := 0, 0

	aggs := map[string]*pkgAgg{}
	var order []string
	for _, result := range results {
		if result.Error != "" {
			continue
		}
		for _, cp := range result.Packages {
			if cp.Price < 5000000 || isNonUmrah(cp.PackageName) {
				continue
			}
			travelName := firstNonEmpty(cp.TravelName, result.Site)
			key := travelName + "|" + cp.URL
			if cp.URL == "" || strings.TrimRight(cp.URL, "/") == strings.TrimRight(result.URL, "/") {
				key = travelName + "|" + cp.PackageName + "|" + strconv.Itoa(cp.Price)
			}
			a, ok := aggs[key]
			if !ok {
				a = &pkgAgg{travel: travelName, cp: cp}
				aggs[key] = a
				order = append(order, key)
			}
			// Cap tiap tanggal dengan nilai paket asalnya, supaya harga/hotel per tanggal
			// tidak hilang saat beberapa entri satu halaman produk digabung.
			ds := normalizeDepartures(cp)
			for i := range ds {
				d := &ds[i]
				d.Price = firstPositive(d.Price, cp.Price)
				d.PriceTriple = firstPositive(d.PriceTriple, cp.PriceTriple)
				d.PriceDouble = firstPositive(d.PriceDouble, cp.PriceDouble)
				d.HotelMakkah = firstNonEmpty(d.HotelMakkah, cp.HotelMakkah)
				d.HotelMadinah = firstNonEmpty(d.HotelMadinah, cp.HotelMadinah)
				d.Airline = firstNonEmpty(d.Airline, cp.Airline)
				d.Airport = firstNonEmpty(d.Airport, cp.Airport)
				d.SeatsTotal = firstPositive(d.SeatsTotal, cp.SeatsTotal)
				d.Full = d.Full || cp.Full
				if d.Seats == 0 && len(ds) == 1 {
					d.Seats = cp.Seats // seat level paket hanya berlaku jika satu tanggal
				}
			}
			a.deps = append(a.deps, ds...)
		}
	}

	stars := buildHotelStars(results, overrides)
	starFill := 0

	travelMap := map[string]*models.Travel{}
	seenDep := map[string]bool{} // travel|nama|tanggal|harga

	for _, key := range order {
		a := aggs[key]
		cp := a.cp
		// gabungkan departure dari paket duplikat
		merged := normalizeDepartures(crawlers.CrawledPackage{Departures: a.deps})

		var deps []crawlers.CrawledDeparture
		for _, d := range merged {
			price := firstPositive(d.Price, cp.Price)
			k := a.travel + "|" + strings.ToLower(cp.PackageName) + "|" + d.Date + "|" + strconv.Itoa(price)
			if seenDep[k] {
				dupDeps++
				continue
			}
			seenDep[k] = true
			deps = append(deps, d)
		}
		if len(deps) == 0 {
			continue
		}

		travel, ok := travelMap[a.travel]
		if !ok {
			rating := 4.5
			if r, ok := ratingOverrides[a.travel]; ok {
				rating = r
			}
			travel = &models.Travel{Name: a.travel, Rating: rating}
			db.Create(travel)
			travelMap[a.travel] = travel
		}

		// Harga paket = termurah di antara tanggal (jika per tanggal berbeda).
		price := cp.Price
		for _, d := range deps {
			if d.Price > 0 && d.Price < price {
				price = d.Price
			}
		}

		duration := cp.Duration
		if duration == 0 {
			duration = durationFromName(cp.PackageName)
		}
		if duration == 0 {
			for _, d := range deps {
				if n := daysBetween(d.Date, d.ReturnDate); n > 0 {
					duration = n + 1
					break
				}
			}
		}

		// Tidak disebut direct = diasumsikan transit (umumnya begitu), ditandai FlightAssumed.
		flight, flightAssumed := "transit", cp.Direct == nil
		if cp.Direct != nil && *cp.Direct {
			flight = "direct"
		}

		dp := cp.DownPayment
		if dp == 0 {
			dp = dpOverrides[a.travel] // kebijakan DP travel yang sudah diriset
		}
		dpAssumed := false
		if dp == 0 {
			// Travel tidak menyebut DP: perkiraan 20% harga, dibulatkan ke 100 ribu.
			dp, dpAssumed = (price/5+50000)/100000*100000, true
		}

		facilities, _ := json.Marshal(nonNil(cp.Facilities))
		excludes, _ := json.Marshal(nonNil(cp.Excludes))

		airline := cp.Airline
		if airline == "" {
			for _, d := range deps {
				if d.Airline != "" {
					airline = d.Airline
					break
				}
			}
		}
		if airline == "" {
			// Maskapai dari flyer travel bila halaman paket tidak menyebutnya.
			if o, ok := findHotelOverride(overrides, a.travel, cp, deps[0].Date); ok && o.SetAirline != "" {
				airline = o.SetAirline
			}
		}

		pkg := models.Package{
			TravelID:        travel.ID,
			Name:            fixPackageName(cp.PackageName, deps),
			Price:           price,
			HotelDistance:   cp.DistanceMakkah,
			Duration:        duration,
			Airline:         airline,
			IsDirect:        flight == "direct",
			FlightType:      flight,
			FlightAssumed:   flightAssumed,
			TransitVia:      cp.TransitVia,
			DownPayment:     dp,
			DPAssumed:       dpAssumed,
			Guide:           cp.Guide,
			Facilities:      string(facilities),
			Excludes:        string(excludes),
			GroupSize:       cp.SeatsTotal,
			SunnahScore:     5,
			URL:             cp.URL,
			DistanceMakkah:  cp.DistanceMakkah,
			DistanceMadinah: cp.DistanceMadinah,
		}
		db.Create(&pkg)
		pkgCount++

		for _, d := range deps {
			hm := cleanHotelName(firstNonEmpty(d.HotelMakkah, cp.HotelMakkah))
			hd := cleanHotelName(firstNonEmpty(d.HotelMadinah, cp.HotelMadinah))
			sm, sd := cp.StarsMakkah, cp.StarsMadinah
			if o, ok := findHotelOverride(overrides, a.travel, cp, d.Date); ok && (hm == "" || hd == "" || o.ReplaceHotels) {
				// Biasanya hanya mengisi yang kosong; replace_hotels untuk data travel yang salah.
				if o.HotelMakkah != "" && (hm == "" || o.ReplaceHotels) {
					hm, sm = o.HotelMakkah, o.StarsMakkah
				}
				if o.HotelMadinah != "" && (hd == "" || o.ReplaceHotels) {
					hd, sd = o.HotelMadinah, o.StarsMadinah
				}
				overrideCount++
			}

			// Bintang adalah fakta hotel: bila paket ini tidak menyebutnya, ambil dari
			// travel lain yang menyebut bintang hotel yang sama.
			if sm == 0 && hm != "" {
				if sm = stars[hotelKey(hm)]; sm > 0 {
					starFill++
				}
			}
			if sd == 0 && hd != "" {
				if sd = stars[hotelKey(hd)]; sd > 0 {
					starFill++
				}
			}

			// Masih tidak diketahui: default bintang 3 (umum untuk hotel paket umroh), ditandai perkiraan.
			smAssumed, sdAssumed := sm == 0 && hm != "", sd == 0 && hd != ""
			if smAssumed {
				sm = 3
			}
			if sdAssumed {
				sd = 3
			}

			ret := d.ReturnDate
			if ret == "" && duration > 0 {
				// "9 hari" = hari ke-1 s/d ke-9 (konvensi situs travel, mis. 14 Jun - 22 Jun).
				ret = addDays(d.Date, duration-1)
			}

			quad := firstPositive(d.Price, price)
			addon := func(p int) int {
				if p > quad {
					return p - quad
				}
				return 0
			}
			triple := addon(firstPositive(d.PriceTriple, cp.PriceTriple))
			double := addon(firstPositive(d.PriceDouble, cp.PriceDouble))
			// Harga utama = Quad. Bila travel tidak memberi harga upgrade kamar, pakai
			// default pasar (ditandai AddonAssumed agar UI menyebutnya perkiraan).
			roomType := "Quad"
			addonAssumed := triple == 0 && double == 0
			if addonAssumed {
				triple, double = defaultAddonTriple, defaultAddonDouble
			}

			seats := d.Seats

			detailPrice := 0
			if d.Price > 0 && d.Price != price {
				detailPrice = d.Price
			}

			db.Create(&models.DetailPackage{
				PackageID:           pkg.ID,
				DepartureDate:       d.Date,
				ReturnDate:          ret,
				HotelMakkah:         hm,
				HotelMadinah:        hd,
				StarsMakkah:         sm,
				StarsMadinah:        sd,
				RoomType:            roomType,
				TotalQuota:          d.SeatsTotal,
				AvailableQuota:      seats,
				IsFull:              d.Full,
				DepartureLocation:   firstNonEmpty(d.Airport, cp.Airport),
				AddonTriple:         triple,
				AddonDouble:         double,
				Guide:               cp.Guide,
				Price:               detailPrice,
				AddonAssumed:        addonAssumed,
				StarsMakkahAssumed:  smAssumed,
				StarsMadinahAssumed: sdAssumed,
			})
			detailCount++
		}
	}

	log.Printf("[import] hotel override diterapkan ke %d detail (%d aturan); %d keberangkatan duplikat dibuang; %d bintang hotel diisi dari travel lain (%d hotel dikenal)", overrideCount, len(overrides), dupDeps, starFill, len(stars))
	return pkgCount, detailCount, len(travelMap)
}

// Default upgrade kamar per orang bila travel tidak mencantumkan harganya.
const (
	defaultAddonTriple = 2500000
	defaultAddonDouble = 3500000
)

var hotelKeyStrip = regexp.MustCompile(`(?i)\b(hotel|makkah|mekkah|madinah|medina|by|the|al|el|ar|as)\b|[^a-z0-9]+`)

// hotelKey menormalkan nama hotel untuk pencocokan antar travel
// ("Jawharat Al Rasheed (Madinah)" == "jawharat rasheed").
func hotelKey(name string) string {
	n := strings.ToLower(cleanHotelName(name))
	if i := strings.IndexAny(n, "/("); i > 0 {
		n = n[:i]
	}
	n = hotelKeyStrip.ReplaceAllString(n, " ")
	return strings.Join(strings.Fields(n), " ")
}

// buildHotelStars: hotel -> bintang yang paling sering disebut travel (1-5).
func buildHotelStars(results []crawlResultLite, overrides []hotelOverride) map[string]int {
	votes := map[string]map[int]int{}
	vote := func(hotel string, s int) {
		k := hotelKey(hotel)
		if k == "" || s < 1 || s > 5 {
			return
		}
		if votes[k] == nil {
			votes[k] = map[int]int{}
		}
		votes[k][s]++
	}
	for _, r := range results {
		for _, cp := range r.Packages {
			vote(cp.HotelMakkah, cp.StarsMakkah)
			vote(cp.HotelMadinah, cp.StarsMadinah)
			for _, d := range cp.Departures {
				vote(d.HotelMakkah, cp.StarsMakkah)
				vote(d.HotelMadinah, cp.StarsMadinah)
			}
		}
	}
	for _, o := range overrides {
		vote(o.HotelMakkah, o.StarsMakkah)
		vote(o.HotelMadinah, o.StarsMadinah)
	}
	out := map[string]int{}
	for k, m := range votes {
		best, bestN := 0, 0
		for s, n := range m {
			if n > bestN || (n == bestN && s < best) {
				best, bestN = s, n
			}
		}
		out[k] = best
	}
	return out
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func addDays(date string, n int) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return ""
	}
	return t.AddDate(0, 0, n).Format("2006-01-02")
}

// isNonUmrah: paket wisata non-umrah yang ikut dijual travel (mis. Ventour "VCATION CHINA ...").
// Disaring juga di sini supaya data lama yang masih tersimpan tidak ikut tampil.
func isNonUmrah(name string) bool {
	u := strings.ToUpper(name)
	return strings.Contains(u, "VCATION") || strings.Contains(u, "VACATION")
}
