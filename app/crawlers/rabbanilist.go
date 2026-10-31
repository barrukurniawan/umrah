package crawlers

import (
	"log"
	"regexp"
	"sort"
	"strings"
	"time"
)

// RabbaniListParser discovers the current Rabbani Tour program categories from
// the site menu (links like https://rabbanitour.com/paket/<category>/) and runs
// the existing RabbaniParser on every category page (it follows pagination).
//
// The old /program-umroh/<category>/ URLs now redirect to /paket-umroh/<category>/
// landing pages that only show stale schedules; /paket/<category>/ is the
// live schedule listing.
type RabbaniListParser struct {
	URL string // page whose menu lists the categories, e.g. https://rabbanitour.com/
}

func NewRabbaniListParser(homeURL string) *RabbaniListParser {
	return &RabbaniListParser{URL: homeURL}
}

var rabbaniCategoryLinkRe = regexp.MustCompile(`href="(https://(?:www\.)?rabbanitour\.com/paket/([a-z0-9-]+)/)"`)

func (r *RabbaniListParser) Crawl() ([]CrawledPackage, error) {
	home := r.URL
	if home == "" {
		home = "https://rabbanitour.com/"
	}
	log.Println("[crawler] visiting:", home)
	html, err := lstFetch(home)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	var categories []string
	for _, m := range rabbaniCategoryLinkRe.FindAllStringSubmatch(html, -1) {
		slug := m[2]
		catURL := "https://rabbanitour.com/paket/" + slug + "/"
		if seen[catURL] || !strings.Contains(slug, "umro") {
			continue
		}
		seen[catURL] = true
		categories = append(categories, catURL)
	}
	sort.Strings(categories)
	log.Printf("[crawler] rabbanitour.com: %d categories discovered\n", len(categories))

	cutoff := lstCutoff()
	var result []CrawledPackage
	dedup := map[string]bool{}
	for i, catURL := range categories {
		if i > 0 {
			time.Sleep(lstPoliteDelay)
		}
		pkgs, err := (&RabbaniParser{URL: catURL}).Crawl()
		if err != nil {
			log.Printf("[crawler] error on %s: %v\n", catURL, err)
			continue
		}
		for _, p := range pkgs {
			p.TravelName = "Rabbani Tour"
			// RabbaniParser sometimes takes a status badge as the name.
			switch strings.ToLower(strings.TrimSpace(p.PackageName)) {
			case "", "full booked", "push selling", "penuh", "booking", "tersedia", "sisa seat":
				p.PackageName = rabbaniCategoryName(catURL)
			}
			if !lstKeepPackage(&p, cutoff) {
				continue
			}
			key := p.PackageName + "|" + strings.Join(p.DepartureDates, ",")
			if dedup[key] {
				continue
			}
			dedup[key] = true
			result = append(result, p)
		}
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].URL != result[j].URL {
			return result[i].URL < result[j].URL
		}
		ti, _ := lstParseDate(result[i].DepartureDates[0])
		tj, _ := lstParseDate(result[j].DepartureDates[0])
		return ti.Before(tj)
	})
	return result, nil
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
