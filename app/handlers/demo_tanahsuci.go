package handlers

import (
	"umrah/app/accounts"
	"umrah/app/demo"

	"github.com/gofiber/fiber/v2"
)

// Fase 3 (umrah mandiri, prototipe) & Fase 4 (layanan Tanah Suci + portal toko mitra).
// Semua data dummy; pembayaran lewat payments.MockGateway (lihat app/demo/shop.go).

// tsJamaah: jamaah yang login, atau render kartu "masuk sebagai Jamaah" (u == nil).
func tsJamaah(c *fiber.Ctx, title string) (*accounts.User, error) {
	if u := accounts.Current(c); u != nil {
		return u, nil
	}
	return nil, c.Render("demo/tanahsuci_masuk", viewer(c, fiber.Map{"Title": title, "Next": c.OriginalURL()}))
}

// tsPostUser: untuk POST, wajib same-origin + login. ok=false berarti respons sudah dikirim.
func tsPostUser(c *fiber.Ctx, back string) (*accounts.User, bool, error) {
	if !accounts.SameOrigin(c) {
		return nil, false, c.SendStatus(fiber.StatusForbidden)
	}
	u := accounts.Current(c)
	if u == nil {
		return nil, false, c.Redirect(back, fiber.StatusSeeOther)
	}
	return u, true, nil
}

func tsWaTo(u *accounts.User) string { return u.Name + " (WA)" }

// ============================================================================================
// Fase 3: umrah mandiri
// ============================================================================================

func registerDemoMandiri(app *fiber.App) {
	app.Get("/mandiri", MandiriPage)
	app.Post("/mandiri/checklist", MandiriSaveChecklist)
	app.Get("/mandiri/katalog", MandiriCatalogPage)
	app.Get("/mandiri/keranjang", MandiriCartPage)
	app.Post("/mandiri/keranjang", MandiriCartUpdate)
	app.Post("/mandiri/checkout", MandiriCheckout)
	app.Get("/mandiri/pesanan/:code", MandiriOrderPage)
	app.Post("/mandiri/pesanan/:code/bayar", MandiriOrderPay)
}

func MandiriPage(c *fiber.Ctx) error {
	u, err := tsJamaah(c, "Umrah mandiri")
	if u == nil {
		return err
	}
	done := demo.MandiriChecklist(u.ID)
	total := demo.MandiriCheckTotal()
	return c.Render("demo/mandiri", viewer(c, fiber.Map{"Title": "Umrah mandiri (prototipe)",
		"Steps": demo.MandiriSteps, "Done": done, "DoneN": len(done), "TotalN": total,
		"Pct": len(done) * 100 / max(total, 1), "Saved": c.Query("ok") == "1",
		"CartN": demo.CartCount(u.ID, "mandiri"), "Orders": demo.MandiriOrdersFor(u.ID)}))
}

func MandiriSaveChecklist(c *fiber.Ctx) error {
	u, ok, err := tsPostUser(c, "/mandiri")
	if !ok {
		return err
	}
	var keys []string
	for _, v := range c.Request().PostArgs().PeekMulti("done") {
		if len(keys) >= 50 {
			break
		}
		keys = append(keys, string(v))
	}
	_ = demo.SaveMandiriChecklist(u.ID, keys)
	return c.Redirect("/mandiri?ok=1#checklist", fiber.StatusSeeOther)
}

func MandiriCatalogPage(c *fiber.Ctx) error {
	u, err := tsJamaah(c, "Katalog umrah mandiri")
	if u == nil {
		return err
	}
	kind := c.Query("jenis", "hotel")
	if len(demo.MandiriByKind(kind)) == 0 {
		kind = "hotel"
	}
	return c.Render("demo/mandiri_katalog", viewer(c, fiber.Map{"Title": "Katalog umrah mandiri (prototipe)",
		"Kind": kind, "Kinds": demo.MandiriKinds, "Items": demo.MandiriByKind(kind),
		"InCart": demo.CartQty(u.ID, "mandiri"), "CartN": demo.CartCount(u.ID, "mandiri"), "Added": c.Query("ok") == "1"}))
}

func mandiriCartRender(c *fiber.Ctx, u *accounts.User, errMsg string) error {
	lines, total := demo.Cart(u.ID, "mandiri")
	form := fiber.Map{"Contact": c.FormValue("contact", u.Name), "Phone": c.FormValue("phone"), "Depart": c.FormValue("depart", demo.AddDays(demo.Today(), 60))}
	return c.Render("demo/mandiri_keranjang", viewer(c, fiber.Map{"Title": "Keranjang umrah mandiri (prototipe)",
		"Lines": lines, "Total": total, "Error": errMsg, "Form": form, "MinDate": demo.AddDays(demo.Today(), 1)}))
}

func MandiriCartPage(c *fiber.Ctx) error {
	u, err := tsJamaah(c, "Keranjang umrah mandiri")
	if u == nil {
		return err
	}
	return mandiriCartRender(c, u, "")
}

// POST /mandiri/keranjang: key, qty, mode=add|set, back=katalog|keranjang
func MandiriCartUpdate(c *fiber.Ctx) error {
	u, ok, err := tsPostUser(c, "/mandiri")
	if !ok {
		return err
	}
	key := c.FormValue("key")
	add := c.FormValue("mode") == "add"
	_ = demo.CartSet(u.ID, "mandiri", key, atoiDef(c.FormValue("qty"), 1), add)
	if c.FormValue("back") == "katalog" {
		kind := "hotel"
		if p := demo.MandiriProduct(key); p != nil {
			kind = p.Kind
		}
		return c.Redirect("/mandiri/katalog?jenis="+kind+"&ok=1", fiber.StatusSeeOther)
	}
	return c.Redirect("/mandiri/keranjang", fiber.StatusSeeOther)
}

func MandiriCheckout(c *fiber.Ctx) error {
	u, ok, err := tsPostUser(c, "/mandiri/keranjang")
	if !ok {
		return err
	}
	o, e := demo.CreateMandiriOrder(u.ID, c.FormValue("contact"), c.FormValue("phone"), c.FormValue("depart"))
	if e != nil {
		return mandiriCartRender(c, u, e.Error())
	}
	return c.Redirect("/mandiri/pesanan/"+o.Code, fiber.StatusSeeOther)
}

func MandiriOrderPage(c *fiber.Ctx) error {
	u, err := tsJamaah(c, "Pesanan umrah mandiri")
	if u == nil {
		return err
	}
	o := demo.MandiriOrderFor(u.ID, c.Params("code"))
	if o == nil {
		return c.Redirect("/mandiri", fiber.StatusSeeOther)
	}
	return c.Render("demo/mandiri_pesanan", viewer(c, fiber.Map{"Title": "Pesanan " + o.Code, "O": o}))
}

func MandiriOrderPay(c *fiber.Ctx) error {
	u, ok, err := tsPostUser(c, "/mandiri")
	if !ok {
		return err
	}
	if o := demo.MandiriOrderFor(u.ID, c.Params("code")); o != nil {
		_ = demo.PayMandiriOrder(o, u.Email)
		return c.Redirect("/mandiri/pesanan/"+o.Code, fiber.StatusSeeOther)
	}
	return c.Redirect("/mandiri", fiber.StatusSeeOther)
}

// ============================================================================================
// Fase 4: layanan Tanah Suci (jamaah) + portal toko mitra
// ============================================================================================

func registerDemoTanahSuci(app *fiber.App) {
	app.Get("/tanah-suci", TanahSuciHub)

	app.Get("/tanah-suci/oleh-oleh", OlehCatalogPage)
	app.Post("/tanah-suci/oleh-oleh/keranjang", OlehCartUpdate)
	app.Get("/tanah-suci/oleh-oleh/checkout", OlehCheckoutPage)
	app.Post("/tanah-suci/oleh-oleh/checkout", OlehCheckout)
	app.Get("/tanah-suci/oleh-oleh/:code", OlehOrderPage)
	app.Post("/tanah-suci/oleh-oleh/:code/bayar", OlehOrderPay)
	app.Post("/tanah-suci/oleh-oleh/:code/lanjut", OlehOrderAdvance)

	app.Get("/tanah-suci/voucher", VoucherPage)
	app.Post("/tanah-suci/voucher", VoucherBuy)
	app.Get("/tanah-suci/voucher/:code", VoucherDetail)
	app.Post("/tanah-suci/voucher/:code/bayar", VoucherPay)

	app.Get("/tanah-suci/pickup", PickupPage)
	app.Post("/tanah-suci/pickup", PickupCreate)
	app.Get("/tanah-suci/pickup/:code", PickupDetail)

	app.Get("/toko", TokoPage)
	app.Get("/toko/pickup", TokoPage)
	app.Get("/toko/riwayat", TokoPage)
	app.Post("/toko/tukar", TokoRedeem)
	app.Post("/toko/pickup/:id/:to", TokoPickupAdvance)
}

func TanahSuciHub(c *fiber.Ctx) error {
	m := fiber.Map{"Title": "Layanan di Tanah Suci", "Merchants": demo.Merchants, "Rate": demo.Cfg.SAR,
		"Oleh": demo.OlehCatalog[:4]}
	if u := accounts.Current(c); u != nil {
		m["Orders"] = demo.ShopOrdersFor(u.ID)
		m["Vouchers"] = demo.VouchersFor(u.ID)
		m["Pickups"] = demo.PickupsFor(u.ID)
		m["CartN"] = demo.CartCount(u.ID, "oleh")
	}
	return c.Render("demo/tanahsuci", viewer(c, m))
}

// ---- oleh-oleh ----

func OlehCatalogPage(c *fiber.Ctx) error {
	u, err := tsJamaah(c, "Oleh-oleh dikirim ke rumah")
	if u == nil {
		return err
	}
	lines, sub := demo.Cart(u.ID, "oleh")
	return c.Render("demo/tanahsuci_oleh", viewer(c, fiber.Map{"Title": "Oleh-oleh dikirim ke rumah",
		"Items": demo.OlehCatalog, "InCart": demo.CartQty(u.ID, "oleh"), "Lines": lines, "Subtotal": sub,
		"Shipping": demo.ShippingFor(sub), "FreeFrom": demo.FreeShipFrom, "Supplier": demo.Supplier,
		"Added": c.Query("ok") == "1"}))
}

// POST: key, qty, mode=add|set, back=checkout|katalog
func OlehCartUpdate(c *fiber.Ctx) error {
	u, ok, err := tsPostUser(c, "/tanah-suci/oleh-oleh")
	if !ok {
		return err
	}
	_ = demo.CartSet(u.ID, "oleh", c.FormValue("key"), atoiDef(c.FormValue("qty"), 1), c.FormValue("mode") == "add")
	if c.FormValue("back") == "checkout" {
		return c.Redirect("/tanah-suci/oleh-oleh/checkout", fiber.StatusSeeOther)
	}
	return c.Redirect("/tanah-suci/oleh-oleh?ok=1#keranjang", fiber.StatusSeeOther)
}

func olehCheckoutRender(c *fiber.Ctx, u *accounts.User, errMsg string) error {
	lines, sub := demo.Cart(u.ID, "oleh")
	if len(lines) == 0 && errMsg == "" {
		return c.Redirect("/tanah-suci/oleh-oleh", fiber.StatusSeeOther)
	}
	ship := demo.ShippingFor(sub)
	form := fiber.Map{"Recipient": c.FormValue("recipient", u.Name), "Phone": c.FormValue("phone"), "Address": c.FormValue("address"),
		"City": c.FormValue("city"), "Postal": c.FormValue("postal")}
	return c.Render("demo/tanahsuci_checkout", viewer(c, fiber.Map{"Title": "Alamat pengiriman oleh-oleh",
		"Lines": lines, "Subtotal": sub, "Shipping": ship, "Total": sub + ship, "Form": form, "Error": errMsg}))
}

func OlehCheckoutPage(c *fiber.Ctx) error {
	u, err := tsJamaah(c, "Alamat pengiriman")
	if u == nil {
		return err
	}
	return olehCheckoutRender(c, u, "")
}

func OlehCheckout(c *fiber.Ctx) error {
	u, ok, err := tsPostUser(c, "/tanah-suci/oleh-oleh")
	if !ok {
		return err
	}
	o, e := demo.CreateShopOrder(u.ID, demo.Address{Recipient: c.FormValue("recipient"), Phone: c.FormValue("phone"),
		Address: c.FormValue("address"), City: c.FormValue("city"), Postal: c.FormValue("postal")})
	if e != nil {
		return olehCheckoutRender(c, u, e.Error())
	}
	return c.Redirect("/tanah-suci/oleh-oleh/"+o.Code, fiber.StatusSeeOther)
}

func OlehOrderPage(c *fiber.Ctx) error {
	u, err := tsJamaah(c, "Lacak oleh-oleh")
	if u == nil {
		return err
	}
	o := demo.ShopOrderFor(u.ID, c.Params("code"))
	if o == nil {
		return c.Redirect("/tanah-suci/oleh-oleh", fiber.StatusSeeOther)
	}
	return c.Render("demo/tanahsuci_order", viewer(c, fiber.Map{"Title": "Pesanan " + o.Code, "O": o,
		"Next": demo.ShopNextStatus(o.Status), "Supplier": demo.Supplier}))
}

func OlehOrderPay(c *fiber.Ctx) error {
	u, ok, err := tsPostUser(c, "/tanah-suci/oleh-oleh")
	if !ok {
		return err
	}
	if o := demo.ShopOrderFor(u.ID, c.Params("code")); o != nil {
		_ = demo.PayShopOrder(o, tsWaTo(u))
		return c.Redirect("/tanah-suci/oleh-oleh/"+o.Code, fiber.StatusSeeOther)
	}
	return c.Redirect("/tanah-suci/oleh-oleh", fiber.StatusSeeOther)
}

func OlehOrderAdvance(c *fiber.Ctx) error {
	u, ok, err := tsPostUser(c, "/tanah-suci/oleh-oleh")
	if !ok {
		return err
	}
	if o := demo.ShopOrderFor(u.ID, c.Params("code")); o != nil {
		_ = demo.AdvanceShopOrder(o, tsWaTo(u))
		return c.Redirect("/tanah-suci/oleh-oleh/"+o.Code, fiber.StatusSeeOther)
	}
	return c.Redirect("/tanah-suci/oleh-oleh", fiber.StatusSeeOther)
}

// ---- voucher prabayar ----

type tsVoucherPreset struct {
	IDR, SAR int64
}

func voucherRender(c *fiber.Ctx, u *accounts.User, errMsg string) error {
	var presets []tsVoucherPreset
	for _, p := range demo.VoucherPresets {
		presets = append(presets, tsVoucherPreset{IDR: p, SAR: demo.SARFor(p)})
	}
	return c.Render("demo/tanahsuci_voucher", viewer(c, fiber.Map{"Title": "Voucher prabayar Tanah Suci",
		"Presets": presets, "Rate": demo.Cfg.SAR, "Min": demo.VoucherMinIDR, "Max": demo.VoucherMaxIDR,
		"Vouchers": demo.VouchersFor(u.ID), "Error": errMsg, "Holder": demo.VoucherHolder}))
}

func VoucherPage(c *fiber.Ctx) error {
	u, err := tsJamaah(c, "Voucher prabayar")
	if u == nil {
		return err
	}
	return voucherRender(c, u, "")
}

func VoucherBuy(c *fiber.Ctx) error {
	u, ok, err := tsPostUser(c, "/tanah-suci/voucher")
	if !ok {
		return err
	}
	amt := rupiahInput(c.FormValue("custom"))
	if amt == 0 {
		amt = rupiahInput(c.FormValue("amount"))
	}
	v, e := demo.BuyVoucher(u.ID, amt)
	if e != nil {
		return voucherRender(c, u, e.Error())
	}
	return c.Redirect("/tanah-suci/voucher/"+v.Code, fiber.StatusSeeOther)
}

func VoucherDetail(c *fiber.Ctx) error {
	u, err := tsJamaah(c, "Voucher prabayar")
	if u == nil {
		return err
	}
	v := demo.VoucherFor(u.ID, c.Params("code"))
	if v == nil {
		return c.Redirect("/tanah-suci/voucher", fiber.StatusSeeOther)
	}
	return c.Render("demo/tanahsuci_voucher_detail", viewer(c, fiber.Map{"Title": "Voucher " + v.Code, "V": v,
		"History": demo.Redemptions(v.ID), "Merchants": demo.Merchants, "Holder": demo.VoucherHolder}))
}

func VoucherPay(c *fiber.Ctx) error {
	u, ok, err := tsPostUser(c, "/tanah-suci/voucher")
	if !ok {
		return err
	}
	if v := demo.VoucherFor(u.ID, c.Params("code")); v != nil {
		_ = demo.PayVoucher(v, tsWaTo(u))
		return c.Redirect("/tanah-suci/voucher/"+v.Code, fiber.StatusSeeOther)
	}
	return c.Redirect("/tanah-suci/voucher", fiber.StatusSeeOther)
}

// ---- pickup di toko ----

func pickupRender(c *fiber.Ctx, u *accounts.User, errMsg string) error {
	qty := map[string]int{}
	for _, p := range demo.StoreCatalog {
		qty[p.Key] = atoiDef(c.FormValue("q_"+p.Key), 0)
	}
	store := c.FormValue("store", c.Query("toko"))
	return c.Render("demo/tanahsuci_pickup", viewer(c, fiber.Map{"Title": "Pickup di toko Makkah/Madinah",
		"Merchants": demo.Merchants, "Items": demo.StoreCatalog, "Slots": demo.PickupSlots(), "Qty": qty,
		"Store": store, "Slot": c.FormValue("slot"), "Pickups": demo.PickupsFor(u.ID), "Error": errMsg, "Rate": demo.Cfg.SAR}))
}

func PickupPage(c *fiber.Ctx) error {
	u, err := tsJamaah(c, "Pickup di toko")
	if u == nil {
		return err
	}
	return pickupRender(c, u, "")
}

func PickupCreate(c *fiber.Ctx) error {
	u, ok, err := tsPostUser(c, "/tanah-suci/pickup")
	if !ok {
		return err
	}
	qty := map[string]int{}
	for _, p := range demo.StoreCatalog {
		qty[p.Key] = atoiDef(c.FormValue("q_"+p.Key), 0)
	}
	p, e := demo.CreatePickup(u.ID, u.Name, c.FormValue("store"), c.FormValue("slot"), qty)
	if e != nil {
		return pickupRender(c, u, e.Error())
	}
	return c.Redirect("/tanah-suci/pickup/"+p.Code, fiber.StatusSeeOther)
}

func PickupDetail(c *fiber.Ctx) error {
	u, err := tsJamaah(c, "Pickup di toko")
	if u == nil {
		return err
	}
	p := demo.PickupFor(u.ID, c.Params("code"))
	if p == nil {
		return c.Redirect("/tanah-suci/pickup", fiber.StatusSeeOther)
	}
	return c.Render("demo/tanahsuci_pickup_detail", viewer(c, fiber.Map{"Title": "Pickup " + p.Code, "P": p, "M": p.Merchant(),
		"Rate": demo.Cfg.SAR}))
}

// ---- portal toko mitra (/toko) ----

type tsTokoResult struct {
	Voucher *demo.Voucher
	Red     *demo.VoucherRedemption
}

func tokoRender(c *fiber.Ctx, extra fiber.Map) error {
	r, ok := requireRole(c, "merchant")
	m := demo.MerchantBySlug(r.Slug)
	if !ok || m == nil {
		return c.Render("demo/toko", viewer(c, fiber.Map{"Title": "Portal toko mitra", "NeedRole": true, "Merchants": demo.Merchants}))
	}
	tab := "/toko"
	switch c.Path() {
	case "/toko/pickup", "/toko/riwayat":
		tab = c.Path()
	}
	pickups := demo.MerchantPickups(m.Slug)
	open := 0
	for _, p := range pickups {
		if p.Status != "diambil" {
			open++
		}
	}
	t := tabs(tab, "/toko", "Tukar voucher", "/toko/pickup", "Pickup", "/toko/riwayat", "Riwayat penukaran")
	t[1].Count = open
	todaySAR, todayN, allSAR, allN := demo.MerchantTotals(m.Slug)
	data := fiber.Map{"Title": m.Name, "M": m, "Tab": tab, "Tabs": t, "Pickups": pickups, "OpenPickups": open,
		"History": demo.MerchantRedemptions(m.Slug, 100), "TodaySAR": todaySAR, "TodayN": todayN, "AllSAR": allSAR, "AllN": allN,
		"Rate": demo.Cfg.SAR, "Max": demo.MaxRedeemSAR, "Samples": demo.ActiveVouchers(6), "Others": demo.Merchants}
	if k := c.Query("kode"); k != "" && tab == "/toko" {
		data["Kode"] = k
		if v := demo.VoucherByCode(k); v != nil {
			data["Preview"] = v
		} else {
			data["Error"] = demo.ErrVoucherNotFound.Error()
		}
	}
	for k, v := range extra {
		data[k] = v
	}
	return c.Render("demo/toko", viewer(c, data))
}

func TokoPage(c *fiber.Ctx) error { return tokoRender(c, nil) }

func TokoRedeem(c *fiber.Ctx) error {
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	r, ok := requireRole(c, "merchant")
	if !ok {
		return c.Redirect("/toko", fiber.StatusSeeOther)
	}
	code := c.FormValue("kode")
	amt := int64(atoiDef(c.FormValue("sar"), 0))
	v, red, err := demo.RedeemVoucher(code, r.Slug, amt, c.FormValue("note"))
	extra := fiber.Map{"Kode": code}
	if v != nil {
		extra["Preview"] = v
	}
	if err != nil {
		extra["Error"] = err.Error()
		extra["SAR"] = c.FormValue("sar")
		extra["Note"] = c.FormValue("note")
	} else {
		extra["Result"] = tsTokoResult{Voucher: v, Red: red}
	}
	return tokoRender(c, extra)
}

func TokoPickupAdvance(c *fiber.Ctx) error {
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	r, ok := requireRole(c, "merchant")
	if !ok {
		return c.Redirect("/toko", fiber.StatusSeeOther)
	}
	id, _ := c.ParamsInt("id")
	if id > 0 {
		_, _ = demo.AdvancePickup(r.Slug, uint(id), c.Params("to"))
	}
	return c.Redirect("/toko/pickup", fiber.StatusSeeOther)
}
