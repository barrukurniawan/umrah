package analytics

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
)

var beaconTypes = map[string]bool{"click_wa": true, "click_web": true, "calc": true, "chat": true}

type beacon struct {
	Type     string `json:"t"`
	Travel   string `json:"travel"`
	Package  string `json:"pkg"`
	Price    int    `json:"price"`
	Position int    `json:"pos"`
}

// BeaconHandler menerima klik dari navigator.sendBeacon (POST /t). Selalu 204 supaya tidak bocor info.
func BeaconHandler(c *fiber.Ctx) error {
	var b beacon
	if len(c.Body()) > 2048 || json.Unmarshal(c.Body(), &b) != nil || !beaconTypes[b.Type] {
		return c.SendStatus(fiber.StatusNoContent)
	}
	visitor, device, bot := FromRequest(c)
	if bot {
		return c.SendStatus(fiber.StatusNoContent)
	}
	if b.Position < 0 || b.Position > 10000 {
		b.Position = 0
	}
	Record(Event{
		Type:     b.Type,
		Visitor:  visitor,
		Device:   device,
		Travel:   truncate(b.Travel, 80),
		Package:  truncate(b.Package, 200),
		Price:    b.Price,
		Position: b.Position,
	})
	return c.SendStatus(fiber.StatusNoContent)
}

type SearchMeta struct {
	Travels string `json:"travels,omitempty"`
	Month   string `json:"month,omitempty"`
	Sort    string `json:"sort,omitempty"`
	Total   int    `json:"total"`
}

type Impression struct {
	Travel   string
	Package  string
	Price    int
	Position int
}

// RecordSearch: halaman 1 dihitung sebagai pencarian; setiap kartu yang tampil dicatat sebagai "impr".
func RecordSearch(c *fiber.Ctx, budget, page int, meta SearchMeta, shown []Impression) {
	visitor, device, bot := FromRequest(c)
	if bot {
		return
	}
	if page <= 1 {
		m, _ := json.Marshal(meta)
		Record(Event{Type: "search", Visitor: visitor, Device: device, Budget: budget, Meta: truncate(string(m), 500)})
	}
	for _, s := range shown {
		Record(Event{Type: "impr", Visitor: visitor, Device: device, Travel: truncate(s.Travel, 80),
			Package: truncate(s.Package, 200), Price: s.Price, Position: s.Position, Budget: budget})
	}
}
