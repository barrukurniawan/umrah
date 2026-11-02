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
		{"makkah", "Al Olayan Golden", "800m"},
		{"makkah", "AL MASSA DAR AL FAYZEEN", "±650m"}, // hitungan peta
		{"makkah", "Fajr Al-Nusuk Ajyad (Makkah)", "550m"}, // Google: "Luluat Al Shark"
	}
	for _, c := range cases {
		if got := geoDistanceLabel(c.city, c.name); got != c.want {
			t.Errorf("%s %q = %q, want %q", c.city, c.name, got, c.want)
		}
	}
	if _, ok := LookupHotelGeo("makkah", "Zad Al Qimmah"); ok {
		t.Error("Zad Al Qimmah belum terverifikasi cabangnya, harus tidak dikenal")
	}
	if u := GmapsMadinahURL("Jawharat Al Rasheed"); !strings.Contains(u, "origin=24.") || !strings.Contains(u, "destination=24.") {
		t.Errorf("rute Madinah harus dari koordinat hotel ke koordinat gerbang: %s", u)
	}
}
