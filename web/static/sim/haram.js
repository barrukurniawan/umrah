// Jelajah Masjidil Haram (open world, lantai dasar). Denah dari OpenStreetMap (data/haram.json, lihat
// scripts/haram-osm). Arah: x = timur, -z = utara; pusat Ka'bah (0, 0). Rencana: PLAN_OPEN_WORLD.md.
import { createSim, THREE, $, TAU, mod } from './common.js?v=49';
import { createCollision, createControls } from './nav.js?v=49';
import { buildHaram, inPoly, ROOF_Y, REG, LIFT, RAMP_END } from './haram-world.js?v=49';
import { buildCrowd } from './haram-crowd.js?v=49';
import { TAKBIR, SAPU_JAGAD, MAQAM_AYAT, AYAT, DZIKIR, RABBIGHFIR } from './doa.js?v=49';

const data = await (await fetch('/static/sim/data/haram.json?v=49')).json();
// Bangunan kecil OSM di pelataran (kotak rendah & bangunan <14 m: pintu toilet/parkir bawah tanah, kios, pos) terlalu
// banyak dan tidak jelas fungsinya: 60% dihapus. Semua yang <25 m dari dinding masjid (di jalur pintu) hilang, sisanya
// dijarangkan merata menurut sudut di sekeliling masjid. Gedung tinggi & jembatan tetap.
{
  const segD = (x, z, a, b) => { const dx = b[0] - a[0], dz = b[1] - a[1], L2 = dx * dx + dz * dz || 1e-9, u = Math.max(0, Math.min(1, ((x - a[0]) * dx + (z - a[1]) * dz) / L2)); return Math.hypot(x - a[0] - u * dx, z - a[1] - u * dz); };
  const dMosque = (x, z) => Math.min(...data.mosque.outer.flatMap((r) => r.map((p, i) => segD(x, z, p, r[(i + 1) % r.length]))));
  const small = data.buildings.filter((b) => !b.bridge && b.h < 14).map((b) => {
    const cx = b.poly.reduce((s, p) => s + p[0], 0) / b.poly.length, cz = b.poly.reduce((s, p) => s + p[1], 0) / b.poly.length;
    return { b, d: dMosque(cx, cz), a: Math.atan2(cz, cx) };
  });
  const drop = new Set(small.filter((o) => o.d < 25).map((o) => o.b));
  const rest = small.filter((o) => !drop.has(o.b)).sort((p, q) => p.a - q.a), want = Math.round(small.length * 0.6) - drop.size;
  rest.forEach((o, i) => { if (Math.floor((i + 1) * want / rest.length) > Math.floor(i * want / rest.length)) drop.add(o.b); }); // merata
  data.buildings = data.buildings.filter((b) => !drop.has(b));
}
const sim = createSim({ part: 'haram', bg: '#e8edef', fogNear: 170, fogFar: 620, far: 1600, fov: 50, camLerp: 6 });
const { scene, camera, S, cam, hooks } = sim;
cam.zoom = 1.9; // bawaan = zoom terjauh (batas common.js 0,55-1,9); cubit/scroll untuk mendekat
scene.add(new THREE.HemisphereLight('#dfe9f2', '#cfcdc8', 0.25));

const col = createCollision(8);
const world = buildHaram(sim, data, col);
const labels = world.labels.map((l) => ({ ...l, el: sim.addLabel(l.text, l.pos, l.cls || '') }));
const crowd = buildCrowd(sim, { world, data, col, LIFT, RAMP_END, REG }); // jamaah (Tahap 3)

// ---------- Pemain ----------
const RADIUS = 0.45, WALK = 12.4, RUN = 19; // 2x kecepatan tawaf, supaya pelataran yang luas cepat dijelajahi
const FOOT = 0.07; // telapak sedikit di atas lantai (tidak berimpit)
const P = { x: 0, y: 0, z: 0, layer: 0 }; // y = tinggi lantai di bawah pemain; layer 0 = lantai dasar, 1 = rooftop
const colOf = () => (P.layer ? world.colR : col);
const view = { yaw: 0, pitch: 0.42 };
const controls = createControls({ stage: sim.stage, canvas: $('simCanvas'), joyEl: $('joy'), knobEl: $('joyKnob') });
// Mode ringan mengganti canvas: pasang ulang geser-kamera di canvas baru.
hooks.lite = () => {
  controls.bindCanvas($('simCanvas'));
  crowd.thin(0.4); // HP lemah: 40 % jamaah
  // HP lemah: kabut & batas pandang lebih dekat (bangunan jauh tidak digambar)
  scene.fog.near = 140; scene.fog.far = 430; camera.far = 470; camera.updateProjectionMatrix();
};

const outers = data.mosque.outer, inners = data.mosque.inner;
const inMataf = (x, z) => world.region.at(x, z) === REG.MATAF;
const indoors = (x, z) => world.region.at(x, z) === REG.HALL;
const [SX, SZ] = data.safa, [MX, MZ] = data.marwah;
const masaDist = world.masaDist;
const gateLabel = (g) => g.name ? `${g.name}${g.ref ? ' (Pintu ' + g.ref + ')' : ''}` : (g.ref ? `Pintu ${g.ref}` : 'Pintu');
const gateShort = (g) => g.ref ? `Pintu ${g.ref}` : (g.name || 'Pintu'); // untuk status kiri atas (sempit di HP)

// ---------- Titik lompat ----------
const faceTo = (fx, fz, tx, tz) => Math.atan2(tx - fx, tz - fz);
const spots = [];
{
  const ha = world.kaaba.C_HA, d = Math.hypot(ha.x, ha.z);
  spots.push({ name: "Ka'bah (garis Hajar Aswad)", x: ha.x / d * 22, z: ha.z / d * 22, face: faceTo(ha.x / d * 22, ha.z / d * 22, 0, 0) });
  const toM = [MX - SX, MZ - SZ], L = Math.hypot(...toM);
  spots.push({ name: 'Bukit Safa', x: SX + toM[0] / L * 7, z: SZ + toM[1] / L * 7, face: faceTo(SX, SZ, 0, 0) });
  spots.push({ name: 'Bukit Marwah', x: MX - toM[0] / L * 7, z: MZ - toM[1] / L * 7, face: faceTo(MX, MZ, SX, SZ) });
  for (const g of data.gates) if (g.main) {
    const k = /^King /.test(g.name || '') ? 16 : 14; // di luar pintu (King: di teras), gerbang terlihat utuh
    const x = g.pos[0] + Math.sin(g.dir) * k, z = g.pos[1] + Math.cos(g.dir) * k;
    spots.push({ name: gateLabel(g), x, z, face: g.dir + Math.PI, gate: g });
  }
  for (const l of data.landmarks || []) { const d = l.dir ?? 0; spots.push({ name: l.name, x: l.pos[0] + Math.sin(d) * 18, z: l.pos[1] + Math.cos(d) * 18, face: d + Math.PI }); } // 18 m di depan, menghadap bangunan
  for (const r of world.roofSpots) spots.push({ ...r, layer: 1 });
}
const SPAWN = spots.find((s) => s.gate && s.gate.ref === '1') || spots[0];
const sel = $('haramJump');
sel.innerHTML = '<option value="">Lompat ke…</option>' + spots.map((s, i) => `<option value="${i}">${s.name}</option>`).join('');
sel.onchange = () => { const s = spots[+sel.value]; sel.value = ''; sel.blur(); if (s && sim.chibi) teleport(s.x, s.z, s.face, s.layer || 0); };

function teleport(x, z, face, layer = 0) {
  P.x = x; P.z = z; P.layer = layer; colOf().resolve(P, RADIUS); P.y = world.floorY(P.x, P.z, P.layer);
  const g = sim.chibi.g; g.position.set(P.x, P.y + FOOT, P.z); g.rotation.y = face; sim.chibi.lastYaw = face;
  view.yaw = face + Math.PI; view.pitch = 0.42;
  const want = hooks.camera(0); if (want) { cam.pos.copy(want.pos); cam.tgt.copy(want.tgt); }
  zoneTick = 0;
}

// ---------- Zona, tempat penting & panel pemandu (Tahap 4) ----------
let zone = '', zoneTick = 0, nearGate = null, zoneKey = '';
const ZONES = {
  out: { short: 'Pelataran', title: 'Pelataran Masjidil Haram', body: '<p>Halaman luas di sekeliling masjid. Pintu biasa masuk rata ke lantai masjid. Gerbang <b>King</b> (King Abdul Aziz, King Fahd, King Abdullah) berada di atas teras: naik tangga, masuk, lalu turun eskalator. <b>Eskalator ke atap</b> ada di samping beberapa pintu (kotak hijau di mini-peta).</p>' },
  in: { short: 'Di dalam masjid', title: 'Di dalam Masjidil Haram', body: '<p>Lantai dasar masjid. Ikuti lorong lurus tanpa tiang (atau arah Ka\'bah di mini-peta) untuk sampai ke <b>mataf</b>, pelataran tempat tawaf.</p>' },
  mataf: { short: 'Mataf', title: 'Mataf', body: '<p>Pelataran terbuka di sekeliling Ka\'bah, tempat tawaf. Tawaf dimulai dari garis sejajar <b>Hajar Aswad</b> (lampu hijau), Ka\'bah di sebelah kiri, 7 putaran.</p>' },
  masa: { short: "Mas'a", title: "Mas'a (tempat sa'i)", body: '<p>Lorong antara bukit <b>Safa</b> (selatan) dan <b>Marwah</b> (utara), tempat sa\'i 7 lintasan. Sa\'i dimulai di <b>Safa</b>.</p>' },
  roof: { short: 'Atap', title: 'Atap Masjidil Haram (rooftop)', body: '<p>Lantai atap yang terbuka. Dari pagar tepi mataf terlihat Ka\'bah dan jamaah tawaf di bawah. Turun lagi lewat eskalator di tepi atap.</p>' },
};
const CONTROLS_HINT = window.matchMedia('(pointer: coarse)').matches
  ? 'Joystick kiri untuk berjalan, geser layar untuk memutar pandangan, cubit untuk zoom.'
  : 'W/A/S/D atau panah untuk berjalan (Shift = cepat), geser mouse untuk memutar pandangan, scroll untuk zoom.';
// Zona warna pintu (papan penunjuk di Masjidil Haram), menurut sudut dari Ka'bah ke pintu jangkarnya.
const GATE_ZONES = [['1', 'Hijau', '#16a34a', 'sisi selatan'], ['79', 'Cokelat', '#92400e', 'sisi barat'], ['40', 'Marun', '#881337', 'sisi barat laut'], ['100', 'Biru', '#1d4ed8', 'sisi utara, perluasan']]
  .map(([ref, name, color, side]) => { const g = data.gates.find((q) => q.ref === ref); return g && { ref, name, color, side, a: Math.atan2(g.pos[1], g.pos[0]) }; }).filter(Boolean);
const gateZone = (g) => {
  const a = Math.atan2(g.pos[1], g.pos[0]); let best = null, bd = 0.75;
  for (const z of GATE_ZONES) { const d = Math.abs(mod(a - z.a + Math.PI, TAU) - Math.PI); if (d < bd) { bd = d; best = z; } }
  return best;
};
const zoneChip = (z) => z ? ` <span style="display:inline-flex;align-items:center;gap:4px;font-weight:600;color:${z.color}"><i style="display:inline-block;width:9px;height:9px;border-radius:3px;background:${z.color}"></i>Zona ${z.name}</span>` : '';
// Tempat penting: kartu muncul saat mendekat (yang paling dekat relatif terhadap jangkauannya).
const kb = world.kaaba, toV = (v) => [v.x, v.z];
const MULTAZAM = toV(kb.toWorld(5.6, 0, 4.6));
const POIS = [
  { id: 'ha', at: toV(kb.C_HA), r: 7, title: 'Hajar Aswad', body: '<p>Batu hitam di sudut Ka\'bah, titik awal dan akhir setiap putaran tawaf. Bila tidak bisa menyentuh atau menciumnya, cukup <b>isyarat tangan kanan</b> dari jauh sambil membaca:</p>', dua: TAKBIR },
  { id: 'ym', at: toV(kb.C_YM), r: 6, title: 'Rukun Yamani', body: '<p>Sudut Ka\'bah sebelum Hajar Aswad. Disunnahkan mengusapnya bila mudah; bila tidak, lewati saja tanpa isyarat. Di antara Rukun Yamani dan Hajar Aswad membaca:</p>', dua: SAPU_JAGAD },
  { id: 'mz', at: MULTAZAM, r: 4.5, title: 'Multazam', body: '<p>Dinding Ka\'bah di antara Hajar Aswad dan pintu Ka\'bah. Tempat yang baik untuk berdoa dengan doa apa saja, bila tidak berdesakan.</p>' },
  { id: 'mq', at: toV(kb.MQ), r: 5, title: 'Maqam Ibrahim', body: '<p>Batu bekas pijakan Nabi Ibrahim saat membangun Ka\'bah. Setelah tawaf, shalat sunnah 2 rakaat di belakangnya (atau di bagian mana saja bila penuh), sambil membaca:</p>', dua: MAQAM_AYAT },
  { id: 'hj', at: toV(kb.HIJR_MID), r: 9, title: 'Hijr Ismail', body: '<p>Area setengah lingkaran berdinding rendah di sisi utara Ka\'bah, termasuk bagian dari Ka\'bah. Tawaf dilakukan <b>di luar</b> Hijr Ismail.</p>' },
  { id: 'sf', at: data.safa, r: 12, title: "Bukit Safa", body: '<p>Titik awal sa\'i. Saat pertama naik ke Safa, baca:</p>', dua: AYAT, after: '<p>Lalu menghadap Ka\'bah, bertakbir, dan berdoa.</p>', sai: true },
  { id: 'mw', at: data.marwah, r: 12, title: 'Bukit Marwah', body: '<p>Ujung lintasan sa\'i. Di sini menghadap Ka\'bah dan berdoa seperti di Safa:</p>', dua: DZIKIR },
];
for (const p of world.zamzam || []) POIS.push({ id: 'zz', at: [p.x, p.z], r: 3.5, title: 'Air Zamzam', body: '<p>Dispenser air Zamzam (putih-biru) tersedia di banyak titik. Disunnahkan minum Zamzam setelah tawaf sambil berdoa apa saja.</p>' });
const SA = data.safa, MA = data.marwah, SAL = Math.hypot(MA[0] - SA[0], MA[1] - SA[1]);
const inGreen = (x, z) => { const f = ((x - SA[0]) * (MA[0] - SA[0]) + (z - SA[1]) * (MA[1] - SA[1])) / (SAL * SAL); return f > 0.3 && f < 0.44 && masaDist(x, z) < 11; };
const GREEN = { id: 'gr', title: 'Lampu hijau', body: '<p>Di antara dua penanda lampu hijau, <b>laki-laki berlari kecil</b>; perempuan berjalan biasa. Doa yang dianjurkan:</p>', dua: RABBIGHFIR };
function nearestPoi() {
  if (P.layer) return null;
  let best = null, bs = 1;
  for (const p of POIS) { const s = Math.hypot(p.at[0] - P.x, p.at[1] - P.z) / p.r; if (s < bs) { bs = s; best = p; } }
  return best || (inGreen(P.x, P.z) ? GREEN : null);
}
// Tombol mulai: simpan posisi, lalu buka simulasi yang sudah ada (Tawaf/Sa'i) dengan tanda "dari Jelajah".
const BTN = 'shrink-0 whitespace-nowrap rounded-lg bg-emerald-700 hover:bg-emerald-800 text-white font-semibold px-2.5 py-1 text-[12px] lg:text-[13px]';
function startPart(part) {
  try { sessionStorage.setItem('ukHaramSpot', JSON.stringify({ x: P.x, z: P.z, layer: P.layer, yaw: view.yaw, face: sim.chibi.g.rotation.y })); } catch (e) {}
  const l = $('simLoading'); $('simLoadingText').textContent = 'Memuat simulasi ' + (part === 'sai' ? "Sa'i" : 'Tawaf') + '…'; l.classList.remove('hidden'); l.classList.add('flex');
  location.href = `/simulasi?bagian=${part}&g=${S.gq}&from=haram`;
}
function updateZone() {
  const z = P.layer ? 'roof' : inMataf(P.x, P.z) ? 'mataf' : indoors(P.x, P.z) ? (masaDist(P.x, P.z) < 11 ? 'masa' : 'in') : 'out';
  let g = null, gd = 7;
  for (const x of data.gates) { const d = Math.hypot(x.pos[0] - P.x, x.pos[1] - P.z); if (d < gd && (x.name || x.ref)) { gd = d; g = x; } }
  const poi = nearestPoi();
  const key = z + '|' + (g ? g.osm : '') + '|' + (poi ? poi.id + poi.at : '');
  if (key === zoneKey) return;
  zoneKey = key; zone = z; nearGate = g;
  const Z = ZONES[z], tawafBtn = z === 'mataf', saiBtn = poi && poi.sai;
  // Kartu ringkas (open world): tombol mulai di baris judul supaya selalu terlihat di panel yang kecil.
  const button = tawafBtn || saiBtn ? `<button type="button" id="btnStartPart" class="${BTN}">${tawafBtn ? 'Mulai Tawaf →' : "Mulai Sa'i →"}</button>` : '';
  const gateLine = g && !poi ? `<p><b>${gateLabel(g)}</b>${zoneChip(gateZone(g))}</p>` : '';
  if (poi) sim.info({ compact: true, button, title: poi.title, body: poi.body, dua: poi.dua, after: poi.after, hint: tawafBtn ? 'Tawaf lengkap 7 putaran dengan panduan & doa ada di simulasi Tawaf.' : saiBtn ? "Sa'i lengkap 7 lintasan dengan panduan & doa ada di simulasi Sa'i." : CONTROLS_HINT });
  else sim.info({ compact: true, button, title: Z.title, body: gateLine + Z.body, hint: CONTROLS_HINT });
  const b = $('btnStartPart'); if (b) b.onclick = () => startPart(tawafBtn ? 'tawaf' : 'sai');
  sim.setCounter();
}
hooks.counter = () => ({ text: sim.chibi ? (nearGate ? gateShort(nearGate) : ZONES[zone || 'out'].short) : 'Jelajah', panel: '', done: 0, active: false });

// ---------- Kait simulasi ----------
hooks.chibiOpts = () => ({ shoulderBare: false }); // idhtiba' hanya saat tawaf
hooks.canWalk = () => false;                      // tombol Jalan tidak dipakai di mode jelajah
hooks.begin = () => {
  S.phase = 'roam'; setupXray();
  // Kembali dari Tawaf/Sa'i (?back=tawaf|sai[-done]): selesai tawaf -> di belakang Maqam Ibrahim; selesai sa'i -> di
  // Marwah; selain itu kembali ke titik terakhir sebelum berangkat.
  const back = sim.query && sim.query.get('back');
  let spot = null; try { spot = JSON.parse(sessionStorage.getItem('ukHaramSpot') || 'null'); } catch (e) {}
  if (back === 'tawaf-done') { const m = kb.MQ, d = Math.hypot(m.x, m.z) || 1; teleport(m.x + m.x / d * 4, m.z + m.z / d * 4, Math.atan2(-m.x, -m.z)); }
  else if (back === 'sai-done') { const s = spots.find((q) => q.name === 'Bukit Marwah'); teleport(s.x, s.z, s.face); }
  else if (back && spot) { teleport(spot.x, spot.z, spot.face, spot.layer || 0); view.yaw = spot.yaw; }
  else teleport(SPAWN.x, SPAWN.z, SPAWN.face);
  if (back) { $('simToast').style.zIndex = 60; sim.toast('Kembali ke Jelajah'); } // di atas mini-peta
  zoneKey = ''; updateZone();
};

// ---------- Siluet karakter saat tertutup tiang/dinding ----------
// Bangunan tetap utuh (tidak dilubangi / tidak transparan). Hanya bagian karakter yang tertutup digambar sebagai
// siluet hijau samar di atasnya: karakter menulis stencil = 1 di piksel yang terlihat, siluet (tanpa uji kedalaman)
// hanya digambar di piksel karakter yang stencil-nya bukan 1. Aktif hanya bila garis kamera->karakter melewati
// tiang atau dinding (cek 2D murah), jadi biasanya tanpa draw call tambahan.
const XRAY_MAT = new THREE.MeshBasicMaterial({
  color: '#0b7a55', transparent: true, opacity: 0.5, depthTest: false, depthWrite: false,
  stencilWrite: true, stencilRef: 1, stencilFunc: THREE.NotEqualStencilFunc, stencilZPass: THREE.ReplaceStencilOp,
});
let xray = null;
function setupXray() {
  if (xray) { scene.remove(xray.g); xray = null; }
  const src = sim.chibi.g, srcNodes = [];
  src.traverse((o) => {
    srcNodes.push(o);
    if (o.isMesh && !o.material.transparent) {
      o.material = o.material.clone();
      Object.assign(o.material, { stencilWrite: true, stencilRef: 1, stencilFunc: THREE.AlwaysStencilFunc, stencilZPass: THREE.ReplaceStencilOp });
    }
  });
  const g = src.clone(true), nodes = [];
  g.traverse((o) => nodes.push(o));
  // siluet cukup dari ±12 bagian terbesar (kepala, badan, lengan, kaki): hemat draw call
  const big = nodes.filter((o) => o.isMesh && !o.material.transparent)
    .map((o) => { o.geometry.computeBoundingSphere(); return [o, o.geometry.boundingSphere.radius]; })
    .sort((p, q) => q[1] - p[1]).slice(0, 12).map(([o]) => o);
  nodes.forEach((o) => { if (o.isMesh) { const on = big.includes(o); o.material = XRAY_MAT; o.renderOrder = 50; o.userData.xr = on; o.visible = on; } });
  g.visible = false; scene.add(g);
  xray = { g, src: srcNodes, dst: nodes };
}
function syncXray() {
  if (!xray || !sim.chibi) return;
  const c = camera.position;
  // garis tengah + dua garis sejajar ±0,6 m (tebal dinding: sudut dinding di tepi celah pintu ikut terdeteksi)
  const dx = P.x - c.x, dz = P.z - c.z, L = Math.hypot(dx, dz) || 1, ox = -dz / L * 0.6, oz = dx / L * 0.6;
  // tinggi garis pandang (kamera -> dada) di titik potong harus di dalam rentang penghalang [lo, h]
  const C = colOf(), wallHit = (sx) => C.crossings(c.x + ox * sx, c.z + oz * sx, P.x + ox * sx, P.z + oz * sx, xrBuf).some((k) => { const y = c.y + (P.y + 1.0 - c.y) * k.u; return y < k.h && y >= (k.lo || 0); });
  const need = !cam.top && (C.circleOnLine(c.x, c.z, P.x, P.z, 0.6) || wallHit(0) || wallHit(1) || wallHit(-1));
  xray.g.visible = need;
  if (!need) return;
  const { src, dst } = xray;
  for (let i = 0; i < src.length; i++) {
    const a = src[i], b = dst[i];
    b.position.copy(a.position); b.quaternion.copy(a.quaternion); b.scale.copy(a.scale);
    if (b.isMesh) b.visible = b.userData.xr && a.visible;
  }
}
const xrBuf = [];
hooks.reset = () => {
  zone = ''; nearGate = null; zoneKey = '';
  sim.info({ compact: true, title: 'Jelajah Masjidil Haram', body: '<p>Pilih karakter, lalu berjalanlah bebas di pelataran, masuk lewat pintu mana pun, dan lihat Ka\'bah dari mataf.</p>', hint: CONTROLS_HINT });
};
let moving = 0;
// ---------- Half Million: selama berdiri di spot "Order here", tumbler ½M burgundy muncul di tangan kanan dan karakter
// menghadap kamera sambil mengangkatnya (siap tangkapan layar). Keluar dari spot: tumbler hilang. ----------
const order = world.orderSpot; let cup = null, orderT = 0;
function makeTumbler() {
  const g = new THREE.Group(); g.name = 'Tumbler';
  const bur = sim.mat('#6b1f2a'), burD = sim.mat('#561822'); // burgundy (merah kecoklatan)
  // ukuran chibi: tumbler sengaja besar (±0,42 satuan rig) supaya terbaca di layar HP
  // pivot grup = titik genggam (0,18 dari dasar) di telapak; orientasi dunia dipaksa TEGAK tiap frame (updateOrder)
  const GR = 0.18;
  const body = new THREE.Mesh(new THREE.CylinderGeometry(0.12, 0.105, 0.42, 20), bur); body.position.y = 0.21 - GR; g.add(body);
  const lid = new THREE.Mesh(new THREE.CylinderGeometry(0.128, 0.128, 0.055, 20), burD); lid.position.y = 0.447 - GR; g.add(lid);
  const cap = new THREE.Mesh(new THREE.CylinderGeometry(0.08, 0.115, 0.05, 20), bur); cap.position.y = 0.5 - GR; g.add(cap);
  const tex = sim.canvasTex(128, 64, (c, w, h) => { c.fillStyle = '#e9cfd3'; c.textAlign = 'center'; c.textBaseline = 'middle'; c.font = 'bold 46px "Helvetica Neue", Arial, sans-serif'; c.fillText('½M', w / 2, h / 2 + 2); });
  const label = new THREE.Mesh(new THREE.PlaneGeometry(0.17, 0.085), new THREE.MeshBasicMaterial({ map: tex, transparent: true, alphaTest: 0.1 })); label.position.set(0, 0.12 - GR, 0.118); g.add(label);
  g.position.set(0, -0.085, 0.15); // di telapak (sisi +z pergelangan = arah telapak)
  return g;
}
const _wq = new THREE.Quaternion(), _dq = new THREE.Quaternion(), _UP = new THREE.Vector3(0, 1, 0);
function updateOrder(dt) {
  if (!order || !sim.chibi) return;
  const near = !P.layer && Math.hypot(P.x - order.x, P.z - order.z) < order.r + 0.4;
  // lingkaran berdenyut; memudar saat pemain berdiri di atasnya (tidak mengganggu foto), kembali saat pergi
  orderT += dt; const pul = Math.sin(orderT * 3.2), m = order.mesh.material; m.opacity += ((near ? 0 : 0.78 + 0.22 * pul) - m.opacity) * Math.min(1, dt * 8);
  const sc = 1 + 0.07 * pul; order.mesh.scale.set(sc, sc, 1); order.mesh.visible = m.opacity > 0.02;
  if (near && !cup) { cup = makeTumbler(); sim.chibi.rig.wrists[0].add(cup); $('simToast').style.zIndex = 60; sim.toast('☕ Tumbler ½M di tangan! Ambil tangkapan layar 📸'); }
  if (!near && cup) { cup.parent.remove(cup); cup = null; }
  if (cup && moving < 0.3) { S.gesture = 0.3; S.gestureKind = 'cup'; sim.face(view.yaw, dt); } // berpose menghadap kamera
  if (cup) { // tumbler selalu tegak, tulisan ½M menghadap arah karakter (= kamera saat berpose)
    cup.parent.getWorldQuaternion(_wq); _dq.setFromAxisAngle(_UP, sim.chibi.g.rotation.y); cup.quaternion.copy(_wq).invert().multiply(_dq);
  }
}
hooks.update = (dt) => {
  const c = controls.read(dt);
  view.yaw += c.yaw; view.pitch = THREE.MathUtils.clamp(view.pitch + c.pitch, 0.05, 1.2);
  const ch = sim.chibi;
  if (ch && S.phase === 'roam') {
    const s = Math.sin(view.yaw), co = Math.cos(view.yaw);
    const speed = c.run ? RUN : WALK;
    const vx = (co * c.x - s * c.y) * speed, vz = (-s * c.x - co * c.y) * speed;
    const ox = P.x, oz = P.z;
    if (vx || vz) colOf().move(P, vx * dt, vz * dt, RADIUS);
    const moved = Math.hypot(P.x - ox, P.z - oz) / Math.max(dt, 1e-4);
    moving = moved;
    // eskalator rooftop: naik ke atas = pindah ke lapisan rooftop, turun sampai bawah = kembali ke lantai dasar
    const up = world.upWalk(P.x, P.z, P.layer); // di rooftop hanya bagian di dalam sumur (atap di atas sisa tanjakan = lantai atap)
    if (up) P.layer = up.y > world.ROOF_TOP - 1 ? 1 : 0; // 1 m di bawah atap: pindah lapisan (naik maupun turun)
    P.y = world.floorY(P.x, P.z, P.layer);
    ch.g.position.set(P.x, P.y + FOOT, P.z);
    if (vx || vz) sim.face(Math.atan2(vx, vz), dt);
    sim.animate(dt, moved > 0.3 ? moved : 0, { run: c.run && moved > 0.3 });
    updateOrder(dt);
    zoneTick -= dt; if (zoneTick <= 0) { zoneTick = 0.25; updateZone(); }
  }
  // Label: hanya yang dekat
  const px = ch ? P.x : 0, pz = ch ? P.z : 0;
  for (const l of labels) { const d = Math.hypot(l.pos.x - px, l.pos.z - pz); l.el.visible = d < l.far && !(l.hideNear && d < l.hideNear); }
  for (const r of world.roofs) r.visible = !cam.top || P.layer === 1;
  crowd.update(dt, camera, P);
  syncXray();
  drawMap();
};
const _p = new THREE.Vector3(), _t = new THREE.Vector3();
// ---------- Kamera orang ketiga ----------
// Arah kamera hanya diatur pemain (geser layar / Q-E). Penghalang (punya tinggi) di garis pandang ke kepala karakter:
//  1. benda rendah yang tetap di bawah garis pandang ke leher (HEAD) -> diabaikan;
//  2. selain itu kamera NAIK secukupnya (sudut dicari dari bawah) supaya kepala terlihat dari atas penghalang;
//  3. bila sampai sudut maksimum masih tertutup -> kamera maju ke depan penghalang.
// Di bawah atap (aula, atau mataf dengan kamera di atas serambi) kamera tidak melewati plafon: naik ke dekat plafon
// sambil mendekat mendatar. Semua tinggi relatif ke lantai pemain (P.y): pelataran satu lantai di atas lantai
// masjid, eskalator di antaranya. Jarak & sudut dihaluskan (camS). Karakter yang tertutup tiang tampil sebagai siluet.
const HEAD = 1.6, PMAX = 1.55, crossBuf = [], crossBufR = []; // PMAX ~89°: dari atas kepala garis pandang tak pernah terhalang
const CAM_CEIL = ROOF_Y - 3.8; // di bawah atap kamera tidak lebih tinggi dari ini (di bawah pita lengkung dekat plafon)
const camS = { k: 1, pitch: 0.42, space: 0, pend: 0, cut: false };
hooks.camera = (dt = 0) => {
  if (!sim.chibi) { // pratinjau sebelum memilih karakter: Ka'bah dari tenggara atas
    const a = performance.now() * 0.00004;
    return { pos: _p.set(Math.sin(a) * 170, 120, Math.cos(a) * 170), tgt: _t.set(0, 0, 0) };
  }
  if (cam.top) return { pos: _p.set(P.x, P.y + 150 * cam.zoom, P.z + 0.01), tgt: _t.set(P.x, P.y, P.z) };
  // Jarak seperti kamera tawaf (13-19 m) x zoom; layar tegak (HP) sedikit lebih jauh.
  const dist = (camera.aspect < 1 ? 17 : 14) * cam.zoom, y0 = P.y, eye = y0 + HEAD;
  const sy = Math.sin(view.yaw), cy = Math.cos(view.yaw);
  const at = (k, pitch) => { const hor = Math.cos(pitch) * dist * k; return [P.x + sy * hor, y0 + 2 + Math.sin(pitch) * dist * k, P.z + cy * hor]; };
  let cr = col.crossings(P.x, P.z, P.x + sy * dist, P.z + cy * dist, crossBuf);
  if (P.layer) { // rooftop: pagar atap + yang menjulang di atas atap (gedung tinggi, portal, menara)
    const hiR = cr.filter((c) => c.h > world.ROOF_TOP + 0.5);
    cr = world.colR.crossings(P.x, P.z, P.x + sy * dist, P.z + cy * dist, crossBufR).concat(hiR).sort((a, b) => a.u - b.u);
  }
  // Jarak datar penghalang pertama yang menutupi kepala bila kamera di (k, pitch); -1 = terlihat.
  // Kamera di atas atap masjid: dinding luar menghalangi sampai puncaknya (tidak bisa mengintip lewat celah pintu,
  // karena atap ada di antaranya).
  const blockedAt = (k, pitch) => {
    const Hc = Math.cos(pitch) * dist * k, yc = y0 + 2 + Math.sin(pitch) * dist * k;
    const overRoof = yc > CAM_CEIL && world.region.at(P.x + sy * Hc, P.z + cy * Hc) === REG.HALL;
    for (const c of cr) {
      const d = c.u * dist; if (d >= Hc) break;
      const y = eye + (yc - eye) * d / Hc;
      if (y < c.h && (y >= c.lo || (overRoof && c.tag === 'wall'))) return d;
    }
    // Lempeng atap aula: garis pandang yang menembusnya di luar sumur eskalator tertutup (naik eskalator di dalam aula).
    if ((yc > ROOF_Y) !== (eye > ROOF_Y)) {
      const f = (ROOF_Y + 0.4 - eye) / (yc - eye), x = P.x + sy * Hc * f, z = P.z + cy * Hc * f;
      if (f > 0 && f < 1 && world.region.at(x, z) === REG.HALL && !world.upWalk(x, z, 1, 0.1)) return Hc * f;
    }
    // Kamera sendiri di dalam lempeng atap (di luar sumur) juga tertutup.
    if (yc > ROOF_Y - 0.3 && yc < world.ROOF_TOP + 0.3) { const x = P.x + sy * Hc, z = P.z + cy * Hc; if (world.region.at(x, z) === REG.HALL && !world.upWalk(x, z, 1, 0.1)) return Hc; }
    // Kamera tidak boleh di dalam blok eskalator: harus >= 1,2 m di atas permukaan tanjakan di posisinya.
    const sr = world.upAt(P.x + sy * Hc, P.z + cy * Hc); if (sr && yc < sr.y + 1.2) return Hc;
    return -1;
  };
  const rideHit = P.layer ? null : world.upWalk(P.x, P.z, 0), ride = !!rideHit; // sedang di eskalator rooftop (di dalam aula)
  const camReg = (k, p) => { const hor = Math.cos(p) * dist * k; return world.region.at(P.x + sy * hor, P.z + cy * hor); };
  // Ruang karakter: pelataran (0) atau masjid (1, aula & mataf); jeda 0,3 dtk supaya tidak bolak-balik di ambang pintu.
  // Di aula kamera selalu di dalam gedung, di bawah plafon.
  // Masuk lewat bordes & eskalator: kamera tetap di luar (di belakang, di atas pelataran) memandang lewat bukaan pintu
  // selama kepala terlihat; sudut boleh diturunkan sedikit. Pindah ke dalam begitu tidak terlihat lagi / sampai bawah.
  let doorPitch = -1;
  if (!P.layer && camS.space === 0 && (world.nearRamp(P.x, P.z, 0) || world.portalAt(P.x, P.z))) {
    for (let p = view.pitch; p >= 0.08; p -= 0.03) {
      const yc = y0 + 2 + Math.sin(p) * dist;
      const hx = P.x + sy * Math.cos(p) * dist, hz = P.z + cy * Math.cos(p) * dist;
      if (yc >= world.floorY(hx, hz) + 1.5 && camReg(1, p) === REG.OUT && blockedAt(1, p) < 0) { doorPitch = p; break; }
    }
  }
  const regNow = P.layer || ride || doorPitch >= 0 ? 0 : world.region.at(P.x, P.z) === REG.OUT && !world.portalAt(P.x, P.z) ? 0 : 1; // lorong portal beratap = dalam; rooftop & eskalator = luar
  if (regNow !== camS.space) { camS.pend = (camS.pend || 0) + dt; if (!dt || camS.pend > 0.3 || (regNow === 1 && world.nearRamp(P.x, P.z, 0))) { camS.space = regNow; camS.pend = 0; camS.cut = true; } } else camS.pend = 0;
  const indoor = camS.space === 1 && !inMataf(P.x, P.z); // selama jeda di ambang pintu tetap aturan dalam
  const ceil = CAM_CEIL - y0 - 2; // tinggi kamera maksimum di bawah atap (di atas titik dasar y0 + 2)
  const kAt = (p) => Math.min(1, ceil / (Math.max(0.05, Math.sin(p)) * dist));
  let k = 1, pitch = view.pitch, camUp = true;
  const roofed = !ride && (indoor || (inMataf(P.x, P.z) && camReg(1, view.pitch) === REG.HALL));
  if (roofed) {
    // Di bawah atap: sudut diturunkan supaya kamera di bawah plafon. Bila garis pandang tertutup (dinding di
    // belakang, pagar eskalator, dsb.) atau kamera akan keluar gedung, kamera NAIK ke dekat plafon sambil mendekat
    // mendatar secukupnya, sampai hampir tegak di atas karakter. Berubah mulus; tidak pernah maju ke kepala.
    pitch = Math.min(pitch, Math.asin(THREE.MathUtils.clamp(ceil / dist, -1, 1)));
    const okAt = (kk, p) => blockedAt(kk, p) < 0 && (!indoor || camReg(kk, p) !== REG.OUT);
    if (!okAt(k, pitch)) {
      let p = pitch, kk = k, ok = false;
      while (p < PMAX) { p = Math.min(PMAX, p + 0.02); kk = kAt(p); if (okAt(kk, p)) { ok = true; break; } }
      pitch = ok ? p : PMAX; k = ok ? kk : kAt(PMAX);
    }
  } else if (ride) {
    // Di eskalator rooftop (dalam aula). Kamera di sisi BAWAH tanjakan (mis. sedang naik, kamera di belakang): tetap di
    // bawah lempeng atap (sudut diturunkan, boleh negatif) dan mendekat bila garis pandang menembus atap di luar sumur.
    // Kamera di sisi ATAS tanjakan (mis. sedang turun): aturan udara terbuka, naik sampai di atas permukaan tanjakan
    // dan melihat lewat sumur.
    const rr = rideHit.r; camUp = sy * rr.n[0] + cy * rr.n[1] > 0;
    if (camUp) { while (pitch < PMAX && blockedAt(1, pitch) >= 0) pitch = Math.min(PMAX, pitch + 0.03); }
    else {
      const lid = ROOF_Y - 0.8 - (y0 + 2); // tinggi kamera maksimum relatif titik dasar (negatif dekat atap)
      const pk = (kk) => Math.min(view.pitch, Math.asin(THREE.MathUtils.clamp(lid / (dist * kk), -0.6, 1))); // sudut agar kamera tepat di batas
      pitch = pk(k);
      while (k > 0.3 && blockedAt(k, pitch) >= 0) { k -= 0.05; pitch = pk(k); }
    }
  } else if (doorPitch >= 0) {
    pitch = doorPitch; // di bordes/eskalator: dari luar lewat bukaan pintu
  } else {
    // Udara terbuka: naik secukupnya sampai kepala terlihat (paling tinggi hampir tegak di atas karakter).
    while (pitch < PMAX && blockedAt(1, pitch) >= 0) pitch = Math.min(PMAX, pitch + 0.03);
  }
  if (camS.cut) dt = 0; // melewati ambang pintu: kamera langsung pindah, tidak melayang menembus dinding
  if (dt) {
    camS.k += (k - camS.k) * Math.min(1, dt * (k < camS.k ? 3 : 1.4));
    camS.pitch += (pitch - camS.pitch) * Math.min(1, dt * (pitch > camS.pitch ? 7 : 1.5));
  } else { camS.k = k; camS.pitch = pitch; }
  const pos = at(camS.k, camS.pitch);
  if (roofed) pos[1] = Math.min(pos[1], y0 + 2 + ceil); // peralihan halus pun tetap di bawah plafon
  if (ride && !camUp) pos[1] = Math.min(pos[1], ROOF_Y - 0.8); // naik eskalator: kamera (yang tertinggal) tetap di bawah atap
  _p.set(pos[0], pos[1], pos[2]); _t.set(P.x, y0 + 2, P.z);
  if (camS.cut) { camS.cut = false; cam.pos.copy(_p); cam.tgt.copy(_t); }
  return { pos: _p, tgt: _t };
};

// ---------- Mini-peta ----------
const mapCv = $('haramMap'), mctx = mapCv.getContext('2d');
const [bx0, bz0, bx1, bz1] = data.bounds, span = Math.max(bx1 - bx0, bz1 - bz0) + 40;
const mcx = (bx0 + bx1) / 2, mcz = (bz0 + bz1) / 2;
const base = document.createElement('canvas'); base.width = base.height = 640;
{
  const g = base.getContext('2d'), S2 = base.width, T = (p) => [(p[0] - mcx) / span * S2 + S2 / 2, (p[1] - mcz) / span * S2 + S2 / 2];
  const path = (ring) => { g.beginPath(); ring.forEach((p, i) => { const [x, y] = T(p); i ? g.lineTo(x, y) : g.moveTo(x, y); }); g.closePath(); };
  g.fillStyle = '#efebe2'; g.fillRect(0, 0, S2, S2);
  g.fillStyle = '#cdc3b0'; data.buildings.forEach((b) => { if (!b.bridge) { path(b.poly); g.fill(); } });
  g.fillStyle = '#e2d8c4'; world.terraces.forEach((tr) => { const Tp = (u, o) => [tr.c[0] + tr.t[0] * u + tr.n[0] * o, tr.c[1] + tr.t[1] * u + tr.n[1] * o]; path([Tp(-tr.W, 0), Tp(tr.W, 0), Tp(tr.W, tr.D + tr.S), Tp(-tr.W, tr.D + tr.S)]); g.fill(); });
  g.fillStyle = '#ffffff'; g.strokeStyle = '#b08d57'; g.lineWidth = 2; outers.forEach((r) => { path(r); g.fill(); g.stroke(); });
  g.fillStyle = '#dfe7ea'; inners.forEach((r) => { path(r); g.fill(); });
  g.strokeStyle = '#16a34a'; g.lineWidth = 3; g.beginPath(); g.moveTo(...T(data.safa)); g.lineTo(...T(data.marwah)); g.stroke();
  g.fillStyle = '#111'; path(data.kaaba.poly); g.fill();
  data.gates.forEach((x) => { if (x.main) { const z = gateZone(x); g.fillStyle = z ? z.color : '#b91c1c'; const [a, b] = T(x.pos); g.beginPath(); g.arc(a, b, z ? 5 : 4, 0, TAU); g.fill(); } });
  g.fillStyle = '#047857'; world.upRamps.forEach((r) => { const W = (u, sv) => [r.c[0] + r.t[0] * u - r.n[0] * sv, r.c[1] + r.t[1] * u - r.n[1] * sv]; path([W(-r.hw, 0), W(r.hw, 0), W(r.hw, r.RE), W(-r.hw, r.RE)]); g.fill(); }); // eskalator ke atap
  g.fillStyle = '#d97706'; (data.landmarks || []).forEach((l) => { const [a, b] = T(l.pos); g.fillRect(a - 4, b - 4, 8, 8); });
  if (data.clockTower) { const [a, b] = T(data.clockTower.pos); g.fillStyle = '#6b5b45'; g.fillRect(a - 5, b - 5, 10, 10); }
}
const mapWrap = $('haramMapWrap'), mapBig = () => mapWrap.classList.contains('big');
function drawMap() {
  const r = mapCv.getBoundingClientRect(), dpr = Math.min(2, window.devicePixelRatio || 1);
  const W = Math.round(r.width * dpr), H = Math.round(r.height * dpr); if (!W || !H) return;
  if (mapCv.width !== W || mapCv.height !== H) { mapCv.width = W; mapCv.height = H; }
  const g = mctx; g.setTransform(1, 0, 0, 1, 0, 0); g.clearRect(0, 0, W, H);
  // Peta kecil: diperbesar 2,2x di sekitar pemain. Peta besar: seluruh area.
  const zoom = mapBig() ? 1 : 2.2, px = sim.chibi ? P.x : 0, pz = sim.chibi ? P.z : 0;
  const scale = W / span * zoom; // piksel per meter
  const ox = mapBig() ? mcx : px, oz = mapBig() ? mcz : pz;
  const ks = mapBig() ? 1 : 0.72; // peta kecil: panah, kerucut & huruf ikut mengecil
  g.save(); g.beginPath(); g.roundRect ? g.roundRect(0, 0, W, H, (mapBig() ? 16 : 12) * dpr) : g.rect(0, 0, W, H); g.clip();
  g.fillStyle = '#efebe2'; g.fillRect(0, 0, W, H);
  const bs = base.width / span; // piksel base per meter
  g.drawImage(base, ((ox - mcx) + span / 2 - W / 2 / scale) * bs, ((oz - mcz) + span / 2 - H / 2 / scale) * bs, W / scale * bs, H / scale * bs, 0, 0, W, H);
  if (sim.chibi) {
    const x = (px - ox) * scale + W / 2, y = (pz - oz) * scale + H / 2, yaw = sim.chibi.g.rotation.y;
    // kerucut pandang kamera
    // kamera memandang ke arah -(sin yaw, cos yaw); sudut kanvas diukur dari +x ke bawah (+z)
    const va = Math.atan2(-Math.cos(view.yaw), -Math.sin(view.yaw)), R = 34 * dpr * ks;
    g.fillStyle = 'rgba(4,120,87,.18)'; g.beginPath(); g.moveTo(x, y); g.arc(x, y, R, va - 0.55, va + 0.55); g.closePath(); g.fill();
    g.translate(x, y); g.rotate(-yaw + Math.PI);
    g.fillStyle = '#047857'; g.strokeStyle = '#fff'; g.lineWidth = 2 * dpr * ks; const a = dpr * ks;
    g.beginPath(); g.moveTo(0, -8 * a); g.lineTo(6 * a, 6 * a); g.lineTo(0, 3 * a); g.lineTo(-6 * a, 6 * a); g.closePath(); g.fill(); g.stroke();
  }
  g.restore();
  g.fillStyle = 'rgba(17,24,39,.75)'; g.font = `bold ${(mapBig() ? 12 : 8) * dpr}px system-ui`; g.fillText('U ↑', (mapBig() ? 12 : 5) * dpr, (mapBig() ? 20 : 11) * dpr);
}
// Ketuk peta kecil = perbesar. Ketuk peta besar = pindah ke titik itu (bila bukan di dalam gedung).
mapCv.addEventListener('click', (e) => {
  if (!mapBig()) { mapWrap.classList.add('big'); $('haramMapClose').hidden = false; $('haramMapHint').hidden = false; return; }
  if (!sim.chibi) return;
  const r = mapCv.getBoundingClientRect(), scale = r.width / span;
  const x = (e.clientX - r.left - r.width / 2) / scale + mcx, z = (e.clientY - r.top - r.height / 2) / scale + mcz;
  const blocked = data.buildings.some((b) => !b.bridge && inPoly(x, z, b.poly)) || Math.hypot(x, z) < 8 || Math.hypot(x, z) > world.boundR;
  if (blocked) { sim.toast('Tidak bisa ke sana. Pilih tempat terbuka.'); return; }
  closeMap(); teleport(x, z, sim.chibi.g.rotation.y);
});
function closeMap() { mapWrap.classList.remove('big'); $('haramMapClose').hidden = true; $('haramMapHint').hidden = true; }
$('haramMapClose').onclick = closeMap;

// ---------- Mulai ----------
sim.start({ pos: new THREE.Vector3(0, 120, 170), tgt: new THREE.Vector3(0, 0, 0) });
Object.assign(window.__sim, {
  P, view, col, data, world, crowd, xray: () => xray,
  goto(x, z, face = 0, layer = 0) { teleport(x, z, face, layer); },
  // Jalan dengan joystick maya (x kanan, y maju) selama `sec` detik simulasi.
  walk(x, y, sec) { controls.st.jx = x; controls.st.jy = y; window.__sim.tick(sec); controls.st.jx = controls.st.jy = 0; },
});
