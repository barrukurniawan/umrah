// Alat geometri open world: pembangun mesh gabungan (per sektor), bantuan poligon, dan peta wilayah (grid).
// Koordinat: x = timur, z = selatan, pusat Ka'bah (0, 0).
import * as THREE from 'three';

export const TAU = Math.PI * 2;

// ---------- Geometri gabungan (non-indexed, warna per verteks) ----------
// sectors > 0: segitiga dibagi ke N irisan sudut dari Ka'bah (menurut titik beratnya), jadi mesh() memberi Group
// berisi N mesh dengan bounding sphere kecil: frustum culling membuang irisan yang tidak terlihat.
export function Builder(sectors = 0) {
  const N = sectors || 1;
  const B = Array.from({ length: N }, () => ({ pos: [], nor: [], col: [], uv: [] }));
  const c = new THREE.Color();
  const _a = new THREE.Vector3(), _b = new THREE.Vector3(), _n = new THREE.Vector3();
  const V = (x, y, z) => new THREE.Vector3(x, y, z);
  const UP = V(0, 1, 0), DOWN = V(0, -1, 0);
  function tri(A, Bv, C, n, color, uvs) {
    _a.subVectors(Bv, A); _b.subVectors(C, A); _n.crossVectors(_a, _b);
    if (_n.dot(n) < 0) { [Bv, C] = [C, Bv]; if (uvs) uvs = [uvs[0], uvs[2], uvs[1]]; }
    let k = 0;
    if (sectors) { const a = Math.atan2((A.z + Bv.z + C.z) / 3, (A.x + Bv.x + C.x) / 3); k = Math.min(N - 1, Math.floor((a / TAU + 1) % 1 * N)); }
    const b = B[k];
    for (const v of [A, Bv, C]) { b.pos.push(v.x, v.y, v.z); b.nor.push(n.x, n.y, n.z); b.col.push(color.r, color.g, color.b); }
    if (uvs) for (const t of uvs) b.uv.push(t[0], t[1]); else b.uv.push(0, 0, 0, 0, 0, 0);
  }
  const api = {
    V, tri,
    color(col) { return c.set(col); },
    // Segi empat A-B-C-D (searah keliling) menghadap n.
    quad(A, Bv, C, D, n, color, uvs) {
      c.set(color);
      tri(A, Bv, C, n, c, uvs && [uvs[0], uvs[1], uvs[2]]); tri(A, C, D, n, c, uvs && [uvs[0], uvs[2], uvs[3]]);
    },
    // Tutup datar (poligon + lubang) pada ketinggian y. uvFn opsional: (x, z) -> [u, v].
    cap(outer, holes, y, color, up = true, uvFn = null) {
      c.set(color);
      const contour = outer.map((p) => new THREE.Vector2(p[0], p[1]));
      const hs = (holes || []).map((h) => h.map((p) => new THREE.Vector2(p[0], p[1])));
      const all = contour.concat(...hs);
      for (const [i, j, k] of THREE.ShapeUtils.triangulateShape(contour, hs)) {
        const P = [all[i], all[j], all[k]];
        tri(V(P[0].x, y, P[0].y), V(P[1].x, y, P[1].y), V(P[2].x, y, P[2].y), up ? UP : DOWN, c, uvFn ? P.map((p) => uvFn(p.x, p.y)) : null);
      }
    },
    // Dinding vertikal sepanjang ring. Normal keluar dari arah putaran ring (luas bertanda);
    // flip = true untuk lubang. uv = {w, h}: u = jarak sepanjang ring / w, v = y / h (tekstur fasad berulang).
    sides(ring, y0, y1, color, flip = false, uv = null, closed = true) {
      c.set(color);
      const sgn = (ringArea(ring) > 0 ? 1 : -1) * (flip ? -1 : 1);
      let acc = 0;
      for (let i = 0; i < ring.length - (closed ? 0 : 1); i++) {
        const a = ring[i], b = ring[(i + 1) % ring.length];
        const dx = b[0] - a[0], dz = b[1] - a[1], L = Math.hypot(dx, dz); if (L < 1e-4) continue;
        const n = V(sgn * dz / L, 0, -sgn * dx / L);
        const A0 = V(a[0], y0, a[1]), B0 = V(b[0], y0, b[1]), A1 = V(a[0], y1, a[1]), B1 = V(b[0], y1, b[1]);
        const u0 = acc / (uv ? uv.w : 1), u1 = (acc + L) / (uv ? uv.w : 1), v0 = uv ? y0 / uv.h : 0, v1 = uv ? y1 / uv.h : 0;
        tri(A0, B0, B1, n, c, uv && [[u0, v0], [u1, v0], [u1, v1]]); tri(A0, B1, A1, n, c, uv && [[u0, v0], [u1, v1], [u0, v1]]);
        acc += L;
      }
    },
    prism(ring, y0, y1, color, bottom = false, uv = null) { api.sides(ring, y0, y1, color, false, uv); api.cap(ring, null, y1, color, true); if (bottom) api.cap(ring, null, y0, color, false); },
    // Balok tipis dari a ke b (dinding). uv: lihat sides (u dihitung dari titik a).
    slab(a, b, t, y0, y1, color, uv = null, u0 = 0) {
      const dx = b[0] - a[0], dz = b[1] - a[1], L = Math.hypot(dx, dz); if (L < 0.05) return;
      const nx = -dz / L * t / 2, nz = dx / L * t / 2;
      const ring = [[a[0] + nx, a[1] + nz], [b[0] + nx, b[1] + nz], [b[0] - nx, b[1] - nz], [a[0] - nx, a[1] - nz]];
      if (uv) {
        // muka panjang bertekstur; u mulai dari u0 supaya pola menyambung antarpotongan dinding
        c.set(color);
        const sgn = ringArea(ring) > 0 ? 1 : -1, va = y0 / uv.h, vb = y1 / uv.h;
        const face = (p, q, ua, ub) => {
          const ex = q[0] - p[0], ez = q[1] - p[1], l = Math.hypot(ex, ez) || 1, n = V(sgn * ez / l, 0, -sgn * ex / l);
          tri(V(p[0], y0, p[1]), V(q[0], y0, q[1]), V(q[0], y1, q[1]), n, c, [[ua, va], [ub, va], [ub, vb]]);
          tri(V(p[0], y0, p[1]), V(q[0], y1, q[1]), V(p[0], y1, p[1]), n, c, [[ua, va], [ub, vb], [ua, vb]]);
        };
        const ua = u0 / uv.w, ub = (u0 + L) / uv.w;
        face(ring[0], ring[1], ua, ub); face(ring[1], ring[2], ub, ub); face(ring[2], ring[3], ub, ua); face(ring[3], ring[0], ua, ua);
        api.cap(ring, null, y1, color, true); if (y0 > 0.1) api.cap(ring, null, y0, color, false);
        return;
      }
      api.prism(ring, y0, y1, color, y0 > 0.1);
    },
    mesh(material) {
      const meshes = [];
      for (const b of B) {
        if (!b.pos.length) continue;
        const g = new THREE.BufferGeometry();
        g.setAttribute('position', new THREE.Float32BufferAttribute(b.pos, 3));
        g.setAttribute('normal', new THREE.Float32BufferAttribute(b.nor, 3));
        g.setAttribute('color', new THREE.Float32BufferAttribute(b.col, 3));
        g.setAttribute('uv', new THREE.Float32BufferAttribute(b.uv, 2));
        g.computeBoundingSphere();
        meshes.push(new THREE.Mesh(g, material));
      }
      if (!sectors) return meshes[0] || new THREE.Mesh(new THREE.BufferGeometry(), material);
      const grp = new THREE.Group(); meshes.forEach((m) => grp.add(m)); return grp;
    },
    get count() { return B.reduce((s, b) => s + b.pos.length / 9, 0); },
  };
  return api;
}

// ---------- Bantuan poligon ----------
export function ringArea(r) {
  let a = 0; for (let i = 0; i < r.length; i++) { const p = r[i], q = r[(i + 1) % r.length]; a += p[0] * q[1] - q[0] * p[1]; }
  return a / 2;
}
export function inPoly(x, z, ring) {
  let c = false;
  for (let i = 0, j = ring.length - 1; i < ring.length; j = i++) {
    const [x1, z1] = ring[i], [x2, z2] = ring[j];
    if ((z1 > z) !== (z2 > z) && x < (x2 - x1) * (z - z1) / (z2 - z1) + x1) c = !c;
  }
  return c;
}
export function segDist(px, pz, a, b) {
  const dx = b[0] - a[0], dz = b[1] - a[1], L2 = dx * dx + dz * dz || 1e-9;
  const u = Math.max(0, Math.min(1, ((px - a[0]) * dx + (pz - a[1]) * dz) / L2));
  return { d: Math.hypot(px - a[0] - u * dx, pz - a[1] - u * dz), u };
}
// Titik-titik berjarak ±spacing sepanjang ring (tengah tiap ruas), dengan arah ruasnya.
export function edgePoints(ring, spacing, closed = true) {
  const out = [];
  for (let i = 0; i < ring.length - (closed ? 0 : 1); i++) {
    const a = ring[i], b = ring[(i + 1) % ring.length], L = Math.hypot(b[0] - a[0], b[1] - a[1]);
    if (L < 1e-3) continue;
    const n = Math.max(1, Math.round(L / spacing)), dir = Math.atan2(b[1] - a[1], b[0] - a[0]);
    for (let k = 0; k < n; k++) { const t = (k + 0.5) / n; out.push({ x: a[0] + (b[0] - a[0]) * t, z: a[1] + (b[1] - a[1]) * t, dir, edge: i, k, n, len: L / n }); }
  }
  return out;
}
// Geser ring sejauh d ke luar (d < 0: ke dalam) dengan sambungan miter (dibatasi supaya sudut tajam tidak memanjang).
export function offsetRing(ring, d) {
  const s = ringArea(ring) > 0 ? 1 : -1, n = ring.length, out = [];
  const nrm = (a, b) => { const dx = b[0] - a[0], dz = b[1] - a[1], L = Math.hypot(dx, dz) || 1; return [s * dz / L, -s * dx / L]; };
  for (let i = 0; i < n; i++) {
    const p = ring[(i - 1 + n) % n], c = ring[i], q = ring[(i + 1) % n];
    const n1 = nrm(p, c), n2 = nrm(c, q); let mx = n1[0] + n2[0], mz = n1[1] + n2[1];
    const ml = Math.hypot(mx, mz) || 1; mx /= ml; mz /= ml;
    const cos = Math.max(0.35, mx * n1[0] + mz * n1[1]);
    out.push([c[0] + mx * d / cos, c[1] + mz * d / cos]);
  }
  return out;
}
// Jarak titik ke ring (ruas terdekat).
export function ringDist(x, z, ring) {
  let best = Infinity;
  for (let i = 0; i < ring.length; i++) { const d = segDist(x, z, ring[i], ring[(i + 1) % ring.length]).d; if (d < best) best = d; }
  return best;
}

// ---------- Peta wilayah (grid 0,5 m) ----------
// Kode per sel: 0 pelataran / luar, lalu kode yang diberikan ke fill(). Diisi dengan scanline (cepat, tanpa inPoly per sel).
export function createRegions(bounds, cell = 0.5, pad = 20) {
  const x0 = bounds[0] - pad, z0 = bounds[1] - pad;
  const W = Math.ceil((bounds[2] - bounds[0] + 2 * pad) / cell), H = Math.ceil((bounds[3] - bounds[1] + 2 * pad) / cell);
  const g = new Uint8Array(W * H);
  return {
    g, W, H, cell, x0, z0,
    fill(ring, code) {
      const xs = [];
      for (let j = 0; j < H; j++) {
        const zc = z0 + (j + 0.5) * cell; xs.length = 0;
        for (let i = 0, k = ring.length - 1; i < ring.length; k = i++) {
          const [xa, za] = ring[i], [xb, zb] = ring[k];
          if ((za > zc) !== (zb > zc)) xs.push(xa + (zc - za) * (xb - xa) / (zb - za));
        }
        if (xs.length < 2) continue;
        xs.sort((p, q) => p - q);
        for (let m = 0; m + 1 < xs.length; m += 2) {
          const i0 = Math.max(0, Math.ceil((xs[m] - x0) / cell - 0.5)), i1 = Math.min(W - 1, Math.floor((xs[m + 1] - x0) / cell - 0.5));
          for (let i = i0; i <= i1; i++) g[j * W + i] = code;
        }
      }
    },
    at(x, z) {
      const i = Math.floor((x - x0) / cell), j = Math.floor((z - z0) / cell);
      return i < 0 || j < 0 || i >= W || j >= H ? 0 : g[j * W + i];
    },
  };
}

// ---------- Tekstur kanvas ----------
// Lengkung runcing (gaya serambi Masjidil Haram) sebagai panel ber-alpha: bagian di bawah lengkung transparan.
export function archPath(g, x0, y0, w, h, spring, closed = true) {
  // x0,y0 = kiri atas bukaan (y ke bawah); spring = tinggi bagian lurus (dari bawah) sebelum melengkung
  const xm = x0 + w / 2, yb = y0 + h, ys = yb - spring;
  g.moveTo(x0, yb); g.lineTo(x0, ys);
  g.quadraticCurveTo(x0, y0 + (ys - y0) * 0.25, xm, y0);
  g.quadraticCurveTo(x0 + w, y0 + (ys - y0) * 0.25, x0 + w, ys);
  g.lineTo(x0 + w, yb); if (closed) g.closePath();
}

// Gabung beberapa geometri (posisi, normal, uv) jadi satu, mis. tiang = batang + kepala + alas (satu draw call).
// Beri atribut warna rata ke geometri (untuk digabung dengan mergeGeos + bahan vertexColors).
export function paint(geo, color) {
  const g = geo.index ? geo.toNonIndexed() : geo, c = new THREE.Color(color), n = g.attributes.position.count, a = new Float32Array(n * 3);
  for (let i = 0; i < n; i++) { a[i * 3] = c.r; a[i * 3 + 1] = c.g; a[i * 3 + 2] = c.b; }
  g.setAttribute('color', new THREE.BufferAttribute(a, 3)); return g;
}
export function mergeGeos(geos) {
  const parts = geos.map((g) => (g.index ? g.toNonIndexed() : g));
  const out = new THREE.BufferGeometry();
  const names = ['position', 'normal', 'uv', ...(parts.every((p) => p.attributes.color) ? ['color'] : [])];
  for (const name of names) {
    const arrs = parts.map((p) => p.attributes[name].array), arr = new Float32Array(arrs.reduce((s, a) => s + a.length, 0));
    let o = 0; arrs.forEach((a) => { arr.set(a, o); o += a.length; });
    out.setAttribute(name, new THREE.BufferAttribute(arr, parts[0].attributes[name].itemSize));
  }
  out.computeBoundingSphere();
  return out;
}

// InstancedMesh dibagi per irisan sudut dari Ka'bah (item: {x, z} atau [x, z]) supaya frustum culling bekerja.
export function sectorInstanced(sim, geo, material, items, place, N = 8, parent) {
  const buckets = Array.from({ length: N }, () => []);
  for (const it of items) {
    const x = it.x ?? it[0], z = it.z ?? it[1], a = Math.atan2(z, x);
    buckets[Math.min(N - 1, Math.floor((a / TAU + 1) % 1 * N))].push(it);
  }
  return buckets.filter((b) => b.length).map((b) => sim.instanced(geo, material, b, place, parent));
}

// Jarak dari titik asal sepanjang arah (dx, dz) sampai menembus ring (Infinity bila tidak kena).
export function rayRing(dx, dz, ring, ox = 0, oz = 0) {
  let best = Infinity;
  for (let i = 0; i < ring.length; i++) {
    const a = ring[i], b = ring[(i + 1) % ring.length], sx = b[0] - a[0], sz = b[1] - a[1], den = dx * sz - dz * sx;
    if (Math.abs(den) < 1e-12) continue;
    const t = ((a[0] - ox) * sz - (a[1] - oz) * sx) / den, v = ((a[0] - ox) * dz - (a[1] - oz) * dx) / den;
    if (t > 0 && v >= 0 && v <= 1 && t < best) best = t;
  }
  return best;
}
