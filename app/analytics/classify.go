package analytics

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Sumber kunjungan: utm_source dulu, lalu referrer (pola sama dengan admin MarryMe).
var utmSources = []struct {
	re   *regexp.Regexp
	name string
}{
	{regexp.MustCompile(`(?i)^(ig|instagram)$`), "Instagram"},
	{regexp.MustCompile(`(?i)^(fb|facebook)$`), "Facebook"},
	{regexp.MustCompile(`(?i)^threads$`), "Threads"},
	{regexp.MustCompile(`(?i)^(wa|whatsapp)$`), "WhatsApp"},
	{regexp.MustCompile(`(?i)^tiktok$`), "TikTok"},
	{regexp.MustCompile(`(?i)^google$`), "Google"},
	{regexp.MustCompile(`(?i)^(x|twitter)$`), "X"},
}

var referrerHosts = []struct {
	re   *regexp.Regexp
	name string
}{
	{regexp.MustCompile(`(^|\.)instagram\.com$`), "Instagram"},
	{regexp.MustCompile(`(^|\.)(facebook\.com|fb\.com|fb\.me)$`), "Facebook"},
	{regexp.MustCompile(`(^|\.)threads\.(net|com)$`), "Threads"},
	{regexp.MustCompile(`(^|\.)(whatsapp\.com|wa\.me)$`), "WhatsApp"},
	{regexp.MustCompile(`(^|\.)tiktok\.com$`), "TikTok"},
	{regexp.MustCompile(`(^|\.)google\.[a-z.]+$`), "Google"},
	{regexp.MustCompile(`(^|\.)(bing\.com|duckduckgo\.com|search\.yahoo\.com)$`), "Mesin cari lain"},
	{regexp.MustCompile(`(^|\.)(t\.co|twitter\.com|x\.com)$`), "X"},
	{regexp.MustCompile(`(^|\.)umrohku\.web\.id$|^localhost$`), "Internal"},
}

func ClassifySource(referrer, utm string, hasFbclid bool) string {
	if utm = strings.TrimSpace(utm); utm != "" {
		for _, s := range utmSources {
			if s.re.MatchString(utm) {
				return s.name
			}
		}
		return "Lainnya"
	}
	if hasFbclid {
		return "Facebook"
	}
	if referrer == "" {
		return "Langsung"
	}
	u, err := url.Parse(referrer)
	if err != nil || u.Hostname() == "" {
		return "Lainnya"
	}
	host := strings.ToLower(u.Hostname())
	for _, s := range referrerHosts {
		if s.re.MatchString(host) {
			return s.name
		}
	}
	return "Lainnya"
}

var botRe = regexp.MustCompile(`(?i)bot|crawl|spider|slurp|facebookexternalhit|whatsapp|telegram|preview|headless|lighthouse|curl|wget|python|go-http|httpclient|uptime|monitor|scan`)
var mobileRe = regexp.MustCompile(`(?i)mobile|android|iphone|ipod|opera mini|iemobile`)
var tabletRe = regexp.MustCompile(`(?i)ipad|tablet`)

func IsBot(ua string) bool { return ua == "" || botRe.MatchString(ua) }

func Device(ua string) string {
	switch {
	case tabletRe.MatchString(ua):
		return "Tablet"
	case mobileRe.MatchString(ua):
		return "HP"
	default:
		return "Desktop"
	}
}

// ClientIP: nginx (container docker) mengirim X-Real-IP; percayai hanya bila koneksi dari jaringan privat.
func ClientIP(c *fiber.Ctx) string {
	ip := c.IP()
	if real := strings.TrimSpace(c.Get("X-Real-IP")); real != "" {
		if p := net.ParseIP(ip); p != nil && (p.IsLoopback() || p.IsPrivate()) {
			return real
		}
	}
	return ip
}

// Middleware mencatat "view" untuk GET halaman (bukan aset, admin, atau beacon).
func ViewMiddleware(c *fiber.Ctx) error {
	path := c.Path()
	page := c.Method() == fiber.MethodGet && !strings.HasPrefix(path, "/static") && !strings.HasPrefix(path, "/admin") &&
		path != "/t" && path != "/favicon.ico"
	ua := c.Get(fiber.HeaderUserAgent)
	if !page || IsBot(ua) {
		return c.Next()
	}
	vid := ensureVID(c)
	err := c.Next()
	if c.Response().StatusCode() != fiber.StatusOK {
		return err
	}
	ip := ClientIP(c)
	city, region, country := Locate(ip)
	Record(Event{
		Type:    "view",
		Visitor: VisitorID(ip, ua),
		VID:     vid,
		Source:  ClassifySource(c.Get(fiber.HeaderReferer), c.Query("utm_source"), c.Query("fbclid") != ""),
		Device:  Device(ua),
		Path:    truncate(path, 200),
		City:    city,
		Region:  region,
		Country: country,
	})
	return err
}

const vidCookie = "uk_vid"

var vidRe = regexp.MustCompile(`^[0-9a-f]{32}$`)

// ensureVID memberi ID anonim acak (cookie 1 tahun) untuk menghitung kunjungan ulang; tanpa data pribadi.
func ensureVID(c *fiber.Ctx) string {
	if v := c.Cookies(vidCookie); vidRe.MatchString(v) {
		return v
	}
	buf := make([]byte, 16)
	rand.Read(buf)
	v := hex.EncodeToString(buf)
	c.Cookie(&fiber.Cookie{
		Name: vidCookie, Value: v, Path: "/", MaxAge: 365 * 24 * 3600, HTTPOnly: true, SameSite: "Lax",
		Secure: c.Protocol() == "https" || c.Get("X-Forwarded-Proto") == "https",
	})
	return v
}

// Visitor harian, ID anonim cookie, dan perangkat untuk event selain view (pencarian, klik).
func FromRequest(c *fiber.Ctx) (visitor, vid, device string, bot bool) {
	ua := c.Get(fiber.HeaderUserAgent)
	if v := c.Cookies(vidCookie); vidRe.MatchString(v) {
		vid = v
	}
	return VisitorID(ClientIP(c), ua), vid, Device(ua), IsBot(ua)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
