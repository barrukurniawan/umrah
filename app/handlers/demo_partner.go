package handlers

import (
	"umrah/app/accounts"
	"umrah/app/demo"
	"umrah/app/repositories"

	"github.com/gofiber/fiber/v2"
)

// Dasbor mitra pembiayaan (fiktif): menerima rujukan, memproses, menyetujui/menolak, mencairkan.

func registerDemoPartner(app *fiber.App) {
	app.Get("/mitra-pembiayaan", PartnerDashboard)
	app.Post("/mitra-pembiayaan/:id<int>", PartnerDecide)
}

func PartnerDashboard(c *fiber.Ctx) error {
	r, ok := requireRole(c, "pembiayaan")
	p := demo.PartnerBySlug(r.Slug)
	if !ok || p == nil {
		return demoRender(c, "demo/role_needed", viewer(c, fiber.Map{"Title": "Dasbor mitra pembiayaan", "Need": "pembiayaan", "Roles": demo.RoleOptions()}))
	}
	apps := demo.PartnerApplications(p.Slug)
	counts := map[string]int{}
	var fee int64
	for _, a := range apps {
		counts[a.Status]++
		fee += a.ReferralFee
	}
	return demoRender(c, "demo/partner", viewer(c, fiber.Map{"Title": p.Name, "P": p, "Apps": apps, "Counts": counts, "Fee": fee, "Err": c.Query("err"),
		"Badge": "Dasbor mitra pembiayaan (contoh)", "Sub": p.Note, "Tabs": tabs("/mitra-pembiayaan", "/mitra-pembiayaan", "Pengajuan masuk", "/inbox", "Notifikasi")}))
}

func PartnerDecide(c *fiber.Ctx) error {
	r, ok := requireRole(c, "pembiayaan")
	if !ok || !accounts.SameOrigin(c) {
		return c.Redirect("/mitra-pembiayaan", fiber.StatusSeeOther)
	}
	if err := demo.PartnerDecide(r.Slug, idParam(c, "id"), c.FormValue("status"), c.FormValue("catatan")); err != nil {
		return c.Redirect("/mitra-pembiayaan?err="+urlq(err.Error()), fiber.StatusSeeOther)
	}
	return c.Redirect("/mitra-pembiayaan", fiber.StatusSeeOther)
}

func repoTravelNames() []string {
	var out []string
	for _, p := range repositories.AllPackages() {
		out = append(out, p.Travel.Name)
	}
	return out
}
