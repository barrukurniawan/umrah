package crawlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

type MuslimPergiParser struct {
	URL        string
	TravelName string
}

func (mp *MuslimPergiParser) Crawl() ([]CrawledPackage, error) {
	if mp.TravelName == "" {
		mp.TravelName = "Muslim Pergi"
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
	)
	if p := findChrome(); p != "" {
		opts = append(opts, chromedp.ExecPath(p))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	var html string
	err := chromedp.Run(ctx,
		chromedp.Navigate(mp.URL),
		chromedp.WaitVisible(`[class*='card'], [class*='listing'], [class*='item']`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.OuterHTML(`html`, &html),
	)
	if err != nil {
		return nil, fmt.Errorf("chromedp: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	var packages []CrawledPackage
	seen := make(map[string]bool)

	doc.Find("[class*='card'], [class*='listing'], [class*='item']").Each(func(_ int, card *goquery.Selection) {
		text := strings.ToLower(card.Text())
		if !strings.Contains(text, "available pax") && !strings.Contains(text, "fully booked") {
			return
		}
		if !strings.Contains(text, "harga mulai") && !strings.Contains(text, "rp") {
			return
		}

		pkg := CrawledPackage{
			TravelName: mp.TravelName,
			URL:        mp.URL,
		}

		card.Find("h1, h2, h3, h4, [class*='title'], [class*='name']").Each(func(_ int, el *goquery.Selection) {
			t := CleanText(el.Text())
			if t != "" && len(t) > 5 && pkg.PackageName == "" {
				pkg.PackageName = t
			}
		})

		card.Find("[class*='price'], [class*='harga']").Each(func(_ int, el *goquery.Selection) {
			if pkg.Price == 0 {
				pkg.Price = ParsePrice(el.Text())
			}
		})

		card.Find("[class*='duration'], [class*='durasi']").Each(func(_ int, el *goquery.Selection) {
			if pkg.Duration == 0 {
				pkg.Duration = ParseDuration(el.Text())
			}
		})

		if pkg.Airline == "" {
			pkg.Airline = KnownAirline(card.Text())
		}

		fullText := card.Text()
		if strings.Contains(strings.ToLower(fullText), "fully booked") {
			pkg.Seats = 0
		}
		if idx := strings.Index(strings.ToLower(fullText), "available pax"); idx >= 0 {
			sub := fullText[idx+len("available pax"):]
			pkg.Seats = ParseSeats(sub)
		}

		if pkg.PackageName == "" {
			lines := strings.Split(card.Text(), "\n")
			for _, line := range lines {
				line = CleanText(line)
				if len(line) > 10 && !strings.HasPrefix(strings.ToLower(line), "available") &&
					!strings.HasPrefix(strings.ToLower(line), "harga") && !strings.HasPrefix(strings.ToLower(line), "fully") {
					pkg.PackageName = line
					break
				}
			}
		}

		if pkg.PackageName != "" {
			key := fmt.Sprintf("%s|%d", pkg.PackageName, pkg.Price)
			if !seen[key] {
				seen[key] = true
				packages = append(packages, pkg)
			}
		}
	})

	return packages, nil
}
