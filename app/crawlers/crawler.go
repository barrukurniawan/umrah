package crawlers

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

// CrawledPackage adalah satu paket umroh hasil crawl.
//
// ATURAN DATA: isi field hanya dengan apa yang benar-benar tertulis/diberikan
// travel (halaman paket, flyer, JSON situs). Jangan menebak. Nilai nol ("" / 0 /
// nil / false) berarti "tidak diketahui" dan TIDAK akan ditampilkan ke pengguna.
type CrawledPackage struct {
	TravelName     string   `json:"travel_name"`
	PackageName    string   `json:"package_name"`
	Price          int      `json:"price"`    // harga termurah (biasanya Quad)
	Duration       int      `json:"duration"` // jumlah hari sesuai yang dipasarkan travel (mis. "9 Hari")
	Airline        string   `json:"airline"`
	HotelMakkah    string   `json:"hotel_makkah"`
	HotelMadinah   string   `json:"hotel_madinah"`
	DepartureDates []string `json:"departure_dates"` // lama; dipakai jika Departures kosong
	Seats          int      `json:"seats"`           // sisa seat (bukan total)
	Airport        string   `json:"airport"`         // kota/bandara keberangkatan
	URL            string   `json:"url"`             // halaman detail paket (unik per paket)

	// --- Field tambahan (opsional) ---
	Departures      []CrawledDeparture `json:"departures,omitempty"` // satu entri per tanggal berangkat
	StarsMakkah     int                `json:"stars_makkah,omitempty"`
	StarsMadinah    int                `json:"stars_madinah,omitempty"`
	DistanceMakkah  int                `json:"distance_makkah,omitempty"`  // meter, HANYA jika ditulis travel
	DistanceMadinah int                `json:"distance_madinah,omitempty"` // meter, HANYA jika ditulis travel
	Direct          *bool              `json:"direct,omitempty"`           // true=direct, false=transit, nil=tidak disebut
	TransitVia      string             `json:"transit_via,omitempty"`      // kota transit, mis. "Muscat"
	Facilities      []string           `json:"facilities,omitempty"`       // termasuk (include), apa adanya dari travel
	Excludes        []string           `json:"excludes,omitempty"`         // tidak termasuk
	PriceTriple     int                `json:"price_triple,omitempty"`
	PriceDouble     int                `json:"price_double,omitempty"`
	SeatsTotal      int                `json:"seats_total,omitempty"`
	Full            bool               `json:"full,omitempty"` // travel menyatakan penuh/sold out
	DownPayment     int                `json:"down_payment,omitempty"`
	Guide           string             `json:"guide,omitempty"` // nama pembimbing/ustadz jika disebut
}

// CrawledDeparture adalah satu tanggal keberangkatan. Field nol = pakai nilai di
// level paket (untuk harga/hotel/maskapai) atau "tidak diketahui".
type CrawledDeparture struct {
	Date         string `json:"date"`                  // YYYY-MM-DD
	ReturnDate   string `json:"return_date,omitempty"` // YYYY-MM-DD, hanya jika tertulis atau dihitung dari durasi resmi
	Price        int    `json:"price,omitempty"`       // jika harga beda per tanggal
	PriceTriple  int    `json:"price_triple,omitempty"`
	PriceDouble  int    `json:"price_double,omitempty"`
	Seats        int    `json:"seats,omitempty"` // sisa seat
	SeatsTotal   int    `json:"seats_total,omitempty"`
	Full         bool   `json:"full,omitempty"`
	HotelMakkah  string `json:"hotel_makkah,omitempty"`
	HotelMadinah string `json:"hotel_madinah,omitempty"`
	Airline      string `json:"airline,omitempty"`
	Airport      string `json:"airport,omitempty"`
}

type SiteConfig struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Parser string `json:"parser"`
}

type Parser interface {
	Crawl() ([]CrawledPackage, error)
}

func NewCollector(domain string) *colly.Collector {
	c := colly.NewCollector(
		colly.AllowedDomains(domain),
	)

	c.SetRequestTimeout(30 * time.Second)

	c.UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"

	c.OnRequest(func(r *colly.Request) {
		log.Println("[crawler] visiting:", r.URL.String())
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("[crawler] error on %s: %v\n", r.Request.URL, err)
	})

	return c
}

func ParsePrice(priceStr string) int {
	cleaned := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, priceStr)

	var price int
	fmt.Sscanf(cleaned, "%d", &price)

	if price > 200000000 {
		price /= 100
	}
	return price
}

func ParseDuration(s string) int {
	lower := strings.ToLower(s)

	idxHR := strings.Index(lower, "hr")
	idxHari := strings.Index(lower, "hari")

	idx := idxHari
	if idxHari < 0 || (idxHR >= 0 && idxHR < idxHari) {
		idx = idxHR
	}
	if idx < 0 {
		return 0
	}

	start := idx - 1
	for start >= 0 && !(s[start] >= '0' && s[start] <= '9') {
		start--
	}
	for start >= 0 && s[start] >= '0' && s[start] <= '9' {
		start--
	}
	start++

	numStr := s[start:idx]
	numStr = strings.TrimSpace(numStr)
	numStr = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, numStr)

	var dur int
	fmt.Sscanf(numStr, "%d", &dur)
	return dur
}

func ParseSeats(seatStr string) int {
	lower := strings.ToLower(seatStr)

	terisiIdx := strings.Index(lower, "terisi")
	sisaIdx := strings.Index(lower, "sisa seat")

	var sub string
	if terisiIdx >= 0 {
		sub = seatStr[terisiIdx+6:]
	} else if sisaIdx >= 0 {
		sub = seatStr[sisaIdx+9:]
	} else {
		return 0
	}

	for len(sub) > 0 && (sub[0] < '0' || sub[0] > '9') {
		sub = sub[1:]
	}
	var numStr string
	for _, c := range sub {
		if c >= '0' && c <= '9' {
			numStr += string(c)
		} else {
			break
		}
	}

	var seats int
	fmt.Sscanf(numStr, "%d", &seats)
	return seats
}

func CleanText(s string) string {
	return strings.TrimSpace(s)
}

func KnownAirline(s string) string {
	lower := strings.ToLower(s)
	airlinesOrdered := []struct {
		key  string
		name string
	}{
		{"garuda", "Garuda Indonesia"},
		{"ga /", "Garuda Indonesia"},
		{"saudia", "Saudia"},
		{"saudi arabian", "Saudia"},
		{"sv /", "Saudia"},
		{"qatar", "Qatar Airways"},
		{"emirates", "Emirates"},
		{"etihad", "Etihad Airways"},
		{"et /", "Etihad Airways"},
		{"ey /", "Etihad Airways"},
		{"turkish", "Turkish Airlines"},
		{"tk /", "Turkish Airlines"},
		{"oman", "Oman Air"},
		{"wy /", "Oman Air"},
		{"royal brunei", "Royal Brunei Airlines"},
		{"scoot", "Scoot"},
		{"indigo", "IndiGo"},
		{"lion", "Lion Air"},
		{"batik", "Batik Air"},
		{"citilink", "Citilink"},
	}
	for _, a := range airlinesOrdered {
		if strings.Contains(lower, a.key) {
			return a.name
		}
	}
	return ""
}

func countMonths(s string) int {
	months := []string{
		"januari", "februari", "maret", "april", "mei", "juni",
		"juli", "agustus", "september", "oktober", "november", "desember",
		"jan", "feb", "mar", "apr", "may", "jun",
		"jul", "aug", "agu", "sep", "okt", "oct", "nov", "des", "dec",
	}
	count := 0
	lower := strings.ToLower(s)
	for _, m := range months {
		count += strings.Count(lower, m)
	}
	return count
}

func isDateLine(s string) bool {
	lower := strings.ToLower(s)
	if countMonths(s) == 0 {
		return false
	}
	return strings.Contains(lower, "2026") || strings.Contains(lower, "2027") || strings.Contains(lower, "2028")
}
