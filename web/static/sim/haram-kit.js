// Kit bangunan Masjidil Haram (Tahap 2): tekstur fasad & lengkung, papan nama pintu, tiang aula persegi besar, fasad
// mataf dua tingkat, lorong Mas'a (tiang, plafon, lampu hijau, bukit Safa/Marwah), garis start tawaf, dispenser Zamzam.
// Semua dibuat dari kode. Geometri statis masuk ke Builder bersama (ctx.solid / ctx.arch) milik haram-world.js.
import * as THREE from 'three';
import { TAU, archPath, mergeGeos, sectorInstanced, rayRing, edgePoints } from './haram-geom.js?v=49';

export const PANEL_W = 8; // lebar satu ulangan tekstur fasad (m)

// ---------- Tekstur ----------
export function makeTextures(sim, { ROOF_Y, WALL_TOP }) {
  const { canvasTex } = sim;
  const noise = (g, w, h, n, a) => { for (let i = 0; i < n; i++) { g.fillStyle = `rgba(105,102,98,${Math.random() * a})`; g.fillRect(Math.random() * w, Math.random() * h, 2 + Math.random() * 7, 1 + Math.random() * 2); } };
  // Fasad: satu panel PANEL_W x WALL_TOP. v = y / WALL_TOP (bawah = lantai masjid = pelataran).
  const facade = canvasTex(256, 544, (g, w, h) => {
    const PX = w / PANEL_W, PY = h / WALL_TOP, Y = (m) => h - m * PY;
    g.fillStyle = '#ecebe7'; g.fillRect(0, 0, w, h); noise(g, w, h, 500, 0.05);
    g.fillStyle = '#dfdcd6'; g.fillRect(0, Y(1.1), w, 1.1 * PY);                    // kaki dinding
    g.fillStyle = 'rgba(110,106,100,.28)'; g.fillRect(0, Y(1.1), w, 3);
    g.fillStyle = '#e3e0db'; g.fillRect(0, 0, 0.5 * PX, h); g.fillRect(w - 0.5 * PX, 0, 0.5 * PX, h); // pilaster
    // Satu jendela lengkung tinggi per panel, dari lantai sampai dekat atap (bukan tiga tingkat); teras King menutup
    // bagian bawahnya. Lengkung runcing setinggi ±0,9 lebar; palang mendatar tiap 2,6 m supaya terbaca sebagai kaca.
    const win = (y0, y1, wm, glass) => {
      const x0 = w / 2 - wm / 2 * PX, ww = wm * PX, top = Y(y1), hh = (y1 - y0) * PY, sp = hh - ww * 0.9; // sp = bagian lurus
      g.fillStyle = '#bfbaaf'; g.beginPath(); archPath(g, x0 - 6, top - 6, ww + 12, hh + 6, sp + 6); g.fill();
      g.fillStyle = glass; g.beginPath(); archPath(g, x0, top, ww, hh, sp); g.fill();
      g.strokeStyle = 'rgba(232,230,224,.55)'; g.lineWidth = 2;
      g.beginPath(); g.moveTo(w / 2, top + 8); g.lineTo(w / 2, top + hh);
      for (let yy = y0 + 2.6; yy < y1 - 2.4; yy += 2.6) { g.moveTo(x0, Y(yy)); g.lineTo(x0 + ww, Y(yy)); }
      g.stroke();
      g.strokeStyle = '#d4a93a'; g.lineWidth = 1.5; g.beginPath(); archPath(g, x0 - 2, top - 2, ww + 4, hh + 2, sp + 2, false); g.stroke();
    };
    win(1.6, 14.2, 2.8, '#46515c');
    g.fillStyle = '#f3f2ef'; g.fillRect(0, 0, w, Y(ROOF_Y));                        // parapet atap
    g.fillStyle = '#cac6be'; g.fillRect(0, Y(ROOF_Y) - 3, w, 4);
    g.fillStyle = '#e0ddd6'; for (let x = 8; x < w; x += 32) { g.beginPath(); archPath(g, x, 6, 16, Y(ROOF_Y) - 12, 8); g.fill(); }
  });
  facade.wrapS = facade.wrapT = THREE.RepeatWrapping;
  // Lengkung runcing ber-alpha (bagian bawah lengkung tembus = celah antar tiang, bukan efek tembus pandang).
  const arch = canvasTex(128, 128, (g, s) => {
    g.fillStyle = '#e6e4df'; g.fillRect(0, 0, s, s); noise(g, s, s, 80, 0.06);
    const A = [s * 0.06, s * 0.14, s * 0.88, s * 0.86, s * 0.42];
    g.globalCompositeOperation = 'destination-out'; g.fillStyle = '#000'; g.beginPath(); archPath(g, ...A); g.fill(); // pemotong harus pekat
    g.globalCompositeOperation = 'source-over';
    g.strokeStyle = '#cfc9bf'; g.lineWidth = 10; g.beginPath(); archPath(g, ...A, false); g.stroke();
    g.strokeStyle = '#d4a93a'; g.lineWidth = 2; g.beginPath(); archPath(g, A[0] - 6, A[1] - 6, A[2] + 12, A[3] + 6, A[4], false); g.stroke();
    g.fillStyle = '#c6c1b8'; for (const x of [s * 0.06, s * 0.94]) { g.beginPath(); g.arc(x, s * 0.1, 5, 0, TAU); g.fill(); }
  });
  arch.anisotropy = 2;
  // Plafon Mas'a (pola kotak seperti simulasi sa'i)
  const ceil = canvasTex(256, 256, (g, s) => {
    g.fillStyle = '#f4f3f1'; g.fillRect(0, 0, s, s); g.strokeStyle = '#d9c79a'; g.lineWidth = 6; g.strokeRect(10, 10, s - 20, s - 20);
    g.lineWidth = 2; g.strokeRect(40, 40, s - 80, s - 80); g.fillStyle = '#e2cf9f'; g.beginPath(); g.arc(s / 2, s / 2, 18, 0, TAU); g.fill();
  });
  ceil.wrapS = ceil.wrapT = THREE.RepeatWrapping;
  // Anak tangga / eskalator (garis anak tangga melintang)
  const stepTex = canvasTex(64, 64, (g, s) => {
    g.fillStyle = '#6e757c'; g.fillRect(0, 0, s, s);
    g.fillStyle = '#4f555b'; for (let y = 0; y < s; y += 8) g.fillRect(0, y, s, 3);
    g.fillStyle = '#e0b43a'; g.fillRect(0, 0, 3, s); g.fillRect(s - 3, 0, 3, s);
  });
  stepTex.wrapS = stepTex.wrapT = THREE.RepeatWrapping;
  return { facade, arch, ceil, stepTex };
}

// Papan nama pintu: atlas kanvas. names: [{key, ar, en, ref}] -> sel 512x128; nums: ['1','2',...] -> sel 128x64.
const GATE_AR = {
  '1': 'باب الملك عبد العزيز', '5': 'باب أجياد', '25': 'باب المروة', '30': 'باب الفتح', '35': 'باب القدس',
  '40': 'باب العمرة', '79': 'باب الملك فهد', '100': 'باب الملك عبد الله',
};
export function plaqueAtlas(sim, names, nums) {
  const { canvasTex } = sim;
  const nameCell = new Map(), numCell = new Map();
  const nameTex = canvasTex(1024, 128 * Math.max(1, Math.ceil(names.length / 2)), (g) => {
    names.forEach((n, i) => {
      const x = (i % 2) * 512, y = Math.floor(i / 2) * 128; nameCell.set(n.key, i);
      g.fillStyle = '#0f3d2a'; g.fillRect(x, y, 512, 128);
      g.strokeStyle = '#d4a93a'; g.lineWidth = 6; g.strokeRect(x + 6, y + 6, 500, 116);
      g.lineWidth = 2; g.strokeRect(x + 14, y + 14, 484, 100);
      const ar = GATE_AR[n.ref] || n.ar || '';
      g.textAlign = 'center'; g.textBaseline = 'middle';
      if (n.ref) {
        g.fillStyle = '#d4a93a'; g.beginPath(); g.arc(x + 62, y + 64, 36, 0, TAU); g.fill();
        g.fillStyle = '#0f3d2a'; g.font = 'bold 40px system-ui, sans-serif'; g.fillText(n.ref, x + 62, y + 66);
      }
      const cx = x + (n.ref ? 290 : 256);
      g.fillStyle = '#f5df9c';
      if (ar) { g.font = 'bold 42px "Geeza Pro", "Noto Naskh Arabic", "Amiri", serif'; g.fillText(ar, cx, y + 46, 380); }
      g.font = '600 24px system-ui, sans-serif'; g.fillStyle = '#efede8'; g.fillText(n.en.toUpperCase(), cx, y + (ar ? 96 : 64), 390);
    });
  });
  const numTex = canvasTex(1024, 64 * Math.max(1, Math.ceil(nums.length / 8)), (g) => {
    nums.forEach((t, i) => {
      const x = (i % 8) * 128, y = Math.floor(i / 8) * 64; numCell.set(t, i);
      g.fillStyle = '#0f3d2a'; g.fillRect(x, y, 128, 64); g.strokeStyle = '#d4a93a'; g.lineWidth = 4; g.strokeRect(x + 4, y + 4, 120, 56);
      g.fillStyle = '#ffffff'; g.font = 'bold 34px system-ui, sans-serif'; g.textAlign = 'center'; g.textBaseline = 'middle'; g.fillText(t, x + 64, y + 34);
    });
  });
  const rows = (n, per) => Math.max(1, Math.ceil(n / per));
  return {
    nameTex, numTex,
    nameUV(key) { const i = nameCell.get(key), R = rows(names.length, 2); if (i == null) return null; const u0 = (i % 2) / 2, v1 = 1 - Math.floor(i / 2) / R; return [[u0, v1 - 1 / R], [u0 + 0.5, v1 - 1 / R], [u0 + 0.5, v1], [u0, v1]]; },
    numUV(t) { const i = numCell.get(t), R = rows(nums.length, 8); if (i == null) return null; const u0 = (i % 8) / 8, v1 = 1 - Math.floor(i / 8) / R; return [[u0, v1 - 1 / R], [u0 + 1 / 8, v1 - 1 / R], [u0 + 1 / 8, v1], [u0, v1]]; },
  };
}

// Panel vertikal (lengkung / papan) dari titik p ke q pada ketinggian y0..y1. uvs: 4 titik [kiri-bawah, kanan-bawah, kanan-atas, kiri-atas].
export function panel(B, p, q, y0, y1, color, uvs = [[0, 0], [1, 0], [1, 1], [0, 1]], nSign = 1) {
  const dx = q[0] - p[0], dz = q[1] - p[1], L = Math.hypot(dx, dz) || 1, V = B.V;
  B.quad(V(p[0], y0, p[1]), V(q[0], y0, q[1]), V(q[0], y1, q[1]), V(p[0], y1, p[1]), V(nSign * dz / L, 0, -nSign * dx / L), color, uvs);
}

// Tiang standar (batang + kepala + alas), dasar di y = 0.
export function pillarGeo(H, r0 = 0.36, r1 = 0.44) {
  const shaft = new THREE.CylinderGeometry(r0, r1, H, 8, 1, true); shaft.translate(0, H / 2, 0);
  const cap = new THREE.BoxGeometry(r1 * 2.5, 0.55, r1 * 2.5); cap.translate(0, H - 0.28, 0);
  const base = new THREE.BoxGeometry(r1 * 2.3, 0.6, r1 * 2.3); base.translate(0, 0.3, 0);
  return mergeGeos([shaft, cap, base]);
}

// Tiang persegi besar (aula & Mas'a): batang + kepala + alas, dasar di y = 0.
export function squarePillarGeo(H, w = 1.6) {
  const shaft = new THREE.BoxGeometry(w, H, w); shaft.translate(0, H / 2, 0);
  const cap = new THREE.BoxGeometry(w * 1.35, 0.8, w * 1.35); cap.translate(0, H - 0.4, 0);
  const base = new THREE.BoxGeometry(w * 1.25, 0.9, w * 1.25); base.translate(0, 0.45, 0);
  return mergeGeos([shaft, cap, base]);
}

// ---------- Fasad mataf: deret tiang dua tingkat + lengkung, atap datar berpagar ----------
export function buildMataf(sim, ctx) {
  const { data, col, M, ROOF_Y, archB, solid, out, kb, nearRamp } = ctx;
  const inner = data.mosque.inner;
  const SP = 9.2, PUSH = 1.3, PR = 0.6; // jarak tiang fasad 9,2 m (separuh jumlah sebelumnya)
  const push = (x, z, d) => { const r = Math.hypot(x, z) || 1; return [x + x / r * d, z + z / r * d]; };
  const cols = [];
  inner.forEach((h) => edgePoints(h, SP).forEach((p) => { const [x, z] = push(p.x, p.z, PUSH); cols.push({ ...p, x, z }); }));
  const keep = cols.filter((c) => !nearRamp(c.x, c.z, 0.8));
  sectorInstanced(sim, pillarGeo(ROOF_Y, PR * 0.85, PR), M.pillar, keep, (o, p) => o.position.set(p.x, 0, p.z), 1);
  keep.forEach((p) => col.addCircle(p.x, p.z, PR));
  // Lengkung di antara dua tiang pada sisi yang sama: tingkat bawah & atas
  for (let i = 0; i < cols.length; i++) {
    const a = cols[i], b = cols[(i + 1) % cols.length];
    if (a.edge !== b.edge || a.k === a.n - 1 || nearRamp(a.x, a.z, 0.8) || nearRamp(b.x, b.z, 0.8)) continue;
    const L = Math.hypot(b.x - a.x, b.z - a.z), ux = (b.x - a.x) / L, uz = (b.z - a.z) / L, e = PR;
    const p = [a.x + ux * e, a.z + uz * e], q = [b.x - ux * e, b.z - uz * e];
    panel(archB, p, q, 3.6, 7.5, '#ffffff');
    panel(archB, p, q, ROOF_Y - 3.4, ROOF_Y, '#ffffff'); // pita lengkung atas: di atas batas tinggi kamera
  }
  // Atap datar (rooftop) tanpa kubah: hanya pagar rendah di tepi atap yang menghadap mataf
  inner.forEach((h) => {
    const ring = h.map((p) => push(p[0], p[1], 0.25));
    for (let i = 0; i < ring.length; i++) ctx.roofB.slab(ring[i], ring[(i + 1) % ring.length], 0.45, ROOF_Y + 0.8, ROOF_Y + 1.9, '#e7e5e1');
  });

  // Garis start tawaf (sejajar Hajar Aswad) + lampu hijau di tepi mataf
  const ha = kb.C_HA, hd = Math.hypot(ha.x, ha.z), hx = ha.x / hd, hz = ha.z / hd;
  const rE = Math.min(...inner.map((h) => rayRing(hx, hz, h)));
  if (isFinite(rE)) {
    const r0 = hd + 2.5, r1 = rE - 0.8, A = Math.atan2(-hz, hx);
    const line = new THREE.Mesh(new THREE.PlaneGeometry(r1 - r0, 0.35), new THREE.MeshBasicMaterial({ color: '#16a34a', transparent: true, opacity: 0.5, depthWrite: false }));
    line.rotation.set(-Math.PI / 2, 0, A); line.position.set(hx * (r0 + r1) / 2, 0.08, hz * (r0 + r1) / 2); sim.scene.add(line);
    const lx = hx * (rE - 1.4), lz = hz * (rE - 1.4);
    const pole = new THREE.Mesh(new THREE.CylinderGeometry(0.12, 0.14, 4.2, 8), M.silver); pole.position.set(lx, 2.1, lz); sim.scene.add(pole);
    const head = new THREE.Mesh(new THREE.BoxGeometry(0.9, 0.6, 0.9), new THREE.MeshBasicMaterial({ color: '#16a34a' })); head.position.set(lx, 4.4, lz); sim.scene.add(head);
    col.addCircle(lx, lz, 0.3);
    out.labels.push({ text: 'Garis awal tawaf', pos: new THREE.Vector3(lx, 5.4, lz), far: 45 });
  }
  out.labels.push({ text: 'Multazam', pos: kb.toWorld(5.6, 3.2, 4.6), far: 30 });
  out.labels.push({ text: 'Rukun Yamani', pos: kb.C_YM.clone().setY(3.2), far: 30 });
  out.labels.push({ text: 'Hijr Ismail', pos: kb.HIJR_MID.clone().setY(2.4), far: 35 });

  // Dispenser Zamzam di sisi aula, di balik tiang fasad
  const zz = [];
  keep.forEach((c, i) => { if (i % 2 === 1) { const [x, z] = push(c.x, c.z, 1.5); if (!nearRamp(x, z, 1)) zz.push({ x, z }); } });
  const body = mergeGeos([
    (() => { const g = new THREE.CylinderGeometry(0.42, 0.42, 1.0, 10); g.translate(0, 0.5, 0); return g; })(),
  ]);
  sectorInstanced(sim, body, sim.mat('#f4f6f8'), zz, (o, p) => o.position.set(p.x, 0, p.z), 1);
  sectorInstanced(sim, new THREE.CylinderGeometry(0.44, 0.44, 0.22, 10), sim.mat('#2f6fb3'), zz, (o, p) => o.position.set(p.x, 1.1, p.z), 1);
  zz.forEach((p) => col.addCircle(p.x, p.z, 0.5));
  zz.slice(0, 40).forEach((p, i) => { if (i % 8 === 0) out.labels.push({ text: 'Air Zamzam', pos: new THREE.Vector3(p.x, 2.2, p.z), far: 22 }); });
  return { cols: keep, zamzam: zz };
}

// ---------- Aula: sedikit tiang persegi besar mengikuti bentuk mataf, lorong lurus bebas tiang dari tiap pintu ----------
export function buildHall(sim, ctx) {
  const { data, col, M, ROOF_Y, region, nearRamp, upAt, masaDist, REG, aisles } = ctx;
  const inner = data.mosque.inner[0];
  const NA = 720, rm = new Float32Array(NA);
  for (let i = 0; i < NA; i++) { const a = i / NA * TAU; rm[i] = rayRing(Math.cos(a), Math.sin(a), inner); }
  const rAt = (a) => { const f = (((a / TAU) % 1) + 1) % 1 * NA, i = Math.floor(f), w = f - i; return rm[i] * (1 - w) + rm[(i + 1) % NA] * w; };
  const mins = data.minarets, W = 1.6, R = W * 0.75;
  const segD = (x, z, a, b) => { const dx = b[0] - a[0], dz = b[1] - a[1], L2 = dx * dx + dz * dz || 1e-9, u = Math.max(0, Math.min(1, ((x - a[0]) * dx + (z - a[1]) * dz) / L2)); return Math.hypot(x - a[0] - u * dx, z - a[1] - u * dz); };
  const free = (x, z) => {
    const Rg = (dx, dz) => region.at(x + dx, z + dz) === REG.HALL;
    return Rg(0, 0) && Rg(4, 0) && Rg(-4, 0) && Rg(0, 4) && Rg(0, -4) && Rg(2.9, 2.9) && Rg(-2.9, 2.9) && Rg(2.9, -2.9) && Rg(-2.9, -2.9)
      && !nearRamp(x, z, 3) && !upAt(x, z, 3) && masaDist(x, z) > 14 // eskalator rooftop di dalam aula: tiang menjauh 3 m && !mins.some((m) => Math.hypot(m[0] - x, m[1] - z) < 4.5)
      && !aisles.some(([a, b]) => segD(x, z, a, b) < 7); // lorong lurus pintu -> Ka'bah tetap lapang
  };
  const SP = 31, pillars = [];
  for (let off = 16; off < 440; off += 31) {
    const pts = []; for (let i = 0; i <= 1440; i++) { const a = i / 1440 * TAU, r = rAt(a) + off; pts.push([Math.cos(a) * r, Math.sin(a) * r]); }
    let acc = 0, next = SP / 2;
    for (let i = 1; i < pts.length; i++) {
      const a = pts[i - 1], b = pts[i], L = Math.hypot(b[0] - a[0], b[1] - a[1]);
      while (next <= acc + L) {
        const t = (next - acc) / L, x = a[0] + (b[0] - a[0]) * t, z = a[1] + (b[1] - a[1]) * t; next += SP;
        if (free(x, z)) pillars.push({ x, z });
      }
      acc += L;
    }
  }
  sectorInstanced(sim, squarePillarGeo(ROOF_Y, W), M.pillar, pillars, (o, p) => { o.position.set(p.x, 0, p.z); o.rotation.y = Math.atan2(p.x, p.z); }, 4);
  pillars.forEach((p) => col.addCircle(p.x, p.z, R));
  return { pillars };
}

// ---------- Mas'a: lorong Safa-Marwah ----------
export function buildMasa(sim, ctx) {
  const { data, col, M, ROOF_Y, solid, region, out, tex, REG } = ctx;
  const S = data.safa, Mw = data.marwah, L = Math.hypot(Mw[0] - S[0], Mw[1] - S[1]);
  const t = [(Mw[0] - S[0]) / L, (Mw[1] - S[1]) / L], n = [-t[1], t[0]]; // n = timur
  const P = (f, o) => [S[0] + t[0] * L * f + n[0] * o, S[1] + t[1] * L * f + n[1] * o];
  const HALF = 11, inHall = (p, m = 1.8) => [[0, 0], [m, 0], [-m, 0], [0, m], [0, -m]].every(([dx, dz]) => region.at(p[0] + dx, p[1] + dz) === REG.HALL);
  // Deret tiang persegi di kedua sisi lorong
  const pil = [];
  for (const side of [-1, 1]) for (let s = 14; s <= L - 14; s += 24) { const p = P(s / L, side * HALF); if (inHall(p)) pil.push({ x: p[0], z: p[1] }); }
  sectorInstanced(sim, squarePillarGeo(ROOF_Y, 1.3), M.pillar, pil, (o, p) => { o.position.set(p.x, 0, p.z); o.rotation.y = Math.atan2(t[0], t[1]); }, 1);
  pil.forEach((p) => col.addCircle(p.x, p.z, 0.95));
  // Plafon berhias (di bawah atap), lantai berjalur
  const ceilB = ctx.Builder(), V = ceilB.V, DOWN = V(0, -1, 0), UP = V(0, 1, 0);
  const W0 = -HALF + 0.5, W1 = HALF - 0.5, steps = Math.ceil(L / 6);
  for (let i = 0; i < steps; i++) {
    const f0 = i / steps, f1 = (i + 1) / steps, a0 = P(f0, W0), a1 = P(f1, W0), b1 = P(f1, W1), b0 = P(f0, W1);
    const u0 = f0 * L / 6, u1 = f1 * L / 6, v0 = W0 / 6, v1 = W1 / 6;
    ceilB.quad(V(a0[0], ROOF_Y - 0.05, a0[1]), V(a1[0], ROOF_Y - 0.05, a1[1]), V(b1[0], ROOF_Y - 0.05, b1[1]), V(b0[0], ROOF_Y - 0.05, b0[1]), DOWN, '#ffffff', [[u0, v0], [u1, v0], [u1, v1], [u0, v1]]);
  }
  const ceiling = ceilB.mesh(new THREE.MeshLambertMaterial({ map: tex.ceil, vertexColors: true })); sim.scene.add(ceiling); out.roofs.push(ceiling);
  const strip = (o0, o1, y, color, f0 = 0.045, f1 = 0.955) => { // berhenti sebelum alas bukit (tidak berimpit)
    const a0 = P(f0, o0), a1 = P(f1, o0), b1 = P(f1, o1), b0 = P(f0, o1);
    solid.quad(V(a0[0], y, a0[1]), V(a1[0], y, a1[1]), V(b1[0], y, b1[1]), V(b0[0], y, b0[1]), UP, color);
  };
  strip(-1.8, 1.8, 0.07, '#eef0f1');                                   // jalur tengah (kursi roda)
  for (const o of [-9.6, -1.9, 1.9, 9.6]) strip(o - 0.11, o + 0.11, 0.08, '#55534d');
  // Lampu hijau (lari kecil bagi laki-laki): ±30-44 % dari Safa
  const G0 = 0.3, G1 = 0.44, glow = ctx.Builder(), gc = '#22c55e';
  for (const o of [-8, -3, 3, 8]) {
    const a = P(G0, o), b = P(G1, o);
    glow.slab(a, b, 0.3, ROOF_Y - 0.45, ROOF_Y - 0.2, gc);
  }
  for (const f of [G0, G1]) glow.slab(P(f, -HALF + 1), P(f, HALF - 1), 0.35, ROOF_Y - 0.5, ROOF_Y - 0.2, gc);
  pil.forEach((p) => { const f = ((p.x - S[0]) * t[0] + (p.z - S[1]) * t[1]) / L; if (f >= G0 - 0.02 && f <= G1 + 0.02) glow.prism([[p.x - 1.0, p.z - 1.0], [p.x + 1.0, p.z - 1.0], [p.x + 1.0, p.z + 1.0], [p.x - 1.0, p.z + 1.0]], ROOF_Y - 1.9, ROOF_Y - 1.5, gc, true); });
  // Lampu gantung
  for (let s = 12; s < L - 6; s += 12) for (const o of [-5.5, 5.5]) {
    const [x, z] = P(s / L, o);
    glow.prism([[x - 0.35, z - 0.35], [x + 0.35, z - 0.35], [x + 0.35, z + 0.35], [x - 0.35, z + 0.35]], ROOF_Y - 2.1, ROOF_Y - 1.5, '#fff3c4', true);
  }
  sim.scene.add(glow.mesh(new THREE.MeshBasicMaterial({ vertexColors: true })));
  const gm = P((G0 + G1) / 2, 0); const fm = new THREE.Mesh(new THREE.PlaneGeometry(L * (G1 - G0), HALF * 2 - 2), new THREE.MeshBasicMaterial({ color: gc, transparent: true, opacity: 0.08, depthWrite: false }));
  fm.rotation.set(-Math.PI / 2, 0, Math.atan2(-t[1], t[0])); fm.position.set(gm[0], 0.16, gm[1]); sim.scene.add(fm);
  out.labels.push({ text: 'Lampu hijau', pos: new THREE.Vector3(gm[0], ROOF_Y - 1.2, gm[1]), far: 45 });

  // Bukit Safa & Marwah (batu low-poly seperti simulasi sa'i, diperkecil), kaca rendah menghadap lorong
  function rock(c, toward, seed, name) {
    const geo = new THREE.IcosahedronGeometry(7, 2), pos = geo.attributes.position, v = new THREE.Vector3();
    const hash = (x, y, z) => { const h = Math.sin(x * 12.9898 + y * 78.233 + z * 37.719 + seed) * 43758.5453; return h - Math.floor(h); };
    for (let i = 0; i < pos.count; i++) {
      v.fromBufferAttribute(pos, i); const k = 0.78 + hash(Math.round(v.x * 3), Math.round(v.y * 3), Math.round(v.z * 3)) * 0.38;
      v.multiplyScalar(k); if (v.y < 0) v.y *= 0.15; pos.setXYZ(i, v.x, v.y, v.z);
    }
    geo.computeVertexNormals();
    const m = new THREE.Mesh(geo, new THREE.MeshLambertMaterial({ color: '#8f8475', flatShading: true }));
    m.scale.set(0.7, 0.45, 0.75); m.position.set(c[0], 0, c[1]); m.rotation.y = Math.atan2(t[0], t[1]); sim.scene.add(m);
    const yaw = Math.atan2(toward[0], toward[1]);
    const glass = new THREE.Mesh(new THREE.CylinderGeometry(6, 6, 1.2, 24, 1, true, -Math.PI / 2, Math.PI), new THREE.MeshLambertMaterial({ color: '#cfe7f2', transparent: true, opacity: 0.35, side: THREE.DoubleSide, depthWrite: false }));
    glass.position.set(c[0], 0.6, c[1]); glass.rotation.y = yaw; sim.scene.add(glass);
    const plat = new THREE.Mesh(new THREE.CircleGeometry(7.5, 32), sim.mat('#f4f5f5')); plat.rotation.x = -Math.PI / 2; plat.position.set(c[0], 0.12, c[1]); sim.scene.add(plat);
    col.addCircle(c[0], c[1], 5.6);
    out.labels.push({ text: name, pos: new THREE.Vector3(c[0], 6, c[1]), far: 80, cls: 'hl' });
  }
  rock(S, t, 1.3, "Bukit Safa · awal sa'i");
  rock(Mw, [-t[0], -t[1]], 7.1, 'Bukit Marwah');
  return { pillars: pil, t, n, L };
}

// ---------- Half Million: kafe 2 lantai di pelataran (bangunan ikonik di kaki hotel) ----------
// Segi delapan memanjang: lantai dasar batu travertine (pintu kaca + panel kaligrafi), pita batu berlogo MCDC, lantai 2
// kaca gelap bertanda "½M" menyala, mahkota batu. Tangga di kiri-kanan (bisa dinaiki) ke teras lantai 2 di belakang.
// Koordinat lokal: x kanan, z depan (menghadap masjid); grup diputar l.dir. Mengembalikan floorAt(x, z) (tangga/teras).
export function buildHalfMillion(sim, ctx, l) {
  const { col, out, Builder } = ctx, { canvasTex } = sim;
  const [cx, cz] = l.pos, th = l.dir, cs = Math.cos(th), sn = Math.sin(th);
  const Wd = (lx, lz) => [cx + cs * lx + sn * lz, cz - sn * lx + cs * lz];                       // lokal -> dunia
  const Lc = (x, z) => { const dx = x - cx, dz = z - cz; return [cs * dx - sn * dz, sn * dx + cs * dz]; }; // dunia -> lokal
  const g = new THREE.Group(); g.position.set(cx, 0, cz); g.rotation.y = th; sim.scene.add(g);
  const ST = '#e9e4da', ST2 = '#d9d2c4', GL = '#2a3139', FR = '#b9b3a8';
  const G0 = 4.6, B1 = 6.2, G2 = 10.2, TOP = 11.0, RISE = B1, SX = 6.3, SX1 = 9.9, Z0 = 4.5, Z1 = -8.5, ZT = -12.5;
  const stone = Builder(), V = stone.V;
  // badan: segi delapan 12 m lebar, dari z = 4,5 (muka depan 6 m) sampai z = -8,5
  const ring = [[-3, Z0], [3, Z0], [6, 1.5], [6, -5.5], [3, Z1], [-3, Z1], [-6, -5.5], [-6, 1.5]];
  const grow = (k) => ring.map(([x, z]) => [x * (1 + k / 6), (z + 2) * (1 + k / 6.5) - 2]); // melebar k meter dari pusat (0, -2)
  stone.prism(ring, 0, G0, ST);                       // lantai dasar
  stone.prism(grow(0.12), G0, B1, ST2, true);         // pita batu antar lantai (sedikit menonjol)
  stone.prism(grow(0.3), G2, TOP, ST2, true);         // mahkota
  ring.forEach(([x, z]) => stone.prism([[x - 0.15, z - 0.15], [x + 0.15, z - 0.15], [x + 0.15, z + 0.15], [x - 0.15, z + 0.15]], B1, G2, ST2)); // tiang sudut lantai kaca
  // undakan depan (3 anak tangga) & pintu kaca berbingkai; pintu kaca lantai 2 di belakang (dari teras)
  for (let i = 0; i < 3; i++) stone.prism([[-3.2, Z0 - 0.01], [3.2, Z0 - 0.01], [3.2, Z0 + 1.2 - i * 0.4], [-3.2, Z0 + 1.2 - i * 0.4]], 0, 0.15 * (i + 1), ST2);
  panel(stone, [-2.35, Z0 + 0.02], [2.35, Z0 + 0.02], 0.45, 3.5, FR, undefined, -1); panel(stone, [-2.2, Z0 + 0.04], [2.2, Z0 + 0.04], 0.45, 3.35, GL, undefined, -1); // muka depan: p -> q dari kiri ke kanan dilihat dari depan, normal +z
  panel(stone, [-1.6, Z1 - 0.02], [1.6, Z1 - 0.02], RISE, RISE + 3.0, GL);
  // panel kaligrafi (hitam, tulisan putih) di atas pintu
  const cal = canvasTex(512, 128, (c, w, h) => {
    c.fillStyle = '#15181b'; c.fillRect(0, 0, w, h); c.strokeStyle = '#8a8378'; c.lineWidth = 3; c.strokeRect(4, 4, w - 8, h - 8);
    c.fillStyle = '#f3efe6'; c.textAlign = 'center'; c.textBaseline = 'middle'; c.direction = 'rtl'; c.font = 'bold 26px "Geeza Pro", "Amiri", "Scheherazade New", serif';
    ['لا إله إلا الله وحده لا شريك له', 'له الملك وله الحمد يحيي ويميت', 'وهو على كل شيء قدير'].forEach((t, i) => c.fillText(t, w / 2, 26 + i * 38));
  });
  const calB = Builder(); panel(calB, [-2.3, Z0 + 0.03], [2.3, Z0 + 0.03], 3.5, 4.45, '#ffffff', undefined, -1);
  g.add(calB.mesh(new THREE.MeshLambertMaterial({ map: cal, vertexColors: true })));
  // lantai 2: kaca gelap berkusen (tekstur berulang tiap 4 m), tanda "½M" putih menyala di muka depan, logo MCDC di pita
  const glassTex = canvasTex(256, 256, (c, w, h) => {
    const gr = c.createLinearGradient(0, 0, 0, h); gr.addColorStop(0, '#3b444e'); gr.addColorStop(1, '#1f252c'); c.fillStyle = gr; c.fillRect(0, 0, w, h);
    c.fillStyle = 'rgba(255,255,255,.07)'; for (let x = 0; x < w; x += 64) c.fillRect(x + 22, 0, 12, h);
    c.fillStyle = '#b9b3a8'; for (let x = 0; x < w; x += 64) c.fillRect(x, 0, 3, h); c.fillRect(0, h / 2 - 2, w, 4);
  });
  glassTex.wrapS = glassTex.wrapT = THREE.RepeatWrapping;
  const glass = Builder(); glass.sides(ring, B1, G2, '#ffffff', false, { w: 4, h: 4 });
  g.add(glass.mesh(new THREE.MeshLambertMaterial({ map: glassTex, vertexColors: true })));
  const sign = canvasTex(512, 256, (c, w, h) => { c.fillStyle = '#ffffff'; c.textAlign = 'center'; c.textBaseline = 'middle'; c.font = 'bold 190px "Helvetica Neue", Arial, sans-serif'; c.fillText('½M', w / 2, h / 2 + 8); });
  const mcdc = canvasTex(512, 192, (c, w, h) => {
    c.fillStyle = '#ffffff'; c.textAlign = 'left'; c.textBaseline = 'middle';
    c.font = 'bold 66px "Helvetica Neue", Arial, sans-serif'; c.fillText('MCDC', 60, 62);
    c.font = '30px "Geeza Pro", "Amiri", serif'; c.direction = 'rtl'; c.textAlign = 'right'; c.fillText('شركة مكة للإنشاء والتعمير', 300, 120);
    c.fillStyle = '#ffffff'; for (const dx of [0, 44]) { c.beginPath(); c.moveTo(380 + dx, 150); c.lineTo(420 + dx, 40); c.lineTo(448 + dx, 40); c.lineTo(408 + dx, 150); c.closePath(); c.fill(); } // tanda logo
  });
  const signB = Builder(); panel(signB, [-2.4, Z0 + 0.04], [2.4, Z0 + 0.04], 7.0, 9.4, '#ffffff', undefined, -1);
  g.add(signB.mesh(new THREE.MeshBasicMaterial({ map: sign, vertexColors: true, transparent: true, alphaTest: 0.1 })));
  const logoB = Builder(); panel(logoB, [-1.6, Z0 + 0.14], [1.6, Z0 + 0.14], 4.8, 6.0, '#ffffff', undefined, -1);
  g.add(logoB.mesh(new THREE.MeshBasicMaterial({ map: mcdc, vertexColors: true, transparent: true, alphaTest: 0.1 })));
  // tangga kiri & kanan: 20 anak tangga pejal dari z = 4,5 (lantai) ke z = -8,5 (teras), pagar 1 m di kedua sisi
  const N = 20, run = (Z0 - Z1) / N, rise = RISE / N;
  for (const sg of [1, -1]) {
    const xa = sg * SX, xb = sg * SX1;
    for (let i = 0; i < N; i++) { const z0 = Z0 - i * run, z1 = z0 - run; stone.prism([[xa, z0], [xb, z0], [xb, z1], [xa, z1]], 0, rise * (i + 1), i % 2 ? '#f0ece4' : '#e8e3d9'); }
    for (const [xr, nx] of [[xb, sg], [xa, -sg]]) { // pagar miring: dua muka (luar & dalam), tebal 0,12
      const xi = xr - nx * 0.12, R = 1.0;
      stone.quad(V(xr, 0, Z0), V(xr, RISE, Z1), V(xr, RISE + R, Z1), V(xr, R, Z0), V(nx, 0, 0), ST2);
      stone.quad(V(xi, R, Z0), V(xi, RISE + R, Z1), V(xi, RISE, Z1), V(xi, 0, Z0), V(-nx, 0, 0), ST2);
      stone.quad(V(xr, R, Z0), V(xr, RISE + R, Z1), V(xi, RISE + R, Z1), V(xi, R, Z0), V(0, 1, 0), ST2);
      const A = Wd(xr, Z0), B = Wd(xr, Z1); col.addSeg(A[0], A[1], B[0], B[1], { h: RISE + 1.4, tag: 'solid' });
    }
  }
  // teras lantai 2 di belakang (blok pejal sampai tanah), pagar belakang & samping
  stone.prism([[-SX1, Z1 + 0.2], [SX1, Z1 + 0.2], [SX1, ZT], [-SX1, ZT]], 0, RISE, ST);
  for (const [a, b] of [[[-SX1, ZT], [SX1, ZT]], [[-SX1, Z1], [-SX1, ZT]], [[SX1, Z1], [SX1, ZT]]]) {
    stone.slab(a, b, 0.15, RISE, RISE + 1.0, ST2); const A = Wd(...a), B = Wd(...b); col.addSeg(A[0], A[1], B[0], B[1], { h: RISE + 1.4, tag: 'solid' });
  }
  g.add(stone.mesh(new THREE.MeshLambertMaterial({ vertexColors: true })));
  col.addPoly([[-3.2, Z0 + 1.2], [3.2, Z0 + 1.2], ...ring.slice(2)].map(([x, z]) => Wd(x, z)), { h: TOP + 0.3, tag: 'solid' });
  out.labels.push({ text: '☕ ' + l.name, pos: new THREE.Vector3(cx, TOP + 1.8, cz), far: 140, cls: 'hl' });
  const floorAt = (x, z) => { const [lx, lz] = Lc(x, z), ax = Math.abs(lx);
    if (lz <= Z0 && lz >= Z1 && ax >= SX && ax <= SX1) return RISE * (Z0 - lz) / (Z0 - Z1);
    if (lz < Z1 && lz >= ZT && ax <= SX1) return RISE;
    return null; };
  const at = (x, z) => { const [lx, lz] = Lc(x, z); return Math.abs(lx) <= SX1 + 0.4 && lz <= Z0 + 1.6 && lz >= ZT - 0.4; };
  // Spot "ORDER HERE" di depan undakan: lingkaran hitam-putih berdenyut (haram.js: pemain yang berdiri di sini mendapat tumbler ½M)
  const spotTex = canvasTex(256, 256, (c, w) => {
    c.fillStyle = '#17181a'; c.beginPath(); c.arc(w / 2, w / 2, 120, 0, TAU); c.fill(); // hitam-putih
    c.strokeStyle = '#ffffff'; c.lineWidth = 9; c.beginPath(); c.arc(w / 2, w / 2, 106, 0, TAU); c.stroke();
    c.fillStyle = '#ffffff'; c.textAlign = 'center'; c.textBaseline = 'middle'; c.font = 'bold 44px "Helvetica Neue", Arial, sans-serif';
    c.fillText('ORDER', w / 2, 102); c.fillText('HERE', w / 2, 148);
    c.font = 'bold 30px "Helvetica Neue", Arial, sans-serif'; c.fillText('½M', w / 2, 196);
  });
  const disc = new THREE.Mesh(new THREE.CircleGeometry(1.4, 40), new THREE.MeshBasicMaterial({ map: spotTex, transparent: true, depthWrite: false }));
  disc.rotation.x = -Math.PI / 2; disc.position.set(0, 0.07, 9.5); g.add(disc); // teks terbaca dari arah pelataran (datang menghadap -z)
  const sp = Wd(0, 9.5); out.labels.push({ text: '☕ Order here', pos: new THREE.Vector3(sp[0], 3.4, sp[1]), far: 45, hideNear: 3 }); // disembunyikan saat pemain dekat (tidak menutupi tumbler)
  return { floorAt, at, spot: { x: sp[0], z: sp[1], r: 1.4, mesh: disc } };
}
