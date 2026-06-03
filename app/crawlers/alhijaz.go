package crawlers

import "github.com/gocolly/colly/v2"

type AlhijazParser struct {
	URL string
}

func (a *AlhijazParser) Crawl() ([]CrawledPackage, error) {
	var packages []CrawledPackage

	c := NewCollector("alhijaz.co")

	c.OnHTML("[class*='card'], [class*='paket'], [class*='package']", func(e *colly.HTMLElement) {
		pkg := CrawledPackage{
			TravelName: "Al Hijaz",
			URL:        a.URL,
		}

		pkg.PackageName = CleanText(e.ChildText("h2, h3, [class*='title']"))
		pkg.Price = ParsePrice(e.ChildText("[class*='price'], [class*='harga']"))
		pkg.Duration = ParseDuration(e.ChildText("[class*='duration'], [class*='durasi']"))
		pkg.Airline = CleanText(e.ChildText("[class*='airline'], [class*='maskapai']"))
		pkg.HotelMakkah = CleanText(e.ChildText("[class*='makkah']"))
		pkg.HotelMadinah = CleanText(e.ChildText("[class*='madinah']"))
		pkg.Seats = ParseSeats(e.ChildText("[class*='seat'], [class*='kursi'], [class*='sisa']"))

		if pkg.PackageName != "" && pkg.Price > 0 {
			packages = append(packages, pkg)
		}
	})

	c.Visit(a.URL)
	c.Wait()
	return packages, nil
}
