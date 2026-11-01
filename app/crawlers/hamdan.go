package crawlers

// HamdanParser crawls Hamdan Tour (https://hamdantour.id/transaksi/paket-umrah),
// an erahajj-platform site. Every package's departures, room prices, hotels,
// airline and flights are read from the AJAX endpoints the detail page itself
// uses (see hamdan_extra.go).
type HamdanParser struct {
	URL string
}

func (h *HamdanParser) Crawl() ([]CrawledPackage, error) {
	listURL := "https://hamdantour.id/transaksi/paket-umrah"
	if h.URL != "" {
		listURL = h.URL
	}
	return hamdanEHCrawl(hamdanEHSite{
		Base:       "https://hamdantour.id",
		ListURL:    listURL,
		TravelName: "Hamdan Tour",
		Workers:    4,
	})
}
