package repositories

import (
	"testing"

	"umrah/app/crawlers"
)

func TestDurationFromName(t *testing.T) {
	cases := map[string]int{
		"Umroh 9D By Garuda":                   9,
		"Umroh Plus Thaif 12 D By Qatar":       12,
		"UMROH PLUS THAIF 9 HARI":              9,
		"Umrah Full Ramadhan 33 Hari":          33,
		"Program Madinah 16-21 Hari":           0, // rentang: tidak pasti
		"Umrah Haqqan Saudia | Hotel Fajr":     0,
		"Umroh 2x Jumat + Thoif 12D By IndiGo": 12,
	}
	for in, want := range cases {
		if got := durationFromName(in); got != want {
			t.Errorf("durationFromName(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestNormalizeDeparturesDedupe(t *testing.T) {
	cp := crawlers.CrawledPackage{Departures: []crawlers.CrawledDeparture{
		{Date: "2026-12-11"},
		{Date: "2026-12-07", ReturnDate: "2026-12-15", Seats: 5},
		{Date: "2026-12-07", HotelMakkah: "Owais"},
		{Date: "11 Des 2026"},                          // format lain, tanggal sama
		{Date: "2026-12-20", ReturnDate: "2026-12-20"}, // pulang = berangkat: dibuang
	}}
	got := normalizeDepartures(cp)
	if len(got) != 3 {
		t.Fatalf("want 3 unique departures, got %d: %+v", len(got), got)
	}
	if got[0].Date != "2026-12-07" || got[0].Seats != 5 || got[0].HotelMakkah != "Owais" || got[0].ReturnDate != "2026-12-15" {
		t.Errorf("merge salah: %+v", got[0])
	}
	if got[2].ReturnDate != "" {
		t.Errorf("return date = departure harus dibuang, got %q", got[2].ReturnDate)
	}
}

func TestHotelKey(t *testing.T) {
	same := [][2]string{
		{"Jawharat Al Rasheed (Madinah)", "Jawharat Rasheed"},
		{"Olayan Golden Hotel", "OLAYAN GOLDEN"},
		{"Anjum / Setaraf", "Anjum Hotel Makkah"},
	}
	for _, p := range same {
		if hotelKey(p[0]) != hotelKey(p[1]) {
			t.Errorf("%q (%q) != %q (%q)", p[0], hotelKey(p[0]), p[1], hotelKey(p[1]))
		}
	}
	if hotelKey("Jawharat Al Rasheed") == hotelKey("Jawharat Ar Rashed") {
		t.Log("ejaan berbeda tidak dicocokkan (sengaja konservatif)")
	}
}
