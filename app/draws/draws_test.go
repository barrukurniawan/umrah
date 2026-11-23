package draws

import (
	"fmt"
	"testing"
)

func sample() []Entry {
	var e []Entry
	for i := 0; i < 40; i++ {
		e = append(e, Entry{Code: fmt.Sprintf("UMR-%04d", i), Void: i%7 == 0})
	}
	return e
}

func TestDeterministic(t *testing.T) {
	e := sample()
	s := Seed(EntriesHash(e), "48213")
	a, b := Pick(e, s, 3), Pick(e, s, 3)
	if fmt.Sprint(a) != fmt.Sprint(b) || len(a) != 3 {
		t.Fatalf("hasil tidak deterministik: %v vs %v", a, b)
	}
	// Urutan input tidak memengaruhi hasil.
	rev := make([]Entry, len(e))
	for i := range e {
		rev[len(e)-1-i] = e[i]
	}
	if fmt.Sprint(Pick(rev, s, 3)) != fmt.Sprint(a) {
		t.Fatal("urutan entri memengaruhi hasil")
	}
	if EntriesHash(rev) != EntriesHash(e) {
		t.Fatal("hash entri bergantung urutan")
	}
}

func TestVoidNeverWins(t *testing.T) {
	e := sample()
	for w := 0; w < 50; w++ {
		s := Seed(EntriesHash(e), fmt.Sprint(w))
		for _, c := range Pick(e, s, 40) {
			for _, x := range e {
				if x.Code == c && x.Void {
					t.Fatalf("entri batal %s menang", c)
				}
			}
		}
	}
}

func TestNoDoubleWin(t *testing.T) {
	e := append(sample(), Entry{Code: "UMR-0001"}, Entry{Code: "UMR-0001"}) // duplikat
	got := Pick(e, Seed(EntriesHash(e), "x"), 100)
	seen := map[string]bool{}
	for _, c := range got {
		if seen[c] {
			t.Fatalf("kode %s menang dua kali", c)
		}
		seen[c] = true
	}
}

func TestWitnessChangesResult(t *testing.T) {
	e := sample()
	h := EntriesHash(e)
	if fmt.Sprint(Pick(e, Seed(h, "111"), 3)) == fmt.Sprint(Pick(e, Seed(h, "222"), 3)) {
		t.Fatal("seed saksi berbeda seharusnya (hampir pasti) memberi hasil berbeda")
	}
}
