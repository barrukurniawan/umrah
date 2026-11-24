package handlers

import (
	"errors"
	"strings"

	"umrah/app/accounts"
	"umrah/app/demo"
	"umrah/app/qris"

	"github.com/gofiber/fiber/v2"
)

// Bayar QRIS + navigasi bawah (Beranda · Market · QR · Riwayat · Profil), khusus mode demo.
// Scan & decode QR sungguhan; pembayaran disimulasikan (lihat app/demo/qrpay.go).

func registerDemoQRIS(app *fiber.App) {
	// Web ini tidak memindai QR (hanya menerbitkan QR tagihan, seperti marketplace). Pemindai lama
	// (QRScanPage/QRDecodePage/QRPay/QRDecodeAPI) dimatikan; tautan lama diarahkan ke beranda.
	gone := func(c *fiber.Ctx) error { return c.Redirect("/", fiber.StatusSeeOther) }
	app.Get("/bayar", gone)
	app.Post("/bayar/decode", gone)
	app.Post("/bayar/qris", gone)
	app.Post("/api/qris/decode", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusGone) })
	app.Get("/riwayat", HistoryPage)
	app.Get("/riwayat/qris/:code", QRReceipt)
	app.Get("/profil", ProfilePage)
}

func QRScanPage(c *fiber.Ctx) error {
	return c.Render("demo/qris_scan", viewer(c, fiber.Map{"Title": "Scan QRIS", "Sample": demo.SampleQR(false), "SampleDyn": demo.SampleQR(true), "SampleSA": demo.SampleQRSaudi(false), "SampleSADyn": demo.SampleQRSaudi(true),
		"Err": c.Query("err")}))
}

type qrView struct {
	*qris.QR
	Acquirer, Criteria, MCCLabel, PAN, NMIDv string
	Fixed                                    bool // nominal dari QR (dinamis)
	Cross                                    bool // simulasi lintas negara (SAR)
	SAR                                      string
	IDR, Rate                                int64
}

func viewQR(q *qris.QR) qrView {
	v := qrView{QR: q, Acquirer: q.AcquirerName(), Criteria: qris.CriteriaLabel(q.Criteria()), MCCLabel: qris.MCCLabel(q.MCC),
		PAN: qris.MaskPAN(q.Acquirer().PAN), NMIDv: q.NMID(), Fixed: q.Dynamic && q.Amount > 0}
	if q.IsSAR() {
		v.Cross, v.Rate = true, demo.SARRateDemo
		v.Fixed = q.Dynamic && q.AmountMinor > 0
		v.SAR, v.IDR = demo.FormatMinor(q.AmountMinor), demo.SARToIDR(q.AmountMinor)
	}
	return v
}

// POST /bayar/decode (qr): baca QR hasil scan, tampilkan konfirmasi bayar.
func QRDecodePage(c *fiber.Ctx) error {
	raw := c.FormValue("qr")
	if len(raw) > 1024 {
		raw = raw[:1024]
	}
	q, err := qris.Decode(raw)
	if err != nil {
		m := fiber.Map{"Title": "QR tidak bisa dibayar", "Raw": qris.Clean(raw), "Reason": err.Error(), "HideChat": true}
		if errors.Is(err, qris.ErrNotEMV) {
			m["NotQRIS"] = true
			m["IsURL"] = strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://")
		}
		return demoRender(c, "demo/qris_gagal", viewer(c, m))
	}
	m := fiber.Map{"Title": "Bayar " + q.MerchantName, "Q": viewQR(q), "Raw": q.Payload, "Limit": demo.QRISLimit, "Err": c.Query("err"), "HideChat": true}
	if u := accounts.Current(c); u != nil {
		m["Fund"] = demo.Fund(u.ID, u.Name)
	}
	return demoRender(c, "demo/qris_bayar", viewer(c, m))
}

// POST /bayar/qris: QR didecode ulang di server (isi dari browser tidak dipercaya), lalu bayar.
func QRPay(c *fiber.Ctx) error {
	u, err := jamaahOrLogin(c)
	if u == nil {
		return err
	}
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	q, derr := qris.Decode(c.FormValue("qr"))
	if derr != nil {
		return c.Redirect("/bayar?err="+urlq(derr.Error()), fiber.StatusSeeOther)
	}
	p, perr := demo.PayQR(demo.PayInput{UserID: u.ID, Holder: u.Name, QR: q, Amount: rupiahInput(c.FormValue("nominal")), ForeignMinor: sarInput(c.FormValue("nominal_sar")),
		Fee: rupiahInput(c.FormValue("tip")), PIN: c.FormValue("pin")})
	if perr != nil {
		m := fiber.Map{"Title": "Bayar " + q.MerchantName, "Q": viewQR(q), "Raw": q.Payload, "Limit": demo.QRISLimit, "Err": perr.Error(),
			"Fund": demo.Fund(u.ID, u.Name), "Nominal": c.FormValue("nominal"), "NominalSAR": c.FormValue("nominal_sar"), "HideChat": true}
		return demoRender(c, "demo/qris_bayar", viewer(c, m))
	}
	return c.Redirect("/riwayat/qris/"+p.Code+"?baru=1", fiber.StatusSeeOther)
}

// POST /api/qris/decode {"qr_content": "..."}: bentuk respons mengikuti QrMobileDecodeResult
// (POST /qr/mobile/decode di core-payment) supaya nanti mudah diganti ke layanan issuer sungguhan.
func QRDecodeAPI(c *fiber.Ctx) error {
	var req struct {
		QrContent string `json:"qr_content"`
	}
	if err := c.BodyParser(&req); err != nil || req.QrContent == "" || len(req.QrContent) > 1024 {
		return c.Status(400).JSON(fiber.Map{"response_code": "INVALID_PAYLOAD", "response_message": "qr_content wajib diisi"})
	}
	q, err := qris.Decode(req.QrContent)
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"response_code": "INVALID_QR", "response_message": err.Error()})
	}
	qrisType := "STATIC"
	if q.Dynamic {
		qrisType = "DYNAMIC"
	}
	fee := q.Fee(q.Amount)
	return c.JSON(fiber.Map{"response_code": "SUCCESS", "data": fiber.Map{
		"qr_type": "QRIS_MPM", "qris_type": qrisType, "merchant_name": q.MerchantName, "merchant_city": q.MerchantCity,
		"merchant_country": q.Country, "merchant_pan": qris.MaskPAN(q.Acquirer().PAN), "merchant_nmid": q.NMID(),
		"merchant_criteria": q.Criteria(), "merchant_category": q.MCC, "acquirer_name": q.AcquirerName(), "currency_code": q.Currency,
		"transaction_amount": q.Amount, "service_charge_option": q.TipIndicator, "service_charge_amount": fee, "total_amount": q.Amount + fee,
		"transaction_limit": demo.QRISLimit, "reference_label": q.Reference, "terminal_id": q.TerminalID, "fund_option": []fiber.Map{{"payMethod": "BANK_ACCOUNT"}},
	}})
}

// sarInput: "35,50" / "35.50" / "35" -> halalah (3550 / 3550 / 3500). Maksimal dua desimal.
func sarInput(s string) int64 {
	s = strings.TrimSpace(strings.Replace(s, ",", ".", 1))
	whole, frac, _ := strings.Cut(s, ".")
	if len(frac) > 2 {
		frac = frac[:2]
	}
	for len(frac) < 2 {
		frac += "0"
	}
	n := rupiahInput(whole)*100 + rupiahInput(frac)
	if n > 100_000_000 {
		return 0
	}
	return n
}

func QRReceipt(c *fiber.Ctx) error {
	u, err := jamaahOrLogin(c)
	if u == nil {
		return err
	}
	p := demo.QRPaymentFor(u.ID, c.Params("code"))
	if p == nil {
		return c.Redirect("/riwayat")
	}
	return demoRender(c, "demo/qris_struk", viewer(c, fiber.Map{"Title": "Pembayaran " + p.MerchantName, "P": p, "New": c.Query("baru") == "1",
		"MCCLabel": qris.MCCLabel(p.MCC), "Fund": demo.Fund(u.ID, u.Name), "Market": demo.MarketOrderByQRRef(p.Code)}))
}

func HistoryPage(c *fiber.Ctx) error {
	u, err := jamaahOrLogin(c)
	if u == nil {
		return err
	}
	kind := c.Query("jenis")
	if kind != "qris" && kind != "umroh" && kind != "belanja" {
		kind = ""
	}
	items := demo.History(u.ID, kind, 100)
	// Kelompokkan per tanggal.
	type group struct {
		Label string
		Items []demo.HistoryItem
	}
	var groups []group
	for _, it := range items {
		l := demo.Tgl(it.At)
		if demo.Tgl(it.At) == demo.Tgl(demo.Today()) {
			l = "Hari ini"
		}
		if len(groups) == 0 || groups[len(groups)-1].Label != l {
			groups = append(groups, group{Label: l})
		}
		groups[len(groups)-1].Items = append(groups[len(groups)-1].Items, it)
	}
	return demoRender(c, "demo/riwayat", viewer(c, fiber.Map{"Title": "Riwayat", "Groups": groups, "Kind": kind}))
}

func ProfilePage(c *fiber.Ctx) error {
	u := accounts.Current(c)
	m := fiber.Map{"Title": "Profil"}
	if u != nil {
		m["Fund"] = demo.Fund(u.ID, u.Name)
		var nb, nf, ne int64
		demo.DB.Model(&demo.Booking{}).Where("user_id = ? AND status <> ?", u.ID, "batal").Count(&nb)
		demo.DB.Model(&demo.FinancingApplication{}).Where("user_id = ?", u.ID).Count(&nf)
		demo.DB.Model(&demo.DrawEntry{}).Where("user_id = ? AND void = ?", u.ID, false).Count(&ne)
		m["NB"], m["NF"], m["NE"] = nb, nf, ne
		m["Goals"] = accounts.CountGoals(u.ID)
	}
	return demoRender(c, "demo/profil", viewer(c, m))
}
