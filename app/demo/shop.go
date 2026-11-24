package demo

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"umrah/app/payments"

	"gorm.io/gorm"
)

// Fase 3 (umrah mandiri, prototipe) & Fase 4 (layanan di Tanah Suci, prototipe).
//
// Semua toko, pemasok, hotel, transport, dan muthawwif di sini FIKTIF. Pembayaran lewat
// payments.MockGateway; tidak ada uang sungguhan. Voucher prabayar hanya simulasi: di produk
// nyata akan diterbitkan bersama mitra penerbit berizin, bukan "saldo umrohku".

// ---------------------------------------------------------------------------------------------
// Toko mitra di Makkah/Madinah
// ---------------------------------------------------------------------------------------------

type Merchant struct {
	Slug, Name, City, Area string
	Hours                  string // jam buka (contoh)
	Landmark               string // patokan lokasi
	Walk                   string // jarak (perkiraan)

	// Market Tanah Suci (lihat market.go). Lokasi = perkiraan/ilustrasi, bukan alamat toko sungguhan.
	Short    string  // nama pendek untuk QRIS (maks 25 karakter)
	Lat, Lng float64 // koordinat ilustrasi
	Rating   float64
	Reviews  int
	Cats     string // kategori utama, dipisah koma (kategori pertama = gambar toko)
	MCC      string // kode kategori merchant untuk QR
	Tagline  string
	Prep     int // menit menyiapkan pesanan
}

// Merchants: Merchants[0] dipakai pengalih peran sebagai "Toko" bawaan. Semua FIKTIF.
var Merchants = []Merchant{
	{Slug: "makkah-ajyad", Name: "Toko Kurma Ajyad (contoh)", City: "Makkah", Area: "Ajyad", Hours: "08.00–24.00", Landmark: "Jalan Ajyad, sisi selatan Masjidil Haram", Walk: "±7 menit jalan kaki",
		Short: "Kurma Ajyad", Lat: 21.41905, Lng: 39.82695, Rating: 4.8, Reviews: 1240, Cats: "kurma,cokelat,kacang,paket", MCC: "5441", Tagline: "Kurma Ajwa, Sukari, dan cokelat kurma", Prep: 15},
	{Slug: "makkah-misfalah", Name: "Toko Oleh-oleh Misfalah (contoh)", City: "Makkah", Area: "Misfalah", Hours: "09.00–23.00", Landmark: "Kawasan Misfalah, dekat deretan hotel jamaah Asia", Walk: "±12 menit jalan kaki",
		Short: "Oleh-oleh Misfalah", Lat: 21.41520, Lng: 39.82380, Rating: 4.7, Reviews: 860, Cats: "sajadah,tasbih,kurma,paket", MCC: "5947", Tagline: "Sajadah, tasbih, peci, dan paket oleh-oleh", Prep: 20},
	{Slug: "madinah-markaziyah", Name: "Toko Kurma Markaziyah (contoh)", City: "Madinah", Area: "Markaziyah Utara", Hours: "08.00–23.30", Landmark: "Markaziyah Utara, arah pintu utara Masjid Nabawi", Walk: "±5 menit jalan kaki",
		Short: "Kurma Markaziyah", Lat: 24.47035, Lng: 39.61205, Rating: 4.9, Reviews: 2130, Cats: "kurma,cokelat,kacang,paket", MCC: "5441", Tagline: "Ajwa Madinah langsung dari kebun mitra", Prep: 15},
	{Slug: "madinah-quba", Name: "Toko Oleh-oleh Quba (contoh)", City: "Madinah", Area: "Quba", Hours: "07.00–22.00", Landmark: "Kawasan Quba, dekat Masjid Quba", Walk: "±15 menit naik mobil dari Nabawi",
		Short: "Oleh-oleh Quba", Lat: 24.43960, Lng: 39.61720, Rating: 4.6, Reviews: 540, Cats: "kurma,sajadah,tasbih,paket", MCC: "5947", Tagline: "Mampir setelah salat di Masjid Quba", Prep: 20},
	{Slug: "madinah-babsalam", Name: "Galeri Sajadah Bab As-Salam (contoh)", City: "Madinah", Area: "Bab As-Salam", Hours: "08.00–24.00", Landmark: "Sisi barat Masjid Nabawi, dekat Bab As-Salam", Walk: "±4 menit jalan kaki",
		Short: "Galeri Sajadah Salam", Lat: 24.46690, Lng: 39.60905, Rating: 4.8, Reviews: 970, Cats: "sajadah,tasbih,parfum", MCC: "5947", Tagline: "Sajadah tebal, sajadah travel, tasbih, dan peci", Prep: 15},
	{Slug: "madinah-abaya", Name: "Butik Abaya Thaybah (contoh)", City: "Madinah", Area: "Markaziyah Selatan", Hours: "09.00–23.30", Landmark: "Markaziyah Selatan, arah pelataran selatan Masjid Nabawi", Walk: "±6 menit jalan kaki",
		Short: "Butik Abaya Thaybah", Lat: 24.46440, Lng: 39.61010, Rating: 4.7, Reviews: 610, Cats: "abaya", MCC: "5621", Tagline: "Abaya, gamis, dan khimar ukuran Indonesia", Prep: 20},
	{Slug: "madinah-anshar", Name: "Parfum & Bakhoor Al-Anshar (contoh)", City: "Madinah", Area: "Markaziyah Timur", Hours: "09.00–24.00", Landmark: "Sisi timur Masjid Nabawi, dekat pintu Baqi", Walk: "±5 menit jalan kaki",
		Short: "Parfum Al-Anshar", Lat: 24.46840, Lng: 39.61500, Rating: 4.8, Reviews: 780, Cats: "parfum,cokelat", MCC: "5977", Tagline: "Parfum non-alkohol, oud, dan bakhoor", Prep: 10},
	{Slug: "madinah-nusantara", Name: "Dapur Nusantara Madinah (contoh)", City: "Madinah", Area: "Markaziyah Utara", Hours: "10.00–23.00", Landmark: "Lantai dasar deretan hotel jamaah Indonesia, utara Masjid Nabawi", Walk: "±7 menit jalan kaki",
		Short: "Dapur Nusantara MDN", Lat: 24.47160, Lng: 39.60940, Rating: 4.6, Reviews: 1520, Cats: "kuliner", MCC: "5812", Tagline: "Nasi padang, rendang, bakso, dan teh tarik", Prep: 20},
	{Slug: "makkah-jabalomar", Name: "Butik Abaya Jabal Omar (contoh)", City: "Makkah", Area: "Jabal Omar", Hours: "09.00–24.00", Landmark: "Kawasan Jabal Omar, sisi barat Masjidil Haram", Walk: "±8 menit jalan kaki",
		Short: "Abaya Jabal Omar", Lat: 21.42030, Lng: 39.82160, Rating: 4.7, Reviews: 430, Cats: "abaya,parfum", MCC: "5621", Tagline: "Abaya hitam klasik dan bordir", Prep: 20},
	{Slug: "makkah-ibrahim", Name: "Parfum Ibrahim Khalil (contoh)", City: "Makkah", Area: "Ibrahim Khalil", Hours: "08.00–24.00", Landmark: "Jalan Ibrahim Khalil, selatan Masjidil Haram", Walk: "±9 menit jalan kaki",
		Short: "Parfum Ibrahim Khalil", Lat: 21.41760, Lng: 39.82480, Rating: 4.8, Reviews: 690, Cats: "parfum,tasbih,sajadah", MCC: "5977", Tagline: "Oud, misk, dan minyak wangi non-alkohol", Prep: 10},
	{Slug: "makkah-syisyah", Name: "Warung Nusantara Syisyah (contoh)", City: "Makkah", Area: "Syisyah", Hours: "10.00–24.00", Landmark: "Kawasan Syisyah, dekat hotel transit jamaah Indonesia", Walk: "±12 menit naik mobil dari Haram",
		Short: "Warung Nusantara MKH", Lat: 21.43420, Lng: 39.84560, Rating: 4.6, Reviews: 1180, Cats: "kuliner", MCC: "5812", Tagline: "Masakan rumahan Indonesia, bisa dibungkus", Prep: 25},
}

func MerchantBySlug(s string) *Merchant {
	for i := range Merchants {
		if Merchants[i].Slug == s {
			return &Merchants[i]
		}
	}
	return nil
}

// ---------------------------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------------------------

// Line: satu baris keranjang/pesanan. Price dalam rupiah (oleh-oleh, mandiri) atau SAR (pickup).
type Line struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Emoji string `json:"emoji,omitempty"`
	Unit  string `json:"unit,omitempty"`
	Price int64  `json:"price"`
	Qty   int    `json:"qty"`
}

func (l Line) Sub() int64 { return l.Price * int64(l.Qty) }

func parseLines(s string) []Line {
	var l []Line
	_ = json.Unmarshal([]byte(s), &l)
	return l
}

func encodeLines(l []Line) string {
	b, _ := json.Marshal(l)
	return string(b)
}

// CartItem: keranjang per akun. Kind: "oleh" (oleh-oleh) | "mandiri".
type CartItem struct {
	ID     uint   `gorm:"primaryKey"`
	UserID uint   `gorm:"uniqueIndex:ux_cart"`
	Kind   string `gorm:"size:10;uniqueIndex:ux_cart"`
	Key    string `gorm:"size:40;uniqueIndex:ux_cart"`
	Qty    int
}

// ShopOrder: pesanan oleh-oleh dikirim ke rumah (dari pemasok di Indonesia).
type ShopOrder struct {
	ID          uint   `gorm:"primaryKey"`
	Code        string `gorm:"size:16;uniqueIndex"`
	UserID      uint   `gorm:"index"`
	Items       string `gorm:"type:text"`
	Subtotal    int64
	Shipping    int64
	Total       int64
	Recipient   string `gorm:"size:80"`
	Phone       string `gorm:"size:30"`
	Address     string `gorm:"size:300"`
	City        string `gorm:"size:60"`
	Postal      string `gorm:"size:10"`
	Status      string `gorm:"size:20;index"` // menunggu_bayar | dibayar | dikemas | dikirim | diterima
	QRPayload   string `gorm:"size:300"`
	Courier     string `gorm:"size:60"`
	Resi        string `gorm:"size:30"`
	PaidAt      *time.Time
	PackedAt    *time.Time
	ShippedAt   *time.Time
	DeliveredAt *time.Time
	CreatedAt   time.Time `gorm:"index"`
}

func (o ShopOrder) Lines() []Line { return parseLines(o.Items) }

// Voucher: voucher prabayar Tanah Suci. Dibeli dalam rupiah, bernilai SAR (kurs tetap saat beli).
type Voucher struct {
	ID         uint   `gorm:"primaryKey"`
	Code       string `gorm:"size:16;uniqueIndex"` // UMV-XXXX-XXXX
	UserID     uint   `gorm:"index"`
	AmountIDR  int64  // yang dibayar jamaah
	Rate       int64  // rupiah per 1 SAR saat beli
	ValueSAR   int64
	BalanceSAR int64
	Status     string `gorm:"size:20;index"` // menunggu_bayar | aktif | habis
	QRPayload  string `gorm:"size:300"`      // tagihan QRIS pembelian
	PaidAt     *time.Time
	ExpiresAt  time.Time
	CreatedAt  time.Time `gorm:"index"`
}

// Masked: kode disamarkan untuk daftar riwayat toko.
func (v Voucher) Masked() string { return MaskVoucher(v.Code) }

// UsedSAR: saldo yang sudah dipakai.
func (v Voucher) UsedSAR() int64 { return v.ValueSAR - v.BalanceSAR }

// BalanceIDR: sisa saldo dalam rupiah pada kurs saat beli.
func (v Voucher) BalanceIDR() int64 { return v.BalanceSAR * v.Rate }

// UsedPct: persen saldo terpakai (untuk bar).
func (v Voucher) UsedPct() int {
	if v.ValueSAR <= 0 {
		return 0
	}
	return int(v.UsedSAR() * 100 / v.ValueSAR)
}

func MaskVoucher(code string) string {
	if len(code) >= 9 {
		return code[:9] + "••••"
	}
	return code
}

type VoucherRedemption struct {
	ID           uint   `gorm:"primaryKey"`
	VoucherID    uint   `gorm:"index"`
	VoucherCode  string `gorm:"size:16"`
	MerchantSlug string `gorm:"size:40;index"`
	AmountSAR    int64
	BalanceAfter int64
	Note         string    `gorm:"size:120"`
	CreatedAt    time.Time `gorm:"index"`
}

func (r VoucherRedemption) MerchantName() string {
	if m := MerchantBySlug(r.MerchantSlug); m != nil {
		return m.Name
	}
	return r.MerchantSlug
}

func (r VoucherRedemption) Masked() string { return MaskVoucher(r.VoucherCode) }

// Pickup: barang dipesan dari Indonesia, diambil di toko mitra pada slot tertentu (harga SAR,
// dibayar di toko: tunai atau voucher prabayar).
type Pickup struct {
	ID           uint   `gorm:"primaryKey"`
	Code         string `gorm:"size:16;uniqueIndex"`
	UserID       uint   `gorm:"index"`
	Name         string `gorm:"size:80"` // nama pengambil
	MerchantSlug string `gorm:"size:40;index"`
	Items        string `gorm:"type:text"`
	TotalSAR     int64
	Slot         string `gorm:"size:16"`       // "2026-11-24 16:00"
	Status       string `gorm:"size:20;index"` // dijadwalkan | siap | diambil
	ReadyAt      *time.Time
	TakenAt      *time.Time
	CreatedAt    time.Time `gorm:"index"`
}

func (p Pickup) Lines() []Line { return parseLines(p.Items) }

func (p Pickup) Merchant() *Merchant { return MerchantBySlug(p.MerchantSlug) }

func (p Pickup) SlotLabel() string { return SlotLabel(p.Slot) }

// MandiriState: checklist umrah mandiri per akun.
type MandiriState struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"uniqueIndex"`
	Done      string `gorm:"type:text"` // kunci checklist, dipisah koma
	UpdatedAt time.Time
}

// MandiriOrder: pesanan prototipe hotel/transport/muthawwif (dummy, tidak diteruskan ke mana pun).
type MandiriOrder struct {
	ID         uint   `gorm:"primaryKey"`
	Code       string `gorm:"size:16;uniqueIndex"`
	UserID     uint   `gorm:"index"`
	Items      string `gorm:"type:text"`
	Total      int64
	Contact    string `gorm:"size:80"`
	Phone      string `gorm:"size:30"`
	DepartDate string `gorm:"size:10"`
	Status     string `gorm:"size:20"` // menunggu_bayar | dibayar
	QRPayload  string `gorm:"size:300"`
	PaidAt     *time.Time
	CreatedAt  time.Time `gorm:"index"`
}

func (o MandiriOrder) Lines() []Line { return parseLines(o.Items) }

func init() {
	RegisterModels(&CartItem{}, &ShopOrder{}, &Voucher{}, &VoucherRedemption{}, &Pickup{}, &MandiriState{}, &MandiriOrder{})
}

// ---------------------------------------------------------------------------------------------
// Katalog (didefinisikan di kode; semua contoh)
// ---------------------------------------------------------------------------------------------

const Supplier = "Pemasok contoh, Indonesia"

type Product struct {
	Key, Name, Desc, Emoji, Unit string
	Price                        int64 // rupiah (oleh-oleh, mandiri) atau SAR (barang toko)
	Kind                         string
}

// OlehCatalog: oleh-oleh dikirim ke rumah di Indonesia (harga rupiah, termasuk kemasan).
var OlehCatalog = []Product{
	{Key: "kurma-ajwa", Name: "Kurma Ajwa", Desc: "Kurma khas Madinah, kemasan kotak 1 kg", Emoji: "🌴", Unit: "kotak 1 kg", Price: 285000},
	{Key: "kurma-sukari", Name: "Kurma Sukari", Desc: "Manis legit, tekstur lembut, kotak 1 kg", Emoji: "🌴", Unit: "kotak 1 kg", Price: 165000},
	{Key: "kurma-khalas", Name: "Kurma Khalas", Desc: "Rasa karamel, kotak 1 kg", Emoji: "🌴", Unit: "kotak 1 kg", Price: 140000},
	{Key: "zamzam-5l", Name: "Air zamzam kemasan resmi", Desc: "Galon 5 liter bersegel dari distributor resmi (contoh)", Emoji: "💧", Unit: "galon 5 liter", Price: 175000},
	{Key: "sajadah", Name: "Sajadah lipat", Desc: "Bahan beludru tipis, muat di tas kabin", Emoji: "🧎", Unit: "lembar", Price: 125000},
	{Key: "tasbih", Name: "Tasbih kayu 99 butir", Desc: "Kayu kokka, cocok untuk oleh-oleh", Emoji: "📿", Unit: "untai", Price: 45000},
	{Key: "parfum", Name: "Parfum non-alkohol", Desc: "Aroma oud & misk, botol roll-on 6 ml", Emoji: "🌸", Unit: "botol", Price: 95000},
	{Key: "kacang-arab", Name: "Kacang Arab campur", Desc: "Pistachio, almond, kacang Arab panggang 500 g", Emoji: "🥜", Unit: "pak 500 g", Price: 85000},
	{Key: "cokelat", Name: "Cokelat kurma", Desc: "Kurma isi almond berlapis cokelat, 250 g", Emoji: "🍫", Unit: "kotak", Price: 120000},
	{Key: "mukena", Name: "Mukena travel", Desc: "Bahan parasut adem, dengan tas kecil", Emoji: "🧕", Unit: "set", Price: 225000},
	{Key: "peci", Name: "Peci rajut putih", Desc: "Ukuran bebas, adem dipakai", Emoji: "🧢", Unit: "buah", Price: 35000},
}

// StoreCatalog: barang yang bisa dipesan lalu diambil di toko mitra (harga SAR, bayar di toko).
var StoreCatalog = []Product{
	{Key: "kurma-ajwa", Name: "Kurma Ajwa", Desc: "1 kg", Emoji: "🌴", Unit: "kg", Price: 70},
	{Key: "kurma-sukari", Name: "Kurma Sukari", Desc: "1 kg", Emoji: "🌴", Unit: "kg", Price: 35},
	{Key: "zamzam-5l", Name: "Air zamzam kemasan resmi", Desc: "Galon 5 liter bersegel", Emoji: "💧", Unit: "galon", Price: 12},
	{Key: "sajadah", Name: "Sajadah lipat", Desc: "Beludru tipis", Emoji: "🧎", Unit: "lembar", Price: 25},
	{Key: "tasbih", Name: "Tasbih kayu", Desc: "99 butir", Emoji: "📿", Unit: "untai", Price: 10},
	{Key: "parfum", Name: "Parfum non-alkohol", Desc: "Roll-on 6 ml", Emoji: "🌸", Unit: "botol", Price: 20},
	{Key: "tas-lipat", Name: "Tas tambahan lipat", Desc: "Untuk bawa oleh-oleh pulang", Emoji: "🧳", Unit: "buah", Price: 45},
}

// MandiriCatalog: katalog prototipe (tidak bisa dipesan sungguhan).
var MandiriCatalog = []Product{
	{Key: "htl-mkh-3", Kind: "hotel", Name: "Hotel bintang 3, Misfalah (contoh)", Desc: "Makkah · ±900 m ke Masjidil Haram · kamar 4 orang", Emoji: "🏨", Unit: "malam", Price: 850000},
	{Key: "htl-mkh-4", Kind: "hotel", Name: "Hotel bintang 4, Ajyad (contoh)", Desc: "Makkah · ±350 m ke Masjidil Haram · kamar 2 orang", Emoji: "🏨", Unit: "malam", Price: 1650000},
	{Key: "htl-mdn-3", Kind: "hotel", Name: "Hotel bintang 3, Markaziyah (contoh)", Desc: "Madinah · ±400 m ke Masjid Nabawi · kamar 4 orang", Emoji: "🏨", Unit: "malam", Price: 900000},
	{Key: "htl-mdn-5", Kind: "hotel", Name: "Hotel bintang 5, depan Nabawi (contoh)", Desc: "Madinah · ±100 m ke Masjid Nabawi · kamar 2 orang", Emoji: "🏨", Unit: "malam", Price: 2400000},
	{Key: "trp-jed-mkh", Kind: "transport", Name: "Jemputan bandara Jeddah → hotel Makkah (contoh)", Desc: "Mobil 4 penumpang + bagasi", Emoji: "🚐", Unit: "mobil", Price: 650000},
	{Key: "trp-mkh-mdn", Kind: "transport", Name: "Mobil Makkah → Madinah (contoh)", Desc: "Mobil 4 penumpang, ±5 jam perjalanan", Emoji: "🚐", Unit: "mobil", Price: 1900000},
	{Key: "trp-kereta", Kind: "transport", Name: "Kereta cepat Makkah → Madinah, ekonomi (perkiraan)", Desc: "Harga perkiraan per orang; tiket asli dibeli di kanal resmi operator", Emoji: "🚄", Unit: "orang", Price: 1150000},
	{Key: "trp-ziarah", Kind: "transport", Name: "Ziarah kota Madinah setengah hari (contoh)", Desc: "Quba, Uhud, kebun kurma · mobil + sopir", Emoji: "🚐", Unit: "mobil", Price: 500000},
	{Key: "mtw-umrah", Kind: "muthawwif", Name: "Muthawwif berbahasa Indonesia (contoh)", Desc: "Mendampingi umrah dari miqat sampai tahallul", Emoji: "🧭", Unit: "hari", Price: 750000},
	{Key: "mtw-ziarah", Kind: "muthawwif", Name: "Pembimbing ziarah Makkah (contoh)", Desc: "Jabal Tsur, Jabal Rahmah, Arafah, Mina", Emoji: "🧭", Unit: "hari", Price: 600000},
}

// MandiriKinds: tab katalog mandiri.
var MandiriKinds = []struct{ Key, Label string }{{"hotel", "Hotel"}, {"transport", "Transport"}, {"muthawwif", "Muthawwif"}}

func findProduct(cat []Product, key string) *Product {
	for i := range cat {
		if cat[i].Key == key {
			return &cat[i]
		}
	}
	return nil
}

func OlehProduct(key string) *Product    { return findProduct(OlehCatalog, key) }
func StoreProduct(key string) *Product   { return findProduct(StoreCatalog, key) }
func MandiriProduct(key string) *Product { return findProduct(MandiriCatalog, key) }

func MandiriByKind(kind string) []Product {
	var out []Product
	for _, p := range MandiriCatalog {
		if kind == "" || p.Kind == kind {
			out = append(out, p)
		}
	}
	return out
}

func catalogFor(kind string) []Product {
	if kind == "mandiri" {
		return MandiriCatalog
	}
	return OlehCatalog
}

// Ongkir oleh-oleh (contoh): Rp25.000, gratis mulai Rp750.000.
const (
	ShippingFee  int64 = 25000
	FreeShipFrom int64 = 750000
	MaxQty             = 20
)

func ShippingFor(subtotal int64) int64 {
	if subtotal == 0 || subtotal >= FreeShipFrom {
		return 0
	}
	return ShippingFee
}

// uniqueCode: kode unik untuk model (kolom "code").
func uniqueCode(model any, gen func() string) string {
	for i := 0; i < 8; i++ {
		c := gen()
		var n int64
		DB.Model(model).Where("code = ?", c).Count(&n)
		if n == 0 {
			return c
		}
	}
	return gen()
}

func newVoucherCode() string { return "UMV-" + randCode(4) + "-" + randCode(4) }

// NormalizeVoucherCode: "umv abcd efgh" / "UMVABCDEFGH" / "abcd-efgh" -> "UMV-ABCD-EFGH" ("" bila tidak valid).
func NormalizeVoucherCode(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
		if b.Len() > 20 {
			return ""
		}
	}
	x := b.String()
	if len(x) == 11 && strings.HasPrefix(x, "UMV") {
		x = x[3:]
	}
	if len(x) != 8 {
		return ""
	}
	return "UMV-" + x[:4] + "-" + x[4:]
}

// ---------------------------------------------------------------------------------------------
// Keranjang
// ---------------------------------------------------------------------------------------------

var ErrNotFound = errors.New("Data tidak ditemukan.")

// CartSet menyetel jumlah barang (0 = hapus). add=true menambah ke jumlah yang ada.
func CartSet(userID uint, kind, key string, qty int, add bool) error {
	if DB == nil || userID == 0 {
		return ErrNotFound
	}
	if kind != "oleh" && kind != "mandiri" {
		return errors.New("Jenis keranjang tidak dikenal.")
	}
	if findProduct(catalogFor(kind), key) == nil {
		return errors.New("Barang tidak ada di katalog.")
	}
	var it CartItem
	err := DB.Where("user_id = ? AND kind = ? AND key = ?", userID, kind, key).Take(&it).Error
	if add && err == nil {
		qty += it.Qty
	}
	qty = min(max(qty, 0), MaxQty)
	if err != nil {
		if qty == 0 {
			return nil
		}
		var n int64
		DB.Model(&CartItem{}).Where("user_id = ? AND kind = ?", userID, kind).Count(&n)
		if n >= 30 {
			return errors.New("Keranjang sudah penuh.")
		}
		return DB.Create(&CartItem{UserID: userID, Kind: kind, Key: key, Qty: qty}).Error
	}
	if qty == 0 {
		return DB.Delete(&it).Error
	}
	return DB.Model(&it).Update("qty", qty).Error
}

// Cart: baris keranjang + subtotal (harga selalu dari katalog, bukan dari input).
func Cart(userID uint, kind string) ([]Line, int64) {
	var items []CartItem
	if DB != nil && userID != 0 {
		DB.Where("user_id = ? AND kind = ?", userID, kind).Order("id").Find(&items)
	}
	cat := catalogFor(kind)
	var out []Line
	var total int64
	for _, it := range items {
		p := findProduct(cat, it.Key)
		if p == nil || it.Qty <= 0 {
			continue
		}
		l := Line{Key: p.Key, Name: p.Name, Emoji: p.Emoji, Unit: p.Unit, Price: p.Price, Qty: it.Qty}
		out = append(out, l)
		total += l.Sub()
	}
	return out, total
}

// CartQty: jumlah per kunci (untuk menandai barang yang sudah di keranjang).
func CartQty(userID uint, kind string) map[string]int {
	m := map[string]int{}
	ls, _ := Cart(userID, kind)
	for _, l := range ls {
		m[l.Key] = l.Qty
	}
	return m
}

func CartCount(userID uint, kind string) int {
	var n int64
	if DB != nil && userID != 0 {
		DB.Model(&CartItem{}).Where("user_id = ? AND kind = ?", userID, kind).Select("COALESCE(SUM(qty),0)").Scan(&n)
	}
	return int(n)
}

func cartClear(tx *gorm.DB, userID uint, kind string) error {
	return tx.Where("user_id = ? AND kind = ?", userID, kind).Delete(&CartItem{}).Error
}

// ---------------------------------------------------------------------------------------------
// Oleh-oleh
// ---------------------------------------------------------------------------------------------

type Address struct {
	Recipient, Phone, Address, City, Postal string
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (a *Address) validate() error {
	a.Recipient, a.Phone, a.Address = clip(a.Recipient, 80), clip(a.Phone, 20), clip(a.Address, 300)
	a.City, a.Postal = clip(a.City, 60), clip(a.Postal, 5)
	switch {
	case len([]rune(a.Recipient)) < 2:
		return errors.New("Nama penerima wajib diisi.")
	case len(digitsOnly(a.Phone)) < 9:
		return errors.New("Nomor HP penerima belum benar.")
	case len([]rune(a.Address)) < 10:
		return errors.New("Alamat lengkap terlalu pendek.")
	case a.City == "":
		return errors.New("Kota wajib diisi.")
	case a.Postal != "" && len(digitsOnly(a.Postal)) != 5:
		return errors.New("Kode pos harus 5 angka.")
	}
	return nil
}

// CreateShopOrder membuat pesanan dari keranjang oleh-oleh + tagihan QRIS (mock).
func CreateShopOrder(userID uint, a Address) (*ShopOrder, error) {
	if DB == nil || userID == 0 {
		return nil, ErrNotFound
	}
	if err := a.validate(); err != nil {
		return nil, err
	}
	ls, sub := Cart(userID, "oleh")
	if len(ls) == 0 {
		return nil, errors.New("Keranjang masih kosong.")
	}
	var pending int64
	DB.Model(&ShopOrder{}).Where("user_id = ? AND status = ?", userID, "menunggu_bayar").Count(&pending)
	if pending >= 5 {
		return nil, errors.New("Selesaikan dulu pembayaran pesanan yang masih menunggu.")
	}
	ship := ShippingFor(sub)
	o := &ShopOrder{Code: uniqueCode(&ShopOrder{}, func() string { return NewCode("OLH", 5) }), UserID: userID,
		Items: encodeLines(ls), Subtotal: sub, Shipping: ship, Total: sub + ship,
		Recipient: a.Recipient, Phone: a.Phone, Address: a.Address, City: a.City, Postal: a.Postal,
		Status: "menunggu_bayar", CreatedAt: time.Now()}
	inv, err := payments.Default.CreateQRIS(payments.Request{Ref: o.Code, Amount: o.Total, Holder: Supplier})
	if err != nil {
		return nil, err
	}
	o.QRPayload = inv.QRPayload
	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(o).Error; err != nil {
			return err
		}
		return cartClear(tx, userID, "oleh")
	})
	if err != nil {
		return nil, err
	}
	return o, nil
}

// ShopOrderFor: pesanan milik userID (nil bila bukan miliknya).
func ShopOrderFor(userID uint, code string) *ShopOrder {
	var o ShopOrder
	if DB == nil || userID == 0 || DB.Where("code = ? AND user_id = ?", code, userID).Take(&o).Error != nil {
		return nil
	}
	return &o
}

func ShopOrdersFor(userID uint) []ShopOrder {
	var out []ShopOrder
	if DB != nil && userID != 0 {
		DB.Where("user_id = ?", userID).Order("created_at DESC").Limit(30).Find(&out)
	}
	return out
}

var shopNext = map[string]string{"dibayar": "dikemas", "dikemas": "dikirim", "dikirim": "diterima"}

// ShopNextStatus: status berikutnya untuk tombol simulasi ("" bila selesai/belum bayar).
func ShopNextStatus(s string) string { return shopNext[s] }

// PayShopOrder: simulasi pembayaran QRIS berhasil.
func PayShopOrder(o *ShopOrder, to string) error {
	if o.Status != "menunggu_bayar" {
		return errors.New("Pesanan ini sudah dibayar.")
	}
	now := time.Now()
	res := DB.Model(&ShopOrder{}).Where("id = ? AND status = ?", o.ID, "menunggu_bayar").Updates(map[string]any{"status": "dibayar", "paid_at": now})
	if res.Error != nil || res.RowsAffected == 0 {
		return errors.New("Pembayaran gagal diproses.")
	}
	o.Status, o.PaidAt = "dibayar", &now
	Notify(UserAudience(o.UserID), "wa", to, "Pembayaran oleh-oleh diterima",
		fmt.Sprintf("Pembayaran pesanan %s sebesar Rp%s sudah diterima. %s sedang menyiapkan paketmu.", o.Code, Rp(o.Total), Supplier),
		"/tanah-suci/oleh-oleh/"+o.Code)
	return nil
}

// AdvanceShopOrder: simulasi status berikutnya (dibayar → dikemas → dikirim → diterima).
func AdvanceShopOrder(o *ShopOrder, to string) error {
	next := shopNext[o.Status]
	if next == "" {
		return errors.New("Status pesanan tidak bisa dilanjutkan.")
	}
	now := time.Now()
	upd := map[string]any{"status": next}
	msg := ""
	switch next {
	case "dikemas":
		upd["packed_at"] = now
		o.PackedAt = &now
		msg = fmt.Sprintf("Paket oleh-oleh %s sedang dikemas oleh %s.", o.Code, Supplier)
	case "dikirim":
		resi := "DEMO" + randCode(10)
		upd["shipped_at"], upd["courier"], upd["resi"] = now, "Kurir contoh", resi
		o.ShippedAt, o.Courier, o.Resi = &now, "Kurir contoh", resi
		msg = fmt.Sprintf("Paket %s sudah dikirim lewat Kurir contoh, resi %s.", o.Code, resi)
	case "diterima":
		upd["delivered_at"] = now
		o.DeliveredAt = &now
		msg = fmt.Sprintf("Paket %s sudah diterima. Semoga bermanfaat untuk keluarga di rumah.", o.Code)
	}
	res := DB.Model(&ShopOrder{}).Where("id = ? AND status = ?", o.ID, o.Status).Updates(upd)
	if res.Error != nil || res.RowsAffected == 0 {
		return errors.New("Status pesanan sudah berubah, muat ulang halaman.")
	}
	o.Status = next
	Notify(UserAudience(o.UserID), "wa", to, "Status oleh-oleh: "+StatusLabel(next), msg, "/tanah-suci/oleh-oleh/"+o.Code)
	return nil
}

// TimelineStep untuk halaman lacak pengiriman.
type TimelineStep struct {
	Status, Label, Note string
	At                  *time.Time
	Done, Current       bool
}

func (o ShopOrder) Timeline() []TimelineStep {
	created := o.CreatedAt
	steps := []TimelineStep{
		{Status: "menunggu_bayar", Label: "Pesanan dibuat", Note: "Menunggu pembayaran QRIS", At: &created},
		{Status: "dibayar", Label: "Pembayaran diterima", Note: "Diteruskan ke " + Supplier, At: o.PaidAt},
		{Status: "dikemas", Label: "Dikemas pemasok", Note: "Kurma & zamzam dikemas aman", At: o.PackedAt},
		{Status: "dikirim", Label: "Dalam pengiriman", Note: "Kurir contoh", At: o.ShippedAt},
		{Status: "diterima", Label: "Diterima di rumah", Note: o.City, At: o.DeliveredAt},
	}
	if o.Resi != "" {
		steps[3].Note = o.Courier + " · resi " + o.Resi
	}
	cur := 0
	for i, s := range steps {
		if s.Status == o.Status {
			cur = i
		}
	}
	for i := range steps {
		steps[i].Done = i <= cur
		steps[i].Current = i == cur
	}
	return steps
}

// ---------------------------------------------------------------------------------------------
// Voucher prabayar
// ---------------------------------------------------------------------------------------------

const (
	VoucherMinIDR int64 = 200_000
	VoucherMaxIDR int64 = 10_000_000
	MaxRedeemSAR  int64 = 5_000
	VoucherHolder       = "Penerbit voucher mitra (contoh)"
)

// VoucherPresets: pilihan nominal rupiah di halaman beli.
var VoucherPresets = []int64{500_000, 1_000_000, 2_000_000, 3_000_000, 5_000_000}

// SARFor: nilai SAR (dibulatkan ke bawah) untuk nominal rupiah pada kurs saat ini.
func SARFor(idr int64) int64 {
	if Cfg.SAR <= 0 {
		return 0
	}
	return idr / Cfg.SAR
}

// BuyVoucher membuat voucher berstatus menunggu_bayar. Rupiah yang ditagih = SAR × kurs (sisa
// pembulatan tidak ditagih).
func BuyVoucher(userID uint, idr int64) (*Voucher, error) {
	if DB == nil || userID == 0 {
		return nil, ErrNotFound
	}
	if idr < VoucherMinIDR || idr > VoucherMaxIDR {
		return nil, fmt.Errorf("Nominal voucher Rp%s sampai Rp%s.", Rp(VoucherMinIDR), Rp(VoucherMaxIDR))
	}
	sar := SARFor(idr)
	if sar <= 0 {
		return nil, errors.New("Nominal terlalu kecil.")
	}
	var pending int64
	DB.Model(&Voucher{}).Where("user_id = ? AND status = ?", userID, "menunggu_bayar").Count(&pending)
	if pending >= 5 {
		return nil, errors.New("Selesaikan dulu pembayaran voucher yang masih menunggu.")
	}
	v := &Voucher{Code: uniqueCode(&Voucher{}, newVoucherCode), UserID: userID,
		AmountIDR: sar * Cfg.SAR, Rate: Cfg.SAR, ValueSAR: sar, BalanceSAR: sar, Status: "menunggu_bayar",
		ExpiresAt: time.Now().In(WIB).AddDate(1, 0, 0), CreatedAt: time.Now()}
	ref := NewCode("TSV", 6) // ref tagihan sengaja tidak memuat kode voucher
	inv, err := payments.Default.CreateQRIS(payments.Request{Ref: ref, Amount: v.AmountIDR, Holder: VoucherHolder})
	if err != nil {
		return nil, err
	}
	v.QRPayload = inv.QRPayload
	if err := DB.Create(v).Error; err != nil {
		return nil, err
	}
	return v, nil
}

func PayVoucher(v *Voucher, to string) error {
	if v.Status != "menunggu_bayar" {
		return errors.New("Voucher ini sudah dibayar.")
	}
	now := time.Now()
	res := DB.Model(&Voucher{}).Where("id = ? AND status = ?", v.ID, "menunggu_bayar").Updates(map[string]any{"status": "aktif", "paid_at": now})
	if res.Error != nil || res.RowsAffected == 0 {
		return errors.New("Pembayaran gagal diproses.")
	}
	v.Status, v.PaidAt = "aktif", &now
	Notify(UserAudience(v.UserID), "wa", to, "Voucher Tanah Suci aktif",
		fmt.Sprintf("Voucher %s senilai %d SAR sudah aktif. Tunjukkan kode QR di toko mitra di Makkah/Madinah.", v.Code, v.ValueSAR),
		"/tanah-suci/voucher/"+v.Code)
	return nil
}

// VoucherFor: voucher milik userID (nil bila bukan miliknya).
func VoucherFor(userID uint, code string) *Voucher {
	var v Voucher
	if DB == nil || userID == 0 || DB.Where("code = ? AND user_id = ?", code, userID).Take(&v).Error != nil {
		return nil
	}
	return &v
}

func VouchersFor(userID uint) []Voucher {
	var vs []Voucher
	if DB != nil && userID != 0 {
		DB.Where("user_id = ?", userID).Order("created_at DESC").Limit(30).Find(&vs)
	}
	return vs
}

// VoucherByCode: dipakai toko untuk cek saldo (kode = rahasia yang ditunjukkan jamaah).
func VoucherByCode(code string) *Voucher {
	code = NormalizeVoucherCode(code)
	var v Voucher
	if DB == nil || code == "" || DB.Where("code = ?", code).Take(&v).Error != nil {
		return nil
	}
	return &v
}

func Redemptions(voucherID uint) []VoucherRedemption {
	var r []VoucherRedemption
	if DB != nil {
		DB.Where("voucher_id = ?", voucherID).Order("created_at DESC, id DESC").Find(&r)
	}
	return r
}

func MerchantRedemptions(slug string, limit int) []VoucherRedemption {
	var r []VoucherRedemption
	if DB != nil {
		DB.Where("merchant_slug = ?", slug).Order("created_at DESC, id DESC").Limit(limit).Find(&r)
	}
	return r
}

// MerchantTotals: total penukaran toko hari ini (WIB) dan seluruhnya.
func MerchantTotals(slug string) (todaySAR int64, todayN int, allSAR int64, allN int) {
	if DB == nil {
		return
	}
	start, _ := time.ParseInLocation("2006-01-02", Today(), WIB)
	type agg struct {
		S int64
		N int
	}
	var t, a agg
	DB.Model(&VoucherRedemption{}).Select("COALESCE(SUM(amount_sar),0) AS s, COUNT(*) AS n").
		Where("merchant_slug = ? AND created_at >= ?", slug, start).Scan(&t)
	DB.Model(&VoucherRedemption{}).Select("COALESCE(SUM(amount_sar),0) AS s, COUNT(*) AS n").
		Where("merchant_slug = ?", slug).Scan(&a)
	return t.S, t.N, a.S, a.N
}

// ActiveVouchers: voucher aktif untuk tombol "simulasi pindai" di toko (hanya demo).
func ActiveVouchers(limit int) []Voucher {
	var vs []Voucher
	if DB != nil {
		DB.Where("status = ? AND balance_sar > 0", "aktif").Order("created_at DESC").Limit(limit).Find(&vs)
	}
	return vs
}

var (
	ErrVoucherNotFound = errors.New("Kode voucher tidak ditemukan. Periksa lagi 8 karakter setelah UMV.")
	ErrVoucherUnpaid   = errors.New("Voucher ini belum dibayar, jadi belum bisa dipakai.")
	ErrVoucherEmpty    = errors.New("Saldo voucher sudah habis.")
	ErrVoucherExpired  = errors.New("Voucher sudah melewati masa berlaku.")
	ErrOverBalance     = errors.New("Nominal melebihi sisa saldo voucher.")
	ErrBadAmount       = fmt.Errorf("Nominal harus 1 sampai %d SAR.", MaxRedeemSAR)
	ErrBadMerchant     = errors.New("Toko tidak dikenal.")
)

// RedeemVoucher memotong saldo voucher di toko mitra. Aman dari dobel potong: update bersyarat
// "balance_sar >= amount" di dalam transaksi.
func RedeemVoucher(code, merchantSlug string, amountSAR int64, note string) (*Voucher, *VoucherRedemption, error) {
	if DB == nil {
		return nil, nil, ErrNotFound
	}
	m := MerchantBySlug(merchantSlug)
	if m == nil {
		return nil, nil, ErrBadMerchant
	}
	if amountSAR <= 0 || amountSAR > MaxRedeemSAR {
		return nil, nil, ErrBadAmount
	}
	v := VoucherByCode(code)
	if v == nil {
		return nil, nil, ErrVoucherNotFound
	}
	switch {
	case v.Status == "menunggu_bayar":
		return v, nil, ErrVoucherUnpaid
	case v.Status == "habis" || v.BalanceSAR <= 0:
		return v, nil, ErrVoucherEmpty
	case !v.ExpiresAt.IsZero() && time.Now().After(v.ExpiresAt):
		return v, nil, ErrVoucherExpired
	case amountSAR > v.BalanceSAR:
		return v, nil, ErrOverBalance
	}
	var red *VoucherRedemption
	err := DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&Voucher{}).Where("id = ? AND status = ? AND balance_sar >= ?", v.ID, "aktif", amountSAR).
			Update("balance_sar", gorm.Expr("balance_sar - ?", amountSAR))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrOverBalance
		}
		if err := tx.Where("id = ?", v.ID).Take(v).Error; err != nil {
			return err
		}
		if v.BalanceSAR <= 0 {
			if err := tx.Model(&Voucher{}).Where("id = ?", v.ID).Update("status", "habis").Error; err != nil {
				return err
			}
			v.Status = "habis"
		}
		red = &VoucherRedemption{VoucherID: v.ID, VoucherCode: v.Code, MerchantSlug: merchantSlug, AmountSAR: amountSAR,
			BalanceAfter: v.BalanceSAR, Note: clip(note, 120), CreatedAt: time.Now()}
		return tx.Create(red).Error
	})
	if err != nil {
		return v, nil, err
	}
	Notify(UserAudience(v.UserID), "wa", "Jamaah", "Voucher dipakai di "+m.Name,
		fmt.Sprintf("Voucher %s dipakai %d SAR di %s (%s). Sisa saldo %d SAR.", v.Code, amountSAR, m.Name, m.City, v.BalanceSAR),
		"/tanah-suci/voucher/"+v.Code)
	Notify("merchant:"+merchantSlug, "wa", m.Name, "Voucher berhasil ditukar",
		fmt.Sprintf("Penukaran %d SAR dari voucher %s berhasil dan masuk rekap penukaran toko.", amountSAR, MaskVoucher(v.Code)), "/toko/riwayat")
	return v, red, nil
}

// ---------------------------------------------------------------------------------------------
// Pickup di toko
// ---------------------------------------------------------------------------------------------

type Slot struct{ Value, Label string }

var (
	slotTimes  = []string{"10:00", "16:00", "21:30"}
	hariPendek = [...]string{"Min", "Sen", "Sel", "Rab", "Kam", "Jum", "Sab"}
)

// PickupSlots: 6 hari ke depan mulai besok × 3 jam (waktu setempat; demo memakai tanggal WIB).
func PickupSlots() []Slot {
	var out []Slot
	d := time.Now().In(WIB)
	for i := 1; i <= 6; i++ {
		day := d.AddDate(0, 0, i).Format("2006-01-02")
		for _, t := range slotTimes {
			v := day + " " + t
			out = append(out, Slot{Value: v, Label: SlotLabel(v)})
		}
	}
	return out
}

func SlotLabel(v string) string {
	t, err := time.ParseInLocation("2006-01-02 15:04", v, WIB)
	if err != nil {
		return v
	}
	return fmt.Sprintf("%s, %d %s · %02d.%02d", hariPendek[t.Weekday()], t.Day(), bulan[t.Month()], t.Hour(), t.Minute())
}

func validSlot(v string) bool {
	for _, s := range PickupSlots() {
		if s.Value == v {
			return true
		}
	}
	return false
}

// CreatePickup: qty per kunci StoreCatalog.
func CreatePickup(userID uint, name, merchantSlug, slot string, qty map[string]int) (*Pickup, error) {
	if DB == nil || userID == 0 {
		return nil, ErrNotFound
	}
	m := MerchantBySlug(merchantSlug)
	if m == nil {
		return nil, errors.New("Pilih toko mitra dulu.")
	}
	if !validSlot(slot) {
		return nil, errors.New("Pilih jadwal ambil yang tersedia.")
	}
	var ls []Line
	var total int64
	for _, p := range StoreCatalog {
		q := qty[p.Key]
		if q <= 0 {
			continue
		}
		if q > MaxQty {
			return nil, fmt.Errorf("Maksimal %d per barang.", MaxQty)
		}
		l := Line{Key: p.Key, Name: p.Name, Emoji: p.Emoji, Unit: p.Unit, Price: p.Price, Qty: q}
		ls = append(ls, l)
		total += l.Sub()
	}
	if len(ls) == 0 {
		return nil, errors.New("Pilih minimal satu barang.")
	}
	var open int64
	DB.Model(&Pickup{}).Where("user_id = ? AND status <> ?", userID, "diambil").Count(&open)
	if open >= 10 {
		return nil, errors.New("Masih ada 10 pickup yang belum diambil.")
	}
	name = clip(name, 80)
	if name == "" {
		name = "Jamaah"
	}
	p := &Pickup{Code: uniqueCode(&Pickup{}, func() string { return NewCode("PU", 5) }), UserID: userID, Name: name,
		MerchantSlug: m.Slug, Items: encodeLines(ls), TotalSAR: total, Slot: slot, Status: "dijadwalkan", CreatedAt: time.Now()}
	if err := DB.Create(p).Error; err != nil {
		return nil, err
	}
	Notify("merchant:"+m.Slug, "wa", m.Name, "Pesanan pickup baru "+p.Code,
		fmt.Sprintf("%s memesan %d jenis barang (%d SAR) untuk diambil %s.", MaskName(p.Name), len(ls), total, p.SlotLabel()), "/toko/pickup")
	Notify(UserAudience(userID), "wa", p.Name, "Pickup dijadwalkan di "+m.Name,
		fmt.Sprintf("Pesanan %s dijadwalkan diambil %s di %s, %s. Kami kabari saat barang siap.", p.Code, p.SlotLabel(), m.Name, m.City),
		"/tanah-suci/pickup/"+p.Code)
	return p, nil
}

// PickupFor: pickup milik userID (nil bila bukan miliknya).
func PickupFor(userID uint, code string) *Pickup {
	var p Pickup
	if DB == nil || userID == 0 || DB.Where("code = ? AND user_id = ?", code, userID).Take(&p).Error != nil {
		return nil
	}
	return &p
}

func PickupsFor(userID uint) []Pickup {
	var ps []Pickup
	if DB != nil && userID != 0 {
		DB.Where("user_id = ?", userID).Order("created_at DESC").Limit(30).Find(&ps)
	}
	return ps
}

// MerchantPickups: pickup untuk satu toko saja (siap dulu, lalu dijadwalkan, lalu selesai).
func MerchantPickups(slug string) []Pickup {
	var ps []Pickup
	if DB != nil {
		DB.Where("merchant_slug = ?", slug).
			Order("CASE status WHEN 'siap' THEN 0 WHEN 'dijadwalkan' THEN 1 ELSE 2 END, slot").Limit(60).Find(&ps)
	}
	return ps
}

// AdvancePickup: dilakukan toko. to: "siap" | "diambil". Hanya pickup milik toko itu.
func AdvancePickup(merchantSlug string, id uint, to string) (*Pickup, error) {
	var p Pickup
	if DB == nil || DB.Where("id = ? AND merchant_slug = ?", id, merchantSlug).Take(&p).Error != nil {
		return nil, ErrNotFound
	}
	from := map[string]string{"siap": "dijadwalkan", "diambil": "siap"}[to]
	if from == "" || p.Status != from {
		return &p, errors.New("Status pickup tidak bisa diubah ke " + StatusLabel(to) + ".")
	}
	now := time.Now()
	upd := map[string]any{"status": to}
	if to == "siap" {
		upd["ready_at"] = now
	} else {
		upd["taken_at"] = now
	}
	res := DB.Model(&Pickup{}).Where("id = ? AND status = ?", p.ID, from).Updates(upd)
	if res.Error != nil || res.RowsAffected == 0 {
		return &p, errors.New("Status pickup sudah berubah.")
	}
	p.Status = to
	m := MerchantBySlug(merchantSlug)
	if to == "siap" {
		p.ReadyAt = &now
		Notify(UserAudience(p.UserID), "wa", p.Name, "Pickup siap diambil",
			fmt.Sprintf("Pesanan %s sudah siap di %s (%s). Tunjukkan kode %s ke kasir, bayar %d SAR tunai atau pakai voucher.", p.Code, m.Name, m.Landmark, p.Code, p.TotalSAR),
			"/tanah-suci/pickup/"+p.Code)
	} else {
		p.TakenAt = &now
		Notify(UserAudience(p.UserID), "wa", p.Name, "Pickup sudah diambil",
			fmt.Sprintf("Pesanan %s sudah diambil di %s. Terima kasih, semoga umrahnya mabrur.", p.Code, m.Name), "/tanah-suci/pickup/"+p.Code)
	}
	return &p, nil
}

// ---------------------------------------------------------------------------------------------
// Umrah mandiri (prototipe)
// ---------------------------------------------------------------------------------------------

type CheckItem struct{ Key, Label string }

type MandiriStep struct {
	Key, Title, Desc, Icon, Kind string // Kind: jenis katalog terkait ("" = tidak ada)
	Items                        []CheckItem
}

var MandiriSteps = []MandiriStep{
	{Key: "visa", Title: "Visa & izin umrah lewat Nusuk", Icon: "📄",
		Desc:  "Buat akun di platform resmi pemerintah Arab Saudi, ajukan visa, lalu ambil izin umrah. Syarat bisa berubah, jadi selalu cek kanal resmi.",
		Items: []CheckItem{{"paspor", "Paspor berlaku minimal 6 bulan saat berangkat"}, {"vaksin", "Vaksin meningitis & sertifikat vaksin internasional"}, {"akun-nusuk", "Akun Nusuk dibuat"}, {"visa", "Visa disetujui"}, {"asuransi", "Asuransi perjalanan aktif"}}},
	{Key: "tiket", Title: "Tiket pesawat", Icon: "✈️",
		Desc:  "Beli tiket pulang-pergi langsung di maskapai atau agen resmi. Prototipe ini tidak menjual tiket.",
		Items: []CheckItem{{"tiket-pp", "Tiket pulang-pergi sudah dibeli"}, {"tiket-tanggal", "Tanggal sesuai masa berlaku visa"}, {"bagasi", "Jatah bagasi cukup untuk oleh-oleh"}}},
	{Key: "hotel", Title: "Hotel Makkah & Madinah", Icon: "🏨", Kind: "hotel",
		Desc:  "Pilih jarak ke masjid sesuai kondisi fisik rombongan. Makin dekat, makin sedikit jalan kaki di siang hari.",
		Items: []CheckItem{{"hotel-makkah", "Hotel Makkah dipesan"}, {"hotel-madinah", "Hotel Madinah dipesan"}}},
	{Key: "transport", Title: "Transport darat", Icon: "🚐", Kind: "transport",
		Desc:  "Jemputan bandara, perjalanan antarkota, dan ziarah. Pastikan kendaraan berizin dan sopir paham rute.",
		Items: []CheckItem{{"jemput", "Jemputan bandara ke hotel"}, {"antarkota", "Perjalanan Makkah ↔ Madinah"}, {"antar-bandara", "Antar ke bandara saat pulang"}}},
	{Key: "muthawwif", Title: "Muthawwif / pembimbing", Icon: "🧭", Kind: "muthawwif",
		Desc:  "Pendamping ibadah sangat dianjurkan untuk umrah pertama, terutama bila ada lansia di rombongan.",
		Items: []CheckItem{{"muthawwif", "Muthawwif dipilih"}, {"manasik", "Manasik/latihan tata cara umrah"}}},
}

func validCheckKey(k string) bool {
	for _, s := range MandiriSteps {
		for _, it := range s.Items {
			if it.Key == k {
				return true
			}
		}
	}
	return false
}

func MandiriChecklist(userID uint) map[string]bool {
	out := map[string]bool{}
	var st MandiriState
	if DB != nil && userID != 0 && DB.Where("user_id = ?", userID).Take(&st).Error == nil {
		for _, k := range strings.Split(st.Done, ",") {
			if k != "" {
				out[k] = true
			}
		}
	}
	return out
}

func MandiriCheckTotal() int {
	n := 0
	for _, s := range MandiriSteps {
		n += len(s.Items)
	}
	return n
}

// SaveMandiriChecklist menyimpan set kunci yang dicentang (kunci tak dikenal diabaikan).
func SaveMandiriChecklist(userID uint, keys []string) error {
	if DB == nil || userID == 0 {
		return ErrNotFound
	}
	seen := map[string]bool{}
	var ok []string
	for _, k := range keys {
		if validCheckKey(k) && !seen[k] {
			seen[k] = true
			ok = append(ok, k)
		}
	}
	var st MandiriState
	if DB.Where("user_id = ?", userID).Take(&st).Error != nil {
		return DB.Create(&MandiriState{UserID: userID, Done: strings.Join(ok, ","), UpdatedAt: time.Now()}).Error
	}
	return DB.Model(&st).Updates(map[string]any{"done": strings.Join(ok, ","), "updated_at": time.Now()}).Error
}

func CreateMandiriOrder(userID uint, contact, phone, depart string) (*MandiriOrder, error) {
	if DB == nil || userID == 0 {
		return nil, ErrNotFound
	}
	contact, phone = clip(contact, 80), clip(phone, 20)
	if len([]rune(contact)) < 2 {
		return nil, errors.New("Nama pemesan wajib diisi.")
	}
	if len(digitsOnly(phone)) < 9 {
		return nil, errors.New("Nomor HP belum benar.")
	}
	d, err := time.ParseInLocation("2006-01-02", depart, WIB)
	if err != nil || d.Before(time.Now().In(WIB)) || d.After(time.Now().AddDate(2, 0, 0)) {
		return nil, errors.New("Pilih tanggal berangkat yang akan datang (maksimal 2 tahun lagi).")
	}
	ls, total := Cart(userID, "mandiri")
	if len(ls) == 0 {
		return nil, errors.New("Keranjang masih kosong.")
	}
	o := &MandiriOrder{Code: uniqueCode(&MandiriOrder{}, func() string { return NewCode("MDR", 5) }), UserID: userID,
		Items: encodeLines(ls), Total: total, Contact: contact, Phone: phone, DepartDate: depart, Status: "menunggu_bayar", CreatedAt: time.Now()}
	inv, err := payments.Default.CreateQRIS(payments.Request{Ref: o.Code, Amount: total, Holder: "Penyedia layanan contoh"})
	if err != nil {
		return nil, err
	}
	o.QRPayload = inv.QRPayload
	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(o).Error; err != nil {
			return err
		}
		return cartClear(tx, userID, "mandiri")
	})
	if err != nil {
		return nil, err
	}
	return o, nil
}

// MandiriOrderFor: pesanan milik userID (nil bila bukan miliknya).
func MandiriOrderFor(userID uint, code string) *MandiriOrder {
	var o MandiriOrder
	if DB == nil || userID == 0 || DB.Where("code = ? AND user_id = ?", code, userID).Take(&o).Error != nil {
		return nil
	}
	return &o
}

func MandiriOrdersFor(userID uint) []MandiriOrder {
	var out []MandiriOrder
	if DB != nil && userID != 0 {
		DB.Where("user_id = ?", userID).Order("created_at DESC").Limit(20).Find(&out)
	}
	return out
}

func PayMandiriOrder(o *MandiriOrder, to string) error {
	if o.Status != "menunggu_bayar" {
		return errors.New("Pesanan ini sudah dibayar.")
	}
	now := time.Now()
	res := DB.Model(&MandiriOrder{}).Where("id = ? AND status = ?", o.ID, "menunggu_bayar").Updates(map[string]any{"status": "dibayar", "paid_at": now})
	if res.Error != nil || res.RowsAffected == 0 {
		return errors.New("Pembayaran gagal diproses.")
	}
	o.Status, o.PaidAt = "dibayar", &now
	Notify(UserAudience(o.UserID), "email", to, "Pesanan umrah mandiri (prototipe) "+o.Code,
		fmt.Sprintf("Simulasi pembayaran Rp%s untuk %d layanan berhasil. Ini prototipe: tidak ada hotel/transport yang benar-benar dipesan.", Rp(o.Total), len(o.Lines())),
		"/mandiri/pesanan/"+o.Code)
	return nil
}

// ---------------------------------------------------------------------------------------------
// Seed
// ---------------------------------------------------------------------------------------------

func linesFrom(cat []Product, qty map[string]int) ([]Line, int64) {
	var out []Line
	var s int64
	for _, p := range cat {
		if q := qty[p.Key]; q > 0 {
			l := Line{Key: p.Key, Name: p.Name, Emoji: p.Emoji, Unit: p.Unit, Price: p.Price, Qty: q}
			out = append(out, l)
			s += l.Sub()
		}
	}
	return out, s
}

// seedShop mengisi contoh untuk akun jamaah demo supaya halaman tidak kosong. Idempoten.
func seedShop() {
	if DB == nil {
		return
	}
	var n int64
	DB.Model(&Voucher{}).Count(&n)
	if n > 0 {
		return
	}
	u := EnsureDemoUser()
	if u == nil {
		return
	}
	now := time.Now()
	hAgo := func(h int) *time.Time { t := now.Add(-time.Duration(h) * time.Hour); return &t }
	rate := Cfg.SAR

	// Oleh-oleh: satu sedang dikirim, satu sudah diterima.
	l1, s1 := linesFrom(OlehCatalog, map[string]int{"kurma-ajwa": 2, "zamzam-5l": 1, "tasbih": 5})
	l2, s2 := linesFrom(OlehCatalog, map[string]int{"kurma-sukari": 3, "sajadah": 2})
	a := Address{Recipient: u.Name, Phone: "0812-0000-1234", Address: "Jl. Contoh Melati No. 12, RT 03/RW 05", City: "Bandung", Postal: "40111"}
	DB.Create(&ShopOrder{Code: NewCode("OLH", 5), UserID: u.ID, Items: encodeLines(l1), Subtotal: s1, Shipping: ShippingFor(s1), Total: s1 + ShippingFor(s1),
		Recipient: a.Recipient, Phone: a.Phone, Address: a.Address, City: a.City, Postal: a.Postal, Status: "dikirim",
		Courier: "Kurir contoh", Resi: "DEMO" + randCode(10), QRPayload: "DEMO-QRIS|seed",
		PaidAt: hAgo(50), PackedAt: hAgo(30), ShippedAt: hAgo(6), CreatedAt: *hAgo(51)})
	DB.Create(&ShopOrder{Code: NewCode("OLH", 5), UserID: u.ID, Items: encodeLines(l2), Subtotal: s2, Shipping: ShippingFor(s2), Total: s2 + ShippingFor(s2),
		Recipient: a.Recipient, Phone: a.Phone, Address: a.Address, City: a.City, Postal: a.Postal, Status: "diterima",
		Courier: "Kurir contoh", Resi: "DEMO" + randCode(10), QRPayload: "DEMO-QRIS|seed",
		PaidAt: hAgo(24 * 20), PackedAt: hAgo(24 * 19), ShippedAt: hAgo(24 * 18), DeliveredAt: hAgo(24 * 16), CreatedAt: *hAgo(24 * 20)})

	// Voucher: satu aktif (sudah dipakai sebagian), satu habis; riwayat di beberapa toko.
	exp := now.In(WIB).AddDate(1, 0, 0)
	v1 := Voucher{Code: newVoucherCode(), UserID: u.ID, AmountIDR: 465 * rate, Rate: rate, ValueSAR: 465, BalanceSAR: 465,
		Status: "aktif", QRPayload: "DEMO-QRIS|seed", PaidAt: hAgo(24 * 3), ExpiresAt: exp, CreatedAt: *hAgo(24 * 3)}
	v2 := Voucher{Code: newVoucherCode(), UserID: u.ID, AmountIDR: 116 * rate, Rate: rate, ValueSAR: 116, BalanceSAR: 116,
		Status: "aktif", QRPayload: "DEMO-QRIS|seed", PaidAt: hAgo(24 * 9), ExpiresAt: exp, CreatedAt: *hAgo(24 * 9)}
	DB.Create(&v1)
	DB.Create(&v2)
	for _, r := range []struct {
		v    *Voucher
		m    string
		amt  int64
		h    int
		note string
	}{
		{&v2, "madinah-markaziyah", 70, 24*8 + 3, "Kurma ajwa 1 kg"},
		{&v2, "madinah-quba", 46, 24 * 7, "Sajadah & tasbih"},
		{&v1, "makkah-ajyad", 85, 30, "Kurma sukari + zamzam"},
		{&v1, "makkah-misfalah", 40, 5, "Parfum non-alkohol ×2"},
		{&v1, "makkah-ajyad", 22, 1, "Tasbih kayu ×2"},
	} {
		r.v.BalanceSAR -= r.amt
		DB.Create(&VoucherRedemption{VoucherID: r.v.ID, VoucherCode: r.v.Code, MerchantSlug: r.m, AmountSAR: r.amt,
			BalanceAfter: r.v.BalanceSAR, Note: r.note, CreatedAt: *hAgo(r.h)})
	}
	DB.Model(&Voucher{}).Where("id = ?", v1.ID).Update("balance_sar", v1.BalanceSAR)
	DB.Model(&Voucher{}).Where("id = ?", v2.ID).Updates(map[string]any{"balance_sar": v2.BalanceSAR, "status": "habis"})

	// Pickup: satu dijadwalkan di Makkah, satu siap di Madinah, satu sudah diambil.
	slots := PickupSlots()
	pa, ta := linesFrom(StoreCatalog, map[string]int{"kurma-ajwa": 2, "zamzam-5l": 2})
	pb, tb := linesFrom(StoreCatalog, map[string]int{"kurma-sukari": 3, "tas-lipat": 1})
	pc, tc := linesFrom(StoreCatalog, map[string]int{"tasbih": 4, "parfum": 1})
	today := now.In(WIB)
	DB.Create(&Pickup{Code: NewCode("PU", 5), UserID: u.ID, Name: u.Name, MerchantSlug: "makkah-ajyad", Items: encodeLines(pa), TotalSAR: ta,
		Slot: slots[1].Value, Status: "dijadwalkan", CreatedAt: *hAgo(3)})
	DB.Create(&Pickup{Code: NewCode("PU", 5), UserID: u.ID, Name: u.Name, MerchantSlug: "madinah-markaziyah", Items: encodeLines(pb), TotalSAR: tb,
		Slot: today.Format("2006-01-02") + " 21:30", Status: "siap", ReadyAt: hAgo(2), CreatedAt: *hAgo(26)})
	DB.Create(&Pickup{Code: NewCode("PU", 5), UserID: u.ID, Name: u.Name, MerchantSlug: "makkah-misfalah", Items: encodeLines(pc), TotalSAR: tc,
		Slot: today.AddDate(0, 0, -4).Format("2006-01-02") + " 16:00", Status: "diambil", ReadyAt: hAgo(24*4 + 3), TakenAt: hAgo(24 * 4), CreatedAt: *hAgo(24 * 5)})

	// Umrah mandiri: beberapa langkah sudah dicentang.
	DB.Create(&MandiriState{UserID: u.ID, Done: "paspor,vaksin,akun-nusuk,tiket-pp", UpdatedAt: now})
}
