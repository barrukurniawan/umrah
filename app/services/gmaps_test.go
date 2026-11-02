package services

import (
	"net/url"
	"strings"
	"testing"
)

func TestCleanHotelName(t *testing.T) {
	cases := map[string]string{
		"⭐ ⭐ Le Meridien Tower":               "Le Meridien Tower",
		"Anjum / Setaraf":                     "Anjum",
		"Fajr Al-Nusuk Ajyad (Makkah)":        "Fajr Al-Nusuk Ajyad",
		"Fajar Badie 4/fajr Badie 5/ Setaraf": "Fajar Badie 4",
		"PRESTIGE EX ELAF AL MASHAER/SETARAF": "PRESTIGE AL MASHAER",
		"MAYSAN AL MAQAM EX FAJAR BADEA 2":    "MAYSAN AL MAQAM",
		"Le Meridien Tower *4":                "Le Meridien Tower",
		"Dar Al Naeem *3":                     "Dar Al Naeem",
		"️ Mariott Jabal Omar":                "Mariott Jabal Omar",
		"":                                    "",
		"Setaraf":                             "",
	}
	for in, want := range cases {
		if got := CleanHotelName(in); got != want {
			t.Errorf("CleanHotelName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGmapsMakkahURL(t *testing.T) {
	if GmapsMakkahURL("") != "" || GmapsMakkahURL(" / Setaraf") != "" {
		t.Fatal("hotel kosong harus menghasilkan URL kosong (bukan #)")
	}
	u, err := url.Parse(GmapsMakkahURL("Zowar Al Bait"))
	if err != nil || u.Host != "www.google.com" {
		t.Fatalf("bad url %v %v", u, err)
	}
	q := u.Query()
	if q.Get("origin") != "Zowar Al Bait Hotel, Makkah" || !strings.Contains(q.Get("destination"), "King Abdul Aziz Gate") || q.Get("travelmode") != "walking" {
		t.Errorf("query salah: %v", q)
	}
	if d := MakkahGate("Marriot Jabal Omar"); !strings.Contains(d, "King Fahd Gate") {
		t.Errorf("Jabal Omar harus ke King Fahd Gate, dapat %s", d)
	}
}

func TestGmapsMakkahURLKnownPlace(t *testing.T) {
	u, _ := url.Parse(GmapsMakkahURL("Al Olayan Golden"))
	q := u.Query()
	if q.Get("origin") != "21.4138065,39.8222496" || q.Get("destination") != "CRCG+2G8 Makkah" {
		t.Errorf("Olayan Golden harus pakai titik pasti, dapat %v", q)
	}
}
