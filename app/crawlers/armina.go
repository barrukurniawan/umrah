package crawlers

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ArminaParser membaca tabel jadwal resmi PT Arminareka Perdana
// (https://jamaah.arminarekaperdana.com/site/jadwal, halaman publik, 15 baris per halaman).
//
// Satu baris = satu keberangkatan: "Mina GA CGK 9D MED (17/03/2027)" | 17 Mar 2027 |
// Quad | Triple | Double | Quota | Sisa. Baris dengan nama program sama (tanpa tanggal)
// digabung menjadi satu paket dengan beberapa tanggal.
type ArminaParser struct {
	URL        string
	TravelName string
}

var (
	armRowRe   = regexp.MustCompile(`(?s)<tr[^>]*data-key="\d+"[^>]*>(.*?)</tr>`)
	armCellRe  = regexp.MustCompile(`(?s)<td[^>]*>(.*?)</td>`)
	armTagRe   = regexp.MustCompile(`<[^>]+>`)
	armLabelRe = regexp.MustCompile(`^(.*?)\s+([A-Z0-9]{2})\s+([A-Z]{3})\s+(\d{1,2})D\s+([A-Z]{3})\b`)
	armParenRe = regexp.MustCompile(`\s*\(\d{1,2}/\d{1,2}/\d{4}\)\s*$`)
)

// Kode bandara asal -> nama kota (yang dipakai situs-situs travel).
var airportCity = map[string]string{
	"CGK": "Jakarta", "SUB": "Surabaya", "KNO": "Medan", "UPG": "Makassar", "BPN": "Balikpapan",
	"DPS": "Bali", "SOC": "Solo", "YIA": "Yogyakarta", "JOG": "Yogyakarta", "PDG": "Padang",
	"PKU": "Pekanbaru", "BTH": "Batam", "PLM": "Palembang", "BDJ": "Banjarmasin", "LOP": "Lombok",
	"KJT": "Kertajati", "SRG": "Semarang", "BTJ": "Banda Aceh", "MDC": "Manado", "PNK": "Pontianak",
	"DJB": "Jambi", "TKG": "Lampung", "AMQ": "Ambon", "BKS": "Bengkulu",
}

func (a *ArminaParser) Crawl() ([]CrawledPackage, error) {
	base := strings.TrimRight(a.URL, "/")
	if base == "" {
		base = "https://jamaah.arminarekaperdana.com/site/jadwal"
	}
	travel := a.TravelName
	if travel == "" {
		travel = "Armina"
	}
	cutoff := lstCutoff()
	byProg := map[string]*CrawledPackage{}
	var order []string

	for page := 1; page <= 30; page++ {
		html, err := lstFetchPolite(fmt.Sprintf("%s?page=%d", base, page))
		if err != nil {
			if page == 1 {
				return nil, err
			}
			break
		}
		rows := armRowRe.FindAllStringSubmatch(html, -1)
		if len(rows) == 0 {
			break
		}
		for _, r := range rows {
			cells := armCellRe.FindAllStringSubmatch(r[1], -1)
			if len(cells) < 8 {
				continue
			}
			txt := func(i int) string { return strings.TrimSpace(lstUnescape(armTagRe.ReplaceAllString(cells[i][1], ""))) }
			label := armParenRe.ReplaceAllString(txt(1), "")
			date, ok := lstParseDate(txt(2))
			if !ok {
				continue
			}
			quad, triple, double := plainMoney(txt(3)), plainMoney(txt(4)), plainMoney(txt(5))
			quota, _ := strconv.Atoi(txt(6))
			left, _ := strconv.Atoi(txt(7))

			m := armLabelRe.FindStringSubmatch(label)
			name, airline, airport, landing, dur := label, "", "", "", 0
			if m != nil {
				prog := strings.TrimSpace(m[1])
				airline = lstAirlineName(m[2])
				airport = airportCity[m[3]]
				dur, _ = strconv.Atoi(m[4])
				switch m[5] {
				case "MED":
					landing = "landing Madinah"
				case "JED":
					landing = "landing Jeddah"
				}
				name = fmt.Sprintf("%s %d Hari", lstTitle(prog), dur)
				if airport != "" {
					name += " Start " + airport
				}
				if landing != "" {
					name += " (" + landing + ")"
				}
			}
			p, ok := byProg[name]
			if !ok {
				p = &CrawledPackage{TravelName: travel, PackageName: name, Duration: dur, Airline: airline, Airport: airport, URL: base}
				byProg[name] = p
				order = append(order, name)
			}
			d := CrawledDeparture{Date: lstISODate(date), Price: quad, SeatsTotal: quota}
			if triple > quad {
				d.PriceTriple = triple
			}
			if double > quad {
				d.PriceDouble = double
			}
			if dur > 0 {
				d.ReturnDate = lstISODate(date.AddDate(0, 0, dur-1))
			}
			if quota > 0 && left <= 0 {
				d.Full = true
			} else {
				d.Seats = left
			}
			p.Departures = append(p.Departures, d)
		}
		if !strings.Contains(html, fmt.Sprintf("page=%d&", page+1)) && !strings.Contains(html, fmt.Sprintf("page=%d\"", page+1)) {
			break
		}
	}

	var out []CrawledPackage
	sort.Strings(order)
	for _, k := range order {
		p := byProg[k]
		// Harga triple/double paket = termurah antar tanggal (per tanggal tetap disimpan).
		for _, d := range p.Departures {
			if d.PriceTriple > 0 && (p.PriceTriple == 0 || d.PriceTriple < p.PriceTriple) {
				p.PriceTriple = d.PriceTriple
			}
			if d.PriceDouble > 0 && (p.PriceDouble == 0 || d.PriceDouble < p.PriceDouble) {
				p.PriceDouble = d.PriceDouble
			}
		}
		if lstFinalize(p, cutoff) {
			out = append(out, *p)
		}
	}
	return out, nil
}

// lstTitle: "UMROH LAILATUL QODAR" -> "Umroh Lailatul Qodar"; teks campuran dibiarkan.
func lstTitle(s string) string {
	if s != strings.ToUpper(s) {
		return s
	}
	w := strings.Fields(strings.ToLower(s))
	for i, x := range w {
		if len(x) > 0 {
			w[i] = strings.ToUpper(x[:1]) + x[1:]
		}
	}
	return strings.Join(w, " ")
}

func lstISODate(t time.Time) string { return t.Format("2006-01-02") }

// plainMoney: "41.300.000" / "41,300,000" / "IDR 33.900.000,-" -> 41300000 (0 bila < 100 ribu).
func plainMoney(s string) int {
	if i := strings.Index(s, ",-"); i >= 0 {
		s = s[:i]
	}
	v := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			v = v*10 + int(c-'0')
		}
	}
	if v < 100000 {
		return 0
	}
	return v
}
