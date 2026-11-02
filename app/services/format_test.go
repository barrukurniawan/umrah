package services

import (
	"strings"
	"testing"
)

func TestShortJuta(t *testing.T) {
	cases := map[int]string{28990000: "28,9 jt", 27988000: "27,9 jt", 23500000: "23,5 jt", 29995000: "29,9 jt", 30000000: "30 jt", 26900000: "26,9 jt", 30050000: "30 jt"}
	for in, want := range cases {
		if got := ShortJuta(in); got != want {
			t.Errorf("ShortJuta(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestPkgNameHTML(t *testing.T) {
	got := string(PkgNameHTML("Umrah 9 Hari Haqqan ✈️ Qatar | Zam-Zam <10> Liter"))
	if strings.Contains(got, "✈") || !strings.Contains(got, "<svg") || !strings.Contains(got, "&lt;10&gt;") {
		t.Errorf("PkgNameHTML: %s", got)
	}
	if StripEmoji("Makan enak 🤤 sekali") != "Makan enak sekali" {
		t.Error("StripEmoji")
	}
}
