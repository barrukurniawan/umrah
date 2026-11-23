package demo

import (
	"strings"
	"sync"
	"time"

	"umrah/app/analytics"
	"umrah/app/services"
)

// ---- #4 Posisi unggulan berbayar (Placement) ----

var (
	plMu       sync.RWMutex
	placements = map[string]Placement{}
)

func refreshPlacements() {
	if DB == nil {
		return
	}
	today := Today()
	var ps []Placement
	DB.Where("active = ? AND starts_at <= ? AND ends_at >= ?", true, today, today).Find(&ps)
	m := map[string]Placement{}
	for _, p := range ps {
		m[p.TravelName] = p
	}
	plMu.Lock()
	placements = m
	plMu.Unlock()
}

// FeaturedRange: sumber "Promo Spesial" untuk services (menggantikan map featuredTravels).
// Paket travel berposisi unggulan tetap tampil di luar rentang budget (tidak disematkan).
func FeaturedRange(travel string) (minBudget, maxBudget int, showOutside, ok bool) {
	plMu.RLock()
	defer plMu.RUnlock()
	p, ok := placements[travel]
	return p.MinBudget, p.MaxBudget, true, ok
}

func ActivePlacement(travel string) *Placement {
	plMu.RLock()
	defer plMu.RUnlock()
	if p, ok := placements[travel]; ok {
		return &p
	}
	return nil
}

func Placements() []Placement {
	var ps []Placement
	DB.Order("active DESC, ends_at DESC").Find(&ps)
	return ps
}

func SavePlacement(p *Placement) error {
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
	}
	err := DB.Save(p).Error
	refreshPlacements()
	return err
}

func DeletePlacement(id uint) {
	DB.Delete(&Placement{}, id)
	refreshPlacements()
}

// PlacementCost: biaya posisi unggulan dalam satu periode bulan (untuk tagihan).
func PlacementCost(travelName, period string) (days int, cost int64) {
	var ps []Placement
	DB.Where("travel_name = ?", travelName).Find(&ps)
	start, _ := time.ParseInLocation("2006-01", period, WIB)
	end := start.AddDate(0, 1, -1)
	for _, p := range ps {
		a, err1 := time.ParseInLocation("2006-01-02", p.StartsAt, WIB)
		b, err2 := time.ParseInLocation("2006-01-02", p.EndsAt, WIB)
		if err1 != nil || err2 != nil {
			continue
		}
		if a.Before(start) {
			a = start
		}
		if b.After(end) {
			b = end
		}
		if today, _ := time.ParseInLocation("2006-01-02", Today(), WIB); b.After(today) {
			b = today
		}
		if !b.Before(a) {
			d := int(b.Sub(a).Hours()/24) + 1
			days += d
			cost += int64(d) * p.PricePerDay
		}
	}
	return
}

// ---- #3 Lead terverifikasi ----

func init() {
	analytics.OnClick = func(kind, travel, pkg, visitor, device string) {
		if !enabled {
			return
		}
		t := TravelByName(travel)
		if t == nil || !t.Fictional {
			return // lead hanya dicatat untuk travel mitra (fiktif) di demo
		}
		k := map[string]string{"click_wa": "wa", "click_web": "web"}[kind]
		RecordLead(Lead{TravelSlug: t.Slug, Package: pkg, Kind: k, Visitor: visitor, Device: device, Source: "umrohku"})
	}
}

// RecordLead menyimpan lead dan menentukan apakah bisa ditagih: lead dari pengunjung yang sama ke
// travel yang sama dalam 24 jam dianggap duplikat (tidak ditagih).
func RecordLead(l Lead) *Lead {
	if DB == nil {
		return nil
	}
	if l.CreatedAt.IsZero() {
		l.CreatedAt = time.Now()
	}
	if l.Status == "" {
		l.Status = "baru"
	}
	l.Billable = true
	if l.Visitor != "" {
		var n int64
		DB.Model(&Lead{}).Where("travel_slug = ? AND visitor = ? AND billable = ? AND created_at > ?", l.TravelSlug, l.Visitor, true,
			l.CreatedAt.Add(-24*time.Hour)).Count(&n)
		if n > 0 {
			l.Billable, l.Reason = false, "Duplikat (pengunjung sama < 24 jam)"
		}
	}
	DB.Create(&l)
	if l.Billable && l.Kind == "form" {
		Notify("travel:"+l.TravelSlug, "wa", "Admin travel", "Lead baru: "+l.Name,
			"Calon jamaah "+l.Name+" ("+l.City+") minta dihubungi untuk paket "+l.Package+". No. WA "+l.Phone+".", "/portal/lead")
	}
	return &l
}

func SetLeadStatus(slug string, id uint, status string) bool {
	switch status {
	case "baru", "dihubungi", "closing", "tidak_valid":
	default:
		return false
	}
	upd := map[string]any{"status": status}
	if status == "tidak_valid" {
		upd["billable"] = false
		upd["reason"] = "Ditandai tidak valid oleh travel (ditinjau admin)"
	}
	return DB.Model(&Lead{}).Where("id = ? AND travel_slug = ?", id, slug).Updates(upd).RowsAffected == 1
}

type LeadFilter struct {
	Slug   string
	Status string
	Period string // "2026-11"
	Limit  int
}

func Leads(f LeadFilter) []Lead {
	q := DB.Model(&Lead{}).Where("travel_slug = ?", f.Slug)
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Period != "" {
		a, b := periodRange(f.Period)
		q = q.Where("created_at >= ? AND created_at < ?", a, b)
	}
	if f.Limit == 0 {
		f.Limit = 200
	}
	var ls []Lead
	q.Order("created_at DESC").Limit(f.Limit).Find(&ls)
	return ls
}

func periodRange(period string) (time.Time, time.Time) {
	a, err := time.ParseInLocation("2006-01", period, WIB)
	if err != nil {
		a = time.Now().In(WIB)
		a = time.Date(a.Year(), a.Month(), 1, 0, 0, 0, 0, WIB)
	}
	return a, a.AddDate(0, 1, 0)
}

func CurrentPeriod() string { return time.Now().In(WIB).Format("2006-01") }

func PrevPeriod(p string) string {
	a, _ := periodRange(p)
	return a.AddDate(0, -1, 0).Format("2006-01")
}

func PeriodLabel(p string) string {
	a, _ := periodRange(p)
	return []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}[a.Month()] + " " + a.Format("2006")
}

// DailyCount: jumlah lead per hari (n hari terakhir) untuk grafik batang.
type DayCount struct {
	Day   string
	Label string
	N     int
	Pct   int
}

func LeadDaily(slug string, days int) []DayCount {
	since := time.Now().In(WIB).AddDate(0, 0, -days+1)
	since = time.Date(since.Year(), since.Month(), since.Day(), 0, 0, 0, 0, WIB)
	var ls []Lead
	DB.Select("created_at").Where("travel_slug = ? AND created_at >= ?", slug, since).Find(&ls)
	counts := map[string]int{}
	for _, l := range ls {
		counts[l.CreatedAt.In(WIB).Format("2006-01-02")]++
	}
	out := make([]DayCount, days)
	mx := 1
	for i := 0; i < days; i++ {
		d := since.AddDate(0, 0, i)
		k := d.Format("2006-01-02")
		out[i] = DayCount{Day: k, Label: d.Format("2"), N: counts[k]}
		mx = max(mx, counts[k])
	}
	for i := range out {
		out[i].Pct = out[i].N * 100 / mx
	}
	return out
}

// PackageStat: tampil/klik/lead/booking per paket untuk dashboard portal.
type PackageStat struct {
	Package  string
	Impr     int
	Leads    int
	Bookings int
	Conv     string
}

func PackageStats(slug string, days int) []PackageStat {
	t := TravelBySlug(slug)
	if t == nil {
		return nil
	}
	since := time.Now().AddDate(0, 0, -days)
	m := map[string]*PackageStat{}
	get := func(p string) *PackageStat {
		if m[p] == nil {
			m[p] = &PackageStat{Package: p}
		}
		return m[p]
	}
	var ls []Lead
	DB.Select("package").Where("travel_slug = ? AND created_at >= ? AND billable = ?", slug, since, true).Find(&ls)
	for _, l := range ls {
		get(l.Package).Leads++
	}
	var bs []Booking
	DB.Select("package_name").Where("travel_slug = ? AND created_at >= ? AND status <> ?", slug, since, "batal").Find(&bs)
	for _, b := range bs {
		get(b.PackageName).Bookings++
	}
	for name, n := range analytics.ImpressionsByPackage(t.Name, since) {
		get(name).Impr += n
	}
	// Tayangan seed: perkiraan 9x lead bila analitik instance demo masih sepi.
	var out []PackageStat
	for _, s := range m {
		if s.Impr < s.Leads*4 {
			s.Impr = s.Leads*9 + 13
		}
		if s.Impr > 0 {
			s.Conv = strings.Replace(formatPct(float64(s.Leads)*100/float64(s.Impr)), ".", ",", 1)
		}
		out = append(out, *s)
	}
	sortStats(out)
	return out
}

func formatPct(f float64) string {
	return strings.TrimSuffix(strings.TrimSuffix(fmtFloat(f, 1), "0"), ".") + "%"
}

var _ = services.PackageKey
