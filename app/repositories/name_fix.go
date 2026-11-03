package repositories

import (
	"regexp"
	"strconv"
	"strings"

	"umrah/app/crawlers"
)

// Perbaikan judul paket yang salah ketik dari travel, berdasarkan tanggal keberangkatan
// yang sebenarnya (tanggal di data travel benar, tapi judulnya tidak ikut diperbarui).
//   - "… Februari 2026" padahal berangkat Februari 2027  -> tahun di judul diganti 2027
//   - "18027 Februari"  padahal 18–27 Februari            -> "18-27 Februari"

var (
	yearRe     = regexp.MustCompile(`\b(20[2-3][0-9])\b`)
	dayRangeRe = regexp.MustCompile(`\b([1-3][0-9])0([0-3][0-9])\b`) // 5 digit: tanggal 2 digit + 0 + tanggal 2 digit (bukan tahun 2026)
)

func fixPackageName(name string, deps []crawlers.CrawledDeparture) string {
	if len(deps) == 0 {
		return name
	}
	years := map[int]bool{}
	for _, d := range deps {
		if len(d.Date) >= 4 {
			if y, err := strconv.Atoi(d.Date[:4]); err == nil {
				years[y] = true
			}
		}
	}
	// Tahun di judul tidak cocok dengan keberangkatan mana pun, dan semua keberangkatan
	// jatuh tepat setahun sesudahnya -> salah ketik tahun.
	name = yearRe.ReplaceAllStringFunc(name, func(s string) string {
		y, _ := strconv.Atoi(s)
		if !years[y] && years[y+1] && len(years) == 1 {
			return strconv.Itoa(y + 1)
		}
		return s
	})
	// "18027" -> "18-27" hanya bila 18 = tanggal berangkat dan 27 = tanggal pulang.
	if len(deps) == 1 && len(deps[0].Date) == 10 && len(deps[0].ReturnDate) == 10 {
		dep, _ := strconv.Atoi(deps[0].Date[8:])
		ret, _ := strconv.Atoi(deps[0].ReturnDate[8:])
		name = dayRangeRe.ReplaceAllStringFunc(name, func(s string) string {
			m := dayRangeRe.FindStringSubmatch(s)
			a, _ := strconv.Atoi(m[1])
			b, _ := strconv.Atoi(m[2])
			if a == dep && b == ret {
				return m[1] + "-" + m[2]
			}
			return s
		})
	}
	return strings.TrimSpace(name)
}
