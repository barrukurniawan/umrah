#!/usr/bin/env python3
"""raw/haram-raw.json (+ extra.json) -> web/static/sim/data/haram.json untuk open world Masjidil Haram.

Pakai:  python3 scripts/haram-osm/build.py          (tanpa jaringan; ambil ulang data dengan fetch.py)
        python3 scripts/haram-osm/build.py --check  (juga menggambar cek.png di folder ini)

Koordinat keluaran = meter dunia simulasi: pusat Ka'bah (0,0), x = timur, z = selatan (sama dengan tawaf.js:
-z = utara). Jarak dikalikan K, kecuali area Ka'bah (radius LOCAL_R meter asli: Ka'bah, Hijr, Maqam, sudut-sudut)
yang tetap ukuran asli. Tinggi bangunan (`h`) juga sudah dikali K.
Data © kontributor OpenStreetMap, lisensi ODbL.
"""
import json, math, pathlib, re, sys

HERE = pathlib.Path(__file__).resolve().parent
ROOT = HERE.parent.parent
OUT = ROOT / "web/static/sim/data/haram.json"

K = 0.5            # skala dunia (lihat PLAN_OPEN_WORLD.md K3)
LOCAL_R = 22.0     # meter asli di sekitar Ka'bah yang tidak dikecilkan
SIMPLIFY = 0.6     # toleransi Douglas-Peucker (meter asli)
GATE_SNAP = 3.0    # jarak maksimum node pintu ke garis masjid (meter asli)
BUILD_R = 650.0    # bangunan sekitar yang diambil (meter asli dari Ka'bah)

MOSQUE_REL = 1472531
KAABA, HIJR, MAQAM = 103914569, 315911894, 473301379
SAI_WAY, TAWAF_WAY, RAML_WAY = 584203412, 584197021, 742725721
BLACK_STONE, CORNER_SHAMI, CORNER_YAMANI, CORNER_IRAQI = 1199442721, 1199442699, 1199442700, 8794911100
SAFA, MARWAH = 4589923995, 4589923996
CLOCK_REL = 12896421  # relasi "The Clock Towers" (kompleks Abraj Al Bait)


def load():
    raw = json.loads((HERE / "raw/haram-raw.json").read_text())
    els = raw["elements"]
    return {(e["type"], e["id"]): e for e in els}, els, raw.get("osm3s", {}).get("timestamp_osm_base", "")


by, ELS, STAMP = load()
kg = by[("way", KAABA)]["geometry"][:-1]
LAT0 = sum(p["lat"] for p in kg) / len(kg)
LON0 = sum(p["lon"] for p in kg) / len(kg)
MX = 111320.0 * math.cos(math.radians(LAT0))
MY = 110574.0


def meters(p):
    """lat/lon -> meter asli relatif pusat Ka'bah (x timur, z selatan)."""
    return ((p["lon"] - LON0) * MX, -(p["lat"] - LAT0) * MY)


def world(x, z, local=False):
    """meter asli -> meter dunia. Dalam LOCAL_R tidak dikecilkan; di luar, jarak radial dipetakan mulus."""
    if local:
        return (x, z)
    r = math.hypot(x, z)
    if r <= LOCAL_R:
        return (x, z)
    s = (LOCAL_R + (r - LOCAL_R) * K) / r
    return (x * s, z * s)


def rnd(v, n=1):
    return round(v, n)


def pt(p, local=False):
    x, z = world(*p, local=local)
    return [rnd(x), rnd(z)]


def poly_out(pts, local=False):
    return [pt(p, local) for p in pts]


def dp(pts, tol):
    """Douglas-Peucker untuk polyline terbuka."""
    if len(pts) < 3:
        return pts
    (x1, z1), (x2, z2) = pts[0], pts[-1]
    dx, dz = x2 - x1, z2 - z1
    L = math.hypot(dx, dz) or 1e-9
    best, idx = -1, 0
    for i in range(1, len(pts) - 1):
        x, z = pts[i]
        d = abs(dz * (x - x1) - dx * (z - z1)) / L if L > 1e-6 else math.hypot(x - x1, z - z1)
        if d > best:
            best, idx = d, i
    if best <= tol:
        return [pts[0], pts[-1]]
    return dp(pts[: idx + 1], tol)[:-1] + dp(pts[idx:], tol)


def simplify_ring(ring, tol=SIMPLIFY):
    """ring tertutup tanpa titik ulang di akhir."""
    if len(ring) < 5:
        return ring
    # pecah di titik terjauh dari titik 0 supaya DP stabil
    far = max(range(len(ring)), key=lambda i: (ring[i][0] - ring[0][0]) ** 2 + (ring[i][1] - ring[0][1]) ** 2)
    a = dp(ring[: far + 1], tol)
    b = dp(ring[far:] + [ring[0]], tol)
    return a[:-1] + b[:-1]


def area(ring):
    s = 0.0
    for i in range(len(ring)):
        x1, z1 = ring[i]
        x2, z2 = ring[(i + 1) % len(ring)]
        s += x1 * z2 - x2 * z1
    return s / 2


def centroid(ring):
    a = area(ring)
    if abs(a) < 1e-9:
        return (sum(p[0] for p in ring) / len(ring), sum(p[1] for p in ring) / len(ring))
    cx = cz = 0.0
    for i in range(len(ring)):
        x1, z1 = ring[i]
        x2, z2 = ring[(i + 1) % len(ring)]
        f = x1 * z2 - x2 * z1
        cx += (x1 + x2) * f
        cz += (z1 + z2) * f
    return (cx / (6 * a), cz / (6 * a))


def inside(p, ring):
    x, z = p
    c = False
    for i in range(len(ring)):
        x1, z1 = ring[i]
        x2, z2 = ring[i - 1]
        if (z1 > z) != (z2 > z) and x < (x2 - x1) * (z - z1) / (z2 - z1) + x1:
            c = not c
    return c


def seg_cross(a, b, c, d):
    def o(p, q, r):
        return (q[0] - p[0]) * (r[1] - p[1]) - (q[1] - p[1]) * (r[0] - p[0])
    return (o(a, b, c) * o(a, b, d) < 0) and (o(c, d, a) * o(c, d, b) < 0)


def self_crossings(ring):
    n, bad = len(ring), 0
    for i in range(n):
        a, b = ring[i], ring[(i + 1) % n]
        for j in range(i + 2, n):
            if i == 0 and j == n - 1:
                continue
            if seg_cross(a, b, ring[j], ring[(j + 1) % n]):
                bad += 1
    return bad


def assemble(ways):
    """gabungkan potongan way (list titik meter) menjadi ring tertutup."""
    segs = [list(w) for w in ways if len(w) >= 2]
    rings = []
    key = lambda p: (round(p[0], 2), round(p[1], 2))
    while segs:
        cur = segs.pop(0)
        changed = True
        while key(cur[0]) != key(cur[-1]) and changed:
            changed = False
            for i, s in enumerate(segs):
                if key(s[0]) == key(cur[-1]):
                    cur += s[1:]
                elif key(s[-1]) == key(cur[-1]):
                    cur += s[::-1][1:]
                elif key(s[-1]) == key(cur[0]):
                    cur = s[:-1] + cur
                elif key(s[0]) == key(cur[0]):
                    cur = s[::-1][:-1] + cur
                else:
                    continue
                segs.pop(i)
                changed = True
                break
        if key(cur[0]) != key(cur[-1]):
            print("  peringatan: ring tidak tertutup,", len(cur), "titik", file=sys.stderr)
            continue
        rings.append(cur[:-1])
    return rings


def way_pts(e):
    return [meters(p) for p in e["geometry"]]


def rel_rings(rel, role):
    return assemble([[meters(p) for p in m["geometry"]] for m in rel.get("members", []) if m["type"] == "way" and m.get("role") == role and m.get("geometry")])


def orient(ring, ccw=True):
    """ccw di bidang x-z (z ke selatan) = area positif."""
    return ring if (area(ring) > 0) == ccw else ring[::-1]


def parse_h(t):
    h = t.get("height") or ""
    m = re.match(r"\s*([\d.]+)", h)
    if m:
        return float(m.group(1))
    lv = t.get("building:levels")
    if lv and re.match(r"^\d+", lv):
        return float(re.match(r"^\d+", lv).group()) * 3.5
    return None


def levels_of(t):
    lv = t.get("level")
    if lv is None:
        return None
    return [s.strip() for s in re.split(r"[;,]", lv)]


def main():
    check = "--check" in sys.argv
    out = {
        "k": K, "localR": LOCAL_R, "origin": [rnd(LAT0, 7), rnd(LON0, 7)],
        "attribution": "© kontributor OpenStreetMap (ODbL)", "osm": STAMP,
    }
    report = []

    # ---------- Masjid ----------
    rel = by[("relation", MOSQUE_REL)]
    outers = [orient(simplify_ring(r)) for r in rel_rings(rel, "outer")]
    inners = [orient(simplify_ring(r), False) for r in rel_rings(rel, "inner")]
    outers.sort(key=lambda r: -abs(area(r)))
    for r in outers + inners:
        nb = self_crossings(r)
        if nb:
            report.append(f"PERINGATAN ring {len(r)} titik punya {nb} silangan")
    out["mosque"] = {
        "outer": [poly_out(r) for r in outers],
        "inner": [poly_out(r) for r in inners],
    }
    report.append(f"masjid: {len(outers)} ring luar ({', '.join(f'{len(r)} titik/{abs(area(r))/1e4:.1f} ha' for r in outers)}), "
                  f"{len(inners)} lubang ({', '.join(f'{len(r)} titik' for r in inners)})")

    # ---------- Ka'bah & sekitarnya (ukuran asli) ----------
    corners = {"hajarAswad": BLACK_STONE, "syami": CORNER_SHAMI, "yamani": CORNER_YAMANI, "iraqi": CORNER_IRAQI}
    k = orient([meters(p) for p in by[("way", KAABA)]["geometry"][:-1]])
    cw = {n: meters(by[("node", i)]) for n, i in corners.items() if ("node", i) in by}
    # Rotasi y (konvensi three.js: x' = x cos + z sin, z' = -x sin + z cos) yang paling pas memetakan sudut kotak
    # lokal tawaf.js (KW=10 di x, KD=12 di z; Hajar Aswad = (+x,+z), Yamani = (-x,+z), Syami = (-x,-z), Iraqi = (+x,-z))
    # ke sudut-sudut OSM.
    local = {"hajarAswad": (5, 6), "yamani": (-5, 6), "syami": (-5, -6), "iraqi": (5, -6)}
    def err(th):
        c, s_ = math.cos(th), math.sin(th)
        return sum((x * c + z * s_ - cw[n][0]) ** 2 + (-x * s_ + z * c - cw[n][1]) ** 2 for n, (x, z) in local.items() if n in cw)
    rot = min((i * math.pi / 1800 for i in range(3600)), key=err)
    c, s_ = math.cos(rot), math.sin(rot)
    # ukuran: proyeksi poligon ke sumbu lokal (x lokal = (c, -s), z lokal = (s, c))
    lx = [p[0] * c - p[1] * s_ for p in k]; lz = [p[0] * s_ + p[1] * c for p in k]
    size = [rnd(max(lx) - min(lx), 2), rnd(max(lz) - min(lz), 2)]
    out["kaaba"] = {"poly": poly_out(k, True), "rot": rnd(rot, 4), "size": size, "fit": rnd(math.sqrt(err(rot) / 4), 2),
                    "h": parse_h(by[("way", KAABA)]["tags"])}
    report.append(f"Ka'bah: rot {math.degrees(rot):.1f}° (tawaf.js 45°), ukuran {size}, galat sudut {out['kaaba']['fit']} m")
    out["hijr"] = poly_out([meters(p) for p in by[("way", HIJR)]["geometry"][:-1]], True)
    mq = [meters(p) for p in by[("way", MAQAM)]["geometry"][:-1]]
    out["maqam"] = pt(centroid(mq), True)
    out["corners"] = {n: pt(meters(by[("node", i)]), True) for n, i in corners.items() if ("node", i) in by}

    # ---------- Sa'i, tawaf, raml ----------
    out["safa"] = pt(meters(by[("node", SAFA)]))
    out["marwah"] = pt(meters(by[("node", MARWAH)]))
    out["saiPath"] = poly_out(dp(way_pts(by[("way", SAI_WAY)]), SIMPLIFY))
    out["tawafPath"] = poly_out(dp(way_pts(by[("way", TAWAF_WAY)]), SIMPLIFY))
    if ("way", RAML_WAY) in by:
        out["raml"] = poly_out(way_pts(by[("way", RAML_WAY)])[:-1])
    report.append(f"Safa {out['safa']}  Marwah {out['marwah']}  (jarak dunia {math.dist(out['safa'], out['marwah']):.0f} m)")

    # ---------- Pintu (lantai dasar, menempel garis masjid) ----------
    segs = []
    for ri, r in enumerate(outers):
        for j in range(len(r)):
            segs.append((r[j], r[(j + 1) % len(r)], ri))
    gates = []
    for e in ELS:
        t = e.get("tags", {})
        if e["type"] != "node" or "entrance" not in t:
            continue
        lv = levels_of(t)
        if lv is not None and "0" not in lv:
            continue
        p = meters(e)
        best = None
        for a, b, ri in segs:
            dx, dz = b[0] - a[0], b[1] - a[1]
            L2 = dx * dx + dz * dz or 1e-9
            u = max(0.0, min(1.0, ((p[0] - a[0]) * dx + (p[1] - a[1]) * dz) / L2))
            q = (a[0] + u * dx, a[1] + u * dz)
            d = math.dist(p, q)
            if best is None or d < best[0]:
                best = (d, q, (dx, dz), ri)
        if best[0] > GATE_SNAP:
            continue
        d, q, (dx, dz), ri = best
        L = math.hypot(dx, dz)
        # ring ccw (area > 0, z ke selatan): normal keluar = (dz, -dx)
        nx, nz = dz / L, -dx / L
        raw = t.get("name:en") or t.get("name") or ""
        ref = t.get("ref", "")
        m = re.match(r"^(.*?)[\s\-–]*(\d+)$", raw) or re.match(r"^(\d+)\s+(.*)$", raw)
        if m and not raw[:1].isdigit():
            ref = ref or m.group(2)
            if m.group(2) == ref:
                raw = m.group(1)
        elif m:
            ref = ref or m.group(1)
            if m.group(1) == ref:
                raw = m.group(2)
        latin = bool(re.search(r"[A-Za-z]", raw))
        name, name_ar = (raw.strip(" -–"), "") if latin else ("", raw.strip())
        gq = world(*q)
        # arah normal setelah skala radial hampir sama; cukup normalisasi ulang
        gates.append({"name": name, **({"nameAr": name_ar} if name_ar else {}), "ref": ref, "main": t.get("entrance") == "main",
                      "pos": [rnd(gq[0]), rnd(gq[1])], "dir": rnd(math.atan2(nx, nz), 3), "ring": ri, "osm": e["id"]})
    gates.sort(key=lambda g: (int(g["ref"]) if g["ref"].isdigit() else 9999, g["name"]))
    out["gates"] = gates
    named = [g for g in gates if g["name"]]
    report.append(f"pintu: {len(gates)} di lantai dasar ({sum(g['main'] for g in gates)} utama, {len(named)} bernama)")

    # ---------- Menara ----------
    mins = []
    for e in ELS:
        if e["type"] == "way" and e.get("tags", {}).get("man_made") == "tower" and e.get("geometry"):
            r = way_pts(e)[:-1] or way_pts(e)
            mins.append(pt(centroid(r)))
    out["minarets"] = mins
    report.append(f"menara: {len(mins)}")

    # ---------- Bangunan sekitar (siluet) ----------
    mosque_outer_m = outers
    skip_ids = {KAABA, HIJR, MAQAM}
    blds = []
    def add_building(rings, t, oid):
        if not rings:
            return
        r = max(rings, key=lambda r: abs(area(r)))
        c = centroid(r)
        a = abs(area(r))
        if math.hypot(*c) > BUILD_R or a < 25:
            return
        if any(inside(c, mo) for mo in mosque_outer_m) and not any(inside(c, mi) for mi in inners):
            return  # di dalam gedung masjid
        if any(inside(c, mi) for mi in inners):
            return  # di mataf (Ka'bah dkk. ditangani terpisah)
        b = t.get("building")
        if b in ("roof", "construction") or t.get("man_made") == "tower":
            return
        h = parse_h(t)
        if h is None:
            h = 6.0 if a < 400 else (18.0 if a < 2500 else 30.0)
        r = orient(simplify_ring(r, 1.0))
        if len(r) < 3:
            return
        if oid == CLOCK_REL:
            # height=601 milik menara jam; alas kompleks Abraj Al Bait ±60 m. Menara dicatat terpisah di titik
            # tengah kompleks (cocok dengan posisi "The Clock Towers" di Google Maps, ±40 m).
            out["clockTower"] = {"pos": pt(centroid(r)), "h": rnd(h * K)}
            h = 60.0
        item = {"poly": poly_out(r), "h": rnd(min(h, 601) * K)}
        n = t.get("name:en") or t.get("name")
        if n and (h >= 40 or a > 3000):
            item["name"] = n
        if b == "bridge":
            item["bridge"] = True
        blds.append(item)
    for e in ELS:
        t = e.get("tags", {})
        if "building" not in t or e["id"] in skip_ids:
            continue
        if e["type"] == "way" and e.get("geometry") and len(e["geometry"]) >= 4:
            add_building([way_pts(e)[:-1]], t, e["id"])
        elif e["type"] == "relation" and e["id"] != MOSQUE_REL:
            add_building(rel_rings(e, "outer"), t, e["id"])
    blds.sort(key=lambda b: -b["h"])
    out["buildings"] = blds
    report.append(f"menara jam: {out.get('clockTower')}")
    report.append(f"bangunan sekitar: {len(blds)} (tertinggi: {', '.join(b.get('name','?') + ' ' + str(b['h']) for b in blds[:4])})")

    # ---------- Plaza pejalan kaki (area tertutup) ----------
    plaza = []
    for e in ELS:
        t = e.get("tags", {})
        if e["type"] != "way" or not e.get("geometry"):
            continue
        g = e["geometry"]
        closed = len(g) >= 4 and g[0] == g[-1]
        if (t.get("highway") == "pedestrian" and closed and t.get("area") == "yes") or (("area:highway" in t or t.get("place") == "square") and closed):
            lv = levels_of(t)
            if lv is not None and "0" not in lv:
                continue
            r = orient(simplify_ring(way_pts(e)[:-1], 1.0))
            if abs(area(r)) > 200:
                plaza.append(poly_out(r))
    out["plaza"] = plaza
    report.append(f"plaza: {len(plaza)}")

    # ---------- Tempat tambahan (tidak ada di OSM) ----------
    extra = json.loads((HERE / "extra.json").read_text()) if (HERE / "extra.json").exists() else []
    out["landmarks"] = [{"name": x["name"], "kind": x.get("kind", "place"), "pos": pt(meters(x)), **({"h": rnd(x["h"] * K)} if x.get("h") else {})} for x in extra]
    report.append(f"tempat tambahan: {', '.join(l['name'] + ' ' + str(l['pos']) for l in out['landmarks'])}")

    # ---------- Batas dunia ----------
    allx = [p[0] for r in out["mosque"]["outer"] for p in r] + [p[0] for b in blds for p in b["poly"]]
    allz = [p[1] for r in out["mosque"]["outer"] for p in r] + [p[1] for b in blds for p in b["poly"]]
    out["bounds"] = [rnd(min(allx)), rnd(min(allz)), rnd(max(allx)), rnd(max(allz))]

    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text(json.dumps(out, ensure_ascii=False, separators=(",", ":")))
    report.append(f"tulis {OUT.relative_to(ROOT)}  {OUT.stat().st_size/1024:.0f} KB  batas {out['bounds']}")
    print("\n".join(report))
    if check:
        draw_check(out)


def draw_check(d):
    from PIL import Image, ImageDraw, ImageFont
    b = d["bounds"]
    S, pad = 1600, 40
    sc = (S - 2 * pad) / max(b[2] - b[0], b[3] - b[1])
    T = lambda p: (pad + (p[0] - b[0]) * sc, pad + (p[1] - b[1]) * sc)
    im = Image.new("RGB", (S, S), "#f7f5ef")
    g = ImageDraw.Draw(im)
    try:
        f = ImageFont.truetype("/System/Library/Fonts/Supplemental/Arial.ttf", 14)
        fb = ImageFont.truetype("/System/Library/Fonts/Supplemental/Arial Bold.ttf", 16)
    except OSError:
        f = fb = ImageFont.load_default()
    for p in d["plaza"]:
        g.polygon([T(q) for q in p], fill="#ebe6d8")
    for bl in d["buildings"]:
        shade = max(120, 220 - int(bl["h"]))
        g.polygon([T(q) for q in bl["poly"]], fill=(shade, shade - 8, shade - 20) if not bl.get("bridge") else (190, 210, 190), outline="#888")
    for bl in d["buildings"]:
        if bl.get("name"):
            c = centroid(bl["poly"])
            g.text(T(c), f"{bl['name'][:22]} ({bl['h']:.0f})", fill="#553", font=f, anchor="mm")
    for r in d["mosque"]["outer"]:
        g.polygon([T(q) for q in r], fill="#fffdf6", outline="#b0161e", width=3)
    for r in d["mosque"]["inner"]:
        g.polygon([T(q) for q in r], fill="#e8eef0", outline="#1e5fb0", width=3)
    g.line([T(q) for q in d["saiPath"]], fill="#169b46", width=3)
    g.polygon([T(q) for q in d["kaaba"]["poly"]], fill="black")
    g.line([T(q) for q in d["hijr"]], fill="#555", width=2)
    for m in d["minarets"]:
        x, y = T(m); g.ellipse([x - 4, y - 4, x + 4, y + 4], fill="#7a4a1e")
    for gt in d["gates"]:
        x, y = T(gt["pos"])
        col = "#c0262d" if gt["main"] else "#2b7a3d"
        r = 6 if gt["main"] else 3
        g.ellipse([x - r, y - r, x + r, y + r], fill=col)
        lab = (gt["ref"] + " " + gt["name"]).strip()
        if gt["main"] or gt["name"]:
            dx, dz = math.sin(gt["dir"]), math.cos(gt["dir"])
            g.text((x + dx * 14, y + dz * 14), lab, fill=col, font=fb if gt["main"] else f, anchor="mm")
        elif gt["ref"]:
            g.text((x + 5, y - 5), gt["ref"], fill=col, font=f)
    for n, key in (("Safa", "safa"), ("Marwah", "marwah")):
        x, y = T(d[key]); g.ellipse([x - 7, y - 7, x + 7, y + 7], fill="#169b46"); g.text((x + 10, y), n, fill="#0d6b30", font=fb, anchor="lm")
    for l in d["landmarks"]:
        x, y = T(l["pos"]); g.rectangle([x - 6, y - 6, x + 6, y + 6], fill="#e07b00"); g.text((x + 10, y), l["name"], fill="#a85a00", font=fb, anchor="lm")
    x, y = T((0, 0)); g.text((x, y + 22), "Ka'bah", fill="black", font=fb, anchor="mm")
    # utara & skala
    g.polygon([(S - 60, 40), (S - 70, 70), (S - 50, 70)], fill="black"); g.text((S - 60, 85), "U", fill="black", font=fb, anchor="mm")
    L = 50 * sc
    g.line([(pad, S - 30), (pad + L, S - 30)], fill="black", width=3)
    g.text((pad + L + 8, S - 30), f"50 m dunia (= {50 / d['k']:.0f} m asli)", fill="black", font=f, anchor="lm")
    g.text((pad, 12), f"Cek denah haram.json  K={d['k']}  OSM {d['osm']}  merah = pintu utama, hijau = pintu lain", fill="black", font=f)
    p = HERE / "cek.png"
    im.save(p)
    print("gambar", p.relative_to(ROOT))


if __name__ == "__main__":
    main()
