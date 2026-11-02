package services

import (
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	os.Setenv("HOTEL_GEO", "../../config/hotel_geo.json")
	os.Exit(m.Run())
}

func TestHotelGeo(t *testing.T) {
	cases := []struct{ city, name, want string }{
		{"madinah", "Jawharat Al Rasheed (Madinah)", "150m"}, // diukur pemilik situs, tanpa ±
		{"madinah", "Jauharat Al Rashed", "150m"},            // variasi ejaan
		{"madinah", "Al-Sham Province", "350m"},
		{"makkah", "Al Olayan Golden", "700m"},
		{"makkah", "AL MASSA DAR AL FAYZEEN", "±650m"},     // hitungan peta
		{"makkah", "Fajr Al-Nusuk Ajyad (Makkah)", "550m"}, // Google: "Luluat Al Shark"
		{"makkah", "Manazel Ajyad/Tara Ajyad", "±1,4 km"},
		{"makkah", "Villa Hilton", "0m"},
		{"makkah", "Owais", "500m"}, // flyer Umrah Bisa: jalan kaki ±500 m
		{"makkah", "Makkah Tower/Al Shafwa", "0m"},
		{"makkah", "Hotel Pullman Zamzam Makkah", "0m"},
		{"makkah", "Zad Al Qimmah", "6,3 km"},
		{"makkah", "Holiday Inn", "3,5 km"},
	}
	for _, c := range cases {
		if got := geoDistanceLabel(c.city, c.name); got != c.want {
			t.Errorf("%s %q = %q, want %q", c.city, c.name, got, c.want)
		}
	}
	if u := GmapsMadinahURL("Jawharat Al Rasheed"); !strings.Contains(u, "origin=24.") || !strings.Contains(u, "destination=24.") {
		t.Errorf("rute Madinah harus dari koordinat hotel ke koordinat gerbang: %s", u)
	}
}
