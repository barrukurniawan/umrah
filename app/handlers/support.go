package handlers

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"umrah/app/accounts"
	"umrah/app/analytics"

	"github.com/gofiber/fiber/v2"
)

// Chat bantuan (kritik/saran/pertanyaan) pengguna <-> admin. Wajib login Google.
// Tanpa websocket: widget melakukan polling.

type supportMsgJSON struct {
	ID     uint   `json:"id"`
	Sender string `json:"sender"`
	Body   string `json:"body"`
	At     string `json:"at"`
}

func supportJSON(ms []accounts.SupportMessage) []supportMsgJSON {
	out := make([]supportMsgJSON, 0, len(ms))
	for _, m := range ms {
		out = append(out, supportMsgJSON{ID: m.ID, Sender: m.Sender, Body: m.Body, At: m.CreatedAt.In(analytics.WIB).Format("02 Jan 15:04")})
	}
	return out
}

func afterParam(c *fiber.Ctx) uint {
	n, _ := strconv.ParseUint(c.Query("after"), 10, 64)
	return uint(n)
}

func cleanBody(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
	if r := []rune(s); len(r) > accounts.SupportMaxLen {
		s = string(r[:accounts.SupportMaxLen])
	}
	return s
}

// GET /api/support/messages?after=<id>&open=1
// open=1: panel sedang terbuka, tandai balasan admin sudah dibaca.
func SupportMessages(c *fiber.Ctx) error {
	u := accounts.Current(c)
	if u == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "login"})
	}
	if c.Query("open") == "1" {
		accounts.SupportMarkRead(u.ID, "admin")
	}
	c.Set("Cache-Control", "no-store")
	return c.JSON(fiber.Map{
		"messages": supportJSON(accounts.SupportMessages(u.ID, afterParam(c))),
		"unread":   accounts.SupportUnread(u.ID, "admin"),
	})
}

// POST /api/support/messages {"body": "...", "context": "/tersimpan"}
func SupportSend(c *fiber.Ctx) error {
	u := accounts.Current(c)
	if u == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "login"})
	}
	var req struct {
		Body    string `json:"body"`
		Context string `json:"context"`
	}
	if !accounts.SameOrigin(c) || json.Unmarshal(c.Body(), &req) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "permintaan tidak valid"})
	}
	body := cleanBody(req.Body)
	if body == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "pesan kosong"})
	}
	if !accounts.SupportAllow(u.ID) {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "Terlalu banyak pesan. Tunggu sebentar ya."})
	}
	ctx := safeNext(req.Context)
	if len(ctx) > 255 {
		ctx = ctx[:255]
	}
	m, err := accounts.SupportSend(u.ID, "user", body, ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "gagal mengirim"})
	}
	return c.JSON(fiber.Map{"message": supportJSON([]accounts.SupportMessage{m})[0]})
}

// --- Admin (path /admin/..., cookie admin) ---

func supportAdminUser(c *fiber.Ctx) (*accounts.User, error) {
	if !isAdmin(c) {
		return nil, c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "admin"})
	}
	id, _ := strconv.ParseUint(c.Params("uid"), 10, 64)
	u := accounts.UserByID(uint(id))
	if u == nil {
		return nil, c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "pengguna tidak ditemukan"})
	}
	return u, nil
}

// GET /admin/api/support/:uid/messages?after=<id>
func AdminSupportMessages(c *fiber.Ctx) error {
	u, err := supportAdminUser(c)
	if u == nil {
		return err
	}
	accounts.SupportMarkRead(u.ID, "user")
	c.Set("Cache-Control", "no-store")
	return c.JSON(fiber.Map{"messages": supportJSON(accounts.SupportMessages(u.ID, afterParam(c)))})
}

// POST /admin/api/support/:uid/messages {"body": "..."}
func AdminSupportReply(c *fiber.Ctx) error {
	u, err := supportAdminUser(c)
	if u == nil {
		return err
	}
	var req struct {
		Body string `json:"body"`
	}
	if !accounts.SameOrigin(c) || json.Unmarshal(c.Body(), &req) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "permintaan tidak valid"})
	}
	body := cleanBody(req.Body)
	if body == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "pesan kosong"})
	}
	m, e := accounts.SupportSend(u.ID, "admin", body, "")
	if e != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "gagal mengirim"})
	}
	return c.JSON(fiber.Map{"message": supportJSON([]accounts.SupportMessage{m})[0]})
}

// GET /admin/api/support/threads: daftar percakapan (untuk refresh daftar di tab Pesan).
func AdminSupportThreads(c *fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "admin"})
	}
	type row struct {
		UserID  uint   `json:"user_id"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture string `json:"picture"`
		Last    string `json:"last"`
		From    string `json:"from"`
		At      string `json:"at"`
		Unread  int64  `json:"unread"`
	}
	var out []row
	for _, t := range accounts.SupportThreads() {
		out = append(out, row{t.UserID, t.Name, t.Email, t.Picture, t.LastBody, t.LastFrom, supportTime(t.LastAt), t.Unread})
	}
	c.Set("Cache-Control", "no-store")
	return c.JSON(fiber.Map{"threads": out, "unread": accounts.SupportUnreadForAdmin()})
}

func supportTime(t time.Time) string {
	t = t.In(analytics.WIB)
	if y, m, d := time.Now().In(analytics.WIB).Date(); t.Year() == y && t.Month() == m && t.Day() == d {
		return t.Format("15:04")
	}
	return t.Format("02 Jan")
}
