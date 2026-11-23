package demo

import "time"

// Travel mitra (fiktif). Travel asli hasil crawl TIDAK punya baris di sini kecuali admin
// mencatat hasil cek PPIU-nya sungguhan (Status "belum" = tidak ada badge).
type Travel struct {
	ID             uint   `gorm:"primaryKey"`
	Slug           string `gorm:"size:80;uniqueIndex"`
	Name           string `gorm:"size:120;uniqueIndex"` // sama persis dengan nama di data paket
	City           string `gorm:"size:80"`
	Fictional      bool   // travel contoh untuk demo
	Partner        bool   // sudah tanda tangan perjanjian: bisa booking + DP lewat umrohku
	PPIUNo         string `gorm:"size:80"`
	PPIUValidUntil string `gorm:"size:10"`
	PPIUStatus     string `gorm:"size:20"` // "terverifikasi" | "belum" | "kedaluwarsa"
	PPIUSource     string `gorm:"size:200"`
	PPIUCheckedAt  *time.Time
	EscrowBank     string `gorm:"size:80"`
	EscrowAccount  string `gorm:"size:40"`
	EscrowHolder   string `gorm:"size:160"`
	Plan           string `gorm:"size:20"` // gratis | pro | premium
	PlanUntil      string `gorm:"size:10"`
	Komisi         float64
	Rating         float64
	Since          int // tahun berdiri
	PIC            string `gorm:"size:80"`
	Phone          string `gorm:"size:30"`
	CreatedAt      time.Time
}

// TravelTerms: ketentuan hasil perjanjian dengan travel mitra (PLAN_CICILAN_PROMO_UNDIAN §2).
type TravelTerms struct {
	ID              uint   `gorm:"primaryKey"`
	TravelSlug      string `gorm:"size:80;uniqueIndex"`
	DPMinUmrohku    int64  // DP minimal lewat umrohku, per orang
	DPNormalTravel  int64  // DP normal bila bayar langsung ke travel, per orang
	PelunasanHMinus int    // pelunasan paling lambat H-xx
	DPHangusRule    string `gorm:"size:400"`
	KomisiPersen    float64
	AgreementRef    string `gorm:"size:120"`
}

// PortalPackage: paket yang dikelola travel sendiri lewat portal (menggantikan scraping).
type PortalPackage struct {
	ID           uint   `gorm:"primaryKey"`
	TravelSlug   string `gorm:"size:80;index"`
	Name         string `gorm:"size:200"`
	Price        int64
	PriceTriple  int64
	PriceDouble  int64
	Duration     int
	Airline      string `gorm:"size:80"`
	Direct       bool
	TransitVia   string `gorm:"size:60"`
	HotelMakkah  string `gorm:"size:120"`
	HotelMadinah string `gorm:"size:120"`
	StarsMakkah  int
	StarsMadinah int
	DistMakkah   int
	DistMadinah  int
	Facilities   string `gorm:"type:text"` // satu per baris
	Excludes     string `gorm:"type:text"`
	Guide        string `gorm:"size:120"`
	Active       bool
	UpdatedAt    time.Time
	Departures   []PortalDeparture `gorm:"foreignKey:PackageID"`
}

type PortalDeparture struct {
	ID         uint   `gorm:"primaryKey"`
	PackageID  uint   `gorm:"index"`
	Date       string `gorm:"size:10"`
	ReturnDate string `gorm:"size:10"`
	SeatsTotal int
	SeatsLeft  int
	Price      int64 // 0 = harga paket
}

// Lead: setiap klik WA/website atau form "minta dihubungi" pada paket travel mitra.
type Lead struct {
	ID          uint   `gorm:"primaryKey"`
	TravelSlug  string `gorm:"size:80;index"`
	Package     string `gorm:"size:200"`
	Kind        string `gorm:"size:20"` // wa | web | form | booking
	Visitor     string `gorm:"size:64;index"`
	Source      string `gorm:"size:40"`
	Device      string `gorm:"size:20"`
	Name        string `gorm:"size:80"` // form lead (dummy)
	Phone       string `gorm:"size:30"`
	City        string `gorm:"size:60"`
	Status      string `gorm:"size:20"` // baru | dihubungi | closing | tidak_valid
	Billable    bool
	Reason      string `gorm:"size:120"` // alasan tidak ditagih
	CreatedAt   time.Time `gorm:"index"`
}

// Placement: posisi unggulan berbayar ("Promo Spesial"), menggantikan map featuredTravels di kode.
type Placement struct {
	ID         uint   `gorm:"primaryKey"`
	TravelName string `gorm:"size:120;index"`
	MinBudget  int
	MaxBudget  int
	StartsAt   string `gorm:"size:10"`
	EndsAt     string `gorm:"size:10"`
	PricePerDay int64
	Active     bool
	CreatedAt  time.Time
}

// InvoiceStatus: status bayar tagihan bulanan travel (isi tagihan dihitung dari data).
type InvoiceStatus struct {
	ID         uint   `gorm:"primaryKey"`
	TravelSlug string `gorm:"size:80;uniqueIndex:ux_inv"`
	Period     string `gorm:"size:7;uniqueIndex:ux_inv"` // "2026-11"
	PaidAt     *time.Time
}

// SubscriptionLog: riwayat langganan portal (mock checkout).
type SubscriptionLog struct {
	ID         uint   `gorm:"primaryKey"`
	TravelSlug string `gorm:"size:80;index"`
	Plan       string `gorm:"size:20"`
	Months     int
	Amount     int64
	From       string `gorm:"size:10"`
	Until      string `gorm:"size:10"`
	CreatedAt  time.Time
}

// Booking lewat umrohku. Harga per orang = harga resmi travel; dana DP & pelunasan selalu ke
// rekening penampungan travel.
type Booking struct {
	ID               uint   `gorm:"primaryKey"`
	Code             string `gorm:"size:16;uniqueIndex"`
	UserID           uint   `gorm:"index"`
	TravelSlug       string `gorm:"size:80;index"`
	TravelName       string `gorm:"size:120"`
	PackageKey       string `gorm:"size:300"`
	PackageName      string `gorm:"size:200"`
	DepartureDate    string `gorm:"size:10"`
	ReturnDate       string `gorm:"size:10"`
	Room             string `gorm:"size:10"` // quad | triple | double
	People           int
	PricePerPerson   int64
	Total            int64 // harga resmi × orang (sebelum potongan)
	DPAmount         int64 // DP yang dibayar jamaah (setelah diskon DP)
	DPDiscount       int64 // diskon DP (ditanggung umrohku)
	CashbackPromised int64 // cashback pelunasan yang dijanjikan
	CashbackApplied  int64
	Pilgrims         string `gorm:"type:text"` // JSON []string nama sesuai paspor
	ContactName      string `gorm:"size:80"`
	ContactPhone     string `gorm:"size:30"`
	City             string `gorm:"size:60"`
	Status           string `gorm:"size:20;index"` // menunggu_dp | dp_lunas | lunas | batal | berangkat | pulang
	PelunasanDue     string `gorm:"size:10"`
	PaidAt           *time.Time // DP lunas
	LunasAt          *time.Time
	CancelledAt      *time.Time
	CancelReason     string `gorm:"size:200"`
	KomisiPersen     float64
	CreatedAt        time.Time `gorm:"index"`
}

// Payment: satu tagihan VA/QRIS (mock) untuk DP atau pelunasan.
type Payment struct {
	ID        uint   `gorm:"primaryKey"`
	BookingID uint   `gorm:"index"`
	Kind      string `gorm:"size:20"` // dp | pelunasan
	Method    string `gorm:"size:10"` // va | qris
	Bank      string `gorm:"size:60"`
	VANumber  string `gorm:"size:40"`
	Holder    string `gorm:"size:160"` // nama rekening penampungan travel
	QRPayload string `gorm:"size:300"`
	Amount    int64
	Status    string `gorm:"size:20"` // menunggu | berhasil | kedaluwarsa
	Ref       string `gorm:"size:40"`
	ExpiresAt time.Time
	PaidAt    *time.Time
	CreatedAt time.Time
}

// PromoRule & PromoUse: diskon DP & cashback pelunasan, selalu dibiayai umrohku.
type PromoRule struct {
	ID         uint   `gorm:"primaryKey"`
	Code       string `gorm:"size:40;index"` // kosong = otomatis berlaku
	Title      string `gorm:"size:120"`
	Kind       string `gorm:"size:30"` // diskon_dp | cashback_pelunasan
	Amount     int64
	Percent    float64
	Cap        int64
	TravelSlug string `gorm:"size:80"`
	MinPrice   int64
	Quota      int
	Used       int
	StartsAt   time.Time
	EndsAt     time.Time
	FundedBy   string `gorm:"size:20"`
	Active     bool
}

type PromoUse struct {
	ID        uint   `gorm:"primaryKey"`
	PromoID   uint   `gorm:"index"`
	BookingID uint   `gorm:"index"`
	Kind      string `gorm:"size:30"`
	Amount    int64
	Status    string `gorm:"size:20"` // dijanjikan | diterapkan | batal
	AppliedAt *time.Time
	CreatedAt time.Time
}

// Draw & DrawEntry: undian umroh berhadiah (gratis, bebas maysir).
type Draw struct {
	ID          uint   `gorm:"primaryKey"`
	Title       string `gorm:"size:150"`
	PrizeDesc   string `gorm:"size:300"`
	Winners     int
	StartsAt    time.Time
	EndsAt      time.Time
	DrawAt      time.Time
	PermitNo    string `gorm:"size:120"`
	Terms       string `gorm:"type:text"`
	Status      string `gorm:"size:20"` // draft | terbuka | ditutup | diundi
	EntriesHash string `gorm:"size:64"`
	WitnessSeed string `gorm:"size:128"`
	Seed        string `gorm:"size:128"`
	Witness     string `gorm:"size:160"`
	ClosedAt    *time.Time
	DrawnAt     *time.Time
}

type DrawEntry struct {
	ID        uint   `gorm:"primaryKey"`
	DrawID    uint   `gorm:"uniqueIndex:ux_draw_booking_pilgrim;index"`
	BookingID uint   `gorm:"uniqueIndex:ux_draw_booking_pilgrim"`
	Pilgrim   int    `gorm:"uniqueIndex:ux_draw_booking_pilgrim"`
	UserID    uint   `gorm:"index"`
	Name      string `gorm:"size:80"` // nama jamaah (ditampilkan disamarkan)
	City      string `gorm:"size:60"`
	Code      string `gorm:"size:16;uniqueIndex"`
	Void      bool
	Winner    int // 0 = bukan pemenang, 1..N = urutan pemenang
	CreatedAt time.Time
}

// FinancingApplication: rujukan pembiayaan syariah ke mitra (umrohku tidak memberi pinjaman).
type FinancingApplication struct {
	ID               uint   `gorm:"primaryKey"`
	Code             string `gorm:"size:16;uniqueIndex"`
	UserID           uint   `gorm:"index"`
	PartnerSlug      string `gorm:"size:40;index"`
	BookingID        uint
	TravelName       string `gorm:"size:120"`
	PackageName      string `gorm:"size:200"`
	PackageKey       string `gorm:"size:300"`
	Amount           int64 // pokok yang diajukan
	Tenor            int
	EstMonthly       int64
	Name             string `gorm:"size:80"`
	Phone            string `gorm:"size:30"`
	Job              string `gorm:"size:60"`
	Income           int64
	City             string `gorm:"size:60"`
	ConsentAt        *time.Time // persetujuan berbagi data (UU PDP) ke mitra tujuan
	ConsentRevokedAt *time.Time
	Status           string `gorm:"size:20"` // diajukan | diproses | disetujui | ditolak | dicairkan | dibatalkan
	Note             string `gorm:"size:300"`
	DecidedAt        *time.Time
	DisbursedAt      *time.Time
	ReferralFee      int64
	CreatedAt        time.Time `gorm:"index"`
}

// SavingsLink: Tabungan umroh dihubungkan ke rekening tabungan umroh di bank syariah mitra (demo).
type SavingsLink struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"index"`
	GoalID    uint   `gorm:"uniqueIndex"`
	Bank      string `gorm:"size:80"`
	AccountNo string `gorm:"size:40"`
	Autodebit int64
	Day       int
	LinkedAt  time.Time
}

// CicilIntent: pengukur minat "Mau cicil?" (#7).
type CicilIntent struct {
	ID        uint   `gorm:"primaryKey"`
	Visitor   string `gorm:"size:64;index"`
	UserID    uint
	Travel    string `gorm:"size:120;index"`
	Package   string `gorm:"size:200"`
	Price     int64
	Path      string `gorm:"size:20"` // sebelum | pembiayaan | lihat
	Tenor     int
	CreatedAt time.Time `gorm:"index"`
}

// InboxMessage: simulasi WA/email (tidak pernah dikirim keluar).
type InboxMessage struct {
	ID        uint   `gorm:"primaryKey"`
	Audience  string `gorm:"size:60;index"` // user:<id> | travel:<slug> | admin | partner:<slug> | merchant:<slug>
	Channel   string `gorm:"size:10"`        // wa | email
	To        string `gorm:"size:120"`
	Subject   string `gorm:"size:160"`
	Body      string `gorm:"type:text"`
	Link      string `gorm:"size:200"`
	Read      bool
	CreatedAt time.Time `gorm:"index"`
}

// Review: hanya dari jamaah dengan booking berstatus "pulang".
type Review struct {
	ID         uint   `gorm:"primaryKey"`
	BookingID  uint   `gorm:"index"`
	UserID     uint
	TravelSlug string `gorm:"size:80;index"`
	Rating     int
	Text       string `gorm:"type:text"`
	Name       string `gorm:"size:80"`
	City       string `gorm:"size:60"`
	Month      string `gorm:"size:20"` // "Okt 2026" bulan berangkat
	Sample     bool   // contoh seed
	CreatedAt  time.Time
}

// PartnerSignup: formulir /mitra (pendaftaran travel).
type PartnerSignup struct {
	ID        uint   `gorm:"primaryKey"`
	Company   string `gorm:"size:160"`
	Brand     string `gorm:"size:120"`
	PPIUNo    string `gorm:"size:80"`
	City      string `gorm:"size:80"`
	PIC       string `gorm:"size:80"`
	Phone     string `gorm:"size:30"`
	Email     string `gorm:"size:120"`
	Note      string `gorm:"size:500"`
	Status    string `gorm:"size:20"` // baru | dihubungi | diverifikasi
	CreatedAt time.Time
}

func init() {
	RegisterModels(&Travel{}, &TravelTerms{}, &PortalPackage{}, &PortalDeparture{}, &Lead{}, &Placement{},
		&InvoiceStatus{}, &SubscriptionLog{}, &Booking{}, &Payment{}, &PromoRule{}, &PromoUse{}, &Draw{}, &DrawEntry{},
		&FinancingApplication{}, &SavingsLink{}, &CicilIntent{}, &InboxMessage{}, &Review{}, &PartnerSignup{})
}
