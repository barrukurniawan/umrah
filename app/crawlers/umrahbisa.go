package crawlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

type UmrahBisaParser struct {
	URL string
}

func (u *UmrahBisaParser) Crawl() ([]CrawledPackage, error) {
	var packages []CrawledPackage
	seen := make(map[string]bool)

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var html string
	err := chromedp.Run(ctx,
		chromedp.Navigate(u.URL),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
		chromedp.OuterHTML(`html`, &html),
	)
	if err != nil {
		return packages, fmt.Errorf("chromedp: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return packages, err
	}

	doc.Find("[class*='card'], [class*='paket'], [class*='package']").Each(func(_ int, e *goquery.Selection) {
		pkg := CrawledPackage{
			TravelName: "Umrah Bisa",
			URL:        u.URL,
		}

		pkg.PackageName = CleanText(e.Find("h2, h3, [class*='title']").First().Text())
		pkg.Price = ParsePrice(e.Find("[class*='price'], [class*='harga']").First().Text())
		pkg.Duration = ParseDuration(e.Find("[class*='duration'], [class*='durasi']").First().Text())
		pkg.Airline = KnownAirline(e.Find("[class*='airline'], [class*='maskapai']").First().Text())

		if pkg.Airline == "" {
			pkg.Airline = KnownAirline(e.Text())
		}

		if pkg.PackageName != "" && pkg.Price > 0 {
			key := fmt.Sprintf("%s|%d", pkg.PackageName, pkg.Price)
			if !seen[key] {
				seen[key] = true
				packages = append(packages, pkg)
			}
		}
	})

	return packages, nil
}
