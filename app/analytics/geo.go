package analytics

import (
	"log"
	"net/netip"

	"github.com/oschwald/maxminddb-golang/v2"
)

// Lokasi dari database lokal DB-IP Lite (CC BY 4.0, https://db-ip.com). IP hanya dipakai saat lookup,
// yang disimpan cuma kota/provinsi/negara. File diperbarui bulanan oleh skrip cron di server.
var geo *maxminddb.Reader

type geoRecord struct {
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Subdivisions []struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
}

// Nama provinsi DB-IP (Inggris) → nama yang biasa dipakai di Indonesia.
var provinsi = map[string]string{
	"Jakarta": "DKI Jakarta", "West Java": "Jawa Barat", "Central Java": "Jawa Tengah", "East Java": "Jawa Timur",
	"Banten": "Banten", "Yogyakarta": "DI Yogyakarta", "Special Region of Yogyakarta": "DI Yogyakarta",
	"Bali": "Bali", "North Sumatra": "Sumatera Utara", "West Sumatra": "Sumatera Barat", "South Sumatra": "Sumatera Selatan",
	"Riau": "Riau", "Riau Islands": "Kepulauan Riau", "Jambi": "Jambi", "Bengkulu": "Bengkulu", "Lampung": "Lampung",
	"Bangka–Belitung Islands": "Bangka Belitung", "Bangka-Belitung Islands": "Bangka Belitung", "Aceh": "Aceh",
	"West Kalimantan": "Kalimantan Barat", "Central Kalimantan": "Kalimantan Tengah", "South Kalimantan": "Kalimantan Selatan",
	"East Kalimantan": "Kalimantan Timur", "North Kalimantan": "Kalimantan Utara",
	"North Sulawesi": "Sulawesi Utara", "Central Sulawesi": "Sulawesi Tengah", "South Sulawesi": "Sulawesi Selatan",
	"Southeast Sulawesi": "Sulawesi Tenggara", "West Sulawesi": "Sulawesi Barat", "Gorontalo": "Gorontalo",
	"West Nusa Tenggara": "Nusa Tenggara Barat", "East Nusa Tenggara": "Nusa Tenggara Timur",
	"Maluku": "Maluku", "North Maluku": "Maluku Utara", "Papua": "Papua", "West Papua": "Papua Barat",
}

func InitGeo(path string) {
	r, err := maxminddb.Open(path)
	if err != nil {
		log.Println("[analytics] lokasi nonaktif (database GeoIP tidak ada):", err)
		return
	}
	geo = r
	log.Println("[analytics] lokasi aktif:", path)
}

// Locate mengembalikan kota, provinsi, dan negara (nama Indonesia untuk provinsi). Kosong bila tidak diketahui.
func Locate(ip string) (city, region, country string) {
	if geo == nil {
		return
	}
	addr, err := netip.ParseAddr(ip)
	if err != nil || addr.IsPrivate() || addr.IsLoopback() {
		return
	}
	var rec geoRecord
	if err := geo.Lookup(addr.Unmap()).Decode(&rec); err != nil {
		return
	}
	city = rec.City.Names["en"]
	if len(rec.Subdivisions) > 0 {
		region = rec.Subdivisions[0].Names["en"]
		if id, ok := provinsi[region]; ok && rec.Country.ISOCode == "ID" {
			region = id
		}
	}
	country = rec.Country.Names["en"]
	if rec.Country.ISOCode == "ID" {
		country = "Indonesia"
	}
	return truncate(city, 60), truncate(region, 60), truncate(country, 60)
}
