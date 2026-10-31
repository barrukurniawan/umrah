package analytics

import (
	"fmt"
	"time"
)

// Lokasi & frekuensi kunjungan. Satu "kunjungan" = satu hari (WIB) dengan minimal satu view dari ID anonim (cookie).

type Location struct {
	Regions, Cities, Countries []Bar
	Known, Unknown             int
}

type Frequency struct {
	Visitors              int
	New, Returning        int
	NewShare, RetShare    string
	AvgVisits             string
	AvgGap                string
	Distribution          []Bar
	SearchRate, WARate    string // % pengunjung yang mencari; % pencari yang klik WA
	Searchers, WAClickers int
}

// Kolom kota/provinsi hanya terisi di event view; hitung per ID anonim (fallback: visitor harian).
const who = "COALESCE(NULLIF(vid, ''), visitor)"

func buildLocation(r Range) Location {
	var loc Location
	from, to := r.From.Unix(), r.To.Unix()
	var regions, cities, countries []kv
	DB.Raw(`SELECT region AS k, COUNT(DISTINCT `+who+`) AS n FROM events
	        WHERE ts >= ? AND ts < ? AND type='view' AND country='Indonesia' AND region <> ''
	        GROUP BY region ORDER BY n DESC LIMIT 12`, from, to).Scan(&regions)
	DB.Raw(`SELECT city || CASE WHEN region <> '' AND region <> city THEN ', ' || region ELSE '' END AS k,
	        COUNT(DISTINCT `+who+`) AS n FROM events
	        WHERE ts >= ? AND ts < ? AND type='view' AND city <> '' GROUP BY k ORDER BY n DESC LIMIT 12`, from, to).Scan(&cities)
	DB.Raw(`SELECT CASE WHEN country = '' THEN 'Tidak diketahui' ELSE country END AS k,
	        COUNT(DISTINCT `+who+`) AS n FROM events
	        WHERE ts >= ? AND ts < ? AND type='view' GROUP BY k ORDER BY n DESC LIMIT 8`, from, to).Scan(&countries)
	total := sumKV(countries)
	for _, c := range countries {
		if c.K == "Tidak diketahui" {
			loc.Unknown = c.N
		}
	}
	loc.Known = total - loc.Unknown
	loc.Regions = bars(regions, sumKV(regions))
	loc.Cities = bars(cities, sumKV(cities))
	loc.Countries = bars(countries, total)
	return loc
}

func pct(a, b int) string {
	if b == 0 {
		return "-"
	}
	return fmt.Sprintf("%.0f%%", float64(a)*100/float64(b))
}

func buildFrequency(r Range) Frequency {
	var f Frequency
	from, to := r.From.Unix(), r.To.Unix()
	type row struct {
		Who       string
		Days      int
		FirstDay  string
		LastDay   string
		FirstEver int64
	}
	var rows []row
	DB.Raw(`SELECT v.who, v.days, v.first_day, v.last_day,
	          (SELECT MIN(ts) FROM events e WHERE e.type='view' AND e.vid = v.who) AS first_ever
	        FROM (SELECT `+who+` AS who, COUNT(DISTINCT date(ts + 25200, 'unixepoch')) AS days,
	                MIN(date(ts + 25200, 'unixepoch')) AS first_day, MAX(date(ts + 25200, 'unixepoch')) AS last_day
	              FROM events WHERE ts >= ? AND ts < ? AND type='view' GROUP BY who) v`, from, to).Scan(&rows)

	f.Visitors = len(rows)
	totalDays, gapSum, gapN := 0, 0.0, 0
	dist := map[string]int{}
	for _, x := range rows {
		totalDays += x.Days
		// Kembali = sudah datang di lebih dari satu hari (di periode ini atau sebelumnya).
		if x.Days >= 2 || (x.FirstEver > 0 && x.FirstEver < from) {
			f.Returning++
		} else {
			f.New++
		}
		switch {
		case x.Days <= 1:
			dist["1 kali"]++
		case x.Days <= 3:
			dist["2–3 kali"]++
		default:
			dist["4+ kali"]++
		}
		if x.Days >= 2 {
			a, errA := time.Parse("2006-01-02", x.FirstDay)
			b, errB := time.Parse("2006-01-02", x.LastDay)
			if errA == nil && errB == nil {
				gapSum += b.Sub(a).Hours() / 24 / float64(x.Days-1)
				gapN++
			}
		}
	}
	f.NewShare, f.RetShare = pct(f.New, f.Visitors), pct(f.Returning, f.Visitors)
	if f.Visitors > 0 {
		f.AvgVisits = fmt.Sprintf("%.1f", float64(totalDays)/float64(f.Visitors))
	} else {
		f.AvgVisits = "-"
	}
	if gapN > 0 {
		f.AvgGap = fmt.Sprintf("%.1f hari", gapSum/float64(gapN))
	} else {
		f.AvgGap = "-"
	}
	var d []kv
	for _, k := range []string{"1 kali", "2–3 kali", "4+ kali"} {
		if dist[k] > 0 {
			d = append(d, kv{k, dist[k]})
		}
	}
	f.Distribution = bars(d, f.Visitors)

	// Corong sederhana: pengunjung → mencari → klik WA (per ID anonim).
	var searchers, wa int64
	DB.Raw(`SELECT COUNT(DISTINCT `+who+`) FROM events WHERE ts >= ? AND ts < ? AND type='search'`, from, to).Scan(&searchers)
	DB.Raw(`SELECT COUNT(DISTINCT `+who+`) FROM events WHERE ts >= ? AND ts < ? AND type='click_wa'
	        AND `+who+` IN (SELECT `+who+` FROM events WHERE ts >= ? AND ts < ? AND type='search')`,
		from, to, from, to).Scan(&wa)
	f.Searchers, f.WAClickers = int(searchers), int(wa)
	f.SearchRate, f.WARate = pct(f.Searchers, f.Visitors), pct(f.WAClickers, f.Searchers)
	return f
}

// lastView: lokasi & sumber dari view terakhir ID anonim ini (untuk baris cari/klik di Live).
func lastView(vid string, ts int64) (e Event) {
	if vid != "" {
		DB.Where("vid = ? AND type = 'view' AND ts <= ?", vid, ts).Order("ts DESC").Limit(1).Find(&e)
	}
	return
}

// visitNumber: kunjungan ke berapa (hari berbeda) untuk ID anonim ini sampai waktu ts.
func visitNumber(vid string, ts int64) int {
	if vid == "" {
		return 0
	}
	var n int64
	DB.Raw(`SELECT COUNT(DISTINCT date(ts + 25200, 'unixepoch')) FROM events
	        WHERE vid = ? AND type='view' AND ts <= ?`, vid, ts).Scan(&n)
	return int(n)
}
