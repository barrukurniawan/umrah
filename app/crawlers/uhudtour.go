package crawlers

import (
	"regexp"
	"strings"
)

// UhudTourParser crawls a single Uhud Tour package detail page
// (https://uhudtour.com/transaksi/paket-umrah/detail/<id>/<slug>) using the same
// per-departure AJAX data as UhudTourListParser.
type UhudTourParser struct {
	URL string
}

func (u *UhudTourParser) Crawl() ([]CrawledPackage, error) {
	if !hamdanEHDetailRe.MatchString(u.URL) {
		return (&UhudTourListParser{URL: u.URL}).Crawl()
	}
	site := uhudTourSite()
	return hamdanEHCrawlCards(site, []hamdanEHCard{{URL: strings.TrimSpace(u.URL)}}), nil
}

func stripTags(s string) string {
	return regexp.MustCompile(`<[^>]+>`).ReplaceAllString(s, "")
}
