package demo

import (
	"crypto/rand"
	"errors"
	"math"
	"strings"
	"time"
)

// Promo (diskon DP & cashback pelunasan) selalu dibiayai umrohku dari komisinya. Harga travel
// tidak pernah diubah, dan promo tidak bisa dicairkan menjadi uang/saldo.

type PromoQuote struct {
	Rule   *PromoRule
	Amount int64
	Error  string
}

func (r PromoRule) value(price int64) int64 {
	v := r.Amount
	if r.Percent > 0 {
		v = int64(math.Round(float64(price) * r.Percent / 100))
	}
	if r.Cap > 0 && v > r.Cap {
		v = r.Cap
	}
	return v
}

func (r PromoRule) usable(travelSlug string, price int64, now time.Time) error {
	switch {
	case !r.Active:
		return errors.New("promo tidak aktif")
	case now.Before(r.StartsAt):
		return errors.New("promo belum dimulai")
	case now.After(r.EndsAt):
		return errors.New("promo sudah berakhir")
	case r.Quota > 0 && r.Used >= r.Quota:
		return errors.New("kuota promo sudah habis")
	case r.TravelSlug != "" && r.TravelSlug != travelSlug:
		return errors.New("promo tidak berlaku untuk travel ini")
	case price < r.MinPrice:
		return errors.New("harga paket di bawah minimum promo (Rp " + Rp(r.MinPrice) + ")")
	}
	return nil
}

// QuotePromos menghitung potongan untuk satu booking: maksimal satu diskon_dp dan satu
// cashback_pelunasan. Promo otomatis (tanpa kode) dipilih yang nilainya terbesar; kode yang
// diketik pengguna menggantikan promo otomatis sejenis.
func QuotePromos(travelSlug string, total int64, code string) (dp PromoQuote, cb PromoQuote, codeErr string) {
	now := time.Now()
	var rules []PromoRule
	DB.Where("active = ? AND code = ?", true, "").Find(&rules)
	for i := range rules {
		r := rules[i]
		if r.usable(travelSlug, total, now) != nil {
			continue
		}
		q := PromoQuote{Rule: &r, Amount: r.value(total)}
		if r.Kind == "diskon_dp" && q.Amount > dp.Amount {
			dp = q
		}
		if r.Kind == "cashback_pelunasan" && q.Amount > cb.Amount {
			cb = q
		}
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	if code != "" {
		var r PromoRule
		if DB.Where("code = ?", code).Take(&r).Error != nil {
			codeErr = "Kode promo tidak ditemukan"
		} else if err := r.usable(travelSlug, total, now); err != nil {
			codeErr = "Kode " + code + ": " + err.Error()
		} else {
			q := PromoQuote{Rule: &r, Amount: r.value(total)}
			if r.Kind == "diskon_dp" {
				dp = q
			} else {
				cb = q
			}
		}
	}
	return
}

func PromoRules() []PromoRule {
	var rs []PromoRule
	DB.Order("active DESC, ends_at DESC").Find(&rs)
	return rs
}

// PromoMonthly: biaya promo vs komisi kotor per bulan (6 bulan terakhir) untuk konsol admin.
type PromoMonth struct {
	Label      string
	Commission int64
	Promo      int64
	PctC, PctP int
}

func PromoMonthly(n int) []PromoMonth {
	p := CurrentPeriod()
	var out []PromoMonth
	var mx int64 = 1
	for i := 0; i < n; i++ {
		var gross, promo int64
		for _, t := range PartnerTravels() {
			inv := BuildInvoice(t, p)
			for _, c := range inv.Commissions {
				gross += c.Gross
				promo += c.Promo
			}
		}
		a, _ := periodRange(p)
		out = append([]PromoMonth{{Label: bulan[a.Month()] + " " + a.Format("06"), Commission: gross, Promo: promo}}, out...)
		mx = max(mx, gross, promo)
		p = PrevPeriod(p)
	}
	for i := range out {
		out[i].PctC = int(out[i].Commission * 100 / mx)
		out[i].PctP = int(out[i].Promo * 100 / mx)
	}
	return out
}

// ---- kode acak yang mudah dibaca ----

const codeAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

func randCode(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = codeAlphabet[int(b[i])%len(codeAlphabet)]
	}
	return string(b)
}

// NewCode: "UMK-7K2Q" dll. dengan prefix.
func NewCode(prefix string, n int) string { return prefix + "-" + randCode(n) }
