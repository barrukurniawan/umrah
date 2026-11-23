// Navigasi open world: tabrakan 2D (segmen dinding + lingkaran tiang), joystick, keyboard, geser-untuk-memutar kamera.
// Koordinat dunia: x = timur, z = selatan (sama dengan tawaf.js), satuan meter dunia.

// ---------- Tabrakan ----------
// Segmen disimpan di grid (sel CELL meter) supaya tiap langkah hanya memeriksa dinding di dekat pemain.
export function createCollision(cell = 8) {
  const segs = [], circles = [], grid = new Map(), seen = new Set();
  const key = (i, j) => i * 100003 + j;
  function addTo(i0, j0, i1, j1, item) {
    for (let i = i0; i <= i1; i++) for (let j = j0; j <= j1; j++) {
      const k = key(i, j); let c = grid.get(k); if (!c) grid.set(k, (c = [])); c.push(item);
    }
  }
  const api = {
    segs, circles,
    // h = tinggi penghalang bagi kamera (0 = tidak menghalangi); move = false: hanya untuk kamera (mis. ambang pintu).
    // tag: 'wall' dinding masjid, 'kaaba', '' lainnya.
    // lo = tepi bawah penghalang bagi kamera (ambang pintu: terbuka di bawah lo).
    addSeg(ax, az, bx, bz, { h = Infinity, lo = 0, move = true, tag = '' } = {}) {
      const s = { ax, az, bx, bz, h, lo, move, tag, id: segs.length, kind: 's' }; segs.push(s);
      addTo(Math.floor(Math.min(ax, bx) / cell), Math.floor(Math.min(az, bz) / cell), Math.floor(Math.max(ax, bx) / cell), Math.floor(Math.max(az, bz) / cell), s);
      return s;
    },
    addPoly(pts, o = {}) { // pts: [[x,z],...] tertutup
      for (let i = 0; i < pts.length; i++) { const a = pts[i], b = pts[(i + 1) % pts.length]; api.addSeg(a[0], a[1], b[0], b[1], o); }
    },
    addCircle(x, z, r) {
      const c = { x, z, r, id: circles.length, kind: 'c' }; circles.push(c);
      addTo(Math.floor((x - r) / cell), Math.floor((z - r) / cell), Math.floor((x + r) / cell), Math.floor((z + r) / cell), c);
      return c;
    },
    near(x, z, r, out = []) {
      out.length = 0; seen.clear();
      const i0 = Math.floor((x - r) / cell), i1 = Math.floor((x + r) / cell), j0 = Math.floor((z - r) / cell), j1 = Math.floor((z + r) / cell);
      for (let i = i0; i <= i1; i++) for (let j = j0; j <= j1; j++) {
        const c = grid.get(key(i, j)); if (!c) continue;
        for (const it of c) { const k = it.kind + it.id; if (!seen.has(k)) { seen.add(k); out.push(it); } }
      }
      return out;
    },
    // Dorong titik (radius r) keluar dari semua penghalang di dekatnya.
    resolve(p, r) {
      const list = api.near(p.x, p.z, r + 0.5, tmp);
      for (let iter = 0; iter < 3; iter++) {
        let moved = false;
        for (const it of list) {
          let qx, qz;
          if (it.kind === 's') {
            if (!it.move) continue;
            const dx = it.bx - it.ax, dz = it.bz - it.az, L2 = dx * dx + dz * dz || 1e-9;
            const u = Math.max(0, Math.min(1, ((p.x - it.ax) * dx + (p.z - it.az) * dz) / L2));
            qx = it.ax + u * dx; qz = it.az + u * dz;
            const ex = p.x - qx, ez = p.z - qz, d = Math.hypot(ex, ez);
            if (d < r) {
              if (d > 1e-6) { p.x = qx + ex / d * r; p.z = qz + ez / d * r; }
              else { const L = Math.sqrt(L2); p.x += -dz / L * r; p.z += dx / L * r; }
              moved = true;
            }
          } else {
            const ex = p.x - it.x, ez = p.z - it.z, d = Math.hypot(ex, ez), rr = r + it.r;
            if (d < rr) { const s = d > 1e-6 ? rr / d : 0; p.x = it.x + (s ? ex * s : rr); p.z = it.z + (s ? ez * s : 0); moved = true; }
          }
        }
        if (!moved) break;
      }
    },
    // Gerak dengan "geser" di sepanjang dinding; langkah dipecah supaya tidak menembus dinding tipis.
    move(p, dx, dz, r) {
      const len = Math.hypot(dx, dz), n = Math.max(1, Math.ceil(len / (r * 0.7)));
      for (let i = 0; i < n; i++) { p.x += dx / n; p.z += dz / n; api.resolve(p, r); }
    },
    // Adakah lingkaran (tiang, menara, dsb.) yang dipotong garis datar a->b (diperlebar pad)?
    circleOnLine(ax, az, bx, bz, pad = 0) {
      const x0 = Math.min(ax, bx), x1 = Math.max(ax, bx), z0 = Math.min(az, bz), z1 = Math.max(az, bz);
      const rx = bx - ax, rz = bz - az, L2 = rx * rx + rz * rz || 1e-9;
      for (const it of api.near((x0 + x1) / 2, (z0 + z1) / 2, Math.max(x1 - x0, z1 - z0) / 2 + 2, tmp2)) {
        if (it.kind !== 'c') continue;
        const u = Math.max(0, Math.min(1, ((it.x - ax) * rx + (it.z - az) * rz) / L2));
        if (Math.hypot(ax + rx * u - it.x, az + rz * u - it.z) < it.r + pad) return true;
      }
      return false;
    },
    // Semua penghalang (tinggi > 0) yang dipotong garis datar a->b: [{u: 0..1 di sepanjang garis, h, tag}], urut u.
    crossings(ax, az, bx, bz, out = []) {
      out.length = 0;
      const x0 = Math.min(ax, bx), x1 = Math.max(ax, bx), z0 = Math.min(az, bz), z1 = Math.max(az, bz);
      const rx = bx - ax, rz = bz - az;
      for (const it of api.near((x0 + x1) / 2, (z0 + z1) / 2, Math.max(x1 - x0, z1 - z0) / 2 + 1, tmp2)) {
        if (it.kind !== 's' || !(it.h > 0)) continue;
        const sx = it.bx - it.ax, sz = it.bz - it.az, den = rx * sz - rz * sx;
        if (Math.abs(den) < 1e-9) continue;
        const u = ((it.ax - ax) * sz - (it.az - az) * sx) / den, v = ((it.ax - ax) * rz - (it.az - az) * rx) / den;
        if (u > 0 && u <= 1 && v >= 0 && v <= 1) out.push({ u, h: it.h, lo: it.lo, tag: it.tag });
      }
      return out.sort((p, q) => p.u - q.u);
    },
  };
  const tmp = [], tmp2 = [];
  return api;
}

// ---------- Kendali ----------
// Joystick kiri bawah (HP), WASD/panah (desktop), geser layar untuk memutar kamera, cubit/scroll untuk zoom
// (zoom ditangani common.js). Hasil: input.move = {x: kanan, y: maju} dalam [-1, 1].
export function createControls({ stage, canvas, joyEl, knobEl }) {
  const st = { x: 0, y: 0, yawDelta: 0, pitchDelta: 0, dragging: false, lastDrag: 0, keys: new Set() };
  // Joystick
  let joyId = null, cx = 0, cy = 0, R = 46;
  const setKnob = (x, y) => { knobEl.style.transform = `translate(${x * R}px, ${y * R}px)`; };
  joyEl.addEventListener('pointerdown', (e) => {
    e.preventDefault(); joyId = e.pointerId; joyEl.setPointerCapture(e.pointerId);
    const r = joyEl.getBoundingClientRect(); cx = r.left + r.width / 2; cy = r.top + r.height / 2; R = r.width * 0.36; onJoy(e);
  });
  function onJoy(e) {
    if (e.pointerId !== joyId) return;
    let dx = (e.clientX - cx) / R, dy = (e.clientY - cy) / R; const m = Math.hypot(dx, dy);
    if (m > 1) { dx /= m; dy /= m; }
    st.jx = dx; st.jy = -dy; setKnob(dx, dy);
  }
  joyEl.addEventListener('pointermove', onJoy);
  const joyUp = (e) => { if (e.pointerId !== joyId) return; joyId = null; st.jx = st.jy = 0; setKnob(0, 0); };
  ['pointerup', 'pointercancel', 'lostpointercapture'].forEach((ev) => joyEl.addEventListener(ev, joyUp));
  joyEl.addEventListener('contextmenu', (e) => e.preventDefault());
  // Geser untuk memutar kamera (1 jari / mouse). Dua jari = cubit zoom (diurus common.js), jadi diabaikan di sini.
  const ptrs = new Map();
  function bindCanvas(cv) {
    cv.addEventListener('pointerdown', (e) => { ptrs.set(e.pointerId, { x: e.clientX, y: e.clientY, t0: performance.now(), x0: e.clientX, y0: e.clientY }); });
    cv.addEventListener('pointermove', (e) => {
      const p = ptrs.get(e.pointerId); if (!p) return;
      if (ptrs.size === 1) {
        const w = stage.clientWidth || 800;
        st.yawDelta -= (e.clientX - p.x) / w * 3.2; st.pitchDelta += (e.clientY - p.y) / w * 2.2;
        st.dragging = true; st.lastDrag = performance.now();
      }
      p.x = e.clientX; p.y = e.clientY;
    });
    const up = (e) => {
      const p = ptrs.get(e.pointerId); ptrs.delete(e.pointerId);
      if (!ptrs.size) st.dragging = false;
      if (p && api.onTap && performance.now() - p.t0 < 300 && Math.hypot(e.clientX - p.x0, e.clientY - p.y0) < 8) api.onTap(e);
    };
    ['pointerup', 'pointercancel', 'pointerleave'].forEach((ev) => cv.addEventListener(ev, up));
  }
  bindCanvas(canvas);
  // Keyboard: W/panah atas maju, S/bawah mundur, A/D atau kiri/kanan geser samping, Q/E putar kamera.
  const K = { KeyW: 'f', ArrowUp: 'f', KeyS: 'b', ArrowDown: 'b', KeyA: 'l', ArrowLeft: 'l', KeyD: 'r', ArrowRight: 'r', KeyQ: 'q', KeyE: 'e', ShiftLeft: 'run', ShiftRight: 'run' };
  addEventListener('keydown', (e) => { const k = K[e.code]; if (!k || e.target.closest?.('input,select,textarea')) return; st.keys.add(k); e.preventDefault(); });
  addEventListener('keyup', (e) => { const k = K[e.code]; if (k) st.keys.delete(k); });
  addEventListener('blur', () => st.keys.clear());
  const api = {
    st, bindCanvas, onTap: null,
    // Dibaca tiap frame: vektor gerak + putaran kamera dari keyboard; reset delta geser.
    read(dt) {
      const k = st.keys;
      let x = (st.jx || 0) + (k.has('r') ? 1 : 0) - (k.has('l') ? 1 : 0);
      let y = (st.jy || 0) + (k.has('f') ? 1 : 0) - (k.has('b') ? 1 : 0);
      const m = Math.hypot(x, y); if (m > 1) { x /= m; y /= m; }
      const yaw = st.yawDelta + ((k.has('q') ? 1 : 0) - (k.has('e') ? 1 : 0)) * 1.8 * dt, pitch = st.pitchDelta;
      st.yawDelta = 0; st.pitchDelta = 0;
      return { x, y, yaw, pitch, run: k.has('run'), dragging: st.dragging || performance.now() - st.lastDrag < 1200 };
    },
  };
  return api;
}
