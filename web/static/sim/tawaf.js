// Simulasi tawaf Umrohku. Semua bentuk dibuat dari kode (tanpa model 3D), supaya ringan di HP.
// Arah: x = timur, -z = utara. Sudut θ diukur berlawanan jarum jam dari timur (dilihat dari atas),
// jadi tawaf = θ bertambah, dan Ka'bah selalu di sebelah kiri.
import { createSim, THREE, $, TAU, mod } from './common.js';

const angOf = (x, z) => Math.atan2(-z, x);
const atAngle = (a, r, y = 0) => new THREE.Vector3(Math.cos(a) * r, y, -Math.sin(a) * r);

const sim = createSim({ part: 'tawaf' });
const { scene, camera, S, input, cam, M, mat, canvasTex, instanced, hooks } = sim;

const R = 19; // jari-jari lintasan tawaf karakter
// ---------- Lantai mataf (lingkaran-lingkaran seperti foto) ----------
const floorTex = canvasTex(1024, 1024, (g, s) => {
  g.fillStyle = '#f3f3f0'; g.fillRect(0, 0, s, s);
  for (let i = 0; i < 3000; i++) { g.fillStyle = `rgba(90,90,80,${Math.random() * 0.03})`; g.fillRect(Math.random() * s, Math.random() * s, 2 + Math.random() * 7, 1 + Math.random() * 2); }
  g.strokeStyle = 'rgba(140,145,150,.45)'; g.lineWidth = 2;
  for (let r = 40; r < 512; r += 22) { g.beginPath(); g.arc(s / 2, s / 2, r, 0, TAU); g.stroke(); }
});
const floor = new THREE.Mesh(new THREE.CircleGeometry(52, 96), new THREE.MeshLambertMaterial({ map: floorTex }));
floor.rotation.x = -Math.PI / 2;
scene.add(floor);
const ground = new THREE.Mesh(new THREE.CircleGeometry(200, 48), M.stone);
ground.rotation.x = -Math.PI / 2; ground.position.y = -0.05;
scene.add(ground);

// ---------- Masjid (oktagon sederhana, instanced) ----------
const octV = (i, ap) => { const R = ap / Math.cos(Math.PI / 8), a = i * Math.PI / 4 + Math.PI / 8; return new THREE.Vector2(R * Math.cos(a), R * Math.sin(a)); };
const octShape = (ap) => { const s = new THREE.Shape(); for (let i = 0; i < 8; i++) { const v = octV(i, ap); i ? s.lineTo(v.x, v.y) : s.moveTo(v.x, v.y); } s.closePath(); return s; };
function octRing(apOut, apIn, height, material, y) {
  const sh = octShape(apOut); sh.holes.push(new THREE.Path(octShape(apIn).getPoints()));
  const g = new THREE.ExtrudeGeometry(sh, { depth: height, bevelEnabled: false, curveSegments: 1 });
  g.rotateX(-Math.PI / 2);
  const m = new THREE.Mesh(g, material); m.position.y = y; scene.add(m); return m;
}
// titik-titik sepanjang keliling oktagon (dalam koordinat dunia x,z) + arah sisi
function octPoints(ap, spacing) {
  const out = [];
  for (let i = 0; i < 8; i++) {
    const a = octV(i, ap), b = octV(i + 1, ap), len = a.distanceTo(b), n = Math.max(1, Math.round(len / spacing));
    for (let k = 0; k < n; k++) {
      const t = (k + 0.5) / n;
      out.push({ x: a.x + (b.x - a.x) * t, z: -(a.y + (b.y - a.y) * t), dir: Math.atan2(-(b.y - a.y), b.x - a.x), side: i, k, n });
    }
  }
  return out;
}
// Serambi Utsmani: tiang, lengkung, atap, kubah
const COL_AP = 49, COL_H = 4.6, SP = 3.4;
const cols = octPoints(COL_AP, SP);
instanced(new THREE.CylinderGeometry(0.32, 0.38, COL_H, 8), M.pillar, cols, (o, p) => o.position.set(p.x, COL_H / 2, p.z));
instanced(new THREE.TorusGeometry(SP / 2 - 0.1, 0.2, 5, 12, Math.PI), M.arch, cols, (o, p) => {
  if (p.k === p.n - 1) { o.scale.setScalar(0); return; } // lengkung hanya di antara dua tiang pada sisi yang sama
  o.position.set(p.x + Math.cos(p.dir) * SP / 2, COL_H - 0.9, p.z + Math.sin(p.dir) * SP / 2); o.rotation.y = -p.dir;
});
octRing(COL_AP + 7, COL_AP - 0.6, 0.6, M.stone, COL_H);
const domeGeo = new THREE.SphereGeometry(1.25, 12, 6, 0, TAU, 0, Math.PI / 2);
instanced(domeGeo, M.dome, [...octPoints(COL_AP + 1.8, 3.2), ...octPoints(COL_AP + 4.8, 3.4)], (o, p) => o.position.set(p.x, COL_H + 0.6, p.z));
// Gedung perluasan: pilar tinggi dengan bukaan gelap, teras atas, lampu, dinding luar
const B_AP = 57;
octRing(B_AP + 6, B_AP + 5, 12, M.dark, 0);
instanced(new THREE.BoxGeometry(1.6, 12, 1.6), M.pillar, octPoints(B_AP, 5.2), (o, p) => { o.position.set(p.x, 6, p.z); o.rotation.y = -p.dir; });
octRing(B_AP + 6, B_AP - 0.8, 0.8, M.stone, 5.6);
octRing(80, B_AP - 0.8, 0.8, M.stone, 12);
octRing(B_AP - 0.2, B_AP - 0.8, 1, M.stoneDark, 12.8);
const poles = octPoints(B_AP + 4, 7);
instanced(new THREE.CylinderGeometry(0.08, 0.1, 4.5, 6), M.silver, poles, (o, p) => o.position.set(p.x, 15, p.z));
instanced(new THREE.BoxGeometry(0.45, 0.25, 0.45), M.white, poles, (o, p) => o.position.set(p.x, 17.3, p.z));
octRing(81, 78.5, 5, M.stone, 12.8);
octRing(81.2, 78.3, 1.1, M.tile, 16.6);

// ---------- Ka'bah ----------
const KW = 10, KD = 12, KH = 13.1, KROT = Math.PI / 4;
const kaaba = new THREE.Group(); kaaba.rotation.y = KROT; scene.add(kaaba);
const toWorld = (x, y, z) => new THREE.Vector3(x, y, z).applyAxisAngle(new THREE.Vector3(0, 1, 0), KROT);
const base = new THREE.Mesh(new THREE.BoxGeometry(KW + 0.9, 0.5, KD + 0.9), mat('#dad6cc')); base.position.y = 0.25; kaaba.add(base);
const body = new THREE.Mesh(new THREE.BoxGeometry(KW, KH, KD), M.black); body.position.y = 0.5 + KH / 2; kaaba.add(body);
const bandTex = canvasTex(1024, 64, (g, w, h) => {
  g.fillStyle = '#151515'; g.fillRect(0, 0, w, h);
  g.strokeStyle = '#d6b04a'; g.lineWidth = 3; g.lineCap = 'round';
  g.beginPath(); g.moveTo(0, 6); g.lineTo(w, 6); g.moveTo(0, h - 6); g.lineTo(w, h - 6); g.stroke();
  g.lineWidth = 2.4;
  for (let x = 14; x < w - 14; x += 9 + Math.random() * 8) {
    g.beginPath(); const y = 20 + Math.random() * 14;
    g.moveTo(x, y); g.quadraticCurveTo(x + 4, y - 10 + Math.random() * 20, x + 8 + Math.random() * 6, 22 + Math.random() * 18); g.stroke();
  }
});
bandTex.wrapS = THREE.RepeatWrapping; bandTex.repeat.set(2, 1);
const band = new THREE.Mesh(new THREE.BoxGeometry(KW + 0.06, 1.1, KD + 0.06), new THREE.MeshLambertMaterial({ map: bandTex }));
band.position.y = 0.5 + KH * 0.68; kaaba.add(band);
// Pintu di sisi timur laut (sisi +x lokal), dekat sudut Hajar Aswad
const door = new THREE.Mesh(new THREE.BoxGeometry(0.15, 3.4, 2), M.gold); door.position.set(KW / 2 + 0.05, 0.5 + 2.2 + 1.7, 3); kaaba.add(door);
// Sudut-sudut (dunia): Hajar Aswad = timur, Iraqi = utara, Syami = barat, Yamani = selatan
const C_HA = toWorld(KW / 2, 0, KD / 2), C_YM = toWorld(-KW / 2, 0, KD / 2);
const A_HA = angOf(C_HA.x, C_HA.z), A_YM = angOf(C_YM.x, C_YM.z);
// Hajar Aswad: bingkai perak di sudut, setinggi ±1,5 m
const haDir = C_HA.clone().setY(0).normalize();
const haFrame = new THREE.Group();
haFrame.position.copy(C_HA.clone().add(haDir.clone().multiplyScalar(0.35))).setY(1.6);
haFrame.rotation.y = Math.atan2(haDir.x, haDir.z);
haFrame.add(new THREE.Mesh(new THREE.TorusGeometry(0.5, 0.14, 8, 22), M.silver));
const haStone = new THREE.Mesh(new THREE.CircleGeometry(0.42, 20), new THREE.MeshStandardMaterial({ color: '#2a1d1d', emissive: '#3a2020', roughness: 0.4 }));
haFrame.add(haStone); scene.add(haFrame);
// Rukun Yamani: penanda tipis di sudut
const ymDir = C_YM.clone().setY(0).normalize();
const ymMark = new THREE.Mesh(new THREE.BoxGeometry(0.5, 2.2, 0.5), mat('#3a3a3a'));
ymMark.position.copy(C_YM.clone().add(ymDir.clone().multiplyScalar(0.05))).setY(1.6); ymMark.rotation.y = KROT; scene.add(ymMark);

// Hijr Ismail: tembok rendah setengah elips di sisi barat laut (sisi -z lokal)
const hijrPts = [];
for (let i = 0; i <= 26; i++) {
  const t = Math.PI * (i / 26);
  hijrPts.push({ x: Math.cos(t) * (KW / 2 + 0.6), z: -KD / 2 - 0.4 - Math.sin(t) * 8.4 });
}
const hijrSeg = [];
for (let i = 0; i < hijrPts.length - 1; i++) {
  const a = hijrPts[i], b = hijrPts[i + 1];
  if (i === 0 || i === hijrPts.length - 2) continue; // celah masuk di kedua ujung
  hijrSeg.push({ a, b });
}
{
  const im = new THREE.InstancedMesh(new THREE.BoxGeometry(1, 1.3, 1.4), M.marble, hijrSeg.length), o = new THREE.Object3D();
  hijrSeg.forEach((s, i) => {
    const dx = s.b.x - s.a.x, dz = s.b.z - s.a.z;
    o.position.set((s.a.x + s.b.x) / 2, 0.65, (s.a.z + s.b.z) / 2); o.rotation.set(0, -Math.atan2(dz, dx), 0); o.scale.set(Math.hypot(dx, dz) + 0.1, 1, 1);
    o.updateMatrix(); im.setMatrixAt(i, o.matrix);
  });
  kaaba.add(im);
  const shape = new THREE.Shape(); shape.moveTo(hijrPts[0].x, -hijrPts[0].z); hijrPts.forEach((p) => shape.lineTo(p.x, -p.z)); shape.closePath();
  const hf = new THREE.Mesh(new THREE.ShapeGeometry(shape), mat('#e4e2dc')); hf.rotation.x = -Math.PI / 2; hf.position.y = 0.03; kaaba.add(hf);
}
const HIJR_MID = toWorld(0, 0, -KD / 2 - 6);

// Maqam Ibrahim: kubah kecil keemasan di depan pintu
const maqam = new THREE.Group();
const MQ = toWorld(KW / 2 + 9, 0, 2.6); maqam.position.copy(MQ); maqam.rotation.y = KROT;
const mqBase = new THREE.Mesh(new THREE.CylinderGeometry(1.2, 1.3, 0.5, 8), mat('#e3dfd6')); mqBase.position.y = 0.25; maqam.add(mqBase);
const mqCage = new THREE.Mesh(new THREE.CylinderGeometry(0.85, 0.85, 1.9, 8, 1, true), new THREE.MeshStandardMaterial({ color: '#d4a93a', metalness: 0.5, roughness: 0.4, side: THREE.DoubleSide, transparent: true, opacity: 0.85 }));
mqCage.position.y = 1.45; maqam.add(mqCage);
const mqDome = new THREE.Mesh(new THREE.SphereGeometry(0.9, 12, 8, 0, TAU, 0, Math.PI / 2), M.gold); mqDome.position.y = 2.4; maqam.add(mqDome);
const mqTop = new THREE.Mesh(new THREE.ConeGeometry(0.12, 0.6, 6), M.gold); mqTop.position.y = 3.5; maqam.add(mqTop);
scene.add(maqam);

// Lampu hijau penanda start (sejajar Hajar Aswad), garis start di lantai
const LAMP = atAngle(A_HA, COL_AP - 1.6);
const lampPole = new THREE.Mesh(new THREE.CylinderGeometry(0.12, 0.14, 4.2, 8), M.silver); lampPole.position.copy(LAMP).setY(2.1); scene.add(lampPole);
const lampHead = new THREE.Mesh(new THREE.BoxGeometry(0.9, 0.6, 0.9), new THREE.MeshBasicMaterial({ color: '#16a34a' })); lampHead.position.copy(LAMP).setY(4.4); scene.add(lampHead);
const glowTex = canvasTex(128, 128, (g, s) => { const gr = g.createRadialGradient(s / 2, s / 2, 0, s / 2, s / 2, s / 2); gr.addColorStop(0, 'rgba(74,222,128,1)'); gr.addColorStop(1, 'rgba(74,222,128,0)'); g.fillStyle = gr; g.fillRect(0, 0, s, s); });
const glow = new THREE.Sprite(new THREE.SpriteMaterial({ map: glowTex, blending: THREE.AdditiveBlending, depthWrite: false, transparent: true }));
glow.position.copy(LAMP).setY(4.4); glow.scale.set(5, 5, 1); scene.add(glow);
{
  const r0 = 9, r1 = COL_AP - 2, line = new THREE.Mesh(new THREE.PlaneGeometry(r1 - r0, 0.35), new THREE.MeshBasicMaterial({ color: '#16a34a', transparent: true, opacity: 0.45, depthWrite: false }));
  line.rotation.x = -Math.PI / 2; line.rotation.z = A_HA; line.position.copy(atAngle(A_HA, (r0 + r1) / 2, 0.04)); scene.add(line);
}

// ---------- Jamaah lain (kapsul sederhana, satu draw call) ----------
const CROWD = 110, crowd = [];
const crowdBody = new THREE.InstancedMesh(new THREE.CapsuleGeometry(0.34, 0.75, 2, 8), new THREE.MeshLambertMaterial({ color: '#ffffff' }), CROWD);
const crowdHead = new THREE.InstancedMesh(new THREE.SphereGeometry(0.24, 8, 6), new THREE.MeshLambertMaterial({ color: '#ffffff' }), CROWD);
const skinTones = ['#f1c7a1', '#d9a47c', '#a8754f', '#7a5236'];
for (let i = 0; i < CROWD; i++) {
  let r = 16 + Math.random() * 18; if (Math.abs(r - R) < 1.8) r += 3.8;
  const woman = Math.random() < 0.35, abaya = woman && Math.random() < 0.6;
  crowd.push({ a: Math.random() * TAU, r, v: (0.5 + Math.random() * 0.6) / r });
  crowdBody.setColorAt(i, new THREE.Color(abaya ? '#2f2f33' : woman ? '#e9e3d6' : '#fbfbf6'));
  crowdHead.setColorAt(i, new THREE.Color(woman ? (abaya ? '#2f2f33' : '#e2dccd') : skinTones[i % 4]));
}
scene.add(crowdBody, crowdHead);
const _o = new THREE.Object3D();
function updateCrowd(dt) {
  crowd.forEach((c, i) => {
    c.a += c.v * dt;
    _o.position.copy(atAngle(c.a, c.r, 0.72)); _o.updateMatrix(); crowdBody.setMatrixAt(i, _o.matrix);
    _o.position.y = 1.45; _o.updateMatrix(); crowdHead.setMatrixAt(i, _o.matrix);
  });
  crowdBody.instanceMatrix.needsUpdate = true; crowdHead.instanceMatrix.needsUpdate = true;
}

// ---------- Kondisi simulasi ----------
const START = A_HA - 0.32;  // berdiri sedikit sebelum garis start
const BASE_SPEED = 6.2;     // m/detik (dipercepat supaya 1 putaran ±20 detik)
const relOf = (a) => mod(a - A_HA, TAU);
const REL_DOOR = 0.45, REL_HIJR = relOf(angOf(HIJR_MID.x, HIJR_MID.z)) - 0.25, REL_YAMANI = relOf(A_YM) - 0.06;

Object.assign(S, { theta: START, travel: 0, done: 0, fired: {}, toFinal: null });

// ---------- Label ----------
hooks.occluder = body;
[
  ['Hajar Aswad', C_HA.clone().setY(4.2), 'hl'], ['Rukun Yamani', C_YM.clone().setY(4.2)], ['Hijr Ismail', HIJR_MID.clone().setY(2.6)],
  ['Maqam Ibrahim', MQ.clone().setY(4.6)], ['Lampu hijau · Start', LAMP.clone().setY(6.2), 'hl'],
].forEach(([t, p, c]) => sim.addLabel(t, p, c));
const { say, info, pauseWith, toast, setCounter: setRound } = sim;
const chibi = () => sim.chibi;

// ---------- UI pemandu ----------
const TAKBIR = { ar: 'بِسْمِ اللهِ، اللهُ أَكْبَرُ', lat: 'Bismillāhi Allāhu akbar', arti: 'Dengan nama Allah, Allah Maha Besar.' };
const SAPU_JAGAD = { ar: 'رَبَّنَا آتِنَا فِي الدُّنْيَا حَسَنَةً وَفِي الْآخِرَةِ حَسَنَةً وَقِنَا عَذَابَ النَّارِ', lat: "Rabbanā ātinā fid-dunyā ḥasanah, wa fil-ākhirati ḥasanah, wa qinā 'adzāban-nār.", arti: 'Ya Tuhan kami, berilah kami kebaikan di dunia dan kebaikan di akhirat, dan lindungilah kami dari azab neraka. (QS. Al-Baqarah: 201)' };
const MAQAM_AYAT = { ar: 'وَاتَّخِذُوا مِنْ مَقَامِ إِبْرَاهِيمَ مُصَلًّى', lat: 'Wattakhiżū mim maqāmi Ibrāhīma muṣallā.', arti: 'Dan jadikanlah sebagian Maqam Ibrahim tempat shalat. (QS. Al-Baqarah: 125)' };

hooks.counter = () => {
  const r = S.phase === 'tawaf' ? Math.min(7, S.done + 1) : S.phase === 'finish' ? 7 : 0;
  return { text: S.phase === 'tawaf' ? `Putaran ${r}/7` : S.phase === 'finish' ? 'Selesai 7/7' : 'Belum mulai', panel: S.phase === 'tawaf' ? `Putaran ${r} dari 7` : '', done: S.done, active: S.phase === 'tawaf' };
};

// ---------- Alur tawaf ----------
const isM = sim.isM;
hooks.canWalk = () => S.phase === 'approach' || S.phase === 'tawaf';
hooks.onBack = () => toast("Tawaf berlawanan arah jarum jam. Ka'bah selalu di sebelah kirimu.");
hooks.begin = () => {
  Object.assign(S, { phase: 'intro', theta: START, travel: 0, done: 0, fired: {}, gesture: 0, gestureKind: 'right', toFinal: null });
  pauseWith([{
    title: 'Sebelum mulai',
    body: `<p>Pastikan kamu sudah <b>berwudhu</b>, karena tawaf harus dalam keadaan suci. Niat cukup di dalam hati.</p>
           <p>Tawaf umrah terdiri dari <b>7 putaran</b>, dimulai dan diakhiri di garis <b>Hajar Aswad</b>, berlawanan arah jarum jam: <b>Ka'bah selalu di sebelah kirimu</b>.</p>` +
      (isM() ? `<p><b>Idhtiba':</b> letakkan bagian tengah kain atas di bawah ketiak kanan dan kedua ujungnya di bahu kiri, sehingga <b>bahu kanan terbuka</b>. Ini hanya selama tawaf.</p>`
        : `<p>Perempuan memakai pakaian biasa yang menutup aurat. Saat ihram, <b>wajah dan telapak tangan tidak ditutup</b> (tanpa cadar dan sarung tangan).</p>`),
    hint: 'Setelah ini, klik tombol Jalan untuk menuju lampu hijau (garis start).',
  }], () => { S.phase = 'approach'; info({ title: 'Menuju garis start', body: '<p>Klik tombol <b>Jalan</b> (atau tahan tombol panah ⬆/➡ di keyboard) untuk berjalan ke garis sejajar Hajar Aswad, ditandai <b>lampu hijau</b> di dinding masjid.</p>' }); });
};

function startLine() {
  S.theta = A_HA; S.phase = 'line';
  S.gestureKind = 'right'; S.gesture = 3.2; say(TAKBIR.ar, TAKBIR.lat, 3.2);
  pauseWith([{
    title: 'Ini garis start',
    body: `<p>Lihat <b>lampu hijau</b> di dinding kanan masjid. Hadapkan badan ke <b>Hajar Aswad</b>.</p>
           <p>Usap dan cium Hajar Aswad. Jika ramai (biasanya begitu), cukup <b>isyaratkan tangan kanan</b> ke arahnya sambil mengucapkan:</p>`,
    dua: TAKBIR,
    actions: `<button type="button" id="btnRepeat" class="rounded-xl border border-gray-200 px-3.5 lg:px-4 py-1.5 lg:py-2.5 text-[13px] lg:text-sm font-semibold text-gray-600 hover:bg-gray-50">Ulangi isyarat</button>`,
    onShow: () => { const b = $('btnRepeat'); if (b) b.onclick = () => { S.gesture = 3.2; say(TAKBIR.ar, TAKBIR.lat, 3.2); }; },
    next: 'Mulai tawaf →',
  }], () => {
    S.phase = 'tawaf'; S.travel = 0; S.done = 0; setRound();
    // Pesan singkat di atas karakter, simulasi tetap berjalan.
    sim.pops([...(isM() ? ['Raml: putaran 1 sampai 3, lelaki sunnahnya jalan cepat'] : []), 'Selama tawaf tidak ada doa khusus']);
    info({ title: 'Putaran 1', body: '<p>Klik <b>Jalan</b>. Ka\'bah di sebelah kirimu. Berdzikir dan berdoalah dengan tenang.</p>' }); });
}

function onRel(prev, rel, round) {
  const once = (key, fn) => { const k = round + ':' + key; if (!S.fired[k]) { S.fired[k] = 1; fn(); } };
  if (prev < REL_DOOR && rel >= REL_DOOR) once('door', () => {
    if (round === 1) info({ title: 'Pintu Ka\'bah & Multazam', body: `<p>Di sebelah kirimu pintu Ka'bah. Dinding di antara pintu dan Hajar Aswad disebut <b>Multazam</b>, tempat yang baik untuk berdoa.</p><p>Bangunan kecil keemasan di depannya adalah <b>Maqam Ibrahim</b>. Shalat 2 rakaat dilakukan di belakangnya setelah tawaf selesai.</p>` });
  });
  if (prev < REL_HIJR && rel >= REL_HIJR) once('hijr', () => {
    if (round === 1) pauseWith([{ title: 'Hijr Ismail', body: `<p>Tembok rendah setengah lingkaran ini adalah <b>Hijr Ismail</b>. Area di dalamnya termasuk bagian Ka'bah.</p><p>Karena itu tawaf harus <b>lewat di luarnya</b>. Jika masuk lewat celahnya dan keluar di sisi lain, putaran itu tidak sah dan harus diulang.</p>` }], () => info({ title: `Putaran ${round}`, body: '<p>Terus berjalan, Ka\'bah di sebelah kirimu.</p>' }));
    else info({ title: `Putaran ${round}`, body: '<p>Tetap di luar Hijr Ismail. Terus berdzikir dan berdoa.</p>' });
  });
  if (prev < REL_YAMANI && rel >= REL_YAMANI) once('yamani', () => {
    if (round === 1) pauseWith([
      { title: 'Rukun Yamani', body: `<p>Usap dengan tangan kanan jika bisa (tanpa dicium); jika ramai, lewati saja. Lalu sampai Hajar Aswad, bacalah:</p>`, dua: SAPU_JAGAD },
    ], () => info({ title: 'Antara Rukun Yamani dan Hajar Aswad', dua: SAPU_JAGAD, hint: 'Terus berjalan menuju garis Hajar Aswad.' }));
    else info({ title: 'Antara Rukun Yamani dan Hajar Aswad', dua: SAPU_JAGAD, hint: `Putaran ${round}. Terus berjalan menuju garis Hajar Aswad.` });
  });
}

function onLine() {
  S.done++; setRound();
  if (S.done >= 7) { finish(); return; }
  const n = S.done + 1;
  S.gesture = 1.8; say(TAKBIR.ar, TAKBIR.lat, 2.2);
  info({
    title: `Putaran ${n} dimulai`,
    body: `<p>Setiap melewati garis Hajar Aswad, isyaratkan tangan kanan ke arahnya sambil bertakbir:</p>`, dua: TAKBIR,
    after: isM() && n === 4 ? '<p><b>Mulai putaran 4, berjalan biasa.</b> Raml hanya pada putaran 1 sampai 3.</p>' : '',
    hint: `Tersisa ${7 - S.done} putaran lagi.`,
  });
}

function finish() {
  S.phase = 'finish'; S.theta = A_HA; setRound();
  const behind = toWorld(KW / 2 + 12.5, 0, 2.6);
  pauseWith([{
    title: 'Alhamdulillah, tawaf selesai!',
    body: `<p>Kamu sudah menyelesaikan <b>7 putaran</b>, berakhir di garis Hajar Aswad.</p>` + (isM() ? '<p>Laki-laki: <b>tutup kembali bahu kanan</b>. Idhtiba\' hanya selama tawaf.</p>' : ''),
    next: 'Lanjut →',
  }, {
    title: 'Shalat 2 rakaat di belakang Maqam Ibrahim',
    body: '<p>Menuju belakang Maqam Ibrahim sambil membaca:</p>', dua: MAQAM_AYAT,
    after: `<p>Shalat sunnah 2 rakaat. Disunnahkan membaca <b>Al-Kafirun</b> di rakaat pertama dan <b>Al-Ikhlas</b> di rakaat kedua. Jika area itu penuh, boleh shalat di bagian mana saja di Masjidil Haram.</p>`,
    onShow: () => { S.toFinal = behind; },
    next: 'Lanjut →',
  }, {
    title: 'Minum Zamzam, lalu menuju Shafa',
    body: '<p>Setelah shalat, minum air <b>Zamzam</b> dan berdoa. Berikutnya adalah <b>sa\'i</b>: 7 kali perjalanan antara bukit Shafa dan Marwah.</p>',
    actions: `<button type="button" id="btnRestart" class="rounded-xl border border-gray-200 px-3.5 lg:px-4 py-1.5 lg:py-2.5 text-[13px] lg:text-sm font-semibold text-gray-600 hover:bg-gray-50">Mulai Ulang</button>`,
    onShow: () => { $('btnRestart').onclick = () => sim.reset(); },
    next: "Lanjut ke Sa'i →",
  }], () => { location.href = '/simulasi?bagian=sai&g=' + S.gender; });
}

hooks.reset = () => {
  S.phase = 'choose';
  info({ title: 'Selamat datang', body: '<p>Pilih karakter untuk memulai simulasi tawaf.</p>' });
};

// ---------- Loop ----------
hooks.update = (dt) => {
  if (!chibi()) return;
  let speed = 0;
  if (sim.walking()) {
    speed = BASE_SPEED * (isM() && S.phase === 'tawaf' && S.done < 3 ? 1.3 : 1);
    const dTh = speed * dt / R, prevTheta = S.theta;
    S.theta += dTh;
    if (S.phase === 'approach' && S.theta >= A_HA && prevTheta < A_HA) startLine();
    else if (S.phase === 'tawaf') {
      const prevRel = mod(S.travel, TAU); S.travel += dTh; const rel = mod(S.travel, TAU);
      if (Math.floor(S.travel / TAU) > S.done) onLine(); else onRel(prevRel, rel, S.done + 1);
    }
  }
  // posisi & arah hadap
  let pos = atAngle(S.theta, R), face = Math.atan2(-Math.sin(S.theta), -Math.cos(S.theta)); // tangen CCW
  if (S.toFinal) {
    const cur = chibi().g.position, d = S.toFinal.clone().sub(cur); d.y = 0;
    if (d.length() > 0.15) { const step = Math.min(d.length(), 5 * dt); cur.add(d.normalize().multiplyScalar(step)); speed = 5; face = Math.atan2(d.x, d.z); pos = cur.clone(); }
    else { pos = S.toFinal.clone(); face = Math.atan2(-pos.x, -pos.z); }
  }
  if (S.gesture > 0 || S.phase === 'line') face = Math.atan2(C_HA.x - pos.x, C_HA.z - pos.z);
  chibi().g.position.copy(pos);
  sim.face(face, dt);
  sim.animate(dt, speed);
};

hooks.camera = (dt) => {
  if (!chibi() || S.phase === 'choose') {
    cam.orbit += dt * 0.08;
    return { pos: new THREE.Vector3(Math.cos(cam.orbit) * 78, 52, Math.sin(cam.orbit) * 78), tgt: new THREE.Vector3(0, 2, 0) };
  }
  if (cam.top) return { pos: new THREE.Vector3(0, 92 * cam.zoom, 46 * cam.zoom), tgt: new THREE.Vector3(0, 0, 4) };
  // Layar tegak (HP): kamera lebih jauh & fokus lebih dekat ke karakter supaya tidak keluar bingkai.
  const tall = camera.aspect < 1, cp = chibi().g.position;
  const a = (S.toFinal ? angOf(cp.x, cp.z) : S.theta) - (tall ? 0.22 : 0.42);
  return { pos: atAngle(a, R + (tall ? 19 : 13) * cam.zoom, (tall ? 21 : 16) * cam.zoom), tgt: cp.clone().multiplyScalar(tall ? 0.8 : 0.62).setY(2) };
};

// Lampu hijau berkedip & jamaah lain tetap bergerak walau karakter belum dipilih.
const baseUpdate = hooks.update;
hooks.update = (dt) => { updateCrowd(dt); glow.material.opacity = 0.55 + Math.sin(performance.now() / 380) * 0.35; baseUpdate(dt); };
sim.start({ pos: new THREE.Vector3(78, 52, 0), tgt: new THREE.Vector3(0, 2, 0) });
