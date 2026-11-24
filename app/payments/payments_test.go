package payments

import (
	"strings"
	"testing"

	"umrah/app/qris"
)

func TestDynamicQRISValid(t *testing.T) {
	p := DynamicQRIS("OLH-7K2Q9", 1250000)
	q, err := qris.Decode(p)
	if err != nil {
		t.Fatal(err)
	}
	if !q.Dynamic || q.Amount != 1250000 || !q.IsQRIS() || !q.IsIDR() || q.MerchantName != DemoMerchantName || q.NMID() != DemoNMID || q.BillNumber != "OLH-7K2Q9" {
		t.Fatalf("payload salah: %+v", q)
	}
	if svg := QRSVG(p, 200); !strings.Contains(svg, `aria-label="QRIS dinamis"`) {
		t.Fatal("QR tidak tergambar")
	}
}

func TestMockGatewayQRIS(t *testing.T) {
	inv, err := MockGateway{}.CreateQRIS(Request{Ref: "UMK-1-DP", Amount: 500000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := qris.Decode(inv.QRPayload); err != nil {
		t.Fatalf("payload checkout tidak valid: %v", err)
	}
}
