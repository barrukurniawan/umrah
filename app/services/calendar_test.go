package services

import (
	"testing"
	"time"
)

func TestDailyTempTone(t *testing.T) {
	cases := map[string]string{
		"2027-01-15": "sejuk",  // Makkah 31°
		"2026-11-15": "nyaman", // 35°
		"2027-04-15": "hangat", // 39°
		"2027-05-15": "panas",  // 43°
		"2027-06-15": "panas",  // 45°, "sangat panas" digabung ke panas
	}
	for d, want := range cases {
		tm, _ := time.Parse("2006-01-02", d)
		if got := dailyTemp(tm); got.Tone != want {
			t.Errorf("%s: tone %s (hi %d), want %s", d, got.Tone, got.MakHi, want)
		}
	}
	// Interpolasi: 1 Des berada di antara 15 Nov (35°) dan 15 Des (32°).
	tm, _ := time.Parse("2006-01-02", "2026-12-01")
	if hi := dailyTemp(tm).MakHi; hi < 32 || hi > 35 {
		t.Errorf("1 Des hi = %d, harus di antara 32 dan 35", hi)
	}
}

func TestCalEvents(t *testing.T) {
	evs, haji := calEvents("2027-03-01")
	kinds := map[string]bool{}
	for _, e := range evs {
		kinds[e.Kind] = true
	}
	if !kinds["ramadhan"] || !kinds["puncak"] || haji {
		t.Errorf("1 Mar 2027 harus Ramadhan + 10 malam terakhir, dapat %+v haji=%v", evs, haji)
	}
	if _, haji := calEvents("2027-04-20"); !haji {
		t.Error("20 Apr 2027 harus musim haji")
	}
	// Umm al-Qura: 1 Syawal 1448 H = 9 Mar 2027, jadi 9 Mar bukan Ramadhan lagi.
	if evs, _ := calEvents("2027-03-09"); len(evs) != 1 || evs[0].Kind != "ied" {
		t.Errorf("9 Mar 2027 harus hanya Idul Fitri, dapat %+v", evs)
	}
}
