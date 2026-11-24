package demo

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"umrah/app/qris"

	"gorm.io/gorm"
)

// Market Tanah Suci ala GoFood (demo): jamaah memilih toko terdekat di Madinah/Makkah, memesan
// barang untuk diambil, lalu membayar dengan QRIS dinamis dari aplikasi pembayaran apa pun.
//
// Prinsip: umrohku TIDAK menyimpan saldo. QR diterbitkan (di produk nyata) oleh mitra PJP berizin BI;
// di demo QR-nya berformat QRIS asli tetapi merchant & penyelenggara fiktif, jadi aplikasi sungguhan
// akan menolaknya. Toko, harga, dan lokasi adalah contoh. Lihat PLAN_MARKET_TANAH_SUCI.md.

// AST: waktu Saudi (UTC+3) untuk jam buka toko.
var AST = time.FixedZone("AST", 3*3600)

const (
	MarketQRValid     = 15 * time.Minute
	MarketMaxQty      = 20
	MarketMaxLines    = 30
	MarketWalkMPerMin = 75.0  // ±4,5 km/jam
	MarketCarMPerMin  = 400.0 // ±24 km/jam di jalan padat sekitar masjid
)

type City struct {
	Key, Name, Mosque, MosqueShort string
	Lat, Lng                       float64
}

var Cities = []City{
	{Key: "madinah", Name: "Madinah", Mosque: "Masjid Nabawi", MosqueShort: "Nabawi", Lat: 24.46720, Lng: 39.61110},
	{Key: "makkah", Name: "Makkah", Mosque: "Masjidil Haram", MosqueShort: "Haram", Lat: 21.42250, Lng: 39.82620},
}

func CityByKey(k string) City {
	for _, c := range Cities {
		if c.Key == k {
			return c
		}
	}
	return Cities[0]
}

type MarketCat struct{ Key, Label, Tone string }

// MarketCats: urutan chip kategori. Tone = kelas latar ilustrasi.
var MarketCats = []MarketCat{
	{"kurma", "Kurma", "bg-amber-100"}, {"cokelat", "Cokelat", "bg-orange-100"}, {"parfum", "Parfum", "bg-violet-100"},
	{"abaya", "Abaya", "bg-slate-200"}, {"sajadah", "Sajadah", "bg-rose-100"}, {"tasbih", "Tasbih & peci", "bg-emerald-100"},
	{"kacang", "Kacang", "bg-yellow-100"}, {"kuliner", "Kuliner RI", "bg-red-100"}, {"paket", "Paket oleh-oleh", "bg-sky-100"},
}

func MarketCatByKey(k string) *MarketCat {
	for i := range MarketCats {
		if MarketCats[i].Key == k {
			return &MarketCats[i]
		}
	}
	return nil
}

func CatTone(k string) string {
	if c := MarketCatByKey(k); c != nil {
		return c.Tone
	}
	return "bg-gray-100"
}

// Item: barang di satu toko (harga SAR).
type Item struct {
	Key, Name, Desc, Cat string
	SAR                  int64
	Badge                string // mis. "Terlaris"
}

func (i Item) IDR() int64 { return i.SAR * SARRateDemo }

func it(key, cat, name, desc string, sar int64, badge ...string) Item {
	b := ""
	if len(badge) > 0 {
		b = badge[0]
	}
	return Item{Key: key, Cat: cat, Name: name, Desc: desc, SAR: sar, Badge: b}
}

var (
	kurmaItems = []Item{
		it("ajwa-500", "kurma", "Kurma Ajwa Madinah", "Kotak 500 g, khas Madinah", 45, "Terlaris"),
		it("ajwa-1kg", "kurma", "Kurma Ajwa Premium", "Kotak 1 kg, ukuran besar", 85),
		it("sukari-1kg", "kurma", "Kurma Sukari", "Manis legit, kotak 1 kg", 35),
		it("medjool-1kg", "kurma", "Kurma Medjool", "Besar & lembut, kotak 1 kg", 60),
		it("khalas-1kg", "kurma", "Kurma Khalas", "Rasa karamel, kotak 1 kg", 30),
	}
	cokelatItems = []Item{
		it("cok-kurma", "cokelat", "Cokelat kurma isi almond", "Kotak 250 g, 20 butir", 30, "Favorit"),
		it("cok-pistachio", "cokelat", "Cokelat pistachio", "Batang 200 g", 22),
		it("cok-mix", "cokelat", "Cokelat Arab aneka rasa", "Toples 500 g, cocok dibagi", 40),
	}
	kacangItems = []Item{
		it("kacang-mix", "kacang", "Kacang Arab campur", "Pistachio, almond, kacang Arab 500 g", 28),
		it("kismis", "kacang", "Kismis hitam", "Pak 500 g", 15),
	}
	paketItems = []Item{
		it("paket-10", "paket", "Paket oleh-oleh 10 orang", "10 kantong: kurma 250 g + tasbih + cokelat", 150, "Hemat"),
		it("paket-keluarga", "paket", "Paket keluarga", "Ajwa 1 kg, cokelat kurma, kacang, sajadah travel", 160),
	}
	sajadahItems = []Item{
		it("sajadah-travel", "sajadah", "Sajadah travel lipat", "Tipis, muat di tas kabin", 25, "Terlaris"),
		it("sajadah-tebal", "sajadah", "Sajadah tebal Turki", "Beludru tebal, motif mihrab", 55),
		it("sajadah-anak", "sajadah", "Sajadah anak", "Ukuran kecil, motif ceria", 18),
	}
	tasbihItems = []Item{
		it("tasbih-kayu", "tasbih", "Tasbih kayu 99 butir", "Kayu kokka", 10),
		it("tasbih-kristal", "tasbih", "Tasbih kristal 33 butir", "Dengan kotak hadiah", 18),
		it("tasbih-lusin", "tasbih", "Tasbih plastik 1 lusin", "Untuk dibagikan", 12, "Hemat"),
		it("peci-rajut", "tasbih", "Peci rajut putih", "Ukuran bebas, adem", 8),
	}
	parfumItems = []Item{
		it("parfum-misk", "parfum", "Parfum misk putih", "Roll-on 6 ml, non-alkohol", 20, "Terlaris"),
		it("parfum-oud", "parfum", "Minyak oud", "Botol 12 ml, non-alkohol", 65),
		it("bakhoor", "parfum", "Bakhoor & tungku kecil", "Set wangi ruangan", 45),
		it("parfum-set", "parfum", "Set parfum 6 botol", "Aneka aroma, cocok untuk oleh-oleh", 75),
	}
	abayaItems = []Item{
		it("abaya-hitam", "abaya", "Abaya hitam klasik", "Bahan jetblack, ukuran S–XXL", 120, "Terlaris"),
		it("abaya-bordir", "abaya", "Abaya bordir lengan", "Bordir emas halus", 180),
		it("gamis-pria", "abaya", "Gamis pria putih", "Bahan adem, ukuran M–XXL", 90),
		it("khimar", "abaya", "Khimar instan", "Panjang menutup dada", 35),
	}
	kulinerItems = []Item{
		it("nasi-rendang", "kuliner", "Nasi rendang", "Nasi, rendang sapi, sayur nangka, sambal ijo", 22, "Terlaris"),
		it("nasi-ayam", "kuliner", "Nasi ayam penyet", "Ayam goreng, sambal terasi, lalapan", 18),
		it("bakso", "kuliner", "Bakso urat", "Kuah kaldu sapi, mi, sawi", 16),
		it("soto", "kuliner", "Soto ayam", "Lengkap dengan nasi", 15),
		it("teh-tarik", "kuliner", "Teh tarik", "Panas atau dingin", 6),
	}
)

func pick(groups ...[]Item) []Item {
	var out []Item
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

// StoreItems: menu per toko (slug toko -> barang).
var StoreItems = map[string][]Item{
	"madinah-markaziyah": pick(kurmaItems, cokelatItems, kacangItems, paketItems),
	"madinah-quba":       pick(kurmaItems[:3], sajadahItems[:2], tasbihItems, paketItems[:1]),
	"madinah-babsalam":   pick(sajadahItems, tasbihItems, parfumItems[:1]),
	"madinah-abaya":      abayaItems,
	"madinah-anshar":     pick(parfumItems, cokelatItems[:1]),
	"madinah-nusantara":  kulinerItems,
	"makkah-ajyad":       pick(kurmaItems, cokelatItems, kacangItems, paketItems),
	"makkah-misfalah":    pick(sajadahItems, tasbihItems, kurmaItems[:2], paketItems[:1]),
	"makkah-jabalomar":   pick(abayaItems, parfumItems[:1]),
	"makkah-ibrahim":     pick(parfumItems, tasbihItems[:2], sajadahItems[:1]),
	"makkah-syisyah":     kulinerItems,
}

func StoreItem(slug, key string) *Item {
	for _, i := range StoreItems[slug] {
		if i.Key == key {
			return &i
		}
	}
	return nil
}

// ItemGroup: menu toko per kategori.
type ItemGroup struct {
	Cat   MarketCat
	Items []Item
}

func StoreMenu(slug string) []ItemGroup {
	var out []ItemGroup
	for _, c := range MarketCats {
		var g []Item
		for _, i := range StoreItems[slug] {
			if i.Cat == c.Key {
				g = append(g, i)
			}
		}
		if len(g) > 0 {
			out = append(out, ItemGroup{Cat: c, Items: g})
		}
	}
	return out
}

// ---- Jarak & tampilan toko ----

// DistanceM: jarak garis lurus (haversine), meter.
func DistanceM(lat1, lng1, lat2, lng2 float64) float64 {
	const r = 6371000.0
	p1, p2 := lat1*math.Pi/180, lat2*math.Pi/180
	dp, dl := (lat2-lat1)*math.Pi/180, (lng2-lng1)*math.Pi/180
	a := math.Sin(dp/2)*math.Sin(dp/2) + math.Cos(p1)*math.Cos(p2)*math.Sin(dl/2)*math.Sin(dl/2)
	return 2 * r * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// DistLabel: "450 m · ±7 mnt jalan" / "3,4 km · ±9 mnt naik mobil". Jarak jalan ≈ 1,3× garis lurus.
func DistLabel(m float64) string {
	route := m * 1.3
	d := fmt.Sprintf("%d m", int(math.Round(m/10)*10))
	if m >= 1000 {
		d = strings.Replace(fmt.Sprintf("%.1f km", m/1000), ".", ",", 1)
	}
	if route <= 2000 {
		return fmt.Sprintf("%s · ±%d mnt jalan", d, max(1, int(math.Ceil(route/MarketWalkMPerMin))))
	}
	return fmt.Sprintf("%s · ±%d mnt naik mobil", d, max(1, int(math.Ceil(route/MarketCarMPerMin))))
}

func (m Merchant) CatList() []string { return strings.Split(m.Cats, ",") }

// Display: nama toko tanpa akhiran "(contoh)" untuk kartu; label "Toko contoh" tampil terpisah.
func (m Merchant) Display() string { return strings.TrimSuffix(m.Name, " (contoh)") }
func (m Merchant) MainCat() string { return m.CatList()[0] }
func (m Merchant) Tone() string    { return CatTone(m.MainCat()) }
func (m Merchant) CityKey() string { return strings.ToLower(m.City) }
func (m Merchant) HasCat(k string) bool {
	for _, c := range m.CatList() {
		if c == k {
			return true
		}
	}
	return false
}
func (m Merchant) CatLabels() string {
	var l []string
	for _, c := range m.CatList() {
		if mc := MarketCatByKey(c); mc != nil {
			l = append(l, mc.Label)
		}
	}
	return strings.Join(l, " · ")
}
func (m Merchant) RatingLabel() string {
	return strings.Replace(fmt.Sprintf("%.1f", m.Rating), ".", ",", 1)
}
func (m Merchant) ReviewsLabel() string {
	if m.Reviews >= 1000 {
		return strings.Replace(fmt.Sprintf("%.1f rb", float64(m.Reviews)/1000), ".", ",", 1)
	}
	return strconv.Itoa(m.Reviews)
}

// MapsDir: petunjuk arah Google Maps (jalan kaki) ke toko.
func (m Merchant) MapsDir() string {
	return fmt.Sprintf("https://www.google.com/maps/dir/?api=1&destination=%.5f,%.5f&travelmode=walking", m.Lat, m.Lng)
}

// MapsEmbed: peta Google tanpa kunci API.
func (m Merchant) MapsEmbed() string {
	return fmt.Sprintf("https://www.google.com/maps/embed?origin=mfe&pb=!1m3!2m1!1s%.5f,%.5f!6i17!3m1!1sid!5m1!1sid", m.Lat, m.Lng)
}

// OpenNow: buka sekarang (waktu Saudi). Format jam "08.00–23.30"; "24.00" = tengah malam.
func (m Merchant) OpenNow() bool {
	parts := strings.Split(m.Hours, "–")
	if len(parts) != 2 {
		return true
	}
	mins := func(s string) int {
		var h, mm int
		fmt.Sscanf(strings.TrimSpace(s), "%d.%d", &h, &mm)
		return h*60 + mm
	}
	now := time.Now().In(AST)
	cur := now.Hour()*60 + now.Minute()
	o, c := mins(parts[0]), mins(parts[1])
	if c <= o { // lewat tengah malam
		return cur >= o || cur < c
	}
	return cur >= o && cur < c
}

// MinSAR: harga termurah di toko.
func (m Merchant) MinSAR() int64 {
	var mn int64
	for _, i := range StoreItems[m.Slug] {
		if mn == 0 || i.SAR < mn {
			mn = i.SAR
		}
	}
	return mn
}

type StoreView struct {
	*Merchant
	Dist  float64
	DistL string
}

// DistA: bagian jarak ("210 m"); DistB: waktu tempuh ("±4 mnt jalan").
func (s StoreView) DistA() string { return strings.SplitN(s.DistL, " · ", 2)[0] }
func (s StoreView) DistB() string {
	if p := strings.SplitN(s.DistL, " · ", 2); len(p) == 2 {
		return p[1]
	}
	return ""
}

// MarketStores: toko di kota, diurutkan dari yang terdekat ke masjid. cat/q opsional.
func MarketStores(city City, cat, q string) []StoreView {
	q = strings.ToLower(strings.TrimSpace(q))
	var out []StoreView
	for i := range Merchants {
		m := &Merchants[i]
		if m.CityKey() != city.Key || len(StoreItems[m.Slug]) == 0 {
			continue
		}
		if cat != "" && !m.HasCat(cat) {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(m.Name+" "+m.Tagline+" "+m.CatLabels()), q) && !storeHasItem(m.Slug, q) {
			continue
		}
		d := DistanceM(city.Lat, city.Lng, m.Lat, m.Lng)
		out = append(out, StoreView{Merchant: m, Dist: d, DistL: DistLabel(d)})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Dist < out[j].Dist })
	return out
}

func storeHasItem(slug, q string) bool {
	for _, i := range StoreItems[slug] {
		if strings.Contains(strings.ToLower(i.Name+" "+i.Desc), q) {
			return true
		}
	}
	return false
}

type ItemView struct {
	Item
	Store StoreView
}

// MarketPicks: barang dari toko-toko di kota (yang berlabel dulu), maks n.
func MarketPicks(city City, cat, q string, n int) []ItemView {
	q = strings.ToLower(strings.TrimSpace(q))
	var out []ItemView
	seen := map[string]bool{}
	for _, s := range MarketStores(city, cat, "") {
		for _, i := range StoreItems[s.Slug] {
			if cat != "" && i.Cat != cat {
				continue
			}
			if q != "" && !strings.Contains(strings.ToLower(i.Name+" "+i.Desc), q) {
				continue
			}
			if seen[i.Name] { // satu barang sama cukup sekali (dari toko terdekat)
				continue
			}
			seen[i.Name] = true
			out = append(out, ItemView{Item: i, Store: s})
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Badge != "" && out[b].Badge == "" })
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// ---- Pesanan ----

type MarketOrder struct {
	ID           uint   `gorm:"primaryKey"`
	Code         string `gorm:"size:16;uniqueIndex"` // MKT-XXXXXX
	UserID       uint   `gorm:"index"`
	MerchantSlug string `gorm:"size:40;index"`
	Items        string `gorm:"type:text"` // []Line, harga SAR
	TotalSAR     int64
	TotalIDR     int64
	Rate         int64
	Name         string `gorm:"size:80"`
	Slot         string `gorm:"size:20"` // segera | 1jam | isya
	Note         string `gorm:"size:200"`
	Status       string `gorm:"size:20;index"` // menunggu_bayar | dibayar | siap | diambil | kedaluwarsa
	QRPayload    string `gorm:"size:400"`
	PickupCode   string `gorm:"size:6"`
	PaidVia      string `gorm:"size:80"`
	QRRef        string `gorm:"size:24"` // kode QRPayment bila dibayar lewat QRIS umrohku
	ExpiresAt    time.Time
	PaidAt       *time.Time
	ReadyAt      *time.Time
	DoneAt       *time.Time
	CreatedAt    time.Time `gorm:"index"`
}

func init() { RegisterModels(&MarketOrder{}) }

func (o MarketOrder) Lines() []Line       { return parseLines(o.Items) }
func (o MarketOrder) Merchant() *Merchant { return MerchantBySlug(o.MerchantSlug) }
func (o MarketOrder) Waiting() bool       { return o.Status == "menunggu_bayar" }
func (o MarketOrder) Paid() bool          { return o.PaidAt != nil }
func (o MarketOrder) SecondsLeft() int    { return max(0, int(time.Until(o.ExpiresAt).Seconds())) }
func (o MarketOrder) ItemCount() (n int) {
	for _, l := range o.Lines() {
		n += l.Qty
	}
	return
}

var MarketSlots = []struct{ Key, Label string }{
	{"segera", "Secepatnya (±20 menit)"}, {"1jam", "1 jam lagi"}, {"isya", "Setelah salat Isya"},
}

func (o MarketOrder) SlotLabel() string {
	for _, s := range MarketSlots {
		if s.Key == o.Slot {
			return s.Label
		}
	}
	return o.Slot
}

func (o MarketOrder) StatusText() string {
	switch o.Status {
	case "menunggu_bayar":
		return "Menunggu pembayaran"
	case "dibayar":
		return "Sedang disiapkan toko"
	case "siap":
		return "Siap diambil"
	case "diambil":
		return "Sudah diambil"
	case "kedaluwarsa":
		return "QR kedaluwarsa"
	}
	return o.Status
}

type MarketStep struct {
	Label string
	At    *time.Time
	Done  bool
}

func (o MarketOrder) Steps() []MarketStep {
	c := o.CreatedAt
	return []MarketStep{
		{"Pesanan dibuat", &c, true},
		{"Dibayar", o.PaidAt, o.PaidAt != nil},
		{"Siap diambil", o.ReadyAt, o.ReadyAt != nil},
		{"Diambil", o.DoneAt, o.DoneAt != nil},
	}
}

var (
	ErrMarketEmpty   = errors.New("pilih minimal satu barang")
	ErrMarketStore   = errors.New("toko tidak ditemukan")
	ErrMarketName    = errors.New("isi nama pengambil")
	ErrMarketState   = errors.New("status pesanan sudah berubah")
	ErrMarketCode    = errors.New("kode ambil tidak cocok")
	ErrMarketExpired = errors.New("QR sudah kedaluwarsa, buat QR baru")
)

// MarketLines: baris pesanan dari qty per barang (barang tak dikenal diabaikan).
func MarketLines(slug string, qty map[string]int) ([]Line, int64) {
	var lines []Line
	var total int64
	for _, i := range StoreItems[slug] { // urutan menu toko
		q := qty[i.Key]
		if q <= 0 {
			continue
		}
		q = min(q, MarketMaxQty)
		lines = append(lines, Line{Key: i.Key, Name: i.Name, Unit: i.Cat, Price: i.SAR, Qty: q})
		total += i.SAR * int64(q)
	}
	if len(lines) > MarketMaxLines {
		lines = lines[:MarketMaxLines]
	}
	return lines, total
}

func CreateMarketOrder(userID uint, slug, name, slot, note string, qty map[string]int) (*MarketOrder, error) {
	m := MerchantBySlug(slug)
	if m == nil || len(StoreItems[slug]) == 0 {
		return nil, ErrMarketStore
	}
	lines, sar := MarketLines(slug, qty)
	if len(lines) == 0 {
		return nil, ErrMarketEmpty
	}
	name = clip(strings.TrimSpace(name), 80)
	if name == "" {
		return nil, ErrMarketName
	}
	okSlot := false
	for _, s := range MarketSlots {
		okSlot = okSlot || s.Key == slot
	}
	if !okSlot {
		slot = "segera"
	}
	now := time.Now()
	o := &MarketOrder{Code: uniqueCode(&MarketOrder{}, func() string { return "MKT-" + randCode(6) }), UserID: userID, MerchantSlug: slug,
		Items: encodeLines(lines), TotalSAR: sar, Rate: SARRateDemo, TotalIDR: sar * SARRateDemo, Name: name, Slot: slot,
		Note: clip(strings.TrimSpace(note), 200), Status: "menunggu_bayar", PickupCode: pickupDigits(), ExpiresAt: now.Add(MarketQRValid), CreatedAt: now}
	o.QRPayload = MarketQR(o, m)
	if err := DB.Create(o).Error; err != nil {
		return nil, err
	}
	return o, nil
}

func pickupDigits() string {
	const d = "0123456789"
	s := randCode(6)
	b := []byte(s)
	for i := range b {
		b[i] = d[int(b[i])%10]
	}
	return string(b)
}

// MarketQR: QRIS dinamis (format EMVCo MPM + CRC) untuk tagihan pesanan. Kode pesanan di tag 62
// (bill number & reference) supaya pembayaran bisa dicocokkan. Penyelenggara & NMID FIKTIF.
func MarketQR(o *MarketOrder, m *Merchant) string {
	return merchantQR(m, o.Code, o.TotalIDR, "UMROHKU")
}

// merchantQR: QRIS dinamis atas nama toko (dipakai pesanan pickup & kasir). terminal = tag 62.07.
func merchantQR(m *Merchant, code string, idr int64, terminal string) string {
	t := func(tag, v string) string { return fmt.Sprintf("%s%02d%s", tag, len([]rune(v)), v) }
	mid := strings.ToUpper(strings.ReplaceAll(m.Slug, "-", ""))
	if len(mid) > 15 {
		mid = mid[:15]
	}
	acq := t("00", "ID.CO.PJPCONTOH.WWW") + t("01", "936099990000067890") + t("02", mid) + t("03", "UKE")
	nmid := t("00", "ID.CO.QRIS.WWW") + t("02", "ID0000DEMOKSA"+fmt.Sprintf("%02d", merchantIndex(m.Slug))) + t("03", "UKE")
	name := m.Short
	if len(name) > 25 {
		name = name[:25]
	}
	body := t("00", "01") + t("01", "12") + t("26", acq) + t("51", nmid) + t("52", m.MCC) + t("53", "360") +
		t("54", strconv.FormatInt(idr, 10)) + t("58", "ID") + t("59", name) + t("60", strings.ToUpper(m.City)) +
		t("62", t("01", code)+t("05", code)+t("07", terminal))
	return qris.WithCRC(body)
}

func merchantIndex(slug string) int {
	for i, m := range Merchants {
		if m.Slug == slug {
			return i + 1
		}
	}
	return 0
}

// expireMarket: tandai kedaluwarsa bila QR lewat waktu (dipanggil saat membaca).
func expireMarket(o *MarketOrder) {
	if o != nil && o.Status == "menunggu_bayar" && time.Now().After(o.ExpiresAt) {
		if DB.Model(&MarketOrder{}).Where("id = ? AND status = ?", o.ID, "menunggu_bayar").Update("status", "kedaluwarsa").RowsAffected > 0 {
			o.Status = "kedaluwarsa"
		}
	}
}

func MarketOrderFor(userID uint, code string) *MarketOrder {
	var o MarketOrder
	if DB.Where("user_id = ? AND code = ?", userID, code).Take(&o).Error != nil {
		return nil
	}
	expireMarket(&o)
	return &o
}

func MarketOrderByCode(code string) *MarketOrder {
	var o MarketOrder
	if DB.Where("code = ?", code).Take(&o).Error != nil {
		return nil
	}
	expireMarket(&o)
	return &o
}

func MarketOrdersFor(userID uint, limit int) []MarketOrder {
	var os []MarketOrder
	DB.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&os)
	for i := range os {
		expireMarket(&os[i])
	}
	return os
}

// ActiveMarketOrders: pesanan yang belum selesai (menunggu bayar yang masih berlaku, dibayar, siap).
func ActiveMarketOrders(userID uint) []MarketOrder {
	var out []MarketOrder
	for _, o := range MarketOrdersFor(userID, 30) {
		if o.Status == "menunggu_bayar" || o.Status == "dibayar" || o.Status == "siap" {
			out = append(out, o)
		}
	}
	return out
}

func MerchantMarketOrders(slug string) []MarketOrder {
	var os []MarketOrder
	DB.Where("merchant_slug = ? AND status IN ?", slug, []string{"dibayar", "siap", "diambil"}).Order("created_at DESC").Limit(100).Find(&os)
	return os
}

// RenewMarketQR: QR baru untuk pesanan kedaluwarsa (15 menit lagi).
func RenewMarketQR(o *MarketOrder) error {
	if o.Status != "kedaluwarsa" && !(o.Status == "menunggu_bayar" && time.Now().After(o.ExpiresAt)) {
		return ErrMarketState
	}
	exp := time.Now().Add(MarketQRValid)
	if DB.Model(&MarketOrder{}).Where("id = ? AND status IN ?", o.ID, []string{"kedaluwarsa", "menunggu_bayar"}).
		Updates(map[string]any{"status": "menunggu_bayar", "expires_at": exp}).RowsAffected == 0 {
		return ErrMarketState
	}
	o.Status, o.ExpiresAt = "menunggu_bayar", exp
	return nil
}

// PayMarketOrder: tandai dibayar (dari aplikasi lain = simulasi, atau dari QRIS umrohku).
func PayMarketOrder(o *MarketOrder, via, qrRef string) error {
	expireMarket(o)
	if o.Status == "kedaluwarsa" {
		return ErrMarketExpired
	}
	now := time.Now()
	res := DB.Model(&MarketOrder{}).Where("id = ? AND status = ?", o.ID, "menunggu_bayar").
		Updates(map[string]any{"status": "dibayar", "paid_at": now, "paid_via": clip(via, 80), "qr_ref": qrRef})
	if res.RowsAffected == 0 {
		return ErrMarketState
	}
	o.Status, o.PaidAt, o.PaidVia, o.QRRef = "dibayar", &now, via, qrRef
	m := o.Merchant()
	Notify(UserAudience(o.UserID), "wa", o.Name, "Pembayaran pesanan "+o.Code+" diterima",
		fmt.Sprintf("Rp %s (SAR %d) via %s. %s sedang menyiapkan pesananmu. Kode ambil: %s. Simulasi demo, tidak ada dana yang berpindah.",
			Rp(o.TotalIDR), o.TotalSAR, via, m.Name, o.PickupCode), "/market/pesanan/"+o.Code)
	return nil
}

// marketPaidByQR: dipanggil setelah PayQR berhasil. Bila QR adalah tagihan pesanan market dengan
// nominal yang sama, pesanan otomatis dibayar.
func marketPaidByQR(q *qris.QR, p *QRPayment) {
	ref := q.Reference
	if ref == "" {
		ref = q.BillNumber
	}
	if !strings.HasPrefix(ref, "MKT-") {
		return
	}
	o := MarketOrderByCode(ref)
	if o == nil || o.TotalIDR != q.Amount || o.QRPayload != q.Payload {
		return
	}
	_ = PayMarketOrder(o, "QRIS umrohku · "+p.Source, p.Code)
}

// MarketOrderByQRRef: pesanan yang dibayar dengan pembayaran QRIS ini (untuk tautan di struk).
func MarketOrderByQRRef(qrCode string) *MarketOrder {
	var o MarketOrder
	if qrCode == "" || DB.Where("qr_ref = ?", qrCode).Take(&o).Error != nil {
		return nil
	}
	return &o
}

// AdvanceMarketOrder: toko menandai siap, lalu menyerahkan (wajib kode ambil yang cocok).
func AdvanceMarketOrder(slug, code, to, pickup string) (*MarketOrder, error) {
	var o MarketOrder
	if DB.Where("merchant_slug = ? AND code = ?", slug, code).Take(&o).Error != nil {
		return nil, ErrMarketState
	}
	now := time.Now()
	var res *gorm.DB
	switch to {
	case "siap":
		res = DB.Model(&MarketOrder{}).Where("id = ? AND status = ?", o.ID, "dibayar").Updates(map[string]any{"status": "siap", "ready_at": now})
	case "diambil":
		if strings.TrimSpace(pickup) != o.PickupCode {
			return &o, ErrMarketCode
		}
		res = DB.Model(&MarketOrder{}).Where("id = ? AND status IN ?", o.ID, []string{"dibayar", "siap"}).
			Updates(map[string]any{"status": "diambil", "ready_at": gorm.Expr("COALESCE(ready_at, ?)", now), "done_at": now})
	default:
		return &o, ErrMarketState
	}
	if res.RowsAffected == 0 {
		return &o, ErrMarketState
	}
	DB.Where("id = ?", o.ID).Take(&o)
	m := o.Merchant()
	if to == "siap" {
		Notify(UserAudience(o.UserID), "wa", o.Name, "Pesanan "+o.Code+" siap diambil",
			fmt.Sprintf("Pesananmu di %s sudah siap. Tunjukkan kode ambil %s ke kasir. Lokasi: %s.", m.Name, o.PickupCode, m.Landmark), "/market/pesanan/"+o.Code)
	}
	return &o, nil
}
