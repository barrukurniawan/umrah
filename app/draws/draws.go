// Package draws: pengundian yang bisa diaudit siapa pun (PLAN_CICILAN_PROMO_UNDIAN §3.5).
//
//  1. Saat undian ditutup: EntriesHash = sha256(kode-kode entri aktif, diurutkan, dipisah "\n").
//  2. Saat mengundi: seed = sha256(EntriesHash + seed_saksi).
//  3. Setiap entri aktif diberi skor sha256(seed + kode); urutkan naik, ambil N teratas.
//
// Tidak ada unsur acak tersembunyi: dengan daftar kode, seed saksi, dan algoritma ini, hasilnya
// bisa dihitung ulang dan harus sama persis.
package draws

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

type Entry struct {
	Code string
	Void bool
}

func sum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// ActiveCodes: kode entri yang tidak batal, unik, terurut.
func ActiveCodes(entries []Entry) []string {
	seen := map[string]bool{}
	var codes []string
	for _, e := range entries {
		if e.Void || e.Code == "" || seen[e.Code] {
			continue
		}
		seen[e.Code] = true
		codes = append(codes, e.Code)
	}
	sort.Strings(codes)
	return codes
}

// EntriesHash dipublikasikan saat undian ditutup.
func EntriesHash(entries []Entry) string { return sum(strings.Join(ActiveCodes(entries), "\n")) }

// Seed menggabungkan hash entri dengan seed saksi.
func Seed(entriesHash, witnessSeed string) string {
	return sum(entriesHash + strings.TrimSpace(witnessSeed))
}

// Score: makin kecil makin dulu terpilih.
func Score(seed, code string) string { return sum(seed + code) }

// Pick memilih n pemenang (kode) dari entri aktif. Entri Void tidak pernah ikut, dan satu kode
// tidak bisa menang dua kali.
func Pick(entries []Entry, seed string, n int) []string {
	codes := ActiveCodes(entries)
	sort.SliceStable(codes, func(i, j int) bool { return Score(seed, codes[i]) < Score(seed, codes[j]) })
	if n > len(codes) {
		n = len(codes)
	}
	if n < 0 {
		n = 0
	}
	return codes[:n]
}
