package crawlers

import "strings"

// LabbaikaParser crawls PT Labbaika Cipta Imani's umrah listing
// (https://labbaika.co.id/transaksi/paket-umrah?page=N), an erahajj-platform
// site. Cards without a published date ("Tanggal Belum Tersedia") are skipped;
// every other package is read per departure from the detail page's AJAX
// endpoints (see hamdan_extra.go).
type LabbaikaParser struct {
	URL string
}

func (l *LabbaikaParser) Crawl() ([]CrawledPackage, error) {
	listURL := "https://labbaika.co.id/transaksi/paket-umrah"
	if strings.Contains(l.URL, "/transaksi/paket-umrah") {
		listURL = strings.SplitN(l.URL, "?", 2)[0]
	}
	return hamdanEHCrawl(hamdanEHSite{
		Base:       "https://labbaika.co.id",
		ListURL:    listURL,
		TravelName: "PT Labbaika Cipta Imani",
		Workers:    5,
	})
}
