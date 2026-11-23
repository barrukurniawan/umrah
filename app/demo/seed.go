package demo

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

	"umrah/app/accounts"
	"umrah/app/draws"
	"umrah/app/repositories"
	"umrah/app/services"
)

// Seed demo: deterministik (rand dengan seed tetap) dan relatif terhadap hari ini, supaya setiap
// reset menghasilkan cerita yang sama: 3 travel mitra fiktif, ±500 lead 60 hari, ±30 booking,
// promo, 2 undian (satu berjalan, satu sudah diundi), pengajuan pembiayaan, ulasan.

type seedPkg struct {
	name                   string
	price, triple, double  int64
	days                   int
	airline, via           string
	direct                 bool
	hm, hd                 string
	sm, sd, dm, dd         int
	every, first, count    int // jadwal: hari pertama dari sekarang, jarak antar keberangkatan, jumlah
	seats                  int
	guide                  string
}

var seedTravels = []struct {
	t     Travel
	terms TravelTerms
	pkgs  []seedPkg
}{
	{
		Travel{Slug: "amanah-demo", Name: "Amanah Demo Tour", City: "Bandung", Fictional: true, Partner: true, PPIUNo: "PPIU-DEMO-0001",
			PPIUStatus: "terverifikasi", PPIUSource: "Contoh data cek ke sistem Kemenhaj (demo)", EscrowBank: "Bank Syariah Contoh",
			EscrowAccount: "7001234561", EscrowHolder: "PT Amanah Demo Tour – Rek. Penampungan Dana Jamaah", Plan: "pro", Komisi: 3,
			Rating: 4.7, Since: 2014, PIC: "Ust. Hendra (contoh)", Phone: "0812-0000-0001"},
		TravelTerms{DPMinUmrohku: 500_000, DPNormalTravel: 3_000_000, PelunasanHMinus: 45, KomisiPersen: 3, AgreementRef: "[isi no. perjanjian]",
			DPHangusRule: "Pembatalan oleh jamaah setelah DP: DP tidak dikembalikan untuk biaya visa & tiket yang sudah dipesan; sisa dana dikembalikan sesuai ketentuan travel."},
		[]seedPkg{
			{name: "Umroh Hemat Berkah 9 Hari", price: 24_900_000, triple: 26_400_000, double: 27_900_000, days: 9, airline: "Saudia", direct: true,
				hm: "Al Massa Bader", hd: "Durrat Al Eiman", sm: 3, sd: 3, dm: 650, dd: 200, first: 18, every: 14, count: 8, seats: 45, guide: "Ust. Hendra Gunawan (contoh)"},
			{name: "Umroh Plus Thaif 12 Hari", price: 29_500_000, triple: 31_000_000, double: 32_500_000, days: 12, airline: "Saudia", direct: true,
				hm: "Elaf Kinda", hd: "Al Mukhtara Al Gharbi", sm: 4, sd: 4, dm: 300, dd: 150, first: 26, every: 21, count: 6, seats: 40},
		},
	},
	{
		Travel{Slug: "baitullah-contoh", Name: "Baitullah Contoh Travel", City: "Jakarta Timur", Fictional: true, Partner: true, PPIUNo: "PPIU-DEMO-0002",
			PPIUStatus: "terverifikasi", PPIUSource: "Contoh data cek ke sistem Kemenhaj (demo)", EscrowBank: "Bank Syariah Contoh",
			EscrowAccount: "7001234562", EscrowHolder: "PT Baitullah Contoh Travel – Rek. Penampungan Dana Jamaah", Plan: "premium", Komisi: 3.5,
			Rating: 4.8, Since: 2009, PIC: "Ibu Nurul (contoh)", Phone: "0812-0000-0002"},
		TravelTerms{DPMinUmrohku: 750_000, DPNormalTravel: 5_000_000, PelunasanHMinus: 45, KomisiPersen: 3.5, AgreementRef: "[isi no. perjanjian]",
			DPHangusRule: "Pembatalan H-60 atau lebih: DP kembali 50%. Kurang dari H-60: DP tidak kembali. Ganti nama jamaah gratis sampai H-30."},
		[]seedPkg{
			{name: "Umroh Nyaman Ajyad 9 Hari", price: 27_900_000, triple: 29_400_000, double: 31_400_000, days: 9, airline: "Garuda Indonesia", direct: true,
				hm: "Jada Ajyad", hd: "Dar Al Eiman Al Haram", sm: 4, sd: 4, dm: 350, dd: 100, first: 14, every: 10, count: 10, seats: 45},
			{name: "Umroh VIP Clock Tower 10 Hari", price: 38_500_000, triple: 41_000_000, double: 44_500_000, days: 10, airline: "Saudia", direct: true,
				hm: "Fairmont Makkah Clock Royal Tower", hd: "Anwar Al Madinah Movenpick", sm: 5, sd: 5, dm: 50, dd: 50, first: 33, every: 28, count: 5, seats: 30},
			{name: "Umroh Ramadhan 15 Hari", price: 42_000_000, triple: 44_000_000, double: 46_500_000, days: 15, airline: "Saudia", direct: true,
				hm: "Le Meridien Towers Makkah", hd: "Golden Tulip Al Ansar", sm: 4, sd: 4, dm: 900, dd: 120, first: -1, count: 2, seats: 45},
		},
	},
	{
		Travel{Slug: "safa-sampel", Name: "Safa Sampel Wisata", City: "Surabaya", Fictional: true, Partner: true, PPIUNo: "PPIU-DEMO-0003",
			PPIUStatus: "terverifikasi", PPIUSource: "Contoh data cek ke sistem Kemenhaj (demo)", EscrowBank: "Bank Syariah Contoh",
			EscrowAccount: "7001234563", EscrowHolder: "PT Safa Sampel Wisata – Rek. Penampungan Dana Jamaah", Plan: "pro", Komisi: 3,
			Rating: 4.6, Since: 2017, PIC: "Bpk. Arif (contoh)", Phone: "0812-0000-0003"},
		TravelTerms{DPMinUmrohku: 1_000_000, DPNormalTravel: 4_000_000, PelunasanHMinus: 40, KomisiPersen: 3, AgreementRef: "[isi no. perjanjian]",
			DPHangusRule: "DP yang sudah dibayar dapat dialihkan ke jadwal lain satu kali tanpa biaya. Pembatalan: DP dipotong biaya visa & tiket aktual."},
		[]seedPkg{
			{name: "Umroh Keluarga Hemat 9 Hari", price: 23_500_000, triple: 25_000_000, double: 26_500_000, days: 9, airline: "Oman Air", via: "Muscat",
				hm: "Maysan Al Mashaer", hd: "Al-Sham Province", sm: 3, sd: 3, dm: 900, dd: 300, first: 21, every: 14, count: 8, seats: 45},
			{name: "Umroh Plus Turki 12 Hari", price: 33_900_000, triple: 35_400_000, double: 36_900_000, days: 12, airline: "Turkish Airlines", via: "Istanbul",
				hm: "Al Kiswah Towers", hd: "Arkan Al Manar", sm: 4, sd: 4, dm: 1200, dd: 250, first: 40, every: 30, count: 4, seats: 35},
		},
	},
}

var seedNames = []string{"Ahmad Fauzi", "Nur Aisyah", "Budi Santoso", "Dewi Lestari", "Rizky Pratama", "Siti Nurhaliza", "Hendra Wijaya",
	"Fitri Handayani", "Agus Salim", "Rina Marlina", "Yusuf Hidayat", "Lina Kartika", "Dimas Saputra", "Ratna Sari", "Fajar Nugroho",
	"Wulan Septiani", "Ilham Maulana", "Maya Anggraini", "Taufik Rahman", "Indah Permata", "Bayu Firmansyah", "Nadia Putri", "Arif Setiawan",
	"Yuni Astuti", "Hasan Basri", "Laila Fitriani", "Doni Kurniawan", "Eka Puspita", "Rahmat Hidayat", "Sri Wahyuni"}

var seedCities = []string{"Bandung", "Jakarta Selatan", "Bekasi", "Depok", "Bogor", "Tangerang", "Surabaya", "Sidoarjo", "Malang", "Semarang",
	"Yogyakarta", "Solo", "Cirebon", "Garut", "Makassar", "Medan", "Palembang", "Pekanbaru", "Balikpapan", "Banjarmasin"}

func seedBusiness() {
	rng := rand.New(rand.NewSource(20261123))
	now := time.Now().In(WIB)
	today := Today()

	for _, st := range seedTravels {
		t := st.t
		checked := now.AddDate(0, 0, -3)
		t.PPIUCheckedAt = &checked
		t.PPIUValidUntil = now.AddDate(2, 4, 0).Format("2006-01-02")
		t.PlanUntil = now.AddDate(0, 1, 7).Format("2006-01-02")
		t.CreatedAt = now.AddDate(0, -5, 0)
		DB.Create(&t)
		terms := st.terms
		terms.TravelSlug = t.Slug
		DB.Create(&terms)
		for _, sp := range st.pkgs {
			p := PortalPackage{TravelSlug: t.Slug, Name: sp.name, Price: sp.price, PriceTriple: sp.triple, PriceDouble: sp.double,
				Duration: sp.days, Airline: sp.airline, Direct: sp.direct, TransitVia: sp.via, HotelMakkah: sp.hm, HotelMadinah: sp.hd,
				StarsMakkah: sp.sm, StarsMadinah: sp.sd, DistMakkah: sp.dm, DistMadinah: sp.dd, Guide: sp.guide, Active: true, UpdatedAt: now,
				Facilities: "Tiket pesawat PP " + sp.airline + "\nVisa umroh\nHotel sesuai paket\nMakan 3x sehari\nBus AC selama di Saudi\nMuthawwif & pembimbing ibadah\nManasik 2x\nAir zamzam 5 liter\nPerlengkapan (koper, ihram/mukena, buku doa)",
				Excludes:   "Paspor\nVaksin meningitis\nTiket domestik ke Jakarta\nKeperluan pribadi"}
			DB.Create(&p)
			var dates []string
			if sp.first < 0 { // Ramadhan 1448 H (perkiraan): akhir Februari 2027
				dates = []string{"2027-02-20", "2027-02-26"}
			} else {
				for i := 0; i < sp.count; i++ {
					dates = append(dates, AddDays(today, sp.first+i*sp.every))
				}
			}
			for i, d := range dates {
				left := sp.seats - rng.Intn(sp.seats/2) - i
				if i == 0 {
					left = 4 + rng.Intn(5)
				}
				DB.Create(&PortalDeparture{PackageID: p.ID, Date: d, ReturnDate: AddDays(d, sp.days-1), SeatsTotal: sp.seats, SeatsLeft: max(0, left)})
			}
		}
	}

	// Posisi unggulan: Alviandra (sama seperti produksi) + satu travel mitra.
	DB.Create(&Placement{TravelName: "Alviandra Travel", MinBudget: 28_000_000, MaxBudget: 29_000_000, StartsAt: AddDays(today, -40), EndsAt: AddDays(today, 50), PricePerDay: 75_000, Active: true, CreatedAt: now})
	DB.Create(&Placement{TravelName: "Baitullah Contoh Travel", MinBudget: 27_000_000, MaxBudget: 28_000_000, StartsAt: AddDays(today, -20), EndsAt: AddDays(today, 40), PricePerDay: 75_000, Active: true, CreatedAt: now})

	// Promo (dibiayai umrohku).
	start, end := now.AddDate(0, -3, 0), now.AddDate(0, 3, 0)
	DB.Create(&PromoRule{Title: "Diskon DP Rp 250 rb khusus booking lewat umrohku", Kind: "diskon_dp", Amount: 250_000, MinPrice: 20_000_000, Quota: 300, StartsAt: start, EndsAt: end, FundedBy: "umrohku", Active: true})
	DB.Create(&PromoRule{Title: "Cashback pelunasan 2% (maks. Rp 750 rb)", Kind: "cashback_pelunasan", Percent: 2, Cap: 750_000, MinPrice: 20_000_000, Quota: 300, StartsAt: start, EndsAt: end, FundedBy: "umrohku", Active: true})
	DB.Create(&PromoRule{Code: "UMROHKU500", Title: "Kode UMROHKU500: diskon DP Rp 500 rb", Kind: "diskon_dp", Amount: 500_000, MinPrice: 25_000_000, Quota: 50, StartsAt: start, EndsAt: end, FundedBy: "umrohku", Active: true})
	DB.Create(&PromoRule{Code: "RAMADHAN1448", Title: "Kode RAMADHAN1448: cashback Rp 1 jt paket Ramadhan", Kind: "cashback_pelunasan", Amount: 1_000_000, MinPrice: 40_000_000, TravelSlug: "baitullah-contoh", Quota: 20, StartsAt: start, EndsAt: end, FundedBy: "umrohku", Active: true})

	// Undian: satu sudah diundi (periode lalu), satu sedang berjalan.
	past := Draw{Title: "Undian Umroh Gratis Periode Jun–Agu 2026", PrizeDesc: "1 paket umroh 9 hari di travel mitra pilihan pemenang", Winners: 1,
		StartsAt: now.AddDate(0, -5, 0), EndsAt: now.AddDate(0, -2, -10), DrawAt: now.AddDate(0, -2, 0), PermitNo: "dalam proses", Status: "terbuka", Terms: drawTerms}
	DB.Create(&past)
	open := Draw{Title: "Undian Umroh Gratis Periode Nov 2026 – Jan 2027", PrizeDesc: "2 paket umroh 9 hari di travel mitra pilihan pemenang", Winners: 2,
		StartsAt: now.AddDate(0, -1, 0), EndsAt: now.AddDate(0, 2, 0), DrawAt: now.AddDate(0, 2, 7), PermitNo: "dalam proses", Status: "terbuka", Terms: drawTerms}
	DB.Create(&open)

	// Akun jamaah dummy (users.db instance demo) + akun demo utama.
	main := EnsureDemoUser()
	var users []*accounts.User
	for i, n := range seedNames {
		if u := ensureUser(fmt.Sprintf("demo:j%02d", i), n, strings.ToLower(strings.ReplaceAll(n, " ", "."))+"@jamaah.contoh"); u != nil {
			users = append(users, u)
		}
	}
	if main == nil || len(users) == 0 {
		log.Println("[demo] akun dummy gagal dibuat (users.db nonaktif?)")
		return
	}

	// Lead 60 hari: lebih ramai menjelang akhir pekan & naik perlahan (efek promo).
	travels := PartnerTravels()
	for day := 59; day >= 0; day-- {
		d := now.AddDate(0, 0, -day)
		base := 3 + (59-day)/15
		if wd := d.Weekday(); wd == time.Friday || wd == time.Saturday || wd == time.Sunday {
			base += 3
		}
		for _, t := range travels {
			pk := seedPkgNames(t.Slug)
			n := base/2 + rng.Intn(base)
			if t.Slug == "baitullah-contoh" {
				n += 2
			}
			for i := 0; i < n; i++ {
				ts := time.Date(d.Year(), d.Month(), d.Day(), 7+rng.Intn(15), rng.Intn(60), rng.Intn(60), 0, WIB)
				kind := []string{"wa", "wa", "wa", "web", "form"}[rng.Intn(5)]
				l := Lead{TravelSlug: t.Slug, Package: pk[rng.Intn(len(pk))], Kind: kind, Visitor: fmt.Sprintf("v%06d", rng.Intn(4000)),
					Source: []string{"Google", "Instagram", "Threads", "WhatsApp", "Langsung", "TikTok"}[rng.Intn(6)],
					Device: []string{"HP", "HP", "HP", "Desktop"}[rng.Intn(4)], Status: "baru", Billable: true, CreatedAt: ts}
				if kind == "form" {
					l.Name, l.City = seedNames[rng.Intn(len(seedNames))], seedCities[rng.Intn(len(seedCities))]
					l.Phone = fmt.Sprintf("08%d%08d", 11+rng.Intn(9), rng.Intn(100000000))
				}
				switch r := rng.Intn(100); {
				case r < 7:
					l.Billable, l.Reason = false, "Duplikat (pengunjung sama < 24 jam)"
				case r < 9:
					l.Billable, l.Status, l.Reason = false, "tidak_valid", "Nomor tidak aktif (disetujui admin)"
				case day > 3 && r < 45:
					l.Status = "dihubungi"
				case day > 3 && r < 55:
					l.Status = "closing"
				}
				DB.Create(&l)
			}
		}
	}

	// Booking: status beragam; DP lunas otomatis masuk undian yang periodenya cocok.
	type bk struct {
		user     *accounts.User
		travel   string
		pkg      int
		dep      int // indeks jadwal
		people   int
		room     string
		status   string
		daysAgo  int
	}
	var plan []bk
	statuses := []string{"dp_lunas", "dp_lunas", "dp_lunas", "lunas", "dp_lunas", "menunggu_dp", "batal", "lunas", "dp_lunas", "pulang"}
	for i := 0; i < 30; i++ {
		u := users[i%len(users)]
		t := travels[i%len(travels)]
		st := statuses[i%len(statuses)]
		ago := 2 + rng.Intn(55)
		if st == "pulang" {
			ago = 100 + rng.Intn(40)
		}
		if st == "menunggu_dp" {
			ago = 0
		}
		plan = append(plan, bk{u, t.Slug, rng.Intn(2), rng.Intn(3), 1 + rng.Intn(4), []string{"quad", "quad", "triple", "double"}[rng.Intn(4)], st, ago})
	}
	// Booking milik akun demo utama: satu DP lunas (bisa dilunasi), satu sudah pulang (bisa diulas).
	plan = append(plan, bk{main, "baitullah-contoh", 0, 1, 2, "quad", "dp_lunas", 6}, bk{main, "amanah-demo", 0, 0, 1, "quad", "pulang", 120})
	for _, p := range plan {
		seedBooking(rng, p.user, p.travel, p.pkg, p.dep, p.people, p.room, p.status, p.daysAgo, &past)
	}

	// Undian lampau: tutup & undi dengan algoritma sungguhan.
	if es := DrawEntries(past.ID); len(es) > 0 {
		var de []draws.Entry
		for _, e := range es {
			de = append(de, draws.Entry{Code: e.Code, Void: e.Void})
		}
		closed := past.EndsAt.Add(24 * time.Hour)
		past.EntriesHash, past.ClosedAt, past.Status = draws.EntriesHash(de), &closed, "ditutup"
		DB.Save(&past)
		RunDraw(past.ID, "71 42 09 (dibacakan saksi)", "Saksi contoh: Ust. Abdullah & notaris [isi]")
		drawn := past.DrawAt
		DB.Model(&Draw{}).Where("id = ?", past.ID).Update("drawn_at", &drawn)
	}

	// Pembiayaan (rujukan).
	fin := []struct {
		u      *accounts.User
		p      string
		amount int64
		tenor  int
		status string
		ago    int
	}{
		{users[1], "amanah", 24_900_000, 12, "dicairkan", 40}, {users[4], "barokah", 27_900_000, 24, "disetujui", 9},
		{users[7], "amanah", 29_500_000, 12, "diproses", 3}, {users[10], "bank-contoh", 23_500_000, 12, "ditolak", 15},
		{users[12], "amanah", 38_500_000, 24, "diajukan", 1}, {main, "amanah", 25_000_000, 12, "diproses", 2},
	}
	for _, f := range fin {
		created := now.AddDate(0, 0, -f.ago)
		a := FinancingApplication{Code: NewCode("PBY", 6), UserID: f.u.ID, PartnerSlug: f.p, TravelName: "Baitullah Contoh Travel",
			PackageName: "Umroh Nyaman Ajyad 9 Hari", Amount: f.amount, Tenor: f.tenor, Name: f.u.Name, Phone: "0812-5555-01" + fmt.Sprint(10+f.ago),
			Job: []string{"Karyawan swasta", "PNS", "Wiraswasta", "Guru"}[f.ago%4], Income: int64(6+f.ago%9) * 1_000_000,
			City: seedCities[f.ago%len(seedCities)], ConsentAt: &created, Status: f.status, CreatedAt: created}
		for _, e := range Estimates(f.amount, f.tenor) {
			if e.Partner.Slug == f.p {
				a.EstMonthly = e.Monthly
			}
		}
		if f.status != "diajukan" && f.status != "diproses" {
			dec := created.Add(48 * time.Hour)
			a.DecidedAt = &dec
		}
		if f.status == "dicairkan" {
			dis := created.Add(96 * time.Hour)
			a.DisbursedAt = &dis
			if p := PartnerBySlug(f.p); p != nil {
				a.ReferralFee = int64(math.Round(float64(a.Amount) * p.FeePct / 100))
			}
		}
		DB.Create(&a)
	}

	// Ulasan contoh (ditandai contoh) untuk setiap travel mitra.
	sampleReviews := []struct{ r int; text string }{
		{5, "Muthawwif sabar sekali membimbing ibu saya yang sudah sepuh. Hotel dekat, jadwal sesuai yang dijanjikan."},
		{5, "Bayar DP lewat umrohku lebih ringan, sisanya kami cicil lewat tabungan. Berangkat tepat waktu, alhamdulillah."},
		{4, "Pelayanan baik, makanan cocok di lidah. Hanya bus di Madinah sempat telat 30 menit."},
		{5, "Manasik jelas, grup WA aktif. Ada ustadz pembimbing yang menjelaskan sejarah di setiap ziarah."},
	}
	for ti, t := range travels {
		for i, sr := range sampleReviews {
			if (i+ti)%4 == 3 && ti > 0 {
				continue
			}
			DB.Create(&Review{TravelSlug: t.Slug, Rating: sr.r, Text: sr.text, Name: seedNames[(i*7+ti*3)%len(seedNames)], City: seedCities[(i+ti*5)%len(seedCities)],
				Month: monthLabel(now.AddDate(0, -2-i, 0).Format("2006-01-02")), Sample: true, CreatedAt: now.AddDate(0, 0, -20*i-ti)})
		}
	}

	// Minat "Mau cicil?" 30 hari.
	for day := 29; day >= 0; day-- {
		for i := 0; i < 3+rng.Intn(6); i++ {
			t := travels[rng.Intn(len(travels))]
			pk := seedPkgNames(t.Slug)
			path := []string{"sebelum", "sebelum", "sebelum", "pembiayaan", "pembiayaan", "lihat"}[rng.Intn(6)]
			tenor := 0
			if path == "pembiayaan" {
				tenor = []int{6, 12, 12, 24}[rng.Intn(4)]
			}
			DB.Create(&CicilIntent{Visitor: fmt.Sprintf("v%06d", rng.Intn(4000)), Travel: t.Name, Package: pk[rng.Intn(len(pk))],
				Price: 25_000_000 + int64(rng.Intn(15))*1_000_000, Path: path, Tenor: tenor, CreatedAt: now.AddDate(0, 0, -day).Add(-time.Duration(rng.Intn(600)) * time.Minute)})
		}
	}

	// Langganan & status tagihan.
	prev := PrevPeriod(CurrentPeriod())
	for _, t := range travels {
		p := PlanByKey(t.Plan)
		if p.Price > 0 {
			a, _ := periodRange(prev)
			DB.Create(&SubscriptionLog{TravelSlug: t.Slug, Plan: t.Plan, Months: 1, Amount: p.Price, From: a.Format("2006-01-02"), Until: a.AddDate(0, 1, -1).Format("2006-01-02"), CreatedAt: a.Add(9 * time.Hour)})
		}
		if t.Slug != "safa-sampel" {
			MarkInvoicePaid(t.Slug, prev)
		}
		Notify("travel:"+t.Slug, "email", t.PIC, "Selamat datang di Portal Mitra umrohku",
			"Paket Anda kini dikelola langsung dari portal (bukan hasil scraping). Ubah harga & sisa seat kapan saja; perubahan tampil di pencarian dalam hitungan detik.", "/portal/paket")
	}

	// Pendaftaran mitra baru.
	DB.Create(&PartnerSignup{Company: "PT Contoh Nur Hidayah Wisata", Brand: "Nur Hidayah Contoh", PPIUNo: "[diisi pendaftar]", City: "Medan", PIC: "Bpk. Rahmat (contoh)", Phone: "0813-0000-1111", Email: "mitra@nurhidayah.contoh", Note: "Ingin mencoba portal & booking DP.", Status: "baru", CreatedAt: now.AddDate(0, 0, -2)})
	DB.Create(&PartnerSignup{Company: "PT Contoh Madinah Berkah", Brand: "Madinah Berkah Contoh", PPIUNo: "[diisi pendaftar]", City: "Makassar", PIC: "Ibu Hasnah (contoh)", Phone: "0813-0000-2222", Email: "halo@madinahberkah.contoh", Status: "dihubungi", CreatedAt: now.AddDate(0, 0, -9)})

	// Tabungan akun demo, sudah terhubung ke rekening tabungan umroh bank syariah mitra (demo).
	goal := accounts.SavingsGoal{UserID: main.ID, Title: "Umroh bareng Ibu", Target: 2 * 27_900_000, TargetDate: AddDays(today, 250), People: 2,
		PackageKey: services.PackageKey("Baitullah Contoh Travel", "Umroh Nyaman Ajyad 9 Hari"), PackageLabel: "Baitullah Contoh Travel · Umroh Nyaman Ajyad 9 Hari",
		PackagePrice: 27_900_000, DownPayment: 750_000}
	if accounts.CreateGoal(&goal) == nil {
		for i := 0; i < 7; i++ {
			d := now.AddDate(0, -7+i, 0)
			accounts.DB.Create(&accounts.SavingsEntry{GoalID: goal.ID, UserID: main.ID, Amount: 2_500_000, Date: time.Date(d.Year(), d.Month(), 25, 0, 0, 0, 0, WIB).Format("2006-01-02"),
				Note: "Autodebet Bank Syariah Contoh", CreatedAt: d})
		}
		DB.Create(&SavingsLink{UserID: main.ID, GoalID: goal.ID, Bank: "Bank Syariah Contoh", AccountNo: "7101 2026 1123", Autodebit: 2_500_000, Day: 25, LinkedAt: now.AddDate(0, -7, 0)})
	}

	DB.Exec("UPDATE promo_rules SET used = (SELECT COUNT(*) FROM promo_uses WHERE promo_uses.promo_id = promo_rules.id AND promo_uses.status <> 'batal')")
	Notify("admin", "email", "admin", "Demo siap", "Data demo dibuat ulang. Gunakan pengalih peran (kanan bawah) untuk berpindah antara Jamaah, Travel, Admin, Mitra pembiayaan, dan Toko.", "/demo")
	seedShop()
	log.Println("[demo] seed selesai")
}

func seedPkgNames(slug string) []string {
	for _, st := range seedTravels {
		if st.t.Slug == slug {
			var n []string
			for _, p := range st.pkgs {
				n = append(n, p.name)
			}
			return n
		}
	}
	return []string{"Paket umroh"}
}

func seedBooking(rng *rand.Rand, u *accounts.User, slug string, pkgIdx, depIdx, people int, room, status string, daysAgo int, _ *Draw) {
	var st = seedTravels[0]
	for _, x := range seedTravels {
		if x.t.Slug == slug {
			st = x
		}
	}
	sp := st.pkgs[pkgIdx%len(st.pkgs)]
	var pp PortalPackage
	if DB.Where("travel_slug = ? AND name = ?", slug, sp.name).Take(&pp).Error != nil {
		return
	}
	var deps []PortalDeparture
	DB.Where("package_id = ?", pp.ID).Order("date").Find(&deps)
	if len(deps) == 0 {
		return
	}
	dep := deps[depIdx%len(deps)]
	now := time.Now()
	created := now.AddDate(0, 0, -daysAgo).Add(-time.Duration(rng.Intn(500)) * time.Minute)
	depDate, retDate := dep.Date, dep.ReturnDate
	if status == "pulang" { // sudah berangkat & pulang: tanggal di masa lalu
		depDate = created.AddDate(0, 0, 50).In(WIB).Format("2006-01-02")
		retDate = AddDays(depDate, sp.days-1)
	}
	price := sp.price
	switch room {
	case "triple":
		price = sp.triple
	case "double":
		price = sp.double
	}
	terms := st.terms
	var names []string
	for i := 0; i < people; i++ {
		if i == 0 {
			names = append(names, u.Name)
		} else {
			names = append(names, seedNames[rng.Intn(len(seedNames))])
		}
	}
	pj, _ := json.Marshal(names)
	dpPer := terms.DPMinUmrohku
	if rng.Intn(3) == 0 {
		dpPer = 2_000_000
	}
	total := price * int64(people)
	cashback := min(int64(math.Round(float64(total)*0.02)), 750_000)
	b := Booking{Code: NewCode("UMK", 6), UserID: u.ID, TravelSlug: slug, TravelName: st.t.Name, PackageKey: services.PackageKey(st.t.Name, sp.name),
		PackageName: sp.name, DepartureDate: depDate, ReturnDate: retDate, Room: room, People: people, PricePerPerson: price, Total: total,
		DPAmount: dpPer * int64(people), DPDiscount: 250_000, CashbackPromised: cashback, Pilgrims: string(pj), ContactName: u.Name,
		ContactPhone: fmt.Sprintf("0812-%04d-%04d", rng.Intn(10000), rng.Intn(10000)), City: seedCities[rng.Intn(len(seedCities))],
		Status: status, PelunasanDue: AddDays(depDate, -terms.PelunasanHMinus), KomisiPersen: terms.KomisiPersen, CreatedAt: created}
	if b.PelunasanDue <= Today() && (status == "dp_lunas") {
		b.PelunasanDue = AddDays(Today(), 5+rng.Intn(20))
	}
	paid := created.Add(time.Duration(20+rng.Intn(200)) * time.Minute)
	if status != "menunggu_dp" {
		b.PaidAt = &paid
	}
	if status == "lunas" || status == "pulang" {
		l := paid.AddDate(0, 0, 10)
		b.LunasAt, b.CashbackApplied = &l, cashback
	}
	if status == "batal" {
		c := paid.AddDate(0, 0, 3)
		b.CancelledAt, b.CancelReason = &c, "Dibatalkan jamaah (sakit), DP mengikuti ketentuan travel"
	}
	DB.Create(&b)
	t := TravelBySlug(slug)
	holder, bank, acct := st.t.EscrowHolder, st.t.EscrowBank, st.t.EscrowAccount
	if t != nil {
		holder, bank, acct = t.EscrowHolder, t.EscrowBank, t.EscrowAccount
	}
	_ = acct
	payStatus := "berhasil"
	var payPaid *time.Time = &paid
	if status == "menunggu_dp" {
		payStatus, payPaid = "menunggu", nil
	}
	DB.Create(&Payment{BookingID: b.ID, Kind: "dp", Method: []string{"va", "qris"}[rng.Intn(2)], Bank: bank, VANumber: "7001" + fmt.Sprintf("%012d", rng.Int63n(1e12)),
		Holder: holder, QRPayload: "DEMO-QRIS|" + b.Code, Amount: b.DPAmount - b.DPDiscount, Status: payStatus, Ref: b.Code + "-DP", ExpiresAt: created.Add(24 * time.Hour), PaidAt: payPaid, CreatedAt: created})
	if b.LunasAt != nil {
		DB.Create(&Payment{BookingID: b.ID, Kind: "pelunasan", Method: "va", Bank: bank, VANumber: "7001" + fmt.Sprintf("%012d", rng.Int63n(1e12)), Holder: holder,
			Amount: b.Remaining(), Status: "berhasil", Ref: b.Code + "-LUNAS", ExpiresAt: b.LunasAt.Add(24 * time.Hour), PaidAt: b.LunasAt, CreatedAt: *b.LunasAt})
	}
	pstat := "dijanjikan"
	if status == "batal" {
		pstat = "batal"
	}
	var dpRule, cbRule PromoRule
	DB.Where("kind = ? AND code = ?", "diskon_dp", "").Take(&dpRule)
	DB.Where("kind = ? AND code = ?", "cashback_pelunasan", "").Take(&cbRule)
	if status != "menunggu_dp" {
		DB.Create(&PromoUse{PromoID: dpRule.ID, BookingID: b.ID, Kind: "diskon_dp", Amount: b.DPDiscount, Status: map[bool]string{true: "batal", false: "diterapkan"}[status == "batal"], AppliedAt: b.PaidAt, CreatedAt: created})
	}
	cbStat := pstat
	if b.LunasAt != nil {
		cbStat = "diterapkan"
	}
	DB.Create(&PromoUse{PromoID: cbRule.ID, BookingID: b.ID, Kind: "cashback_pelunasan", Amount: cashback, Status: cbStat, AppliedAt: b.LunasAt, CreatedAt: created})
	if b.PaidAt != nil {
		CreateEntries(&b)
		if status == "batal" {
			DB.Model(&DrawEntry{}).Where("booking_id = ?", b.ID).Update("void", true)
		}
	}
	if status == "dp_lunas" || status == "lunas" {
		DB.Model(&PortalDeparture{}).Where("id = ?", dep.ID).Update("seats_left", max(0, dep.SeatsLeft-people))
	}
}

const drawTerms = `1. Penyelenggara: PT umrohku [isi]. Izin undian gratis berhadiah dari Kementerian Sosial: [dalam proses].
2. GRATIS. Tidak ada biaya ikut, tidak ada kupon yang dijual, dan harga paket tidak dinaikkan.
3. Setiap jamaah dalam booking yang DP-nya lunas di periode undian otomatis mendapat 1 kode. Tidak ada cara lain untuk mendapat kode.
4. Booking yang dibatalkan sebelum pengundian: kodenya batal dan tidak ikut diundi.
5. Hadiah dibiayai anggaran promosi penyelenggara, bukan dari dana jamaah. Hadiah tidak dapat diuangkan [atau isi kebijakan]. Pajak hadiah ditanggung [isi].
6. Saat undian ditutup, hash sha256 daftar kode diumumkan. Pengundian memakai seed yang dibacakan saksi; siapa pun bisa menghitung ulang hasilnya.
7. Mekanisme ditinjau dewan syariah / ustadz pendamping [isi].`

// ---- reset & pekerjaan berkala ----

// Reset menghapus semua data demo (termasuk akun dummy di users.db) lalu mengisi seed ulang.
func Reset() {
	if DB == nil {
		return
	}
	modelMu.Lock()
	ms := append([]any(nil), demoModels...)
	modelMu.Unlock()
	DB.Migrator().DropTable(ms...)
	DB.AutoMigrate(ms...)
	if accounts.DB != nil {
		var us []accounts.User
		accounts.DB.Where("google_sub LIKE ?", "demo:%").Find(&us)
		for _, u := range us {
			accounts.DeleteUser(u.ID)
		}
	}
	seedBusiness()
	refreshCache()
	ReloadPackages()
	log.Println("[demo] reset selesai")
}

func startJobs() {
	go func() {
		for {
			ExpireJob()
			refreshPlacements()
			time.Sleep(time.Hour)
		}
	}()
	go func() {
		for {
			now := time.Now().In(WIB)
			next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, WIB)
			if !next.After(now) {
				next = next.AddDate(0, 0, 1)
			}
			time.Sleep(time.Until(next))
			Reset()
		}
	}()
}

var _ = repositories.Reload
