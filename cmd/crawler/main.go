package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"umrah/app/crawlers"
)

type Site struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	Parser     string `json:"parser"`
	TravelName string `json:"travel_name,omitempty"`
}

type CrawlResult struct {
	Timestamp string                    `json:"timestamp"`
	Site      string                    `json:"site"`
	URL       string                    `json:"url"`
	Packages  []crawlers.CrawledPackage `json:"packages"`
	Error     string                    `json:"error,omitempty"`
}

func main() {
	data, err := os.ReadFile("config/sites.json")
	if err != nil {
		log.Fatalf("gagal membaca config/sites.json: %v", err)
	}

	var sites []Site
	if err := json.Unmarshal(data, &sites); err != nil {
		log.Fatalf("gagal parse config: %v", err)
	}

	ts := time.Now().Format("2006-01-02_15-04")
	os.MkdirAll("output", 0755)

	var allResults []CrawlResult

	for _, site := range sites {
		log.Printf("[main] crawling %s (%s)...\n", site.Name, site.Parser)

		var parser crawlers.Parser

		switch site.Parser {
		case "hamdan":
			parser = &crawlers.HamdanParser{URL: site.URL}
		case "taiba":
			parser = &crawlers.TaibaParser{URL: site.URL}
		case "alhijaz":
			parser = &crawlers.AlhijazParser{URL: site.URL}
		case "mustajab":
			tn := site.TravelName
			if tn == "" {
				tn = site.Name
			}
			parser = &crawlers.MustajabParser{URL: site.URL, TravelName: tn}
		case "muslimpergi":
			tn := site.TravelName
			if tn == "" {
				tn = site.Name
			}
			parser = &crawlers.MuslimPergiParser{URL: site.URL, TravelName: tn}
		case "rabbani":
			parser = &crawlers.RabbaniParser{URL: site.URL}
		case "umrahbisa_product":
			parser = &crawlers.UmrahBisaProductParser{URL: site.URL}
		case "uhudtour_product":
			parser = &crawlers.UhudTourParser{URL: site.URL}
		case "umi_travel_product":
			parser = &crawlers.UmiTravelParser{URL: site.URL}
		default:
			log.Printf("[main] parser '%s' tidak dikenal, skip\n", site.Parser)
			continue
		}

		result := CrawlResult{
			Timestamp: time.Now().Format(time.RFC3339),
			Site:      site.Name,
			URL:       site.URL,
		}

		pkgs, err := parser.Crawl()
		if err != nil {
			result.Error = err.Error()
			log.Printf("[main] error crawling %s: %v\n", site.Name, err)
		} else {
			result.Packages = pkgs
			log.Printf("[main] %s: %d paket ditemukan\n", site.Name, len(pkgs))
		}

		allResults = append(allResults, result)

		siteFile := fmt.Sprintf("output/%s_%s.json", ts, sanitizeName(site.Name))
		siteData, _ := json.MarshalIndent(result, "", "  ")
		os.WriteFile(siteFile, siteData, 0644)
		log.Printf("[main] disimpan ke %s\n", siteFile)
	}

	// Inject manual data
	allResults = append(allResults, getManualData())

	allFile := filepath.Join("output", fmt.Sprintf("all_%s.json", ts))
	allData, _ := json.MarshalIndent(allResults, "", "  ")
	os.WriteFile(allFile, allData, 0644)

	log.Printf("[main] selesai. total %d site, output di %s\n", len(allResults), allFile)
}

func sanitizeName(name string) string {
	r := strings.NewReplacer(" ", "_", ".", "", "/", "")
	return strings.ToLower(r.Replace(name))
}

func getManualData() CrawlResult {
	return CrawlResult{
		Timestamp: time.Now().Format(time.RFC3339),
		Site:      "Data Manual",
		URL:       "manual",
		Packages: []crawlers.CrawledPackage{
			{
				TravelName:     "PT. Khasanah Global Travelindo",
				PackageName:    "Umrah Khasanah 10 Hari",
				Price:          25900000,
				Duration:       10,
				Airline:        "Saudia Airlines",
				HotelMakkah:    "Maysan Almaqam",
				HotelMadinah:   "Araik Taiba",
				DepartureDates: []string{"13 Okt 2026"},
				Seats:          45,
				Airport:        "Jakarta",
				URL:            "manual-khasanah",
			},
			{
				TravelName:     "PT Labbaika Cipta Imani",
				PackageName:    "Labbaika Umrah & Hajj Tour 9 Hari",
				Price:          26900000,
				Duration:       9,
				Airline:        "Saudi Airlines",
				HotelMakkah:    "Almassa Badr",
				HotelMadinah:   "Jawharat Al Rasheed",
				DepartureDates: []string{"28 Okt 2026"},
				Seats:          45,
				Airport:        "Jakarta",
				URL:            "manual-labbaika",
			},
			{
				TravelName:     "PT Rahmah Grup Internasional",
				PackageName:    "Umrah Rahmah Paket Hemat 9 Hari",
				Price:          23900000,
				Duration:       9,
				Airline:        "Etihad Airways",
				HotelMakkah:    "Emaar Noor",
				HotelMadinah:   "Jawharat Arrasyid",
				DepartureDates: []string{"17 Sep 2026", "4 Okt 2026", "17 Okt 2026", "1 Nov 2026", "11 Nov 2026"},
				Seats:          45,
				Airport:        "Jakarta",
				URL:            "manual-rahmah-hemat",
			},
			{
				TravelName:     "PT Rahmah Grup Internasional",
				PackageName:    "Umrah Rahmah Paket Reguler 9 Hari",
				Price:          26900000,
				Duration:       9,
				Airline:        "Etihad Airways",
				HotelMakkah:    "Winner Inn Ajyad",
				HotelMadinah:   "Jawharat Arrasyid",
				DepartureDates: []string{"17 Sep 2026", "4 Okt 2026", "17 Okt 2026", "1 Nov 2026", "11 Nov 2026"},
				Seats:          45,
				Airport:        "Jakarta",
				URL:            "manual-rahmah-reguler",
			},
		},
	}
}
