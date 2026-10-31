package accounts

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	SessionCookie = "uk_sess"
	stateCookie   = "uk_oauth"
)

// Hanya identitas dasar: nama, email, foto. Tidak ada akses Drive/Gmail/kontak.
func oauthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

func Enabled() bool { return DB != nil && os.Getenv("GOOGLE_CLIENT_ID") != "" && os.Getenv("GOOGLE_CLIENT_SECRET") != "" }

func secure(c *fiber.Ctx) bool { return c.Protocol() == "https" || c.Get("X-Forwarded-Proto") == "https" }

// safeNext: hanya path lokal ("/..."), cegah redirect ke situs lain.
func safeNext(next string) string {
	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.HasPrefix(next, "/\\") {
		return "/"
	}
	return next
}

func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Middleware memasang pengguna yang sedang login ke c.Locals("user") (nil bila tamu).
// Yang belum menyetujui kebijakan privasi disimpan sebagai "pending" dan diperlakukan seperti tamu.
func Middleware(c *fiber.Ctx) error {
	if tok := c.Cookies(SessionCookie); tok != "" {
		if u := UserBySession(tok); u != nil {
			if u.ConsentAt == nil {
				c.Locals("pending_user", u)
			} else {
				c.Locals("user", u)
			}
		}
	}
	return c.Next()
}

func Current(c *fiber.Ctx) *User {
	u, _ := c.Locals("user").(*User)
	return u
}

func Pending(c *fiber.Ctx) *User {
	u, _ := c.Locals("pending_user").(*User)
	return u
}

// SameOrigin: tolak POST lintas situs (CSRF) untuk aksi yang mengubah data akun.
func SameOrigin(c *fiber.Ctx) bool {
	origin := c.Get("Origin")
	if origin == "" {
		origin = c.Get("Referer")
	}
	host := c.Get("X-Forwarded-Host")
	if host == "" {
		host = c.Hostname()
	}
	return origin != "" && (strings.HasPrefix(origin, "https://"+host) || strings.HasPrefix(origin, "http://"+host))
}

func GoogleLogin(c *fiber.Ctx) error {
	if !Enabled() {
		return c.Status(fiber.StatusServiceUnavailable).SendString("Login belum tersedia.")
	}
	state, verifier := randHex(16), oauth2.GenerateVerifier()
	// state|verifier|next disimpan di cookie singkat; dicocokkan saat Google mengembalikan pengguna.
	c.Cookie(&fiber.Cookie{
		Name: stateCookie, Value: state + "|" + verifier + "|" + safeNext(c.Query("next", "/")),
		Path: "/auth", MaxAge: 600, HTTPOnly: true, Secure: secure(c), SameSite: "Lax",
	})
	url := oauthConfig().AuthCodeURL(state, oauth2.S256ChallengeOption(verifier), oauth2.SetAuthURLParam("prompt", "select_account"))
	return c.Redirect(url, fiber.StatusFound)
}

type googleProfile struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

func GoogleCallback(c *fiber.Ctx) error {
	parts := strings.SplitN(c.Cookies(stateCookie), "|", 3)
	c.Cookie(&fiber.Cookie{Name: stateCookie, Value: "", Path: "/auth", MaxAge: -1, HTTPOnly: true, Secure: secure(c), SameSite: "Lax"})
	if !Enabled() || len(parts) != 3 || c.Query("state") == "" || c.Query("state") != parts[0] {
		return c.Redirect("/?login=gagal", fiber.StatusFound)
	}
	if c.Query("error") != "" || c.Query("code") == "" {
		return c.Redirect(safeNext(parts[2]), fiber.StatusFound) // pengguna membatalkan di layar Google
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cfg := oauthConfig()
	tok, err := cfg.Exchange(ctx, c.Query("code"), oauth2.VerifierOption(parts[1]))
	if err != nil {
		log.Println("[akun] tukar kode gagal:", err)
		return c.Redirect("/?login=gagal", fiber.StatusFound)
	}
	prof, err := fetchProfile(ctx, cfg.Client(ctx, tok))
	if err != nil || prof.Sub == "" || !prof.EmailVerified {
		log.Println("[akun] profil Google gagal:", err)
		return c.Redirect("/?login=gagal", fiber.StatusFound)
	}
	u, err := UpsertGoogleUser(prof.Sub, prof.Email, prof.Name, prof.Picture)
	if err != nil {
		log.Println("[akun] simpan pengguna gagal:", err)
		return c.Redirect("/?login=gagal", fiber.StatusFound)
	}
	sess, exp, err := NewSession(u.ID)
	if err != nil {
		return c.Redirect("/?login=gagal", fiber.StatusFound)
	}
	c.Cookie(&fiber.Cookie{Name: SessionCookie, Value: sess, Path: "/", Expires: exp, HTTPOnly: true, Secure: secure(c), SameSite: "Lax"})

	next := safeNext(parts[2])
	if u.ConsentAt == nil {
		return c.Redirect("/akun/persetujuan?next="+next, fiber.StatusFound)
	}
	return c.Redirect(next, fiber.StatusFound)
}

func fetchProfile(ctx context.Context, client *http.Client) (googleProfile, error) {
	var p googleProfile
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://openidconnect.googleapis.com/v1/userinfo", nil)
	resp, err := client.Do(req)
	if err != nil {
		return p, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return p, fmt.Errorf("userinfo status %d", resp.StatusCode)
	}
	return p, json.NewDecoder(resp.Body).Decode(&p)
}

func Logout(c *fiber.Ctx) error {
	if SameOrigin(c) {
		DeleteSession(c.Cookies(SessionCookie))
		c.Cookie(&fiber.Cookie{Name: SessionCookie, Value: "", Path: "/", MaxAge: -1, HTTPOnly: true, Secure: secure(c), SameSite: "Lax"})
	}
	return c.Redirect("/", fiber.StatusSeeOther)
}
