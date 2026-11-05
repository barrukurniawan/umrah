package services

import (
	"net/url"
	"strings"
)

// WASourceNote ditambahkan ke pesan WhatsApp agar travel tahu calon jamaah datang dari Umrohku.
const WASourceNote = "(Saya dapat info ini dari umrohku.web.id)"

// WATagUmrohku menambahkan WASourceNote ke parameter text tautan WhatsApp.
// Tautan yang sudah menyebut "umrohku.web.id" atau bukan URL valid dikembalikan apa adanya.
func WATagUmrohku(link string) string {
	u, err := url.Parse(link)
	if err != nil || link == "#" {
		return link
	}
	q := u.Query()
	text := q.Get("text")
	if strings.Contains(strings.ToLower(text), "umrohku.web.id") {
		return link
	}
	if text == "" {
		text = WASourceNote
	} else {
		text = strings.TrimRight(text, " \n") + "\n\n" + WASourceNote
	}
	q.Set("text", text)
	u.RawQuery = q.Encode()
	return u.String()
}
