package demo

import (
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"umrah/app/payments"
)

// Fungsi template demo. Selalu didaftarkan (template diparse sekali saat boot), tapi tanpa
// DEMO_MODE semuanya mengembalikan nilai kosong sehingga blok {{if demoOn}} tidak pernah tampil.

type RoleOption struct {
	Value, Label, Hint, Icon, Color string
}

func RoleOptions() []RoleOption {
	opts := []RoleOption{{Value: "jamaah", Label: "Jamaah · Siti Rahmawati", Hint: "Cari paket, cicil, booking + DP, undian", Icon: "🧕", Color: "bg-emerald-100"}}
	for _, t := range PartnerTravels() {
		opts = append(opts, RoleOption{Value: "travel:" + t.Slug, Label: "Travel · " + t.Name, Hint: "Portal: paket, seat, lead, booking, tagihan", Icon: "🧳", Color: "bg-sky-100"})
	}
	opts = append(opts, RoleOption{Value: "admin", Label: "Admin umrohku", Hint: "PPIU, promo, undian, pendapatan", Icon: "🛠️", Color: "bg-gray-100"})
	for _, p := range Cfg.Partners {
		if p.Kind == "pembiayaan" {
			opts = append(opts, RoleOption{Value: "pembiayaan:" + p.Slug, Label: p.Name, Hint: "Proses pengajuan pembiayaan", Icon: "🏦", Color: "bg-amber-100"})
			break // satu mitra cukup untuk pengalih; mitra lain bisa dari /mitra-pembiayaan
		}
	}
	for _, m := range Merchants {
		opts = append(opts, RoleOption{Value: "merchant:" + m.Slug, Label: "Toko · " + m.Name, Hint: "Tukar voucher & serahkan pickup di " + m.City, Icon: "🏪", Color: "bg-rose-100"})
		break
	}
	return opts
}

var statusLabels = map[string]string{
	"menunggu_dp": "Menunggu DP", "dp_lunas": "DP lunas", "lunas": "Lunas", "batal": "Batal", "berangkat": "Berangkat", "pulang": "Sudah pulang",
	"menunggu": "Menunggu bayar", "berhasil": "Berhasil", "kedaluwarsa": "Kedaluwarsa",
	"baru": "Baru", "dihubungi": "Dihubungi", "closing": "Closing", "tidak_valid": "Tidak valid",
	"diajukan": "Diajukan", "diproses": "Diproses", "disetujui": "Disetujui", "ditolak": "Ditolak", "dicairkan": "Dicairkan", "dibatalkan": "Dibatalkan",
	"draft": "Draf", "terbuka": "Terbuka", "ditutup": "Ditutup", "diundi": "Sudah diundi",
	"terverifikasi": "Terverifikasi", "belum": "Belum dicek",
	"dijanjikan": "Dijanjikan", "diterapkan": "Diterapkan",
	"menunggu_bayar": "Menunggu bayar", "dibayar": "Dibayar", "dikemas": "Dikemas", "dikirim": "Dikirim", "diterima": "Diterima",
	"aktif": "Aktif", "habis": "Habis", "siap": "Siap diambil", "diambil": "Sudah diambil", "dijadwalkan": "Dijadwalkan",
	"lunas_tagihan": "Lunas", "belum_bayar": "Belum dibayar",
}

var statusTones = map[string]string{
	"menunggu_dp": "bg-amber-100 text-amber-800", "dp_lunas": "bg-emerald-100 text-emerald-800", "lunas": "bg-emerald-600 text-white",
	"batal": "bg-gray-200 text-gray-600", "berangkat": "bg-sky-100 text-sky-800", "pulang": "bg-violet-100 text-violet-800",
	"menunggu": "bg-amber-100 text-amber-800", "berhasil": "bg-emerald-100 text-emerald-800", "kedaluwarsa": "bg-gray-200 text-gray-600",
	"baru": "bg-sky-100 text-sky-800", "dihubungi": "bg-amber-100 text-amber-800", "closing": "bg-emerald-100 text-emerald-800", "tidak_valid": "bg-gray-200 text-gray-600",
	"diajukan": "bg-sky-100 text-sky-800", "diproses": "bg-amber-100 text-amber-800", "disetujui": "bg-emerald-100 text-emerald-800",
	"ditolak": "bg-rose-100 text-rose-700", "dicairkan": "bg-emerald-600 text-white", "dibatalkan": "bg-gray-200 text-gray-600",
	"draft": "bg-gray-100 text-gray-600", "terbuka": "bg-emerald-100 text-emerald-800", "ditutup": "bg-amber-100 text-amber-800", "diundi": "bg-violet-100 text-violet-800",
	"terverifikasi": "bg-emerald-100 text-emerald-800", "belum": "bg-gray-100 text-gray-600",
	"dijanjikan": "bg-amber-100 text-amber-800", "diterapkan": "bg-emerald-100 text-emerald-800",
	"menunggu_bayar": "bg-amber-100 text-amber-800", "dibayar": "bg-emerald-100 text-emerald-800", "dikemas": "bg-sky-100 text-sky-800",
	"dikirim": "bg-sky-100 text-sky-800", "diterima": "bg-emerald-600 text-white",
	"aktif": "bg-emerald-100 text-emerald-800", "habis": "bg-gray-200 text-gray-600", "siap": "bg-emerald-100 text-emerald-800",
	"diambil": "bg-gray-200 text-gray-600", "dijadwalkan": "bg-sky-100 text-sky-800",
	"lunas_tagihan": "bg-emerald-100 text-emerald-800", "belum_bayar": "bg-amber-100 text-amber-800",
}

func StatusLabel(s string) string {
	if l, ok := statusLabels[s]; ok {
		return l
	}
	return s
}

// Rp: format rupiah "27.500.000".
func Rp(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	out := strings.Join(append([]string{s}, parts...), ".")
	if neg {
		return "-" + out
	}
	return out
}

// RpShort: "500 rb", "3 jt", "27,5 jt".
func RpShort(n int64) string {
	switch {
	case n >= 1_000_000:
		v := float64(n) / 1_000_000
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d jt", int64(v))
		}
		return strings.Replace(fmt.Sprintf("%.1f jt", v), ".", ",", 1)
	case n >= 1000:
		return fmt.Sprintf("%d rb", n/1000)
	}
	return fmt.Sprint(n)
}

var bulan = [...]string{"", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}

// Tgl: "2027-01-15" atau time.Time -> "15 Jan 2027".
func Tgl(v any) string {
	var t time.Time
	switch x := v.(type) {
	case string:
		p, err := time.ParseInLocation("2006-01-02", x, WIB)
		if err != nil {
			return x
		}
		t = p
	case time.Time:
		t = x.In(WIB)
	case *time.Time:
		if x == nil {
			return "-"
		}
		t = x.In(WIB)
	default:
		return ""
	}
	if t.IsZero() {
		return "-"
	}
	return fmt.Sprintf("%d %s %d", t.Day(), bulan[t.Month()], t.Year())
}

func TglJam(v any) string {
	var t time.Time
	switch x := v.(type) {
	case time.Time:
		t = x
	case *time.Time:
		if x == nil {
			return "-"
		}
		t = *x
	default:
		return ""
	}
	t = t.In(WIB)
	return fmt.Sprintf("%d %s %d, %02d.%02d", t.Day(), bulan[t.Month()], t.Year(), t.Hour(), t.Minute())
}

func Ago(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "baru saja"
	case d < time.Hour:
		return fmt.Sprintf("%d menit lalu", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d jam lalu", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%d hari lalu", int(d.Hours()/24))
	}
	return Tgl(t)
}

// MaskName: "Budi Kurniawan" -> "B***i K." (untuk pengumuman pemenang).
func MaskName(name string) string {
	f := strings.Fields(name)
	if len(f) == 0 {
		return "-"
	}
	r := []rune(f[0])
	first := string(r[0]) + "***"
	if len(r) > 1 {
		first += string(r[len(r)-1])
	}
	if len(f) > 1 {
		first += " " + string([]rune(f[len(f)-1])[0]) + "."
	}
	return first
}

func TemplateFuncs() map[string]any {
	return map[string]any{
		"demoOn":    Enabled,
		"demoRoles": func() []RoleOption { if !enabled { return nil }; return RoleOptions() },
		"demoTravel": TravelByName,
		"demoTerms": func(travelName string) *TravelTerms {
			if t := TravelByName(travelName); t != nil && t.Partner {
				return TermsFor(t.Slug)
			}
			return nil
		},
		"demoPortal":  IsPortalPackage,
		"demoDraw":    func() *Draw { if !enabled { return nil }; return OpenDraw() },
		"statusLabel": StatusLabel,
		"statusTone":  func(s string) string { return statusTones[s] },
		"rp":          Rp,
		"rpShort":     RpShort,
		"tgl":         Tgl,
		"tglJam":      TglJam,
		"ago":         Ago,
		"maskName":    MaskName,
		"fakeQR":      func(payload string, size int) template.HTML { return template.HTML(payments.FakeQRSVG(payload, size)) },
		"i64":         func(n int) int64 { return int64(n) },
		"sub64":       func(a, b int64) int64 { return a - b },
		"add64":       func(a, b int64) int64 { return a + b },
		"mul64":       func(a int64, b int) int64 { return a * int64(b) },
		"pct64": func(a, b int64) int {
			if b <= 0 {
				return 0
			}
			return int(min(100, a*100/b))
		},
		"seq": func(n int) []int {
			s := make([]int, n)
			for i := range s {
				s[i] = i + 1
			}
			return s
		},
		"daysUntil": DaysUntil,
		"dpOptions": func(min int64) []int64 { return []int64{min, min * 2, min * 4, min * 10} },
		"atoi64": func(s string) int64 { n, _ := strconv.ParseInt(s, 10, 64); return n },
		"addDays365": func() string { return AddDays(Today(), 365) },
		"addDays730": func() string { return AddDays(Today(), 730) },
		"splitPipe": func(s string) []string { return strings.Split(s, "|") },
		"periodLabel": PeriodLabel,
		"planName":  func(k string) string { return PlanByKey(k).Name },
		"partnerName": func(s string) string {
			if p := PartnerBySlug(s); p != nil {
				return p.Name
			}
			return s
		},
		"travelBySlug": TravelBySlug,
		"pctOf": func(a, b int) int {
			if b <= 0 {
				return 0
			}
			return min(100, a*100/b)
		},
		"pct64of": func(a, b int64) int {
			if b <= 0 {
				return 0
			}
			return int(min(100, a*100/b))
		},
	}
}
