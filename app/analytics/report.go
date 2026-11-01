package analytics

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Hari dihitung dalam WIB.
const wibOffset = 7 * 3600

type Range struct {
	From, To time.Time // [From, To)
	Days     int
	Label    string
}

// RangeFor: hari ini (days=1) atau N hari terakhir termasuk hari ini, WIB.
func RangeFor(days int) Range {
	now := time.Now().In(WIB)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, WIB)
	label := fmt.Sprintf("%d hari terakhir", days)
	if days <= 1 {
		days, label = 1, "Hari ini"
	}
	return Range{From: today.AddDate(0, 0, -(days - 1)), To: today.AddDate(0, 0, 1), Days: days, Label: label}
}

func (r Range) prev() Range {
	return Range{From: r.From.AddDate(0, 0, -r.Days), To: r.From, Days: r.Days}
}

type Stat struct {
	Value int
	Prev  int
	Delta string // "+12%", "-5%", "" bila tidak ada pembanding
	Up    bool
}

func stat(cur, prev int) Stat {
	s := Stat{Value: cur, Prev: prev}
	if prev > 0 {
		d := float64(cur-prev) / float64(prev) * 100
		s.Delta = fmt.Sprintf("%+.0f%%", d)
		s.Up = d >= 0
	}
	return s
}

type Bar struct {
	Label string
	Value int
	Pct   int // lebar/tinggi relatif 0-100
	Share string
}

type DayRow struct {
	Label    string
	Visitors int
	Searches int
	Clicks   int
	Pct      int
}

type TravelRow struct {
	Travel                    string
	Impr, WA, Web, Calc, CTRp int
	CTR                       string
}

type PackageRow struct {
	Travel, Package string
	Price, WA       int
}

type LiveRow struct {
	Time, Type, Source, Device, Detail, Place string
	Visit                                     int // kunjungan ke-N (hari berbeda); 1 = baru
}

type Report struct {
	Range                              Range
	Visitors, Views, Searches, WA, Web Stat
	Days                               []DayRow
	Sources, Devices                   []Bar
	Travels                            []TravelRow
	TopPackages                        []PackageRow
	Positions                          []Bar
	Budgets, FilterTravels, Months     []Bar
	ZeroResults                        int
	Live                               []LiveRow
	LiveVisitors                       int
	Location                           Location
	Frequency                          Frequency
}

func count(r Range, where string, args ...any) int {
	var n int64
	q := DB.Model(&Event{}).Where("ts >= ? AND ts < ?", r.From.Unix(), r.To.Unix())
	if where != "" {
		q = q.Where(where, args...)
	}
	q.Count(&n)
	return int(n)
}

func uniqueVisitors(r Range) int {
	var n int64
	DB.Model(&Event{}).Where("ts >= ? AND ts < ? AND type = 'view'", r.From.Unix(), r.To.Unix()).
		Distinct("visitor").Count(&n)
	return int(n)
}

type kv struct {
	K string
	N int
}

func bars(rows []kv, total int) []Bar {
	max := 0
	for _, r := range rows {
		if r.N > max {
			max = r.N
		}
	}
	out := make([]Bar, 0, len(rows))
	for _, r := range rows {
		b := Bar{Label: r.K, Value: r.N}
		if max > 0 {
			b.Pct = r.N * 100 / max
		}
		if total > 0 {
			b.Share = fmt.Sprintf("%.0f%%", float64(r.N)*100/float64(total))
		}
		out = append(out, b)
	}
	return out
}

func sumKV(rows []kv) int {
	t := 0
	for _, r := range rows {
		t += r.N
	}
	return t
}

// Build menyusun seluruh isi dashboard untuk satu rentang waktu.
func Build(r Range) Report {
	rep := Report{Range: r}
	if DB == nil {
		return rep
	}
	p := r.prev()
	rep.Visitors = stat(uniqueVisitors(r), uniqueVisitors(p))
	rep.Views = stat(count(r, "type = 'view'"), count(p, "type = 'view'"))
	rep.Searches = stat(count(r, "type = 'search'"), count(p, "type = 'search'"))
	rep.WA = stat(count(r, "type = 'click_wa'"), count(p, "type = 'click_wa'"))
	rep.Web = stat(count(r, "type = 'click_web'"), count(p, "type = 'click_web'"))

	from, to := r.From.Unix(), r.To.Unix()

	// Harian (per jam bila "hari ini").
	bucket := "date(ts + 25200, 'unixepoch')"
	if r.Days == 1 {
		bucket = "strftime('%H', ts + 25200, 'unixepoch')"
	}
	type dayAgg struct {
		B                  string
		Visitors, Searches int
		Clicks             int
	}
	var agg []dayAgg
	DB.Raw(`SELECT `+bucket+` AS b,
	        COUNT(DISTINCT CASE WHEN type='view' THEN visitor END) AS visitors,
	        SUM(type='search') AS searches,
	        SUM(type IN ('click_wa','click_web')) AS clicks
	        FROM events WHERE ts >= ? AND ts < ? GROUP BY b`, from, to).Scan(&agg)
	byB := map[string]dayAgg{}
	for _, a := range agg {
		byB[a.B] = a
	}
	maxV := 1
	if r.Days == 1 {
		for h := 0; h < 24; h++ {
			k := fmt.Sprintf("%02d", h)
			a := byB[k]
			rep.Days = append(rep.Days, DayRow{Label: k + ":00", Visitors: a.Visitors, Searches: a.Searches, Clicks: a.Clicks})
		}
	} else {
		for d := r.From; d.Before(r.To); d = d.AddDate(0, 0, 1) {
			a := byB[d.Format("2006-01-02")]
			rep.Days = append(rep.Days, DayRow{Label: d.Format("02/01"), Visitors: a.Visitors, Searches: a.Searches, Clicks: a.Clicks})
		}
	}
	for _, d := range rep.Days {
		if d.Visitors > maxV {
			maxV = d.Visitors
		}
	}
	for i := range rep.Days {
		rep.Days[i].Pct = rep.Days[i].Visitors * 100 / maxV
	}

	// Sumber & perangkat (pengunjung unik).
	var src, dev []kv
	DB.Raw(`SELECT source AS k, COUNT(DISTINCT visitor) AS n FROM events
	        WHERE ts >= ? AND ts < ? AND type='view' AND source <> 'Internal' GROUP BY source ORDER BY n DESC`, from, to).Scan(&src)
	DB.Raw(`SELECT device AS k, COUNT(DISTINCT visitor) AS n FROM events
	        WHERE ts >= ? AND ts < ? AND type='view' GROUP BY device ORDER BY n DESC`, from, to).Scan(&dev)
	rep.Sources = bars(src, sumKV(src))
	rep.Devices = bars(dev, sumKV(dev))

	// Per travel.
	DB.Raw(`SELECT travel, SUM(type='impr') AS impr, SUM(type='click_wa') AS wa,
	        SUM(type='click_web') AS web, SUM(type='calc') AS calc
	        FROM events WHERE ts >= ? AND ts < ? AND travel <> '' GROUP BY travel
	        ORDER BY wa DESC, web DESC, impr DESC`, from, to).Scan(&rep.Travels)
	for i := range rep.Travels {
		t := &rep.Travels[i]
		if t.Impr > 0 {
			ctr := float64(t.WA+t.Web) * 100 / float64(t.Impr)
			t.CTR = fmt.Sprintf("%.1f%%", ctr)
			t.CTRp = int(ctr)
		} else {
			t.CTR = "-"
		}
	}
	DB.Raw(`SELECT travel, package, MAX(price) AS price, COUNT(*) AS wa FROM events
	        WHERE ts >= ? AND ts < ? AND type='click_wa' GROUP BY travel, package
	        ORDER BY wa DESC LIMIT 10`, from, to).Scan(&rep.TopPackages)

	// Posisi kartu yang diklik WA (1-5 = halaman pertama).
	var pos []kv
	DB.Raw(`SELECT CASE WHEN position BETWEEN 1 AND 5 THEN 'Posisi ' || position ELSE 'Halaman 2+' END AS k,
	        COUNT(*) AS n FROM events WHERE ts >= ? AND ts < ? AND type IN ('click_wa','click_web')
	        GROUP BY k ORDER BY k`, from, to).Scan(&pos)
	rep.Positions = bars(pos, sumKV(pos))

	// Pencarian: budget, filter travel, bulan, hasil kosong.
	type srow struct {
		Budget int
		Meta   string
	}
	var searches []srow
	DB.Raw(`SELECT budget, meta FROM events WHERE ts >= ? AND ts < ? AND type='search'`, from, to).Scan(&searches)
	budget := map[string]int{}
	travels := map[string]int{}
	months := map[string]int{}
	monthNames := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	for _, s := range searches {
		budget[budgetBucket(s.Budget)]++
		var m SearchMeta
		json.Unmarshal([]byte(s.Meta), &m)
		if m.Total == 0 {
			rep.ZeroResults++
		}
		for _, t := range strings.Split(m.Travels, ",") {
			if t = strings.TrimSpace(t); t != "" {
				travels[t]++
			}
		}
		var mi int
		if _, err := fmt.Sscanf(m.Month, "%d", &mi); err == nil && mi >= 1 && mi <= 12 {
			months[monthNames[mi]]++
		} else {
			months["Semua bulan"]++
		}
	}
	rep.Budgets = bars(sortedBuckets(budget), len(searches))
	rep.FilterTravels = bars(topKV(travels, 10), len(searches))
	rep.Months = bars(topKV(months, 13), len(searches))

	// Live: 30 menit terakhir.
	since := time.Now().Add(-30 * time.Minute).Unix()
	var live []Event
	DB.Where("ts >= ? AND type <> 'impr'", since).Order("ts DESC").Limit(50).Find(&live)
	var lv int64
	DB.Model(&Event{}).Where("ts >= ?", since).Distinct("visitor").Count(&lv)
	rep.LiveVisitors = int(lv)
	for _, e := range live {
		detail := e.Path
		switch e.Type {
		case "search":
			detail = "Budget Rp " + rupiah(e.Budget)
		case "click_wa", "click_web", "calc":
			detail = e.Travel
			if e.Package != "" {
				detail += " — " + e.Package
			}
		}
		loc := e
		if e.Type != "view" {
			if v := lastView(e.VID, e.TS); v.ID != 0 {
				loc = v
				e.Source = v.Source
			}
		}
		place := loc.City
		if loc.Region != "" && loc.Region != loc.City {
			place = joinNonEmpty(place, loc.Region)
		}
		if loc.Country != "" && loc.Country != "Indonesia" {
			place = joinNonEmpty(place, loc.Country)
		}
		rep.Live = append(rep.Live, LiveRow{
			Time: time.Unix(e.TS, 0).In(WIB).Format("15:04"), Type: typeLabel(e.Type),
			Source: e.Source, Device: e.Device, Detail: detail, Place: place, Visit: visitNumber(e.VID, e.TS),
		})
	}
	rep.Location = buildLocation(r)
	rep.Frequency = buildFrequency(r)
	return rep
}

func budgetBucket(b int) string {
	jt := b / 1000000
	switch {
	case jt < 25:
		return "< 25 jt"
	case jt < 30:
		return "25–29 jt"
	case jt < 35:
		return "30–34 jt"
	case jt < 40:
		return "35–39 jt"
	default:
		return "≥ 40 jt"
	}
}

func sortedBuckets(m map[string]int) []kv {
	order := []string{"< 25 jt", "25–29 jt", "30–34 jt", "35–39 jt", "≥ 40 jt"}
	out := []kv{}
	for _, k := range order {
		if m[k] > 0 {
			out = append(out, kv{k, m[k]})
		}
	}
	return out
}

func topKV(m map[string]int, n int) []kv {
	out := make([]kv, 0, len(m))
	for k, v := range m {
		out = append(out, kv{k, v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].N > out[j].N })
	if len(out) > n {
		out = out[:n]
	}
	return out
}

func typeLabel(t string) string {
	switch t {
	case "view":
		return "Buka halaman"
	case "search":
		return "Cari paket"
	case "click_wa":
		return "Klik WA"
	case "click_web":
		return "Klik website"
	case "calc":
		return "Kalkulator"
	case "chat":
		return "Chat"
	case "cal_open":
		return "Buka kalender"
	case "cal_day":
		return "Klik tanggal kalender"
	}
	return t
}

func rupiah(n int) string {
	s := fmt.Sprintf("%d", n)
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	return strings.Join(append([]string{s}, parts...), ".")
}

func joinNonEmpty(a, b string) string {
	if a == "" {
		return b
	}
	return a + ", " + b
}
