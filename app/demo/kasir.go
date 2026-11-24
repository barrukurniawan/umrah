package demo

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"umrah/app/qris"
)

// Mode kasir (pay-at-shop): jamaah belanja langsung di toko, kasir mengetik total lalu menekan
// "Buat QR". QR dinamisnya sama dengan pesanan pickup (merchantQR); jamaah membayar dari aplikasi
// pembayaran apa pun. Di produk nyata QR diterbitkan payment gateway berizin dan dana di-split otomatis
// ke toko & komisi umrohku. Di demo: merchant fiktif, tidak ada dana yang berpindah.

const (
	KasirMinSAR int64 = 1
	KasirMaxSAR int64 = 2000 // ±Rp 8,6 jt, di bawah batas QRIS Rp 10 jt
)

type StoreCharge struct {
	ID           uint   `gorm:"primaryKey"`
	Code         string `gorm:"size:16;uniqueIndex"` // KSR-XXXXXX
	MerchantSlug string `gorm:"size:40;index"`
	AmountSAR    int64
	AmountIDR    int64
	Rate         int64
	Note         string `gorm:"size:120"`
	Status       string `gorm:"size:20;index"` // menunggu_bayar | dibayar | kedaluwarsa | batal
	QRPayload    string `gorm:"size:400"`
	PaidVia      string `gorm:"size:80"`
	ExpiresAt    time.Time
	PaidAt       *time.Time
	CreatedAt    time.Time `gorm:"index"`
}

func init() { RegisterModels(&StoreCharge{}) }

func (c StoreCharge) Waiting() bool       { return c.Status == "menunggu_bayar" }
func (c StoreCharge) Merchant() *Merchant { return MerchantBySlug(c.MerchantSlug) }
func (c StoreCharge) StatusText() string {
	switch c.Status {
	case "menunggu_bayar":
		return "Menunggu pembayaran"
	case "dibayar":
		return "Lunas"
	case "kedaluwarsa":
		return "Kedaluwarsa"
	case "batal":
		return "Dibatalkan"
	}
	return c.Status
}

var (
	ErrKasirAmount = fmt.Errorf("isi total belanja %d–%d SAR", KasirMinSAR, KasirMaxSAR)
	ErrKasirState  = errors.New("status transaksi sudah berubah")
)

func expireCharge(c *StoreCharge) {
	if c != nil && c.Status == "menunggu_bayar" && time.Now().After(c.ExpiresAt) {
		if DB.Model(&StoreCharge{}).Where("id = ? AND status = ?", c.ID, "menunggu_bayar").Update("status", "kedaluwarsa").RowsAffected > 0 {
			c.Status = "kedaluwarsa"
		}
	}
}

// CreateCharge: kasir membuat tagihan QR untuk belanja langsung di toko.
func CreateCharge(slug string, sar int64, note string) (*StoreCharge, error) {
	m := MerchantBySlug(slug)
	if m == nil {
		return nil, ErrMarketStore
	}
	if sar < KasirMinSAR || sar > KasirMaxSAR {
		return nil, ErrKasirAmount
	}
	now := time.Now()
	c := &StoreCharge{Code: uniqueCode(&StoreCharge{}, func() string { return "KSR-" + randCode(6) }), MerchantSlug: slug,
		AmountSAR: sar, Rate: SARRateDemo, AmountIDR: sar * SARRateDemo, Note: clip(strings.TrimSpace(note), 120),
		Status: "menunggu_bayar", ExpiresAt: now.Add(MarketQRValid), CreatedAt: now}
	c.QRPayload = merchantQR(m, c.Code, c.AmountIDR, "KASIR01")
	if err := DB.Create(c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

func ChargeFor(slug, code string) *StoreCharge {
	var c StoreCharge
	if DB.Where("merchant_slug = ? AND code = ?", slug, code).Take(&c).Error != nil {
		return nil
	}
	expireCharge(&c)
	return &c
}

// ChargesToday: transaksi kasir hari ini (waktu Saudi), terbaru dulu.
func ChargesToday(slug string) []StoreCharge {
	now := time.Now().In(AST)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, AST)
	var cs []StoreCharge
	DB.Where("merchant_slug = ? AND created_at >= ?", slug, start).Order("created_at DESC").Limit(50).Find(&cs)
	for i := range cs {
		expireCharge(&cs[i])
	}
	return cs
}

func PayCharge(c *StoreCharge, via string) error {
	expireCharge(c)
	if c.Status == "kedaluwarsa" {
		return ErrMarketExpired
	}
	now := time.Now()
	if DB.Model(&StoreCharge{}).Where("id = ? AND status = ?", c.ID, "menunggu_bayar").
		Updates(map[string]any{"status": "dibayar", "paid_at": now, "paid_via": clip(via, 80)}).RowsAffected == 0 {
		return ErrKasirState
	}
	c.Status, c.PaidAt, c.PaidVia = "dibayar", &now, via
	return nil
}

func CancelCharge(c *StoreCharge) error {
	if DB.Model(&StoreCharge{}).Where("id = ? AND status = ?", c.ID, "menunggu_bayar").Update("status", "batal").RowsAffected == 0 {
		return ErrKasirState
	}
	c.Status = "batal"
	return nil
}

// ChargeNMID: NMID dari QR (untuk tampilan).
func (c StoreCharge) NMID() string {
	if q, err := qris.Decode(c.QRPayload); err == nil {
		return q.NMID()
	}
	return ""
}
