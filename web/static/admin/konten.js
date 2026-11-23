// Generator gambar postingan Umrohku (admin, tab Konten). Menggambar di <canvas> 1080x1350
// dari data crawl terbaru (#kData) lalu bisa diunduh sebagai PNG + caption siap salin.
(function () {
    var D = JSON.parse(document.getElementById('kData').textContent || '{}');
    var cv = document.getElementById('kCanvas'), g = cv.getContext('2d');
    var W = 1080, H = 1350, P = 64;
    var C = { bg: '#F7F4EC', em: '#065F46', em2: '#047857', gold: '#E4C35A', ink: '#1F2937', mute: '#6B7280', line: '#E5E7EB', rose: '#E11D48', card: '#FFFFFF' };
    var F = 'Inter, "Helvetica Neue", Arial, sans-serif';
    var BLN = ['', 'Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
    var state = { t: 'termurah' };

    // ---------- util ----------
    function rp(n) { return 'Rp' + String(n).replace(/\B(?=(\d{3})+(?!\d))/g, '.'); }
    function jt(n) { var t = Math.floor(n / 100000), w = Math.floor(t / 10), f = t % 10; return w + (f ? ',' + f : '') + ' jt'; }
    function tgl(s) { var p = s.split('-'); return +p[2] + ' ' + BLN[+p[1]] + ' ' + p[0]; }
    function tglPendek(s) { var p = s.split('-'); return +p[2] + ' ' + BLN[+p[1]]; }
    function font(w, s) { g.font = w + ' ' + s + 'px ' + F; }
    function rr(x, y, w, h, r, fill) { g.beginPath(); g.roundRect(x, y, w, h, r); g.fillStyle = fill; g.fill(); }
    function fit(text, maxW) {
        if (g.measureText(text).width <= maxW) return text;
        while (text.length > 1 && g.measureText(text + '…').width > maxW) text = text.slice(0, -1);
        return text.trim() + '…';
    }
    function wrap(text, maxW, maxLines) {
        var words = String(text).split(/\s+/), lines = [], cur = '';
        words.forEach(function (w) {
            var t = cur ? cur + ' ' + w : w;
            if (g.measureText(t).width > maxW && cur) { lines.push(cur); cur = w; } else cur = t;
        });
        if (cur) lines.push(cur);
        if (lines.length > maxLines) { lines = lines.slice(0, maxLines); lines[maxLines - 1] = fit(lines[maxLines - 1] + '…', maxW); }
        return lines;
    }
    var imgs = {};
    function img(src) {
        if (imgs[src]) return imgs[src];
        var im = new Image(); im.onload = function () { im.ok = true; draw(); }; im.onerror = function () { im.bad = true; };
        im.src = src; imgs[src] = im; return im;
    }
    function logo(d, x, y, s) {
        var im = img('/static/logos/' + d.slug + '.png');
        rr(x, y, s, s, 18, '#fff'); g.strokeStyle = C.line; g.lineWidth = 2; g.beginPath(); g.roundRect(x, y, s, s, 18); g.stroke();
        if (im.ok) {
            var r = Math.min((s - 16) / im.width, (s - 16) / im.height), w = im.width * r, h = im.height * r;
            g.drawImage(im, x + (s - w) / 2, y + (s - h) / 2, w, h);
        } else {
            font('800', 30); g.fillStyle = C.em; g.textAlign = 'center'; g.fillText(d.initials || '?', x + s / 2, y + s / 2 + 11); g.textAlign = 'left';
        }
    }

    // ---------- kerangka ----------
    function kaaba(x, y, s) {
        rr(x, y, s, s, s * .18, '#111827');
        g.fillStyle = C.gold; g.fillRect(x, y + s * .26, s, s * .13);
        g.fillStyle = 'rgba(228,195,90,.55)'; g.fillRect(x + s * .62, y + s * .5, s * .2, s * .38);
    }
    function frame(kicker, title, sub, noDate) {
        g.fillStyle = C.bg; g.fillRect(0, 0, W, H);
        kaaba(P, 50, 56);
        font('800', 42); g.fillStyle = C.em; g.fillText('Umrohku', P + 74, 94);
        if (!noDate) { font('600', 24); g.fillStyle = C.mute; g.textAlign = 'right'; g.fillText('Data ' + (D.updated || '') + ' WIB', W - P, 90); g.textAlign = 'left'; }
        // blok judul (lebih pendek bila tanpa subjudul)
        font('800', 64);
        var nl = wrap(title, W - 2 * P - 88, 2).length;
        var top = 140, h = 150 + nl * 74 + (sub ? 50 : 0);
        var gr = g.createLinearGradient(0, top, W, top + h); gr.addColorStop(0, C.em); gr.addColorStop(1, C.em2);
        rr(P, top, W - 2 * P, h, 36, gr);
        g.save(); g.globalAlpha = .12; g.fillStyle = '#fff'; g.beginPath(); g.arc(W - P - 40, top + 40, 150, 0, 7); g.fill(); g.restore();
        font('800', 26); g.fillStyle = C.gold; g.fillText(kicker.toUpperCase(), P + 44, top + 66);
        font('800', 64); g.fillStyle = '#fff';
        var tl = wrap(title, W - 2 * P - 88, 2); tl.forEach(function (l, i) { g.fillText(l, P + 44, top + 140 + i * 74); });
        if (sub) { font('500', 30); g.fillStyle = 'rgba(255,255,255,.85)'; g.fillText(fit(sub, W - 2 * P - 88), P + 44, top + 140 + tl.length * 74 + 10); }
        return top + h + 28;
    }
    function footer(note) {
        var y = H - 92;
        font('500', 21); g.fillStyle = C.mute;
        wrap(note || 'Harga dari situs resmi travel saat data diambil dan bisa berubah. Konfirmasi ke travel sebelum DP.', 600, 2)
            .forEach(function (l, i) { g.fillText(l, P, y + 22 + i * 28); });
        font('800', 30); var t = 'umrohku.web.id', tw = g.measureText(t).width + 56;
        rr(W - P - tw, y - 6, tw, 64, 32, C.em); g.fillStyle = '#fff'; g.fillText(t, W - P - tw + 28, y + 37);
    }
    function empty(y, text) { font('600', 32); g.fillStyle = C.mute; g.textAlign = 'center'; g.fillText(text, W / 2, y + 200); g.textAlign = 'left'; }

    // Baris paket: rank, logo, travel, nama paket, info, harga (+badge opsional).
    function rows(list, y, opt) {
        var n = list.length, gap = 14, rh = Math.min(140, Math.floor((H - 130 - y - gap * (n - 1)) / Math.max(n, 1)));
        list.forEach(function (d, i) {
            var x = P, w = W - 2 * P, yy = y + i * (rh + gap);
            rr(x, yy, w, rh, 26, C.card);
            font('800', 26); g.fillStyle = C.gold; g.textAlign = 'center'; g.fillText(String(i + 1), x + 34, yy + rh / 2 + 9); g.textAlign = 'left';
            logo(d, x + 62, yy + (rh - 84) / 2, 84);
            var tx = x + 166, priceW = 300, tw = w - 166 - priceW - 24;
            font('800', 31); g.fillStyle = C.ink; g.fillText(fit(d.travel, tw), tx, yy + 44);
            font('500', 24); g.fillStyle = C.mute; g.fillText(fit(d.package, tw), tx, yy + 80);
            var meta = [tgl(d.date), d.duration ? d.duration + ' hari' : '', d.airline ? d.airline + (d.direct ? ' · langsung' : '') : ''].filter(Boolean).join(' · ');
            font('600', 23); g.fillStyle = C.em2; g.fillText(fit(meta, tw), tx, yy + 114);
            g.textAlign = 'right';
            if (opt && opt.seat) {
                var s = 'Sisa ' + d.seats + ' seat'; font('800', 26); var sw = g.measureText(s).width + 32;
                rr(x + w - 26 - sw, yy + 22, sw, 46, 23, '#FFE4E6'); g.fillStyle = C.rose; g.fillText(s, x + w - 42, yy + 54);
                font('800', 30); g.fillStyle = C.ink; g.fillText(rp(d.price), x + w - 28, yy + 112);
            } else {
                font('600', 21); g.fillStyle = C.mute; g.fillText('per orang', x + w - 28, yy + 50);
                font('800', 36); g.fillStyle = C.em; g.fillText(rp(d.price), x + w - 28, yy + 96);
            }
            g.textAlign = 'left';
        });
    }

    // ---------- template ----------
    function month() { var k = document.getElementById('kMonth').value; return (D.months || []).find(function (m) { return m.key === k; }) || (D.months || [])[0]; }

    var T = {
        termurah: function () {
            var m = month();
            if (!m) { empty(frame('Paket umrah termurah', 'Belum ada data', ''), 'Data belum tersedia'); footer(); return ''; }
            var y = frame('Paket umrah termurah', 'Berangkat ' + m.title, m.count + ' jadwal dari ' + m.travels + ' travel · mulai ' + rp(m.min));
            rows(m.cheapest, y); footer();
            var list = m.cheapest.map(function (d, i) { return (i + 1) + '. ' + d.travel + ', ' + rp(d.price) + ' (' + tglPendek(d.date) + ', ' + d.duration + ' hari)'; }).join('\n');
            return 'Umrah ' + m.title + ' paling murah ternyata mulai ' + rp(m.min) + ' 👇\n\n' + list +
                '\n\nAda ' + m.count + ' jadwal dari ' + m.travels + ' travel bulan ini. Bandingkan semuanya (hotel, maskapai, sisa seat) di umrohku.web.id, gratis.\n\n' +
                'Data ' + D.updated + ' WIB dari situs resmi travel, harga bisa berubah.\n\n#umroh #paketumroh #umrohmurah #umroh' + m.key.slice(0, 4) + ' #persiapanumroh';
        },
        seat: function () {
            var L = D.lowSeats || [];
            var y = frame('Sisa seat menipis', 'Jadwal yang hampir penuh', 'Sisa seat paling sedikit, keberangkatan 4 bulan ke depan');
            if (!L.length) empty(y, 'Belum ada travel yang menyebut sisa seat'); else rows(L, y, { seat: true });
            footer('Sisa seat sesuai situs travel saat data diambil. Bisa berubah cepat, konfirmasi ke travel.');
            return 'Buat yang lagi nimbang-nimbang: jadwal ini sisa seat-nya tinggal sedikit ⏳\n\n' +
                L.map(function (d, i) { return (i + 1) + '. ' + d.travel + ', ' + tglPendek(d.date) + ': sisa ' + d.seats + ' seat (' + rp(d.price) + ')'; }).join('\n') +
                '\n\nCek tanggal lain dan bandingkan harganya di umrohku.web.id\n\n#umroh #paketumroh #jadwalumroh #umrohkeluarga #infoumroh';
        },
        tren: function () {
            var M = (D.months || []).slice(0, 9);
            var y = frame('Tren harga umrah', 'Harga termurah per bulan', 'Paket termurah yang masih bisa dipesan di tiap bulan');
            var x0 = P + 10, x1 = W - P - 10, top = y + 70, bot = H - 210, n = M.length;
            rr(P, y, W - 2 * P, H - 130 - y - 20, 30, C.card);
            if (!n) { empty(y, 'Belum ada data'); footer(); return ''; }
            var max = Math.max.apply(null, M.map(function (m) { return m.min; })), lo = Math.min.apply(null, M.map(function (m) { return m.min; }));
            var base = 0; // mulai dari 0 supaya selisih harga tidak terlihat dilebih-lebihkan
            var bw = (x1 - x0 - 40) / n;
            M.forEach(function (m, i) {
                var h = (m.min - base) / (max - base || 1) * (bot - top - 60), bx = x0 + 20 + i * bw + bw * .18, w = bw * .64;
                var cheapest = m.min === lo;
                rr(bx, bot - h, w, h, 14, cheapest ? C.gold : C.em2);
                font('800', 22); g.fillStyle = C.ink; g.textAlign = 'center'; g.fillText(jt(m.min), bx + w / 2, bot - h - 14);
                font('600', 22); g.fillStyle = C.mute; var p = m.key.split('-'); g.fillText(BLN[+p[1]], bx + w / 2, bot + 34); g.fillText("'" + p[0].slice(2), bx + w / 2, bot + 60);
                g.textAlign = 'left';
            });
            var gaps = [];
            for (var i = 1; i < M.length; i++) {
                var a = M[i - 1].key.split('-'), b = M[i].key.split('-'), ia = +a[0] * 12 + +a[1], ib = +b[0] * 12 + +b[1];
                for (var j = ia + 1; j < ib; j++) gaps.push(BLN[(j - 1) % 12 + 1]);
            }
            footer('Batang emas = bulan termurah.' + (gaps.length ? ' ' + gaps.join(' & ') + ': belum ada jadwal dari travel.' : '') + ' Harga per orang saat data diambil.');
            var cheap = M.find(function (m) { return m.min === lo; }), dear = M.find(function (m) { return m.min === max; });
            return 'Kapan umrah paling murah? Ini harga termurah per bulan dari ' + D.travels + ' travel 📊\n\n' +
                M.map(function (m) { return '• ' + m.title + ': mulai ' + rp(m.min); }).join('\n') +
                '\n\nTermurah: ' + cheap.title + '. Termahal: ' + dear.title + '.\n\nBandingkan semua paketnya di umrohku.web.id\n\n#umroh #hargaumroh #paketumroh #umrohmurah #tipsumroh';
        },
        ramadhan: function () {
            var R = D.ramadhan || [];
            var y = frame('Umrah Ramadhan', 'Paket Ramadhan termurah', 'Berangkat saat Ramadhan ' + (D.ramLabel || '') + ' (perkiraan)');
            if (!R.length) empty(y, 'Belum ada paket Ramadhan'); else rows(R, y);
            footer('Tanggal Ramadhan perkiraan (±1 hari). Harga dari situs resmi travel saat data diambil.');
            return 'Mau umrah pas Ramadhan? Ini paket termurah yang berangkat ' + (D.ramLabel || '') + ' 🌙\n\n' +
                R.map(function (d, i) { return (i + 1) + '. ' + d.travel + ', ' + rp(d.price) + ' (' + tglPendek(d.date) + ', ' + d.duration + ' hari)'; }).join('\n') +
                '\n\nKursi Ramadhan biasanya cepat habis. Bandingkan di umrohku.web.id\n\n#umrohramadhan #umroh #paketumroh #ramadhan #itikaf';
        },
        tips: function () {
            var k = document.getElementById('kKicker').value, t = document.getElementById('kTitle').value;
            var pts = document.getElementById('kPoints').value.split('\n').map(function (s) { return s.trim(); }).filter(Boolean).slice(0, 6);
            var y = frame(k || 'Persiapan umrah', t || 'Judul', '', true);
            var n = pts.length || 1, gap = 14, rh = Math.min(150, Math.floor((H - 130 - y - gap * (n - 1)) / n));
            pts.forEach(function (p, i) {
                var yy = y + i * (rh + gap); rr(P, yy, W - 2 * P, rh, 26, C.card);
                g.beginPath(); g.arc(P + 52, yy + rh / 2, 26, 0, 7); g.fillStyle = C.em; g.fill();
                font('800', 26); g.fillStyle = '#fff'; g.textAlign = 'center'; g.fillText(String(i + 1), P + 52, yy + rh / 2 + 9); g.textAlign = 'left';
                font('600', 30); g.fillStyle = C.ink;
                var ls = wrap(p, W - 2 * P - 130, Math.max(1, Math.floor((rh - 20) / 38)));
                var sy = yy + rh / 2 - (ls.length - 1) * 19 + 10;
                ls.forEach(function (l, j) { g.fillText(l, P + 100, sy + j * 38); });
            });
            footer('Checklist lengkap & pengingat tenggat: umrohku.web.id/persiapan');
            return (t || '') + ' 👇\n\n' + pts.map(function (p, i) { return (i + 1) + '. ' + p; }).join('\n') +
                '\n\nChecklist lengkap (bisa dicentang & ada tenggatnya) di umrohku.web.id/persiapan\n\n#persiapanumroh #tipsumroh #umroh #manasikumroh #jamaahumroh';
        }
    };

    var PRESETS = [
        { k: 'Persiapan umrah', t: 'Cek paspormu sebelum DP', p: ['Masa berlaku minimal 6 bulan saat berangkat', 'Nama di paspor = nama di tiket & visa', 'Buat lewat app M-Paspor, bayar sebelum batas waktu', 'Bawa KTP, KK, akta lahir/buku nikah asli', 'Simpan scan paspor di HP & email'] },
        { k: 'Wajib sebelum berangkat', t: 'Vaksin meningitis & ICV', p: ['Wajib untuk jamaah umrah', 'Paling lambat 10 hari sebelum berangkat', 'Di BKK, RSUD, atau klinik penerbit ICV', 'Nama di ICV harus sama dengan paspor', 'Simpan e-ICV yang dikirim lewat email'] },
        { k: 'Aplikasi wajib', t: 'Daftar akun Nusuk', p: ['Aplikasi resmi Pemerintah Arab Saudi', 'Unduh di Google Play / App Store', 'Daftar pakai nama & nomor paspor', 'Dipakai untuk izin ibadah & jadwal Raudhah', 'Pesan Raudhah sebelum tiba di Madinah'] },
        { k: 'Saat ihram', t: 'Larangan ihram yang sering terlupa', p: ['Memakai wewangian (termasuk sabun wangi)', 'Memotong rambut atau kuku', 'Laki-laki: pakaian berjahit & penutup kepala', 'Perempuan: menutup wajah & telapak tangan', 'Bertengkar, berkata kotor, berburu'] },
        { k: 'Sebelum pilih travel', t: 'Cek 5 hal ini sebelum DP', p: ['Izin PPIU Kemenag masih berlaku', 'Tanggal berangkat & pulang jelas', 'Hotel, jarak ke masjid, dan maskapai tertulis', 'Apa saja yang tidak termasuk harga', 'Bandingkan dulu di umrohku.web.id'] },
        { k: 'Perlengkapan', t: 'Barang yang wajib dibawa', p: ['2 set kain ihram + sabuk (laki-laki)', 'Sandal yang tidak menutup mata kaki', 'Sabun & sampo tanpa pewangi', 'Power bank maks. 10.000 mAh di kabin', 'Obat pribadi + surat dokter'] }
    ];
    var ps = document.getElementById('kPreset');
    PRESETS.forEach(function (p, i) { var o = document.createElement('option'); o.value = i; o.textContent = p.t; ps.appendChild(o); });
    function preset(i) { var p = PRESETS[i]; document.getElementById('kKicker').value = p.k; document.getElementById('kTitle').value = p.t; document.getElementById('kPoints').value = p.p.join('\n'); }
    ps.onchange = function () { preset(+ps.value); draw(); };
    preset(0);

    var cap = document.getElementById('kCaption');
    function draw() {
        g.textAlign = 'left'; g.textBaseline = 'alphabetic';
        cap.value = T[state.t]() || '';
        document.querySelectorAll('.k-o').forEach(function (el) { el.style.display = el.classList.contains('k-o-' + state.t) ? '' : 'none'; });
        document.querySelectorAll('.k-tab').forEach(function (b) { b.setAttribute('aria-pressed', b.dataset.t === state.t); });
    }
    document.querySelectorAll('.k-tab').forEach(function (b) { b.onclick = function () { state.t = b.dataset.t; draw(); }; });
    ['kMonth', 'kKicker', 'kTitle', 'kPoints'].forEach(function (id) { document.getElementById(id).addEventListener('input', draw); });
    document.getElementById('kCopy').onclick = function () {
        navigator.clipboard.writeText(cap.value).then(function () { var b = document.getElementById('kCopy'); b.textContent = 'Tersalin ✓'; setTimeout(function () { b.textContent = 'Salin caption'; }, 1500); });
    };
    document.getElementById('kDownload').onclick = function () {
        var a = document.createElement('a');
        var name = 'umrohku-' + state.t + (state.t === 'termurah' ? '-' + document.getElementById('kMonth').value : '') + '-' + new Date().toISOString().slice(0, 10) + '.png';
        a.download = name; a.href = cv.toDataURL('image/png'); a.click();
    };
    if (document.fonts && document.fonts.ready) document.fonts.ready.then(draw);
    draw();
})();
