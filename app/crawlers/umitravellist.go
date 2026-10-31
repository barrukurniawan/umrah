package crawlers

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// UmiTravelListParser discovers the current UMI Tour & Travel packages from the
// paginated listing page (https://umi.travel/listings?page=N) and scrapes each
// product page with the existing UmiTravelParser.
type UmiTravelListParser struct {
	URL string // listing URL, e.g. https://umi.travel/listings
}

func NewUmiTravelListParser(listURL string) *UmiTravelListParser {
	return &UmiTravelListParser{URL: listURL}
}

var umiListLinkRe = regexp.MustCompile(`href="((?:https?://(?:www\.)?umi\.travel)?/listings/[a-z0-9][a-z0-9-]*)"`)

const umiListMaxPages = 30

func (u *UmiTravelListParser) Crawl() ([]CrawledPackage, error) {
	listURL := u.URL
	if listURL == "" {
		listURL = "https://umi.travel/listings"
	}
	base, err := url.Parse(listURL)
	if err != nil {
		return nil, err
	}

	// 1. Discover product URLs, following ?page=N until a page adds nothing new.
	seen := map[string]bool{}
	var productURLs []string
	for page := 1; page <= umiListMaxPages; page++ {
		pageURL := listURL
		if page > 1 {
			q := base.Query()
			q.Set("page", fmt.Sprintf("%d", page))
			pu := *base
			pu.RawQuery = q.Encode()
			pageURL = pu.String()
		}
		log.Println("[crawler] visiting:", pageURL)
		html, err := lstFetch(pageURL)
		if err != nil {
			if page == 1 {
				return nil, err
			}
			log.Printf("[crawler] error on %s: %v\n", pageURL, err)
			break
		}
		added := 0
		for _, m := range umiListLinkRe.FindAllStringSubmatch(html, -1) {
			ref, err := base.Parse(m[1])
			if err != nil {
				continue
			}
			abs := ref.String()
			if seen[abs] {
				continue
			}
			seen[abs] = true
			if !umiIsUmrahSlug(abs) {
				continue
			}
			productURLs = append(productURLs, abs)
			added++
		}
		if added == 0 {
			break
		}
		time.Sleep(lstPoliteDelay)
	}
	log.Printf("[crawler] umi.travel: %d product links discovered\n", len(productURLs))

	// 2. Scrape each product with the existing per-product parser.
	cutoff := lstCutoff()
	var result []CrawledPackage
	for i, pu := range productURLs {
		if i > 0 {
			time.Sleep(lstPoliteDelay)
		}
		// Our own copy of the page: used to skip non-umrah products and to fill
		// fields UmiTravelParser no longer finds (hotels, departure city).
		html, err := lstFetch(pu)
		if err != nil {
			log.Printf("[crawler] error on %s: %v\n", pu, err)
			continue
		}
		info := umiPageInfo(html)
		if info.tipe != "" && !strings.Contains(strings.ToLower(info.tipe), "umro") &&
			!strings.Contains(strings.ToLower(info.tipe), "umra") {
			continue
		}
		time.Sleep(lstPoliteDelay)

		pkgs, err := (&UmiTravelParser{URL: pu}).Crawl()
		if err != nil {
			log.Printf("[crawler] error on %s: %v\n", pu, err)
			continue
		}
		for _, p := range pkgs {
			p.TravelName = "UMI Tour & Travel"
			p.URL = pu
			if p.HotelMakkah == "" {
				p.HotelMakkah = info.hotelMakkah
			}
			if p.HotelMadinah == "" {
				p.HotelMadinah = info.hotelMadinah
			}
			// UmiTravelParser always sets CGK; use the page's departure city instead.
			if info.airport != "" {
				p.Airport = info.airport
			}
			if lstKeepPackage(&p, cutoff) {
				result = append(result, p)
			}
		}
	}
	return result, nil
}

// umiIsUmrahSlug keeps umrah/umroh products and skips hajj / halal-tour listings.
func umiIsUmrahSlug(u string) bool {
	slug := strings.ToLower(u[strings.LastIndex(u, "/")+1:])
	if !strings.Contains(slug, "umro") && !strings.Contains(slug, "umra") {
		return false
	}
	return !strings.Contains(slug, "haji") && !strings.Contains(slug, "hajj")
}

type umiInfo struct {
	tipe, hotelMakkah, hotelMadinah, airport string
}

var umiCityAirport = map[string]string{
	"JAKARTA": "CGK", "SURABAYA": "SUB", "DENPASAR": "DPS", "BALI": "DPS",
	"MAKASSAR": "UPG", "MEDAN": "KNO", "SOLO": "SOC", "BATAM": "BTH",
	"BANDUNG": "BDO", "YOGYAKARTA": "YIA", "SEMARANG": "SRG", "BALIKPAPAN": "BPN",
}

// umiPageInfo reads the "Tipe Paket" / "Maskapai" header block and the
// "<hotel> / Hotel Rate / Lokasi : / <city>" hotel cards of a product page.
func umiPageInfo(html string) umiInfo {
	var info umiInfo
	lines := lstHTMLLines(html)
	for i, l := range lines {
		switch {
		case l == "Tipe Paket" && i+1 < len(lines) && info.tipe == "":
			info.tipe = lines[i+1]
		case l == "Maskapai" && i+3 < len(lines) && info.airport == "":
			// Maskapai, <airline>, <CITY>, <date>
			info.airport = umiCityAirport[strings.ToUpper(lines[i+2])]
		case strings.HasPrefix(strings.ToLower(l), "lokasi") && i+1 < len(lines):
			city := strings.ToLower(strings.Trim(lines[i+1], " :"))
			if strings.HasPrefix(strings.ToLower(l), "lokasi :") && len(l) > 9 {
				city = strings.ToLower(strings.TrimSpace(l[9:]))
			}
			name := ""
			for k := i - 1; k >= 0 && k >= i-3; k-- {
				c := lines[k]
				if strings.EqualFold(c, "Hotel Rate") || strings.EqualFold(c, "Hotel") {
					continue
				}
				name = c
				break
			}
			if len(name) < 3 || len(name) > 60 {
				continue
			}
			switch city {
			case "makkah", "mekkah", "mekah":
				if info.hotelMakkah == "" {
					info.hotelMakkah = name
				}
			case "madinah":
				if info.hotelMadinah == "" {
					info.hotelMadinah = name
				}
			}
		}
	}
	return info
}
