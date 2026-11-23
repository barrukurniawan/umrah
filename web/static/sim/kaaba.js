// Ka'bah, Hajar Aswad, Rukun Yamani, Hijr Ismail, dan Maqam Ibrahim. Dipakai tawaf.js dan open world (haram.js).
// Ukuran asli (10 x 12 m, tinggi 13,1 m). Koordinat: x = timur, -z = utara, pusat Ka'bah di (0, 0).
// opts.rot   : rotasi y Ka'bah (bawaan PI/4 seperti simulasi tawaf; open world memakai nilai dari OSM)
// opts.maqam : [x, z] posisi Maqam Ibrahim (bawaan: di depan pintu, seperti simulasi tawaf)
import * as THREE from 'three';

const TAU = Math.PI * 2;
export const KW = 10, KD = 12, KH = 13.1;

export function buildKaaba(kit, scene, opts = {}) {
  const { M, mat, canvasTex } = kit;
  const KROT = opts.rot ?? Math.PI / 4;
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
  const MQ = opts.maqam ? new THREE.Vector3(opts.maqam[0], 0, opts.maqam[1]) : toWorld(KW / 2 + 9, 0, 2.6); maqam.position.copy(MQ); maqam.rotation.y = KROT;
  const mqBase = new THREE.Mesh(new THREE.CylinderGeometry(1.2, 1.3, 0.5, 8), mat('#e3dfd6')); mqBase.position.y = 0.25; maqam.add(mqBase);
  const mqCage = new THREE.Mesh(new THREE.CylinderGeometry(0.85, 0.85, 1.9, 8, 1, true), new THREE.MeshStandardMaterial({ color: '#d4a93a', metalness: 0.5, roughness: 0.4, side: THREE.DoubleSide, transparent: true, opacity: 0.85 }));
  mqCage.position.y = 1.45; maqam.add(mqCage);
  const mqDome = new THREE.Mesh(new THREE.SphereGeometry(0.9, 12, 8, 0, TAU, 0, Math.PI / 2), M.gold); mqDome.position.y = 2.4; maqam.add(mqDome);
  const mqTop = new THREE.Mesh(new THREE.ConeGeometry(0.12, 0.6, 6), M.gold); mqTop.position.y = 3.5; maqam.add(mqTop);
  scene.add(maqam);
  return { kaaba, body, toWorld, KROT, C_HA, C_YM, HIJR_MID, MQ, maqam, haFrame, ymMark, hijrPts };
}
