package services

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// Gerbang Masjidil Haram yang dipakai sebagai tujuan rute jalan kaki dari hotel.
// Tujuan sengaja gerbang/pelataran, bukan Ka'bah: jarak yang dirasakan jamaah
// adalah jalan kaki sampai pintu masjid.
const (
	gateKingAbdulAziz = "King Abdul Aziz Gate, Masjid al-Haram, Makkah"
	gateKingFahd      = "King Fahd Gate, Masjid al-Haram, Makkah"
	nabawiCourtyard   = "Al-Masjid an-Nabawi, Madinah"
)

// Hotel di sisi barat (Jabal Omar, Anjum) lebih dekat ke King Fahd Gate. Hotel lain
// (Abraj Al Bait, Ibrahim Khalil, Misfalah, Ajyad) ada di sisi selatan dan paling
// dekat ke King Abdul Aziz Gate, yang jadi default.
var westSideHotels = []string{"jabal omar", "marriott", "mariott", "jumeirah", "jumairah", "hilton suites", "hilton convention", "conrad", "hyatt", "anjum", "address"}

var (
	hotelParenRe = regexp.MustCompile(`\([^)]*\)`)
	hotelStarsRe = regexp.MustCompile(`\*\s*\d|\d\s*\*`)
	hotelExRe    = regexp.MustCompile(`(?i)\s+ex\s+`)
	hotelSetRe   = regexp.MustCompile(`(?i)\bsetaraf\b`)
)

// CleanHotelName merapikan nama hotel hasil crawler untuk pencarian Google Maps:
// buang emoji/bintang, "(Makkah)", "/ Setaraf", dan ambil opsi pertama jika ada
// beberapa hotel ("Fajar Badie 4/fajr Badie 5/ Setaraf" -> "Fajar Badie 4").
// Nama lama "X EX Y" diambil X, kecuali X cuma satu kata ("Prestige EX Elaf Al
// Mashaer" -> "Prestige Al Mashaer").
func CleanHotelName(s string) string {
	s = hotelParenRe.ReplaceAllString(s, " ")
	s = hotelStarsRe.ReplaceAllString(s, " ")
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '/' || r == '-' || r == '\'' || r == '&' {
			return r
		}
		return ' '
	}, s)
	for _, part := range strings.Split(s, "/") {
		part = strings.TrimSpace(hotelSetRe.ReplaceAllString(part, " "))
		if part == "" {
			continue
		}
		if loc := hotelExRe.FindStringIndex(part); loc != nil {
			head, old := strings.Fields(part[:loc[0]]), strings.Fields(part[loc[1]:])
			if len(head) == 1 && len(old) > 1 {
				head = append(head, old[1:]...)
			}
			part = strings.Join(head, " ")
		}
		return strings.Join(strings.Fields(part), " ")
	}
	return ""
}

func hotelQuery(hotel, city string) string {
	if !strings.Contains(strings.ToLower(hotel), "hotel") {
		hotel += " Hotel"
	}
	return hotel + ", " + city
}

func directionsURL(origin, destination string) string {
	q := url.Values{}
	q.Set("api", "1")
	q.Set("origin", origin)
	q.Set("destination", destination)
	q.Set("travelmode", "walking")
	return "https://www.google.com/maps/dir/?" + q.Encode()
}

// MakkahGate memilih gerbang Masjidil Haram terdekat untuk hotel (perkiraan per sisi).
func MakkahGate(hotel string) string {
	h := strings.ToLower(hotel)
	for _, k := range westSideHotels {
		if strings.Contains(h, k) {
			return gateKingFahd
		}
	}
	return gateKingAbdulAziz
}

// Titik pasti untuk hotel yang pencarian namanya di Google Maps bisa meleset, plus tujuan
// khusus bila diminta pemilik situs. Kunci = potongan nama hotel (huruf kecil).
type hotelPlace struct{ Origin, Dest string }

var makkahPlaces = map[string]hotelPlace{
	// Al Olayan Golden Hotel (maps.app.goo.gl/HPE7nQ8BVp3S15S96), rute ke WC 3 Masjidil Haram.
	"olayan golden": {Origin: "21.4138065,39.8222496", Dest: "CRCG+2G8 Makkah"},
}

// GmapsMakkahURL: rute jalan kaki dari hotel ke gerbang Masjidil Haram.
// Kosong jika nama hotel tidak diketahui, supaya template tidak membuat link.
func GmapsMakkahURL(hotel string) string {
	h := CleanHotelName(hotel)
	if h == "" {
		return ""
	}
	lh := strings.ToLower(h)
	for k, p := range makkahPlaces {
		if strings.Contains(lh, k) {
			return directionsURL(p.Origin, p.Dest)
		}
	}
	if g, ok := LookupHotelGeo("makkah", hotel); ok && g.GateLat != 0 {
		return directionsURL(latLng(g.Lat, g.Lng), latLng(g.GateLat, g.GateLng)) // ke gerbang terdekat
	}
	return directionsURL(hotelQuery(h, "Makkah"), MakkahGate(h))
}

func latLng(lat, lng float64) string { return fmt.Sprintf("%.7f,%.7f", lat, lng) }

// GmapsMadinahURL: rute jalan kaki dari hotel ke pelataran Masjid Nabawi.
func GmapsMadinahURL(hotel string) string {
	h := CleanHotelName(hotel)
	if h == "" {
		return ""
	}
	if g, ok := LookupHotelGeo("madinah", hotel); ok && g.GateLat != 0 {
		return directionsURL(latLng(g.Lat, g.Lng), latLng(g.GateLat, g.GateLng)) // ke gerbang pelataran terdekat
	}
	return directionsURL(hotelQuery(h, "Madinah"), nabawiCourtyard)
}
