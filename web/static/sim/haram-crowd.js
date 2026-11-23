// Kerumunan open world (Tahap 3): jamaah tawaf di mataf, sa'i di Mas'a, keluar-masuk pintu (pelataran -> pintu; di
// gerbang King lewat teras, bordes & eskalator turun -> lorong -> mataf), berjalan-jalan di pelataran, dan duduk berbaris menghadap Ka'bah di aula.
// Tiap lintasan dicek dulu dengan peta tabrakan (dinding, tiang, gedung, pagar eskalator), jadi tidak ada yang
// menembus tembok. Yang jauh / di belakang kamera tidak digambar (createCrowd + cull).
import { createCrowd, SKIN_TONES } from './common.js?v=49';
import { TAU, rayRing } from './haram-geom.js?v=49';

export const CROWD_PLAN = { tawaf: 48, sai: 22, walk: 30, plaza: 14, sit: 26 };
const FOOT = 0.07; // telapak sedikit di atas lantai (seperti pemain)

export function buildCrowd(sim, { world, data, col, LIFT, RAMP_END, REG }) {
  let seed = 20261105; const rnd = () => ((seed = (seed * 16807) % 2147483647) / 2147483647); // berbenih: sama tiap muat
  const pick = (a) => a[Math.floor(rnd() * a.length)];
  const KINDS = [
    { gender: 'm', opts: { shoulderBare: true } },                                  // 0 laki-laki ber-idhtiba' (tawaf)
    { gender: 'm', opts: { shoulderBare: false } },                                 // 1 laki-laki
    { gender: 'f', opts: { cloth: '#f4efe6', veil: '#ece6dc' } },                   // 2 perempuan putih
    { gender: 'f', opts: { cloth: '#2f2f33', veil: '#26262a', shoe: '#2a2a2a' } },  // 3 perempuan hitam
  ];
  const kindFor = (tawaf) => { const r = rnd(); return r < 0.62 ? (tawaf ? 0 : 1) : r < 0.82 ? 2 : 3; };
  const people = [];
  const inner = data.mosque.inner[0];
  const NA = 360, rm = new Float32Array(NA);
  for (let i = 0; i < NA; i++) { const a = i / NA * TAU; rm[i] = rayRing(Math.cos(a), Math.sin(a), inner); }
  const rmAt = (x, z) => { const a = ((Math.atan2(z, x) / TAU) % 1 + 1) % 1; return rm[Math.floor(a * NA) % NA]; };
  const rmMin = Math.min(...rm);
  // titik bebas (tidak di dalam tiang/menara/dll.) & ruas lurus bebas (tidak memotong dinding/gedung/pagar/tiang)
  const pointFree = (x, z, pad = 0.7) => !col.circleOnLine(x, z, x, z, pad);
  const segFree = (ax, az, bx, bz, pad = 0.55) => {
    if (col.circleOnLine(ax, az, bx, bz, pad)) return false;
    const dx = bx - ax, dz = bz - az, L = Math.hypot(dx, dz) || 1, ox = -dz / L * pad, oz = dx / L * pad;
    for (const s of [0, 1, -1]) for (const c of col.crossings(ax + ox * s, az + oz * s, bx + ox * s, bz + oz * s, buf)) if (c.h > 0 && c.lo < 3 + LIFT) return false;
    return true;
  };
  const buf = [];

  // ---------- Tawaf: berputar berlawanan arah jarum jam (dilihat dari atas), Ka'bah di kiri ----------
  const R0 = 16.5, R1 = Math.min(39, rmMin - 2.5); // di luar Hijr & Maqam, di dalam tepi mataf
  for (let i = 0; i < CROWD_PLAN.tawaf; i++) {
    const v = 0.6 + rnd() * 0.6, r = R0 + rnd() * (R1 - R0);
    people.push({ role: 'tawaf', kind: kindFor(true), a: rnd() * TAU, r, w: v / r, speed: v * 1.6, y: FOOT });
  }

  // ---------- Sa'i: lajur searah, berbalik di Safa & Marwah; laki-laki berlari kecil di zona lampu hijau ----------
  const S = data.safa, M = data.marwah, SL = Math.hypot(M[0] - S[0], M[1] - S[1]);
  const st = [(M[0] - S[0]) / SL, (M[1] - S[1]) / SL], sn = [-st[1], st[0]];
  const SP = (f, o) => [S[0] + st[0] * SL * f + sn[0] * o, S[1] + st[1] * SL * f + sn[1] * o];
  const F0 = 0.05, F1 = 0.95, G0 = 0.3, G1 = 0.44;
  const laneOK = (o) => { for (let f = F0; f <= F1; f += 2 / SL) { const [x, z] = SP(f, o); if (world.region.at(x, z) !== REG.HALL || world.nearRamp(x, z, 0.8) || !pointFree(x, z, 0.8)) return false; } return true; };
  const lanes = { 1: [], [-1]: [] };
  for (let o = 2.4; o <= 9.6; o += 0.6) { if (laneOK(-o)) lanes[1].push(-o); if (laneOK(o)) lanes[-1].push(o); } // ke Marwah: sisi barat; ke Safa: sisi timur
  if (lanes[1].length && lanes[-1].length) for (let i = 0; i < CROWD_PLAN.sai; i++) {
    const dir = rnd() < 0.5 ? 1 : -1;
    people.push({ role: 'sai', kind: kindFor(false), f: F0 + rnd() * (F1 - F0), dir, o: pick(lanes[dir]), v: 1.1 + rnd() * 0.4, speed: 0, y: FOOT });
  }

  // ---------- Keluar-masuk pintu: pelataran (tangga teras King) -> ambang -> (eskalator turun) -> lorong -> mataf ----------
  const routes = [];
  const gl = world.gates.filter((g) => g.open && !g.synthetic).sort((a, b) => ({ king: 3, main: 2, side: 1 }[b.kind] - { king: 3, main: 2, side: 1 }[a.kind]) || b.hw - a.hw);
  for (const g of gl) {
    if (routes.length >= 12) break;
    const ramp = g.kind === 'king' ? world.ramps.find((r) => r.gates.includes(g)) : null;
    const tr = world.terraces.find((t) => t.g === g);
    const at = (u, s) => [g.c[0] + g.t[0] * u - g.n[0] * s, g.c[1] + g.t[1] * u - g.n[1] * s];
    for (let tries = 0; tries < 6; tries++) {
      const u = (rnd() * 2 - 1) * Math.max(0, g.hw - 0.8), far = (tr ? tr.D + tr.S : 0) + 8 + rnd() * 10;
      const P0 = at(u, -far), P1 = at(u, -1.2);
      const P2 = ramp ? [ramp.c[0] + ramp.t[0] * u - ramp.n[0] * (RAMP_END + 1.5), ramp.c[1] + ramp.t[1] * u - ramp.n[1] * (RAMP_END + 1.5)] : at(u, 3);
      const d = Math.hypot(P2[0], P2[1]) || 1, R3 = rmAt(P2[0], P2[1]) - 3, P3 = [P2[0] / d * R3, P2[1] / d * R3];
      if (world.region.at(P0[0], P0[1]) !== REG.OUT || world.upAt(P0[0], P0[1], 1) || world.landmarkAt(P0[0], P0[1])) continue;
      if (!segFree(...P0, ...P1) || !segFree(...P2, ...P3) || d <= R3 + 2) continue;
      routes.push([P0, P1, P2, P3]); break;
    }
  }
  const polyLen = (pts) => { const L = [0]; for (let i = 1; i < pts.length; i++) L.push(L[i - 1] + Math.hypot(pts[i][0] - pts[i - 1][0], pts[i][1] - pts[i - 1][1])); return L; };
  routes.forEach((pts) => (pts.L = polyLen(pts)));
  if (routes.length) for (let i = 0; i < CROWD_PLAN.walk; i++) {
    const rt = routes[i % routes.length];
    people.push({ role: 'walk', kind: kindFor(false), rt, d: rnd() * rt.L[3], dir: rnd() < 0.6 ? 1 : -1, v: 1.2 + rnd() * 0.5, wait: 0, speed: 0, y: FOOT });
  }

  // ---------- Pelataran: berjalan-jalan dekat pintu, berhenti sejenak, pilih tujuan baru ----------
  const plazaOK = (x, z) => world.region.at(x, z) === REG.OUT && !world.nearRamp(x, z, 1) && !world.portalAt(x, z) && !world.upAt(x, z, 1) && !world.landmarkAt(x, z) && pointFree(x, z, 1)
    && !data.buildings.some((b) => !b.bridge && Math.abs(b.poly[0][0] - x) < 120 && inPoly(x, z, b.poly)) && Math.hypot(x, z) < world.boundR - 30;
  const gatesOpen = world.gates.filter((g) => g.open && !g.synthetic);
  const newTarget = (p) => {
    for (let k = 0; k < 12; k++) {
      const a = rnd() * TAU, d = 6 + rnd() * 20, x = p.x + Math.cos(a) * d, z = p.z + Math.sin(a) * d;
      if (plazaOK(x, z) && segFree(p.x, p.z, x, z, 0.6)) { p.tx = x; p.tz = z; return true; }
    }
    return false;
  };
  for (let i = 0, guard = 0; i < CROWD_PLAN.plaza && guard < 400; guard++) {
    const g = pick(gatesOpen), k = 8 + rnd() * 30, lat = (rnd() - 0.5) * 30;
    const x = g.c[0] + g.n[0] * k + g.t[0] * lat, z = g.c[1] + g.n[1] * k + g.t[1] * lat;
    if (!plazaOK(x, z)) continue;
    const p = { role: 'plaza', kind: kindFor(false), x, z, v: 1.0 + rnd() * 0.4, wait: rnd() * 3, speed: 0, y: world.floorY(x, z) + FOOT, yaw: rnd() * TAU };
    if (!newTarget(p)) continue;
    people.push(p); i++;
  }

  // ---------- Duduk berbaris menghadap Ka'bah di aula, dekat fasad mataf ----------
  const aisleD = (x, z) => Math.min(...world.ramps.map((r) => {
    const b = [r.c[0] - r.n[0] * (RAMP_END + 1), r.c[1] - r.n[1] * (RAMP_END + 1)], L2 = b[0] * b[0] + b[1] * b[1] || 1;
    const u = Math.max(0, Math.min(1, (x * b[0] + z * b[1]) / L2)); return Math.hypot(x - b[0] * u, z - b[1] * u);
  }));
  const sitOK = (x, z) => world.region.at(x, z) === REG.HALL && !world.nearRamp(x, z, 2) && !world.upAt(x, z, 1.5) && pointFree(x, z, 0.9) && aisleD(x, z) > 4 && world.masaDist(x, z) > 13;
  for (let i = 0, guard = 0; i < CROWD_PLAN.sit && guard < 300; guard++) {
    const a = rnd() * TAU, ca = Math.cos(a), sa = Math.sin(a), r = rm[Math.floor(a / TAU * NA) % NA] + 4 + rnd() * 6;
    const tx = -sa, tz = ca, n = 3 + Math.floor(rnd() * 4); // satu saf 3-6 orang
    for (let j = 0; j < n && i < CROWD_PLAN.sit; j++) {
      const x = ca * r + tx * (j - n / 2) * 1.15, z = sa * r + tz * (j - n / 2) * 1.15;
      if (!sitOK(x, z)) continue;
      people.push({ role: 'sit', kind: kindFor(false), x, z, yaw: Math.atan2(-x, -z), pose: 'sit', speed: 0, y: FOOT }); i++;
    }
  }

  // ---------- Buat instanced crowd ----------
  const counts = KINDS.map((_, i) => people.filter((p) => p.kind === i).length);
  const crowd = createCrowd(sim.scene, KINDS.map((k, i) => ({ ...k, count: Math.max(1, counts[i]) })));
  people.forEach((p, i) => { p.scale = 0.8 + rnd() * 0.12; p.skin = SKIN_TONES[i % SKIN_TONES.length]; p.x = p.x ?? 0; p.z = p.z ?? 0; p.yaw = p.yaw ?? 0; crowd.add(p); });

  const at = (pts, d) => { // titik & arah pada polyline di jarak d
    const L = pts.L; let i = 1; while (i < pts.length - 1 && L[i] < d) i++;
    const a = pts[i - 1], b = pts[i], t = (d - L[i - 1]) / ((L[i] - L[i - 1]) || 1);
    return [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t, b[0] - a[0], b[1] - a[1]];
  };
  function step(p, dt) {
    if (p.role === 'tawaf') {
      p.a += p.w * dt; p.x = Math.cos(p.a) * p.r; p.z = -Math.sin(p.a) * p.r; p.yaw = Math.atan2(-Math.sin(p.a), -Math.cos(p.a));
    } else if (p.role === 'sai') {
      const run = p.kind <= 1 && p.f > G0 && p.f < G1 ? 1.6 : 1, v = p.v * run;
      p.f += p.dir * v * dt / SL;
      if (p.f > F1 || p.f < F0) { p.f = Math.max(F0, Math.min(F1, p.f)); p.dir = -p.dir; p.o = pick(lanes[p.dir]); }
      [p.x, p.z] = SP(p.f, p.o); p.yaw = Math.atan2(st[0] * p.dir, st[1] * p.dir); p.speed = v * 1.6;
    } else if (p.role === 'walk') {
      if (p.wait > 0) { p.wait -= dt; p.speed = 0; return; }
      p.d += p.dir * p.v * dt;
      const Lend = p.rt.L[3];
      if (p.d >= Lend || p.d <= 0) { p.d = Math.max(0, Math.min(Lend, p.d)); p.dir = -p.dir; p.wait = 1 + rnd() * 4; }
      const [x, z, dx, dz] = at(p.rt, p.d);
      p.x = x; p.z = z; p.yaw = Math.atan2(dx * p.dir, dz * p.dir); p.speed = p.v * 1.6; p.y = world.floorY(x, z) + FOOT;
    } else if (p.role === 'plaza') {
      if (p.wait > 0) { p.wait -= dt; p.speed = 0; if (p.wait <= 0 && !newTarget(p)) p.wait = 2; return; }
      const dx = p.tx - p.x, dz = p.tz - p.z, d = Math.hypot(dx, dz);
      if (d < 0.3) { p.wait = 1 + rnd() * 4; p.speed = 0; return; }
      const s = Math.min(d, p.v * dt); p.x += dx / d * s; p.z += dz / d * s; p.yaw = Math.atan2(dx, dz); p.speed = p.v * 1.6; p.y = world.floorY(p.x, p.z) + FOOT;
    }
  }
  // Langkah simulasi + gambar: hanya yang dekat pemain (< 80 m) dan tidak jauh di belakang kamera.
  const DRAW = 80;
  function update(dt, cam, P) {
    const cx = cam.position.x, cz = cam.position.z, fx = P.x - cx, fz = P.z - cz, fl = Math.hypot(fx, fz) || 1;
    for (const p of people) step(p, dt);
    crowd.update(dt, (p) => {
      const dx = p.x - P.x, dz = p.z - P.z; if (dx * dx + dz * dz > DRAW * DRAW) return false;
      return ((p.x - cx) * fx + (p.z - cz) * fz) / fl > -8; // di depan kamera (dengan kelonggaran)
    });
  }
  return { people, crowd, update, routes, lanes, thin: (f) => crowd.thin(f) };
}

function inPoly(x, z, ring) {
  let c = false;
  for (let i = 0, j = ring.length - 1; i < ring.length; j = i++) {
    const [x1, z1] = ring[i], [x2, z2] = ring[j];
    if ((z1 > z) !== (z2 > z) && x < (x2 - x1) * (z - z1) / (z2 - z1) + x1) c = !c;
  }
  return c;
}
