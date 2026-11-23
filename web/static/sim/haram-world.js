// Pembangun dunia open world dari data/haram.json. Lantai masjid, mataf, Mas'a, dan pelataran di y = 0.
// Gerbang "King" (gerbang raksasa) berdiri di teras setinggi satu lantai (LIFT) bertangga lebar: masuk gerbang King ->
// bordes -> eskalator TURUN ke lantai masjid. Pintu lain masuk rata; di samping pintu utama ada eskalator NAIK ke
// rooftop (atap datar yang bisa dijelajahi, lapisan tabrakan `colR`). Geometri statis digabung per jenis & irisan sudut.
// Mengisi `col` (lantai dasar) dan `colR` (rooftop). Kit detail (aula, fasad mataf, Mas'a) ada di haram-kit.js.
import * as THREE from 'three';
import { buildKaaba, KW, KD } from './kaaba.js?v=31';
import { Builder, inPoly, segDist, ringArea, createRegions, offsetRing, mergeGeos, paint, TAU } from './haram-geom.js?v=49';
import { makeTextures, plaqueAtlas, panel, buildMataf, buildHall, buildMasa, buildHalfMillion, PANEL_W } from './haram-kit.js?v=49';
import { createCollision } from './nav.js?v=49';

export { inPoly };
export const LIFT = 5;                  // teras gerbang King & bordes di baliknya (satu lantai di atas lantai masjid)
export const ROOF_Y = 15.5;             // bawah atap aula
export const ROOF_TOP = ROOF_Y + 0.8;   // lantai rooftop
export const WALL_TOP = ROOF_Y + 1.6;   // puncak dinding (parapet rooftop 0,8 m)
export const REG = { OUT: 0, HALL: 1, MATAF: 2 };
export const RAMP_LAND = 3, RAMP_RUN = LIFT / Math.tan(Math.PI / 6), RAMP_END = RAMP_LAND + RAMP_RUN; // eskalator turun King 30°
const UP_LAND = 3, UP_RUN = ROOF_TOP / Math.tan(35 * Math.PI / 180), UP_W = 3.6;                    // eskalator naik ke rooftop 35°
export const WELL_L = 8;                // sumur atap: lubang di atap dari bordes atas sampai 8 m ke bawah tanjakan
const WALL_T = 1.2;
const GATE = { king: { w: 9, h: 10.4 }, main: { w: 6, h: 10 }, side: { w: 3.2, h: 7 } }; // King: 6 orang berjajar; bukaan tinggi
const PORTAL = { king: { PW: 3.2, PD: 4.5 }, main: { PW: 1.3, PD: 1.6 } }; // lebar pilar & tonjolan portal
const TERR = { W: 9, D: 20 };            // teras King: lebar tambahan di kiri-kanan portal, kedalaman dari dinding

export function buildHaram(sim, data, col) {
  const { scene, M, mat, canvasTex } = sim;
  const colR = createCollision(8); // tabrakan lapisan rooftop
  const lmFloors = [], lmAts = []; // lantai berjalan & tapak tempat tambahan (Half Million)
  const vmat = (o = {}) => new THREE.MeshLambertMaterial({ vertexColors: true, ...o });
  const out = { roofs: [], labels: [] };
  const outers = data.mosque.outer, inner = data.mosque.inner;
  const tex = makeTextures(sim, { ROOF_Y, WALL_TOP });
  // Half Million: menghadap pusat masjid; posisi OSM digeser 2 m ke kiri & 10 m ke depan supaya tangga/teras tidak
  // menabrak hotel di belakangnya (dihitung dari poligon gedung).
  for (const l of data.landmarks || []) if (l.kind === 'cafe' && !l.dir) {
    const th = Math.atan2(-l.pos[0], -l.pos[1]), lx = -2, lz = 10;
    l.pos = [l.pos[0] + Math.cos(th) * lx + Math.sin(th) * lz, l.pos[1] - Math.sin(th) * lx + Math.cos(th) * lz]; l.dir = th;
  }

  // ---------- Peta wilayah: pelataran / aula / mataf ----------
  const region = createRegions(data.bounds, 0.5);
  outers.forEach((r) => region.fill(r, REG.HALL)); inner.forEach((h) => region.fill(h, REG.MATAF));
  const [SX, SZ] = data.safa, [MX, MZ] = data.marwah;
  const masaDist = (x, z) => {
    const dx = MX - SX, dz = MZ - SZ, L2 = dx * dx + dz * dz, u = Math.max(0, Math.min(1, ((x - SX) * dx + (z - SZ) * dz) / L2));
    return Math.hypot(x - SX - u * dx, z - SZ - u * dz);
  };

  // ---------- Pintu ----------
  const cum = outers.map((r) => { const c = [0]; for (let k = 0; k < r.length; k++) { const a = r[k], b = r[(k + 1) % r.length]; c.push(c[k] + Math.hypot(b[0] - a[0], b[1] - a[1])); } return c; });
  const edgeFrame = (ri, k) => {
    const r = outers[ri], a = r[k], b = r[(k + 1) % r.length], dx = b[0] - a[0], dz = b[1] - a[1], L = Math.hypot(dx, dz) || 1e-6;
    const t = [dx / L, dz / L], m = [(a[0] + b[0]) / 2, (a[1] + b[1]) / 2], s = ringArea(r) > 0 ? 1 : -1;
    let n = [s * t[1], -s * t[0]]; // normal keluar menurut arah putaran ring; dicek dengan peta wilayah
    const outP = region.at(m[0] + n[0], m[1] + n[1]) === REG.OUT, outM = region.at(m[0] - n[0], m[1] - n[1]) === REG.OUT;
    if (!outP && outM) n = [-n[0], -n[1]];
    return { a, b, L, t, n };
  };
  const gates = [];
  const addGate = (g, ri, k, u, kind, synthetic = false) => {
    const f = edgeFrame(ri, k), { w, h } = GATE[kind], ul = u * f.L;
    gates.push({ g, ring: ri, k, pp: cum[ri][k] + ul, kind, w, hw: w / 2, h, c: [f.a[0] + f.t[0] * ul, f.a[1] + f.t[1] * ul], t: f.t, n: f.n, open: true, synthetic });
  };
  for (const g of data.gates) {
    const r = outers[g.ring]; let best = null;
    r.forEach((a, k) => { const s = segDist(g.pos[0], g.pos[1], a, r[(k + 1) % r.length]); if (!best || s.d < best.d) best = { ...s, k }; });
    if (!best || best.d > 1.5) continue;
    addGate(g, g.ring, best.k, best.u, /^King /.test(g.name || '') ? 'king' : g.main ? 'main' : 'side');
  }
  // Ring tanpa pintu lantai dasar di OSM (perluasan King Abdullah): pintu tanpa nomor di sisi panjang yang menghadap Ka'bah.
  outers.forEach((r, ri) => {
    if (gates.some((g) => g.ring === ri)) return;
    for (let k = 0; k < r.length; k++) {
      const f = edgeFrame(ri, k), m = [(f.a[0] + f.b[0]) / 2, (f.a[1] + f.b[1]) / 2], d = Math.hypot(m[0], m[1]) || 1;
      if (f.L < 14 || -(f.n[0] * m[0] + f.n[1] * m[1]) / d < 0.75) continue;
      const cnt = f.L >= 34 ? 2 : 1;
      for (let i = 0; i < cnt; i++) addGate({ name: '', ref: '', main: false, synthetic: true }, ri, k, (i + 1) / (cnt + 1), 'side', true);
    }
  });

  // ---------- Bordes + eskalator per kelompok pintu (pintu berdempetan digabung jadi satu deret) ----------
  gates.sort((a, b) => a.ring - b.ring || a.pp - b.pp);
  const clusters = [];
  for (const g of gates) {
    const last = clusters[clusters.length - 1], p = last && last[last.length - 1];
    if (p && p.ring === g.ring && (g.pp - g.hw) - (p.pp + p.hw) < 2.0 && p.t[0] * g.t[0] + p.t[1] * g.t[1] > 0.94) last.push(g); else clusters.push([g]);
  }
  const wp = (r, u, s) => [r.c[0] + r.t[0] * u - r.n[0] * s, r.c[1] + r.t[1] * u - r.n[1] * s]; // s = arah turun (ke dalam masjid)
  const RANK = { king: 3, main: 2, side: 1 };
  // Eskalator turun hanya untuk gerbang King (pintu lain masuk rata).
  let ramps = clusters.filter((cl) => cl.some((g) => g.kind === 'king')).map((cl) => {
    const f = cl[0], l = cl[cl.length - 1];
    const p0 = [f.c[0] - f.t[0] * f.hw, f.c[1] - f.t[1] * f.hw], p1 = [l.c[0] + l.t[0] * l.hw, l.c[1] + l.t[1] * l.hw];
    let tx = p1[0] - p0[0], tz = p1[1] - p0[1]; const W = Math.hypot(tx, tz) || 1; tx /= W; tz /= W;
    let n = [tz, -tx]; if (n[0] * f.n[0] + n[1] * f.n[1] < 0) n = [-n[0], -n[1]];
    return { c: [(p0[0] + p1[0]) / 2, (p0[1] + p1[1]) / 2], t: [tx, tz], n, hw: W / 2, gates: cl, kind: 'king', H: LIFT, RL: RAMP_LAND, RUN: RAMP_RUN, RE: RAMP_END, s0: -1 };
  });
  ramps = ramps.filter((r) => {
    for (let s = 1.5; s <= r.RE + 0.8; s += 1) for (const u of [-r.hw + 0.4, 0, r.hw - 0.4]) { const p = wp(r, u, s); if (region.at(p[0], p[1]) === REG.OUT) { r.gates.forEach((g) => (g.kind = 'main')); return false; } }
    return true; // tidak muat: gerbang tetap dibuka tapi rata (tanpa teras)
  });
  const rampY = (r, s) => (s <= r.RL ? r.H : Math.max(0, r.H * (1 - (s - r.RL) / r.RUN)));
  const corners = (r, m = 0) => [[-r.hw - m, r.s0 - m], [r.hw + m, r.s0 - m], [r.hw + m, r.RE + m], [-r.hw - m, r.RE + m]].map(([u, s]) => wp(r, u, s));
  const RC = 16, rkey = (i, j) => i * 100003 + j;
  const makeGrid = (list) => {
    const grid = new Map();
    for (const r of list) {
      const cs = corners(r, 2), xs = cs.map((p) => p[0]), zs = cs.map((p) => p[1]);
      for (let i = Math.floor(Math.min(...xs) / RC); i <= Math.floor(Math.max(...xs) / RC); i++)
        for (let j = Math.floor(Math.min(...zs) / RC); j <= Math.floor(Math.max(...zs) / RC); j++) { const k = rkey(i, j); if (!grid.has(k)) grid.set(k, []); grid.get(k).push(r); }
    }
    return (x, z, m = 0, land = 0) => { // land: perluasan bordes ke sisi dinding (rooftop)
      const l = grid.get(rkey(Math.floor(x / RC), Math.floor(z / RC))); if (!l) return null;
      for (const r of l) {
        const dx = x - r.c[0], dz = z - r.c[1], s = -(dx * r.n[0] + dz * r.n[1]), u = dx * r.t[0] + dz * r.t[1];
        const uMin = -r.hw - m - (land && s <= r.RL ? land : 0);
        if (u >= uMin && u <= r.hw + m && s >= r.s0 - m && s <= r.RE + m) return { r, s, u, y: rampY(r, Math.max(0, s)) };
      }
      return null;
    };
  };
  const rampAt = makeGrid(ramps);
  const nearRamp = (x, z, m = 0) => rampAt(x, z, m) !== null;

  // ---------- Teras gerbang King (setinggi LIFT, tangga lebar ke pelataran) ----------
  const terraces = [];
  const inBld = (x, z) => data.buildings.some((b) => !b.bridge && Math.abs(b.poly[0][0] - x) < 150 && inPoly(x, z, b.poly)) || (data.landmarks || []).some((l) => Math.hypot(l.pos[0] - x, l.pos[1] - z) < (l.kind === 'cafe' ? 17 : 5));
  for (const r of ramps) for (const g of r.gates) if (g.kind === 'king') {
    const PW = PORTAL.king.PW;
    let W = g.hw + PW + TERR.W, D = TERR.D, ok = false;
    const free = (W, D) => {
      for (let o = 1; o <= D + RAMP_RUN + 1; o += 2) for (let u = -W - 1; u <= W + 1; u += 2) {
        const x = g.c[0] + g.t[0] * u + g.n[0] * o, z = g.c[1] + g.t[1] * u + g.n[1] * o;
        if (region.at(x, z) !== REG.OUT || inBld(x, z)) return false;
      }
      return true;
    };
    for (const [w, d] of [[W, D], [g.hw + PW + 4, D], [W, 14], [g.hw + PW + 4, 12]]) if (free(w, d)) { W = w; D = d; ok = true; break; }
    if (!ok) { W = g.hw + PW + 2; D = 9; }
    // sisi belakang teras mengikuti garis dinding (dinding bisa menekuk) -> tidak ada lubang antara teras & dinding
    const per = cum[g.ring][outers[g.ring].length], ringPt = (pp) => {
      const r = outers[g.ring]; pp = ((pp % per) + per) % per; let k = 0; while (k < r.length - 1 && cum[g.ring][k + 1] < pp) k++;
      const a = r[k], b = r[(k + 1) % r.length], L = (cum[g.ring][k + 1] - cum[g.ring][k]) || 1, u = (pp - cum[g.ring][k]) / L;
      return [a[0] + (b[0] - a[0]) * u, a[1] + (b[1] - a[1]) * u];
    };
    const back = [ringPt(g.pp - W)];
    for (let k = 0; k < outers[g.ring].length; k++) { const c = cum[g.ring][k]; for (const sh of [0, per, -per]) if (c + sh > g.pp - W && c + sh < g.pp + W) back.push([c + sh, outers[g.ring][k]]); }
    const mid = back.slice(1).sort((p, q) => p[0] - q[0]).map((p) => p[1]);
    const F = (u, o) => [g.c[0] + g.t[0] * u + g.n[0] * o, g.c[1] + g.t[1] * u + g.n[1] * o];
    const poly = [back[0], ...mid, ringPt(g.pp + W), F(W, D), F(-W, D)];
    terraces.push({ c: g.c, t: g.t, n: g.n, W, D, S: RAMP_RUN, g, ring: g.ring, poly });
  }
  // pintu lain yang jatuh di bentang teras: ditutup (ambangnya setinggi teras, tanpa eskalator turun)
  for (const g of gates) if (g.kind !== 'king') for (const tr of terraces) {
    const dx = g.c[0] - tr.c[0], dz = g.c[1] - tr.c[1], u = dx * tr.t[0] + dz * tr.t[1], o = dx * tr.n[0] + dz * tr.n[1];
    if (g.ring === tr.ring && Math.abs(u) < tr.W + g.hw + 1 && Math.abs(o) < 3) g.open = false;
  }
  const terraceAt = (x, z) => {
    for (const tr of terraces) {
      const dx = x - tr.c[0], dz = z - tr.c[1], u = dx * tr.t[0] + dz * tr.t[1], o = dx * tr.n[0] + dz * tr.n[1];
      if (Math.abs(u) > tr.W + 1 || o > tr.D + tr.S || o < -12 || region.at(x, z) !== REG.OUT) continue;
      if (o > tr.D) { if (Math.abs(u) <= tr.W) return LIFT * (1 - (o - tr.D) / tr.S); continue; } // tangga lebar
      if (inPoly(x, z, tr.poly)) return LIFT;
    }
    return null;
  };

  // ---------- Eskalator naik ke rooftop: di DALAM aula, sejajar dinding, di samping pintu utama ----------
  // Masuk pintu biasa -> belok ke samping -> eskalator naik menyusuri sisi dalam dinding dan menembus atap lewat
  // sumur (lubang berpagar) ke rooftop. Ujung bawah dekat pintu, bordes atas menjauhi pintu. Gerbang King: turun.
  const upRamps = [];
  {
    const cand = gates.filter((g) => g.open && g.kind !== 'king').sort((a, b) => RANK[b.kind] - RANK[a.kind] || (a.synthetic ? 1 : 0) - (b.synthetic ? 1 : 0));
    const RE = UP_LAND + UP_RUN, hw = UP_W / 2;
    for (const g of cand) {
      if (upRamps.length >= 14) break;
      if (upRamps.some((q) => Math.hypot(q.g.c[0] - g.c[0], q.g.c[1] - g.c[1]) < 55)) continue;
      const gw = g.hw + (PORTAL[g.kind] ? PORTAL[g.kind].PW : 0) + 1.2; // dari tepi pilar portal
      let placed = false;
      // off = jarak sumbu eskalator ke dalam dari garis dinding: menempel dinding (celah 0,2 m); bila ada pintu samping
      // di bentangnya, dicoba lebih ke dalam dengan lorong 2,4 m antara dinding & eskalator supaya pintu itu tetap lewat.
      for (const off of [WALL_T / 2 + 0.2 + hw, WALL_T / 2 + 2.4 + hw]) {
        const tight = off < WALL_T / 2 + 1 + hw;
        for (const dir of [1, -1]) {
          const p0 = g.pp + dir * gw, p1 = p0 + dir * (RE + 0.6), lo = Math.min(p0, p1), hi = Math.max(p0, p1);
          if (gates.some((q) => q !== g && q.open && q.ring === g.ring && q.pp + q.hw > lo - 0.8 && q.pp - q.hw < hi + 0.8 && (tight || q.kind !== 'side'))) continue;
          let ok = true;
          for (let sx = -0.6; sx <= RE + 2 && ok; sx += 1.5) for (const o of [off - hw - 0.4, off, off + hw + 0.4]) {
            const x = g.c[0] + g.t[0] * dir * (gw + sx) - g.n[0] * o, z = g.c[1] + g.t[1] * dir * (gw + sx) - g.n[1] * o;
            if (region.at(x, z) !== REG.HALL || nearRamp(x, z, 1.5) || masaDist(x, z) < 14 || upRamps.some((q) => Math.hypot(q.c[0] - x, q.c[1] - z) < 8)) { ok = false; break; }
          }
          if (!ok) continue;
          // bingkai: s = 0 di bordes atas (jauh dari pintu), bertambah menuruni tanjakan ke arah pintu; u melintang (+ ke dalam)
          const cx = g.c[0] + g.t[0] * dir * (gw + RE) - g.n[0] * off, cz = g.c[1] + g.t[1] * dir * (gw + RE) - g.n[1] * off;
          upRamps.push({ c: [cx, cz], t: [-g.n[0], -g.n[1]], n: [dir * g.t[0], dir * g.t[1]], hw, H: ROOF_TOP, RL: UP_LAND, RUN: UP_RUN, RE, s0: 0, up: true, g, ring: g.ring, tight });
          placed = true; break;
        }
        if (placed) break;
      }
    }
  }
  const upAt = makeGrid(upRamps);
  // Permukaan eskalator yang bisa diinjak dari lapisan `layer`: di rooftop hanya bagian di dalam sumur (atap di atas
  // sisa tanjakan tetap lantai atap); di lantai dasar seluruh tanjakan.
  const upWalk = (x, z, layer = 0, m = 0) => { const h = upAt(x, z, m); return h && (!layer || h.s <= h.r.RL + WELL_L) ? h : null; };
  // Tinggi lantai di (x, z) pada lapisan `layer` (0 = lantai dasar, 1 = rooftop).
  const floorY = (x, z, layer = 0) => {
    if (layer) { const h = upWalk(x, z, 1); return h ? h.y : ROOF_TOP; }
    let h = rampAt(x, z); if (h) return h.s <= RAMP_LAND ? LIFT : h.y;
    h = upAt(x, z); if (h) return h.y;
    const t = terraceAt(x, z); if (t != null) return t;
    for (const f of lmFloors) { const y = f(x, z); if (y != null) return y; } // tangga & teras Half Million
    return 0;
  };

  // ---------- Pembangun bersama ----------
  const solid = Builder(4), facadeB = Builder(4), roofB = Builder(), archB = Builder(4), stepB = Builder();
  const nameB = Builder(), numB = Builder(), V = solid.V;

  // ---------- Pelataran (setinggi lantai masjid), berlubang di masjid ----------
  const tileTex = canvasTex(512, 512, (g, s) => {
    g.fillStyle = '#f3f3f1'; g.fillRect(0, 0, s, s);
    for (let i = 0; i < 1400; i++) { g.fillStyle = `rgba(105,102,98,${Math.random() * 0.03})`; g.fillRect(Math.random() * s, Math.random() * s, 2 + Math.random() * 8, 1 + Math.random() * 2); }
    g.strokeStyle = 'rgba(150,140,120,.35)'; g.lineWidth = 3;
    for (let x = 0; x <= s; x += 128) { g.beginPath(); g.moveTo(x, 0); g.lineTo(x, s); g.stroke(); g.beginPath(); g.moveTo(0, x); g.lineTo(s, x); g.stroke(); }
  });
  tileTex.wrapS = tileTex.wrapT = THREE.RepeatWrapping;
  const [bx0, bz0, bx1, bz1] = data.bounds, GR = Math.max(-bx0, bx1, -bz0, bz1) + 140;
  const groundB = Builder();
  groundB.cap([[-GR, -GR], [GR, -GR], [GR, GR], [-GR, GR]], outers, 0, '#ffffff', true, (x, z) => [x / 6, z / 6]);
  scene.add(groundB.mesh(new THREE.MeshLambertMaterial({ map: tileTex, vertexColors: true })));
  out.groundR = GR;

  // ---------- Lantai masjid & mataf (y = 0) ----------
  const floorB = Builder();
  outers.forEach((r) => floorB.cap(r, inner.filter((h) => inPoly(h[0][0], h[0][1], r)), 0.04, '#f9fbfd')); // marmer putih (sedikit dingin: cahaya matahari hangat)
  scene.add(floorB.mesh(vmat()));
  const matafTex = canvasTex(1024, 1024, (g, s) => {
    g.fillStyle = '#fafbfc'; g.fillRect(0, 0, s, s);
    for (let i = 0; i < 2600; i++) { g.fillStyle = `rgba(90,90,80,${Math.random() * 0.03})`; g.fillRect(Math.random() * s, Math.random() * s, 2 + Math.random() * 7, 1 + Math.random() * 2); }
    g.strokeStyle = 'rgba(140,145,150,.42)'; g.lineWidth = 2;
    for (let r = 40; r < 720; r += 22) { g.beginPath(); g.arc(s / 2, s / 2, r, 0, TAU); g.stroke(); }
  });
  const MR = 62;
  const matafB = Builder();
  inner.forEach((h) => matafB.cap(h, null, 0.06, '#ffffff', true, (x, z) => [x / (2 * MR) + 0.5, 0.5 - z / (2 * MR)]));
  scene.add(matafB.mesh(new THREE.MeshLambertMaterial({ map: matafTex, vertexColors: true })));

  // ---------- Dinding luar (fasad jendela lengkung) + celah pintu ----------
  const uvF = { w: PANEL_W, h: WALL_TOP };
  const gBase = (g) => (g.kind === 'king' ? LIFT : 0);
  outers.forEach((r, ri) => {
    // celah dalam koordinat keliling ring (boleh melewati sudut), digabung bila tumpang tindih
    // Pintu berportal: celah dinding dilebarkan setengah lebar pilar, jadi ujung dinding tersembunyi di dalam pilar
    // (tidak ada dua bidang berimpit = tidak berkelap-kelip). Ambang (lintel) digantikan balok atas portal.
    const gaps = gates.filter((g) => g.ring === ri && g.open).map((g) => {
      const pt = PORTAL[g.kind], gw = g.hw + (pt ? pt.PW / 2 : 0);
      return [g.pp - gw, g.pp + gw, gBase(g) + g.h, !!pt];
    }).sort((a, b) => a[0] - b[0]);
    const merged = [];
    for (const g of gaps) { const l = merged[merged.length - 1]; if (l && g[0] < l[1]) { l[1] = Math.max(l[1], g[1]); l[2] = Math.min(l[2], g[2]); l[3] = l[3] && g[3]; } else merged.push(g.slice()); }
    for (let k = 0; k < r.length; k++) {
      const a = r[k], b = r[(k + 1) % r.length], e0 = cum[ri][k], e1 = cum[ri][k + 1], L = e1 - e0;
      if (L < 0.05) continue;
      const P = (pp) => { const u = (pp - e0) / L; return [a[0] + (b[0] - a[0]) * u, a[1] + (b[1] - a[1]) * u]; };
      const piece = (p0, p1) => { const A = P(p0), B = P(p1); facadeB.slab(A, B, WALL_T, 0, WALL_TOP, '#ffffff', uvF, p0); col.addSeg(A[0], A[1], B[0], B[1], { h: WALL_TOP, tag: 'wall' }); };
      let cur = e0;
      for (const [g0, g1, gtop, portal] of merged) {
        if (g1 <= e0 || g0 >= e1) continue;
        const q0 = Math.max(g0, e0), q1 = Math.min(g1, e1);
        if (q0 > cur) piece(cur, q0);
        const l0 = P(q0), l1 = P(q1);
        if (!portal) solid.slab(l0, l1, WALL_T, gtop, WALL_TOP - 0.03, '#e1ded8');
        col.addSeg(l0[0], l0[1], l1[0], l1[1], { h: WALL_TOP, lo: gtop, move: false, tag: 'wall' }); // ambang: kamera saja
        cur = Math.max(cur, q1);
      }
      if (cur < e1) piece(cur, e1);
      colR.addSeg(a[0], a[1], b[0], b[1], { h: WALL_TOP, tag: 'wall' }); // rooftop: parapet sepanjang tepi luar
    }
  });

  // ---------- Bordes & eskalator (turun di gerbang King, naik ke rooftop di dalam aula) ----------
  const LAND_UP = 0.06, wells = outers.map(() => []);
  for (const r of [...ramps, ...upRamps]) {
    const H = r.H, RL = r.RL, RE = r.RE, RUN = r.RUN, up = !!r.up;
    const surf = (s) => rampY(r, s);
    const P3 = (u, s, y) => { const p = wp(r, u, s); return V(p[0], y, p[1]); };
    const hw = r.hw, inward = V(-r.n[0], 0, -r.n[1]), upv = V(0, 1, 0);
    const slopeN = inward.clone().multiplyScalar(H / RUN).add(upv).normalize();
    // Dinding miring tipis (pagar/balustrade) antara uIn..uOut dari s = sFrom sampai ujung bawah, setinggi hgt di atas permukaan.
    const slopeWall = (uIn, uOut, hgt, color, sFrom) => {
      const ss = [sFrom, Math.max(sFrom, RL), RE].filter((s, i, a) => i === 0 || s > a[i - 1] + 1e-3);
      for (let i = 0; i + 1 < ss.length; i++) {
        const s0 = ss[i], s1 = ss[i + 1], y0 = s1 <= RL + 1e-3 ? H + LAND_UP : surf(s0), y1 = s1 <= RL + 1e-3 ? H + LAND_UP : surf(s1);
        for (const [u, sg] of [[uIn, -1], [uOut, 1]]) solid.quad(P3(u, s0, y0), P3(u, s1, y1), P3(u, s1, y1 + hgt), P3(u, s0, y0 + hgt), V(r.t[0] * sg, 0, r.t[1] * sg), color);
        const tn = inward.clone().multiplyScalar(-(y1 - y0) / (s1 - s0)).add(upv).normalize();
        solid.quad(P3(uIn, s0, y0 + hgt), P3(uOut, s0, y0 + hgt), P3(uOut, s1, y1 + hgt), P3(uIn, s1, y1 + hgt), tn, '#cac6be');
      }
      solid.quad(P3(uIn, RE, 0), P3(uOut, RE, 0), P3(uOut, RE, hgt), P3(uIn, RE, hgt), inward, color);
    };
    solid.prism([wp(r, -hw, 0), wp(r, hw, 0), wp(r, hw, RL), wp(r, -hw, RL)], 0, H + LAND_UP, '#f3f4f4'); // bordes (6 cm di atas lantai sekitarnya)
    const wide = hw * 2 >= 5;
    const esc = wide ? [[-hw + 0.3, -hw + 1.6], [hw - 1.6, hw - 0.3]] : [[-hw + 0.3, -hw + 1.6]];
    const stairs = wide ? [[-hw + 1.75, hw - 1.75]] : [[-hw + 1.75, hw - 0.3]];
    const vr = RUN / 3.2;
    for (const [u0, u1] of esc) {
      stepB.quad(P3(u0, RL, H + 0.03), P3(u1, RL, H + 0.03), P3(u1, RE, 0.05), P3(u0, RE, 0.05), slopeN, '#ffffff', [[0, vr], [1, vr], [1, 0], [0, 0]]);
      const u = u0 < 0 ? u1 + 0.075 : u0 - 0.075; // pagar eskalator di sisi tangga saja
      slopeWall(u - 0.06, u + 0.06, 1.0, '#3d464f', RL - 0.4);
    }
    const N = Math.max(8, Math.round(H / 0.31)), rise = H / N, run = RUN / N, stepUp = V(0, 1, 0);
    for (const [u0, u1] of stairs) {
      for (let i = 0; i < N; i++) {
        const s0 = RL + i * run, y0 = H - i * rise, y1 = H - (i + 1) * rise;
        if (i > 0) solid.quad(P3(u0, s0, y0), P3(u1, s0, y0), P3(u1, s0, y1), P3(u0, s0, y1), inward, '#eceeee'); // tegak pertama = muka bordes
        if (i < N - 1) solid.quad(P3(u0, s0, y1), P3(u1, s0, y1), P3(u1, s0 + run, y1), P3(u0, s0 + run, y1), stepUp, i % 2 ? '#f8f9fa' : '#f2f3f4');
      }
    }
    for (const [u0, u1] of [[-hw, -hw + 0.3], [hw - 0.3, hw], ...(wide ? [[-hw + 1.6, -hw + 1.75], [hw - 1.75, hw - 1.6]] : [[-hw + 1.6, -hw + 1.75]])])
      solid.quad(P3(u0, RL, H + 0.02), P3(u1, RL, H + 0.02), P3(u1, RE, 0.04), P3(u0, RE, 0.04), slopeN, '#e3e5e6');
    // sisi blok (segitiga di bawah tangga) dan pagar pembatas. Eskalator rooftop: sisi dinding (u < 0) terbuka di
    // bordes (menyambung ke atap), ujung bordes berpagar.
    for (const [u, sg] of [[-hw, -1], [hw, 1]]) {
      const nn = V(r.t[0] * sg, 0, r.t[1] * sg);
      solid.tri(P3(u, RL, 0), P3(u, RE, 0), P3(u, RL, H), nn, solid.color('#f3f4f4'));
      slopeWall(sg < 0 ? u : u - 0.3, sg < 0 ? u + 0.3 : u, 1.1, '#eae8e3', up ? 0 : 0.72);
    }
    const seg = (c, u0, s0, u1, s1, o) => { const A = wp(r, u0, s0), B = wp(r, u1, s1); c.addSeg(A[0], A[1], B[0], B[1], o); };
    const rail = { h: H + 1.4, tag: 'solid' };
    // pagar samping: dipotong per <= 5 m, tinggi (bagi kamera) mengikuti kemiringan -> kamera tidak perlu naik di bagian rendah
    const sideRail = (c, u, sFrom, sTo = RE) => { for (let a0 = sFrom; a0 < sTo - 1e-3;) { const a1 = Math.min(sTo, a0 + 5); seg(c, u, a0, u, a1, { h: rampY(r, Math.max(0, a0)) + 1.4, tag: 'solid' }); a0 = a1; } };
    if (!up) { sideRail(col, -hw, r.s0); sideRail(col, hw, r.s0); continue; }
    // Lantai dasar: kedua sisi & ujung bordes (s = 0) tertutup (blok eskalator pejal), masuk hanya dari ujung bawah.
    sideRail(col, -hw, -0.75); sideRail(col, hw, -0.75); seg(col, -hw, -0.02, hw, -0.02, { h: H + LAND_UP, tag: 'solid' }); // ujung atas terbuka ke atap: tinggi = bordes
    // Sumur atap: lubang di lempeng atap sepanjang bordes + bagian atas tanjakan (kamera dari belakang/bawah tetap
    // melihat lewat lubang), dipagari di rooftop; ujung bordes (s = 0) terbuka ke atap. Bordes mengisi lubang (celah
    // 8 cm di sisi), jadi tidak ada dua bidang berimpit.
    const WL = RL + WELL_L, gap = 0.08, pw = 0.28;
    wells[r.ring].push([wp(r, -hw - gap, -0.02), wp(r, hw + gap, -0.02), wp(r, hw + gap, WL), wp(r, -hw - gap, WL)]);
    for (const [u0, u1] of [[-hw - gap - pw, -hw - gap], [hw + gap, hw + gap + pw]]) // pagar sisi: mulai saat pagar eskalator sendiri turun di bawah atap
      solid.prism([wp(r, u0, RL + 0.8), wp(r, u1, RL + 0.8), wp(r, u1, WL + pw), wp(r, u0, WL + pw)], ROOF_TOP, ROOF_TOP + 1.1, '#eae8e3');
    solid.prism([wp(r, -hw - gap - pw, WL), wp(r, hw + gap + pw, WL), wp(r, hw + gap + pw, WL + pw), wp(r, -hw - gap - pw, WL + pw)], ROOF_TOP, ROOF_TOP + 1.1, '#eae8e3');
    sideRail(colR, -hw, -1.4, WL + pw); sideRail(colR, hw, -1.4, WL + pw); seg(colR, -hw - 0.4, WL + pw / 2, hw + 0.4, WL + pw / 2, { h: ROOF_TOP + 1.4, tag: 'solid' });
    out.labels.push({ text: 'Eskalator ke atap', pos: (() => { const p = wp(r, 0, RE - 1); return new THREE.Vector3(p[0], 4.5, p[1]); })(), far: 45 });
    out.labels.push({ text: 'Eskalator turun', pos: (() => { const p = wp(r, 0, -1.5); return new THREE.Vector3(p[0], ROOF_TOP + 2.6, p[1]); })(), far: 22 });
  }

  // ---------- Teras gerbang King ----------
  const tileB = Builder();
  for (const tr of terraces) {
    const T = (u, o) => [tr.c[0] + tr.t[0] * u + tr.n[0] * o, tr.c[1] + tr.t[1] * u + tr.n[1] * o];
    const T3 = (u, o, y) => { const p = T(u, o); return V(p[0], y, p[1]); };
    const { W, D, S: SR } = tr, outN = V(tr.n[0], 0, tr.n[1]), upv = V(0, 1, 0);
    // badan teras: tutup atas (poligon yang menempel dinding) bertekstur pelataran, sisi batu berpagar
    tileB.cap(tr.poly, null, LIFT, '#ffffff', true, (x, z) => [x / 6, z / 6]);
    const P0 = tr.poly[0], P1 = tr.poly[tr.poly.length - 3]; // ujung dinding kiri (-W) & kanan (+W)
    for (const [wallPt, u, sg] of [[P0, -W, -1], [P1, W, 1]]) {
      const front = T(u, D), V3 = (p, y) => V(p[0], y, p[1]);
      const ex = front[0] - wallPt[0], ez = front[1] - wallPt[1], el = Math.hypot(ex, ez) || 1;
      let nx = -ez / el, nz = ex / el; if (nx * tr.t[0] + nz * tr.t[1] < 0) { nx = -nx; nz = -nz; } if (sg < 0) { nx = -nx; nz = -nz; }
      const nn = V(nx * (sg < 0 ? -1 : 1) * (sg < 0 ? -1 : 1), 0, nz);
      solid.quad(V3(wallPt, 0), V3(front, 0), V3(front, LIFT), V3(wallPt, LIFT), V(nx, 0, nz), '#d8d5cf');
      solid.tri(T3(u, D, 0), T3(u, D + SR, 0), T3(u, D, LIFT), V(tr.t[0] * sg, 0, tr.t[1] * sg), solid.color('#d8d5cf'));
      // pagar: sepanjang sisi teras lalu menurun di sisi tangga (dua muka, tebal 0,35 m ke arah dalam)
      const inPt = (p) => [p[0] - nx * 0.35, p[1] - nz * 0.35];
      for (const [A, B] of [[wallPt, front], [inPt(wallPt), inPt(front)]]) solid.quad(V3(A, LIFT), V3(B, LIFT), V3(B, LIFT + 1.1), V3(A, LIFT + 1.1), V(nx, 0, nz), '#eae8e3');
      for (const uu of [u, u - sg * 0.35]) solid.quad(T3(uu, D, LIFT), T3(uu, D + SR, 0), T3(uu, D + SR, 1.1), T3(uu, D, LIFT + 1.1), V(tr.t[0] * sg, 0, tr.t[1] * sg), '#eae8e3');
      col.addSeg(wallPt[0], wallPt[1], front[0], front[1], { h: LIFT + 1.4, tag: 'solid' });
      col.addSeg(...front, ...T(u, D + SR), { h: LIFT + 1.4, tag: 'solid' });
      void nn;
    }
    // tangga lebar ke pelataran
    const N = 16, rise = LIFT / N, run = SR / N;
    for (let i = 0; i < N; i++) {
      const o0 = D + i * run, y0 = LIFT - i * rise, y1 = LIFT - (i + 1) * rise;
      if (i > 0) solid.quad(T3(-W, o0, y0), T3(W, o0, y0), T3(W, o0, y1), T3(-W, o0, y1), outN, '#e9e8e6');
      else solid.quad(T3(-W, o0, LIFT), T3(W, o0, LIFT), T3(W, o0, y1), T3(-W, o0, y1), outN, '#e9e8e6');
      if (i < N - 1) solid.quad(T3(-W, o0, y1), T3(W, o0, y1), T3(W, o0 + run, y1), T3(-W, o0 + run, y1), upv, i % 2 ? '#f4f3f1' : '#f1f0ed'); // anak tangga terbawah = pelataran
    }
  }

  // ---------- Gerbang: portal King raksasa, portal pintu utama, lengkung & nomor di semua pintu ----------
  const portals = [];
  const named = gates.filter((g) => g.open && g.kind !== 'side' && (g.g.name || g.g.ref));
  const nums = [...new Set(gates.filter((g) => g.open && g.kind === 'side' && g.g.ref).map((g) => g.g.ref))];
  const atlas = plaqueAtlas(sim, named.map((g) => ({ key: g.g.osm, en: g.g.name || `Gate ${g.g.ref}`, ar: g.g.nameAr, ref: g.g.ref })), nums);
  const numDone = new Set();
  for (const g of gates) {
    const G = (u, s) => [g.c[0] + g.t[0] * u - g.n[0] * s, g.c[1] + g.t[1] * u - g.n[1] * s];
    const rect = (u0, u1, s0, s1) => [G(u0, s0), G(u1, s0), G(u1, s1), G(u0, s1)];
    const hw = g.hw, base = gBase(g), top = base + g.h;
    // kiri/kanan dilihat dari pelataran (menghadap -n): papan nama tidak boleh tercermin
    const R = g.t[0] * g.n[1] - g.t[1] * g.n[0] > 0 ? 1 : -1;
    if (!g.open) { // pintu tertutup (mis. di bentang teras King): daun pintu di muka luar dinding
      panel(solid, G(-hw, -0.7), G(hw, -0.7), base, base + Math.min(g.h, 5.2), '#6b4f34');
      continue;
    }
    if (g.kind === 'king' || g.kind === 'main') {
      const king = g.kind === 'king', { PW, PD } = PORTAL[g.kind], PH = king ? LIFT + 21 : WALL_TOP + 1.5;
      for (const [u0, u1] of [[-hw - PW, -hw], [hw, hw + PW]]) {
        const ring = rect(u0, u1, -PD, 0.65);
        solid.prism(ring, 0, PH, king ? '#d9d6d0' : '#e2dfda'); col.addPoly(ring, { h: PH + 0.7, tag: 'wall' });
      }
      solid.prism(rect(-hw - PW, hw + PW, -PD, 0.66), top, PH, king ? '#d9d6d0' : '#e2dfda', true);
      colR.addPoly(rect(-hw - PW, hw + PW, -PD, 0.66), { h: PH + 0.7, tag: 'wall' }); // menonjol di atas rooftop
      solid.prism(rect(-hw - PW - 0.35, hw + PW + 0.35, -PD - 0.35, 1.0), PH, PH + 0.7, '#cac6be', true);
      solid.prism(rect(-hw - PW - 0.06, hw + PW + 0.06, -PD - 0.08, -PD + 0.1), top + 0.3, top + 0.75, '#c9a13b', true); // pita emas (sedikit menonjol: tidak berimpit)
      const fr = [G(-hw, -PD), G(hw, -PD)], bk = [G(-hw, 0.66), G(hw, 0.66)];
      col.addSeg(fr[0][0], fr[0][1], fr[1][0], fr[1][1], { h: PH + 0.7, lo: top, move: false, tag: 'wall' });
      col.addSeg(bk[0][0], bk[0][1], bk[1][0], bk[1][1], { h: PH + 0.7, lo: top, move: false, tag: 'wall' }); // atap lorong portal
      portals.push({ c: g.c, t: g.t, n: g.n, hw, PD });
      const uv = atlas.nameUV(g.g.osm);
      if (uv) { const py = king ? top + 1.2 : top + 0.5, ph = king ? 2.4 : 1.4; panel(nameB, G(-R * (hw - 0.2), -PD - 0.05), G(R * (hw - 0.2), -PD - 0.05), py, py + ph, '#ffffff', uv); }
      if (king) {
        // deret merlon di puncak portal
        const W = hw + PW + 0.35, nM = Math.round(W * 2 / 1.5);
        for (let i = 0; i < nM; i++) {
          const u = -W + (i + 0.5) * (2 * W / nM);
          solid.prism(rect(u - 0.3, u + 0.3, -PD - 0.3, -PD + 0.3), PH + 0.7, PH + 1.6, '#e2dfda', true);
        }
      }
    } else {
      const uv = g.g.ref && !numDone.has(g.g.ref) && atlas.numUV(g.g.ref);
      if (uv) { numDone.add(g.g.ref); panel(numB, G(-R, -0.7), G(R, -0.7), top + 0.4, top + 1.4, '#ffffff', uv); }
    }
  }

  // ---------- Atap (lempeng; disembunyikan saat tampak atas) ----------
  const roofTex = canvasTex(256, 256, (g, s) => { // ubin marmer rooftop (lebih putih dari pelataran)
    g.fillStyle = '#f5f4f0'; g.fillRect(0, 0, s, s);
    for (let i = 0; i < 500; i++) { g.fillStyle = `rgba(105,102,98,${Math.random() * 0.025})`; g.fillRect(Math.random() * s, Math.random() * s, 2 + Math.random() * 6, 1 + Math.random() * 2); }
    g.strokeStyle = 'rgba(150,145,130,.3)'; g.lineWidth = 2; for (let x = 0; x <= s; x += 64) { g.beginPath(); g.moveTo(x, 0); g.lineTo(x, s); g.stroke(); g.beginPath(); g.moveTo(0, x); g.lineTo(s, x); g.stroke(); }
  });
  roofTex.wrapS = roofTex.wrapT = THREE.RepeatWrapping;
  const roofTopB = Builder(4);
  outers.forEach((r, ri) => {
    const holes = [...inner.filter((h) => inPoly(h[0][0], h[0][1], r)), ...wells[ri]]; // mataf + sumur eskalator
    roofTopB.cap(r, holes, ROOF_TOP, '#ffffff', true, (x, z) => [x / 4, z / 4]); roofB.cap(r, holes, ROOF_Y, '#f4f3f1', false);
    holes.forEach((h) => roofB.sides(h, ROOF_Y, ROOF_TOP, '#d7d4cd', true));
  });
  // rooftop: pagar tepi mataf (visual di haram-kit, digeser 0,25 m ke luar) + menara
  inner.forEach((h) => {
    const ring = h.map((p) => { const d = Math.hypot(p[0], p[1]) || 1; return [p[0] + p[0] / d * 0.25, p[1] + p[1] / d * 0.25]; });
    ring.forEach((p, i) => { const q = ring[(i + 1) % ring.length]; colR.addSeg(p[0], p[1], q[0], q[1], { h: ROOF_TOP + 1.1, tag: 'wall' }); });
  });
  // Tepi atap menghadap mataf (+ pagar rendah): penghalang kamera saja; di bawah plafon terbuka.
  inner.forEach((h) => h.forEach((p, i) => { const q = h[(i + 1) % h.length]; col.addSeg(p[0], p[1], q[0], q[1], { h: ROOF_TOP + 1.2, lo: ROOF_Y, move: false, tag: 'roof' }); }));

  // ---------- Kit: fasad mataf, aula bertiang, Mas'a ----------
  const kb = buildKaaba(sim, scene, { rot: data.kaaba.rot, maqam: data.maqam });
  // Lorong lurus dari ujung bawah tiap eskalator ke Ka'bah: dibiarkan bebas tiang supaya jamaah tidak bingung arah.
  const aisles = [...ramps.map((r) => [wp(r, 0, RAMP_END + 1), [0, 0]]),
    ...gates.filter((g) => g.open && g.kind === 'main').map((g) => [[g.c[0] - g.n[0] * 3, g.c[1] - g.n[1] * 3], [0, 0]]),
    ...upRamps.map((r) => [wp(r, 0, r.RE), wp(r, 0, r.RE + 18)])]; // ruang bebas tiang di depan ujung bawah eskalator rooftop
  const ctx = { data, col, M, ROOF_Y, LIFT, archB, solid, roofB, out, kb, nearRamp, upAt, region, masaDist, REG, tex, Builder, aisles };
  const mataf = buildMataf(sim, ctx), hall = buildHall(sim, ctx), masa = buildMasa(sim, ctx);

  // ---------- Menara ----------
  const MIN_H = 46, mins = data.minarets, MH = LIFT + MIN_H;
  { // satu InstancedMesh: batang, 2 balkon, kepala, kerucut emas digabung (warna per verteks) -> 1 draw call
    const part = (g, y, color) => { g.translate(0, y, 0); return paint(g, color); };
    const geo = mergeGeos([
      part(new THREE.CylinderGeometry(1.2, 1.7, MH, 10), MH / 2, '#f1efec'),
      part(new THREE.CylinderGeometry(2.3, 2.3, 0.7, 12), LIFT + 18, '#cac6be'), part(new THREE.CylinderGeometry(2.3, 2.3, 0.7, 12), LIFT + 32, '#cac6be'),
      part(new THREE.CylinderGeometry(0.9, 1.1, 5, 10), MH + 2.5, '#fbfbf6'), part(new THREE.ConeGeometry(1.0, 4.5, 10), MH + 7.2, '#d4a93a'),
    ]);
    sim.instanced(geo, vmat(), mins, (o, p) => o.position.set(p[0], 0, p[1]));
  }
  mins.forEach((p) => { col.addCircle(p[0], p[1], 1.7); colR.addCircle(p[0], p[1], 1.7); });

  // ---------- Bangunan sekitar (siluet, berdiri di pelataran) ----------
  const bldB = Builder(4), bridgeB = Builder();
  const tones = ['#d9cfbd', '#cfc3ad', '#e3dccd', '#c8bba3', '#ddd3c2'];
  data.buildings.forEach((b, i) => {
    if (b.bridge) { bridgeB.prism(b.poly, ROOF_Y - 2.2, ROOF_Y - 0.6, '#ddd3bd', true); return; }
    // Gedung OSM bisa tumpang tindih / berbagi sisi: tiap kelas (i % 4) sedikit diciutkan & ditinggikan berbeda, dan
    // satu kelas satu warna, jadi tidak ada dua bidang berimpit dengan warna beda (tidak berkelap-kelip).
    const kc = i % 4, h = Math.max(1.2, b.h) + kc * 0.07, poly = kc ? offsetRing(b.poly, -0.07 * kc) : b.poly;
    bldB.prism(poly, -0.3, h, tones[kc]);
    col.addPoly(b.poly, { h: h + 0.2 });
  });

  // Clock Tower
  if (data.clockTower) {
    const [cx, cz] = data.clockTower.pos, Ht = data.clockTower.h, Wd = 24;
    const ct = new THREE.Group(); ct.position.set(cx, 0, cz); scene.add(ct);
    const sq = (w) => [[cx - w / 2, cz - w / 2], [cx + w / 2, cz - w / 2], [cx + w / 2, cz + w / 2], [cx - w / 2, cz + w / 2]];
    bldB.prism(sq(Wd), 0, Ht * 0.8, '#d2c6ae'); bldB.prism(sq(Wd * 0.8), Ht * 0.8, Ht * 0.9, '#e6dcc6');
    const spire = new THREE.Mesh(new THREE.ConeGeometry(Wd * 0.28, Ht * 0.12, 8), M.gold); spire.position.y = Ht * 0.96; ct.add(spire);
    const faceTex = canvasTex(256, 256, (g, s) => {
      g.fillStyle = '#f7f6ef'; g.beginPath(); g.arc(s / 2, s / 2, s * 0.46, 0, TAU); g.fill();
      g.strokeStyle = '#1f6b3a'; g.lineWidth = 14; g.stroke();
      g.strokeStyle = '#222'; g.lineWidth = 8; g.lineCap = 'round';
      g.beginPath(); g.moveTo(s / 2, s / 2); g.lineTo(s / 2, s * 0.18); g.moveTo(s / 2, s / 2); g.lineTo(s * 0.72, s * 0.56); g.stroke();
    });
    const faceMat = new THREE.MeshBasicMaterial({ map: faceTex, transparent: true });
    ct.add(new THREE.Mesh(mergeGeos([0, 1, 2, 3].map((i) => {
      const g = new THREE.PlaneGeometry(Wd * 0.85, Wd * 0.85), a = i * Math.PI / 2;
      g.rotateY(a); g.translate(Math.sin(a) * (Wd / 2 + 0.05), Ht * 0.72, Math.cos(a) * (Wd / 2 + 0.05)); return g;
    })), faceMat));
    out.labels.push({ text: 'Clock Tower (Abraj Al Bait)', pos: new THREE.Vector3(cx, Ht * 0.55, cz), far: 9999 });
  }

  scene.add(bldB.mesh(vmat())); scene.add(bridgeB.mesh(vmat())); // setelah Clock Tower (badannya ikut pembangun ini)

  // ---------- Tempat tambahan (Half Million, dll.) ----------
  for (const l of data.landmarks || []) {
    if (l.kind === 'cafe') { const hm = buildHalfMillion(sim, ctx, l); lmFloors.push(hm.floorAt); lmAts.push(hm.at); out.orderSpot = hm.spot; continue; }
    const [x, z] = l.pos, g = new THREE.Group(); g.position.set(x, 0, z); scene.add(g);
    const box = new THREE.Mesh(new THREE.BoxGeometry(7, 4.2, 6), mat(l.kind === 'cafe' ? '#6b4a33' : '#8a8f99')); box.position.y = 2.1; g.add(box);
    const awn = new THREE.Mesh(new THREE.BoxGeometry(7.6, 0.35, 2.4), mat(l.kind === 'cafe' ? '#c58b3c' : '#cfd3da')); awn.position.set(0, 3.2, 3.6); g.add(awn);
    col.addPoly([[x - 3.5, z - 3], [x + 3.5, z - 3], [x + 3.5, z + 3], [x - 3.5, z + 3]], { h: 4.4 });
    out.labels.push({ text: (l.kind === 'cafe' ? '☕ ' : '') + l.name, pos: new THREE.Vector3(x, 6, z), far: 140, cls: 'hl' });
  }

  // ---------- Ka'bah: penghalang & label ----------
  const kc = [[1, 1], [-1, 1], [-1, -1], [1, -1]].map(([sx, sz]) => { const v = kb.toWorld(sx * (KW + 0.9) / 2, 0, sz * (KD + 0.9) / 2); return [v.x, v.z]; });
  col.addPoly(kc, { h: 14, tag: 'kaaba' });
  const hp = kb.hijrPts.map((p) => { const v = kb.toWorld(p.x, 0, p.z); return [v.x, v.z]; });
  for (let i = 1; i < hp.length - 2; i++) col.addSeg(hp[i][0], hp[i][1], hp[i + 1][0], hp[i + 1][1], { h: 1.4 }); // celah masuk di kedua ujung
  col.addCircle(kb.MQ.x, kb.MQ.z, 1.4);
  out.kaaba = kb;
  out.labels.push({ text: "Ka'bah", pos: new THREE.Vector3(0, 16, 0), far: 9999, cls: 'hl' });
  out.labels.push({ text: 'Maqam Ibrahim', pos: kb.MQ.clone().setY(4.4), far: 40 });
  out.labels.push({ text: 'Hajar Aswad', pos: kb.C_HA.clone().setY(3.6), far: 35 });

  // ---------- Label pintu utama ----------
  for (const gt of gates) {
    const g = gt.g; if (!(g.main || g.name) || gt.synthetic) continue;
    const lab = g.name ? `${g.name}${g.ref ? ' · ' + g.ref : ''}` : `Pintu ${g.ref}`;
    const king = gt.kind === 'king', off = king ? 5.5 : gt.kind === 'main' ? 2.2 : 1.5;
    const y = king ? LIFT + 24.5 : gt.kind === 'main' ? WALL_TOP + 3 : 9;
    out.labels.push({ text: lab, pos: new THREE.Vector3(gt.c[0] + gt.n[0] * off, y, gt.c[1] + gt.n[1] * off), far: king ? 90 : g.main ? 60 : 35 });
  }

  // ---------- Selesaikan mesh gabungan ----------
  scene.add(facadeB.mesh(new THREE.MeshLambertMaterial({ map: tex.facade, vertexColors: true })));
  scene.add(solid.mesh(vmat()));
  scene.add(stepB.mesh(new THREE.MeshLambertMaterial({ map: tex.stepTex, vertexColors: true })));
  scene.add(archB.mesh(new THREE.MeshLambertMaterial({ map: tex.arch, vertexColors: true, alphaTest: 0.5, side: THREE.DoubleSide })));
  scene.add(nameB.mesh(new THREE.MeshLambertMaterial({ map: atlas.nameTex, vertexColors: true, side: THREE.DoubleSide })));
  scene.add(numB.mesh(new THREE.MeshLambertMaterial({ map: atlas.numTex, vertexColors: true, side: THREE.DoubleSide })));
  const roof = roofB.mesh(vmat()); scene.add(roof); out.roofs.push(roof);
  const roofTop = roofTopB.mesh(new THREE.MeshLambertMaterial({ map: roofTex, vertexColors: true })); scene.add(roofTop); out.roofs.push(roofTop);
  scene.add(tileB.mesh(new THREE.MeshLambertMaterial({ map: tileTex, vertexColors: true })));

  // Batas dunia: lingkaran tak terlihat
  const RB = GR - 20, NB = 64;
  for (let i = 0; i < NB; i++) { const a0 = i / NB * TAU, a1 = (i + 1) / NB * TAU; col.addSeg(Math.cos(a0) * RB, Math.sin(a0) * RB, Math.cos(a1) * RB, Math.sin(a1) * RB, { h: 0 }); }
  out.boundR = RB;
  // Di lorong portal (bagian luar gerbang yang beratap)?
  const portalAt = (x, z) => portals.some((q) => {
    const dx = x - q.c[0], dz = z - q.c[1], s = -(dx * q.n[0] + dz * q.n[1]), u = dx * q.t[0] + dz * q.t[1];
    return Math.abs(u) <= q.hw && s >= -q.PD && s <= 0.7;
  });
  // Titik "Lompat ke" di rooftop: tepi mataf (memandang Ka'bah) di selatan & utara, dan di atas tiap eskalator rooftop.
  const roofSpots = [];
  for (const [name, ang] of [['Atap: tepi mataf selatan', Math.PI / 2], ['Atap: tepi mataf utara', -Math.PI / 2], ['Atap: tepi mataf barat', Math.PI]]) {
    const dx = Math.cos(ang), dz = Math.sin(ang), rr = Math.min(...inner.map((h) => { let best = Infinity; for (let i = 0; i < h.length; i++) { const a = h[i], b = h[(i + 1) % h.length], sx = b[0] - a[0], sz = b[1] - a[1], den = dx * sz - dz * sx; if (Math.abs(den) < 1e-9) continue; const t = (a[0] * sz - a[1] * sx) / den, v = (a[0] * dz - a[1] * dx) / den; if (t > 0 && v >= 0 && v <= 1) best = Math.min(best, t); } return best; }));
    const x = dx * (rr + 3), z = dz * (rr + 3);
    if (region.at(x, z) === REG.HALL) roofSpots.push({ name, x, z, face: Math.atan2(-x, -z) });
  }
  Object.assign(out, {
    floorY, region, ramps, gates, masaDist, nearRamp, portalAt, colR, upRamps, upAt, upWalk, landmarkAt: (x, z) => lmAts.some((f) => f(x, z)), terraces, terraceAt, roofSpots, ROOF_TOP, zamzam: mataf.zamzam,
    stats: { gates: gates.length, closed: gates.filter((g) => !g.open).map((g) => g.g.ref || '?'), ramps: ramps.length, terraces: terraces.length, upRamps: upRamps.map((r) => (r.g.g.ref || r.g.g.name || '?') + (r.tight ? '' : '*')), hallPillars: hall.pillars.length, masaPillars: masa.pillars.length, facade: mataf.cols.length },
  });
  return out;
}
