package crawlers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

type AlhijazParser struct {
	URL string
}

func (a *AlhijazParser) Crawl() ([]CrawledPackage, error) {
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

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var html string
	err := chromedp.Run(ctx,
		chromedp.Navigate(a.URL),
		chromedp.WaitVisible(`[data-card-ref]`, chromedp.ByQuery),
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

	doc.Find("[data-card-ref]").Each(func(_ int, card *goquery.Selection) {
		pkg := CrawledPackage{
			TravelName: "Al Hijaz",
			URL:        a.URL,
		}

		pkg.PackageName = strings.TrimSpace(card.Find("h3").First().Text())

		priceEl := card.Find("p.text-orange-600, p.text-orange-400").First()
		pkg.Price = parseJutaStr(strings.TrimSpace(priceEl.Text()))

		flightEl := card.Find("span.font-medium.text-gray-700, span.font-medium.text-slate-200").First()
		flightCode := strings.TrimSpace(flightEl.Text())
		if strings.HasPrefix(flightCode, "SV") {
			pkg.Airline = "Saudia"
		} else if strings.HasPrefix(flightCode, "GA") {
			pkg.Airline = "Garuda Indonesia"
		} else if strings.HasPrefix(flightCode, "EK") {
			pkg.Airline = "Emirates"
		} else if strings.HasPrefix(flightCode, "QR") {
			pkg.Airline = "Qatar Airways"
		} else if al := KnownAirline(card.Text()); al != "" {
			pkg.Airline = al
		}

		card.Find("[class*='text-[10px]']").Each(func(_ int, label *goquery.Selection) {
			t := strings.TrimSpace(label.Text())
			lower := strings.ToLower(t)
			if lower == "mekkah" && pkg.HotelMakkah == "" {
				hotelP := label.Next()
				if hotelP.Length() > 0 {
					hotel := strings.TrimSpace(hotelP.Text())
					if !strings.HasPrefix(hotel, "★") && !strings.HasPrefix(hotel, "±") {
						pkg.HotelMakkah = hotel
					}
				}
			}
			if lower == "madinah" && pkg.HotelMadinah == "" {
				hotelP := label.Next()
				if hotelP.Length() > 0 {
					hotel := strings.TrimSpace(hotelP.Text())
					if !strings.HasPrefix(hotel, "★") && !strings.HasPrefix(hotel, "±") {
						pkg.HotelMadinah = hotel
					}
				}
			}
		})

		seatsSection := card.Find(".seat-info-section")
		if seatsSection.Length() > 0 {
			seatsText := seatsSection.Text()
			pkg.Seats = ParseSeats(seatsText)

			seatsSection.Find("span").Each(func(_ int, sp *goquery.Selection) {
				t := strings.TrimSpace(sp.Text())
				lower := strings.ToLower(t)
				if lower == "berangkat" {
					next := sp.Next()
					if next.Length() > 0 {
						depDate := strings.TrimSpace(next.Text())
						if depDate != "" && len(depDate) > 5 && countMonths(depDate) > 0 {
							pkg.DepartureDates = append(pkg.DepartureDates, depDate)
						}
					}
				}
			})
		}

		if pkg.Duration == 0 && pkg.PackageName != "" {
			pkg.Duration = ParseDuration(pkg.PackageName)
		}

		if pkg.Price < 10000 && pkg.PackageName != "" {
			pkg.Price = parseJutaStr(card.Find("p.text-orange-600, p.text-orange-400").First().Text())
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

func parseJutaStr(s string) int {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "rp")
	s = strings.Join(strings.Fields(s), "")

	parts := strings.Split(s, "jt")
	if len(parts) == 0 {
		return 0
	}
	numPart := strings.ReplaceAll(parts[0], ",", ".")
	var num float64
	fmt.Sscanf(numPart, "%f", &num)
	return int(num * 1000000)
}

func findChrome() string {
	for _, name := range []string{"chromium-browser", "chromium", "google-chrome", "google-chrome-stable"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	if _, err := os.Stat("/snap/bin/chromium"); err == nil {
		return "/snap/bin/chromium"
	}
	return ""
}
