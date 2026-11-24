package handlers

import (
	"encoding/json"
	"html/template"
	"time"
	"strings"

	"umrah/app/accounts"
	"umrah/app/demo"
	"umrah/app/payments"
	"umrah/app/qris"

	"github.com/gofiber/fiber/v2"
	qrcode "github.com/skip2/go-qrcode"
)

// Market Tanah Suci ala GoFood (demo): toko terdekat di Madinah/Makkah, pesan untuk diambil,
// bayar QRIS dinamis, petunjuk arah Google Maps. Lihat app/demo/market.go & PLAN_MARKET_TANAH_SUCI.md.

const marketCityCookie = "uk_kota"

func registerDemoMarket(app *fiber.App) {
	// Di subdomain demo, beranda = market (ala GrabFood, ambil di toko). Pencarian paket pindah ke /paket.
	app.Get("/", MarketHome)
	app.Get("/paket", HomePage)
	app.Get("/market", func(c *fiber.Ctx) error {
		q := string(c.Request().URI().QueryString())
		if q != "" {
			q = "?" + q
		}
		return c.Redirect("/"+q, fiber.StatusFound)
	})
	app.Get("/market/pesanan/:code/qr.png", MarketOrderQRPNG)
	app.Get("/market/toko/:slug", MarketStorePage)
	app.Post("/market/toko/:slug/checkout", MarketCheckoutPage)
	app.Post("/market/pesan", MarketPlaceOrder)
	app.Get("/market/pesanan", MarketOrdersPage)
	app.Get("/market/pesanan/:code", MarketOrderPage)
	app.Get("/market/pesanan/:code/status", MarketOrderStatus)
	app.Post("/market/pesanan/:code/simulasi", MarketOrderSimulatePay)
	app.Post("/market/pesanan/:code/qr-baru", MarketOrderRenew)
	app.Post("/market/pesanan/:code/siap", MarketOrderDemoReady)

	app.Get("/toko/pesanan", TokoPage)
	app.Post("/toko/pesanan/:code/:to", TokoMarketAdvance)
}

// marketCity: ?kota= (disimpan di cookie) > cookie > Madinah.
func marketCity(c *fiber.Ctx) demo.City {
	if k := c.Query("kota"); k == "madinah" || k == "makkah" {
		c.Cookie(&fiber.Cookie{Name: marketCityCookie, Value: k, Path: "/", MaxAge: 30 * 24 * 3600, SameSite: "Lax"})
		return demo.CityByKey(k)
	}
	return demo.CityByKey(c.Cookies(marketCityCookie))
}

func MarketHome(c *fiber.Ctx) error {
	city := marketCity(c)
	cat := c.Query("kat")
	if demo.MarketCatByKey(cat) == nil {
		cat = ""
	}
	q := strings.TrimSpace(c.Query("q"))
	if len(q) > 60 {
		q = q[:60]
	}
	stores := demo.MarketStores(city, cat, q)
	near := stores
	if len(near) > 5 {
		near = near[:5]
	}
	type cj struct {
		Key  string  `json:"key"`
		Name string  `json:"name"`
		Lat  float64 `json:"lat"`
		Lng  float64 `json:"lng"`
	}
	var cs []cj
	for _, ct := range demo.Cities {
		cs = append(cs, cj{ct.Key, ct.Name, ct.Lat, ct.Lng})
	}
	b, _ := json.Marshal(cs)
	m := fiber.Map{"Title": "Market " + city.Name, "City": city, "Cities": demo.Cities, "Cats": demo.MarketCats, "Cat": cat, "Q": q,
		"Stores": stores, "Near": near, "Picks": demo.MarketPicks(city, cat, q, 6), "Rate": demo.SARRateDemo, "CitiesJSON": template.JS(b)}
	if mc := demo.MarketCatByKey(cat); mc != nil {
		m["CatLabel"] = mc.Label
	}
	if u := accounts.Current(c); u != nil {
		m["Active"] = demo.ActiveMarketOrders(u.ID)
	}
	return demoRender(c, "demo/market", viewer(c, m))
}

func marketStore(c *fiber.Ctx, slug string) (*demo.StoreView, demo.City) {
	mer := demo.MerchantBySlug(slug)
	if mer == nil || len(demo.StoreItems[slug]) == 0 {
		return nil, demo.City{}
	}
	city := demo.CityByKey(mer.CityKey())
	d := demo.DistanceM(city.Lat, city.Lng, mer.Lat, mer.Lng)
	return &demo.StoreView{Merchant: mer, Dist: d, DistL: demo.DistLabel(d)}, city
}

func MarketStorePage(c *fiber.Ctx) error {
	s, city := marketStore(c, c.Params("slug"))
	if s == nil {
		return c.Redirect("/market")
	}
	return demoRender(c, "demo/market_toko", viewer(c, fiber.Map{"Title": s.Name, "S": s, "City": city, "Menu": demo.StoreMenu(s.Slug),
		"Rate": demo.SARRateDemo, "HideChat": true}))
}

func marketQty(c *fiber.Ctx, slug string) map[string]int {
	qty := map[string]int{}
	for _, i := range demo.StoreItems[slug] {
		if n := atoiDef(c.FormValue("q_"+i.Key), 0); n > 0 {
			qty[i.Key] = n
		}
	}
	return qty
}

func marketCheckoutRender(c *fiber.Ctx, u *accounts.User, slug string, errMsg string) error {
	s, city := marketStore(c, slug)
	if s == nil {
		return c.Redirect("/market", fiber.StatusSeeOther)
	}
	lines, sar := demo.MarketLines(slug, marketQty(c, slug))
	if len(lines) == 0 {
		return c.Redirect("/market/toko/"+slug, fiber.StatusSeeOther)
	}
	name := c.FormValue("nama")
	if name == "" {
		name = u.Name
	}
	slot := c.FormValue("slot")
	if slot == "" {
		slot = "segera"
	}
	return demoRender(c, "demo/market_checkout", viewer(c, fiber.Map{"Title": "Cek pesanan", "S": s, "City": city, "Lines": lines,
		"TotalSAR": sar, "TotalIDR": sar * demo.SARRateDemo, "Rate": demo.SARRateDemo, "Slots": demo.MarketSlots, "Slot": slot,
		"Name": name, "Note": c.FormValue("catatan"), "Err": errMsg, "HideChat": true}))
}

// POST /market/toko/:slug/checkout: dari keranjang di halaman toko ke halaman cek pesanan.
func MarketCheckoutPage(c *fiber.Ctx) error {
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	u := accounts.Current(c)
	if u == nil {
		return c.Redirect("/auth/google/login?next="+urlq("/market/toko/"+c.Params("slug")), fiber.StatusSeeOther)
	}
	return marketCheckoutRender(c, u, c.Params("slug"), "")
}

func MarketPlaceOrder(c *fiber.Ctx) error {
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	u, err := jamaahOrLogin(c)
	if u == nil {
		return err
	}
	slug := c.FormValue("toko")
	o, oerr := demo.CreateMarketOrder(u.ID, slug, c.FormValue("nama"), c.FormValue("slot"), c.FormValue("catatan"), marketQty(c, slug))
	if oerr != nil {
		return marketCheckoutRender(c, u, slug, oerr.Error())
	}
	return c.Redirect("/market/pesanan/"+o.Code, fiber.StatusSeeOther)
}

func MarketOrdersPage(c *fiber.Ctx) error {
	u, err := jamaahOrLogin(c)
	if u == nil {
		return err
	}
	return demoRender(c, "demo/market_list", viewer(c, fiber.Map{"Title": "Pesanan market", "Orders": demo.MarketOrdersFor(u.ID, 50)}))
}

func marketOrderData(o *demo.MarketOrder, fresh bool) fiber.Map {
	m := o.Merchant()
	city := demo.CityByKey(m.CityKey())
	d := fiber.Map{"O": o, "M": m, "City": city, "Fresh": fresh, "Dist": demo.DistLabel(demo.DistanceM(city.Lat, city.Lng, m.Lat, m.Lng)),
		"NMID": ""}
	if q, err := qris.Decode(o.QRPayload); err == nil {
		d["NMID"] = q.NMID()
	}
	if o.Waiting() {
		d["QRSVG"] = template.HTML(payments.QRSVG(o.QRPayload, 256))
	} else if o.Paid() && o.Status != "diambil" {
		d["PickupSVG"] = template.HTML(payments.QRSVG("UMROHKU-AMBIL:"+o.Code+":"+o.PickupCode, 160))
	}
	return d
}

func marketOrderOf(c *fiber.Ctx) (*accounts.User, *demo.MarketOrder, error) {
	u, err := jamaahOrLogin(c)
	if u == nil {
		return nil, nil, err
	}
	o := demo.MarketOrderFor(u.ID, c.Params("code"))
	if o == nil {
		return u, nil, c.Redirect("/market/pesanan", fiber.StatusSeeOther)
	}
	return u, o, nil
}

func MarketOrderPage(c *fiber.Ctx) error {
	_, o, err := marketOrderOf(c)
	if o == nil {
		return err
	}
	d := marketOrderData(o, c.Query("baru") == "1")
	d["Title"] = "Pesanan " + o.Code
	d["HideChat"] = true
	return demoRender(c, "demo/market_pesanan", viewer(c, d))
}

// GET .../status: potongan htmx. Animasi centang hanya saat status baru saja berubah.
func MarketOrderStatus(c *fiber.Ctx) error {
	_, o, err := marketOrderOf(c)
	if o == nil {
		return err
	}
	fresh := c.Get("HX-Request") == "true" && o.Paid() && o.Status == "dibayar"
	c.Set("Cache-Control", "no-store")
	return c.Render("demo/market_status", marketOrderData(o, fresh))
}

// GET .../qr.png: unduh QR tagihan untuk dibuka dari galeri di aplikasi pembayaran lain.
func MarketOrderQRPNG(c *fiber.Ctx) error {
	_, o, err := marketOrderOf(c)
	if o == nil {
		return err
	}
	if !o.Waiting() || time.Now().After(o.ExpiresAt) {
		return c.Redirect("/market/pesanan/"+o.Code, fiber.StatusSeeOther)
	}
	png, perr := qrcode.Encode(o.QRPayload, qrcode.Medium, 640)
	if perr != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	c.Set("Content-Type", "image/png")
	c.Set("Cache-Control", "no-store")
	c.Set("Content-Disposition", `attachment; filename="QRIS-`+o.Code+`.png"`)
	return c.Send(png)
}

var marketSimApps = map[string]bool{"DANA": true, "ShopeePay": true, "GoPay": true, "OVO": true}

func MarketOrderSimulatePay(c *fiber.Ctx) error {
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	_, o, err := marketOrderOf(c)
	if o == nil {
		return err
	}
	app := c.FormValue("app")
	if !marketSimApps[app] {
		app = "aplikasi lain"
	}
	_ = demo.PayMarketOrder(o, app+" (simulasi)", "")
	return c.Redirect("/market/pesanan/"+o.Code+"?baru=1", fiber.StatusSeeOther)
}

func MarketOrderRenew(c *fiber.Ctx) error {
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	_, o, err := marketOrderOf(c)
	if o == nil {
		return err
	}
	_ = demo.RenewMarketQR(o)
	return c.Redirect("/market/pesanan/"+o.Code, fiber.StatusSeeOther)
}

// POST .../siap: jalan pintas demo (satu orang memeragakan sisi toko).
func MarketOrderDemoReady(c *fiber.Ctx) error {
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	_, o, err := marketOrderOf(c)
	if o == nil {
		return err
	}
	_, _ = demo.AdvanceMarketOrder(o.MerchantSlug, o.Code, "siap", "")
	return c.Redirect("/market/pesanan/"+o.Code, fiber.StatusSeeOther)
}

// POST /toko/pesanan/:code/:to (siap | diambil + kode).
func TokoMarketAdvance(c *fiber.Ctx) error {
	if !accounts.SameOrigin(c) {
		return c.SendStatus(fiber.StatusForbidden)
	}
	r, ok := requireRole(c, "merchant")
	if !ok {
		return c.Redirect("/toko", fiber.StatusSeeOther)
	}
	_, err := demo.AdvanceMarketOrder(r.Slug, c.Params("code"), c.Params("to"), c.FormValue("kode"))
	if err != nil {
		return c.Redirect("/toko/pesanan?err="+urlq(err.Error())+"&kode="+urlq(c.Params("code")), fiber.StatusSeeOther)
	}
	return c.Redirect("/toko/pesanan?ok="+urlq(c.Params("code")), fiber.StatusSeeOther)
}
