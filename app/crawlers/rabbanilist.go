package crawlers

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"
)

// RabbaniListParser crawls Rabbani Tour (rabbanitour.com, WordPress).
//
// Every departure is its own "jadwal_umroh" post with a detail page
// (https://rabbanitour.com/jadwal/<program>-<d>-<mon>-<yyyy>/). The list of
// posts comes from the site's public REST API (the same data the category
// pages render), which also carries the status taxonomy ("full-booked",
// "segera-habis", ...). Each detail page states: program name, date, "N Hari",
// Direct/Transit, airline(s), Hotel Madinah / Hotel Mekkah (sometimes with
// "*4" stars), bonus facilities, Include / Eksclude lists, Quad/Triple/Double
// prices and "Harga starting dari Jakarta".
//
// The site never shows a return date, remaining seats, a guide or a DP, so
// those stay empty. One package is emitted per detail page (URL = that page).
type RabbaniListParser struct {
	URL string // site root, e.g. https://rabbanitour.com/
}

func NewRabbaniListParser(homeURL string) *RabbaniListParser {
	return &RabbaniListParser{URL: homeURL}
}

type rabbaniJadwal struct {
	Link  string `json:"link"`
	Title struct {
		Rendered string `json:"rendered"`
	} `json:"title"`
	ClassList []string `json:"class_list"`
}

var rabbaniTitleDateRe = regexp.MustCompile(`(?i)(\d{1,2})\s+(` + lstMonthAlt + `)\.?\s+(20\d{2})\s*$`)

func (r *RabbaniListParser) Crawl() ([]CrawledPackage, error) {
	root := strings.TrimSuffix(r.URL, "/")
	if root == "" || !strings.Contains(root, "rabbanitour.com") {
		root = "https://rabbanitour.com"
	}
	if i := strings.Index(root, "rabbanitour.com"); i >= 0 {
		root = root[:i+len("rabbanitour.com")]
	}

	// 1. All schedule posts from the REST API.
	var posts []rabbaniJadwal
	for page := 1; page <= 10; page++ {
		u := fmt.Sprintf("%s/wp-json/wp/v2/jadwal_umroh?per_page=100&page=%d&_fields=link,title,class_list", root, page)
		log.Println("[crawler] visiting:", u)
		body, err := lstFetchPolite(u)
		if err != nil {
			if page == 1 {
				return nil, err
			}
			break // WordPress answers 400 past the last page
		}
		var batch []rabbaniJadwal
		if err := json.Unmarshal([]byte(body), &batch); err != nil {
			if page == 1 {
				return nil, fmt.Errorf("rabbani: jadwal_umroh API: %w", err)
			}
			break
		}
		posts = append(posts, batch...)
		if len(batch) < 100 {
			break
		}
		time.Sleep(lstPoliteDelay)
	}
	log.Printf("[crawler] rabbanitour.com: %d schedule posts\n", len(posts))

	cutoff := lstCutoff()
	var result []CrawledPackage
	seen := map[string]bool{}
	fetched := 0
	for _, jp := range posts {
		link := strings.TrimSpace(jp.Link)
		if link == "" || seen[link] {
			continue
		}
		seen[link] = true
		// Skip past departures without fetching (title ends with "8 Nov 2026").
		title := lstUnescape(jp.Title.Rendered)
		if m := rabbaniTitleDateRe.FindString(title); m != "" {
			if t, ok := lstParseDate(m); ok && t.Before(cutoff) {
				continue
			}
		}
		status := ""
		for _, c := range jp.ClassList {
			if strings.HasPrefix(c, "status_jadwal-") {
				status = strings.TrimPrefix(c, "status_jadwal-")
			}
		}

		if fetched > 0 {
			time.Sleep(lstPoliteDelay / 2)
		}
		fetched++
		html, err := lstFetchPolite(link)
		if err != nil {
			log.Printf("[crawler] error on %s: %v\n", link, err)
			continue
		}
		p, ok := parseRabbaniJadwal(html, link)
		if !ok {
			log.Printf("[crawler] rabbanitour.com: no package data on %s\n", link)
			continue
		}
		if status == "full-booked" {
			for i := range p.Departures {
				p.Departures[i].Full = true
			}
		}
		if !lstFinalize(&p, cutoff) {
			continue
		}
		result = append(result, p)
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Departures[0].Date != result[j].Departures[0].Date {
			return result[i].Departures[0].Date < result[j].Departures[0].Date
		}
		return result[i].URL < result[j].URL
	})
	return result, nil
}

var rabbaniDurRe = regexp.MustCompile(`(?i)^(\d{1,2})\s*hari$`)

// parseRabbaniJadwal reads one /jadwal/... detail page.
func parseRabbaniJadwal(html, pageURL string) (CrawledPackage, bool) {
	lines := lstHTMLLines(html)
	p := CrawledPackage{TravelName: "Rabbani Tour", URL: pageURL}
	var dep CrawledDeparture

	// Header block: "Paket" / <name> / <date> / "9 Hari" / "Direct|Transit" / <airline>.
	for i := 0; i+2 < len(lines); i++ {
		if lines[i] != "Paket" {
			continue
		}
		t, ok := lstParseDate(lines[i+2])
		if !ok {
			continue
		}
		p.PackageName = lines[i+1]
		dep.Date = t.Format(lstISO)
		for j := i + 3; j < len(lines) && j <= i+6; j++ {
			l := lines[j]
			if m := rabbaniDurRe.FindStringSubmatch(l); m != nil {
				fmt.Sscanf(m[1], "%d", &p.Duration)
				continue
			}
			switch strings.ToLower(l) {
			case "direct":
				v := true
				p.Direct = &v
				continue
			case "transit":
				v := false
				p.Direct = &v
				continue
			}
			if _, isDate := lstParseDate(l); isDate {
				break
			}
		}
		break
	}
	if dep.Date == "" || p.PackageName == "" {
		return p, false
	}

	isHeader := func(l string) bool {
		switch l {
		case "Maskapai", "Hotel Madinah", "Hotel Mekkah", "Hotel Makkah", "Fasilitas Lainnya",
			"Include dan Eksclude", "Include", "Eksclude", "Exclude", "Catatan", "Harga Mulai", "*":
			return true
		}
		return strings.HasPrefix(l, "Terakhir update")
	}
	for i, l := range lines {
		next := ""
		if i+1 < len(lines) {
			next = lines[i+1]
		}
		switch l {
		case "Maskapai":
			if next != "" && !isHeader(next) {
				if a := lstAirlines(next); a != "" {
					p.Airline = a
				}
			}
		case "Hotel Madinah":
			if next != "" && !isHeader(next) && p.HotelMadinah == "" {
				p.HotelMadinah, p.StarsMadinah = lstSplitStars(lstCleanHotel(next))
				p.HotelMadinah = lstCleanHotel(p.HotelMadinah)
			}
		case "Hotel Mekkah", "Hotel Makkah":
			if next != "" && !isHeader(next) && p.HotelMakkah == "" {
				p.HotelMakkah, p.StarsMakkah = lstSplitStars(lstCleanHotel(next))
				p.HotelMakkah = lstCleanHotel(p.HotelMakkah)
			}
		case "Fasilitas Lainnya":
			// Next line is a generic intro sentence; the bonus items follow.
			if len(p.Facilities) == 0 && i+2 < len(lines) {
				p.Facilities = append(p.Facilities, lstListItems(lines, i+2, isHeader)...)
			}
		case "Include":
			p.Facilities = append(p.Facilities, lstListItems(lines, i+1, func(s string) bool {
				return isHeader(s) || strings.HasPrefix(s, "Tergantung Paket")
			})...)
		case "Eksclude", "Exclude":
			if len(p.Excludes) == 0 {
				p.Excludes = lstListItems(lines, i+1, func(s string) bool {
					return isHeader(s) || strings.HasPrefix(s, "Tergantung Paket")
				})
			}
		case "Harga Mulai":
			if v := lstMoney(next); v > 0 && p.Price == 0 {
				p.Price = v
			}
		case "Quad":
			if v := lstMoney(next); v > 0 {
				p.Price = v
			}
		case "Triple":
			if v := lstMoney(next); v > 0 {
				p.PriceTriple = v
			}
		case "Double":
			if v := lstMoney(next); v > 0 {
				p.PriceDouble = v
			}
		}
		if strings.HasPrefix(l, "Harga starting dari ") && p.Airport == "" {
			city := strings.Fields(strings.TrimPrefix(l, "Harga starting dari "))
			if len(city) > 0 {
				p.Airport = lstCity(strings.Trim(city[0], ",."))
			}
		}
	}
	p.Facilities = lstDedupStrings(p.Facilities)
	if p.Price == 0 {
		return p, false
	}
	p.Departures = []CrawledDeparture{dep}
	return p, true
}

func lstDedupStrings(in []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, s := range in {
		k := strings.ToLower(strings.TrimSpace(s))
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, s)
	}
	return out
}

// rabbaniCategoryName turns https://rabbanitour.com/paket/umroh-plus-turki/ into "Umroh Plus Turki".
func rabbaniCategoryName(catURL string) string {
	slug := strings.Trim(catURL[strings.Index(catURL, "/paket/")+len("/paket/"):], "/")
	words := strings.Split(slug, "-")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
