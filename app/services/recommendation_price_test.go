package services

import (
	"testing"

	"umrah/app/models"
)

func TestAvailablePrice(t *testing.T) {
	// harga paket 30jt berasal dari tanggal yang sudah dibuang (penuh); sisa: 33jt & 32jt
	got := availablePrice(30000000, []models.DetailPackage{{Price: 33000000}, {Price: 32000000}})
	if got != 32000000 {
		t.Errorf("got %d", got)
	}
	if got := availablePrice(30000000, []models.DetailPackage{{Price: 0}, {Price: 31000000}}); got != 30000000 {
		t.Errorf("Price 0 = harga paket; got %d", got)
	}
}
