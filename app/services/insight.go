package services

import (
	"fmt"
	"time"
)

// Insight perjalanan berdasarkan bulan keberangkatan: iklim rata-rata Makkah & Madinah (klimatologi
// jangka panjang, bukan prakiraan), musim, keramaian, momen penting (perkiraan kalender Umm al-Qura),
// dan tips ibadah. Semua statis, tanpa layanan luar.

type climate struct{ MakHi, MakLo, MadHi, MadLo int }

// Rata-rata suhu maksimum/minimum harian (°C) per bulan, dibulatkan.
var climates = [13]climate{
	{},
	{31, 19, 24, 12}, {32, 19, 26, 14}, {35, 21, 30, 17}, {39, 25, 35, 22},
	{43, 28, 40, 26}, {45, 29, 43, 29}, {44, 30, 43, 29}, {43, 30, 43, 30},
	{43, 29, 42, 28}, {39, 26, 37, 23}, {35, 23, 30, 18}, {32, 21, 26, 14},
}

type monthInfo struct {
	Season, Tone string // tone: sejuk | peralihan | hangat | panas | ekstrem
	Crowd        int    // 1 sepi, 2 sedang, 3 ramai (di luar momen khusus)
	CrowdNote    string
	Rain         string
}

var months = [13]monthInfo{
	{},
	{"musim dingin yang sejuk", "sejuk", 3, "masih ramai jamaah Indonesia karena libur awal tahun dan cuaca yang nyaman", "sesekali hujan ringan"},
	{"musim dingin yang sejuk", "sejuk", 2, "cukup ramai menjelang Ramadhan", "jarang hujan"},
	{"peralihan menuju hangat", "peralihan", 2, "ramainya bergantung pada Ramadhan dan Idul Fitri", "hampir tidak ada hujan"},
	{"hangat menuju musim panas", "hangat", 2, "relatif sedang setelah Lebaran", "kering"},
	{"musim panas", "panas", 1, "umumnya sepi karena mendekati musim haji", "kering"},
	{"puncak musim panas", "ekstrem", 1, "sepi untuk umroh karena berdekatan dengan musim haji", "kering"},
	{"puncak musim panas", "ekstrem", 2, "mulai ramai lagi setelah haji, bertepatan libur sekolah Indonesia", "kering"},
	{"puncak musim panas", "ekstrem", 1, "termasuk low season: lengang dan harga paket biasanya lebih murah", "kering"},
	{"musim panas", "panas", 1, "masih low season, cocok untuk yang mencari harga hemat", "kering"},
	{"akhir musim panas", "hangat", 2, "mulai ramai, awal musim umroh", "kering"},
	{"awal musim sejuk", "peralihan", 2, "ramai seiring dimulainya musim sejuk; puncaknya biasanya Desember", "mulai ada peluang hujan ringan"},
	{"musim dingin yang sejuk", "sejuk", 3, "high season: libur akhir tahun membuat jamaah Indonesia sangat ramai dan harga cenderung tertinggi", "peluang hujan ringan, kadang deras sesaat"},
}

var monthNames = [13]string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

// Perkiraan momen Hijriah (±1 hari, tergantung rukyat). Rentang inklusif "2006-01-02".
type event struct{ From, To, Label, Kind string }

var events = []event{
	{"2026-02-18", "2026-03-19", "Ramadhan 1447 H", "ramadhan"},
	{"2026-03-09", "2026-03-19", "10 malam terakhir Ramadhan 1447 H (puncak keramaian)", "puncak"},
	{"2026-04-25", "2026-06-05", "Musim haji 1447 H: visa umroh biasanya dihentikan sementara", "haji"},
	{"2027-02-08", "2027-03-09", "Ramadhan 1448 H", "ramadhan"},
	{"2027-02-27", "2027-03-09", "10 malam terakhir Ramadhan 1448 H (puncak keramaian)", "puncak"},
	{"2027-04-14", "2027-05-26", "Musim haji 1448 H: visa umroh biasanya dihentikan sementara", "haji"},
	{"2028-01-28", "2028-02-26", "Ramadhan 1449 H", "ramadhan"},
	{"2028-02-16", "2028-02-26", "10 malam terakhir Ramadhan 1449 H (puncak keramaian)", "puncak"},
	{"2028-04-03", "2028-05-14", "Musim haji 1449 H: visa umroh biasanya dihentikan sementara", "haji"},
	// Libur sekolah Indonesia (perkiraan) membuat jamaah keluarga ramai.
	{"2026-12-19", "2027-01-03", "Libur sekolah akhir semester Indonesia", "libur"},
	{"2027-06-26", "2027-07-11", "Libur kenaikan kelas Indonesia", "libur"},
	{"2027-12-18", "2028-01-02", "Libur sekolah akhir semester Indonesia", "libur"},
}

type Insight struct {
	Key, Travel, Package string
	DateLabel            string // keberangkatan pertama, "2 Des 2026"
	OtherDates           int    // jumlah tanggal lain
	Month, Year          string
	Season, Tone         string
	MakHi, MakLo         int
	MadHi, MadLo         int
	Rain                 string
	CrowdLevel           int
	CrowdLabel           string
	Summary              string
	Events               []string
	Tips                 []string
}

func shortDate(t time.Time) string {
	return fmt.Sprintf("%d %s %d", t.Day(), monthNames[t.Month()][:3], t.Year())
}

// BuildInsight menyusun insight dari tanggal keberangkatan pertama yang belum lewat.
func BuildInsight(cp *CurrentPackage) *Insight {
	if cp == nil || len(cp.Departures) == 0 {
		return nil
	}
	dep, err := time.Parse("2006-01-02", cp.Departures[0])
	if err != nil {
		return nil
	}
	m := int(dep.Month())
	c, mi := climates[m], months[m]
	in := &Insight{
		Key: cp.Key, Travel: cp.Travel, Package: cp.Name, DateLabel: shortDate(dep), OtherDates: len(cp.Departures) - 1,
		Month: monthNames[m], Year: fmt.Sprint(dep.Year()), Season: mi.Season, Tone: mi.Tone,
		MakHi: c.MakHi, MakLo: c.MakLo, MadHi: c.MadHi, MadLo: c.MadLo, Rain: mi.Rain, CrowdLevel: mi.Crowd,
	}

	// Momen yang beririsan dengan perjalanan (tanggal berangkat s.d. + durasi).
	days := max(cp.Duration, 9)
	tripEnd := dep.AddDate(0, 0, days-1)
	crowdNote := mi.CrowdNote
	ramadhan := false
	for _, e := range events {
		from, err1 := time.Parse("2006-01-02", e.From)
		to, err2 := time.Parse("2006-01-02", e.To)
		if err1 != nil || err2 != nil {
			continue
		}
		// Jumlah hari perjalanan yang jatuh di dalam momen ini.
		overlap := int(minTime(tripEnd, to).Sub(maxTime(dep, from)).Hours()/24) + 1
		if overlap <= 0 {
			continue
		}
		switch e.Kind {
		case "ramadhan":
			ramadhan = true
			in.CrowdLevel = 3
			in.Events = append(in.Events, e.Label)
			crowdNote = "bertepatan dengan Ramadhan, salah satu waktu paling ramai dan paling dicari; umroh di bulan Ramadhan pahalanya setara haji (HR. Bukhari & Muslim)"
		case "puncak":
			if overlap < 3 { // hanya menyerempet, jangan dilebih-lebihkan
				continue
			}
			in.CrowdLevel = 3
			in.Events = append(in.Events, e.Label)
			crowdNote = fmt.Sprintf("masuk %d hari di 10 malam terakhir Ramadhan, puncak keramaian sepanjang tahun", min(overlap, 10))
		case "haji":
			in.Events = append(in.Events, e.Label)
			crowdNote = "berdekatan dengan musim haji; pastikan ke travel bahwa visa umroh masih berlaku untuk tanggal ini"
		case "libur":
			in.Events = append(in.Events, e.Label)
			in.CrowdLevel = 3
		}
	}
	in.CrowdLabel = map[int]string{1: "Sepi (low season)", 2: "Sedang", 3: "Ramai (high season)"}[in.CrowdLevel]

	in.Summary = fmt.Sprintf("Paket %s di bulan %s %s yang kamu simpan masuk %s di Arab Saudi. "+
		"Rata-rata suhu di Makkah sekitar %d°C saat siang dan %d°C saat malam; di Madinah %d°C siang dan %d°C malam. "+
		"Keramaian bulan ini %s.",
		cp.Travel, in.Month, in.Year, mi.Season, c.MakHi, c.MakLo, c.MadHi, c.MadLo, crowdNote)

	// Tips sesuai suhu & keramaian.
	switch mi.Tone {
	case "sejuk":
		in.Tips = append(in.Tips,
			fmt.Sprintf("Malam di Madinah bisa turun sampai %d°C: bawa jaket atau sweater tipis dan kaus kaki.", c.MadLo),
			"Udara kering: bawa pelembap bibir dan kulit, serta minum air yang cukup meski tidak terasa haus.",
			"Tawaf siang hari terasa nyaman, waktu yang baik untuk ibadah lebih lama di pelataran.")
	case "peralihan", "hangat":
		in.Tips = append(in.Tips,
			"Siang mulai terik: pakai payung atau topi saat di luar masjid, dan bawa botol semprot air.",
			"Malam masih nyaman untuk tawaf dan i'tikaf di Masjidil Haram.")
	default:
		in.Tips = append(in.Tips,
			fmt.Sprintf("Siang bisa mencapai %d°C: lakukan tawaf dan sa'i setelah Isya atau ba'da Subuh, hindari pukul 11.00–16.00 di area terbuka.", c.MakHi),
			"Bawa payung UV, semprotan air, dan sandal yang tidak panas; minum zamzam rutin untuk mencegah dehidrasi.",
			"Perhatikan jamaah lansia: waspadai tanda heatstroke (pusing, mual, kulit panas dan kering).")
	}
	if ramadhan {
		in.Tips = append(in.Tips,
			"Ramadhan: jadwal makan mengikuti sahur dan iftar; iftar bersama di pelataran masjid biasanya tersedia, datang lebih awal karena cepat penuh.",
			"Tarawih dan tahajud di Masjidil Haram sangat padat: datang 1–2 jam sebelum waktu shalat dan bawa sajadah tipis.")
	}
	if in.CrowdLevel >= 3 {
		in.Tips = append(in.Tips,
			"Masjidil Haram paling padat ba'da Isya hingga tengah malam dan setelah Subuh. Untuk tawaf yang lebih lengang, coba pukul 09.00–11.00 atau lantai atas (mataf lantai 2/3).",
			"Pesan jadwal Raudhah lewat aplikasi Nusuk jauh hari, karena slot cepat habis saat ramai.")
	} else {
		in.Tips = append(in.Tips,
			"Lebih lengang: kesempatan baik untuk mendapat slot Raudhah (pesan lewat Nusuk) dan tawaf lebih dekat ke Ka'bah.")
	}
	return in
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
