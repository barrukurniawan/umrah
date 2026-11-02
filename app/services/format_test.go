package services

import "testing"

func TestShortJuta(t *testing.T) {
	cases := map[int]string{28990000: "28,99 jt", 23500000: "23,5 jt", 29995000: "29,995 jt", 30000000: "30 jt", 26900000: "26,9 jt"}
	for in, want := range cases {
		if got := ShortJuta(in); got != want {
			t.Errorf("ShortJuta(%d) = %q, want %q", in, got, want)
		}
	}
}
