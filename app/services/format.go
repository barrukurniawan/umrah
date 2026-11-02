package services

import "strconv"

// ShortJuta menulis harga dalam juta dengan SATU angka di belakang koma, dipotong (tidak
// dibulatkan ke atas, supaya "29 jt" tidak muncul untuk 28.990.000): 28990000 -> "28,9 jt",
// 23500000 -> "23,5 jt", 30000000 -> "30 jt". Harga lengkap tetap tampil di kartu.
func ShortJuta(n int) string {
	tenths := n / 100000 // satuan 0,1 juta, dipotong
	whole, frac := tenths/10, tenths%10
	if frac == 0 {
		return strconv.Itoa(whole) + " jt"
	}
	return strconv.Itoa(whole) + "," + strconv.Itoa(frac) + " jt"
}
