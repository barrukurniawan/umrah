// Package payments: antarmuka gateway pembayaran. Demo memakai MockGateway (tidak ada uang
// sungguhan). Nanti cukup menambah implementasi Xendit/Midtrans/bank tanpa mengubah alur booking.
//
// Prinsip: setiap tagihan dibuat ATAS NAMA rekening penampungan travel (Holder), bukan umrohku.
package payments

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type Request struct {
	Ref     string // kode unik tagihan, mis. "UMK-7K2Q-DP"
	Amount  int64
	Bank    string // bank rekening penampungan travel
	Account string // nomor rekening penampungan travel (untuk prefix VA)
	Holder  string // "PT ... - Rek. Penampungan Dana Jamaah"
	Payer   string
}

type Invoice struct {
	Method    string // va | qris
	Bank      string
	VANumber  string
	Holder    string
	QRPayload string
	ExpiresAt time.Time
}

type Gateway interface {
	Name() string
	CreateVA(r Request) (Invoice, error)
	CreateQRIS(r Request) (Invoice, error)
}

// MockGateway: nomor VA & payload QRIS deterministik dari Ref. Pembayaran "berhasil" hanya lewat
// tombol simulasi di demo.
type MockGateway struct{}

func (MockGateway) Name() string { return "Simulasi (demo)" }

func digits(s string, n int) string {
	h := sha256.Sum256([]byte(s))
	var b strings.Builder
	for _, x := range h {
		b.WriteByte('0' + x%10)
		if b.Len() == n {
			break
		}
	}
	return b.String()
}

func (MockGateway) CreateVA(r Request) (Invoice, error) {
	if r.Amount <= 0 {
		return Invoice{}, fmt.Errorf("nominal tidak valid")
	}
	prefix := "8808"
	if len(r.Account) >= 4 {
		prefix = r.Account[len(r.Account)-4:]
	}
	return Invoice{Method: "va", Bank: r.Bank, VANumber: prefix + digits(r.Ref, 12), Holder: r.Holder,
		ExpiresAt: time.Now().Add(24 * time.Hour)}, nil
}

func (MockGateway) CreateQRIS(r Request) (Invoice, error) {
	if r.Amount <= 0 {
		return Invoice{}, fmt.Errorf("nominal tidak valid")
	}
	payload := fmt.Sprintf("DEMO-QRIS|%s|%d|%s", r.Ref, r.Amount, r.Holder)
	return Invoice{Method: "qris", Bank: r.Bank, Holder: r.Holder, QRPayload: payload, ExpiresAt: time.Now().Add(30 * time.Minute)}, nil
}

var Default Gateway = MockGateway{}

// FakeQRSVG menggambar pola mirip kode QR dari payload (TIDAK bisa dipindai; hanya ilustrasi demo).
func FakeQRSVG(payload string, size int) string {
	const n = 25
	h := sha256.Sum256([]byte(payload))
	seed := hex.EncodeToString(h[:])
	bit := func(i int) bool {
		hh := sha256.Sum256([]byte(fmt.Sprintf("%s|%d", seed, i/256)))
		return hh[(i%256)/8]>>(uint(i)%8)&1 == 1
	}
	finder := func(x, y int) (bool, bool) { // (bagian finder?, hitam?)
		for _, o := range [][2]int{{0, 0}, {n - 7, 0}, {0, n - 7}} {
			dx, dy := x-o[0], y-o[1]
			if dx >= -1 && dx <= 7 && dy >= -1 && dy <= 7 {
				if dx < 0 || dy < 0 || dx > 6 || dy > 6 {
					return true, false
				}
				ring := dx == 0 || dy == 0 || dx == 6 || dy == 6
				core := dx >= 2 && dx <= 4 && dy >= 2 && dy <= 4
				return true, ring || core
			}
		}
		return false, false
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="-2 -2 %d %d" width="%d" height="%d" shape-rendering="crispEdges" role="img" aria-label="QRIS contoh (tidak bisa dipindai)"><rect x="-2" y="-2" width="%d" height="%d" fill="#fff"/><path fill="#111" d="`, n+4, n+4, size, size, n+4, n+4)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			isF, black := finder(x, y)
			if !isF {
				black = bit(y*n + x)
			}
			if black {
				fmt.Fprintf(&b, "M%d %dh1v1h-1z", x, y)
			}
		}
	}
	b.WriteString(`"/></svg>`)
	return b.String()
}
