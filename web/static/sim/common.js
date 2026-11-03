// Bagian bersama simulasi Umrohku (tawaf & sa'i): renderer, karakter chibi, kotak pemandu,
// tombol Jalan, audio murottal, label, dan loop. Tiap adegan mengisi `sim.hooks`.
import * as THREE from 'three';
export { THREE };

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
  // ---------- Karakter chibi (menghadap +z; sisi kanannya ada di -x) ----------
  function limb(r, len, material) {
    const pivot = new THREE.Group(); const m = new THREE.Mesh(new THREE.CapsuleGeometry(r, len, 3, 8), material); m.position.y = -len / 2 - r * 0.4; pivot.add(m); return { pivot, m };
  }
  // clothColor/veilColor: untuk ilustrasi (mis. tukang cukur berbaju biasa).
  function makeChibi(gender, { shoulderBare = gender === 'm', scale = 2, clothColor = null, veilColor = null, draped = false } = {}) {
    const g = new THREE.Group(), parts = {};
    const cloth = clothColor ? mat(clothColor) : gender === 'm' ? M.white : mat('#f4efe6'), veil = mat(veilColor || '#e6ddcf');
    const ihram = gender === 'm' && !clothColor;
    if (gender === 'm') {
      const legL = limb(0.11, 0.38, M.skin), legR = limb(0.11, 0.38, M.skin);
      legL.pivot.position.set(-0.15, 0.62, 0); legR.pivot.position.set(0.15, 0.62, 0); g.add(legL.pivot, legR.pivot);
      parts.legL = legL.pivot; parts.legR = legR.pivot;
      const izar = new THREE.Mesh(new THREE.CylinderGeometry(0.34, 0.42, 0.62, 14), cloth); izar.position.y = 0.72; g.add(izar);
      if (ihram) { // izar: gulungan di pinggang + tumpukan kain di depan
        const hem = mat('#e4ded0');
        const roll = new THREE.Mesh(new THREE.TorusGeometry(0.345, 0.035, 6, 20), hem); roll.rotation.x = Math.PI / 2; roll.position.y = 1.02; g.add(roll);
        const fold = new THREE.Mesh(new THREE.BoxGeometry(0.035, 0.6, 0.02), hem); fold.position.set(0.12, 0.72, 0.39); fold.rotation.x = -0.12; g.add(fold);
      }
      if (!ihram || !draped) {
        // Kain atas sederhana (dipakai di simulasi); saat tawaf bahu kanan terbuka.
        const torso = new THREE.Mesh(new THREE.CylinderGeometry(0.3, 0.35, 0.55, 14), cloth); torso.position.y = 1.25; g.add(torso);
        if (ihram && shoulderBare) { const sh = new THREE.Mesh(new THREE.SphereGeometry(0.16, 10, 8), M.skin); sh.position.set(-0.27, 1.43, 0.02); g.add(sh); }
      } else {
        // Kain atas berlipat (dipakai di ilustrasi Niat/Tahallul): rida menutup kedua bahu,
        // ujungnya dilempar ke bahu kiri -> selempang kain lebar melintang dada.
        const rida = mat('#f5f2ea'), band = mat('#ece7db');
        const torso = new THREE.Mesh(new THREE.CylinderGeometry(0.33, 0.355, 0.55, 14), rida); torso.position.y = 1.25; g.add(torso);
        for (const sx of [-1, 1]) { const cap = new THREE.Mesh(new THREE.SphereGeometry(0.19, 12, 8), rida); cap.scale.set(1.15, 0.75, 1); cap.position.set(sx * 0.28, 1.47, 0); g.add(cap); }
        // selempang: torus yang dipipihkan jadi pita lebar mengikuti badan (bukan tali bulat)
        const sash = new THREE.Mesh(new THREE.TorusGeometry(0.38, 0.055, 8, 32), band); sash.rotation.set(Math.PI / 2, 0.62, 0); sash.position.y = 1.29; sash.scale.set(1, 0.86, 2.4); g.add(sash);
      }
    } else {
      const dress = new THREE.Mesh(new THREE.CylinderGeometry(0.3, 0.52, 1.35, 16), cloth); dress.position.y = 0.72; g.add(dress);
      const cape = new THREE.Mesh(new THREE.CylinderGeometry(0.26, 0.5, 0.55, 16), veil); cape.position.y = 1.42; g.add(cape);
    }
    for (const side of [-1, 1]) {
      // Ilustrasi: lengan terbuka (ihram tak berlengan). Simulasi: seperti semula, lengan kanan terbuka saat idhtiba'.
      const arm = limb(0.085, 0.36, ihram && (draped || (shoulderBare && side === -1)) ? M.skin : cloth);
      arm.pivot.position.set(side * 0.38, 1.45, 0); g.add(arm.pivot);
      const hand = new THREE.Mesh(new THREE.SphereGeometry(0.09, 8, 6), M.skin); hand.position.y = -0.62; arm.pivot.add(hand);
      parts[side < 0 ? 'armR' : 'armL'] = arm.pivot;
    }
    for (const side of [-1, 1]) {
      const foot = new THREE.Mesh(new THREE.BoxGeometry(0.17, 0.08, 0.3), mat('#7a5a3c')); foot.position.set(side * 0.15, 0.04, 0.05); g.add(foot);
    }
    const head = new THREE.Group(); head.position.y = 2.0; g.add(head); parts.head = head;
    head.add(new THREE.Mesh(new THREE.SphereGeometry(0.44, 20, 16), M.skin));
    if (gender === 'm') {
      const hair = new THREE.Mesh(new THREE.SphereGeometry(0.455, 20, 12, 0, TAU, 0, Math.PI * 0.42), M.hair); hair.rotation.x = -0.35; head.add(hair);
    } else {
      const hijab = new THREE.Mesh(new THREE.SphereGeometry(0.5, 20, 16), veil); hijab.position.z = -0.1; hijab.scale.set(1, 1.04, 1); head.add(hijab);
      const rim = new THREE.Mesh(new THREE.TorusGeometry(0.4, 0.07, 8, 24), veil); rim.position.z = 0.17; head.add(rim);
    }
    for (const side of [-1, 1]) {
      const eye = new THREE.Mesh(new THREE.SphereGeometry(0.055, 8, 6), M.black); eye.position.set(side * 0.15, 0.02, 0.4); eye.scale.set(1, 1.25, 0.6); head.add(eye);
      const blush = new THREE.Mesh(new THREE.SphereGeometry(0.06, 8, 6), mat('#f59e9e')); blush.position.set(side * 0.24, -0.1, 0.36); blush.scale.set(1.3, 0.6, 0.4); head.add(blush);
    }
    const sh = new THREE.Mesh(new THREE.CircleGeometry(0.75, 20), new THREE.MeshBasicMaterial({ map: canvasTex(64, 64, (c, s) => { const gr = c.createRadialGradient(s / 2, s / 2, 0, s / 2, s / 2, s / 2); gr.addColorStop(0, 'rgba(0,0,0,.35)'); gr.addColorStop(1, 'rgba(0,0,0,0)'); c.fillStyle = gr; c.fillRect(0, 0, s, s); }), transparent: true, depthWrite: false }));
    sh.rotation.x = -Math.PI / 2; sh.position.y = 0.02; g.add(sh);
    g.scale.setScalar(scale);
    return { g, parts };
  }
  return { M, mat, canvasTex, makeChibi };
}

export function createSim(opts = {}) {
  const bg = opts.bg || '#e7ede6';
  const stage = $('simStage'), canvas = $('simCanvas');
  const renderer = new THREE.WebGLRenderer({ canvas, antialias: true, powerPreference: 'low-power' });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 1.5));
  renderer.outputColorSpace = THREE.SRGBColorSpace;
  const scene = new THREE.Scene();
  scene.background = new THREE.Color(bg);
  scene.fog = new THREE.Fog(bg, opts.fogNear || 130, opts.fogFar || 260);
  const camera = new THREE.PerspectiveCamera(opts.fov || 42, 1, 0.5, opts.far || 420);
  scene.add(new THREE.HemisphereLight('#ffffff', '#cbbf9f', 1.25));
  const sun = new THREE.DirectionalLight('#fff4df', 1.5);
  sun.position.set(-40, 80, 30);
  scene.add(sun);

  const { M, mat, canvasTex, makeChibi } = chibiKit();
  function instanced(geo, material, items, place, parent = scene) {
    const im = new THREE.InstancedMesh(geo, material, items.length), o = new THREE.Object3D();
    items.forEach((it, i) => { o.position.set(0, 0, 0); o.rotation.set(0, 0, 0); o.scale.set(1, 1, 1); place(o, it); o.updateMatrix(); im.setMatrixAt(i, o.matrix); });
    parent.add(im); return im;
  }

  // ---------- Kondisi bersama ----------
  const S = { phase: 'choose', gender: 'm', block: false, walkT: 0, gesture: 0, gestureKind: 'right' };
  const input = { fwd: false, back: false, auto: false };
  const cam = { top: false, zoom: 1, pos: new THREE.Vector3(70, 50, 70), tgt: new THREE.Vector3(), orbit: 0.6 };
  const hooks = {
    canWalk: () => false, update: () => {}, camera: () => null, begin: () => {}, reset: () => {},
    onBack: null, occluder: null, chibiOpts: () => ({}), counter: () => ({ text: 'Belum mulai', panel: '', done: 0, active: false }),
  };
  const sim = { THREE, scene, camera, renderer, stage, S, input, cam, M, mat, canvasTex, instanced, makeChibi, hooks, chibi: null };
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
  function cardHTML(c) {
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
  sim.begin = (gender) => {
    S.gender = gender;
    if (sim.chibi) scene.remove(sim.chibi.g);
    sim.chibi = makeChibi(gender, hooks.chibiOpts(gender)); scene.add(sim.chibi.g);
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
  document.querySelectorAll('.pick').forEach((b) => (b.onclick = () => sim.begin(b.dataset.gender)));
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
  canvas.addEventListener('wheel', (e) => { e.preventDefault(); cam.zoom = THREE.MathUtils.clamp(cam.zoom * (1 + e.deltaY * 0.001), 0.55, 1.9); }, { passive: false });
  let pinch = 0;
  canvas.addEventListener('touchmove', (e) => {
    if (e.touches.length !== 2) return;
    const d = Math.hypot(e.touches[0].clientX - e.touches[1].clientX, e.touches[0].clientY - e.touches[1].clientY);
    if (pinch) cam.zoom = THREE.MathUtils.clamp(cam.zoom * (pinch / d), 0.55, 1.9);
    pinch = d; e.preventDefault();
  }, { passive: false });
  canvas.addEventListener('touchend', () => (pinch = 0));
  let backWarned = 0;
  sim.walking = () => !!sim.chibi && !S.block && hooks.canWalk() && (input.fwd || input.auto);
  sim.backPressed = () => {
    if (!hooks.onBack || !input.back || S.block || !hooks.canWalk() || performance.now() - backWarned < 2500) return;
    backWarned = performance.now(); hooks.onBack();
  };

  // ---------- Animasi karakter ----------
  // gestureKind: 'right' = isyarat tangan kanan (Hajar Aswad), 'doa' = kedua tangan terangkat (Shafa/Marwah)
  sim.animate = (dt, speed, { run = false } = {}) => {
    const p = sim.chibi.parts, moving = speed > 0;
    S.walkT += dt * (moving ? speed * (run ? 2.1 : 1.6) : 0);
    const amp = run ? 1.05 : 0.7, sw = moving ? Math.sin(S.walkT) : 0;
    const ease = (o, v) => (o.rotation.x += (v - o.rotation.x) * Math.min(1, dt * 12));
    if (p.legL) { ease(p.legL, sw * amp); ease(p.legR, -sw * amp); }
    if (S.gesture > 0) {
      S.gesture -= dt;
      if (S.gestureKind === 'doa') { ease(p.armR, -1.75); ease(p.armL, -1.75); }
      else { p.armR.rotation.x += (-2.5 - p.armR.rotation.x) * Math.min(1, dt * 10); ease(p.armL, -sw * 0.6); }
    } else { ease(p.armL, -sw * (run ? 0.9 : 0.6)); ease(p.armR, sw * (run ? 0.9 : 0.6)); }
    p.head.position.y = 2.0 + (moving ? Math.abs(Math.sin(S.walkT)) * (run ? 0.1 : 0.06) : 0);
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
    const k = 1 - Math.exp(-dt * 2.8);
    cam.pos.lerp(want.pos, k); cam.tgt.lerp(want.tgt, k);
    camera.position.copy(cam.pos); camera.lookAt(cam.tgt);
  }
  const clock = new THREE.Clock();
  function step(dt) { hooks.update(dt); sim.backPressed(); updateCamera(dt); }
  function frame() {
    requestAnimationFrame(frame);
    if (document.hidden) return;
    const dt = Math.min(clock.getDelta(), 0.05);
    step(dt); syncFwdBtn();
    renderer.render(scene, camera);
    const w = stage.clientWidth, h = stage.clientHeight;
    labels.forEach((l) => placeEl(l.el, l.pos, w, h, true, l.visible));
    if (sim.chibi) { placeEl(bubble, sim.chibi.g.position.clone().setY(2.7 * sim.chibi.g.scale.y), w, h); if (performance.now() > bubbleUntil) bubble.style.opacity = 0; }
    else bubble.style.opacity = 0;
  }
  sim.start = (initialCam) => {
    if (initialCam) { cam.pos.copy(initialCam.pos); cam.tgt.copy(initialCam.tgt); }
    // Kait pengujian (dipakai skrip uji otomatis; tidak memengaruhi pengguna).
    window.__sim = { S, input, sim, tick(sec) { for (let t = 0; t < sec; t += 0.05) step(0.05); syncFwdBtn(); } };
    sim.reset();
    // Dari tombol "Lanjut ke Sa'i": karakter yang sama langsung dimulai.
    const g = new URLSearchParams(location.search).get('g');
    if (g === 'm' || g === 'f') { history.replaceState(null, '', location.pathname + '?bagian=' + (opts.part || 'tawaf')); sim.begin(g); }
    frame();
  };
  return sim;
}
