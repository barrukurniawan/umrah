// Karakter chibi Umrohku: dikembangkan dari rig karakter wedding-summer (marryme.web.id):
// kepala besar berpipi, mata berkilau yang berkedip, tangan bulat, dan langkah kaki yang menapak (IK).
// Pakaian khusus umrah: ihram 2 lembar (izar + rida, bisa idhtiba'), gamis + khimar untuk perempuan.
// Menghadap +z; kaki di y = 0; sisi kanan karakter ada di -x (arms[0], legs[0]).
import * as THREE from 'three';
const T = THREE;

export const WALK_SPEED = 3.8;     // kecepatan jalan "alami" dalam satuan lokal rig
export const RIG_SCALE = 0.78;     // tinggi rig dinormalkan supaya setinggi chibi lama (±2,45 satuan)
const ARM_SPLAY = 0.2;            // lengan sedikit terbuka supaya tangan tidak menembus izar/gamis

const DETAIL = {
  full: { sph: [24, 16], head: [48, 32], tube: [20, 6], lathe: 32, oval: 40, cap: [64, 22], hood: [24, 64], ridaRows: 18, locks: true, fine: true, lite: false },
  crowd: { sph: [7, 5], head: [12, 8], tube: [5, 3], lathe: 10, oval: 8, cap: [14, 5], hood: [5, 14], ridaRows: 8, locks: false, fine: false, lite: true },
};

// Kain handuk ihram: putih gading dengan serat halus (tekstur kanvas kecil, diulang).
let towelTex = null;
function towel() {
  if (towelTex) return towelTex;
  const c = document.createElement('canvas'); c.width = c.height = 64;
  const g = c.getContext('2d'); g.fillStyle = '#ffffff'; g.fillRect(0, 0, 64, 64);
  for (let y = 0; y < 64; y += 2) { g.fillStyle = `rgba(150,140,120,${0.05 + (y % 4 ? 0.02 : 0.05)})`; g.fillRect(0, y, 64, 1); }
  for (let i = 0; i < 260; i++) { g.fillStyle = `rgba(120,110,95,${Math.random() * 0.07})`; g.fillRect(Math.random() * 64, Math.random() * 64, 1, 1 + Math.random() * 2); }
  towelTex = new T.CanvasTexture(c); towelTex.colorSpace = T.SRGBColorSpace; towelTex.wrapS = towelTex.wrapT = T.RepeatWrapping; towelTex.repeat.set(6, 4);
  return towelTex;
}

// gender 'm' | 'f'
// opts: skin, shoulderBare (idhtiba'), outfit ('ihram' | 'casual'), cloth (warna gamis/baju), veil (warna khimar),
//       detail ('full' | 'crowd'), bald, lambert (material ringan)
export function buildRig(gender, opts = {}) {
  const D = DETAIL[opts.detail || 'full'];
  const isM = gender === 'm', ihram = isM && (opts.outfit || 'ihram') === 'ihram';
  const root = new T.Group(); root.name = 'Umrohku chibi';
  const body = new T.Group(); body.name = 'Body'; root.add(body);
  const Mat = opts.lambert ? T.MeshLambertMaterial : T.MeshStandardMaterial;
  const material = (color, roughness = 0.82, extra = {}) => new Mat(opts.lambert ? { color, ...extra } : { color, roughness, ...extra });
  const skinHex = opts.skin || '#e8ad7e';
  const skin = material(skinHex); skin.userData.skin = 1;
  const innerEar = material(new T.Color(skinHex).multiplyScalar(0.84)); innerEar.userData.skin = 0.84;
  const noseWarm = material(new T.Color(skinHex).multiplyScalar(0.95)); noseWarm.userData.skin = 0.95;
  const hair = material('#24232a', 0.84);
  const cream = material('#fff9eb'), iris = material('#392b27'), ink = material('#24202a'), mouth = material('#6b302c');
  const sphere = new T.SphereGeometry(1, D.sph[0], D.sph[1]);

  function mesh(parent, name, geo, mat) { const o = new T.Mesh(geo, mat); o.name = name; parent.add(o); return o; }
  function ell(parent, name, mat, x, y, z, sx, sy, sz) { const o = mesh(parent, name, sphere, mat); o.position.set(x, y, z); o.scale.set(sx, sy, sz); return o; }
  function line(parent, name, points, r, mat, closed = false) {
    return mesh(parent, name, new T.TubeGeometry(new T.CatmullRomCurve3(points.map((p) => new T.Vector3(...p)), closed), D.tube[0] * (closed ? 3 : 1), r, D.tube[1], closed), mat);
  }
  function lathe(parent, name, profile, mat, zScale = 1, segs = D.lathe) {
    const o = mesh(parent, name, new T.LatheGeometry(profile.map((p) => new T.Vector2(...p)), segs), mat); o.scale.z = zScale; return o;
  }
  // Lipatan kain: riak kecil mengelilingi lathe (dipakai izar, gamis, khimar).
  function folds(o, n, amp, below = Infinity) {
    o.geometry = o.geometry.clone(); const p = o.geometry.attributes.position;
    for (let i = 0; i < p.count; i++) {
      const x = p.getX(i), y = p.getY(i), z = p.getZ(i); if (y > below) continue;
      const f = 1 + amp * Math.cos(Math.atan2(z, x) * n); p.setXYZ(i, x * f, y, z * f);
    }
    o.geometry.computeVertexNormals(); return o;
  }

  const clothHex = opts.cloth || (isM ? '#fbfaf5' : '#f4efe6');
  const cloth = ihram ? material('#ffffff', 0.95, { map: towel() }) : material(clothHex, 0.9);
  const hem = ihram ? material('#ece7dc', 0.95) : material(new T.Color(clothHex).multiplyScalar(0.9), 0.9);
  const leather = material('#7a5235', 0.7), sole = material('#4a3324', 0.8);

  // ---------- Badan ----------
  const torsoProfile = [[0, 0.78], [0.33, 0.78], [0.40, 0.80], [0.425, 0.88], [0.415, 1.10], [0.395, 1.38], [0.355, 1.51], [0.24, 1.60], [0.145, 1.63], [0, 1.63]];
  ell(body, 'Neck', skin, 0, 1.66, 0, 0.15, 0.20, 0.14);
  if (isM) {
    if (ihram) {
      // Izar: kain bawah dari pinggang sampai bawah lutut, sedikit mengembang supaya langkah tetap di dalam.
      folds(lathe(body, 'Izar', [[0, 0.40], [0.47, 0.40], [0.485, 0.43], [0.475, 0.60], [0.445, 0.82], [0.425, 0.98], [0.41, 1.04], [0, 1.04]], cloth, 0.94), 11, 0.025, 0.95);
      line(body, 'Izar front overlap', [[0.16, 1.02, 0.37], [0.175, 0.80, 0.39], [0.19, 0.43, 0.405]], 0.014, hem);
      if (!D.lite) lathe(body, 'Izar hem', [[0.465, 0.40], [0.49, 0.405], [0.49, 0.43], [0.48, 0.435]], hem, 0.94);
      // Rida: kain atas. Saat idhtiba' kain lewat di bawah ketiak kanan dan dilempar ke bahu kiri.
      const bare = !!opts.shoulderBare;
      if (bare || !D.lite) lathe(body, 'Bare chest', torsoProfile, skin, 0.78);
      const cut = (x) => (bare ? 1.22 + (x + 0.42) * 0.73 : 9);
      const prof = [];
      for (let i = 0, N = D.ridaRows; i <= N; i++) { // profil rapat supaya tepi potongan halus
        const u = i / N, k = Math.min(torsoProfile.length - 2, Math.floor(u * (torsoProfile.length - 3)) + 1);
        const a = torsoProfile[k], b = torsoProfile[k + 1], f = u * (torsoProfile.length - 3) + 1 - k;
        prof.push([(a[0] + (b[0] - a[0]) * f) * 1.06 + 0.004, a[1] + (b[1] - a[1]) * f]);
      }
      prof[0] = [0.455, 0.90];
      const segs = D.lite ? D.lathe : D.lathe * 2, pos = [], uv = [], idx = [], edge = [];
      for (let c = 0; c <= segs; c++) {
        const phi = (c / segs) * Math.PI * 2, sx = Math.sin(phi), cz = Math.cos(phi);
        let lastIn = null; const col = [];
        // bagian bawah rida sebulat izar (menutup pinggang tanpa celah), dada mengikuti badan
        for (let r = 0; r < prof.length; r++) { const [rad, y] = prof[r], zs = T.MathUtils.lerp(0.95, 0.8, T.MathUtils.clamp((y - 0.92) / 0.36, 0, 1)); col.push([rad * sx, y, rad * cz * zs]); }
        // vertex di atas garis potong diturunkan ke titik potong (tepi rapi, tanpa gerigi)
        let clipped = false;
        for (let r = 0; r < col.length; r++) {
          const p = col[r];
          if (!clipped && p[1] <= cut(p[0])) { lastIn = p; continue; }
          if (!clipped && lastIn) {
            const a = lastIn, b = p, fa = a[1] - cut(a[0]), fb = b[1] - cut(b[0]), t = fa / (fa - fb);
            lastIn = [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t, a[2] + (b[2] - a[2]) * t];
          }
          clipped = true; col[r] = lastIn || p;
        }
        edge.push(col[col.length - 1]);
        col.forEach((p, r) => { pos.push(...p); uv.push(c / segs * 3, r / prof.length * 2); });
        if (c < segs) for (let r = 0; r < prof.length - 1; r++) { const a = c * prof.length + r, b = a + prof.length; idx.push(a, b, a + 1, a + 1, b, b + 1); }
      }
      const g = new T.BufferGeometry(); g.setAttribute('position', new T.Float32BufferAttribute(pos, 3)); g.setAttribute('uv', new T.Float32BufferAttribute(uv, 2)); g.setIndex(idx); g.computeVertexNormals();
      mesh(body, 'Rida', g, cloth).material.side = T.DoubleSide;
      if (bare && !D.lite) {
        // pita tepi kain mengikuti garis potong (depan, bahu kiri, belakang)
        const pts = edge.filter((_, i) => i % 2 === 0).map((p) => [p[0] * 1.015, p[1] - 0.004, p[2] * 1.015]);
        line(body, 'Rida folded edge', pts, 0.022, hem, true);
      } else if (!bare) {
        // ujung rida dililit: selempang lebar dari bahu kanan ke pinggang kiri + lipatan bahu
        line(body, 'Rida wrapped fold', [[-0.33, 1.52, 0.17], [-0.16, 1.40, 0.36], [0.06, 1.20, 0.37], [0.27, 1.02, 0.32], [0.40, 0.95, 0.12]], 0.03, hem);
        if (!D.lite) line(body, 'Rida soft fold', [[-0.12, 1.58, 0.24], [0.02, 1.44, 0.355], [0.20, 1.27, 0.355]], 0.012, hem);
      }
    } else {
      // Baju biasa (mis. tukang cukur): kemeja polos + celana.
      lathe(body, 'Shirt', torsoProfile, cloth, 0.78);
      lathe(body, 'Shirt hem', [[0.40, 0.81], [0.426, 0.82], [0.426, 0.855], [0.417, 0.86]], hem, 0.78);
      const placket = mesh(body, 'Placket', new T.BoxGeometry(0.052, 0.67, 0.019), hem); placket.position.set(0, 1.19, 0.329);
      for (let y = 0.94; y < 1.5; y += 0.16) ell(body, 'Button', cream, 0, y, 0.35, 0.018, 0.018, 0.010);
    }
  } else {
    // Gamis panjang sampai mata kaki, longgar dan berlipat lembut.
    folds(lathe(body, 'Gamis', [[0, 0.10], [0.52, 0.10], [0.545, 0.13], [0.54, 0.30], [0.49, 0.66], [0.44, 0.95], [0.42, 1.20], [0.40, 1.40], [0.34, 1.54], [0.22, 1.62], [0, 1.64]], cloth, 0.9), 13, 0.024, 1.0);
    if (!D.lite) lathe(body, 'Gamis hem', [[0.51, 0.10], [0.55, 0.105], [0.55, 0.135], [0.54, 0.14]], hem, 0.9);
  }

  // ---------- Kepala (bentuk superellipsoid wedding-summer) ----------
  const head = new T.Group(); head.name = 'Head'; head.position.set(0, 2.23, 0); body.add(head);
  const headPower = 2.5, headHeight = 0.70, headDepth = 0.72;
  const sp = (v, p) => Math.sign(v) * Math.abs(v) ** p;
  const faceWidth = (y) => 0.77 * (1 + 0.035 * Math.exp(-(((y + 0.25) / 0.24) ** 2)));
  const headGeo = new T.SphereGeometry(1, D.head[0], D.head[1]), hp = headGeo.attributes.position;
  for (let i = 0; i < hp.count; i++) {
    const y = headHeight * sp(hp.getY(i), 2 / headPower);
    hp.setXYZ(i, faceWidth(y) * sp(hp.getX(i), 2 / headPower), y, headDepth * sp(hp.getZ(i), 2 / headPower));
  }
  headGeo.computeVertexNormals(); mesh(head, 'Head', headGeo, skin);
  const faceZ = (x, y) => headDepth * Math.max(0.005, 1 - Math.abs(x / faceWidth(y)) ** headPower - Math.abs(y / headHeight) ** headPower) ** (1 / headPower);
  function faceOval(parent, name, mat, cx, cy, rx, ry, offset = 0.007) {
    const geo = new T.CircleGeometry(1, D.oval), p = geo.attributes.position;
    for (let i = 0; i < p.count; i++) { const x = cx + p.getX(i) * rx, y = cy + p.getY(i) * ry; p.setXYZ(i, x, y, faceZ(x, y) + offset); }
    geo.computeVertexNormals(); return mesh(parent, name, geo, mat);
  }
  function faceShape(name, shape, mat, offset = 0.01) {
    const geo = new T.ShapeGeometry(shape, D.fine ? 24 : 6), p = geo.attributes.position;
    for (let i = 0; i < p.count; i++) p.setZ(i, faceZ(p.getX(i), p.getY(i)) + offset);
    geo.computeVertexNormals(); return mesh(head, name, geo, mat);
  }
  const eyes = [];
  const cheekMats = D.fine ? [0.06, 0.07, 0.09].map((o) => material('#e79181', 0.82, { transparent: true, opacity: o, depthWrite: false })) : [];
  const freckle = D.fine ? material('#ba7466', 0.82, { transparent: true, opacity: 0.32, depthWrite: false }) : null;
  for (const s of [-1, 1]) {
    cheekMats.forEach((m, i) => faceOval(head, 'Cheek tint', m, s * 0.462, -0.207, 0.161 - i * 0.039, 0.109 - i * 0.025, 0.010 + i * 0.002));
    if (freckle && opts.freckles !== false) for (const [x, y, r] of [[0.398, -0.181, 0.009], [0.437, -0.167, 0.007], [0.478, -0.191, 0.008], [0.447, -0.223, 0.008]]) faceOval(head, 'Freckle', freckle, s * x, y, r, r * 1.2, 0.018);
    const eye = new T.Group(); eye.name = 'Eye'; head.add(eye); eyes.push(eye);
    faceOval(eye, 'Eye white', cream, s * 0.282, -0.017, 0.140, 0.175, 0.010);
    faceOval(eye, 'Iris', iris, s * 0.270, -0.030, 0.098, 0.139, 0.014);
    if (D.fine) faceOval(eye, 'Pupil', ink, s * 0.267, -0.026, 0.080, 0.121, 0.018);
    faceOval(eye, 'Eye glint', cream, s * 0.267 - 0.027, 0.039, 0.028, 0.031, 0.023);
    if (D.fine) faceOval(eye, 'Small glint', cream, s * 0.267 + 0.033, -0.077, 0.010, 0.012, 0.024);
    eye.children.forEach((o) => o.geometry.translate(-s * 0.282, 0.017, 0)); eye.position.set(s * 0.282, -0.017, 0);
    line(head, 'Brow', [[s * 0.398, 0.223, faceZ(s * 0.398, 0.223) + 0.013], [s * 0.302, 0.251, faceZ(s * 0.302, 0.251) + 0.013], [s * 0.201, 0.226, faceZ(s * 0.201, 0.226) + 0.013]], 0.022, hair);
  }
  ell(head, 'Nose', skin, 0, -0.139, faceZ(0, -0.139) + 0.022, 0.063, 0.043, 0.045);
  if (!D.lite) ell(head, 'Nose tip', noseWarm, 0, -0.146, faceZ(0, -0.146) + 0.056, 0.038, 0.025, 0.020);
  const ms = new T.Shape(); ms.moveTo(-0.209, -0.284); ms.quadraticCurveTo(0, -0.325, 0.209, -0.284); ms.bezierCurveTo(0.196, -0.394, 0.111, -0.452, 0, -0.453); ms.bezierCurveTo(-0.114, -0.45, -0.195, -0.393, -0.209, -0.284);
  faceShape('Smile', ms, mouth, 0.010);
  if (D.fine) {
    const teeth = new T.Shape(); teeth.moveTo(-0.171, -0.302); teeth.quadraticCurveTo(0, -0.334, 0.172, -0.302); teeth.lineTo(0.149, -0.333); teeth.quadraticCurveTo(0, -0.361, -0.149, -0.333); teeth.closePath();
    faceShape('Teeth', teeth, cream, 0.014);
    const tongue = new T.Shape(); tongue.moveTo(-0.096, -0.422); tongue.quadraticCurveTo(-0.090, -0.385, 0.007, -0.389); tongue.quadraticCurveTo(0.086, -0.388, 0.109, -0.421); tongue.quadraticCurveTo(0.009, -0.467, -0.096, -0.422);
    faceShape('Tongue', tongue, material('#d97a7d'), 0.017);
  }

  if (isM) {
    for (const s of [-1, 1]) {
      ell(head, 'Ear', skin, s * 0.755, -0.12, 0.005, 0.178, 0.204, 0.120);
      if (!D.lite) ell(head, 'Ear inset', innerEar, s * 0.801, -0.116, 0.099, 0.094, 0.126, 0.029);
    }
    if (!opts.bald) addHair();
    else ell(head, 'Shaved scalp shade', material(new T.Color(skinHex).lerp(new T.Color('#3a3330'), 0.18)), 0, 0.12, -0.02, 0.79, 0.74, 0.73);
  } else addHijab();

  // Rambut pendek (rig asli): lapisan dasar + helai membulat (hanya detail penuh).
  function addHair() {
    const positions = [], uvs = [], indices = [], around = D.cap[0], rows = D.cap[1];
    for (let row = 0; row <= rows; row++) for (let col = 0; col <= around; col++) {
      const phi = (col / around) * Math.PI * 2, front = Math.max(0, Math.cos(phi)), back = Math.max(0, -Math.cos(phi));
      const limit = 1.82 + 0.25 * back ** 2 - 0.68 * front ** 3, theta = 0.001 + (limit - 0.001) * row / rows;
      const k = D.locks ? 1 : 1.04; // tanpa helai: lapisan dasar sedikit lebih tebal
      positions.push(0.81 * k * sp(Math.sin(theta) * Math.sin(phi), 0.82), 0.065 + 0.785 * k * sp(Math.cos(theta), 0.82), 0.76 * k * sp(Math.sin(theta) * Math.cos(phi), 0.82));
      uvs.push(col / around, 1 - row / rows);
      if (row < rows && col < around) { const a = row * (around + 1) + col, b = a + around + 1; indices.push(a, b, a + 1, a + 1, b, b + 1); }
    }
    const hg = new T.BufferGeometry(); hg.setAttribute('position', new T.Float32BufferAttribute(positions, 3)); hg.setAttribute('uv', new T.Float32BufferAttribute(uvs, 2)); hg.setIndex(indices); hg.computeVertexNormals();
    mesh(head, 'Hair underlayer', hg, hair);
    if (!D.locks) return;
    function hairLock(points, width, depth) {
      const curve = new T.CatmullRomCurve3(points.map((p) => new T.Vector3(...p))), pos = [], uv = [], idx = [];
      const steps = 20, sides = 10;
      for (let i = 0; i <= steps; i++) {
        const t = i / steps, p = curve.getPoint(t), tangent = curve.getTangent(t).normalize();
        const normal = new T.Vector3(p.x / 0.81 ** 2, (p.y - 0.065) / 0.785 ** 2, p.z / 0.76 ** 2).normalize();
        normal.addScaledVector(tangent, -normal.dot(tangent)).normalize();
        const side = new T.Vector3().crossVectors(tangent, normal).normalize();
        const full = Math.max(0.002, Math.sin(Math.PI * t) ** 0.43) * (1 - 0.16 * t), w = width * full, d = depth * full;
        for (let j = 0; j <= sides; j++) {
          const a = (j / sides) * Math.PI * 2, q = p.clone().addScaledVector(side, Math.cos(a) * w).addScaledVector(normal, Math.sin(a) * d);
          pos.push(q.x, q.y, q.z); uv.push(j / sides, t);
          if (i < steps && j < sides) { const A = i * (sides + 1) + j, B = A + sides + 1; idx.push(A, B, A + 1, A + 1, B, B + 1); }
        }
      }
      const g = new T.BufferGeometry(); g.setAttribute('position', new T.Float32BufferAttribute(pos, 3)); g.setAttribute('uv', new T.Float32BufferAttribute(uv, 2)); g.setIndex(idx); g.computeVertexNormals(); mesh(head, 'Hair lock', g, hair);
    }
    for (let i = 0; i < 10; i++) {
      const phi = 1.03 + i * (Math.PI * 2 - 2.06) / 9, back = Math.max(0, -Math.cos(phi)), end = 1.86 + 0.20 * back;
      hairLock([0.25, 0.78, 1.30, end].map((theta, k) => { const a = phi - 0.18 + k * 0.06; return [0.83 * sp(Math.sin(theta) * Math.sin(a), 0.82), 0.070 + 0.81 * sp(Math.cos(theta), 0.82), 0.785 * sp(Math.sin(theta) * Math.cos(a), 0.82)]; }), 0.151, 0.084);
    }
    hairLock([[-0.40, 0.64, 0.35], [-0.59, 0.40, 0.62], [-0.69, 0.10, 0.61], [-0.65, -0.15, 0.44]], 0.126, 0.093);
    hairLock([[-0.25, 0.79, 0.25], [-0.37, 0.60, 0.62], [-0.54, 0.31, 0.75], [-0.50, 0.13, 0.71]], 0.161, 0.112);
    hairLock([[-0.13, 0.84, 0.30], [-0.23, 0.62, 0.73], [-0.20, 0.39, 0.82], [0.012, 0.26, 0.78]], 0.166, 0.115);
    hairLock([[0.02, 0.84, 0.36], [0.12, 0.67, 0.72], [0.29, 0.44, 0.80], [0.35, 0.28, 0.72]], 0.159, 0.114);
    hairLock([[0.23, 0.78, 0.32], [0.41, 0.60, 0.64], [0.56, 0.32, 0.71], [0.53, 0.08, 0.62]], 0.159, 0.106);
    hairLock([[0.47, 0.64, 0.28], [0.65, 0.39, 0.43], [0.72, 0.09, 0.41], [0.66, -0.15, 0.39]], 0.123, 0.088);
    hairLock([[-0.18, 0.86, 0.17], [-0.26, 0.76, 0.52], [-0.12, 0.65, 0.81], [0.07, 0.72, 0.82]], 0.160, 0.113);
    hairLock([[-0.11, 0.89, 0.17], [0.10, 0.87, 0.45], [0.29, 0.78, 0.64], [0.47, 0.66, 0.57]], 0.160, 0.118);
  }

  // Khimar: tudung dengan bukaan wajah (dari character-hijab wedding-summer) + kain panjang
  // menutup bahu dan dada seperti atasan mukena.
  function addHijab() {
    const veilHex = opts.veil || '#efe9df';
    const veil = material(veilHex, 0.9), edgeM = material(new T.Color(veilHex).multiplyScalar(0.92), 0.85);
    const pw = (v) => Math.sign(v) * Math.abs(v) ** 0.8;
    const [rows, cols] = D.hood, v = [], uv = [], ix = [];
    for (let r = 0; r <= rows; r++) for (let c = 0; c <= cols; c++) {
      const th = 1.0 + (Math.PI - 1.0) * r / rows, phi = (c / cols) * Math.PI * 2;
      v.push(0.855 * pw(Math.sin(th) * Math.sin(phi)), 0.055 + 0.91 * pw(Math.sin(th) * Math.cos(phi)), 0.815 * pw(Math.cos(th)));
      uv.push(c / cols, r / rows);
      if (r < rows && c < cols) { const a = r * (cols + 1) + c, b = a + cols + 1; ix.push(a, a + 1, b, a + 1, b + 1, b); }
    }
    const hood = new T.BufferGeometry(); hood.setAttribute('position', new T.Float32BufferAttribute(v, 3)); hood.setAttribute('uv', new T.Float32BufferAttribute(uv, 2)); hood.setIndex(ix); hood.computeVertexNormals();
    const iv = [], iuv = [], iix = [], IR = D.fine ? 6 : 2, IC = cols;
    for (let row = 0; row <= IR; row++) for (let col = 0; col <= IC; col++) {
      const phi = (col / IC) * Math.PI * 2, u = row / IR;
      const x = 0.69 * Math.sin(phi), y = -0.005 + 0.635 * Math.cos(phi), w = faceWidth(y);
      const z = 0.72 * Math.max(0.005, 1 - Math.abs(x / w) ** 2.5 - Math.abs(y / 0.70) ** 2.5) ** 0.4 + 0.013;
      const ox = 0.855 * pw(Math.sin(1) * Math.sin(phi)), oy = 0.055 + 0.91 * pw(Math.sin(1) * Math.cos(phi)), oz = 0.815 * pw(Math.cos(1));
      iv.push(T.MathUtils.lerp(x, ox, u), T.MathUtils.lerp(y, oy, u), T.MathUtils.lerp(z, oz, u)); iuv.push(col / IC, u);
      if (row < IR && col < IC) { const a = row * (IC + 1) + col, b = a + IC + 1; iix.push(a, a + 1, b, a + 1, b + 1, b); }
    }
    const inset = new T.BufferGeometry(); inset.setAttribute('position', new T.Float32BufferAttribute(iv, 3)); inset.setAttribute('uv', new T.Float32BufferAttribute(iuv, 2)); inset.setIndex(iix); inset.computeVertexNormals();
    mesh(head, 'Hijab face surround', inset, veil);
    mesh(head, 'Hijab hood', hood, veil);
    const edge = [];
    for (let i = 0; i < 48; i++) { const p = (i / 48) * Math.PI * 2; edge.push([0.858 * pw(Math.sin(1) * Math.sin(p)), 0.055 + 0.913 * pw(Math.sin(1) * Math.cos(p)), 0.819 * pw(Math.cos(1))]); }
    if (!D.lite) line(head, 'Hijab face binding', edge, 0.018, edgeM, true);
    // Kain khimar panjang: menutup bahu dan lengan atas, bagian depan sedikit lebih panjang.
    // Profil A-line (bahu membulat, jatuh lurus ke bawah); depan & belakang lebih panjang dari sisi.
    const kh = lathe(body, 'Khimar', [[0.69, 1.02], [0.70, 1.12], [0.69, 1.30], [0.63, 1.48], [0.50, 1.66], [0.34, 1.83], [0.20, 1.93]], veil, 0.9, D.fine ? 48 : D.lathe);
    kh.geometry = kh.geometry.clone(); const p = kh.geometry.attributes.position;
    for (let i = 0; i < p.count; i++) {
      const x = p.getX(i), y = p.getY(i), z = p.getZ(i), a = Math.atan2(z, x);
      const low = T.MathUtils.clamp((1.5 - y) / 0.48, 0, 1), front = Math.max(0, Math.sin(a)), back = Math.max(0, -Math.sin(a));
      const f = 1 + 0.035 * low * Math.cos(a * 8);
      p.setXYZ(i, x * f * (1 - 0.06 * low), y - low * (0.20 * front ** 1.5 + 0.10 * back ** 1.5), z * f);
    }
    kh.geometry.computeVertexNormals(); kh.material.side = T.DoubleSide;
    if (D.fine) for (const s of [-1, 1]) line(body, 'Khimar fold', [[s * 0.22, 1.86, 0.20], [s * 0.38, 1.62, 0.40], [s * 0.42, 1.30, 0.52], [s * 0.36, 1.02, 0.56]], 0.012, edgeM);
  }

  // ---------- Lengan & kaki (pivot kaku; kaki memakai IK seperti rig asli) ----------
  const arms = [], legs = [], elbows = [], wrists = [], knees = [], feet = [];
  // Lengan model "kapsul bersendi": tiap ruas tabung meruncing yang diapit bola sendi berjari-jari sama,
  // jadi bahu-siku-pergelangan selalu menyambung mulus. Sendi bahu ditanam di tepi badan: saat lengan
  // berayun, bola bahu berputar di tempat dan tetap menyatu dengan badan.
  const seg = (parent, name, rTop, rBot, len, mat, y0 = 0) => {
    const g = new T.CylinderGeometry(rTop, rBot, len, D.lite ? 8 : 18, 1, true); g.translate(0, y0 - len / 2, 0);
    return mesh(parent, name, g, mat);
  };
  const ball = (parent, name, r, mat, y = 0) => ell(parent, name, mat, 0, y, 0, r, r, r);
  for (const s of [-1, 1]) {
    const right = s < 0, bareArm = ihram && right && opts.shoulderBare;
    const wide = !isM ? 1.08 : 1; // lengan baju perempuan sedikit lebih longgar
    const upperMat = !isM ? cloth : ihram ? skin : skin, foreMat = isM ? skin : cloth;
    const arm = new T.Group(); arm.name = right ? 'Right arm' : 'Left arm'; arm.position.set(s * 0.385, 1.46, 0); arm.rotation.z = s * ARM_SPLAY; body.add(arm); arms.push(arm);
    const R0 = 0.15 * wide, R1 = 0.125 * wide, R2 = 0.102 * wide, UP = 0.29, FORE = 0.22;
    ball(arm, 'Shoulder joint', R0, bareArm || isM ? skin : cloth);
    seg(arm, 'Upper arm', R0, R1, UP, upperMat);
    if (ihram && !bareArm) {
      // rida jatuh menutupi bahu & lengan atas (kain longgar, ujungnya sedikit mengembang)
      ell(arm, 'Rida shoulder', cloth, s * 0.01, 0.0, 0, 0.19, 0.18, 0.195);
      const drape = seg(arm, 'Rida sleeve', 0.185, 0.2, 0.2, cloth, -0.02); drape.material.side = T.DoubleSide;
      if (!D.lite) { const h = mesh(arm, 'Rida sleeve hem', new T.TorusGeometry(0.198, 0.014, 6, 24), hem); h.rotation.x = Math.PI / 2; h.position.y = -0.22; }
    } else if (isM && !ihram) {
      // kemeja lengan pendek
      ell(arm, 'Shirt shoulder', cloth, s * 0.01, 0.0, 0, 0.175, 0.17, 0.18);
      seg(arm, 'Shirt sleeve', 0.172, 0.165, 0.19, cloth, -0.02);
    }
    const elbow = new T.Group(); elbow.position.set(0, -UP, 0); arm.add(elbow); elbows.push(elbow);
    ball(elbow, 'Elbow joint', R1, foreMat);
    seg(elbow, 'Forearm', R1, R2, FORE, foreMat);
    if (!isM && !D.lite) { const c = mesh(elbow, 'Sleeve cuff', new T.TorusGeometry(R2 + 0.004, 0.016, 6, 24), hem); c.rotation.x = Math.PI / 2; c.position.y = -FORE + 0.02; }
    const wrist = new T.Group(); wrist.position.set(0, -FORE, 0); elbow.add(wrist); wrists.push(wrist);
    ball(wrist, 'Wrist', 0.1, skin);
    // tangan mitten (seperti rig wedding-summer) + ibu jari menghadap ke depan-dalam
    ell(wrist, 'Hand', skin, 0, -0.085, 0.012, 0.132, 0.142, 0.128);
    if (!D.lite) ell(wrist, 'Thumb', skin, -s * 0.07, -0.05, 0.095, 0.055, 0.07, 0.055);

    const leg = new T.Group(); leg.position.set(s * 0.216, 0.835, 0); body.add(leg); legs.push(leg);
    const thigh = isM && !ihram ? material('#3b3d48') : cloth, shin = isM ? (ihram ? skin : thigh) : cloth;
    const tw = isM && !ihram ? 1 : 0.72; // di balik izar/gamis paha dibuat ramping supaya tidak menembus kain saat melangkah
    const hidden = D.lite && (ihram || !isM); // kerumunan: bagian yang tertutup kain tidak dibuat
    if (!hidden) lathe(leg, 'Thigh', [[0, -0.365], [0.13 * tw, -0.365], [0.153 * tw, -0.31], [0.166 * tw, -0.04], [0.145 * tw, 0.05], [0, 0.06]], thigh, 1.08);
    const knee = new T.Group(); knee.position.y = -0.345; leg.add(knee); knees.push(knee);
    if (!hidden) ell(knee, 'Knee', ihram ? skin : thigh, 0, 0, 0, isM && !ihram ? 0.151 : 0.115, isM && !ihram ? 0.145 : 0.12, isM && !ihram ? 0.163 : 0.12);
    if (!(D.lite && !isM)) lathe(knee, 'Shin', ihram ? [[0, -0.335], [0.10, -0.335], [0.112, -0.24], [0.128, -0.06], [0.12, 0.025], [0, 0.035]] : [[0, -0.335], [0.135, -0.335], [0.152, -0.285], [0.153, -0.045], [0.14, 0.025], [0, 0.035]], shin, 1.08);
    const foot = new T.Group(); foot.position.y = -0.335; knee.add(foot); feet.push(foot);
    if (isM && ihram) {
      // Sandal ihram: punggung kaki & tumit terbuka, tali kulit menyilang.
      ell(foot, 'Bare foot', skin, 0, -0.03, 0.10, 0.15, 0.08, 0.235);
      if (!D.lite) ell(foot, 'Ankle', skin, 0, 0.045, -0.01, 0.10, 0.11, 0.10);
      ell(foot, 'Sandal sole', sole, 0, -0.105, 0.10, 0.18, 0.035, 0.29);
      if (!D.lite) line(foot, 'Sandal strap', [[-0.15, -0.06, 0.20], [-0.09, 0.03, 0.22], [0, 0.05, 0.225], [0.09, 0.03, 0.22], [0.15, -0.06, 0.20]], 0.03, leather);
      if (!D.lite) line(foot, 'Toe strap', [[-0.12, -0.06, 0.06], [0, 0.04, 0.09], [0.12, -0.06, 0.06]], 0.024, leather);
    } else if (isM) {
      ell(foot, 'Shoe', material('#2b2a30', 0.6), 0, 0.003, 0.105, 0.182, 0.14, 0.282);
      ell(foot, 'Shoe sole', material('#151519'), 0, -0.114, 0.101, 0.188, 0.041, 0.288);
    } else {
      // Kaus kaki & sepatu lembut perempuan (kaki tertutup).
      ell(foot, 'Soft shoe', material(opts.shoe || '#9a8f86', 0.7), 0, -0.01, 0.105, 0.17, 0.12, 0.27);
      ell(foot, 'Shoe sole', material('#5c524c', 0.8), 0, -0.105, 0.10, 0.176, 0.035, 0.278);
    }
  }
  return { root, body, head, arms, legs, elbows, wrists, knees, feet, eyes, material, skin };
}

// Gabungkan mesh per material di tiap pivot kaku (lebih sedikit draw call), seperti batchRigid wedding-summer.
function mergeGeos(parts) {
  let vCount = 0; const hasUV = parts.every((p) => p.geometry.attributes.uv);
  for (const p of parts) vCount += p.geometry.attributes.position.count;
  const pos = new Float32Array(vCount * 3), nor = new Float32Array(vCount * 3), uv = hasUV ? new Float32Array(vCount * 2) : null, idx = [];
  const extra = parts[0].attrs ? Object.keys(parts[0].attrs) : [], ex = Object.fromEntries(extra.map((k) => [k, new Float32Array(vCount * parts[0].attrs[k].length)]));
  let off = 0;
  for (const p of parts) {
    const g = p.geometry, n = g.attributes.position.count;
    pos.set(g.attributes.position.array, off * 3); nor.set(g.attributes.normal.array, off * 3);
    if (uv) uv.set(g.attributes.uv.array, off * 2);
    if (g.index) for (let i = 0; i < g.index.count; i++) idx.push(g.index.getX(i) + off); else for (let i = 0; i < n; i++) idx.push(i + off);
    for (const k of extra) { const v = p.attrs[k]; for (let i = 0; i < n; i++) for (let j = 0; j < v.length; j++) ex[k][(off + i) * v.length + j] = v[j]; }
    off += n;
  }
  const geo = new T.BufferGeometry();
  geo.setAttribute('position', new T.BufferAttribute(pos, 3)); geo.setAttribute('normal', new T.BufferAttribute(nor, 3));
  if (uv) geo.setAttribute('uv', new T.BufferAttribute(uv, 2));
  for (const k of extra) geo.setAttribute(k, new T.BufferAttribute(ex[k], parts[0].attrs[k].length));
  geo.setIndex(idx); geo.computeBoundingSphere(); return geo;
}
function batchRigid(group) {
  group.children.filter((o) => o.isGroup).forEach(batchRigid);
  const byMat = new Map();
  for (const o of [...group.children]) if (o.isMesh) { const a = byMat.get(o.material) || []; a.push(o); byMat.set(o.material, a); }
  for (const [m, objs] of byMat) {
    if (objs.length < 2) continue;
    const parts = objs.map((o) => { o.updateMatrix(); const g = o.geometry.clone(); g.applyMatrix4(o.matrix); return { geometry: g }; });
    const joined = new T.Mesh(mergeGeos(parts), m); joined.name = objs.map((o) => o.name).join(' / ');
    parts.forEach((p) => p.geometry.dispose()); objs.forEach((o) => group.remove(o)); group.add(joined);
  }
}

const smooth = (x) => { x = T.MathUtils.clamp(x, 0, 1); return x * x * (3 - 2 * x); };

// Karakter pemain: rig lengkap + animasi jalan/lari (IK kaki menapak) + isyarat tangan.
export function createChibi(gender, opts = {}) {
  const rig = buildRig(gender, opts);
  const { root, body, head, arms, legs, elbows, wrists, knees, feet, eyes } = rig;
  batchRigid(body);
  let blend = 0, runB = 0, phase = 0, turnLean = 0, gestW = 0, gestKind = 'right', time = 0;
  const invBody = new T.Matrix4(), target = new T.Vector3(), delta = new T.Vector3();
  const chain = new T.Quaternion(), footQ = new T.Quaternion(), axisX = new T.Vector3(1, 0, 0);
  const thigh = 0.345, shinL = 0.335;
  function poseLeg(i, z, lift, pitch) {
    const c = Math.cos(pitch), s = Math.sin(pitch);
    const support = 0.114 * c + 0.101 * s + Math.hypot(0.041 * c, 0.288 * s);
    target.set(i === 0 ? -0.216 : 0.216, support + lift, z).applyMatrix4(invBody);
    delta.copy(target).sub(legs[i].position);
    const sideways = Math.atan2(delta.x, -delta.y), down = Math.hypot(delta.x, delta.y), fwd = delta.z;
    const len = T.MathUtils.clamp(Math.hypot(down, fwd), 0.08, thigh + shinL - 0.00001);
    const alpha = Math.acos(T.MathUtils.clamp((thigh ** 2 + len ** 2 - shinL ** 2) / (2 * thigh * len), -1, 1));
    const bend = Math.acos(T.MathUtils.clamp((len ** 2 - thigh ** 2 - shinL ** 2) / (2 * thigh * shinL), -1, 1));
    legs[i].rotation.order = 'ZXY'; legs[i].rotation.set(Math.atan2(-fwd, down) - alpha, 0, sideways);
    knees[i].rotation.x = bend;
    chain.copy(body.quaternion).multiply(legs[i].quaternion).multiply(knees[i].quaternion).invert();
    footQ.setFromAxisAngle(axisX, pitch); feet[i].quaternion.copy(chain).multiply(footQ);
  }
  // m: { dt, speed, distance | cadence, turn, run, gesture: null | 'right' | 'doa' }  (semua dalam satuan lokal rig)
  function animate(m) {
    const dt = T.MathUtils.clamp(m.dt || 0, 0, 0.05); time += dt;
    const speed = Math.max(0, m.speed || 0), dist = Math.max(0, m.distance ?? speed * dt);
    const ratio = T.MathUtils.clamp(speed / WALK_SPEED, 0, 1), k = 1 - Math.exp(-dt * 12);
    blend += (smooth(speed / 0.32) - blend) * (1 - Math.exp(-dt * 16));
    runB += ((m.run && speed > 0 ? 1 : 0) - runB) * k;
    turnLean += (T.MathUtils.clamp(m.turn || 0, -5, 5) - turnLean) * (1 - Math.exp(-dt * 9));
    if (m.gesture) gestKind = m.gesture;
    gestW += ((m.gesture ? 1 : 0) - gestW) * (1 - Math.exp(-dt * 9));
    const stride = 0.62 + 0.48 * ratio + 0.22 * runB, stance = 0.5 - 0.12 * runB; // langkah sedikit pendek: kaki tetap di dalam kain
    // cadence (siklus/detik) dipakai bila simulasi dipercepat: langkah tetap wajar walau lajunya cepat.
    phase = m.phase != null ? m.phase : (phase + (m.cadence != null ? m.cadence * dt * blend : dist / stride)) % 1; // phase: untuk pose diam (pratinjau)
    const ang = phase * Math.PI * 2, wave = Math.cos(ang), idle = 1 - blend, t = time;
    body.position.set(Math.sin(ang) * 0.018 * blend, -0.012 - (0.055 + 0.085 * ratio) * blend + (0.010 + 0.035 * runB) * Math.cos(ang * 2) * blend + Math.sin(t * 2.1) * 0.006 * idle, 0);
    body.rotation.set((0.025 + 0.10 * runB) * blend + 0.005 * Math.sin(t * 2.1) * idle, 0.04 * Math.sin(ang) * blend, -0.024 * Math.sin(ang) * blend - turnLean * 0.017 * blend);
    body.updateMatrix(); invBody.copy(body.matrix).invert();
    for (let i = 0; i < 2; i++) {
      const cycle = (phase + i * 0.5) % 1, travel = stride * stance;
      let z, lift = 0, pitch = 0;
      if (cycle < stance) { const u = cycle / stance; z = travel * (0.5 - u); pitch = -0.10 * (1 - smooth(u / 0.18)) + 0.16 * smooth((u - 0.76) / 0.24); }
      else {
        const u = (cycle - stance) / (1 - stance), arc = Math.sin(Math.PI * u);
        z = travel * (u - 0.5) - stride * Math.sin(2 * Math.PI * u) / (2 * Math.PI); // kaki diam di tanah saat menapak
        lift = (0.06 + 0.03 * ratio + 0.09 * runB) * arc * arc;
        pitch = 0.16 * (1 - smooth(u)) - 0.10 * smooth((u - 0.63) / 0.37);
      }
      poseLeg(i, z * blend, lift * blend, pitch * blend);
      const sign = i === 0 ? 1 : -1, swing = sign * wave;
      arms[i].rotation.x = swing * (0.20 + 0.16 * ratio + 0.30 * runB) * blend - 0.025 * idle;
      arms[i].rotation.z = (i === 0 ? -1 : 1) * (ARM_SPLAY + 0.017 * Math.sin(t * 1.6) * idle) - turnLean * 0.012 * blend;
      arms[i].rotation.y = -sign * 0.045 * Math.sin(ang) * blend;
      elbows[i].rotation.x = -0.12 - (0.10 + 0.08 * Math.sin(ang + sign * 0.7)) * blend - 0.95 * runB * blend;
      wrists[i].rotation.x = 0.04 * Math.sin(ang - sign * 0.4) * blend;
      wrists[i].rotation.z = sign * 0.035 * Math.sin(ang - 0.45) * blend;
    }
    // Isyarat: 'right' = tangan kanan terangkat ke arah Hajar Aswad; 'doa' = kedua telapak terangkat di depan dada.
    if (gestW > 0.001) {
      const w = gestW, L = (o, prop, v) => (o.rotation[prop] += (v - o.rotation[prop]) * w);
      if (gestKind === 'doa') {
        for (let i = 0; i < 2; i++) { const s = i === 0 ? -1 : 1; L(arms[i], 'x', -1.2); L(arms[i], 'z', s * 0.32); L(arms[i], 'y', 0); L(elbows[i], 'x', -1.2); L(wrists[i], 'x', 0.7); L(wrists[i], 'z', -s * 0.5); }
      } else {
        L(arms[0], 'x', -2.35); L(arms[0], 'z', -0.22); L(arms[0], 'y', 0); L(elbows[0], 'x', -0.25); L(wrists[0], 'x', 0.5);
      }
    }
    head.rotation.set(-0.017 * blend + 0.012 * Math.sin(ang * 2 - 0.25) * blend - 0.05 * runB, 0.035 * Math.sin(t * 0.73) * idle - 0.024 * Math.sin(ang) * blend + turnLean * 0.032, -body.rotation.z * 0.6 + 0.012 * Math.sin(t * 1.3) * idle);
    const cyc = t % 4.8, pulse = (c, w) => Math.max(0, 1 - Math.abs(cyc - c) / w);
    const lid = Math.max(0.045, 1 - Math.max(pulse(3.65, 0.10), pulse(3.94, 0.08)));
    eyes.forEach((e) => (e.scale.y = lid));
  }
  animate({ dt: 0 });
  return { ...rig, animate };
}

// ---------- Kerumunan jamaah: chibi ringan, di-instance per jenis pakaian ----------
// Badan+kepala, 2 lengan, 2 kaki = 5 InstancedMesh per jenis. Warna kulit per-instance
// (hanya bagian kulit yang ikut berubah, lewat atribut skinMask).
function bakeGroup(group, stopAt = []) {
  // geometri dalam ruang lokal `group`, kecuali anak-anak yang ada di stopAt (pivot terpisah)
  const parts = []; group.updateMatrixWorld(true);
  const inv = new T.Matrix4().copy(group.matrixWorld).invert();
  group.traverse((o) => {
    if (!o.isMesh) return;
    for (let p = o.parent; p && p !== group; p = p.parent) if (stopAt.includes(p)) return;
    const g = o.geometry.clone(); g.applyMatrix4(new T.Matrix4().multiplyMatrices(inv, o.matrixWorld));
    const c = o.material.color, s = o.material.userData.skin;
    parts.push({ geometry: g, attrs: { color: s ? [s, s, s] : [c.r, c.g, c.b], skinMask: [s ? 1 : 0] } });
  });
  const geo = mergeGeos(parts); parts.forEach((p) => p.geometry.dispose()); return geo;
}
let crowdMat = null;
function crowdMaterial() {
  if (crowdMat) return crowdMat;
  crowdMat = new T.MeshLambertMaterial({ vertexColors: true });
  crowdMat.onBeforeCompile = (sh) => {
    sh.vertexShader = sh.vertexShader.replace('#include <color_vertex>', `
      vColor = vec3(1.0);
      vColor *= color;
      #ifdef USE_INSTANCING_COLOR
        vColor = mix(vColor, vColor * instanceColor.rgb, skinMask);
      #endif`).replace('void main() {', 'attribute float skinMask;\nvoid main() {');
  };
  return crowdMat;
}
export const SKIN_TONES = ['#f1c7a1', '#e8ad7e', '#d39a6e', '#b07a52', '#8a5a3a'];

// kinds: [{ gender, opts, count }]; return { add(person), update(list, dt) }
// Tiap person: { kind, x, z, yaw, scale, phase, speed(world/s), skin }
export function createCrowd(scene, kinds) {
  const shadowTex = (() => { const c = document.createElement('canvas'); c.width = c.height = 64; const g = c.getContext('2d'); const gr = g.createRadialGradient(32, 32, 0, 32, 32, 32); gr.addColorStop(0, 'rgba(0,0,0,.32)'); gr.addColorStop(1, 'rgba(0,0,0,0)'); g.fillStyle = gr; g.fillRect(0, 0, 64, 64); const t = new T.CanvasTexture(c); return t; })();
  const shadowGeo = new T.PlaneGeometry(1, 1); shadowGeo.rotateX(-Math.PI / 2);
  const total = kinds.reduce((a, k) => a + k.count, 0);
  const shadow = new T.InstancedMesh(shadowGeo, new T.MeshBasicMaterial({ map: shadowTex, transparent: true, depthWrite: false }), total);
  shadow.frustumCulled = false; scene.add(shadow);
  const mats = crowdMaterial();
  const K = kinds.map((k) => {
    const rig = buildRig(k.gender, { ...k.opts, detail: 'crowd', lambert: true });
    rig.root.updateMatrixWorld(true);
    // lengan sedikit menekuk di siku (pose santai) sebelum dipanggang
    rig.elbows.forEach((e) => (e.rotation.x = -0.25));
    const pivots = [...rig.arms, ...rig.legs];
    const mk = (geo) => { const m = new T.InstancedMesh(geo, mats, k.count); m.frustumCulled = false; m.count = 0; scene.add(m); return m; };
    const bodyGeo = bakeGroup(rig.body, pivots);
    const armGeo = rig.arms.map((a) => bakeGroup(a)), legGeo = rig.legs.map((l) => bakeGroup(l));
    const meshes = { body: mk(bodyGeo), arms: armGeo.map(mk), legs: legGeo.map(mk) };
    return { ...k, meshes, armPos: rig.arms.map((a) => a.position.clone()), armRotZ: rig.arms.map((a) => a.rotation.z), legPos: rig.legs.map((l) => l.position.clone()), n: 0 };
  });
  let keep = 1; // bagian jamaah yang digambar (mode ringan: 0.5)
  const people = [], _m = new T.Matrix4(), _r = new T.Matrix4(), _q = new T.Quaternion(), _p = new T.Vector3(), _s = new T.Vector3(), _e = new T.Euler(), col = new T.Color();
  function add(person) {
    const k = K[person.kind]; person.slot = k.n++; people.push(person);
    col.set(person.skin || SKIN_TONES[0]);
    for (const m of [k.meshes.body, ...k.meshes.arms, ...k.meshes.legs]) { m.setColorAt(person.slot, col); m.instanceColor.needsUpdate = true; }
    person.phase = person.phase ?? Math.random() * Math.PI * 2;
    return person;
  }
  const shown = (k) => Math.ceil(k.n * keep);
  function update(dt) {
    let vis = 0;
    for (const p of people) {
      const k = K[p.kind], sc = p.scale * RIG_SCALE;
      if (p.slot >= shown(k)) continue;
      const moving = p.speed > 0.05, rate = moving ? 1.25 + p.speed * 0.32 : 0; // langkah per detik (siklus)
      p.phase += dt * rate * Math.PI * 2;
      const sw = moving ? Math.sin(p.phase) : 0, bob = moving ? Math.abs(Math.cos(p.phase)) * 0.035 : 0;
      _q.setFromAxisAngle(_s.set(0, 1, 0), p.yaw);
      _m.compose(_p.set(p.x, bob * sc, p.z), _q, _s.set(sc, sc, sc));
      k.meshes.body.setMatrixAt(p.slot, _m);
      const ampL = k.gender === 'f' ? 0.32 : 0.5, ampA = 0.32;
      for (let i = 0; i < 2; i++) {
        const sgn = i === 0 ? 1 : -1;
        _r.makeRotationFromEuler(_e.set(-sgn * sw * ampL, 0, 0)); _r.setPosition(k.legPos[i]);
        k.meshes.legs[i].setMatrixAt(p.slot, _r.premultiply(_m));
        _r.makeRotationFromEuler(_e.set(sgn * sw * ampA, 0, k.armRotZ[i])); _r.setPosition(k.armPos[i]);
        k.meshes.arms[i].setMatrixAt(p.slot, _r.premultiply(_m));
      }
      _m.compose(_p.set(p.x, 0.02, p.z), _q.identity(), _s.set(1.5 * sc, 1, 1.5 * sc)); shadow.setMatrixAt(vis++, _m);
    }
    for (const k of K) for (const m of [k.meshes.body, ...k.meshes.arms, ...k.meshes.legs]) { m.count = shown(k); m.instanceMatrix.needsUpdate = true; }
    shadow.count = vis; shadow.instanceMatrix.needsUpdate = true;
  }
  // Mode ringan: hanya sebagian jamaah yang digambar (urutan masuk acak, jadi sebarannya tetap merata).
  const thin = (fraction) => { keep = fraction; };
  return { add, update, people, thin };
}
