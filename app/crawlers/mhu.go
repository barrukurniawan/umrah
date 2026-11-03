package crawlers

// MHUParser: PT Manasik Haji Umrah (https://manasikhajiumrah.com/transaksi/paket-umrah),
// situs platform erahajj yang sama dengan Hamdan/Uhud/Labbaika; dibaca lewat parser bersama.
type MHUParser struct {
	URL, TravelName string
}

func (m *MHUParser) Crawl() ([]CrawledPackage, error) {
	site := hamdanEHSite{
		Base:       "https://manasikhajiumrah.com",
		ListURL:    "https://manasikhajiumrah.com/transaksi/paket-umrah",
		TravelName: "MHU",
		Workers:    3,
	}
	if m.URL != "" {
		site.ListURL = m.URL
	}
	if m.TravelName != "" {
		site.TravelName = m.TravelName
	}
	return hamdanEHCrawl(site)
}
