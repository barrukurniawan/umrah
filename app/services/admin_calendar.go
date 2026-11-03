package services

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"umrah/app/repositories"
)

// Kalender ketersediaan untuk admin: per tanggal, berapa keberangkatan dari travel mana,
// mana yang masih ada seat, penuh, atau seatnya tidak disebut travel. Dibaca langsung dari
// data terbaru (dimuat ulang tiap crawl), jadi selalu mutakhir.

const (
	SeatOpen    = "ada"   // travel menyebut sisa seat > 0
	SeatFull    = "penuh" // travel menyatakan penuh
	SeatUnknown = "tanpa" // travel tidak menyebut sisa seat (dianggap masih dibuka)
)

type AdminDep struct {
	Travel, Package, URL, Airline, Status string
	Price, Seats, Duration                int
	ReturnDate                            string
}

type AdminDay struct {
	Date                       string
	Day                        int
	Open, Full, Unknown        int // jumlah keberangkatan per status
	Travels                    int
	Past, Today, Selected, Haj bool
}

type AdminMonth struct {
	Key, Title    string
	Lead          []int
	Days          []AdminDay
	EmptyDates    []string // tanggal tanpa keberangkatan yang masih bisa dipesan (bukan musim haji)
	FullOnlyDates []string // ada keberangkatan tapi semuanya penuh
	OpenCount     int
}

type AdminTravelRow struct {
	Name                string
	Open, Full, Unknown int
	Next                string // keberangkatan terdekat yang masih bisa dipesan
}

type AdminCalendarView struct {
	Months      []AdminMonth
	Month       AdminMonth
	MonthKey    string
	Selected    string
	SelectedFmt string
	Deps        []AdminDep
	Travels     []string
	Travel      string
	TravelRows  []AdminTravelRow
}

var idMonths = [...]string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

// AdminCalendar: bulan (YYYY-MM), tanggal terpilih (YYYY-MM-DD), filter travel (opsional).
func AdminCalendar(monthKey, selected, travel string) AdminCalendarView {
	wib := time.FixedZone("WIB", 7*3600)
	now := time.Now().In(wib)
	today := now.Format("2006-01-02")

	type entry struct {
		dep    AdminDep
		travel string
	}
	byDate := map[string][]entry{}
	travelSet := map[string]bool{}
	for _, p := range repositories.AllPackages() {
		tn := p.Travel.Name
		travelSet[tn] = true
		for _, d := range p.Details {
			if d.DepartureDate < today {
				continue
			}
			st := SeatUnknown
			if d.IsFull {
				st = SeatFull
			} else if d.AvailableQuota > 0 {
				st = SeatOpen
			}
			price := d.Price
			if price == 0 {
				price = p.Price
			}
			byDate[d.DepartureDate] = append(byDate[d.DepartureDate], entry{travel: tn, dep: AdminDep{
				Travel: TravelDisplayName(tn), Package: StripEmoji(p.Name), URL: p.URL, Airline: p.Airline,
				Status: st, Price: price, Seats: d.AvailableQuota, Duration: p.Duration, ReturnDate: d.ReturnDate,
			}})
		}
	}
	v := AdminCalendarView{Travel: travel, Selected: selected}
	v.Travels = sortedKeys(travelSet)
	match := func(e entry) bool { return travel == "" || e.travel == travel }

	// Ringkasan per travel (semua tanggal mendatang).
	rows := map[string]*AdminTravelRow{}
	for date, es := range byDate {
		for _, e := range es {
			r := rows[e.travel]
			if r == nil {
				r = &AdminTravelRow{Name: TravelDisplayName(e.travel)}
				rows[e.travel] = r
			}
			switch e.dep.Status {
			case SeatOpen:
				r.Open++
			case SeatFull:
				r.Full++
			default:
				r.Unknown++
			}
			if e.dep.Status != SeatFull && (r.Next == "" || date < r.Next) {
				r.Next = date
			}
		}
	}
	for _, r := range rows {
		v.TravelRows = append(v.TravelRows, *r)
	}
	sort.Slice(v.TravelRows, func(i, j int) bool { return v.TravelRows[i].Name < v.TravelRows[j].Name })

	// 6 bulan ke depan.
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, wib)
	for i := 0; i < CalendarMonths; i++ {
		m := first.AddDate(0, i, 0)
		am := AdminMonth{Key: m.Format("2006-01"), Title: fmt.Sprintf("%s %d", idMonths[m.Month()], m.Year())}
		am.Lead = make([]int, (int(m.Weekday())+6)%7)
		for d := m; d.Month() == m.Month(); d = d.AddDate(0, 0, 1) {
			date := d.Format("2006-01-02")
			_, haji := calEvents(date)
			day := AdminDay{Date: date, Day: d.Day(), Past: date < today, Today: date == today, Selected: date == selected, Haj: haji}
			tset := map[string]bool{}
			for _, e := range byDate[date] {
				if !match(e) {
					continue
				}
				tset[e.travel] = true
				switch e.dep.Status {
				case SeatOpen:
					day.Open++
				case SeatFull:
					day.Full++
				default:
					day.Unknown++
				}
			}
			day.Travels = len(tset)
			if !day.Past {
				if day.Open+day.Unknown > 0 {
					am.OpenCount++
				} else if day.Full > 0 {
					am.FullOnlyDates = append(am.FullOnlyDates, date)
				} else if !haji {
					am.EmptyDates = append(am.EmptyDates, date)
				}
			}
			am.Days = append(am.Days, day)
		}
		v.Months = append(v.Months, am)
	}
	v.Month = v.Months[0]
	for _, m := range v.Months {
		if m.Key == monthKey {
			v.Month = m
		}
	}
	v.MonthKey = v.Month.Key

	if selected != "" {
		if t, err := time.Parse("2006-01-02", selected); err == nil {
			v.SelectedFmt = fmt.Sprintf("%d %s %d", t.Day(), idMonths[t.Month()], t.Year())
		}
		for _, e := range byDate[selected] {
			if match(e) {
				v.Deps = append(v.Deps, e.dep)
			}
		}
		rank := map[string]int{SeatOpen: 0, SeatUnknown: 1, SeatFull: 2}
		sort.SliceStable(v.Deps, func(i, j int) bool {
			a, b := v.Deps[i], v.Deps[j]
			if rank[a.Status] != rank[b.Status] {
				return rank[a.Status] < rank[b.Status]
			}
			if a.Travel != b.Travel {
				return a.Travel < b.Travel
			}
			return a.Price < b.Price
		})
	}
	return v
}

// DayShort: "2026-12-05" -> "5"; untuk daftar tanggal kosong per bulan.
func DayShort(date string) string {
	if len(date) == 10 {
		return strings.TrimLeft(date[8:], "0")
	}
	return date
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
