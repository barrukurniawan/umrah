// Bagian bersama simulasi Umrohku (tawaf, sa'i & jelajah): renderer, karakter chibi, kotak pemandu,
// tombol Jalan, audio murottal, label, dan loop. Tiap adegan mengisi `sim.hooks`.
import * as THREE from 'three';
import { createChibi, createCrowd, RIG_SCALE, SKIN_TONES, LOOKS } from './chibi.js?v=49';
export { THREE, createCrowd, SKIN_TONES };

export const $ = (id) => document.getElementById(id);
export const TAU = Math.PI * 2;
export const mod = (a, n) => ((a % n) + n) % n;

export function chibiKit() {
  // ---------- Material, tekstur & karakter chibi (juga dipakai pembuat ilustrasi) ----------
  const mat = (color, o = {}) => new THREE.MeshLambertMaterial({ color, ...o });
  const M = {
    marble: mat('#f1f0ec'), stone: mat('#e6dcc4'), stoneDark: mat('#cdbf9f'), pillar: mat('#efe6d2'),
    arch: mat('#b98d5a'), dome: mat('#e9e2d3'), dark: mat('#4d453b'), black: mat('#121212'),
    gold: new THREE.MeshStandardMaterial({ color: '#d4a93a', metalness: 0.6, roughness: 0.35 }),
    silver: new THREE.MeshStandardMaterial({ color: '#d9d9d9', metalness: 0.8, roughness: 0.3 }),
    tile: mat('#5d7fae'), white: mat('#fbfbf6'), skin: mat('#f1c7a1'), hair: mat('#2b2420'),
  };
  function canvasTex(w, h, draw) {
    const c = document.createElement('canvas'); c.width = w; c.height = h;
    draw(c.getContext('2d'), w, h);
    const t = new THREE.CanvasTexture(c); t.colorSpace = THREE.SRGBColorSpace; t.anisotropy = 4;
    return t;
  }
  // ---------- Karakter chibi (rig gaya wedding-summer di chibi.js; menghadap +z, sisi kanannya di -x) ----------
  // clothColor/veilColor: untuk ilustrasi (mis. tukang cukur berbaju biasa).
  // draped: ihram dengan kedua bahu tertutup (ilustrasi Niat/Tahallul).
  const shadowMat = new THREE.MeshBasicMaterial({ map: canvasTex(64, 64, (c, s) => { const gr = c.createRadialGradient(s / 2, s / 2, 0, s / 2, s / 2, s / 2); gr.addColorStop(0, 'rgba(0,0,0,.35)'); gr.addColorStop(1, 'rgba(0,0,0,0)'); c.fillStyle = gr; c.fillRect(0, 0, s, s); }), transparent: true, depthWrite: false });
  // look: preset tampilan (chibi.js LOOKS), mis. 'beard' = pria berjenggot rambut ikal.
  function makeChibi(gender, { shoulderBare = gender === 'm', scale = 2, clothColor = null, veilColor = null, draped = false, skin = null, bald = false, look = null } = {}) {
    const rig = createChibi(gender, {
      shoulderBare: shoulderBare && !draped, outfit: gender === 'm' && clothColor ? 'casual' : 'ihram',
      cloth: clothColor, veil: veilColor, skin: skin || (look && LOOKS[look] && LOOKS[look].skin) || '#eab48a', bald, look,
    });
    const g = new THREE.Group(); rig.root.scale.setScalar(RIG_SCALE); g.add(rig.root);
    const sh = new THREE.Mesh(new THREE.CircleGeometry(0.75, 20), shadowMat); sh.rotation.x = -Math.PI / 2; sh.position.y = 0.02; g.add(sh);
    g.scale.setScalar(scale);
    const parts = { head: rig.head, armR: rig.arms[0], armL: rig.arms[1], legR: rig.legs[0], legL: rig.legs[1] };
    return { g, parts, rig };
  }
  return { M, mat, canvasTex, makeChibi };
}

export function createSim(opts = {}) {
  const bg = opts.bg || '#e7ede6';
  // Mode ringan untuk HP lemah: ketajaman 1x, tanpa antialias, jamaah separuh. Diingat selama sesi
  // (pindah Tawaf -> Sa'i langsung ringan tanpa mengukur ulang).
  const LITE_KEY = 'ukSimLite';
  let startLite = false; try { startLite = sessionStorage.getItem(LITE_KEY) === '1'; } catch (e) {}
  const makeRenderer = (cv, lite) => {
    const r = new THREE.WebGLRenderer({ canvas: cv, antialias: !lite, powerPreference: 'low-power' });
    r.setPixelRatio(lite ? 1 : Math.min(window.devicePixelRatio || 1, 1.5));
    r.outputColorSpace = THREE.SRGBColorSpace; return r;
  };
  const stage = $('simStage');
  let canvas = $('simCanvas'), renderer = makeRenderer(canvas, startLite);
  const scene = new THREE.Scene();
  scene.background = new THREE.Color(bg);
  scene.fog = new THREE.Fog(bg, opts.fogNear || 130, opts.fogFar || 260);
  const camera = new THREE.PerspectiveCamera(opts.fov || 42, 1, 0.5, opts.far || 420);
  // Masjidil Haram: batu putih gading (bukan krem kekuningan); pantulan tanah & matahari netral.
  scene.add(new THREE.HemisphereLight('#ffffff', '#cfcdc8', 1.25));
  const sun = new THREE.DirectionalLight('#fffbf3', 1.5);
  sun.position.set(-40, 80, 30);
  scene.add(sun);

  const { M, mat, canvasTex, makeChibi } = chibiKit();
  // batu bangunan di simulasi 3D: putih gading, lengkung krem lembut (ilustrasi chibiKit tetap memakai warna asal)
  for (const [k, c] of [['stone', '#ece9e3'], ['stoneDark', '#cdc9c1'], ['pillar', '#f1efeb'], ['arch', '#d6d0c6'], ['dome', '#ecebe7']]) M[k].color.set(c);
  function instanced(geo, material, items, place, parent = scene) {
    const im = new THREE.InstancedMesh(geo, material, items.length), o = new THREE.Object3D();
    items.forEach((it, i) => { o.position.set(0, 0, 0); o.rotation.set(0, 0, 0); o.scale.set(1, 1, 1); place(o, it); o.updateMatrix(); im.setMatrixAt(i, o.matrix); });
    parent.add(im); return im;
  }

  // ---------- Kondisi bersama ----------
  const S = { phase: 'choose', gender: 'm', look: '', gq: 'm', block: false, walkT: 0, gesture: 0, gestureKind: 'right' }; // gq: kode karakter untuk ?g= (m, f, mb = pria berjenggot)
  const input = { fwd: false, back: false, auto: false };
  const cam = { top: false, zoom: 1, pos: new THREE.Vector3(70, 50, 70), tgt: new THREE.Vector3(), orbit: 0.6 };
  const hooks = {
    canWalk: () => false, update: () => {}, camera: () => null, begin: () => {}, reset: () => {},
    onBack: null, occluder: null, chibiOpts: () => ({}), lite: () => {}, counter: () => ({ text: 'Belum mulai', panel: '', done: 0, active: false }),
  };
  const sim = { THREE, scene, camera, renderer, stage, S, input, cam, M, mat, canvasTex, instanced, makeChibi, hooks, chibi: null, lite: startLite, fps: null };
  const isM = () => S.gender === 'm';
  sim.isM = isM;

  // ---------- Label & gelembung ----------
  const labelsEl = $('simLabels'); const labels = [];
  sim.addLabel = (text, pos, cls = '') => {
    const el = document.createElement('div'); el.className = 'sim-label ' + cls; el.innerHTML = `<span>${text}</span>`; labelsEl.appendChild(el);
    const l = { el, pos, visible: true }; labels.push(l); return l;
  };
  const bubble = document.createElement('div'); bubble.className = 'sim-bubble'; bubble.style.opacity = 0;
  bubble.innerHTML = '<div class="rounded-xl bg-white shadow-lg px-3 py-1.5 text-center"><div class="ar text-lg leading-tight text-emerald-800" id="bubAr"></div><div class="text-[11px] italic text-gray-600" id="bubLat"></div></div>';
  labelsEl.appendChild(bubble);
  let bubbleUntil = 0;
  // Gelembung di atas karakter: doa (Arab + latin) atau pesan singkat (teks saja).
  let popTimers = [];
  function showBubble(ar, lat, secs, plain) {
    $('bubAr').textContent = ar; $('bubAr').style.display = ar ? '' : 'none';
    const l = $('bubLat'); l.textContent = lat;
    l.className = plain ? 'text-xs font-semibold text-emerald-800 max-w-[220px] whitespace-normal leading-snug' : 'text-[11px] italic text-gray-600';
    bubble.style.opacity = 1; bubbleUntil = performance.now() + secs * 1000;
  }
  sim.say = (ar, lat, secs = 2.6) => { popTimers.forEach(clearTimeout); popTimers = []; showBubble(ar, lat, secs, false); };
  // Beberapa pesan berurutan tanpa menghentikan simulasi.
  sim.pops = (items, gap = 3.2) => {
    popTimers.forEach(clearTimeout); popTimers = [];
    items.forEach((t, i) => popTimers.push(setTimeout(() => showBubble('', t, gap - 0.2, true), i * gap * 1000)));
  };
  const _v = new THREE.Vector3(), ray = new THREE.Raycaster(), _d = new THREE.Vector3();
  function hidden(world) {
    if (!hooks.occluder) return false;
    _d.copy(world).sub(camera.position); const dist = _d.length();
    ray.set(camera.position, _d.normalize()); ray.far = dist - 1.2;
    return ray.intersectObject(hooks.occluder, false).length > 0;
  }
  function placeEl(el, world, w, h, occl = false, show = true) {
    _v.copy(world).project(camera);
    const vis = show && _v.z < 1 && Math.abs(_v.x) < 1.2 && Math.abs(_v.y) < 1.2 && !(occl && hidden(world));
    el.style.display = vis ? '' : 'none';
    if (vis) el.style.transform = `translate(${(_v.x * 0.5 + 0.5) * w}px, ${(-_v.y * 0.5 + 0.5) * h}px) translate(-50%, -100%)`;
  }

  // ---------- Kotak pemandu ----------
  const guide = $('guide');
  // Kartu ringkas untuk Jelajah (open world): panel kecil, huruf lebih kecil, tombol di baris judul.
  function compactHTML(c) {
    let h = `<div class="fade-in rounded-xl border border-gray-200 bg-white px-2.5 py-1.5 lg:p-3">`;
    h += `<div class="flex items-start justify-between gap-2"><h2 class="font-bold text-gray-900 text-[13px] lg:text-[15px] leading-snug">${c.title}</h2>${c.button || ''}</div>`;
    if (c.body) h += `<div class="mt-0.5 text-[12px] lg:text-[13px] text-gray-600 leading-snug space-y-1">${c.body}</div>`;
    if (c.dua) h += `<div class="mt-1 rounded-lg bg-emerald-50 px-2.5 py-1"><p class="ar text-[17px] lg:text-lg leading-relaxed text-emerald-900 text-right" dir="rtl">${c.dua.ar}</p>`
      + `<p class="text-[11.5px] italic text-emerald-800 leading-snug">${c.dua.lat}</p><p class="text-[10.5px] text-gray-500 leading-snug">${c.dua.arti}</p></div>`;
    if (c.after) h += `<div class="mt-1 text-[12px] lg:text-[13px] text-gray-600 leading-snug">${c.after}</div>`;
    if (c.hint) h += `<p class="hidden lg:block mt-1.5 text-[11px] text-emerald-700 font-medium">${c.hint}</p>`;
    return h + '</div>';
  }
  function cardHTML(c) {
    if (c.compact) return compactHTML(c);
    let h = `<div class="${c.pause ? 'card-in border-amber-200 bg-amber-50 border-l-4 border-l-amber-400 lg:border-l' : 'fade-in border-gray-200 bg-white'} rounded-2xl border p-3 lg:p-4">`;
    if (c.pause) h += `<p class="hidden lg:flex items-center gap-1.5 text-[10px] lg:text-[11px] font-bold uppercase tracking-wide text-amber-700 mb-0.5 lg:mb-2"><svg viewBox="0 0 24 24" class="w-3 h-3 lg:w-3.5 lg:h-3.5" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><circle cx="12" cy="12" r="9"/><path d="M12 11v5M12 8h.01"/></svg>Sekilas Info</p>`;
    h += `<h2 class="font-bold text-gray-900 text-[14px] lg:text-lg leading-snug">${c.title}</h2>`;
    if (c.body) h += `<div class="mt-1 lg:mt-2 text-[12.5px] lg:text-sm text-gray-600 leading-snug lg:leading-relaxed space-y-1 lg:space-y-2">${c.body}</div>`;
    if (c.dua) {
      // Doa pendek di HP: Arab & latin satu baris supaya kotak tetap ringkas tanpa terpotong.
      const short = c.dua.ar.length < 40;
      h += `<div class="mt-1.5 lg:mt-3 rounded-xl bg-emerald-50 px-3 lg:px-4 py-1.5 lg:py-3">`
        + (short ? `<div class="flex flex-row-reverse items-baseline justify-between gap-2 lg:block">` : '')
        + `<p class="ar ${c.dua.ar.length > 100 ? 'text-base lg:text-xl' : 'text-lg lg:text-2xl'} leading-relaxed lg:leading-loose text-emerald-900 text-right" dir="rtl">${c.dua.ar}</p>`
        + `<p class="text-[12.5px] lg:text-sm italic text-emerald-800 lg:mt-1">${c.dua.lat}</p>` + (short ? '</div>' : '')
        + `<p class="text-[11px] lg:text-xs text-gray-500 lg:mt-1 leading-snug">${c.dua.arti}</p></div>`;
    }
    if (c.extra) h += c.extra;
    if (c.after) h += `<div class="mt-2 lg:mt-3 text-[12.5px] lg:text-sm text-gray-600 leading-snug lg:leading-relaxed space-y-1 lg:space-y-2">${c.after}</div>`;
    if (c.hint) h += `<p class="mt-2 lg:mt-3 text-[11px] lg:text-xs text-emerald-700 font-medium">${c.hint}</p>`;
    // Tombol aksi menempel di bawah area panel, jadi selalu terlihat walau teksnya panjang.
    if (c.actions) h += `<div class="sticky bottom-0 -mx-3 -mb-3 lg:-mx-4 lg:-mb-4 mt-1.5 lg:mt-3 px-3 lg:px-4 pt-1.5 pb-3 lg:pb-4 rounded-b-2xl bg-amber-50 flex flex-wrap gap-2">${c.actions}</div>`;
    return h + '</div>';
  }
  let steps = [], stepDone = null, lastInfo = '';
  sim.pauseWith = (list, onDone) => {
    S.block = true; steps = list.slice(); stepDone = onDone; input.fwd = false; tapWalk = false; renderStep(); duck(true);
  };
  function renderStep() {
    const c = steps[0], last = steps.length === 1; lastInfo = '';
    guide.innerHTML = cardHTML({ ...c, pause: true, actions: (c.actions || '') + `<button id="btnNext" type="button" class="rounded-xl bg-emerald-700 hover:bg-emerald-800 text-white font-semibold px-4 lg:px-5 py-1.5 lg:py-2.5 text-[13px] lg:text-sm">${c.next || (last ? 'Lanjut jalan →' : 'Lanjut →')}</button>` });
    if (c.onShow) c.onShow();
    $('guideScroll').scrollTop = 0;
    $('btnNext').onclick = () => {
      steps.shift();
      if (steps.length) { renderStep(); return; }
      S.block = false; duck(false); const f = stepDone; stepDone = null; f && f();
    };
  }
  sim.info = (c) => { const h = cardHTML(c); if (h === lastInfo) return; lastInfo = h; guide.innerHTML = h; $('guideScroll').scrollTop = 0; };
  let toastT = 0;
  sim.toast = (msg) => { const t = $('simToast'); t.textContent = msg; t.classList.remove('hidden'); clearTimeout(toastT); toastT = setTimeout(() => t.classList.add('hidden'), 2600); };
  function showPending() {
    const b = $('btnNext'); if (!b) return;
    sim.toast(window.innerWidth < 1024 ? 'Baca penjelasan di bawah, lalu tekan Lanjut.' : 'Baca penjelasan di panel kanan, lalu tekan Lanjut.');
    b.classList.remove('nudge'); void b.offsetWidth; b.classList.add('nudge');
  }
  sim.setCounter = () => {
    const c = hooks.counter();
    $('simRound').textContent = c.text;
    $('panelRound').textContent = c.panel || '';
    $('simDots').innerHTML = Array.from({ length: 7 }, (_, i) => `<i class="block w-2 h-2 rounded-full ${i < c.done ? 'bg-emerald-600' : (c.active && i === c.done ? 'bg-emerald-300 animate-pulse' : 'bg-gray-200')}"></i>`).join('');
  };

  // ---------- Audio murottal (pelan; posisi diputar dilanjutkan antarhalaman) ----------
  const audio = $('simAudio'); let audioOn = sessionStorage.getItem('ukAudioOff') !== '1';
  const VOL = 0.3, VOL_DUCK = 0.12;
  audio.volume = VOL;
  function duck(on) { audio.volume = on ? VOL_DUCK : VOL; }
  sim.duck = duck;
  // Lanjutkan posisi hanya bila file audionya sama (Tawaf: Al-Baqarah 195–205, Sa'i: Al-Baqarah 157–170).
  audio.addEventListener('loadedmetadata', () => {
    const t = +sessionStorage.getItem('ukAudioT') || 0;
    if (sessionStorage.getItem('ukAudioSrc') === audio.currentSrc && t > 0 && t < audio.duration) audio.currentTime = t;
    sessionStorage.removeItem('ukAudioT');
  }, { once: true });
  addEventListener('pagehide', () => { if (!audio.paused) { sessionStorage.setItem('ukAudioT', audio.currentTime); sessionStorage.setItem('ukAudioSrc', audio.currentSrc); } });
  sim.playAudio = () => { if (audioOn) audio.play().catch(() => {}); };
  function syncAudioBtn() {
    $('audioOn').style.display = audioOn ? '' : 'none'; $('audioOff').style.display = audioOn ? 'none' : '';
    const b = $('btnAudio'); b.setAttribute('aria-pressed', audioOn); b.setAttribute('aria-label', audioOn ? 'Matikan murottal' : 'Nyalakan murottal');
    b.classList.toggle('text-emerald-700', audioOn); b.classList.toggle('text-gray-400', !audioOn);
  }
  $('btnAudio').onclick = () => {
    audioOn = !audioOn; sessionStorage.setItem('ukAudioOff', audioOn ? '0' : '1'); syncAudioBtn();
    if (!audioOn) audio.pause(); else if (S.phase !== 'choose') sim.playAudio();
  };
  syncAudioBtn();

  // ---------- Mulai & ulang ----------
  sim.begin = (gender, look = '') => {
    S.gender = gender; S.look = look || ''; S.gq = gender + (look === 'beard' ? 'b' : '');
    if (sim.chibi) scene.remove(sim.chibi.g);
    sim.chibi = makeChibi(gender, { ...hooks.chibiOpts(gender), look: S.look || null }); scene.add(sim.chibi.g);
    Object.assign(S, { block: false, gesture: 0, walkT: 0 });
    $('chooseOverlay').classList.add('hidden'); $('simControls').classList.remove('hidden');
    cam.top = false; syncCamBtn();
    sim.playAudio();
    hooks.begin(gender);
    sim.setCounter();
  };
  sim.reset = () => {
    S.phase = 'choose'; S.block = false; steps = []; popTimers.forEach(clearTimeout); popTimers = []; input.fwd = input.back = false; tapWalk = false;
    if (sim.chibi) { scene.remove(sim.chibi.g); sim.chibi = null; }
    audio.pause(); duck(false);
    $('chooseOverlay').classList.remove('hidden'); $('simControls').classList.add('hidden');
    if (window.ukTab) window.ukTab(opts.part || 'tawaf'); // kembali ke tab pilih karakter
    hooks.reset();
    sim.setCounter();
  };
  $('btnReset').onclick = () => sim.reset();
  document.querySelectorAll('.pick').forEach((b) => (b.onclick = () => sim.begin(b.dataset.gender, b.dataset.look || '')));
  // Di HP tombol kamera hanya ikon: saat mode atas aktif, tombolnya hijau.
  function syncCamBtn() {
    const b = $('btnCam'), t = cam.top ? 'Ikuti karakter' : 'Lihat dari atas';
    $('camLabel').textContent = t; b.title = t; b.setAttribute('aria-label', t);
    b.classList.toggle('!bg-emerald-700', cam.top); b.classList.toggle('!text-white', cam.top);
  }
  $('btnCam').onclick = () => { cam.top = !cam.top; syncCamBtn(); };
  $('chkAuto').onchange = (e) => (input.auto = e.target.checked);

  // ---------- Tombol Jalan: klik = jalan terus (klik lagi untuk berhenti); tahan = selama ditahan ----------
  const fwdBtn = $('btnFwd');
  let tapWalk = false, downAt = 0, wasTapWalk = false;
  fwdBtn.addEventListener('pointerdown', (e) => {
    e.preventDefault();
    if (S.block) { showPending(); return; }
    wasTapWalk = tapWalk; tapWalk = false; downAt = performance.now();
    input.fwd = !wasTapWalk; fwdBtn.classList.add('on');
  });
  const fwdUp = () => {
    if (!downAt) return;
    const tap = performance.now() - downAt < 280; downAt = 0; fwdBtn.classList.remove('on');
    if (tap && !wasTapWalk) { tapWalk = true; input.fwd = true; } else { input.fwd = false; }
  };
  ['pointerup', 'pointerleave', 'pointercancel'].forEach((ev) => fwdBtn.addEventListener(ev, fwdUp));
  fwdBtn.addEventListener('contextmenu', (e) => e.preventDefault());
  let fwdState = '';
  function syncFwdBtn() {
    const canWalk = !!sim.chibi && !S.block && hooks.canWalk();
    const walking = canWalk && (input.fwd || input.auto);
    const holding = walking && downAt && input.fwd;
    const st = holding ? 'hold' : walking ? 'walk' : canWalk ? 'idle' : 'off';
    if (st === fwdState) return; fwdState = st;
    fwdBtn.classList.toggle('wait-tap', st === 'idle');
    fwdBtn.classList.toggle('opacity-60', st === 'off');
    $('fwdIcon').innerHTML = walking ? '<rect x="6" y="5" width="4" height="14" rx="1"/><rect x="14" y="5" width="4" height="14" rx="1"/>' : '<path d="M9 5l8 7-8 7z"/>';
    const [top, main] = st === 'hold' ? ['Lepas untuk', 'Berhenti'] : st === 'walk' ? ['Klik untuk', 'Berhenti'] : ['Klik untuk', 'Jalan'];
    $('fwdLabel').innerHTML = `<span class="text-[10px] font-medium text-emerald-100 mb-0.5">${top}</span><span class="text-base">${main}</span>`;
  }
  const FWD = ['ArrowUp', 'ArrowRight', 'KeyW', 'KeyD'], BACK = ['ArrowDown', 'ArrowLeft', 'KeyS', 'KeyA'];
  addEventListener('keydown', (e) => { if (FWD.includes(e.code)) { if (!input.fwd && S.block) showPending(); input.fwd = true; e.preventDefault(); } if (BACK.includes(e.code)) { input.back = true; e.preventDefault(); } });
  addEventListener('keyup', (e) => { if (FWD.includes(e.code)) input.fwd = false; if (BACK.includes(e.code)) input.back = false; });
  addEventListener('blur', () => { input.fwd = input.back = false; });
  let pinch = 0;
  function bindCanvas(cv) { // dipasang ulang bila canvas diganti (mode ringan)
    cv.addEventListener('wheel', (e) => { e.preventDefault(); cam.zoom = THREE.MathUtils.clamp(cam.zoom * (1 + e.deltaY * 0.001), 0.55, 1.9); }, { passive: false });
    cv.addEventListener('touchmove', (e) => {
      if (e.touches.length !== 2) return;
      const d = Math.hypot(e.touches[0].clientX - e.touches[1].clientX, e.touches[0].clientY - e.touches[1].clientY);
      if (pinch) cam.zoom = THREE.MathUtils.clamp(cam.zoom * (pinch / d), 0.55, 1.9);
      pinch = d; e.preventDefault();
    }, { passive: false });
    cv.addEventListener('touchend', () => (pinch = 0));
  }
  bindCanvas(canvas);
  let backWarned = 0;
  sim.walking = () => !!sim.chibi && !S.block && hooks.canWalk() && (input.fwd || input.auto);
  sim.backPressed = () => {
    if (!hooks.onBack || !input.back || S.block || !hooks.canWalk() || performance.now() - backWarned < 2500) return;
    backWarned = performance.now(); hooks.onBack();
  };

  // ---------- Animasi karakter ----------
  // gestureKind: 'right' = isyarat tangan kanan (Hajar Aswad), 'doa' = kedua tangan terangkat (Shafa/Marwah)
  // Simulasi dipercepat (±6 m/detik), jadi irama langkah diatur tersendiri supaya tetap wajar.
  sim.animate = (dt, speed, { run = false } = {}) => {
    const c = sim.chibi, ws = c.g.scale.x * RIG_SCALE, moving = speed > 0;
    const gesture = S.gesture > 0 ? (S.gestureKind === 'doa' || S.gestureKind === 'cup' ? S.gestureKind : 'right') : null;
    if (S.gesture > 0) S.gesture -= dt;
    const turn = c.lastYaw == null ? 0 : (mod(c.g.rotation.y - c.lastYaw + Math.PI, TAU) - Math.PI) / Math.max(dt, 1e-3);
    c.lastYaw = c.g.rotation.y;
    c.rig.animate({ dt, speed: moving ? speed / ws : 0, cadence: moving ? (run ? 2.3 : 1.55) : 0, run: run && moving, gesture, turn: -turn });
  };
  sim.face = (dir, dt) => {
    const g = sim.chibi.g; const dr = mod(dir - g.rotation.y + Math.PI, TAU) - Math.PI;
    g.rotation.y += dr * Math.min(1, dt * 10);
  };

  // ---------- Loop ----------
  function resize() {
    const w = stage.clientWidth, h = stage.clientHeight;
    renderer.setSize(w, h, false); camera.aspect = w / h; camera.updateProjectionMatrix();
  }
  new ResizeObserver(resize).observe(stage); resize();
  function updateCamera(dt) {
    const want = hooks.camera(dt); if (!want) return;
    const k = 1 - Math.exp(-dt * (opts.camLerp || 2.8));
    cam.pos.lerp(want.pos, k); cam.tgt.lerp(want.tgt, k);
    camera.position.copy(cam.pos); camera.lookAt(cam.tgt);
  }
  const clock = new THREE.Clock();
  function step(dt) { hooks.update(dt); sim.backPressed(); updateCamera(dt); }
  // Antialias hanya bisa diatur saat renderer dibuat, jadi mode ringan memakai canvas & renderer baru
  // (geometri/tekstur diunggah ulang otomatis oleh three.js).
  sim.setLite = () => {
    if (sim.lite) return;
    sim.lite = true; try { sessionStorage.setItem(LITE_KEY, '1'); } catch (e) {}
    const next = canvas.cloneNode(false); canvas.replaceWith(next);
    renderer.dispose(); renderer.forceContextLoss();
    canvas = next; renderer = makeRenderer(canvas, true); sim.renderer = renderer;
    bindCanvas(canvas); resize(); hooks.lite();
  };
  // Ukur FPS selama 2,5 detik (setelah 0,8 detik pemanasan). Di bawah 40 fps -> mode ringan.
  const probe = { from: 0, n: 0, done: startLite };
  function measure(now) {
    if (probe.done) return;
    if (!probe.from) { probe.from = now + 800; return; }
    if (now < probe.from) return;
    probe.n++;
    if (now - probe.from < 2500) return;
    probe.done = true; sim.fps = Math.round(probe.n * 1000 / (now - probe.from));
    if (sim.fps < 40) sim.setLite();
  }
  function frame() {
    requestAnimationFrame(frame);
    if (document.hidden) { probe.from = 0; probe.n = 0; return; } // tab tersembunyi: ukur ulang dari awal
    const dt = Math.min(clock.getDelta(), 0.05);
    step(dt); syncFwdBtn();
    renderer.render(scene, camera);
    measure(performance.now());
    const w = stage.clientWidth, h = stage.clientHeight;
    labels.forEach((l) => placeEl(l.el, l.pos, w, h, true, l.visible));
    if (sim.chibi) { placeEl(bubble, sim.chibi.g.position.clone().setY(2.7 * sim.chibi.g.scale.y), w, h); if (performance.now() > bubbleUntil) bubble.style.opacity = 0; }
    else bubble.style.opacity = 0;
  }
  // ---------- Datang dari Jelajah (open world): tombol kembali ----------
  // ?from=haram dari Jelajah -> tanda di sessionStorage (bertahan Tawaf -> Sa'i) sampai kembali ke Jelajah.
  const ss = (fn) => { try { return fn(); } catch (e) { return null; } };
  sim.backToHaram = (done) => {
    const g = sim.chibi ? '&g=' + S.gq : '';
    const l = $('simLoading'); if (l) { $('simLoadingText').textContent = 'Memuat Jelajah Masjidil Haram…'; l.classList.remove('hidden'); l.classList.add('flex'); }
    location.href = `/simulasi?bagian=haram${g}&back=${opts.part}${done ? '-done' : ''}`;
  };
  function addBackButton() {
    // baris kedua kolom kiri atas (di HP baris pertama sudah penuh: Umrohku + status)
    const row = $('simRound') && $('simRound').parentElement && $('simRound').parentElement.parentElement && $('simRound').parentElement.parentElement.parentElement; if (!row) return;
    const b = document.createElement('button'); b.type = 'button'; b.id = 'btnBackHaram';
    b.className = 'pointer-events-auto inline-flex items-center gap-1 rounded-full bg-white/90 shadow px-3 py-1.5 text-xs font-bold text-emerald-700';
    b.innerHTML = '<svg viewBox="0 0 24 24" class="w-3.5 h-3.5" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round"><path d="M15 18l-6-6 6-6"/></svg>Jelajah';
    b.setAttribute('aria-label', 'Kembali ke Jelajah Masjidil Haram'); b.onclick = () => sim.backToHaram(false);
    row.appendChild(b);
  }
  sim.start = (initialCam) => {
    if (initialCam) { cam.pos.copy(initialCam.pos); cam.tgt.copy(initialCam.tgt); }
    sim.query = new URLSearchParams(location.search); // dibaca sebelum URL dirapikan di bawah
    if (opts.part === 'haram') ss(() => sessionStorage.removeItem('ukFromHaram'));
    else if (sim.query.get('from') === 'haram') ss(() => sessionStorage.setItem('ukFromHaram', '1'));
    sim.fromHaram = opts.part !== 'haram' && ss(() => sessionStorage.getItem('ukFromHaram')) === '1';
    if (sim.fromHaram) addBackButton();
    // Kait pengujian (dipakai skrip uji otomatis; tidak memengaruhi pengguna).
    window.__sim = { S, input, sim, tick(sec) { for (let t = 0; t < sec; t += 0.05) step(0.05); syncFwdBtn(); } };
    if (sim.lite) hooks.lite();
    sim.reset();
    // Dari tombol "Lanjut ke Sa'i": karakter yang sama langsung dimulai.
    const g = sim.query.get('g');
    if (/^[mf]b?$/.test(g || '')) { history.replaceState(null, '', location.pathname + '?bagian=' + (opts.part || 'tawaf')); sim.begin(g[0], g[1] === 'b' ? 'beard' : ''); }
    frame();
  };
  return sim;
}
