package handlers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"umrah/app/accounts"
	"umrah/app/services"

	"github.com/gofiber/fiber/v2"
)

// Data umum untuk header (tombol Masuk / menu avatar) di semua halaman publik.
func viewer(c *fiber.Ctx, m fiber.Map) fiber.Map {
	m["User"] = accounts.Current(c)
	m["LoginEnabled"] = accounts.Enabled()
	m["Path"] = c.OriginalURL()
	return m
}

func safeNext(next string) string {
	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.HasPrefix(next, "/\\") {
		return "/"
	}
	return next
}

// --- Persetujuan saat login pertama ---

func ConsentPage(c *fiber.Ctx) error {
	u := accounts.Pending(c)
	if u == nil {
		if accounts.Current(c) != nil {
			return c.Redirect(safeNext(c.Query("next", "/")))
		}
		return c.Redirect("/")
	}
	return c.Render("account_consent", fiber.Map{"Pending": u, "Next": safeNext(c.Query("next", "/"))})
}

func ConsentAccept(c *fiber.Ctx) error {
	u := accounts.Pending(c)
	if u == nil || !accounts.SameOrigin(c) {
		return c.Redirect("/", fiber.StatusSeeOther)
	}
	accounts.SetConsent(u.ID)
	return c.Redirect(safeNext(c.FormValue("next", "/")), fiber.StatusSeeOther)
}

// ConsentDecline: menolak = akun yang baru dibuat langsung dihapus.
func ConsentDecline(c *fiber.Ctx) error {
	u := accounts.Pending(c)
	if u != nil && accounts.SameOrigin(c) {
		accounts.DeleteUser(u.ID)
	}
	c.ClearCookie(accounts.SessionCookie)
	return c.Redirect("/", fiber.StatusSeeOther)
}

// --- ♡ Simpan paket ---

type savedRow struct {
	accounts.SavedPackage
	Current   *services.CurrentPackage
	PriceDown int // selisih bila harga sekarang lebih murah dari saat disimpan
}

func SavedPage(c *fiber.Ctx) error {
	u := accounts.Current(c)
	m := fiber.Map{}
	if u != nil {
		saved := accounts.ListSaved(u.ID)
		keys := map[string]bool{}
		for _, s := range saved {
			keys[s.Key] = true
		}
		cur := services.CurrentByKeys(keys)
		rows := make([]savedRow, 0, len(saved))
		for _, s := range saved {
			r := savedRow{SavedPackage: s, Current: cur[s.Key]}
			if r.Current != nil && r.Current.MinPrice < s.Price {
				r.PriceDown = s.Price - r.Current.MinPrice
			}
			rows = append(rows, r)
		}
		m["Saved"] = rows
	}
	return c.Render("saved", viewer(c, m))
}

type keyReq struct {
	Key  string   `json:"key"`
	Keys []string `json:"keys"`
}

func snapshot(userID uint, cp *services.CurrentPackage) accounts.SavedPackage {
	return accounts.SavedPackage{
		UserID: userID, Key: cp.Key, Travel: cp.Travel, Package: cp.Name, Price: cp.MinPrice, Duration: cp.Duration,
		Airline: cp.Airline, HotelMakkah: cp.HotelMakkah, HotelMadinah: cp.HotelMadinah, Departures: cp.DatesLabel,
	}
}

// SavedToggle: POST /api/saved/toggle {key}. Data paket diambil dari server, bukan dari browser.
func SavedToggle(c *fiber.Ctx) error {
	u := accounts.Current(c)
	if u == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "login"})
	}
	var req keyReq
	if !accounts.SameOrigin(c) || json.Unmarshal(c.Body(), &req) != nil || req.Key == "" || len(req.Key) > 300 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "bad request"})
	}
	if accounts.SavedKeys(u.ID)[req.Key] {
		accounts.Unsave(u.ID, req.Key)
		return c.JSON(fiber.Map{"saved": false})
	}
	cp := services.CurrentByKeys(map[string]bool{req.Key: true})[req.Key]
	if cp == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "paket tidak ditemukan"})
	}
	if accounts.CountSaved(u.ID) >= 100 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "maksimal 100 paket tersimpan"})
	}
	if err := accounts.Save(snapshot(u.ID, cp)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "gagal menyimpan"})
	}
	return c.JSON(fiber.Map{"saved": true})
}

// SavedImport: paket yang di-♡ sebelum login (disimpan di browser) dipindah ke akun.
func SavedImport(c *fiber.Ctx) error {
	u := accounts.Current(c)
	var req keyReq
	if u == nil || !accounts.SameOrigin(c) || json.Unmarshal(c.Body(), &req) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "bad request"})
	}
	if len(req.Keys) > 50 {
		req.Keys = req.Keys[:50]
	}
	keys := map[string]bool{}
	for _, k := range req.Keys {
		if k != "" && len(k) <= 300 {
			keys[k] = true
		}
	}
	n := 0
	for _, cp := range services.CurrentByKeys(keys) {
		if accounts.CountSaved(u.ID) >= 100 {
			break
		}
		if accounts.Save(snapshot(u.ID, cp)) == nil {
			n++
		}
	}
	return c.JSON(fiber.Map{"imported": n})
}

// --- Halaman akun: unduh data & hapus akun ---

func AccountPage(c *fiber.Ctx) error {
	u := accounts.Current(c)
	if u == nil {
		return c.Redirect("/auth/google/login?next=/akun")
	}
	return c.Render("account", viewer(c, fiber.Map{"SavedCount": accounts.CountSaved(u.ID), "Deleted": false}))
}

func AccountExport(c *fiber.Ctx) error {
	u := accounts.Current(c)
	if u == nil {
		return c.Redirect("/auth/google/login?next=/akun")
	}
	data := fiber.Map{
		"exported_at": time.Now().Format(time.RFC3339),
		"akun":        fiber.Map{"nama": u.Name, "email": u.Email, "dibuat": u.CreatedAt, "login_terakhir": u.LastLoginAt, "persetujuan": u.ConsentAt},
		"tersimpan":   accounts.ListSaved(u.ID),
		"tabungan":    exportSavings(u.ID),
	}
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="umrohku-data-%s.json"`, time.Now().Format("20060102")))
	return c.JSON(data)
}

func AccountDelete(c *fiber.Ctx) error {
	u := accounts.Current(c)
	if u == nil || !accounts.SameOrigin(c) {
		return c.Redirect("/", fiber.StatusSeeOther)
	}
	if strings.TrimSpace(strings.ToUpper(c.FormValue("confirm"))) != "HAPUS" {
		return c.Render("account", viewer(c, fiber.Map{"SavedCount": accounts.CountSaved(u.ID), "Error": `Ketik HAPUS untuk konfirmasi.`}))
	}
	accounts.DeleteUser(u.ID)
	c.ClearCookie(accounts.SessionCookie)
	return c.Redirect("/?akun=dihapus", fiber.StatusSeeOther)
}

func exportSavings(userID uint) []fiber.Map {
	var out []fiber.Map
	for _, g := range accounts.ListGoals(userID) {
		out = append(out, fiber.Map{"tujuan": g, "setoran": accounts.ListEntries(g.ID)})
	}
	return out
}
