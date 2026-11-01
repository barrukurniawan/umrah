package crawlers

// UhudTourListParser crawls the Uhud Tour umrah listing
// (https://uhudtour.com/transaksi/paket-umrah), an erahajj-platform site.
// Each /transaksi/paket-umrah/detail/... package is read per departure from the
// detail page's AJAX endpoints (see hamdan_extra.go).
type UhudTourListParser struct {
	// URL of the listing page; defaults to https://uhudtour.com/transaksi/paket-umrah.
	URL string
}

func NewUhudTourListParser(url string) *UhudTourListParser {
	return &UhudTourListParser{URL: url}
}

const uhudListDefaultURL = "https://uhudtour.com/transaksi/paket-umrah"

func uhudTourSite() hamdanEHSite {
	return hamdanEHSite{
		Base:       "https://uhudtour.com",
		ListURL:    uhudListDefaultURL,
		TravelName: "Uhud Tour",
		Workers:    4,
	}
}

func (u *UhudTourListParser) Crawl() ([]CrawledPackage, error) {
	site := uhudTourSite()
	if u.URL != "" {
		site.ListURL = u.URL
	}
	return hamdanEHCrawl(site)
}
