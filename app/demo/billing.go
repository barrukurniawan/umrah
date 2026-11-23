package demo

import (
	"math"
	"sort"
	"strconv"
	"time"
)

// Tagihan bulanan travel mitra: lead valid × CPL + komisi booking (dikurangi promo yang
// ditanggung umrohku) + langganan portal + posisi unggulan. Isi dihitung dari data; yang disimpan
// hanya status bayarnya (InvoiceStatus).

type InvoiceLine struct {
	Label  string
	Detail string
	Qty    int
	Unit   int64
	Amount int64
	Minus  bool
}

type CommissionRow struct {
	Booking    Booking
	Gross      int64
	Promo      int64
	Net        int64
	PromoExtra int64 // promo melebihi komisi: biaya promosi umrohku
}

type Invoice struct {
	Travel      Travel
	Period      string
	Label       string
	Number      string
	Lines       []InvoiceLine
	Commissions []CommissionRow
	LeadsValid  int
	LeadsAll    int
	Total       int64
	PromoCost   int64 // promo yang ditanggung umrohku (dari komisi + kelebihan)
	PromoExtra  int64
	Paid        *time.Time
	DueDate     string
}

func BuildInvoice(t Travel, period string) Invoice {
	inv := Invoice{Travel: t, Period: period, Label: PeriodLabel(period)}
	a, b := periodRange(period)
	inv.Number = "INV/UMK/" + a.Format("200601") + "/" + t.Slug
	inv.DueDate = b.AddDate(0, 0, 9).Format("2006-01-02")

	// Lead
	var all, valid int64
	DB.Model(&Lead{}).Where("travel_slug = ? AND created_at >= ? AND created_at < ?", t.Slug, a, b).Count(&all)
	DB.Model(&Lead{}).Where("travel_slug = ? AND created_at >= ? AND created_at < ? AND billable = ?", t.Slug, a, b, true).Count(&valid)
	inv.LeadsAll, inv.LeadsValid = int(all), int(valid)
	inv.Lines = append(inv.Lines, InvoiceLine{Label: "Lead terverifikasi", Detail: strconv.Itoa(int(all-valid)) + " lead duplikat/tidak valid tidak ditagih",
		Qty: int(valid), Unit: Cfg.CPL, Amount: valid * Cfg.CPL})

	// Komisi booking (DP lunas di periode ini, belum batal)
	var bs []Booking
	DB.Where("travel_slug = ? AND paid_at >= ? AND paid_at < ? AND status <> ?", t.Slug, a, b, "batal").Order("paid_at").Find(&bs)
	var gross, net int64
	for _, bk := range bs {
		g := int64(math.Round(float64(bk.Total) * bk.KomisiPersen / 100))
		var promo int64
		DB.Model(&PromoUse{}).Where("booking_id = ? AND status <> ?", bk.ID, "batal").Select("COALESCE(SUM(amount),0)").Scan(&promo)
		row := CommissionRow{Booking: bk, Gross: g, Promo: promo, Net: g - promo}
		if row.Net < 0 {
			row.PromoExtra, row.Net = -row.Net, 0
		}
		inv.PromoCost += promo
		inv.PromoExtra += row.PromoExtra
		gross += g
		net += row.Net
		inv.Commissions = append(inv.Commissions, row)
	}
	if len(bs) > 0 {
		inv.Lines = append(inv.Lines, InvoiceLine{Label: "Komisi booking (kotor)", Detail: strconv.Itoa(len(bs)) + " booking DP lunas", Qty: len(bs), Amount: gross})
		if gross-net > 0 {
			inv.Lines = append(inv.Lines, InvoiceLine{Label: "Promo ditanggung umrohku", Detail: "Diskon DP & cashback jamaah, dipotong dari komisi", Amount: gross - net, Minus: true})
		}
	}

	// Langganan portal
	var subs []SubscriptionLog
	DB.Where("travel_slug = ? AND created_at >= ? AND created_at < ?", t.Slug, a, b).Find(&subs)
	if len(subs) > 0 {
		for _, s := range subs {
			p := PlanByKey(s.Plan)
			inv.Lines = append(inv.Lines, InvoiceLine{Label: "Langganan portal " + p.Name, Detail: Tgl(s.From) + " – " + Tgl(s.Until), Qty: s.Months, Unit: p.Price, Amount: s.Amount})
		}
	} else if p := PlanByKey(t.Plan); p.Price > 0 {
		inv.Lines = append(inv.Lines, InvoiceLine{Label: "Langganan portal " + p.Name, Detail: "Perpanjangan otomatis bulanan", Qty: 1, Unit: p.Price, Amount: p.Price})
	}

	// Posisi unggulan
	if days, cost := PlacementCost(t.Name, period); days > 0 {
		inv.Lines = append(inv.Lines, InvoiceLine{Label: "Posisi unggulan \"Promo Spesial\"", Detail: strconv.Itoa(days) + " hari tayang", Qty: days, Amount: cost})
	}

	for _, l := range inv.Lines {
		if l.Minus {
			inv.Total -= l.Amount
		} else {
			inv.Total += l.Amount
		}
	}
	var st InvoiceStatus
	if DB.Where("travel_slug = ? AND period = ?", t.Slug, period).Take(&st).Error == nil {
		inv.Paid = st.PaidAt
	}
	return inv
}

func MarkInvoicePaid(slug, period string) {
	now := time.Now()
	var st InvoiceStatus
	if DB.Where("travel_slug = ? AND period = ?", slug, period).Take(&st).Error != nil {
		st = InvoiceStatus{TravelSlug: slug, Period: period}
	}
	st.PaidAt = &now
	DB.Save(&st)
}

// Revenue: ringkasan pendapatan umrohku per sumber dalam satu periode (konsol admin).
type Revenue struct {
	Period      string
	Label       string
	Leads       int64
	Commission  int64 // komisi bersih
	PromoCost   int64
	PromoExtra  int64
	Subs        int64
	Placement   int64
	Referral    int64
	Total       int64
	ByTravel    []Invoice
	Bookings    int
	GMV         int64 // nilai paket yang dipesan lewat umrohku (DP lunas)
}

func BuildRevenue(period string) Revenue {
	r := Revenue{Period: period, Label: PeriodLabel(period)}
	for _, t := range PartnerTravels() {
		inv := BuildInvoice(t, period)
		r.ByTravel = append(r.ByTravel, inv)
		for _, l := range inv.Lines {
			switch {
			case l.Label == "Lead terverifikasi":
				r.Leads += l.Amount
			case len(l.Label) > 9 && l.Label[:9] == "Langganan":
				r.Subs += l.Amount
			case len(l.Label) > 6 && l.Label[:6] == "Posisi":
				r.Placement += l.Amount
			}
		}
		for _, c := range inv.Commissions {
			r.Commission += c.Net
			r.GMV += c.Booking.Total
			r.Bookings++
		}
		r.PromoCost += inv.PromoCost
		r.PromoExtra += inv.PromoExtra
	}
	a, b := periodRange(period)
	DB.Model(&FinancingApplication{}).Where("disbursed_at >= ? AND disbursed_at < ?", a, b).Select("COALESCE(SUM(referral_fee),0)").Scan(&r.Referral)
	r.Total = r.Leads + r.Commission + r.Subs + r.Placement + r.Referral - r.PromoExtra
	return r
}

// ---- util kecil ----

func fmtFloat(f float64, prec int) string { return strconv.FormatFloat(f, 'f', prec, 64) }

func sortStats(s []PackageStat) {
	sort.Slice(s, func(i, j int) bool {
		if s[i].Leads != s[j].Leads {
			return s[i].Leads > s[j].Leads
		}
		return s[i].Package < s[j].Package
	})
}
