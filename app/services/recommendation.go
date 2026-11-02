package services

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"umrah/app/models"
	"umrah/app/repositories"
)

type FilterInput struct {
	Budget   int
	Priority string
	Advanced []string
	Page     int
	Sort     string
	Month    string
	Travels  string
}

// Travel partner/promo: selalu di urutan teratas, tapi hanya pada budget tertentu
// (dipilih pemilik situs). Di luar rentang ini paketnya tidak ditampilkan sama sekali.
type featuredRange struct{ MinBudget, MaxBudget int }

var featuredTravels = map[string]featuredRange{
	"Alviandra Travel": {MinBudget: 28000000, MaxBudget: 29000000},
}

func isFeatured(travel string) bool { _, ok := featuredTravels[travel]; return ok }

// Paket promo boleh sedikit di atas budget (mis. Rp28,5 jt sudah tampil di budget Rp28 jt).
const featuredBudgetSlack = 500000

type ScoredPackage struct {
	models.Package
	Key               string // PackageKey, untuk ♡ simpan
	Featured          bool
	Score             int
	FacilityList      []string
	FacilityGeneric   bool   // travel belum merinci; FacilityList = fasilitas umum umroh
	SameDetails       bool   // semua tanggal: hotel & harga kamar sama (ditampilkan sekali)
	VaryingPrice      bool   // harga berbeda antar tanggal: tampilkan harga di setiap baris
	ShuttleMakkah     string // keterangan shuttle hotel Makkah ("" = tidak ada / tidak diketahui)
	ExcludeList       []string
	DepartureDatesStr string
	DistanceMakkah    string
	DistanceMadinah   string
}

func GetRecommendations(input FilterInput) ([]ScoredPackage, int) {
	var allPackages []models.Package
	repositories.DB.Preload("Travel").Preload("Details").Find(&allPackages)

	advanceFilter := make(map[string]bool)
	for _, a := range input.Advanced {
		if a != "" {
			advanceFilter[a] = true
		}
	}

	travelFilter := make(map[string]bool)
	hasTravelFilter := false
	if input.Travels != "" {
		for _, t := range strings.Split(input.Travels, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				travelFilter[t] = true
				hasTravelFilter = true
			}
		}
	}

	// Hanya keberangkatan mulai hari ini (WIB); tanggal "2006-01-02" aman dibandingkan sebagai string.
	today := time.Now().In(time.FixedZone("WIB", 7*3600)).Format("2006-01-02")

	// Filter bulan diterapkan per tanggal: kartu hanya berisi tanggal di bulan itu, sehingga
	// harga, cek budget, dan ringkasan tanggal ikut bulan yang dipilih. Bulan yang sudah lewat
	// tahun ini berarti tahun depan.
	monthPrefix := ""
	if m, err := strconv.Atoi(input.Month); err == nil && m >= 1 && m <= 12 {
		now := time.Now().In(time.FixedZone("WIB", 7*3600))
		y := now.Year()
		if m < int(now.Month()) {
			y++
		}
		monthPrefix = fmt.Sprintf("%d-%02d", y, m)
	}

	var scored []ScoredPackage
	for _, pkg := range allPackages {
		// Buang jadwal yang sudah lewat; paket tanpa jadwal mendatang tidak ditampilkan.
		var upcoming []models.DetailPackage
		for _, d := range pkg.Details {
			// Tanggal yang dinyatakan penuh oleh travel tidak ditampilkan.
			if d.DepartureDate != "" && d.DepartureDate >= today && !d.IsFull &&
				(monthPrefix == "" || strings.HasPrefix(d.DepartureDate, monthPrefix)) {
				upcoming = append(upcoming, d)
			}
		}
		if len(upcoming) == 0 {
			continue
		}
		// Harga tiap tanggal dibuat eksplisit (0 = sama dengan harga paket saat import),
		// supaya bisa ditampilkan di setiap baris bila harganya berbeda-beda.
		varying := false
		for i := range upcoming {
			if upcoming[i].Price == 0 {
				upcoming[i].Price = pkg.Price
			}
			varying = varying || upcoming[i].Price != upcoming[0].Price
		}
		pkg.Details = upcoming
		pkg.Price = availablePrice(pkg.Price, upcoming)

		budget := input.Budget
		if fr, ok := featuredTravels[pkg.Travel.Name]; ok {
			if input.Budget < fr.MinBudget || input.Budget > fr.MaxBudget {
				continue
			}
			budget += featuredBudgetSlack
		}
		if pkg.Price > budget {
			continue
		}

		if hasTravelFilter && !travelFilter[pkg.Travel.Name] {
			continue
		}

		// Jarak: klaim travel dulu; jika tidak ada, perkiraan dari hotel yang dikenal
		// (diberi "±"); jika hotel tidak dikenal, kosong (chip tidak menampilkan jarak).
		// Urutan sumber jarak: koordinat hotel (diukur pemilik situs / hitungan peta, sama untuk
		// semua travel) -> klaim travel -> tabel perkiraan lama.
		distMakkah, distMadinah := "", ""
		if len(pkg.Details) > 0 {
			distMakkah = geoDistanceLabel("makkah", pkg.Details[0].HotelMakkah)
			distMadinah = geoDistanceLabel("madinah", pkg.Details[0].HotelMadinah)
		}
		if distMakkah == "" {
			distMakkah = formatMeters(pkg.DistanceMakkah)
		}
		if distMadinah == "" {
			distMadinah = formatMeters(pkg.DistanceMadinah)
		}
		if len(pkg.Details) > 0 {
			if distMakkah == "" {
				if d := getDistanceMakkah(pkg.Details[0].HotelMakkah); d != "" {
					distMakkah = "±" + d
				}
			}
			if distMadinah == "" {
				if d := getDistanceMadinah(pkg.Details[0].HotelMadinah); d != "" {
					distMadinah = "±" + d
				}
			}
		}

		// Override DB values dynamically (tidak diketahui = paling jauh saat sort)
		pkg.HotelDistance = parseMinDistance(strings.TrimPrefix(distMakkah, "±"))
		if distMakkah == "" {
			pkg.HotelDistance = unknownDistance
		}
		pkg.IsNearHaram = pkg.HotelDistance <= 550

		// Apply advanced filters always if set (regardless of priority)
		if advanceFilter["direct"] && !pkg.IsDirect {
			continue
		}
		if advanceFilter["transit"] && pkg.IsDirect {
			continue
		}
		if advanceFilter["near_haram"] && !pkg.IsNearHaram {
			continue
		}
		if advanceFilter["family_friendly"] && !pkg.IsKidFriendly && !pkg.IsSeniorFriendly {
			continue
		}
		if advanceFilter["quad"] || advanceFilter["triple"] || advanceFilter["double"] {
			hasRoom := false
			for _, d := range pkg.Details {
				rt := strings.ToLower(d.RoomType)
				if (advanceFilter["quad"] && rt == "quad") ||
					(advanceFilter["triple"] && rt == "triple") ||
					(advanceFilter["double"] && rt == "double") {
					hasRoom = true
					break
				}
			}
			if !hasRoom {
				continue
			}
		}

		score := 0

		if advanceFilter["near_haram"] {
			if pkg.IsNearHaram {
				score += 20
			}
			if pkg.HotelDistance <= 300 {
				score += 10
			}
		}
		if advanceFilter["family_friendly"] {
			if pkg.Travel.Name == "Taiba Medina" {
				score += 25
			}
			if pkg.IsKidFriendly || pkg.IsSeniorFriendly {
				score += 15
			}
			if pkg.IsDirect {
				score += 10
			}
			if pkg.IsFamily || pkg.IsSenior {
				score += 10
			}
			if pkg.HotelDistance <= 400 {
				score += 10
			} else {
				score += 5
			}
		}

		if pkg.SunnahScore >= 8 {
			score += 5
		}
		if pkg.HotelDistance <= 500 {
			score += 5
		}
		if pkg.Travel.Rating >= 4.5 {
			score += 5
		}

		scored = append(scored, ScoredPackage{
			Package:           pkg,
			Key:               PackageKey(pkg.Travel.Name, pkg.Name),
			Featured:          isFeatured(pkg.Travel.Name),
			Score:             score,
			FacilityList:      facilitiesOrGeneric(pkg.Facilities),
			FacilityGeneric:   len(parseFacilities(pkg.Facilities)) == 0,
			SameDetails:       sameDetails(pkg.Details),
			VaryingPrice:      varying,
			ShuttleMakkah:     shuttleNote(pkg.Details),
			ExcludeList:       parseFacilities(pkg.Excludes),
			DepartureDatesStr: formatDepartureDatesStr(pkg.Details),
			DistanceMakkah:    distMakkah,
			DistanceMadinah:   distMadinah,
		})
	}

	// Server-side sort
	sort.Slice(scored, func(i, j int) bool {
		// Paket promo partner selalu paling atas, apa pun urutan yang dipilih.
		if scored[i].Featured != scored[j].Featured {
			return scored[i].Featured
		}
		switch input.Sort {
		case "price_asc":
			return scored[i].Price < scored[j].Price
		case "price_desc":
			return scored[i].Price > scored[j].Price
		case "distance_asc":
			return scored[i].HotelDistance < scored[j].HotelDistance
		case "duration_asc":
			return scored[i].Duration < scored[j].Duration
		case "duration_desc":
			return scored[i].Duration > scored[j].Duration
		case "dp_asc":
			return scored[i].DownPayment < scored[j].DownPayment
		case "direct_only":
			return scored[i].Price < scored[j].Price
		default:
			if scored[i].Featured != scored[j].Featured {
				return scored[i].Featured
			}
			return scored[i].Score > scored[j].Score
		}
	})

	// Filter direct only
	if input.Sort == "direct_only" {
		filtered := make([]ScoredPackage, 0)
		for _, sp := range scored {
			if sp.IsDirect {
				filtered = append(filtered, sp)
			}
		}
		scored = filtered
	}

	total := len(scored)

	if input.Page < 1 {
		input.Page = 1
	}
	start := (input.Page - 1) * 5
	if start >= len(scored) {
		return []ScoredPackage{}, total
	}
	end := start + 5
	if end > len(scored) {
		end = len(scored)
	}

	return scored[start:end], total
}

// availablePrice: harga termurah di antara tanggal yang ditampilkan. d.Price 0
// berarti sama dengan harga paket (harga termurah semua tanggal saat import).
func availablePrice(pkgPrice int, details []models.DetailPackage) int {
	min := 0
	for _, d := range details {
		p := d.Price
		if p == 0 {
			p = pkgPrice
		}
		if min == 0 || p < min {
			min = p
		}
	}
	if min == 0 {
		return pkgPrice
	}
	return min
}

// Fasilitas yang umumnya termasuk di paket umroh reguler, dipakai bila travel
// belum merinci (ditampilkan dengan keterangan "umumnya termasuk").
var genericFacilities = []string{"Tiket pesawat PP", "Visa umroh", "Hotel Makkah & Madinah", "Bus AC selama di Saudi", "Makan 3x sehari", "Muthawif / pembimbing", "Air zamzam"}

func facilitiesOrGeneric(raw string) []string {
	if l := parseFacilities(raw); len(l) > 0 {
		return l
	}
	return genericFacilities
}

func shuttleNote(ds []models.DetailPackage) string {
	if len(ds) == 0 {
		return ""
	}
	if h, ok := LookupHotelGeo("makkah", ds[0].HotelMakkah); ok && h.Shuttle {
		if h.ShuttleNote != "" {
			return h.ShuttleNote
		}
		return "Ada shuttle hotel ke Masjidil Haram"
	}
	return ""
}

func sameDetails(ds []models.DetailPackage) bool {
	for _, d := range ds[1:] {
		a := ds[0]
		if d.HotelMakkah != a.HotelMakkah || d.HotelMadinah != a.HotelMadinah || d.StarsMakkah != a.StarsMakkah ||
			d.StarsMadinah != a.StarsMadinah || d.AddonTriple != a.AddonTriple || d.AddonDouble != a.AddonDouble ||
			d.AddonAssumed != a.AddonAssumed || d.RoomType != a.RoomType || d.Guide != a.Guide {
			return false
		}
	}
	return len(ds) > 0
}

func parseFacilities(raw string) []string {
	var list []string
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return []string{}
	}
	return list
}

func formatDepartureDatesStr(details []models.DetailPackage) string {
	if len(details) == 0 {
		return "-"
	}

	var dates []time.Time
	seen := make(map[string]bool)
	for _, d := range details {
		if d.DepartureDate == "" {
			continue
		}
		if !seen[d.DepartureDate] {
			seen[d.DepartureDate] = true
			if t, err := time.Parse("2006-01-02", d.DepartureDate); err == nil {
				dates = append(dates, t)
			}
		}
	}

	if len(dates) == 0 {
		return "-"
	}

	sort.Slice(dates, func(i, j int) bool {
		return dates[i].Before(dates[j])
	})

	months := map[time.Month]string{
		time.January: "Jan", time.February: "Feb", time.March: "Mar", time.April: "Apr",
		time.May: "Mei", time.June: "Jun", time.July: "Jul", time.August: "Agu",
		time.September: "Sep", time.October: "Okt", time.November: "Nov", time.December: "Des",
	}

	var parts []string
	type group struct {
		Month time.Month
		Year  int
		Days  []int
	}
	var groups []group

	for _, d := range dates {
		if len(groups) > 0 {
			last := &groups[len(groups)-1]
			if last.Month == d.Month() && last.Year == d.Year() {
				last.Days = append(last.Days, d.Day())
				continue
			}
		}
		groups = append(groups, group{Month: d.Month(), Year: d.Year(), Days: []int{d.Day()}})
	}

	for i, g := range groups {
		var daysStr []string
		for _, day := range g.Days {
			daysStr = append(daysStr, strconv.Itoa(day))
		}
		daysJoined := strings.Join(daysStr, ", ")

		str := fmt.Sprintf("%s %s", daysJoined, months[g.Month])
		if i == len(groups)-1 {
			str += fmt.Sprintf(" %d", g.Year)
		} else if groups[i].Year != groups[len(groups)-1].Year {
			str += fmt.Sprintf(" %d", g.Year)
		}
		parts = append(parts, str)
	}

	return strings.Join(parts, ", ")
}

const unknownDistance = 99999

func formatMeters(m int) string {
	switch {
	case m <= 0:
		return ""
	case m >= 1000:
		return strings.Replace(strconv.FormatFloat(float64(m)/1000, 'f', -1, 64), ".", ",", 1) + " km"
	default:
		return strconv.Itoa(m) + "m"
	}
}

func getDistanceMakkah(hotel string) string {
	h := strings.ToLower(hotel)
	if strings.TrimSpace(h) == "" {
		return ""
	}
	if strings.Contains(h, "marwa rotana") {
		return "250m"
	}
	if strings.Contains(h, "zamzam") {
		return "150-250m"
	}
	if strings.Contains(h, "azka") || strings.Contains(h, "safa") {
		return "250-350m"
	}
	if strings.Contains(h, "anjum") {
		return "450-550m"
	}
	if strings.Contains(h, "maysan") || strings.Contains(h, "almaqam") || strings.Contains(h, "al maqam") {
		return "550-600m"
	}
	if strings.Contains(h, "shohada") {
		return "600-700m"
	}
	if strings.Contains(h, "almassa grand") || strings.Contains(h, "grand al masa") || strings.Contains(h, "al massa grand") || strings.Contains(h, "ramada dar") || strings.Contains(h, "bader al massa") {
		return "700m"
	}
	if strings.Contains(h, "al massa dar") || strings.Contains(h, "faiezeen") || strings.Contains(h, "fayzeen") || strings.Contains(h, "dar faizin") {
		return "750m"
	}
	if strings.Contains(h, "jada ajyad") {
		return "750-900m"
	}
	if strings.Contains(h, "nada ajyad") {
		return "800-950m"
	}
	if strings.Contains(h, "mashaer") {
		return "850-1000m"
	}
	if strings.Contains(h, "majestic") {
		return "900-1100m"
	}
	if strings.Contains(h, "le meridien tower") || strings.Contains(h, "meridien tower") {
		return "1,5 km"
	}
	if strings.Contains(h, "ibis styles") || strings.Contains(h, "ibis style") {
		return "2,5 km"
	}
	if strings.Contains(h, "fajr badea") {
		return "2 km"
	}
	return "" // hotel tidak dikenal: jangan mengarang jarak
}

func getDistanceMadinah(hotel string) string {
	h := strings.ToLower(hotel)
	if strings.TrimSpace(h) == "" {
		return ""
	}
	if strings.Contains(h, "maden taibah") || strings.Contains(h, "madinah taibah") {
		return "200m"
	}
	if strings.Contains(h, "concorde") || strings.Contains(h, "dar al khair") {
		return "120m"
	}
	if strings.Contains(h, "shaza") {
		return "180m"
	}
	if strings.Contains(h, "plaza inn") {
		return "250m"
	}
	if strings.Contains(h, "ritz") {
		return "300m"
	}
	if strings.Contains(h, "odst") {
		return "350m"
	}
	// Dicek langsung oleh pemilik situs (100-200 m), sama dengan flyer Umrah Bisa (±150 m).
	if strings.Contains(h, "jauharat") || strings.Contains(h, "jawharat") || strings.Contains(h, "rasheed") || strings.Contains(h, "rashed") {
		return "150m"
	}
	if strings.Contains(h, "arkan") || strings.Contains(h, "manar") {
		return "500m"
	}
	if strings.Contains(h, "al ansar golden") || strings.Contains(h, "golden tulip") {
		return "600m"
	}
	if strings.Contains(h, "almukhtaro") || strings.Contains(h, "al mukhtara") || strings.Contains(h, "alghorbi") || strings.Contains(h, "al gharbi") {
		return "1,2 km"
	}
	// Dicek pemilik situs di Google Maps (jalan kaki 350 m / 5 menit ke tepi pelataran).
	if strings.Contains(h, "sham province") || strings.Contains(h, "province al sham") {
		return "350m"
	}
	if strings.Contains(h, "sky view") {
		return "1,5 km"
	}
	return "" // hotel tidak dikenal: jangan mengarang jarak
}

func parseMinDistance(dist string) int {
	if strings.Contains(dist, "km") {
		d := strings.ReplaceAll(dist, "km", "")
		d = strings.ReplaceAll(d, ",", ".")
		d = strings.TrimSpace(d)
		val, _ := strconv.ParseFloat(d, 64)
		return int(val * 1000)
	}
	d := strings.Split(dist, "-")[0]
	d = strings.ReplaceAll(d, "m", "")
	d = strings.TrimSpace(d)
	val, _ := strconv.Atoi(d)
	if val == 0 {
		return 1000
	}
	return val
}
