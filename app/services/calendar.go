package services

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"umrah/app/models"
	"umrah/app/repositories"
)

// Mode Kalender: keberangkatan per tanggal untuk 6 bulan ke depan, dengan suhu rata-rata
// iklim harian Makkah/Madinah, tema cuaca, dan momen (libur sekolah, Ramadhan, musim haji).
// Aturan data sama dengan mode Budget: tanggal lewat dan tanggal penuh tidak dihitung.

const (
	CalendarMonths   = 6
	calendarNoBudget = 50000000 // slider di ujung kanan = semua harga
)

var wib = time.FixedZone("WIB", 7*3600)

// Momen tambahan khusus kalender (melengkapi `events` di insight.go).
var calendarExtraEvents = []event{
	{"2026-03-20", "2026-03-21", "Idul Fitri 1447 H (perkiraan)", "ied"},
	{"2027-03-09", "2027-03-10", "Idul Fitri 1448 H (perkiraan)", "ied"},
	{"2028-02-26", "2028-02-27", "Idul Fitri 1449 H (perkiraan)", "ied"},
}

type CalEvent struct{ Label, Kind string }

type CalTemp struct {
	MakHi, MakLo, MadHi, MadLo int
	Tone, ToneLabel            string // sejuk | nyaman | hangat | panas
}

type CalDay struct {
	Date     string // 2006-01-02
	Day      int
	Travels  int
	Packages int
	MinPrice int
	Temp     CalTemp
	Events   []CalEvent // tanpa haji
	Haji     bool
	Past     bool
	Today    bool
	Featured bool
}

type CalMonthTab struct {
	Key, Label string // "2026-11", "Nov 26"
	Count      int
	Active     bool
}

type CalendarView struct {
	Key, Title  string // "2026-11", "November 2026"
	PrevKey     string // kosong = tidak bisa mundur
	NextKey     string
	Lead        []int // sel kosong sebelum tanggal 1 (Senin = kolom pertama)
	Days        []CalDay
	Tabs        []CalMonthTab
	Departures  int
	TravelCount int
	MinPrice    int
	Budget      int      // 0 = semua harga
	Tones       []string // tingkat cuaca yang muncul di bulan ini (urutan legenda)
}

type CalPackage struct {
	Key, Travel, Name, URL           string
	Price, Duration                  int
	ReturnDate, Airline, FlightType  string
	HotelMakkah, HotelMadinah        string
	StarsMakkah, StarsMadinah, Seats int
	Featured                         bool
}

type CalTravelGroup struct {
	Travel   string
	Packages []CalPackage
}

type CalendarDayView struct {
	Date, Weekday, Long string
	Temp                CalTemp
	Season, Crowd, Rain string
	Events              []CalEvent
	Haji                bool
	Groups              []CalTravelGroup
	Travels, Packages   int
}

type calRow struct {
	pkg models.Package
	d   models.DetailPackage
}

// calendarRows: keberangkatan mendatang yang tidak penuh, sesuai filter budget & travel.
func calendarRows(budget int, travels []string, from, to string) []calRow {
	var pkgs []models.Package
	pkgs = repositories.AllPackages()
	today := time.Now().In(wib).Format("2006-01-02")
	tf := map[string]bool{}
	for _, t := range travels {
		if t = strings.TrimSpace(t); t != "" {
			tf[t] = true
		}
	}
	var out []calRow
	for _, p := range pkgs {
		if len(tf) > 0 && !tf[p.Travel.Name] {
			continue
		}
		// Partner promo selalu tampil di Kalender (aturan budget 28-29 jt hanya untuk mode Budget).
		_, featured := featuredTravels[p.Travel.Name]
		for _, d := range p.Details {
			if d.IsFull || d.DepartureDate < today || d.DepartureDate < from || d.DepartureDate > to {
				continue
			}
			price := d.Price
			if price == 0 {
				price = p.Price
			}
			if budget > 0 && price > budget && !featured {
				continue
			}
			out = append(out, calRow{p, d})
		}
	}
	return out
}

func rowPrice(r calRow) int {
	if r.d.Price > 0 {
		return r.d.Price
	}
	return r.pkg.Price
}

// dailyTemp: interpolasi garis lurus antar pertengahan bulan (tanggal 15) dari tabel iklim bulanan.
func dailyTemp(t time.Time) CalTemp {
	y, m, d := t.Date()
	var a time.Time
	if d >= 15 {
		a = time.Date(y, m, 15, 0, 0, 0, 0, time.UTC)
	} else {
		a = time.Date(y, m-1, 15, 0, 0, 0, 0, time.UTC)
	}
	b := a.AddDate(0, 1, 0)
	f := t.Sub(a).Hours() / b.Sub(a).Hours()
	ca, cb := climates[int(a.Month())], climates[int(b.Month())]
	lerp := func(x, y int) int { return int(float64(x) + float64(y-x)*f + 0.5) }
	ct := CalTemp{MakHi: lerp(ca.MakHi, cb.MakHi), MakLo: lerp(ca.MakLo, cb.MakLo), MadHi: lerp(ca.MadHi, cb.MadHi), MadLo: lerp(ca.MadLo, cb.MadLo)}
	switch {
	case ct.MakHi < 33:
		ct.Tone, ct.ToneLabel = "sejuk", "Sejuk"
	case ct.MakHi < 38:
		ct.Tone, ct.ToneLabel = "nyaman", "Nyaman"
	case ct.MakHi < 42:
		ct.Tone, ct.ToneLabel = "hangat", "Hangat"
	default: // "sangat panas" digabung ke panas supaya legenda ringkas
		ct.Tone, ct.ToneLabel = "panas", "Panas"
	}
	return ct
}

func calEvents(date string) (evs []CalEvent, haji bool) {
	all := append(append([]event{}, events...), calendarExtraEvents...)
	for _, e := range all {
		if date < e.From || date > e.To {
			continue
		}
		if e.Kind == "haji" {
			haji = true
			continue
		}
		evs = append(evs, CalEvent{Label: e.Label, Kind: e.Kind})
	}
	return
}

func calMonthRange(now time.Time) (first, last time.Time) {
	first = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return first, first.AddDate(0, CalendarMonths-1, 0)
}

func shortMonth(t time.Time) string {
	return fmt.Sprintf("%s %02d", monthNames[t.Month()][:3], t.Year()%100)
}

// Hasil per bulan disimpan di memori. Data paket hanya berubah saat aplikasi restart (import
// ulang), dan simpanan dikosongkan setiap ganti hari karena "lewat"/"hari ini" ikut berubah.
var (
	calCacheMu  sync.Mutex
	calCacheDay string
	calCache    = map[string]CalendarView{}
)

// CalendarMonth membangun grid satu bulan ("2026-11"); bulan di luar 6 bulan ke depan dijepit.
func CalendarMonth(key string, budget int, travels []string) CalendarView {
	today := time.Now().In(wib).Format("2006-01-02")
	ck := key + "|" + fmt.Sprint(budget) + "|" + strings.Join(travels, ",")
	calCacheMu.Lock()
	if calCacheDay != today {
		calCache, calCacheDay = map[string]CalendarView{}, today
	}
	v, ok := calCache[ck]
	calCacheMu.Unlock()
	if ok {
		return v
	}
	v = buildCalendarMonth(key, budget, travels)
	calCacheMu.Lock()
	if len(calCache) < 500 {
		calCache[ck] = v
	}
	calCacheMu.Unlock()
	return v
}

func buildCalendarMonth(key string, budget int, travels []string) CalendarView {
	if budget >= calendarNoBudget {
		budget = 0
	}
	now := time.Now().In(wib)
	first, last := calMonthRange(now)
	m, err := time.Parse("2006-01", key)
	if err != nil || m.Before(first) {
		m = first
	}
	if m.After(last) {
		m = last
	}
	end := last.AddDate(0, 1, -1)
	rows := calendarRows(budget, travels, first.Format("2006-01-02"), end.Format("2006-01-02"))

	v := CalendarView{Key: m.Format("2006-01"), Title: fmt.Sprintf("%s %d", monthNames[m.Month()], m.Year()), Budget: budget}
	if m.After(first) {
		v.PrevKey = m.AddDate(0, -1, 0).Format("2006-01")
	}
	if m.Before(last) {
		v.NextKey = m.AddDate(0, 1, 0).Format("2006-01")
	}
	perMonth := map[string]int{}
	type agg struct {
		travels map[string]bool
		pkgs    map[uint]bool
		min     int
		feat    bool
	}
	byDate := map[string]*agg{}
	monthTravels := map[string]bool{}
	for _, r := range rows {
		perMonth[r.d.DepartureDate[:7]]++
		if r.d.DepartureDate[:7] != v.Key {
			continue
		}
		a := byDate[r.d.DepartureDate]
		if a == nil {
			a = &agg{travels: map[string]bool{}, pkgs: map[uint]bool{}}
			byDate[r.d.DepartureDate] = a
		}
		a.travels[r.pkg.Travel.Name] = true
		a.pkgs[r.pkg.ID] = true
		if p := rowPrice(r); a.min == 0 || p < a.min {
			a.min = p
		}
		_, f := featuredTravels[r.pkg.Travel.Name]
		a.feat = a.feat || f
		monthTravels[r.pkg.Travel.Name] = true
		v.Departures++
		if p := rowPrice(r); v.MinPrice == 0 || p < v.MinPrice {
			v.MinPrice = p
		}
	}
	v.TravelCount = len(monthTravels)
	for i := 0; i < CalendarMonths; i++ {
		t := first.AddDate(0, i, 0)
		k := t.Format("2006-01")
		v.Tabs = append(v.Tabs, CalMonthTab{Key: k, Label: shortMonth(t), Count: perMonth[k], Active: k == v.Key})
	}
	v.Lead = make([]int, (int(m.Weekday())+6)%7)
	today := now.Format("2006-01-02")
	for d := m; d.Month() == m.Month(); d = d.AddDate(0, 0, 1) {
		date := d.Format("2006-01-02")
		evs, haji := calEvents(date)
		// Garis di kotak tanggal hanya libur sekolah & Ramadhan (legenda ringkas). Momen lain
		// (10 malam terakhir, Idul Fitri) tetap disebut di pop-up tanggal.
		var bars []CalEvent
		for _, e := range evs {
			if e.Kind == "libur" || e.Kind == "ramadhan" {
				bars = append(bars, e)
			}
		}
		day := CalDay{Date: date, Day: d.Day(), Temp: dailyTemp(d), Events: bars, Haji: haji, Past: date < today, Today: date == today}
		if a := byDate[date]; a != nil {
			day.Travels, day.Packages, day.MinPrice, day.Featured = len(a.travels), len(a.pkgs), a.min, a.feat
		}
		v.Days = append(v.Days, day)
	}
	// Legenda hanya memuat tingkat cuaca yang benar-benar tampil di bulan ini.
	for _, t := range []string{"nyaman", "sejuk", "hangat", "panas"} {
		for _, d := range v.Days {
			if !d.Haji && d.Temp.Tone == t {
				v.Tones = append(v.Tones, t)
				break
			}
		}
	}
	return v
}

var weekdayNames = [7]string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}

// CalendarDay: isi pop-up satu tanggal. ok=false bila tanggal tidak valid.
func CalendarDay(date string, budget int, travels []string) (CalendarDayView, bool) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return CalendarDayView{}, false
	}
	if budget >= calendarNoBudget {
		budget = 0
	}
	evs, haji := calEvents(date)
	mi := months[t.Month()]
	v := CalendarDayView{
		Date: date, Weekday: weekdayNames[t.Weekday()], Long: fmt.Sprintf("%d %s %d", t.Day(), monthNames[t.Month()], t.Year()),
		Temp: dailyTemp(t), Season: mi.Season, Crowd: mi.CrowdNote, Rain: mi.Rain, Events: evs, Haji: haji,
	}
	groups := map[string][]CalPackage{}
	for _, r := range calendarRows(budget, travels, date, date) {
		_, f := featuredTravels[r.pkg.Travel.Name]
		groups[r.pkg.Travel.Name] = append(groups[r.pkg.Travel.Name], CalPackage{
			Key: PackageKey(r.pkg.Travel.Name, r.pkg.Name), Travel: r.pkg.Travel.Name, Name: r.pkg.Name, URL: r.pkg.URL,
			Price: rowPrice(r), Duration: r.pkg.Duration, ReturnDate: r.d.ReturnDate, Airline: r.pkg.Airline, FlightType: r.pkg.FlightType,
			HotelMakkah: r.d.HotelMakkah, HotelMadinah: r.d.HotelMadinah, StarsMakkah: r.d.StarsMakkah, StarsMadinah: r.d.StarsMadinah,
			Seats: r.d.AvailableQuota, Featured: f,
		})
	}
	for name, ps := range groups {
		sort.Slice(ps, func(i, j int) bool { return ps[i].Price < ps[j].Price })
		v.Groups = append(v.Groups, CalTravelGroup{Travel: name, Packages: ps})
		v.Packages += len(ps)
	}
	// Partner promo paling atas, lalu travel dengan harga termurah.
	sort.Slice(v.Groups, func(i, j int) bool {
		a, b := v.Groups[i].Packages[0], v.Groups[j].Packages[0]
		if a.Featured != b.Featured {
			return a.Featured
		}
		return a.Price < b.Price
	})
	v.Travels = len(v.Groups)
	return v, true
}

// ResetCalendarCache dipanggil setelah data paket dimuat ulang.
func ResetCalendarCache() {
	calCacheMu.Lock()
	calCache, calCacheDay = map[string]CalendarView{}, ""
	calCacheMu.Unlock()
}
