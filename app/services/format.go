package services

import (
	"strconv"
	"strings"
)

// ShortJuta menulis harga dalam juta TANPA pembulatan: 28990000 -> "28,99 jt",
// 23500000 -> "23,5 jt", 29995000 -> "29,995 jt". Pembulatan ke atas/bawah bisa
// menyesatkan calon jamaah, jadi angka selalu sama persis dengan harga asli.
func ShortJuta(n int) string {
	v := strconv.FormatFloat(float64(n)/1e6, 'f', 6, 64)
	v = strings.TrimRight(strings.TrimRight(v, "0"), ".")
	return strings.Replace(v, ".", ",", 1) + " jt"
}
