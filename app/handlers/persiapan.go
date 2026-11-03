package handlers

import (
	"encoding/json"
	"html/template"
	"strings"
	"time"

	"umrah/app/accounts"
	"umrah/app/services"

	"github.com/gofiber/fiber/v2"
)

// GET /persiapan: checklist Persiapan Umrah. Tanpa login tetap bisa dicentang (disimpan di
// browser); setelah login progres disimpan di akun dan digabung dengan yang ada di browser.
func PrepPage(c *fiber.Ctx) error {
	m := fiber.Map{"Title": "Persiapan Umrah", "Sections": services.PrepSections()}
	if u := accounts.Current(c); u != nil {
		st := accounts.GetPrep(u.ID)
		b, _ := json.Marshal(fiber.Map{"checks": st.CheckList(), "depart": st.DepartDate, "gender": st.Gender})
		m["State"] = template.JS(b) // JSON siap pakai di <script type="application/json">
		m["DepartHint"] = departHint(u.ID)
	}
	return c.Render("persiapan", viewer(c, m))
}

type prepReq struct {
	Checks []string `json:"checks"`
	Depart string   `json:"depart"`
	Gender string   `json:"gender"`
}

// POST /api/persiapan: simpan seluruh progres (bukan per item, supaya sederhana & idempoten).
func PrepSave(c *fiber.Ctx) error {
	u := accounts.Current(c)
	if u == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "login"})
	}
	var req prepReq
	if !accounts.SameOrigin(c) || len(c.Body()) > 8000 || json.Unmarshal(c.Body(), &req) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "bad request"})
	}
	valid, seen := services.PrepKeys(), map[string]bool{}
	var keys []string
	for _, k := range req.Checks {
		if valid[k] && !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	st := accounts.PrepState{UserID: u.ID, Checks: strings.Join(keys, ",")}
	if _, err := time.Parse("2006-01-02", req.Depart); err == nil {
		st.DepartDate = req.Depart
	}
	if req.Gender == "m" || req.Gender == "f" {
		st.Gender = req.Gender
	}
	if err := accounts.SavePrep(st); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "gagal menyimpan"})
	}
	return c.JSON(fiber.Map{"ok": true})
}

// departHint: tanggal berangkat terdekat dari paket tersimpan / target tabungan (saran isian).
func departHint(userID uint) string {
	today := time.Now().Format("2006-01-02")
	best := ""
	for _, g := range accounts.ListGoals(userID) {
		if d := g.TargetDate; d != "" && d >= today && (best == "" || d < best) {
			best = d
		}
	}
	return best
}
