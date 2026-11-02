package services

import (
	"html/template"
	"strconv"
	"strings"
)

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

// planeSVG: ikon pesawat garis (sama dengan ikon maskapai di kartu), pengganti emoji ✈️.
const planeSVG = `<svg class="inline-block w-3.5 h-3.5 -mt-0.5 align-middle" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-label="pesawat"><path d="M17.8 19.2 16 11l3.5-3.5C21 6 21.5 4 21 3c-1-.5-3 0-4.5 1.5L13 8 4.8 6.2c-.5-.1-.9.1-1.1.5l-.3.5c-.2.5-.1 1 .3 1.3L9 12l-2 3H4l-1 1 3 2 2 3 1-1v-3l3-2 3.5 5.3c.3.4.8.5 1.3.3l.5-.2c.4-.3.6-.7.5-1.2z"/></svg>`

func isEmoji(r rune) bool {
	return (r >= 0x1F000 && r <= 0x1FAFF) || (r >= 0x2600 && r <= 0x27BF) || r == 0xFE0F || r == 0x200D || r == 0x2B50 || r == 0x2B55
}

// StripEmoji membuang emoji berwarna dari teks travel (mis. fasilitas "Makan enak 🤤").
func StripEmoji(s string) string {
	out := strings.Map(func(r rune) rune {
		if isEmoji(r) {
			return -1
		}
		return r
	}, s)
	return strings.Join(strings.Fields(out), " ")
}

// PkgNameHTML: nama paket dengan ✈️ diganti ikon pesawat bertema; emoji lain dibuang.
func PkgNameHTML(s string) template.HTML {
	parts := strings.Split(strings.ReplaceAll(s, "✈️", "✈"), "✈")
	for i, p := range parts {
		parts[i] = template.HTMLEscapeString(StripEmoji(p))
	}
	return template.HTML(strings.Join(parts, " "+planeSVG+" "))
}

// TravelDisplayName: nama travel untuk judul, tanpa badan usaha ("PT Labbaika Cipta Imani"
// -> "Labbaika Cipta Imani"). Nama asli tetap dipakai untuk data, filter, dan analitik.
func TravelDisplayName(name string) string {
	n := strings.TrimSpace(name)
	for _, p := range []string{"PT. ", "PT ", "CV. ", "CV "} {
		if len(n) > len(p) && strings.EqualFold(n[:len(p)], p) {
			n = strings.TrimSpace(n[len(p):])
		}
	}
	return n
}

// TravelSlug: "UMI Tour & Travel" -> "umi-tour-travel" (nama file logo).
func TravelSlug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(TravelDisplayName(name)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

// TravelInitials: "Labbaika Cipta Imani" -> "LC" (cadangan bila logo belum ada).
func TravelInitials(name string) string {
	var out []rune
	for _, w := range strings.Fields(TravelDisplayName(name)) {
		r := []rune(w)
		if len(r) > 0 && ((r[0] >= 'A' && r[0] <= 'Z') || (r[0] >= 'a' && r[0] <= 'z')) {
			out = append(out, []rune(strings.ToUpper(string(r[0])))...)
		}
		if len(out) == 2 {
			break
		}
	}
	return string(out)
}
