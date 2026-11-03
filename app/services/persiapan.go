package services

import (
	"encoding/json"
	"html/template"
	"log"
	"os"
	"sync"
)

// Checklist halaman /persiapan, dari config/persiapan.json (bisa diubah tanpa ubah kode).

type PrepLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type PrepItem struct {
	Key     string          `json:"key"`
	Title   string          `json:"title"`
	Tag     string          `json:"tag"`    // wajib | opsional | cek
	Gender  string          `json:"gender"` // m | f | "" (semua)
	DueDays int             `json:"due_days"`
	Desc    template.HTML   `json:"desc"`
	Tips    []template.HTML `json:"tips"`
	Link    *PrepLink       `json:"link"`
}

type PrepSection struct {
	Key   string     `json:"key"`
	Title string     `json:"title"`
	Icon  string     `json:"icon"`
	Items []PrepItem `json:"items"`
}

var (
	prepMu       sync.RWMutex
	prepSections []PrepSection
)

func loadPrep() []PrepSection {
	b, err := os.ReadFile("config/persiapan.json")
	if err != nil {
		log.Println("[persiapan]", err)
		return nil
	}
	var doc struct {
		Sections []PrepSection `json:"sections"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		log.Println("[persiapan] config/persiapan.json tidak valid:", err)
		return nil
	}
	return doc.Sections
}

// ReloadPrep membaca ulang config/persiapan.json (dipanggil saat SIGHUP).
func ReloadPrep() {
	s := loadPrep()
	prepMu.Lock()
	prepSections = s
	prepMu.Unlock()
}

func PrepSections() []PrepSection {
	prepMu.RLock()
	s := prepSections
	prepMu.RUnlock()
	if s == nil {
		ReloadPrep()
		prepMu.RLock()
		s = prepSections
		prepMu.RUnlock()
	}
	return s
}

// PrepKeys: semua key yang valid (untuk menyaring data yang dikirim klien).
func PrepKeys() map[string]bool {
	keys := map[string]bool{}
	for _, s := range PrepSections() {
		for _, it := range s.Items {
			keys[it.Key] = true
		}
	}
	return keys
}
