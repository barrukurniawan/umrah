package services

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"
)

// Koordinat hotel dan jarak ke pelataran masjid (config/hotel_geo.json).
// Makkah: garis lurus ke tepi pelataran Masjidil Haram. Madinah: ke gerbang pelataran
// Nabawi terdekat. verified=true berarti diukur langsung oleh pemilik situs.

type HotelGeo struct {
	City             string   `json:"city"` // "makkah" | "madinah"
	Name             string   `json:"name"`
	Aliases          []string `json:"aliases"`
	Lat              float64  `json:"lat"`
	Lng              float64  `json:"lng"`
	DistanceM        int      `json:"distance_m"`
	Gate             string   `json:"gate"`
	GateLat, GateLng float64
	Verified         bool   `json:"verified"`
	Shuttle          bool   `json:"shuttle"`      // hotel punya shuttle ke Haram (ada sumbernya)
	ShuttleNote      string `json:"shuttle_note"` // mis. "Shuttle gratis hotel ke sekitar Abraj Al Bait"
}

func (h *HotelGeo) UnmarshalJSON(b []byte) error {
	type alias HotelGeo
	var raw struct {
		alias
		GateLat float64 `json:"gate_lat"`
		GateLng float64 `json:"gate_lng"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*h = HotelGeo(raw.alias)
	h.GateLat, h.GateLng = raw.GateLat, raw.GateLng
	return nil
}

var (
	geoMu    sync.RWMutex
	geoIndex map[string]HotelGeo // city|nama-normal -> data; nil = belum dimuat
)

func geoKey(city, name string) string {
	return city + "|" + strings.Join(strings.Fields(strings.ToLower(CleanHotelName(name))), " ")
}

// ReloadHotelGeo membaca ulang config/hotel_geo.json (dipanggil saat SIGHUP).
func ReloadHotelGeo() {
	geoMu.Lock()
	geoIndex = nil
	geoMu.Unlock()
	LookupHotelGeo("", "")
}

func loadHotelGeo() map[string]HotelGeo {
	idx := map[string]HotelGeo{}
	path := os.Getenv("HOTEL_GEO")
	if path == "" {
		path = "config/hotel_geo.json"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return idx
	}
	var doc struct {
		Hotels []HotelGeo `json:"hotels"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		log.Printf("[geo] %s tidak valid: %v", path, err)
		return idx
	}
	for _, h := range doc.Hotels {
		idx[geoKey(h.City, h.Name)] = h
		for _, a := range h.Aliases {
			idx[geoKey(h.City, a)] = h
		}
	}
	log.Printf("[geo] %d hotel dengan koordinat", len(doc.Hotels))
	return idx
}

// LookupHotelGeo mencari hotel berdasarkan nama persis seperti di data travel (atau variasinya).
func LookupHotelGeo(city, name string) (HotelGeo, bool) {
	geoMu.RLock()
	idx := geoIndex
	geoMu.RUnlock()
	if idx == nil {
		geoMu.Lock()
		if geoIndex == nil {
			geoIndex = loadHotelGeo()
		}
		idx = geoIndex
		geoMu.Unlock()
	}
	if strings.TrimSpace(name) == "" {
		return HotelGeo{}, false
	}
	h, ok := idx[geoKey(city, name)]
	return h, ok
}

// geoDistanceLabel: "150m" (diukur langsung) atau "±650m" (hitungan peta); "" bila tidak dikenal.
func geoDistanceLabel(city, hotel string) string {
	h, ok := LookupHotelGeo(city, hotel)
	if !ok || h.DistanceM < 0 || (h.DistanceM == 0 && !h.Verified) {
		return ""
	}
	if h.DistanceM == 0 {
		return "0m" // menempel pada pelataran (diukur pemilik situs)
	}
	if h.Verified {
		return formatMeters(h.DistanceM)
	}
	return "±" + formatMeters(h.DistanceM)
}
