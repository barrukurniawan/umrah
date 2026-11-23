package demo

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
)

// Rujukan pembiayaan syariah: umrohku TIDAK memberi pinjaman. Pengajuan diteruskan ke mitra
// berizin (fiktif di demo) hanya setelah jamaah menyetujui berbagi data (UU PDP), dan persetujuan
// itu bisa dicabut selama pengajuan belum diputuskan.

type Estimate struct {
	Partner  Partner
	Tenor    int
	Pokok    int64
	Ujrah    int64
	Admin    int64
	Total    int64
	Monthly  int64
	UjrahPct float64
}

// Estimates: contoh perhitungan per mitra untuk satu tenor (bukan penawaran).
func Estimates(pokok int64, tenor int) []Estimate {
	var out []Estimate
	for _, p := range Cfg.Partners {
		u, ok := p.Ujrah[strconv.Itoa(tenor)]
		if !ok {
			continue
		}
		e := Estimate{Partner: p, Tenor: tenor, Pokok: pokok, UjrahPct: u, Admin: p.Admin}
		e.Ujrah = int64(math.Round(float64(pokok) * u / 100))
		e.Total = pokok + e.Ujrah + e.Admin
		e.Monthly = int64(math.Ceil(float64(e.Total)/float64(tenor)/1000) * 1000)
		out = append(out, e)
	}
	return out
}

// SavingPlan: cicil sebelum berangkat (tanpa utang): setoran per bulan sampai H-pelunasan.
type SavingPlan struct {
	Target   int64
	Initial  int64
	Months   int
	Monthly  int64
	Deadline string
}

func PlanSaving(target, initial int64, deadline string) SavingPlan {
	months := max(1, DaysUntil(deadline)/30)
	rest := max(0, target-initial)
	return SavingPlan{Target: target, Initial: initial, Months: months, Deadline: deadline,
		Monthly: int64(math.Ceil(float64(rest)/float64(months)/1000) * 1000)}
}

type FinancingInput struct {
	UserID      uint
	PartnerSlug string
	BookingID   uint
	TravelName  string
	PackageName string
	PackageKey  string
	Amount      int64
	Tenor       int
	Name        string
	Phone       string
	Job         string
	Income      int64
	City        string
	Consent     bool
}

func ApplyFinancing(in FinancingInput) (*FinancingApplication, error) {
	p := PartnerBySlug(in.PartnerSlug)
	if p == nil || p.Kind == "" {
		return nil, errors.New("pilih mitra pembiayaan")
	}
	if !in.Consent {
		return nil, errors.New("pengajuan hanya bisa diteruskan bila Anda menyetujui berbagi data ke " + p.Name)
	}
	if in.Amount < 1_000_000 || in.Amount > 200_000_000 {
		return nil, errors.New("nominal pembiayaan Rp 1 jt – Rp 200 jt")
	}
	est := Estimates(in.Amount, in.Tenor)
	var monthly int64
	for _, e := range est {
		if e.Partner.Slug == p.Slug {
			monthly = e.Monthly
		}
	}
	if monthly == 0 {
		return nil, errors.New("tenor tidak tersedia di mitra ini")
	}
	if len(in.Name) < 3 || len(in.Phone) < 8 {
		return nil, errors.New("isi nama dan nomor WhatsApp")
	}
	now := time.Now()
	a := &FinancingApplication{Code: NewCode("PBY", 6), UserID: in.UserID, PartnerSlug: p.Slug, BookingID: in.BookingID,
		TravelName: truncate(in.TravelName, 120), PackageName: truncate(in.PackageName, 200), PackageKey: truncate(in.PackageKey, 300),
		Amount: in.Amount, Tenor: in.Tenor, EstMonthly: monthly, Name: truncate(in.Name, 80), Phone: truncate(in.Phone, 30),
		Job: truncate(in.Job, 60), Income: in.Income, City: truncate(in.City, 60), ConsentAt: &now, Status: "diajukan", CreatedAt: now}
	if err := DB.Create(a).Error; err != nil {
		return nil, err
	}
	Notify("partner:"+p.Slug, "email", p.Name, "Rujukan baru dari umrohku: "+a.Code,
		fmt.Sprintf("%s (%s) mengajukan pembiayaan umroh Rp %s, tenor %d bulan, untuk %s. Data dibagikan dengan persetujuan jamaah (UU PDP).",
			a.Name, a.City, Rp(a.Amount), a.Tenor, a.PackageName), "/mitra-pembiayaan")
	Notify(UserAudience(a.UserID), "wa", a.Phone, "Pengajuan pembiayaan terkirim",
		fmt.Sprintf("Pengajuan %s sudah diteruskan ke %s. Status bisa dipantau di akun Anda. Angka resmi (margin/ujrah, angsuran) datang dari mitra, bukan dari umrohku.", a.Code, p.Name), "/akun/pembiayaan")
	return a, nil
}

func UserFinancing(userID uint) []FinancingApplication {
	var as []FinancingApplication
	DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&as)
	return as
}

func PartnerApplications(slug string) []FinancingApplication {
	var as []FinancingApplication
	// Pengajuan yang persetujuannya dicabut tidak lagi terlihat oleh mitra.
	DB.Where("partner_slug = ? AND consent_revoked_at IS NULL", slug).Order("created_at DESC").Find(&as)
	return as
}

func AllApplications() []FinancingApplication {
	var as []FinancingApplication
	DB.Order("created_at DESC").Limit(200).Find(&as)
	return as
}

// RevokeConsent: jamaah mencabut persetujuan berbagi data; pengajuan dibatalkan.
func RevokeConsent(userID, id uint) error {
	var a FinancingApplication
	if DB.Where("id = ? AND user_id = ?", id, userID).Take(&a).Error != nil {
		return errors.New("pengajuan tidak ditemukan")
	}
	if a.Status != "diajukan" && a.Status != "diproses" {
		return errors.New("pengajuan sudah diputuskan mitra")
	}
	now := time.Now()
	a.ConsentRevokedAt, a.Status, a.Note = &now, "dibatalkan", "Persetujuan berbagi data dicabut jamaah"
	DB.Save(&a)
	Notify("partner:"+a.PartnerSlug, "email", "", "Persetujuan dicabut: "+a.Code, "Jamaah mencabut persetujuan berbagi data. Hapus data pengajuan ini dari sistem mitra.", "/mitra-pembiayaan")
	return nil
}

// PartnerDecide: diproses | disetujui | ditolak | dicairkan.
func PartnerDecide(slug string, id uint, status, note string) error {
	var a FinancingApplication
	if DB.Where("id = ? AND partner_slug = ? AND consent_revoked_at IS NULL", id, slug).Take(&a).Error != nil {
		return errors.New("pengajuan tidak ditemukan")
	}
	ok := map[string][]string{"diproses": {"diajukan"}, "disetujui": {"diajukan", "diproses"}, "ditolak": {"diajukan", "diproses"}, "dicairkan": {"disetujui"}}
	valid := false
	for _, from := range ok[status] {
		valid = valid || a.Status == from
	}
	if !valid {
		return errors.New("perubahan status tidak valid")
	}
	now := time.Now()
	a.Status, a.Note = status, truncate(note, 300)
	p := PartnerBySlug(slug)
	switch status {
	case "disetujui", "ditolak":
		a.DecidedAt = &now
	case "dicairkan":
		a.DisbursedAt = &now
		if p != nil {
			a.ReferralFee = int64(math.Round(float64(a.Amount) * p.FeePct / 100))
		}
		// Dana dicairkan mitra langsung ke rekening penampungan travel.
		if a.BookingID != 0 {
			if b, err := GetBooking(a.BookingID); err == nil && b.UserID == a.UserID && b.Status == "dp_lunas" {
				if pay, err := CreatePelunasan(b, "va"); err == nil {
					markPaid(pay, b, now)
				}
			}
		}
	}
	DB.Save(&a)
	pname := slug
	if p != nil {
		pname = p.Name
	}
	msg := map[string]string{
		"diproses":  "Pengajuan %s sedang diproses %s.",
		"disetujui": "Alhamdulillah, pengajuan %s disetujui %s. Mitra akan menghubungi Anda untuk akad.",
		"ditolak":   "Mohon maaf, pengajuan %s belum disetujui %s. Anda tetap bisa memakai jalur cicil sebelum berangkat (tanpa utang).",
		"dicairkan": "Dana pembiayaan %s sudah dicairkan %s langsung ke rekening penampungan travel.",
	}[status]
	Notify(UserAudience(a.UserID), "wa", a.Phone, "Status pembiayaan: "+StatusLabel(status), fmt.Sprintf(msg, a.Code, pname), "/akun/pembiayaan")
	return nil
}
