package repositories

import (
	"testing"

	"umrah/app/crawlers"
)

func TestFixPackageName(t *testing.T) {
	d := func(a, b string) []crawlers.CrawledDeparture { return []crawlers.CrawledDeparture{{Date: a, ReturnDate: b}} }
	cases := []struct {
		name string
		deps []crawlers.CrawledDeparture
		want string
	}{
		{"Umrah Tengah Ramadhan 10 Hari 18027 Februari 2026", d("2027-02-18", "2027-02-27"), "Umrah Tengah Ramadhan 10 Hari 18-27 Februari 2027"},
		{"Umrah Full Ramadhan 33 Hari 8 Feb-12 Maret 2026", d("2027-02-08", "2027-03-12"), "Umrah Full Ramadhan 33 Hari 8 Feb-12 Maret 2027"},
		{"Umrah Desember 2026", d("2026-12-20", ""), "Umrah Desember 2026"},
		{"Umrah Liburan 26 Des 2026-3 Jan 2027", d("2026-12-26", "2027-01-03"), "Umrah Liburan 26 Des 2026-3 Jan 2027"},
		{"Paket 2026 Promo", []crawlers.CrawledDeparture{{Date: "2026-12-01"}, {Date: "2027-01-10"}}, "Paket 2026 Promo"},
		{"Umroh Plus City Tour Oman & Thaif (Promo, normal Rp31,5 jt, s.d. 5 Des 2026)", []crawlers.CrawledDeparture{{Date: "2027-01-14"}, {Date: "2027-02-10"}}, "Umroh Plus City Tour Oman & Thaif (Promo, normal Rp31,5 jt, s.d. 5 Des 2026)"},
		{"Umrah 12 Hari 1009 Januari", d("2027-01-10", "2027-01-21"), "Umrah 12 Hari 1009 Januari"},
		{"Umrah Ceria Januari 2026", []crawlers.CrawledDeparture{{Date: "2027-01-02"}, {Date: "2027-01-03"}}, "Umrah Ceria Januari 2027"},
		{"Umroh 2 Desember 2026", d("2026-12-02", "2026-12-26"), "Umroh 2 Desember 2026"},
	}
	for _, c := range cases {
		if got := fixPackageName(c.name, c.deps); got != c.want {
			t.Errorf("%q -> %q, want %q", c.name, got, c.want)
		}
	}
}
