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

	allFile := filepath.Join("output", fmt.Sprintf("all_%s.json", ts))
	allData, _ := json.MarshalIndent(allResults, "", "  ")
	os.WriteFile(allFile, allData, 0644)

	log.Printf("[main] selesai. total %d site, output di %s\n", len(allResults), allFile)
}

func sanitizeName(name string) string {
	r := strings.NewReplacer(" ", "_", ".", "", "/", "")
	return strings.ToLower(r.Replace(name))
}
