package crawlers

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

type CrawledPackage struct {
	TravelName     string   `json:"travel_name"`
	PackageName    string   `json:"package_name"`
	Price          int      `json:"price"`
	Duration       int      `json:"duration"`
	Airline        string   `json:"airline"`
	HotelMakkah    string   `json:"hotel_makkah"`
	HotelMadinah   string   `json:"hotel_madinah"`
	DepartureDates []string `json:"departure_dates"`
	Seats          int      `json:"seats"`
	Airport        string   `json:"airport"`
	URL            string   `json:"url"`
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
		{"saudia", "Saudia"},
		{"saudi arabian", "Saudia"},
		{"qatar", "Qatar Airways"},
		{"emirates", "Emirates"},
		{"etihad", "Etihad Airways"},
		{"oman", "Oman Air"},
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
