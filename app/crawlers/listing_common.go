package crawlers

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Shared helpers for the listing parsers (umitravellist.go, umrahbisalist.go,
// rabbanilist.go). All identifiers are prefixed with "lst" to avoid clashes
// with helpers in other parser files.

const (
	lstUserAgent   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
	lstMinPrice    = 5000000
	lstMaxPrice    = 200000000
	lstPoliteDelay = 700 * time.Millisecond
)

// lstFetch downloads a page and returns its body as a string.
func lstFetch(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", lstUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// lstCutoff returns today's date (Asia/Jakarta, 00:00). Departures before it are dropped.
func lstCutoff() time.Time {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	now := time.Now().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// lstMonthAlt is a regexp alternation of Indonesian/English month names
// (longest first, so RE2 leftmost-first matching picks the full name).
const lstMonthAlt = `januari|februari|maret|april|mei|juni|juli|agustus|september|oktober|november|desember|` +
	`january|february|march|may|june|july|august|october|december|` +
	`sept|jan|feb|mar|apr|jun|jul|agu|agt|aug|sep|okt|oct|nov|des|dec`

// lstMonthNum maps a month token (any case) to its number, 0 if unknown.
func lstMonthNum(s string) int {
	s = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(s), "."))
	switch {
	case strings.HasPrefix(s, "jan"):
		return 1
	case strings.HasPrefix(s, "feb"):
		return 2
	case strings.HasPrefix(s, "mar"):
		return 3
	case strings.HasPrefix(s, "apr"):
		return 4
	case s == "mei" || s == "may":
		return 5
	case strings.HasPrefix(s, "jun"):
		return 6
	case strings.HasPrefix(s, "jul"):
		return 7
	case strings.HasPrefix(s, "agu") || strings.HasPrefix(s, "agt") || strings.HasPrefix(s, "aug"):
		return 8
	case strings.HasPrefix(s, "sep"):
		return 9
	case strings.HasPrefix(s, "okt") || strings.HasPrefix(s, "oct"):
		return 10
	case strings.HasPrefix(s, "nov"):
		return 11
	case strings.HasPrefix(s, "des") || strings.HasPrefix(s, "dec"):
		return 12
	}
	return 0
}

var lstSingleDateRe = regexp.MustCompile(`(?i)^\s*(\d{1,2})\s+(` + lstMonthAlt + `)\.?\s+(20\d{2})\s*$`)

// lstParseDate parses "15 Jan 2027", "15 Januari 2027", "13 agustus 2026", ...
func lstParseDate(s string) (time.Time, bool) {
	m := lstSingleDateRe.FindStringSubmatch(s)
	if m == nil {
		return time.Time{}, false
	}
	var d, y int
	fmt.Sscanf(m[1], "%d", &d)
	fmt.Sscanf(m[3], "%d", &y)
	return lstMakeDate(y, lstMonthNum(m[2]), d)
}

func lstMakeDate(y, mo, d int) (time.Time, bool) {
	if mo < 1 || mo > 12 || d < 1 || d > 31 || y < 2000 {
		return time.Time{}, false
	}
	t := time.Date(y, time.Month(mo), d, 0, 0, 0, 0, time.UTC)
	if t.Day() != d {
		return time.Time{}, false
	}
	return t, true
}

// lstFormatDate renders a date as "2 Jan 2006" (understood by the importer).
func lstFormatDate(t time.Time) string {
	return t.Format("2 Jan 2006")
}

// lstKeepPackage drops past departure dates (normalising the remaining ones to
// "2 Jan 2006") and reports whether the package is still worth emitting.
func lstKeepPackage(p *CrawledPackage, cutoff time.Time) bool {
	if p.Price < lstMinPrice || p.Price > lstMaxPrice {
		return false
	}
	var kept []string
	seen := map[string]bool{}
	for _, ds := range p.DepartureDates {
		t, ok := lstParseDate(ds)
		if !ok || t.Before(cutoff) {
			continue
		}
		f := lstFormatDate(t)
		if !seen[f] {
			seen[f] = true
			kept = append(kept, f)
		}
	}
	p.DepartureDates = kept
	return len(kept) > 0
}

var (
	lstScriptRe   = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	lstStyleRe    = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	lstNoscriptRe = regexp.MustCompile(`(?is)<noscript[^>]*>.*?</noscript>`)
	lstTagRe      = regexp.MustCompile(`<[^>]+>`)
	lstSpaceRe    = regexp.MustCompile(`[ \t\r\f\v\x{00a0}]+`)
)

// lstHTMLLines strips scripts/styles/tags and returns the non-empty text lines.
func lstHTMLLines(html string) []string {
	s := lstScriptRe.ReplaceAllString(html, "")
	s = lstStyleRe.ReplaceAllString(s, "")
	s = lstNoscriptRe.ReplaceAllString(s, "")
	s = lstTagRe.ReplaceAllString(s, "\n")
	s = lstUnescape(s)
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(lstSpaceRe.ReplaceAllString(l, " "))
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

var lstEntityReplacer = strings.NewReplacer(
	"&amp;", "&", "&#038;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`,
	"&#039;", "'", "&#39;", "'", "&apos;", "'", "&nbsp;", " ", "&#8211;", "–",
	"&#8212;", "—", "&ndash;", "–", "&mdash;", "—", "&#8217;", "’",
)

func lstUnescape(s string) string { return lstEntityReplacer.Replace(s) }

var lstOgTitleRe = regexp.MustCompile(`<meta[^>]+property=["']og:title["'][^>]+content=(?:"([^"]+)"|'([^']+)')`)

func lstOgTitle(html string) string {
	if m := lstOgTitleRe.FindStringSubmatch(html); m != nil {
		return strings.TrimSpace(lstUnescape(m[1] + m[2]))
	}
	return ""
}
