package demo

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"umrah/app/draws"
)

// Undian umroh berhadiah: setiap jamaah dalam booking yang DP-nya lunas di periode undian otomatis
// mendapat satu kode. Gratis, tanpa beli kupon, harga paket tidak dinaikkan, hadiah dari anggaran
// promosi umrohku.

func OpenDraw() *Draw {
	if DB == nil {
		return nil
	}
	var d Draw
	if DB.Where("status = ?", "terbuka").Order("ends_at").Take(&d).Error != nil {
		return nil
	}
	return &d
}

func Draws() []Draw {
	var ds []Draw
	DB.Order("CASE status WHEN 'terbuka' THEN 0 WHEN 'ditutup' THEN 1 WHEN 'draft' THEN 2 ELSE 3 END, starts_at DESC").Find(&ds)
	return ds
}

func PublicDraws() []Draw {
	var ds []Draw
	DB.Where("status <> ?", "draft").Order("CASE status WHEN 'terbuka' THEN 0 WHEN 'ditutup' THEN 1 ELSE 2 END, starts_at DESC").Find(&ds)
	return ds
}

func GetDraw(id uint) (*Draw, error) {
	var d Draw
	if err := DB.Take(&d, id).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func DrawEntries(drawID uint) []DrawEntry {
	var es []DrawEntry
	DB.Where("draw_id = ?", drawID).Order("code").Find(&es)
	return es
}

func (d Draw) Count() (active, void int) {
	var a, v int64
	DB.Model(&DrawEntry{}).Where("draw_id = ? AND void = ?", d.ID, false).Count(&a)
	DB.Model(&DrawEntry{}).Where("draw_id = ? AND void = ?", d.ID, true).Count(&v)
	return int(a), int(v)
}

func (d Draw) ActiveCount() int { a, _ := d.Count(); return a }

func (d Draw) WinnersList() []DrawEntry {
	var es []DrawEntry
	DB.Where("draw_id = ? AND winner > 0", d.ID).Order("winner").Find(&es)
	return es
}

func UserEntries(userID uint) []DrawEntry {
	var es []DrawEntry
	DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&es)
	return es
}

// CreateEntries: dipanggil saat DP lunas. Satu entri per jamaah; idempoten.
func CreateEntries(b *Booking) []string {
	if b.PaidAt == nil {
		return nil
	}
	var ds []Draw
	DB.Where("status = ? AND starts_at <= ? AND ends_at >= ?", "terbuka", *b.PaidAt, *b.PaidAt).Find(&ds)
	var codes []string
	for _, d := range ds {
		for i, name := range b.PilgrimNames() {
			e := DrawEntry{DrawID: d.ID, BookingID: b.ID, Pilgrim: i + 1, UserID: b.UserID, Name: name, City: b.City, CreatedAt: time.Now()}
			var existing DrawEntry
			if DB.Where("draw_id = ? AND booking_id = ? AND pilgrim = ?", d.ID, b.ID, i+1).Take(&existing).Error == nil {
				codes = append(codes, existing.Code)
				continue
			}
			for try := 0; try < 5; try++ {
				e.Code = "UMR-" + randCode(4) + "-" + randCode(2)
				if DB.Create(&e).Error == nil {
					codes = append(codes, e.Code)
					break
				}
			}
		}
	}
	return codes
}

// VoidEntries: booking batal sebelum pengundian -> entri tidak ikut.
func VoidEntries(bookingID uint) {
	DB.Exec(`UPDATE draw_entries SET void = 1 WHERE booking_id = ? AND draw_id IN (SELECT id FROM draws WHERE status IN ('terbuka','ditutup'))`, bookingID)
}

func entriesFor(drawID uint) []draws.Entry {
	var out []draws.Entry
	for _, e := range DrawEntries(drawID) {
		out = append(out, draws.Entry{Code: e.Code, Void: e.Void})
	}
	return out
}

func SetDrawStatus(id uint, status string) error {
	d, err := GetDraw(id)
	if err != nil {
		return err
	}
	now := time.Now()
	switch {
	case status == "terbuka" && d.Status == "draft":
	case status == "ditutup" && d.Status == "terbuka":
		d.EntriesHash = draws.EntriesHash(entriesFor(d.ID))
		d.ClosedAt = &now
	default:
		return errors.New("perubahan status tidak valid")
	}
	d.Status = status
	return DB.Save(d).Error
}

// RunDraw mengundi dengan seed saksi. Hanya untuk undian berstatus "ditutup".
func RunDraw(id uint, witnessSeed, witness string) error {
	d, err := GetDraw(id)
	if err != nil {
		return err
	}
	if d.Status != "ditutup" {
		return errors.New("tutup undian dulu supaya hash daftar entri terbit sebelum mengundi")
	}
	witnessSeed = strings.TrimSpace(witnessSeed)
	if len(witnessSeed) < 3 || len(witnessSeed) > 100 {
		return errors.New("seed saksi 3–100 karakter (mis. angka yang dibacakan saksi)")
	}
	entries := entriesFor(d.ID)
	if draws.EntriesHash(entries) != d.EntriesHash {
		return errors.New("daftar entri berubah sejak undian ditutup; pengundian dihentikan")
	}
	seed := draws.Seed(d.EntriesHash, witnessSeed)
	winners := draws.Pick(entries, seed, d.Winners)
	now := time.Now()
	for i, code := range winners {
		DB.Model(&DrawEntry{}).Where("draw_id = ? AND code = ?", d.ID, code).Update("winner", i+1)
		var e DrawEntry
		if DB.Where("draw_id = ? AND code = ?", d.ID, code).Take(&e).Error == nil {
			Notify(UserAudience(e.UserID), "wa", e.Name, "Selamat! Anda pemenang undian",
				fmt.Sprintf("Kode %s terpilih sebagai pemenang ke-%d %s. Hadiah: %s. Tim umrohku akan menghubungi Anda untuk verifikasi.", code, i+1, d.Title, d.PrizeDesc), fmt.Sprintf("/undian/%d", d.ID))
		}
	}
	d.WitnessSeed, d.Witness, d.Seed, d.Status, d.DrawnAt = witnessSeed, truncate(witness, 160), seed, "diundi", &now
	return DB.Save(d).Error
}

func SaveDraw(d *Draw) error {
	if d.Status == "" {
		d.Status = "draft"
	}
	if d.Winners < 1 || d.Winners > 100 {
		return errors.New("jumlah pemenang 1–100")
	}
	if !d.EndsAt.After(d.StartsAt) || d.DrawAt.Before(d.EndsAt) {
		return errors.New("periode tidak valid (selesai setelah mulai, pengundian setelah selesai)")
	}
	return DB.Save(d).Error
}

// WriteMinutesCSV: berita acara (entri, hash, seed, pemenang).
func WriteMinutesCSV(w io.Writer, d *Draw) {
	cw := csv.NewWriter(w)
	cw.Write([]string{"Berita acara", d.Title})
	cw.Write([]string{"Hadiah", d.PrizeDesc})
	cw.Write([]string{"Hash daftar entri (sha256)", d.EntriesHash})
	cw.Write([]string{"Seed saksi", d.WitnessSeed})
	cw.Write([]string{"Saksi", d.Witness})
	cw.Write([]string{"Seed akhir = sha256(hash + seed saksi)", d.Seed})
	cw.Write([]string{})
	cw.Write([]string{"kode", "status", "skor sha256(seed+kode)", "pemenang ke"})
	for _, e := range DrawEntries(d.ID) {
		st, score, win := "aktif", "", ""
		if e.Void {
			st = "batal"
		} else if d.Seed != "" {
			score = draws.Score(d.Seed, e.Code)
		}
		if e.Winner > 0 {
			win = fmt.Sprint(e.Winner)
		}
		cw.Write([]string{e.Code, st, score, win})
	}
	cw.Flush()
}
