package crawlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// UhudTourListParser starts from the Uhud Tour umrah listing page
// (https://uhudtour.com/transaksi/paket-umrah), discovers every current
// /transaksi/paket-umrah/detail/... package (following ?page=N while new links
// appear) and extracts each package from its listing card: title, departure
// date, hotels, airline, airport, remaining seats and "Harga mulai" price.
// Duration is read from the detail page's "Program Hari" select
// (<select name="program_hari"><option value="9">Umrah 9 Hari</option>).
type UhudTourListParser struct {
	// URL of the listing page; defaults to https://uhudtour.com/transaksi/paket-umrah.
	URL string
}

func NewUhudTourListParser(url string) *UhudTourListParser {
	return &UhudTourListParser{URL: url}
}

const (
	uhudListDefaultURL = "https://uhudtour.com/transaksi/paket-umrah"
	uhudListMaxPages   = 20
)

var (
	uhudListDetailRe   = regexp.MustCompile(`/transaksi/paket-umrah/detail/\d+/`)
	uhudListDateRe     = regexp.MustCompile(`(?i)(\d{1,2})\s+([a-z]+)\s+(20\d{2})`)
	uhudListCityRe     = regexp.MustCompile(`\s*\([^)]*\)\s*$`)
	uhudListAirportRe  = regexp.MustCompile(`\(([A-Z]{3})\)`)
	uhudListDurationRe = regexp.MustCompile(`(?is)<select[^>]*name="program_hari"[^>]*>.*?<option[^>]*value="(\d+)"`)
	uhudListMonths     = map[string]time.Month{
		"januari": 1, "jan": 1, "january": 1,
		"februari": 2, "feb": 2, "february": 2,
		"maret": 3, "mar": 3, "march": 3,
		"april": 4, "apr": 4,
		"mei": 5, "may": 5,
		"juni": 6, "jun": 6, "june": 6,
		"juli": 7, "jul": 7, "july": 7,
		"agustus": 8, "agu": 8, "agt": 8, "aug": 8, "august": 8,
		"september": 9, "sep": 9, "sept": 9,
		"oktober": 10, "okt": 10, "oct": 10, "october": 10,
		"november": 11, "nov": 11,
		"desember": 12, "des": 12, "dec": 12, "december": 12,
	}
)

func uhudListGet(u string) (string, error) {
	log.Println("[crawler] visiting:", u)
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("uhudtour: %s returned HTTP %d", u, resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	return string(b), err
}

func uhudListParseDate(s string) (time.Time, bool) {
	m := uhudListDateRe.FindStringSubmatch(s)
	if len(m) < 4 {
		return time.Time{}, false
	}
	mon, ok := uhudListMonths[strings.ToLower(m[2])]
	if !ok {
		return time.Time{}, false
	}
	day, _ := strconv.Atoi(m[1])
	year, _ := strconv.Atoi(m[3])
	return time.Date(year, mon, day, 0, 0, 0, 0, time.UTC), true
}

func (u *UhudTourListParser) Crawl() ([]CrawledPackage, error) {
	listURL := u.URL
	if listURL == "" {
		listURL = uhudListDefaultURL
	}

	cutoff := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	if today := time.Now().UTC().Truncate(24 * time.Hour); today.After(cutoff) {
		cutoff = today
	}

	var packages []CrawledPackage
	seen := map[string]bool{}

	for page := 1; page <= uhudListMaxPages; page++ {
		pageURL := listURL
		if page > 1 {
			pu, err := url.Parse(listURL)
			if err != nil {
				break
			}
			q := pu.Query()
			q.Set("page", strconv.Itoa(page))
			pu.RawQuery = q.Encode()
			pageURL = pu.String()
		}

		html, err := uhudListGet(pageURL)
		if err != nil {
			if page == 1 {
				return nil, err
			}
			log.Printf("[crawler] uhudtour list page %d: %v\n", page, err)
			break
		}
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			return nil, err
		}

		newLinks := 0
		doc.Find("div.list-paket-umrah").Each(func(_ int, card *goquery.Selection) {
			title := card.Find("a.paket-title")
			href, _ := title.Attr("href")
			if href == "" || !uhudListDetailRe.MatchString(href) {
				return
			}
			if strings.HasPrefix(href, "/") {
				href = "https://uhudtour.com" + href
			}
			if seen[href] {
				return
			}
			seen[href] = true
			newLinks++

			pkg := CrawledPackage{
				TravelName:     "Uhud Tour",
				PackageName:    CleanText(strings.Join(strings.Fields(title.Text()), " ")),
				URL:            href,
				DepartureDates: []string{},
			}

			var dep time.Time
			hasDate := false
			card.Find("table tr").Each(func(_ int, tr *goquery.Selection) {
				icon, _ := tr.Find("i").Attr("class")
				val := CleanText(strings.Join(strings.Fields(tr.Find("td").Last().Text()), " "))
				lower := strings.ToLower(val)
				switch {
				case strings.Contains(icon, "fa-calendar"):
					if t, ok := uhudListParseDate(val); ok {
						dep, hasDate = t, true
					}
				case strings.Contains(icon, "fa-building"):
					name := strings.TrimSpace(uhudListCityRe.ReplaceAllString(val, ""))
					switch {
					case strings.Contains(lower, "(makkah)") || strings.Contains(lower, "(mekkah)"):
						if pkg.HotelMakkah == "" {
							pkg.HotelMakkah = name
						}
					case strings.Contains(lower, "(madinah)"):
						if pkg.HotelMadinah == "" {
							pkg.HotelMadinah = name
						}
					}
				case strings.Contains(icon, "fa-plane"):
					pkg.Airline = KnownAirline(val)
					if pkg.Airline == "" {
						pkg.Airline = val
					}
				case strings.Contains(icon, "fa-map-marker"):
					if m := uhudListAirportRe.FindStringSubmatch(val); len(m) > 1 {
						pkg.Airport = m[1]
					} else {
						pkg.Airport = val
					}
				}
			})

			if !hasDate {
				if t, ok := uhudListParseDate(pkg.PackageName); ok {
					dep, hasDate = t, true
				}
			}
			if !hasDate || dep.Before(cutoff) {
				return
			}
			pkg.DepartureDates = append(pkg.DepartureDates, dep.Format("02 Jan 2006"))

			pkg.Seats = ParseSeats(card.Find(".m-list-paket-sisa-seat span").Text())
			pkg.Price = ParsePrice(card.Find(".harga-paket").Text())
			if pkg.Airport == "" {
				pkg.Airport = "CGK"
			}

			packages = append(packages, pkg)
		})

		if newLinks == 0 {
			break
		}
	}

	// Duration lives only on the detail page ("Program Hari" select).
	for i := range packages {
		html, err := uhudListGet(packages[i].URL)
		if err != nil {
			log.Printf("[crawler] uhudtour detail %s: %v\n", packages[i].URL, err)
			continue
		}
		if m := uhudListDurationRe.FindStringSubmatch(html); len(m) > 1 {
			if d, _ := strconv.Atoi(m[1]); d > 0 && d < 60 {
				packages[i].Duration = d
			}
		}
	}

	var out []CrawledPackage
	for _, p := range packages {
		if p.Price > 0 && p.PackageName != "" {
			out = append(out, p)
		}
	}
	return out, nil
}
