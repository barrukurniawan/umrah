// Simulasi sa'i Umrohku: lorong Mas'a dari bukit Shafa (x = 0) ke bukit Marwah (x = L).
// Ka'bah berada di sisi -z (barat). Semua bentuk dibuat dari kode supaya ringan.
import { createSim, THREE, $ } from './common.js';

const sim = createSim({ part: 'sai', bg: '#efebe2', fogNear: 70, fogFar: 230 });
const { scene, camera, S, cam, M, mat, canvasTex, instanced, hooks } = sim;

const L = 150;                       // panjang lorong di simulasi (aslinya ±400 m)
const W = 26;                        // lebar lorong
const X0 = 3, X1 = L - 3;            // titik berdiri di kaki bukit Shafa / Marwah
const GZ0 = L * 0.23, GZ1 = L * 0.51; // zona lampu hijau (dipanjangkan 2x supaya doa terbaca sampai selesai)
const LANE = { toMarwah: -5.5, toSafa: 5.5 };
const CEIL = 9.6;
const BASE_SPEED = 6.2;

// ---------- Lantai, garis jalur, pembatas ----------
const floorTex = canvasTex(512, 512, (g, s) => {
  g.fillStyle = '#f4f3ef'; g.fillRect(0, 0, s, s);
  for (let i = 0; i < 1600; i++) { g.fillStyle = `rgba(110,110,100,${Math.random() * 0.035})`; g.fillRect(Math.random() * s, Math.random() * s, 2 + Math.random() * 8, 1 + Math.random() * 2); }
  g.strokeStyle = 'rgba(150,150,145,.35)'; g.lineWidth = 2;
  for (let x = 0; x <= s; x += 128) { g.beginPath(); g.moveTo(x, 0); g.lineTo(x, s); g.stroke(); g.beginPath(); g.moveTo(0, x); g.lineTo(s, x); g.stroke(); }
});
floorTex.wrapS = floorTex.wrapT = THREE.RepeatWrapping; floorTex.repeat.set((L + 40) / 8, W / 8);
const floor = new THREE.Mesh(new THREE.PlaneGeometry(L + 40, W), new THREE.MeshLambertMaterial({ map: floorTex }));
floor.rotation.x = -Math.PI / 2; floor.position.set(L / 2, 0, 0); scene.add(floor);
const lineMat = new THREE.MeshBasicMaterial({ color: '#4b4a46' });
for (const z of [-9.5, -1.8, 1.8, 9.5]) {
  const ln = new THREE.Mesh(new THREE.PlaneGeometry(L - 6, 0.22), lineMat); ln.rotation.x = -Math.PI / 2; ln.position.set(L / 2, 0.02, z); scene.add(ln);
}
// Jalur tengah (kursi roda) dengan pagar rendah
const mid = new THREE.Mesh(new THREE.PlaneGeometry(L - 6, 3.6), mat('#e9e6dd')); mid.rotation.x = -Math.PI / 2; mid.position.set(L / 2, 0.015, 0); scene.add(mid);
for (const z of [-1.8, 1.8]) { const rail = new THREE.Mesh(new THREE.BoxGeometry(L - 16, 0.08, 0.08), M.silver); rail.position.set(L / 2, 1, z); scene.add(rail); }
const railPosts = []; for (let x = 8; x <= L - 8; x += 6) for (const z of [-1.8, 1.8]) railPosts.push({ x, z });
instanced(new THREE.CylinderGeometry(0.05, 0.05, 1, 6), M.silver, railPosts, (o, p) => o.position.set(p.x, 0.5, p.z));

// ---------- Pilar, lengkung, dinding, langit-langit, lampu gantung ----------
const pillars = []; for (let x = -6; x <= L + 6; x += 6) for (const z of [-12, 12]) pillars.push({ x, z });
instanced(new THREE.BoxGeometry(1.4, CEIL, 1.4), M.pillar, pillars, (o, p) => o.position.set(p.x, CEIL / 2, p.z));
instanced(new THREE.TorusGeometry(2.9, 0.22, 5, 12, Math.PI), M.arch, pillars.filter((p) => p.x < L + 6), (o, p) => o.position.set(p.x + 3, CEIL - 3.2, p.z));
for (const z of [-13.6, 13.6]) { const wall = new THREE.Mesh(new THREE.BoxGeometry(L + 30, CEIL, 0.6), mat('#e7dcc6')); wall.position.set(L / 2, CEIL / 2, z); scene.add(wall); }
const ceilTex = canvasTex(256, 256, (g, s) => {
  g.fillStyle = '#f3ede0'; g.fillRect(0, 0, s, s); g.strokeStyle = '#d9c79a'; g.lineWidth = 6; g.strokeRect(10, 10, s - 20, s - 20);
  g.lineWidth = 2; g.strokeRect(40, 40, s - 80, s - 80);
});
ceilTex.wrapS = ceilTex.wrapT = THREE.RepeatWrapping; ceilTex.repeat.set((L + 30) / 6, W / 6);
const ceiling = new THREE.Mesh(new THREE.BoxGeometry(L + 30, 0.4, W + 2), new THREE.MeshLambertMaterial({ map: ceilTex }));
ceiling.position.set(L / 2, CEIL + 0.2, 0); scene.add(ceiling);
const lamps = []; for (let x = 6; x <= L - 6; x += 12) for (const z of [-6, 6]) lamps.push({ x, z });
instanced(new THREE.CylinderGeometry(0.03, 0.03, 1.4, 4), M.gold, lamps, (o, p) => o.position.set(p.x, CEIL - 0.7, p.z));
instanced(new THREE.SphereGeometry(0.42, 10, 8), new THREE.MeshBasicMaterial({ color: '#fff3c4' }), lamps, (o, p) => o.position.set(p.x, CEIL - 1.5, p.z));

// ---------- Lampu hijau: lari kecil untuk laki-laki ----------
const greenMat = new THREE.MeshBasicMaterial({ color: '#22c55e' });
for (const z of [-8, -3, 3, 8]) { const bar = new THREE.Mesh(new THREE.BoxGeometry(GZ1 - GZ0, 0.18, 0.3), greenMat); bar.position.set((GZ0 + GZ1) / 2, CEIL - 0.15, z); scene.add(bar); }
for (const x of [GZ0, GZ1]) { const cross = new THREE.Mesh(new THREE.BoxGeometry(0.35, 0.25, W - 2), greenMat); cross.position.set(x, CEIL - 0.2, 0); scene.add(cross); }
const greenPill = []; pillars.forEach((p) => { if (p.x >= GZ0 - 1 && p.x <= GZ1 + 1) greenPill.push(p); });
instanced(new THREE.BoxGeometry(1.5, 0.35, 1.5), greenMat, greenPill, (o, p) => o.position.set(p.x, CEIL - 1.2, p.z));
const greenFloor = new THREE.Mesh(new THREE.PlaneGeometry(GZ1 - GZ0, W - 4), new THREE.MeshBasicMaterial({ color: '#22c55e', transparent: true, opacity: 0.08, depthWrite: false }));
greenFloor.rotation.x = -Math.PI / 2; greenFloor.position.set((GZ0 + GZ1) / 2, 0.03, 0); scene.add(greenFloor);

// ---------- Bukit Shafa & Marwah (batu low-poly) ----------
function rock(cx, seed) {
  const geo = new THREE.IcosahedronGeometry(7, 2), pos = geo.attributes.position, v = new THREE.Vector3();
  const hash = (x, y, z) => { const h = Math.sin(x * 12.9898 + y * 78.233 + z * 37.719 + seed) * 43758.5453; return h - Math.floor(h); };
  for (let i = 0; i < pos.count; i++) {
    v.fromBufferAttribute(pos, i); const k = 0.78 + hash(Math.round(v.x * 3), Math.round(v.y * 3), Math.round(v.z * 3)) * 0.38;
    v.multiplyScalar(k); if (v.y < 0) v.y *= 0.15; pos.setXYZ(i, v.x, v.y, v.z);
  }
  geo.computeVertexNormals();
  const m = new THREE.Mesh(geo, new THREE.MeshLambertMaterial({ color: '#8f8475', flatShading: true }));
  m.scale.set(1.15, 0.62, 1.3); m.position.set(cx, 0, 0); scene.add(m);
  const glass = new THREE.Mesh(new THREE.CylinderGeometry(9.5, 9.5, 1.3, 24, 1, true, 0, Math.PI), new THREE.MeshLambertMaterial({ color: '#cfe7f2', transparent: true, opacity: 0.35, side: THREE.DoubleSide }));
  glass.position.set(cx, 0.65, 0); glass.rotation.y = cx < L / 2 ? 0 : Math.PI; // sisi kaca menghadap lorong scene.add(glass);
  const plat = new THREE.Mesh(new THREE.CircleGeometry(11, 32), mat('#ebe8e0')); plat.rotation.x = -Math.PI / 2; plat.position.set(cx, 0.01, 0); scene.add(plat);
}
rock(-8, 1.3); rock(L + 8, 7.1);

// Panah arah Ka'bah (muncul saat berdoa di bukit)
const kaabaArrow = new THREE.Group();
{
  const am = new THREE.MeshBasicMaterial({ color: '#047857', transparent: true, opacity: 0.85, depthWrite: false });
  const shaft = new THREE.Mesh(new THREE.PlaneGeometry(0.5, 3), am); shaft.rotation.x = -Math.PI / 2; shaft.position.z = -1.5;
  const tip = new THREE.Mesh(new THREE.CircleGeometry(0.9, 3), am); tip.rotation.x = -Math.PI / 2; tip.rotation.z = Math.PI / 2; tip.position.z = -3.3;
  kaabaArrow.add(shaft, tip); kaabaArrow.position.y = 0.05; kaabaArrow.visible = false; scene.add(kaabaArrow);
}
const kaabaDir = (atMarwah) => new THREE.Vector3(atMarwah ? -0.35 : 0.35, 0, -1).normalize();

// ---------- Jamaah lain (dua arah) ----------
const CROWD = 90, crowd = [];
const crowdBody = new THREE.InstancedMesh(new THREE.CapsuleGeometry(0.34, 0.75, 2, 8), new THREE.MeshLambertMaterial({ color: '#ffffff' }), CROWD);
const crowdHead = new THREE.InstancedMesh(new THREE.SphereGeometry(0.24, 8, 6), new THREE.MeshLambertMaterial({ color: '#ffffff' }), CROWD);
const skinTones = ['#f1c7a1', '#d9a47c', '#a8754f', '#7a5236'];
for (let i = 0; i < CROWD; i++) {
  const toM = i % 2 === 0;
  let z = toM ? -2.6 - Math.random() * 8.6 : 2.6 + Math.random() * 8.6;
  if (Math.abs(z - (toM ? LANE.toMarwah : LANE.toSafa)) < 1.3) z += toM ? -2.4 : 2.4;
  const woman = Math.random() < 0.35, abaya = woman && Math.random() < 0.6;
  crowd.push({ x: 6 + Math.random() * (L - 12), z, v: (toM ? 1 : -1) * (2.2 + Math.random() * 1.6) });
  crowdBody.setColorAt(i, new THREE.Color(abaya ? '#2f2f33' : woman ? '#e9e3d6' : '#fbfbf6'));
  crowdHead.setColorAt(i, new THREE.Color(woman ? (abaya ? '#2f2f33' : '#e2dccd') : skinTones[i % 4]));
}
scene.add(crowdBody, crowdHead);
const _o = new THREE.Object3D();
function updateCrowd(dt) {
  crowd.forEach((c, i) => {
    c.x += c.v * dt; if (c.x > L - 6) c.x = 6; if (c.x < 6) c.x = L - 6;
    _o.position.set(c.x, 0.72, c.z); _o.updateMatrix(); crowdBody.setMatrixAt(i, _o.matrix);
    _o.position.y = 1.45; _o.updateMatrix(); crowdHead.setMatrixAt(i, _o.matrix);
  });
  crowdBody.instanceMatrix.needsUpdate = true; crowdHead.instanceMatrix.needsUpdate = true;
}

// ---------- Label & mini-peta ----------
sim.addLabel('Bukit Shafa · Start', new THREE.Vector3(-2, 6.5, 0), 'hl');
sim.addLabel('Bukit Marwah · Finish', new THREE.Vector3(L + 2, 6.5, 0), 'hl');
sim.addLabel('Lampu hijau', new THREE.Vector3((GZ0 + GZ1) / 2, CEIL - 0.8, -3));
const arrowLabel = sim.addLabel("Arah Ka'bah", new THREE.Vector3(), 'hl'); arrowLabel.visible = false;
const map = $('simMap');
map.classList.remove('hidden');
map.innerHTML = `<div class="flex items-center gap-1.5 text-[10px] font-bold text-gray-600"><span>Shafa</span>
  <div class="relative flex-1 h-2 rounded-full bg-gray-200"><div class="absolute inset-y-0 rounded-full bg-green-500/70" style="left:${GZ0 / L * 100}%;width:${(GZ1 - GZ0) / L * 100}%"></div>
  <div id="mapDot" class="absolute top-1/2 -translate-x-1/2 -translate-y-1/2 w-3.5 h-3.5 rounded-full bg-emerald-700 ring-2 ring-white shadow" style="left:0%"></div></div><span>Marwah</span></div>`;
const mapDot = $('mapDot');

// ---------- Teks pemandu ----------
const AYAT = { ar: 'إِنَّ الصَّفَا وَالْمَرْوَةَ مِنْ شَعَائِرِ اللهِ، أَبْدَأُ بِمَا بَدَأَ اللهُ بِهِ', lat: "Innaṣ-ṣafā wal-marwata min sya'ā'irillāh. Abda'u bimā bada'allāhu bih.", arti: 'Sesungguhnya Shafa dan Marwah termasuk syiar-syiar Allah (QS. Al-Baqarah: 158). Aku memulai dengan apa yang Allah mulai.' };
const DZIKIR = {
  ar: 'اللهُ أَكْبَرُ، اللهُ أَكْبَرُ، اللهُ أَكْبَرُ. لَا إِلَهَ إِلَّا اللهُ وَحْدَهُ لَا شَرِيكَ لَهُ، لَهُ الْمُلْكُ وَلَهُ الْحَمْدُ، وَهُوَ عَلَى كُلِّ شَيْءٍ قَدِيرٌ. لَا إِلَهَ إِلَّا اللهُ وَحْدَهُ، أَنْجَزَ وَعْدَهُ، وَنَصَرَ عَبْدَهُ، وَهَزَمَ الْأَحْزَابَ وَحْدَهُ',
  lat: "Allāhu akbar (3×). Lā ilāha illallāhu waḥdahū lā syarīka lah, lahul-mulku wa lahul-ḥamdu wa huwa 'alā kulli syai'in qadīr. Lā ilāha illallāhu waḥdah, anjaza wa'dah, wa naṣara 'abdah, wa hazamal-aḥzāba waḥdah.",
  arti: 'Allah Maha Besar (3×). Tiada tuhan selain Allah semata, tiada sekutu bagi-Nya; milik-Nya kerajaan dan pujian, dan Dia Mahakuasa atas segala sesuatu. Tiada tuhan selain Allah semata; Dia menepati janji-Nya, menolong hamba-Nya, dan mengalahkan golongan-golongan musuh sendirian. (HR. Muslim)',
};
const RABBIGHFIR = {
  ar: 'رَبِّ اغْفِرْ وَارْحَمْ وَاعْفُ وَتَكَرَّمْ، وَتَجَاوَزْ عَمَّا تَعْلَمُ، إِنَّكَ تَعْلَمُ مَا لَا نَعْلَمُ، إِنَّكَ أَنْتَ اللهُ الْأَعَزُّ الْأَكْرَمُ',
  lat: "Rabbighfir warḥam wa'fu wa takarram, wa tajāwaz 'ammā ta'lam, innaka ta'lamu mā lā na'lam, innaka antallāhul-a'azzul-akram.",
  arti: 'Ya Rabb, ampunilah, rahmatilah, maafkanlah, dan muliakanlah; hapuskanlah apa yang Engkau ketahui. Sesungguhnya Engkau mengetahui apa yang tidak kami ketahui. Sesungguhnya Engkaulah Allah Yang Mahaperkasa lagi Mahamulia.',
};
const COUNT_CARD = `<div class="mt-2 grid grid-cols-2 gap-2 text-center text-xs">
  <div class="rounded-xl border border-rose-200 bg-rose-50 px-2 py-2"><p class="font-bold text-rose-700">✕ Salah</p><p class="text-gray-600 mt-0.5">Shafa → Marwah → Shafa dihitung 1</p></div>
  <div class="rounded-xl border border-emerald-200 bg-emerald-50 px-2 py-2"><p class="font-bold text-emerald-700">✓ Benar</p><p class="text-gray-600 mt-0.5">Shafa → Marwah = 1, Marwah → Shafa = 2, dst.</p></div></div>`;

const { info, pauseWith, toast, isM } = sim;
const chibi = () => sim.chibi;
const hillName = (atMarwah) => (atMarwah ? 'Marwah' : 'Shafa');

Object.assign(S, { x: 26, z: LANE.toSafa, dir: -1, done: 0, fired: {}, hold: 0 });

hooks.chibiOpts = () => ({ shoulderBare: false, scale: 1.6 }); // bahu kanan ditutup lagi setelah tawaf; 80% ukuran tawaf
hooks.canWalk = () => S.phase === 'approach' || S.phase === 'sai';
hooks.onBack = () => toast(S.dir > 0 ? 'Sedang menuju Marwah. Ikuti arah lintasan ini sampai ujung.' : 'Sedang menuju Shafa. Ikuti arah lintasan ini sampai ujung.');
hooks.counter = () => {
  const n = Math.min(7, S.done + 1), small = window.innerWidth < 640;
  if ((S.phase === 'hill' || S.hold > 0) && S.done > 0 && S.phase !== 'finish')
    return { text: `${small ? '' : 'Lintasan '}${S.done}/7 selesai`, panel: `Lintasan ${S.done} selesai`, done: S.done, active: false };
  if (S.phase === 'sai' && small) return { text: `${n}/7 · ke ${S.dir > 0 ? 'Marwah' : 'Shafa'}`, panel: `Lintasan ${n} dari 7`, done: S.done, active: true };
  if (S.phase === 'sai') return { text: `Lintasan ${n}/7 · ${S.dir > 0 ? 'ke Marwah' : 'ke Shafa'}`, panel: `Lintasan ${n} dari 7`, done: S.done, active: true };
  if (S.phase === 'finish') return { text: 'Selesai 7/7', panel: '', done: 7, active: false };
  return { text: 'Belum mulai', panel: '', done: 0, active: false };
};

function prayAtHill(atMarwah, secs) {
  S.gestureKind = 'doa'; S.gesture = secs;
  kaabaArrow.rotation.y = (() => { const d = kaabaDir(atMarwah); return Math.atan2(-d.x, -d.z); })();
  kaabaArrow.visible = true; arrowLabel.visible = true;
}
function stopPray() { S.gesture = 0; kaabaArrow.visible = false; arrowLabel.visible = false; }

hooks.begin = () => {
  Object.assign(S, { phase: 'intro', x: 26, z: LANE.toSafa, dir: -1, done: 0, fired: {}, hold: 0, gesture: 0 });
  stopPray();
  pauseWith([{
    title: "Sebelum sa'i",
    body: `<p>Sa'i dilakukan <b>setelah tawaf</b>: <b>7 lintasan</b> antara bukit Shafa dan Marwah, <b>dimulai di Shafa dan selesai di Marwah</b>.</p>
           <p>Dianjurkan dalam keadaan suci, tetapi menurut mayoritas ulama wudhu tidak wajib untuk sa'i. Boleh beristirahat sejenak bila lelah.</p>` +
      (isM() ? '<p>Bahu kanan sudah <b>ditutup kembali</b>, karena idhtiba\' hanya saat tawaf.</p>' : ''),
    hint: 'Setelah ini, klik tombol Jalan untuk menuju bukit Shafa.',
  }], () => { S.phase = 'approach'; info({ title: 'Menuju bukit Shafa', body: '<p>Klik <b>Jalan</b> untuk berjalan ke bukit Shafa, titik awal sa\'i.</p>' }); });
};

function arriveSafaFirst() {
  S.phase = 'hill'; S.x = X0; prayAtHill(false, 999);
  pauseWith([{
    title: 'Di bukit Shafa',
    body: '<p>Saat mendekati Shafa (hanya di awal sa\'i), bacalah:</p>', dua: AYAT, next: 'Lanjut →',
  }, {
    title: "Menghadap Ka'bah, angkat kedua tangan",
    body: `<p>Naik ke Shafa, <b>hadapkan badan ke arah Ka'bah</b> (lihat panah hijau), lalu <b>angkat kedua tangan seperti berdoa</b>, bukan isyarat seperti di Hajar Aswad. Bacalah:</p>`,
    dua: DZIKIR, after: '<p>Lalu berdoa dengan doa apa saja. Dzikir dan doa ini diulang <b>3 kali</b>.</p>', next: 'Mulai sa\'i →',
  }], () => {
    stopPray(); S.phase = 'sai'; S.dir = 1; S.done = 0; sim.setCounter();
    // Pesan singkat di atas karakter, simulasi tetap berjalan.
    sim.pops(["Selama sa'i tidak ada doa khusus", ...(isM() ? ['Di zona lampu hijau, lelaki sunnahnya lari kecil'] : [])]);
    info({ title: 'Lintasan 1: menuju Marwah', body: '<p>Klik <b>Jalan</b>. Berdzikir dan berdoalah sepanjang jalan.</p>' }); });
}

function onGreen(enter, lap) {
  const key = lap + ':' + (enter ? 'in' : 'out'); if (S.fired[key]) return; S.fired[key] = 1;
  if (enter) {
    // Doa dibaca selama berada di antara dua lampu hijau (boleh, tidak wajib).
    const run = isM() ? '<b>Laki-laki lari kecil</b> sampai lampu hijau berikutnya, jika memungkinkan.' : '<b>Perempuan tetap berjalan biasa.</b>';
    if (lap === 1) pauseWith([{
      title: 'Zona lampu hijau',
      body: `<p>${run} Zona ini mengenang Siti Hajar yang berlari mencari air untuk Ismail. Sambil melintas, bacalah:</p>`,
      dua: RABBIGHFIR,
    }], () => info({ title: 'Zona lampu hijau', body: `<p>${run}</p>`, dua: RABBIGHFIR }));
    else info({ title: `Lintasan ${lap} · zona lampu hijau`, body: `<p>${run}</p>`, dua: RABBIGHFIR });
  } else {
    info({ title: `Lintasan ${lap} · keluar zona hijau`, body: `<p>Kembali berjalan biasa menuju ${hillName(S.dir > 0)}. Terus berdzikir dan berdoa.</p>` });
  }
}

function arriveEnd() {
  const atMarwah = S.dir > 0;
  S.done++;
  if (S.done >= 7) { finish(); return; }
  const next = hillName(!atMarwah);
  if (S.done === 1) {
    S.phase = 'hill'; prayAtHill(true, 999); sim.setCounter();
    pauseWith([{
      title: 'Tiba di Marwah: lintasan 1 selesai (1/7)',
      body: '<p>Perjalanan dari Shafa ke Marwah sudah dihitung <b>satu lintasan</b>. Kembali ke Shafa nanti dihitung lintasan kedua.</p>', extra: COUNT_CARD, next: 'Lanjut →',
    }, {
      title: "Menghadap Ka'bah, angkat kedua tangan",
      body: '<p>Di Marwah lakukan seperti di Shafa: menghadap Ka\'bah, angkat kedua tangan, lalu baca dzikir (tanpa ayat Innaṣ-ṣafā):</p>', dua: DZIKIR,
      after: '<p>Kemudian berdoa, lalu kembali menuju Shafa.</p>', next: `Lintasan 2: ke ${next} →`,
    }], () => { stopPray(); S.phase = 'sai'; S.dir = -1; sim.setCounter(); info({ title: `Lintasan 2: menuju ${next}`, body: '<p>Klik <b>Jalan</b>. Berdzikir dan berdoalah sepanjang jalan.</p>' }); });
    return;
  }
  // Lintasan 2-6: berhenti sebentar untuk berdoa, lalu otomatis berbalik dan lanjut.
  S.hold = 2.6; prayAtHill(atMarwah, 2.6); sim.setCounter();
  info({
    title: `Tiba di ${hillName(atMarwah)}: lintasan ${S.done} selesai (${S.done}/7)`,
    body: `<p>Menghadap Ka'bah, angkat kedua tangan, berdzikir dan berdoa. Lalu lanjut ke <b>${next}</b> untuk lintasan ${S.done + 1}.</p>`,
    hint: `Tersisa ${7 - S.done} lintasan lagi.`,
  });
}

function finish() {
  S.phase = 'finish'; S.x = X1; prayAtHill(true, 999); sim.setCounter();
  pauseWith([{
    title: "Alhamdulillah, sa'i selesai!",
    body: '<p>Kamu sudah menyelesaikan <b>7 lintasan</b>, berakhir di <b>Marwah</b>. Berdoalah di sini seperti sebelumnya.</p>', next: 'Lanjut →',
  }, {
    title: 'Tahallul: memotong rambut',
    body: isM()
      ? '<p>Laki-laki <b>mencukur gundul</b> (lebih utama) atau <b>memendekkan rambut</b> secara merata di seluruh kepala.</p>'
      : '<p>Perempuan <b>memotong ujung rambut</b> kira-kira <b>seruas jari</b>. Tidak dicukur.</p>',
    after: '<p>Setelah tahallul, larangan-larangan ihram telah selesai.</p>', next: 'Lanjut →',
  }, {
    title: 'Alhamdulillah, umrahmu selesai',
    body: '<p>Rangkaian umrah lengkap: <b>ihram, tawaf, sa\'i, dan tahallul</b>. Semoga Allah menerima ibadahmu dan memudahkan langkahmu ke Tanah Suci.</p>',
    actions: '<a href="/simulasi?bagian=tawaf" class="rounded-xl border border-gray-200 px-3.5 lg:px-4 py-1.5 lg:py-2.5 text-[13px] lg:text-sm font-semibold text-gray-600 hover:bg-gray-50">Ulangi Tawaf</a>',
    next: 'Mulai Ulang',
  }], () => sim.reset());
}

hooks.reset = () => {
  S.phase = 'choose'; stopPray();
  info({ title: 'Selamat datang', body: "<p>Pilih karakter untuk memulai simulasi sa'i.</p>" });
};

// ---------- Loop ----------
hooks.update = (dt) => {
  updateCrowd(dt);
  if (!chibi()) return;
  let speed = 0, run = false;
  if (S.hold > 0) {
    S.hold -= dt;
    if (S.hold <= 0) { stopPray(); S.dir = -S.dir; sim.setCounter(); info({ title: `Lintasan ${S.done + 1}: menuju ${hillName(S.dir > 0)}`, body: '<p>Terus berjalan sambil berdzikir dan berdoa.</p>', hint: `Tersisa ${7 - S.done} lintasan lagi.` }); }
  } else if (sim.walking()) {
    const inGreen = S.phase === 'sai' && S.x > GZ0 && S.x < GZ1;
    run = inGreen && isM();
    speed = BASE_SPEED * (run ? 1.55 : 1);
    const px = S.x;
    S.x += S.dir * speed * dt;
    if (S.phase === 'approach' && S.x <= X0) { S.x = X0; arriveSafaFirst(); }
    else if (S.phase === 'sai') {
      const lap = S.done + 1;
      const enterAt = S.dir > 0 ? GZ0 : GZ1, exitAt = S.dir > 0 ? GZ1 : GZ0;
      if ((px - enterAt) * (S.x - enterAt) <= 0 && px !== S.x) onGreen(true, lap);
      if ((px - exitAt) * (S.x - exitAt) <= 0 && px !== S.x) onGreen(false, lap);
      if (S.dir > 0 && S.x >= X1) { S.x = X1; arriveEnd(); }
      else if (S.dir < 0 && S.x <= X0) { S.x = X0; arriveEnd(); }
    }
  }
  // pindah jalur dengan halus sesuai arah
  const praying = S.gesture > 0 && S.gestureKind === 'doa';
  const laneZ = praying ? -2.5 : S.dir > 0 ? LANE.toMarwah : LANE.toSafa; // saat berdoa: maju ke depan bukit
  S.z += (laneZ - S.z) * Math.min(1, dt * (praying ? 2.5 : speed > 0 ? 1.2 : 0.4));
  const g = chibi().g; g.position.set(S.x, 0, S.z);
  if (kaabaArrow.visible) { const d = kaabaDir(S.x > L / 2); kaabaArrow.position.set(S.x + d.x * 1.6, 0.05, S.z + d.z * 1.6); arrowLabel.pos.set(S.x + d.x * 7, 0.6, S.z + d.z * 7); }
  let face = S.dir > 0 ? Math.PI / 2 : -Math.PI / 2; // menghadap +x / -x
  if (S.gesture > 0 && S.gestureKind === 'doa') { const d = kaabaDir(S.x > L / 2); face = Math.atan2(d.x, d.z); }
  sim.face(face, dt);
  sim.animate(dt, speed, { run });
  mapDot.style.left = (Math.max(0, Math.min(L, S.x)) / L * 100) + '%';
};

hooks.camera = (dt) => {
  let pos, tgt;
  if (!chibi() || S.phase === 'choose') {
    cam.orbit += dt * 0.12;
    pos = new THREE.Vector3(L / 2 + Math.sin(cam.orbit) * 50, 46, 60); tgt = new THREE.Vector3(L / 2 + Math.sin(cam.orbit) * 30, 0, 0);
  } else if (cam.top) {
    const tall = camera.aspect < 1, c = chibi().g.position;
    pos = new THREE.Vector3(tall ? c.x : L / 2, (tall ? 70 : 78) * cam.zoom, (tall ? 55 : 70) * cam.zoom); tgt = new THREE.Vector3(tall ? c.x : L / 2, 0, 0);
  } else {
    const tall = camera.aspect < 1, c = chibi().g.position, back = (tall ? 15 : 12) * cam.zoom;
    const praying = S.gesture > 0 && S.gestureKind === 'doa';
    if (praying) { // dari belakang karakter yang menghadap Ka'bah
      const d = kaabaDir(S.x > L / 2), back = (tall ? 15 : 12) * cam.zoom;
      pos = new THREE.Vector3(c.x - d.x * back, (tall ? 8.5 : 8) * cam.zoom, c.z - d.z * back); tgt = new THREE.Vector3(c.x + d.x * 6, 1.5, c.z + d.z * 6);
    } else {
      pos = new THREE.Vector3(c.x - S.dir * back, (tall ? 8 : 6.5) * cam.zoom, c.z + (S.dir > 0 ? -2 : 2)); tgt = new THREE.Vector3(c.x + S.dir * 10, 2, c.z * 0.6);
    }
    // Kamera tetap di dalam lorong (tidak menembus dinding, bukit, atau langit-langit).
    pos.z = THREE.MathUtils.clamp(pos.z, -12, 12); pos.x = THREE.MathUtils.clamp(pos.x, 1, L - 1); pos.y = Math.min(pos.y, CEIL - 0.8);
  }
  return { pos, tgt };
};

// Langit-langit disembunyikan saat kamera berada di atasnya (mode lihat dari atas / pembuka).
const baseUpdate = hooks.update;
hooks.update = (dt) => { baseUpdate(dt); ceiling.visible = camera.position.y < CEIL - 0.5; };
sim.start({ pos: new THREE.Vector3(L / 2, 46, 60), tgt: new THREE.Vector3(L / 2, 0, 0) });
