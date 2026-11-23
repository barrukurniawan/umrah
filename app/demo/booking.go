package demo

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"umrah/app/models"
	"umrah/app/payments"
	"umrah/app/repositories"
	"umrah/app/services"

	"gorm.io/gorm"
)

// ---- paket dari data pencarian ----

// PkgOption: satu paket travel mitra + tanggal yang masih bisa dipesan.
type PkgOption struct {
	Key        string
	Travel     Travel
	Terms      TravelTerms
	Package    models.Package
	Departures []models.DetailPackage
}

// FindPackage mencari paket berdasarkan kunci stabil (travel|nama) di data paket terkini,
// hanya tanggal mendatang yang belum penuh.
func FindPackage(key string) (*PkgOption, error) {
	for _, p := range repositories.AllPackages() {
		if services.PackageKey(p.Travel.Name, p.Name) != key {
			continue
		}
		t := TravelByName(p.Travel.Name)
		if t == nil || !t.Partner {
			return nil, errors.New("travel ini belum menjadi mitra booking umrohku")
		}
		tm := TermsFor(t.Slug)
		if tm == nil {
			return nil, errors.New("ketentuan travel belum diatur")
		}
		opt := &PkgOption{Key: key, Travel: *t, Terms: *tm, Package: p}
		today := Today()
		for _, d := range p.Details {
			if d.DepartureDate > AddDays(today, 7) && !d.IsFull {
				if d.Price == 0 {
					d.Price = p.Price
				}
				opt.Departures = append(opt.Departures, d)
			}
		}
		if len(opt.Departures) == 0 {
			return nil, errors.New("tidak ada tanggal keberangkatan yang masih bisa dipesan")
		}
		return opt, nil
	}
	return nil, errors.New("paket tidak ditemukan")
}

func (o *PkgOption) Departure(date string) *models.DetailPackage {
	for i := range o.Departures {
		if o.Departures[i].DepartureDate == date {
			return &o.Departures[i]
		}
	}
	return nil
}

// RoomPrice: harga per orang untuk tipe kamar (harga resmi travel).
func RoomPrice(d *models.DetailPackage, room string) int64 {
	p := int64(d.Price)
	switch room {
	case "triple":
		return p + int64(d.AddonTriple)
	case "double":
		return p + int64(d.AddonDouble)
	}
	return p
}

// ---- membuat booking ----

type BookingInput struct {
	UserID       uint
	Key          string
	Date         string
	Room         string
	Pilgrims     []string
	ContactName  string
	ContactPhone string
	City         string
	DPPerPerson  int64 // DP yang dipilih per orang (>= DP minimal umrohku)
	PromoCode    string
	Method       string // va | qris
}

type Quote struct {
	Opt            *PkgOption
	Dep            *models.DetailPackage
	People         int
	PricePerPerson int64
	Total          int64
	DPMin          int64 // per booking
	DPNormal       int64 // per booking, DP normal bila langsung ke travel
	DPCredit       int64 // DP yang dicatat travel
	DPDiscount     int64
	DPPay          int64 // yang dibayar jamaah sekarang
	Cashback       int64
	Pelunasan      int64 // sisa setelah DP & cashback
	PelunasanDue   string
	DPRule         *PromoRule
	CBRule         *PromoRule
	CodeErr        string
}

func BuildQuote(in BookingInput) (*Quote, error) {
	opt, err := FindPackage(in.Key)
	if err != nil {
		return nil, err
	}
	d := opt.Departure(in.Date)
	if d == nil {
		d = &opt.Departures[0]
	}
	people := len(in.Pilgrims)
	if people < 1 {
		people = 1
	}
	if people > 9 {
		return nil, errors.New("maksimal 9 jamaah per booking")
	}
	if d.AvailableQuota > 0 && people > d.AvailableQuota {
		return nil, fmt.Errorf("sisa seat tanggal ini tinggal %d", d.AvailableQuota)
	}
	if in.Room != "triple" && in.Room != "double" {
		in.Room = "quad"
	}
	q := &Quote{Opt: opt, Dep: d, People: people, PricePerPerson: RoomPrice(d, in.Room)}
	q.Total = q.PricePerPerson * int64(people)
	q.DPMin = opt.Terms.DPMinUmrohku * int64(people)
	q.DPNormal = opt.Terms.DPNormalTravel * int64(people)
	q.DPCredit = in.DPPerPerson * int64(people)
	if q.DPCredit < q.DPMin {
		q.DPCredit = q.DPMin
	}
	if q.DPCredit > q.Total {
		q.DPCredit = q.Total
	}
	dp, cb, codeErr := QuotePromos(opt.Travel.Slug, q.Total, in.PromoCode)
	q.CodeErr = codeErr
	if dp.Rule != nil {
		q.DPRule, q.DPDiscount = dp.Rule, min(dp.Amount, q.DPCredit/2) // DP yang dibayar tidak boleh nol
	}
	if cb.Rule != nil {
		q.CBRule, q.Cashback = cb.Rule, cb.Amount
	}
	q.DPPay = q.DPCredit - q.DPDiscount
	q.Pelunasan = max(0, q.Total-q.DPCredit-q.Cashback)
	q.PelunasanDue = AddDays(d.DepartureDate, -opt.Terms.PelunasanHMinus)
	if q.PelunasanDue < Today() {
		q.PelunasanDue = AddDays(Today(), 3)
	}
	return q, nil
}

func CreateBooking(in BookingInput) (*Booking, *Payment, error) {
	q, err := BuildQuote(in)
	if err != nil {
		return nil, nil, err
	}
	if q.CodeErr != "" {
		return nil, nil, errors.New(q.CodeErr)
	}
	var names []string
	for _, n := range in.Pilgrims {
		if n = strings.TrimSpace(n); n != "" {
			names = append(names, truncate(n, 80))
		}
	}
	if len(names) == 0 {
		return nil, nil, errors.New("isi nama jamaah sesuai paspor")
	}
	pj, _ := json.Marshal(names)
	room := in.Room
	if room != "triple" && room != "double" {
		room = "quad"
	}
	b := &Booking{
		Code: NewCode("UMK", 6), UserID: in.UserID, TravelSlug: q.Opt.Travel.Slug, TravelName: q.Opt.Travel.Name,
		PackageKey: in.Key, PackageName: q.Opt.Package.Name, DepartureDate: q.Dep.DepartureDate, ReturnDate: q.Dep.ReturnDate,
		Room: room, People: q.People, PricePerPerson: q.PricePerPerson, Total: q.Total, DPAmount: q.DPCredit,
		DPDiscount: q.DPDiscount, CashbackPromised: q.Cashback, Pilgrims: string(pj), ContactName: truncate(in.ContactName, 80),
		ContactPhone: truncate(in.ContactPhone, 30), City: truncate(in.City, 60), Status: "menunggu_dp", PelunasanDue: q.PelunasanDue,
		KomisiPersen: q.Opt.Terms.KomisiPersen, CreatedAt: time.Now(),
	}
	var pay *Payment
	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(b).Error; err != nil {
			return err
		}
		for _, pr := range []struct {
			r    *PromoRule
			amt  int64
			kind string
		}{{q.DPRule, q.DPDiscount, "diskon_dp"}, {q.CBRule, q.Cashback, "cashback_pelunasan"}} {
			if pr.r == nil || pr.amt <= 0 {
				continue
			}
			res := tx.Model(&PromoRule{}).Where("id = ? AND (quota = 0 OR used < quota)", pr.r.ID).Update("used", gorm.Expr("used + 1"))
			if res.RowsAffected == 0 {
				return errors.New("kuota promo baru saja habis, coba lagi")
			}
			if err := tx.Create(&PromoUse{PromoID: pr.r.ID, BookingID: b.ID, Kind: pr.kind, Amount: pr.amt, Status: "dijanjikan", CreatedAt: time.Now()}).Error; err != nil {
				return err
			}
		}
		p, err := newPayment(tx, b, "dp", b.DPAmount-b.DPDiscount, in.Method)
		pay = p
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	return b, pay, nil
}

func newPayment(tx *gorm.DB, b *Booking, kind string, amount int64, method string) (*Payment, error) {
	t := TravelBySlug(b.TravelSlug)
	if t == nil {
		return nil, errors.New("travel tidak ditemukan")
	}
	req := payments.Request{Ref: b.Code + "-" + strings.ToUpper(kind) + "-" + randCode(3), Amount: amount, Bank: t.EscrowBank, Account: t.EscrowAccount, Holder: t.EscrowHolder, Payer: b.ContactName}
	var inv payments.Invoice
	var err error
	if method == "qris" {
		inv, err = payments.Default.CreateQRIS(req)
	} else {
		method = "va"
		inv, err = payments.Default.CreateVA(req)
	}
	if err != nil {
		return nil, err
	}
	p := &Payment{BookingID: b.ID, Kind: kind, Method: method, Bank: inv.Bank, VANumber: inv.VANumber, Holder: inv.Holder,
		QRPayload: inv.QRPayload, Amount: amount, Status: "menunggu", Ref: req.Ref, ExpiresAt: inv.ExpiresAt, CreatedAt: time.Now()}
	return p, tx.Create(p).Error
}

// ---- membaca ----

func (b Booking) PilgrimNames() []string {
	var n []string
	json.Unmarshal([]byte(b.Pilgrims), &n)
	return n
}

func (b Booking) RoomLabel() string {
	return map[string]string{"quad": "Quad (ber-4)", "triple": "Triple (ber-3)", "double": "Double (ber-2)"}[b.Room]
}

// Remaining: sisa pelunasan setelah DP & cashback.
func (b Booking) Remaining() int64 {
	return max(0, b.Total-b.DPAmount-b.CashbackPromised)
}

func GetBooking(id uint) (*Booking, error) {
	var b Booking
	if err := DB.Take(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func UserBookings(userID uint) []Booking {
	var bs []Booking
	DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&bs)
	return bs
}

func TravelBookings(slug, status string) []Booking {
	var bs []Booking
	q := DB.Where("travel_slug = ?", slug)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Order("created_at DESC").Limit(300).Find(&bs)
	return bs
}

func BookingPayments(id uint) []Payment {
	var ps []Payment
	DB.Where("booking_id = ?", id).Order("created_at").Find(&ps)
	return ps
}

func BookingPromos(id uint) []PromoUse {
	var ps []PromoUse
	DB.Where("booking_id = ?", id).Find(&ps)
	return ps
}

func GetPayment(id uint) (*Payment, error) {
	var p Payment
	if err := DB.Take(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// ---- pembayaran (simulasi) ----

// CreatePelunasan membuat tagihan pelunasan; cashback yang dijanjikan langsung diterapkan
// sebagai potongan tagihan (bukan saldo).
func CreatePelunasan(b *Booking, method string) (*Payment, error) {
	if b.Status != "dp_lunas" {
		return nil, errors.New("pelunasan hanya untuk booking yang DP-nya sudah lunas")
	}
	var open Payment
	if DB.Where("booking_id = ? AND kind = ? AND status = ?", b.ID, "pelunasan", "menunggu").Take(&open).Error == nil {
		return &open, nil
	}
	var p *Payment
	err := DB.Transaction(func(tx *gorm.DB) error {
		var err error
		p, err = newPayment(tx, b, "pelunasan", b.Remaining(), method)
		return err
	})
	return p, err
}

// SimulatePaid: tombol "Simulasikan pembayaran berhasil" (pengganti callback gateway).
func SimulatePaid(paymentID uint) (*Booking, error) {
	p, err := GetPayment(paymentID)
	if err != nil {
		return nil, err
	}
	if p.Status == "berhasil" {
		return GetBooking(p.BookingID)
	}
	if p.Status != "menunggu" {
		return nil, errors.New("tagihan sudah tidak berlaku")
	}
	b, err := GetBooking(p.BookingID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	return b, markPaid(p, b, now)
}

func markPaid(p *Payment, b *Booking, now time.Time) error {
	err := DB.Transaction(func(tx *gorm.DB) error {
		p.Status, p.PaidAt = "berhasil", &now
		if err := tx.Save(p).Error; err != nil {
			return err
		}
		switch p.Kind {
		case "dp":
			if b.Status != "menunggu_dp" {
				return errors.New("booking tidak menunggu DP")
			}
			b.Status, b.PaidAt = "dp_lunas", &now
		case "pelunasan":
			b.Status, b.LunasAt = "lunas", &now
			b.CashbackApplied = b.CashbackPromised
			tx.Model(&PromoUse{}).Where("booking_id = ? AND kind = ? AND status = ?", b.ID, "cashback_pelunasan", "dijanjikan").
				Updates(map[string]any{"status": "diterapkan", "applied_at": &now})
		}
		if p.Kind == "dp" {
			tx.Model(&PromoUse{}).Where("booking_id = ? AND kind = ? AND status = ?", b.ID, "diskon_dp", "dijanjikan").
				Updates(map[string]any{"status": "diterapkan", "applied_at": &now})
		}
		return tx.Save(b).Error
	})
	if err != nil {
		return err
	}
	if p.Kind == "dp" {
		afterDPPaid(b)
	} else {
		Notify(UserAudience(b.UserID), "wa", b.ContactPhone, "Pelunasan diterima",
			fmt.Sprintf("Alhamdulillah, pelunasan booking %s sebesar Rp %s sudah diterima %s. Cashback Rp %s sudah dipotong dari tagihan. Status: LUNAS.",
				b.Code, Rp(p.Amount), b.TravelName, Rp(b.CashbackApplied)), fmt.Sprintf("/booking/%d", b.ID))
		Notify("travel:"+b.TravelSlug, "email", "finance", "Pelunasan masuk: "+b.Code,
			fmt.Sprintf("Pelunasan Rp %s untuk booking %s (%d jamaah) masuk ke rekening penampungan. Cashback Rp %s ditanggung umrohku (dipotong dari komisi).",
				Rp(p.Amount), b.Code, b.People, Rp(b.CashbackApplied)), "/portal/booking")
	}
	return nil
}

// afterDPPaid: kunci seat, catat lead booking, buat kode undian, kirim konfirmasi.
func afterDPPaid(b *Booking) {
	lockSeats(b, -b.People)
	RecordLead(Lead{TravelSlug: b.TravelSlug, Package: b.PackageName, Kind: "booking", Name: b.ContactName, Phone: b.ContactPhone,
		City: b.City, Status: "closing", Source: "booking umrohku", CreatedAt: time.Now()})
	DB.Model(&Lead{}).Where("travel_slug = ? AND kind = ? AND name = ? AND billable = ?", b.TravelSlug, "booking", b.ContactName, true).
		Updates(map[string]any{"billable": false, "reason": "Booking: ditagih lewat komisi"})
	codes := CreateEntries(b)
	msg := fmt.Sprintf("Assalamu'alaikum %s, DP booking %s sebesar Rp %s sudah diterima di rekening penampungan %s.\n\nPaket: %s\nBerangkat: %s · %d jamaah\nPelunasan Rp %s paling lambat %s.",
		firstWord(b.ContactName), b.Code, Rp(b.DPAmount-b.DPDiscount), b.TravelName, b.PackageName, Tgl(b.DepartureDate), b.People, Rp(b.Remaining()), Tgl(b.PelunasanDue))
	if b.CashbackPromised > 0 {
		msg += fmt.Sprintf("\nCashback Rp %s akan dipotong otomatis dari tagihan pelunasan.", Rp(b.CashbackPromised))
	}
	Notify(UserAudience(b.UserID), "wa", b.ContactPhone, "DP diterima · "+b.Code, msg, fmt.Sprintf("/booking/%d", b.ID))
	Notify(UserAudience(b.UserID), "email", b.ContactName, "E-kuitansi DP "+b.Code, "Kuitansi pembayaran DP terlampir di halaman booking. Simpan untuk arsip Anda.", fmt.Sprintf("/booking/%d/kuitansi", b.ID))
	if len(codes) > 0 {
		Notify(UserAudience(b.UserID), "wa", b.ContactPhone, "Kode undian umroh",
			"Selamat, booking "+b.Code+" otomatis ikut undian umroh gratis. Kode undianmu: "+strings.Join(codes, ", ")+". Tanpa biaya, tidak perlu membeli apa pun.", "/akun/undian")
	}
	Notify("travel:"+b.TravelSlug, "wa", "Admin travel", "Booking baru: "+b.Code,
		fmt.Sprintf("%s memesan %s untuk %d jamaah, berangkat %s. DP Rp %s masuk ke rekening penampungan. Seat sudah dikunci.",
			b.ContactName, b.PackageName, b.People, Tgl(b.DepartureDate), Rp(b.DPAmount)), "/portal/booking")
}

func firstWord(s string) string {
	if f := strings.Fields(s); len(f) > 0 {
		return f[0]
	}
	return s
}

// lockSeats mengubah sisa seat paket portal yang sesuai (delta negatif = kunci).
func lockSeats(b *Booking, delta int) {
	var pkgs []PortalPackage
	DB.Where("travel_slug = ? AND name = ?", b.TravelSlug, b.PackageName).Find(&pkgs)
	changed := false
	for _, p := range pkgs {
		res := DB.Model(&PortalDeparture{}).Where("package_id = ? AND date = ?", p.ID, b.DepartureDate).
			Update("seats_left", gorm.Expr("MAX(0, MIN(seats_total, seats_left + ?))", delta))
		changed = changed || res.RowsAffected > 0
	}
	if changed {
		ReloadPackages()
	}
}

// ReloadPackages menulis ulang paket portal dan memuat ulang data pencarian.
func ReloadPackages() {
	if err := WritePackagesFile(); err != nil {
		log.Println("[demo] tulis paket:", err)
		return
	}
	if err := repositories.Reload(); err == nil {
		services.ResetCalendarCache()
	}
}

// ---- status oleh travel & pembatalan ----

func CancelBooking(b *Booking, reason string) {
	if b.Status == "batal" {
		return
	}
	wasPaid := b.Status == "dp_lunas" || b.Status == "lunas"
	now := time.Now()
	b.Status, b.CancelledAt, b.CancelReason = "batal", &now, reason
	DB.Save(b)
	DB.Model(&PromoUse{}).Where("booking_id = ? AND status <> ?", b.ID, "batal").Update("status", "batal")
	DB.Model(&Payment{}).Where("booking_id = ? AND status = ?", b.ID, "menunggu").Update("status", "kedaluwarsa")
	VoidEntries(b.ID)
	if wasPaid {
		lockSeats(b, b.People)
	}
	Notify(UserAudience(b.UserID), "wa", b.ContactPhone, "Booking dibatalkan · "+b.Code, "Booking "+b.Code+" dibatalkan: "+reason+". Kode undian dari booking ini tidak ikut diundi.", fmt.Sprintf("/booking/%d", b.ID))
}

// TravelSetStatus: travel menandai jamaah berangkat / sudah pulang (membuka ulasan).
func TravelSetStatus(slug string, id uint, status string) error {
	var b Booking
	if DB.Where("id = ? AND travel_slug = ?", id, slug).Take(&b).Error != nil {
		return errors.New("booking tidak ditemukan")
	}
	switch {
	case status == "berangkat" && (b.Status == "lunas" || b.Status == "dp_lunas"):
	case status == "pulang" && b.Status == "berangkat":
	default:
		return errors.New("perubahan status tidak valid")
	}
	b.Status = status
	DB.Save(&b)
	if status == "pulang" {
		Notify(UserAudience(b.UserID), "wa", b.ContactPhone, "Bagaimana umrohmu?",
			"Alhamdulillah sudah kembali ke tanah air. Bantu jamaah lain dengan ulasan jujur untuk "+b.TravelName+". Ulasan diberi tanda \"Berangkat via umrohku\".", fmt.Sprintf("/booking/%d/ulasan", b.ID))
	}
	return nil
}

// ExpireJob: booking menunggu DP > 24 jam dan booking yang lewat batas pelunasan dibatalkan.
func ExpireJob() {
	if DB == nil {
		return
	}
	var bs []Booking
	DB.Where("status = ? AND created_at < ?", "menunggu_dp", time.Now().Add(-24*time.Hour)).Find(&bs)
	for i := range bs {
		CancelBooking(&bs[i], "DP tidak dibayar dalam 24 jam")
	}
	bs = nil
	DB.Where("status = ? AND pelunasan_due < ?", "dp_lunas", Today()).Find(&bs)
	for i := range bs {
		CancelBooking(&bs[i], "Melewati batas pelunasan H-"+fmt.Sprint(termsDays(bs[i].TravelSlug))+" (aturan DP mengikuti ketentuan travel)")
	}
	DB.Model(&Payment{}).Where("status = ? AND expires_at < ?", "menunggu", time.Now()).Update("status", "kedaluwarsa")
}

func termsDays(slug string) int {
	if t := TermsFor(slug); t != nil {
		return t.PelunasanHMinus
	}
	return 45
}

// ---- ulasan ----

func AddReview(b *Booking, rating int, text, name string) error {
	if b.Status != "pulang" {
		return errors.New("ulasan hanya untuk jamaah yang sudah pulang")
	}
	var n int64
	DB.Model(&Review{}).Where("booking_id = ?", b.ID).Count(&n)
	if n > 0 {
		return errors.New("booking ini sudah memberi ulasan")
	}
	if rating < 1 || rating > 5 || len(strings.TrimSpace(text)) < 10 {
		return errors.New("beri bintang 1–5 dan ulasan minimal 10 huruf")
	}
	return DB.Create(&Review{BookingID: b.ID, UserID: b.UserID, TravelSlug: b.TravelSlug, Rating: rating, Text: truncate(text, 1500),
		Name: truncate(name, 80), City: b.City, Month: monthLabel(b.DepartureDate), CreatedAt: time.Now()}).Error
}

func Reviews(slug string) []Review {
	var rs []Review
	DB.Where("travel_slug = ?", slug).Order("created_at DESC").Find(&rs)
	return rs
}

func ReviewStats(slug string) (avg float64, n int) {
	for _, r := range Reviews(slug) {
		avg += float64(r.Rating)
		n++
	}
	if n > 0 {
		avg = math.Round(avg/float64(n)*10) / 10
	}
	return
}

func monthLabel(date string) string {
	t, err := time.ParseInLocation("2006-01-02", date, WIB)
	if err != nil {
		return ""
	}
	return bulan[t.Month()] + " " + t.Format("2006")
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
