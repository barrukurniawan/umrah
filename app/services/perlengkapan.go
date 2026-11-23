package services

import (
	"encoding/json"
	"log"
	"net/url"
	"os"
	"sync"
)

// Daftar produk halaman /perlengkapan (affiliate Shopee), dari config/perlengkapan.json.
// Link affiliate diisi belakangan; selama kosong tombol mengarah ke pencarian Shopee.

type Product struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Reason   string `json:"reason"` // satu kalimat alasan
	Img      string `json:"img"`
	Search   string `json:"search"` // kata kunci cadangan untuk pencarian Shopee
	URL      string `json:"url"`    // link affiliate Shopee (boleh kosong)
	Featured bool   `json:"featured"`
}

// Href: link affiliate bila ada, kalau tidak pencarian Shopee dengan kata kunci.
func (p Product) Href() string {
	if p.URL != "" {
		return p.URL
	}
	q := p.Search
	if q == "" {
		q = p.Name
	}
	return "https://shopee.co.id/search?keyword=" + url.QueryEscape(q)
}

var (
	productMu sync.RWMutex
	products  []Product
)

func loadProducts() []Product {
	b, err := os.ReadFile("config/perlengkapan.json")
	if err != nil {
		log.Println("[perlengkapan]", err)
		return nil
	}
	var doc struct {
		Items []Product `json:"items"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		log.Println("[perlengkapan] config/perlengkapan.json tidak valid:", err)
		return nil
	}
	return doc.Items
}

// ReloadProducts membaca ulang config/perlengkapan.json (dipanggil saat SIGHUP).
func ReloadProducts() {
	p := loadProducts()
	productMu.Lock()
	products = p
	productMu.Unlock()
}

func Products() []Product {
	productMu.RLock()
	p := products
	productMu.RUnlock()
	if p == nil {
		ReloadProducts()
		productMu.RLock()
		p = products
		productMu.RUnlock()
	}
	return p
}

// ProductKeys: key yang valid (menyaring beacon klik dari klien).
func ProductKeys() map[string]bool {
	keys := map[string]bool{}
	for _, p := range Products() {
		keys[p.Key] = true
	}
	return keys
}
