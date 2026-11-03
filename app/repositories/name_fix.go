package repositories

import (
	"fmt"
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
	monthYearRe = regexp.MustCompile(`(?i)\b(jan(?:uari)?|feb(?:ruari)?|mar(?:et)?|apr(?:il)?|mei|jun(?:i)?|jul(?:i)?|agu(?:stus)?|agt|sep(?:tember)?|okt(?:ober)?|nov(?:ember)?|des(?:ember)?)\s+(20[2-3][0-9])\b`)
	dayRangeRe  = regexp.MustCompile(`\b([1-3][0-9])0([0-3][0-9])\b`) // 5 digit: tanggal 2 digit + 0 + tanggal 2 digit (bukan tahun 2026)
)

func fixPackageName(name string, deps []crawlers.CrawledDeparture) string {
	if len(deps) == 0 {
		return name
	}
	// Bulan+tahun keberangkatan/pulang yang sebenarnya, mis. "2027-02".
	have := map[string]bool{}
	for _, d := range deps {
		for _, x := range []string{d.Date, d.ReturnDate} {
			if len(x) >= 7 {
				have[x[:7]] = true
			}
		}
	}
	// "Februari 2026" padahal tidak ada jadwal Feb 2026 tapi ada Feb 2027 -> salah ketik tahun.
	// Tahun yang tidak menempel pada nama bulan (mis. batas promo "s.d. 5 Des 2026") tidak diubah
	// kecuali bulannya juga bulan keberangkatan.
	name = monthYearRe.ReplaceAllStringFunc(name, func(s string) string {
		m := monthYearRe.FindStringSubmatch(s)
		mo := monthNum(m[1])
		y, _ := strconv.Atoi(m[2])
		if mo == 0 {
			return s
		}
		cur, next := fmt.Sprintf("%d-%02d", y, mo), fmt.Sprintf("%d-%02d", y+1, mo)
		if !have[cur] && have[next] {
			return m[1] + " " + strconv.Itoa(y+1)
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

func monthNum(s string) int {
	switch strings.ToLower(s)[:3] {
	case "jan":
		return 1
	case "feb":
		return 2
	case "mar":
		return 3
	case "apr":
		return 4
	case "mei":
		return 5
	case "jun":
		return 6
	case "jul":
		return 7
	case "agu", "agt":
		return 8
	case "sep":
		return 9
	case "okt":
		return 10
	case "nov":
		return 11
	case "des":
		return 12
	}
	return 0
}
