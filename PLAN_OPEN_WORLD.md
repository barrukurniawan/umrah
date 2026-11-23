# Rencana: Jelajah Masjidil Haram (open world)

Disusun 2026-11-05 oleh Claude Fable 5.1 bersama Barru. Pengerjaan tiap tahap: Claude Opus 5.5, effort **xhigh**
(revisi kecil: high). Berkas ini adalah sumber kebenaran lintas sesi: baca dulu, kerjakan tahap berikutnya,
lalu perbarui bagian **Status** di bawah.

## 1. Tujuan

Satu menu baru di halaman simulasi, **"Jelajah"** (`/simulasi?bagian=haram`): pemain (chibi yang sudah ada)
berjalan bebas di pelataran, masuk lewat pintu mana pun, melihat Ka'bah dari mataf, masuk ke lorong Mas'a, dan dari
sana bisa langsung **Mulai Tawaf** atau **Mulai Sa'i** memakai simulasi yang sudah jadi. Target utama tetap HP.

Yang **tidak** masuk lingkup awal: lantai 1/2/atap, interior hotel, Zamzam bawah tanah, jalur bus, siang/malam.

## 2. Keputusan yang sudah diambil

| # | Keputusan | Alasan / alternatif |
|---|-----------|--------------------|
| K1 | **Denah dari OpenStreetMap**, diambil sekali oleh skrip dan **disimpan ke repo** (`web/static/sim/data/haram.json`). Tidak ada panggilan ke OSM saat halaman dibuka. | Data OSM lengkap (lihat §3). Overpass sering menolak/lambat, jadi jangan dipakai saat runtime. |
| K2 | **Lantai dasar saja** (`level=0`). Pintu di level -1/1/2 diabaikan. | Lantai lain menyusul kalau lantai dasar sudah enak dimainkan. |
| K3 | **Skala K = 0,5** untuk semua jarak dan bangunan; **Ka'bah, Hijr Ismail, Maqam Ibrahim tetap ukuran asli** (10×12 m seperti di `tawaf.js`). | Pada skala asli, pelataran ke Ka'bah 3–5 menit jalan. Dengan K=0,5 dunia ±440×400 m, masih seukuran dunia tawaf sekarang (tanah r=200, kabut 260). Ka'bah sengaja tidak dikecilkan supaya tetap menonjol. |
| K4 | **Sistem koordinat sama dengan `tawaf.js`**: pusat Ka'bah = (0,0), x = timur, −z = utara, satuan meter dunia. Proyeksi: ekuirektangular di sekitar pusat Ka'bah (galat < 1 cm pada radius 1 km). | Objek tawaf bisa dipindah ke sini tanpa ubah angka. |
| K5 | **Tawaf & Sa'i tidak digabung ke dalam open world.** Dari open world, tombol "Mulai Tawaf"/"Mulai Sa'i" membuka halaman yang sudah ada (`?bagian=tawaf&g=m`). Tombol "Kembali ke Jelajah" di sana mengembalikan pemain ke titik terakhir (disimpan di `sessionStorage`). | Kedua simulasi sudah diuji dan punya alur tersendiri (kartu pemandu, doa, hitung putaran). Menggabungkannya menggandakan risiko. Penggabungan sungguhan = tahap opsional terakhir. |
| K6 | **Tabrakan berbasis segmen 2D** (dinding, tepi hotel, Ka'bah) + **lingkaran** (tiang). Tidak ada navmesh. | Cukup untuk "jalan bebas, tidak tembus tembok", murah di HP, mudah dibuat dari poligon OSM. |
| K7 | **Kendali**: HP = joystick virtual kiri + geser kanan untuk putar kamera; desktop = WASD/panah + geser mouse. Ditambah **"Lompat ke…"** (daftar pintu & tempat) untuk yang tidak mau jalan jauh. Tombol "Jalan" model lama tidak dipakai di mode ini. | Jalur tawaf/sa'i tetap; open world butuh kemudi. |
| K8 | **Kamera**: orang ketiga di belakang punggung (bisa diputar, zoom cubit/scroll yang sudah ada) + mode atas (tombol `btnCam` yang sudah ada). **Mini-map 2D** (kanvas) selalu tampil: denah, titik pemain, nomor pintu. | Peta yang diminta Barru = mini-map ini, dibuat dari poligon yang sama. |
| K9 | **Hotel & Clock Tower = siluet** (kotak ekstrusi dari poligon OSM × tinggi × K), warna datar, tidak bisa dimasuki. Pelataran dibatasi "pagar tak terlihat" di radius ±300 m dunia. | Performa, dan bukan bagian ibadah. |
| K10 | **Semua geometri dari kode** (seperti sekarang): tiang, lengkung, atap, menara, kubah. Tanpa berkas glTF. | Konsisten dengan gaya chibi; cache gratis; tidak ada aset berat. |
| K11 | **Atribusi**: tulis "Denah: © kontributor OpenStreetMap (ODbL)" di halaman simulasi; `haram.json` ikut dipublikasikan apa adanya (sudah terpenuhi karena dilayani dari `/static`). | Kewajiban lisensi ODbL untuk basis data turunan. |
| K12 | Gambar peta yang dikirim Barru (JOM KENAL, peta Hajj, peta 3D, Gate Map satelit, peta kota) **hanya acuan nama**. Tidak ada yang ditiru gaya/garisnya. | Hak cipta pihak lain. **Catatan**: Gate Map satelit itu terbalik (selatan di atas). Patokan arah = OSM/Google Maps (utara di atas): Mas'a di sisi timur, Safa tenggara, Marwah timur laut, King Abdul Aziz selatan menghadap Clock Tower, King Fahd barat daya, Umrah barat, King Abdullah barat laut. |
| K13 | **Lantai bertingkat** (masukan Barru 2026-11-05): lantai masjid, mataf, dan Mas'a di y = 0; **pelataran dan ambang semua pintu satu lantai lebih tinggi** (`LIFT` = 5). Masuk pintu mana pun → bordes → **eskalator/tangga turun** (30°, 8,7 m) ke lantai masjid. Ka'bah tetap terlihat dari luar pintu karena tingginya. | Seperti aslinya: dari Pintu King Abdul Aziz dll. jamaah turun satu lantai. Pintu yang berdempetan (< 2 m) digabung jadi satu deret eskalator; pintu yang tidak muat bordes (12 dari 135) jadi pintu tertutup. |
| K14 | **Gerbang "King"** (King Abdul Aziz 1, King Fahd 79, King Abdullah 100) = **portal raksasa**: bukaan 9 × 10,4 m (6 orang berjajar), portal menjulang ±10 m di atas dinding, bingkai lengkung iwan, plakat Arab + Inggris + nomor, merlon, finial emas. Pintu utama lain: portal kecil + plakat; pintu biasa: lengkung + nomor. | Masukan Barru 2026-11-05. Barru menyebut ada 4 gerbang King; di OSM lantai dasar hanya 3 (yang keempat kemungkinan King Salman, belum terpetakan): tambahkan lewat `extra.json`/data manual bila Barru mengirim lokasinya. |
| K15 | **Karakter tertutup tiang/dinding → siluet hijau samar** (stencil): bangunan tetap utuh, tidak dilubangi/transparan; hanya bagian karakter yang tertutup yang digambar di atasnya. | Aula penuh tiang membuat karakter sering tertutup; kamera tidak boleh bergerak sendiri, menembus, atau maju ke kepala (masukan Barru Tahap 1). Bisa dimatikan bila Barru tidak suka. |

## 3. Data OSM yang tersedia (dicek 2026-11-05)

Kotak: lat 21.4190–21.4265, lon 39.8215–39.8310. Pusat Ka'bah ≈ 21.42251, 39.82617.

- **Relasi 1472531** "Great Mosque of Mecca": multipolygon lantai dasar, 29 way luar + 1 lubang (mataf), 577 titik;
  `height=21`, `building:levels=2`. Ini garis masjid.
- **Way 103914569** Ka'bah (`height=15`), **way 315911894** Al-Hijr (`height=1.23`), **way 473301379** Maqam Ibrahim,
  node Black Stone 1199442721, Shami Corner 1199442699, Yemen Corner 1199442700, Iraqi Corner 8794911100.
- **Node As Safa 4589923995, Al Marwa 4589923996** (`natural=peak`); **way 584203412** "Sa'i / Al Masa'a Safa–Marwa";
  **way 584197021** "Tawaf" (lintasan); **way 742725721** "Al Raml Area".
- **166 node `entrance`**, 14 di antaranya `entrance=main` dengan nama & nomor: King Abdul Aziz 1, Ajyad 5, Bani Shaibah 21/22,
  Bab Al Marwah 25, Quraish 28, Fateh 30, Al Quds 35, Hudaybiyah 39, Umra 40, King Fahd 79, King Abdullah 100, dll.
  Saring `level` = 0 atau kosong.
- **13 way `man_made=tower`** = menara (posisi asli). 20 node `mast` = tiang telekom (abaikan).
- Hotel dengan tinggi: Clock Towers (rel. 12896421, 601 m), Abraj As Safwah 160, Dar Al Tawhid 70, Le Meridien 75,
  Al Safa Royal Palace 50, Jabal Omar (Conrad/Hyatt 100), Makkah Millennium 95, Al Jawar 115, dll.
- 72 way `highway=pedestrian` (pelataran, jembatan Ajyad/Safa/Nabi), `indoor=wall` (beberapa dinding dalam), tangga.

**Tempat yang tidak ada di OSM** (dicek 2026-11-05: Half Million dan Archi tidak terpetakan di seluruh Makkah) ditaruh
manual di `scripts/haram-osm/extra.json`: `{ "name", "lat", "lon", "kind": "cafe|hotel|shop", "h": tinggi opsional }`.
`build.py` menggabungkannya ke `haram.json` sebagai `landmarks`, lalu tampil sebagai papan nama di dunia, titik di mini-map,
dan pilihan "Lompat ke…". Koordinat diambil dari Google Maps (klik kanan → salin lat,lon). Sudah terisi: **Half Million** cabang Al Haram
(21.42013, 39.82393; ±350 m barat daya Ka'bah, seberang Ibrahim Al Khalil Rd dari Pintu King Abdul Aziz).
Masih menunggu dari Barru: **Archi** (jenis tempat dan koordinat).

Cara ambil (Overpass menolak POST mentah dengan 406; harus form-encoded + User-Agent):
```
curl -s -A "umrohku-sim/1.0" --data-urlencode "data@query.txt" https://overpass-api.de/api/interpreter -o raw.json
```

## 4. Arsitektur

```
scripts/haram-osm/
  query.txt        Overpass QL (kotak di atas, out geom)
  fetch.py         jalankan Overpass -> raw/haram-raw.json (di-commit, supaya build bisa diulang tanpa jaringan)
  build.py         raw -> web/static/sim/data/haram.json (proyeksi meter, skala K, penyederhanaan, pemilahan)
  README.md        cara pakai + catatan lisensi

web/static/sim/
  data/haram.json  denah siap pakai (target < 120 KB; aktifkan compress untuk /static/sim/data di main.go)
  kaaba.js         (dipindah dari tawaf.js) buildKaaba(kit) -> {group, cornersWorld, hijrSegments, maqam}
  haram-world.js   pembangun geometri dari haram.json: lantai, dinding+pintu, tiang, atap, menara, mas'a, hotel
  nav.js           tabrakan segmen/lingkaran, joystick, kemudi WASD, mini-map
  haram.js         halaman: createSim({part:'haram', freeRoam:true}) + hooks + kerumunan + titik masuk tawaf/sa'i
  common.js        + opsi freeRoam (sembunyikan tombol Jalan, pasang joystick), + sim.saveSpot()/sim.loadSpot()
  tawaf.js, sai.js memakai kaaba.js; tombol "Kembali ke Jelajah" bila datang dari haram

web/templates/simulasi.html   tab "Jelajah", mini-map, joystick, atribusi OSM, preload haram.js & haram.json
app/handlers/handler.go       bagian=haram -> Part "haram", judul "Jelajah Masjidil Haram"
```

### Format `haram.json` (semua koordinat sudah dunia: meter × K, x timur, z selatan)
```json
{ "k": 0.5, "origin": [21.42251, 39.82617], "attribution": "© OpenStreetMap contributors, ODbL",
  "mosque": { "outer": [[x,z],...], "inner": [[x,z],...] },
  "kaaba":  { "poly": [...], "rot": 0.78 }, "hijr": [...], "maqam": [x,z], "blackStone": [x,z],
  "safa": [x,z], "marwah": [x,z], "saiPath": [[x,z],...], "tawafPath": [[x,z],...], "raml": [...],
  "gates":  [{ "name": "King Fahd", "ref": "79", "pos": [x,z], "main": true }, ...],
  "minarets": [[x,z],...],
  "buildings": [{ "name": "Clock Towers", "poly": [...], "h": 300 }, ...],
  "plaza": [[...],...] }
```

### Alur masuk ibadah (K5)
- Pemain masuk zona mataf (jarak ke Ka'bah < 30) → kartu `sim.info` "Mulai Tawaf" → `/simulasi?bagian=tawaf&g=<gender>&from=haram`.
- Pemain masuk zona Safa (jarak < 8) → "Mulai Sa'i" → `/simulasi?bagian=sai&g=<gender>&from=haram`.
- `sessionStorage.ukHaramSpot = {x,z,yaw,gender}` disimpan saat keluar; halaman tawaf/sa'i dengan `from=haram`
  menampilkan tombol "Kembali ke Jelajah" (kartu akhir), yang membuka `?bagian=haram` dan memulihkan posisi.

### Anggaran performa (ukur dengan `renderer.info`)
- Penuh: ≤ 350 k segitiga, ≤ 120 draw call, ≥ 50 fps di laptop biasa, ≥ 30 fps di HP menengah.
- Ringan (`hooks.lite`): kerumunan 40 %, hotel tanpa detail, kabut lebih dekat (160/220), ≤ 180 k segitiga.
- Geometri statis digabung per **sektor** (8 irisan sudut dari Ka'bah) supaya frustum culling bekerja saat kamera di dekat tembok.

## 5. Tahapan

Setiap tahap: selesai → uji otomatis (§6) → Barru coba di HP → commit → lanjut. Perkiraan 1 tahap ≈ 1 sesi.

### Tahap 0 – Data (model: Opus 5.5 xhigh)
1. `fetch.py` + `build.py` + `haram.json`. Proyeksi, skala K, buang titik < 0,6 m (Douglas–Peucker), lubang mataf jadi `inner`.
2. Pintu: cocokkan tiap node `entrance` ke segmen garis masjid terdekat (≤ 3 m); simpan posisi di garis + arah normal. Hanya level 0.
3. Gambar **cek.png** (matplotlib/PIL) dari `haram.json`: garis masjid, pintu bernomor, Safa/Marwah, Ka'bah, hotel.
   **Kriteria lulus**: cek.png cocok dengan Google Maps (utara di atas): Safa tenggara, Marwah timur laut, King Abdul Aziz selatan
   menghadap Clock Tower, King Fahd barat daya, King Abdullah barat laut, Half Million di selatan seberang jalan.
4. Pindahkan Ka'bah ke `kaaba.js`; tawaf.js memakainya; sheet screenshot tawaf tidak berubah.

### Tahap 1 – Bisa jalan-jalan (Opus 5.5 xhigh)
1. `haram.js` + `haram-world.js` tahap awal: tanah pelataran (tekstur marmer), lantai masjid dari poligon (ShapeGeometry),
   dinding dari segmen (tinggi 21×K = 10,5) dengan celah pintu lebar 8, Ka'bah dari `kaaba.js`, hotel siluet, menara.
2. `nav.js`: tabrakan segmen+lingkaran dengan geser (slide), joystick, WASD, kamera orang ketiga, mini-map, "Lompat ke…".
3. `common.js` opsi `freeRoam`; template: tab Jelajah, elemen mini-map/joystick, atribusi.
4. Handler `bagian=haram`.
   **Kriteria lulus**: pemain bisa jalan dari Pintu King Fahd ke Ka'bah tanpa tembus dinding; mini-map benar; 60 fps laptop.

### Tahap 2 – Kit bangunan (Opus 5.5 xhigh)
1. Serambi: tiang + lengkung sepanjang sisi dalam garis masjid (generalisasi `octPoints` → `edgePoints(poly, spacing)`), atap, kubah, lampu.
2. Gerbang utama: lengkung besar + nomor pintu + papan nama (label HTML `sim.addLabel`, muncul < 60 m).
3. Mas'a: lorong tertutup dari Safa ke Marwah (pakai pola `sai.js`: pilar, langit-langit, lampu hijau), bukit Safa/Marwah kecil.
4. Mataf: lantai lingkaran (tekstur `tawaf.js`), Hijr, Maqam, Hajar Aswad, Multazam, keran Zamzam.
5. Gabung geometri per sektor; ukur anggaran performa; mode ringan.
   **Kriteria lulus**: screenshot 5 sudut (4 gerbang + atas) terlihat "ini Masjidil Haram"; anggaran §4 terpenuhi.

### Tahap 3 – Kerumunan & kehidupan (Opus 5.5 high)
1. `createCrowd` 150 orang (ringan 60): berjalan antar titik tujuan (pintu ↔ mataf ↔ Safa) mengikuti jalur sederhana
   (garis lurus + tabrakan yang sama), tawaf berputar di mataf (pakai logika kerumunan `tawaf.js`), sa'i bolak-balik di Mas'a.
2. Orang duduk/salat di serambi (pose statis instanced).
   **Kriteria lulus**: tidak ada yang tembus tembok; fps tetap.

### Tahap 4 – Masuk ibadah & konten (Opus 5.5 high)
1. Zona Mulai Tawaf / Mulai Sa'i, simpan-pulihkan posisi, tombol "Kembali ke Jelajah" di tawaf/sa'i.
2. Kartu info saat mendekati tempat: Hajar Aswad, Multazam, Hijr Ismail, Maqam Ibrahim, Zamzam, Safa, Marwah,
   pintu-pintu utama (teks singkat + doa bila ada), memakai `sim.info`/`sim.pops` yang sudah ada.
3. Murottal lanjut dari halaman lain (mekanisme sudah ada).
   **Kriteria lulus**: alur Jelajah → Tawaf → Kembali → Sa'i → Kembali mulus di HP.

### Tahap 5 – Rilis (Opus 5.5 high)
1. `?v=` naik, preload, `compress` untuk `/static/sim/data`, atribusi, teks menu di beranda.
2. Uji puppeteer di situs live (tawaf, sai, haram), lalu commit & deploy (lihat memori `deploy-umrohku`).

### Opsional (nanti)
- Lantai 1 & atap; tawaf di dalam open world (tanpa pindah halaman); waktu salat (jamaah berbaris); malam.

## 6. Cara uji (tiap tahap)
- Server lokal: build ke scratchpad, `APP_PORT=8091` (memori `sim-visual-check`).
- Puppeteer-core dari `/Users/barrukurniawan/Documents/wedding-summer/node_modules/puppeteer-core`, Chrome `--use-angle=swiftshader`.
- `window.__sim` + tambahan untuk haram: `__sim.goto(x,z)`, `__sim.walk(dx,dz,sec)`; cek `renderer.info.render`.
- Sheet screenshot: dari Pintu King Fahd, King Abdul Aziz, King Abdullah, Umrah, dan tampak atas; simpan di scratchpad `prev/`.
- Regresi tawaf/sa'i: bandingkan sheet lama (`prev/sheet-sesudah.png`, `prev/sheet-gerakan.png`).

## 7. Risiko & jawaban
- **Overpass tidak stabil** → raw di-commit; build tidak butuh jaringan.
- **Geometri OSM berlubang/tumpang tindih** → `build.py` memvalidasi poligon (shapely `buffer(0)`), gambar cek.png.
- **HP berat** → anggaran §4, mode ringan, sektor, hotel siluet; ukur sebelum menambah detail.
- **Joystick canggung** → "Lompat ke…" dan ketuk di mini-map sebagai jalur alternatif.
- **Lingkup membengkak** → lantai dasar saja; fitur opsional ditahan sampai Tahap 5 rilis.

## 8. Status
- [x] Tahap 0 – Data
- [x] Tahap 1 – Jalan-jalan
- [x] Tahap 2 – Kit bangunan
- [x] Tahap 3 – Kerumunan
- [x] Tahap 4 – Masuk ibadah & konten
- [ ] Tahap 5 – Rilis

Catatan kemajuan (tulis tanggal + apa yang selesai + yang belum):
- 2026-11-05: rencana disusun; data OSM dicek (§3); belum ada kode.
- 2026-11-05 **Tahap 0 selesai** (Opus 5.5):
  - `scripts/haram-osm/` (query, fetch.py, build.py, raw 388 KB, extra.json, README, cek.png).
  - `web/static/sim/data/haram.json` 38 KB (gzip 12 KB): 3 ring luar masjid (bangunan utama+Mas'a, sisi utara,
    perluasan King Abdullah), 1 lubang mataf, 126 pintu lantai dasar (13 bernama), 13 menara, 172 bangunan sekitar,
    Half Million. Area Ka'bah radius 22 m tidak dikecilkan; di luarnya jarak radial dipetakan mulus
    (`r' = 22 + (r-22)·K`), bukan dikali K mentah, supaya tidak ada lompatan di tepi area.
  - Ka'bah OSM: rotasi 33,5° (tawaf.js pakai 45°), dipasang dengan mencocokkan 4 sudut (galat 1,2 m). Simpan di `kaaba.rot`.
  - Tidak ada `plaza` area di OSM (plaza = semua yang bukan bangunan): Tahap 1 pakai satu bidang tanah besar.
  - Jembatan (`bridge: true`) ada di antara bangunan utama dan perluasan: gambar melayang, tanpa tabrakan.
  - Gerbang 16 hanya punya nama Arab (`nameAr`). Bab Al Salam ada di level -1, jadi tidak masuk.
  - `kaaba.js` (buildKaaba) dipisah dari tawaf.js; uji regresi dengan Math.random berbenih: segitiga/draw call/geometri
    identik (198.573 / 90 / 87), screenshot sama kecuali posisi jamaah. Semua `?v=` naik ke 31.
  - Clock Towers punya `height=601` untuk seluruh kompleks (alas besar). Tahap 1: alas ±60 m asli + menara jam di tengah sisi utara.
  - Masih menunggu Barru: koordinat & jenis tempat **Archi**.
- 2026-11-05 **Tahap 1 selesai** (Opus 5.5), belum di-commit/deploy, menunggu Barru coba:
  - Berkas baru: `nav.js` (tabrakan segmen+lingkaran di grid 8 m, geser di dinding, `clearUntil` untuk kamera; joystick,
    WASD/panah, Shift lari, Q/E putar, geser layar putar kamera), `haram-world.js` (Builder geometri gabungan, lantai,
    mataf bertekstur lingkaran, dinding 10,5 m dengan celah pintu 6 m utama / 3,2 m lain + ambang di atasnya, atap
    lempeng berlubang mataf, kubah kecil di tepi atap, 100+ tiang serambi di tepi mataf, 13 menara, 172 gedung siluet,
    jembatan melayang, Clock Tower 300 m dengan 4 jam, Half Million, Ka'bah dari kaaba.js rotasi OSM, Hijr & Maqam
    sebagai penghalang, penanda bukit Safa/Marwah, label pintu utama), `haram.js` (pemain, kamera orang ketiga 14 m /
    17 m di layar tegak, tidak keluar pintu saat di dalam masjid, tampak atas dari btnCam, zona Pelataran/Di dalam
    masjid/Mataf/Mas'a + pintu terdekat di panel, mini-peta 2,2x yang mengikuti pemain, ketuk = peta besar, ketuk
    peta besar = pindah, "Lompat ke…" 12 pintu utama + Ka'bah + Safa + Marwah + Half Million).
  - `common.js`: opsi `camLerp`. Template: tab **Jelajah**, joystick/peta/select hanya untuk `Part == "haram"`,
    atribusi OSM di pojok panggung + catatan kaki. Handler: `bagian=haram`. Semua `?v=` → 32 (chibi/kaaba tetap 31).
  - Uji: King Fahd → mataf 21,8 detik, 0 kali menembus dinding; tabrak tembok berhenti 0,45 m (= radius);
    joystick sentuh di HP 390×844 bergerak 15 m / 2,5 dtk; GPU 61 fps, 65 rb segitiga, 70 draw call
    (anggaran 350 rb / 120). Tawaf 306 rb / 90 dan Sa'i 228 rb / 86 tidak berubah, tanpa error.
  - Catatan untuk Tahap 2: aula dalam masjid masih kosong (hanya tiang di tepi mataf); lantai dalam & luar warnanya
    mirip; lorong Mas'a belum ada (baru bukit Safa/Marwah); kamera bisa menembus tiang (hanya dinding yang dicek).
  - Masukan Barru 2026-11-05: kamera bawaan = zoom terjauh (`cam.zoom = 1.9`), kecepatan 2x (jalan 12,4, Shift 19 m/dtk),
    titik muncul/lompat pintu 14 m di luar pintu. Uji ulang: King Fahd → mataf 10,9 detik, 0 tembus.
  - Kamera (setelah 3 putaran masukan Barru; yang DITOLAK: kamera menempel kepala karena gedung dianggap setinggi
    langit; kamera mencari arah terbuka sendiri = bergerak ke mana-mana; shader tembus pandang/bolong). Yang dipakai:
    arah kamera HANYA dari pemain (putar otomatis ke belakang punggung juga dimatikan). Tiap penghalang punya tinggi
    `h` (dan `lo` untuk ambang pintu & tepi atap serambi). Penghalang yang tetap di bawah garis pandang ke leher (1,6 m)
    diabaikan; selain itu kamera NAIK secukupnya (sampai ~89°, dari atas tak pernah terhalang), baru maju bila perlu.
    Di dalam masjid & di mataf saat kamera di atas serambi: kamera di bawah atap. Jarak/sudut dihaluskan (`camS`).
    Uji raycast "kepala tertutup": gang 0%, pintu 1 → mataf 1%, menyusuri gedung 6% (sesaat, saat kamera naik);
    menempel gedung rendah/sedang/tinggi: tidak tertutup, kamera 32 m dari karakter.
- 2026-11-05 **Tahap 2 selesai** (Opus 5.5), belum di-commit/deploy, menunggu Barru coba:
  - Berkas baru: `haram-geom.js` (Builder per irisan sudut, poligon, `offsetRing`, `rayRing`, peta wilayah grid 0,5 m
    untuk aula/mataf/pelataran, `mergeGeos`, `sectorInstanced`), `haram-kit.js` (tekstur fasad jendela lengkung 2 lantai,
    panel lengkung runcing ber-alpha, plafon Mas'a, anak tangga/eskalator, atlas plakat nama & nomor pintu; fasad mataf
    2 tingkat lengkung + kubah + finial; 483 tiang aula mengikuti bentuk mataf (jarak 11 m) + lengkung dekat plafon;
    Mas'a: 55 tiang 2 sisi, plafon berhias, lajur & jalur tengah, lampu hijau 30-44 % dari Safa, lampu gantung, bukit
    Safa/Marwah batu + kaca rendah; garis awal tawaf + lampu hijau; label Multazam, Rukun Yamani, Hijr Ismail;
    dispenser Zamzam di balik tiang fasad).
  - `haram-world.js` ditulis ulang: K13 (lantai, bordes, eskalator 94 deret, `floorY`), K14 (portal), dinding bertekstur
    fasad (celah pintu dalam koordinat keliling ring), pelataran = tutup berlubang di masjid pada y = LIFT, ring 0
    (perluasan King Abdullah) yang di OSM tanpa pintu lantai dasar diberi 9 pintu tanpa nomor di sisi yang menghadap
    Ka'bah. Atap ROOF_Y = 15,5; dinding sampai 17,1.
  - Kamera (`haram.js`): semua tinggi relatif lantai pemain. Di dalam masjid kamera selalu di dalam gedung dan di bawah
    `CAM_CEIL` (11,7 m; di bawah pita lengkung dekat plafon supaya panel lengkung tidak menutupi layar). Bila tertutup,
    kamera NAIK ke dekat plafon sambil mendekat mendatar (mulus, sampai hampir tegak di atas karakter), tidak pernah maju ke
    kepala. Di pelataran: naik saja (tanpa memendek). Kamera di atas atap tidak bisa "mengintip" lewat celah pintu.
    Melewati ambang pintu (dengan jeda 0,3 dtk; lorong portal dihitung "dalam") = kamera langsung pindah (cut), tidak
    melayang menembus dinding. Mode ringan: kabut 140/430, batas pandang 470.
  - Uji: tanpa error; 135 pintu (12 tertutup), 94 deret eskalator; uji tekan 13.780 frame jalan acak dekat pintu:
    0 lompatan tinggi mendadak, 0 tembus dinding; King Abdul Aziz pelataran 5 m -> bordes -> eskalator -> lantai 0.
    Kepala tertutup: masuk pintu 1 & aula King Fahd 0 %, tengah Mas'a 0 %, aula samping 7 %, tepi mataf dari serambi
    ~30 % (siluet tampil 100 % saat tertutup). Posisi sulit (menempel eskalator, dinding dalam): kamera 10-13 m di dekat
    plafon, tidak tertutup. GPU 61 fps, 96-134 rb segitiga, 86-120 draw call. Tawaf 309 rb / 90, Sa'i 226 rb / 86, tanpa error.
  - Catatan untuk tahap berikut: sesaat di puncak eskalator (±0,6 dtk) pandangan agak dekat dari atas karena dinding
    tepat di belakang; serambi Utsmani rendah tidak dibuat (fasad mataf setinggi aula) supaya kamera sederhana; tiang
    tidak menghalangi kamera (siluet yang menangani).
- 2026-11-05 **Revisi Tahap 2 dari masukan Barru** (screenshot di browser asli):
  - Tiang aula terlalu banyak & seperti lidi -> 63 tiang persegi besar (1,6 m, jarak ±22 m), dan **lorong lurus bebas tiang
    dari ujung tiap eskalator ke Ka'bah** (7 m kiri-kanan) supaya jamaah tidak bingung arah. Lengkung aula & Mas'a dihapus;
    tiang Mas'a juga persegi (29, jarak 12 m). Tiang fasad mataf lebih tebal & jarang (71, jarak 4,6 m).
  - Kubah + pucuk lancip di tepi atap dihapus (aslinya rooftop datar): diganti pagar rendah di tepi atap. Hiasan bulat
    lancip emas di pilar portal juga dihapus.
  - Motif lengkung runcing di semua pintu (dan bingkai iwan King) dihapus; bukaan dipertinggi: pintu biasa 6,5 m, utama 9 m,
    King 10,4 m (sebelumnya lengkung yang padat menyisakan ±2,8 m = setinggi kepala).
  - BUG yang ketemu: potongan lengkung di tekstur tidak pernah berlubang (pemotong `destination-out` memakai sisa warna noise
    yang hampir transparan, alpha tersisa 249/255). Diperbaiki (`fillStyle = '#000'`). Pelajaran: cek visual dengan GPU asli
    (`--enable-gpu`, lihat `js/gpusheet.js` di scratchpad), jangan hanya swiftshader, dan baca piksel tekstur bila ragu.
  - Uji ulang: 0 error, 0 tembus dinding / 0 jatuh (13.780 frame), kepala tertutup: tepi mataf 25 %, Mas'a pinggir 23-29 %,
    aula samping 6 %, masuk pintu & tengah Mas'a 0 % (siluet tampil saat tertutup). GPU 61 fps, 65-102 rb segitiga, <= 110 draw call.
- 2026-11-05 **Revisi 2 Tahap 2** (masukan Barru): tiang dikurangi 50 % lagi (fasad mataf 71 -> 35, jarak 9,2 m; aula
  63 -> 35, jarak 31 m; Mas'a 29 -> 12, jarak 24 m). Kelap-kelip di Pintu Ajyad = z-fighting (dua bidang berimpit):
  ujung dinding bertekstur jendela tepat di sisi pilar portal. Diperbaiki: celah dinding pintu berportal dilebarkan
  setengah pilar (ujung dinding tersembunyi di dalam pilar, ambang diganti balok portal), pagar/sisi bordes mulai setelah
  dinding, bordes naik 6 cm (ambang), pagar eskalator luar dihapus (menumpuk di pagar bordes), daun pintu/plakat 10 cm di
  depan dinding, gedung OSM yang tumpang tindih dibedakan sedikit ukuran/tingginya per kelas warna, alas Safa/Marwah &
  zona hijau dinaikkan. Pemeriksa otomatis `js/zfight.js` (scratchpad): pasangan segitiga berimpit beda warna 1.634 -> 1
  (dasar Ka'bah, tak terlihat). Siluet: juga cek dua garis sejajar ±0,6 m (sudut dinding). Uji: 0 error, 0 tembus/jatuh,
  siluet 100 % saat tertutup, GPU 61 fps.
- 2026-11-05 **Tahap 3 selesai** (Opus 5.5), belum di-commit/deploy:
  - `haram-crowd.js` (baru): 140 jamaah berbenih (ringan 40 %): 48 tawaf (berlawanan jarum jam, r 16,5-39 m, di luar
    Hijr & Maqam, laki-laki ber-idhtiba'), 22 sa'i (lajur searah ke Marwah sisi barat / ke Safa sisi timur, berbalik di
    ujung, laki-laki lari kecil di lampu hijau), 30 keluar-masuk 12 pintu (pelataran -> ambang -> eskalator -> lorong
    bebas tiang -> tepi dalam mataf, berhenti 1-5 dtk lalu balik; tinggi dari `floorY`), 14 berjalan-jalan di pelataran
    (tujuan acak yang ruasnya bebas), 26 duduk selonjor berbaris (saf 3-6) menghadap Ka'bah di aula dekat fasad.
    Semua lintasan dicek dengan peta tabrakan (dinding, gedung, pagar eskalator, tiang, menara; ruas + dua garis ±0,55 m).
  - `chibi.js` createCrowd: dukung `y` (tinggi lantai), `pose: 'sit'`, dan `update(dt, cull)` (slot diisi rapat tiap
    frame; yang > 80 m dari pemain / di belakang kamera tidak dikirim ke GPU). Tanpa `cull` perilaku lama (tawaf/sa'i sama).
  - Pintu kecil yang eskalatornya menjorok ke lorong sa'i ditutup (13, 20, 21, 22); Bab Al Marwah tetap terbuka.
  - Draw call: sektor 6 -> 4, menara 4 -> 1 mesh, Clock Tower digabung (badan ke pembangun gedung, 4 muka jam 1 mesh).
    Catatan: karakter pemain sendiri ±41 draw call (chibi.js), jadi anggaran realistis <= 125.
  - Semua `?v=` simulasi -> 36 (chibi.js berubah; kaaba.js tetap 31).
  - Uji: 60 dtk simulasi, tiap langkah tiap orang: 0 memotong dinding/gedung/pagar, 0 di dalam tiang, 0 lompatan tinggi.
    GPU 61 fps; penuh 107-241 rb segitiga, 94-121 draw call; ringan 85-138 rb. Tawaf 309 rb / 90, Sa'i 228 rb / 86, tanpa error.
    Uji tekan pemain tetap 0 jatuh / 0 tembus.
  - Belum: jamaah tidak menghindari pemain (lewat menembus karakter); salat berjamaah/sujud belum ada (opsional nanti).
- 2026-11-05 17:5x: masukan Barru "masuk gerbang ber-eskalator kamera langsung zoom ke kepala". Diperbaiki: selama di
  lorong portal, bordes, dan eskalator (saat MASUK), kamera tetap di luar di belakang karakter memandang lewat bukaan pintu
  (sudut boleh turun, kamera >= pelataran + 1,5 m); pindah (cut) ke dalam baru setelah karakter sampai bawah eskalator.
  Film uji King Abdul Aziz: tidak ada lagi kepala memenuhi layar. Belum diuji ulang: camtest/occl penuh (besok).

- 2026-11-05: lantai dalam masjid (aula, mataf, Mas'a, bordes & tangga) diganti putih (sedikit dingin supaya tidak tampak kuning di bawah cahaya matahari hangat); pelataran luar tetap. Berkas haram ?v=37.
- 2026-11-05 ~18:00: **DEPLOY ke produksi atas permintaan Barru** (Jelajah dibiarkan terbuka untuk publik, sebelum Tahap 4-5). Backup VPS /tmp/umrah-code-before-jelajah.tgz. Live dicek: /, tawaf, sai, haram 200; haram/tawaf/sai 61 fps tanpa error. Belum di-commit ke git.
- 2026-11-06 **Lantai bertingkat versi 2** (koreksi Barru: hanya gerbang King yang eskalatornya TURUN; pintu lain eskalator NAIK ke rooftop): pelataran & lantai masjid sama-sama y = 0. Gerbang King (3; yang ke-4 belum ada di OSM) berdiri di **teras** setinggi 5 m (poligon yang menempel garis dinding + tangga lebar 16 anak tangga), lalu bordes & eskalator turun seperti sebelumnya. Pintu lain masuk rata. **10 eskalator rooftop** (35°, ±23 m) tegak lurus dari pelataran ke dinding di samping pintu utama/biasa (Umrah 40, Fateh 30, 22, 2, 90, 74, 4 di perluasan King Abdullah); parapet atap terbuka selebar eskalator. **Rooftop bisa dijelajahi**: lapisan tabrakan kedua `colR` (parapet luar, pagar tepi mataf, portal, menara), `floorY(x, z, layer)`, pemain pindah lapisan di eskalator; kamera rooftop = udara terbuka dengan penghalang `colR` + yang menjulang di atas atap. Titik 'Lompat ke' rooftop (tepi mataf selatan/utara/barat), zona 'Atap'. Fasad 3 deret jendela dari lantai 0. Uji: lantai dasar 13.780 frame 0 jatuh/0 tembus; rooftop 3.660 frame tanpa jatuh/keluar pagar; 10 eskalator naik-turun benar; kerumunan 0 tembus; z-fighting 1 (dasar Ka'bah); GPU 61 fps <= 246 rb segitiga / <= 126 draw call.
- 2026-11-06: lantai bertingkat v2 DEPLOY ke produksi (backup /tmp/umrah-code-before-lantai2.tgz). Live: /, tawaf, sai, haram 200; 61 fps tanpa error. Berikutnya: Tahap 4.
- 2026-11-06 **Tahap 4 selesai & DEPLOY** (backup /tmp/umrah-code-before-tahap4.tgz):
  - Kartu tempat saat mendekat (yang terdekat relatif jangkauannya): Hajar Aswad (TAKBIR), Rukun Yamani (SAPU_JAGAD),
    Multazam, Maqam Ibrahim (MAQAM_AYAT), Hijr Ismail, Zamzam (dispenser), Safa (AYAT), Marwah (DZIKIR), lampu hijau
    (RABBIGHFIR). Teks doa di `doa.js` DISALIN APA ADANYA dari tawaf.js/sai.js (tidak ada doa baru).
  - Di mataf: tombol **Mulai Tawaf**; di Safa: **Mulai Sa'i** -> simpan posisi (`sessionStorage.ukHaramSpot`) -> buka
    `?bagian=tawaf|sai&g=..&from=haram`. common.js: `sim.query` (dibaca sebelum URL dirapikan), tanda `ukFromHaram`
    (bertahan Tawaf -> Sa'i), tombol "‹ Jelajah" di baris kedua kiri atas, `sim.backToHaram(done)`. Kartu akhir Tawaf
    & Sa'i: tombol **Kembali ke Jelajah** (`back=tawaf-done` -> 4 m di belakang Maqam; `back=sai-done` -> Marwah;
    tanpa -done -> titik semula).
  - Zona warna pintu (info Barru): Hijau = King Abdul Aziz (selatan), Cokelat = King Fahd (barat), Marun = Umrah
    (barat laut), Biru = King Abdullah (utara); pintu lain ikut jangkar terdekat (< 43°), sisi timur/Mas'a tanpa zona.
    Tampil di titik pintu mini-peta & kartu pintu.
  - Koordinat Google dari Barru: King Fahd sama persis dengan OSM; King Abdul Aziz & King Abdullah berbeda ±95 m
    (titik tengah bangunan, bukan pintu) -> tetap pakai titik pintu OSM. Gerbang King = 3 (dikonfirmasi Barru).
  - Murottal: Jelajah & Tawaf memakai berkas yang sama -> posisi lanjut otomatis (mekanisme lama).
  - Uji e2e di HP 390x844 (lokal & live): Jelajah -> Mulai Tawaf -> 7 putaran -> Kembali (4 m di belakang Maqam) ->
    Safa -> Mulai Sa'i -> 7 lintasan -> Kembali (Marwah) -> tombol Jelajah di tengah tawaf -> titik semula. 0 error.
    Tawaf 305 rb / Sa'i 228 rb, 61 fps.

- 2026-11-06 Review UI Barru (DEPLOY, backup /tmp/umrah-code-before-ui-ringkas.tgz): mini-peta 132 -> 80 px; peta besar dibungkus #haramMapWrap dengan tombol '✕ Tutup' & petunjuk di dalam peta; panel keterangan Jelajah -60 % (HP: panggung 83 %, panel ±143 px; desktop: kartu melayang 340 px kanan bawah), kartu ringkas (`sim.info({compact})`, tombol Mulai di baris judul). Tawaf/Sa'i tetap ukuran lama. ?v=40.
- 2026-11-06 **Karakter pria berjenggot** (BELUM deploy, menunggu penilaian Barru): preset `LOOKS.beard` di chibi.js (rambut ikal bervolume `addWavyHair`, jenggot+kumis `addBeard` mengikuti permukaan kepala, alis tebal, senyum tertutup, kulit #d8a173, sabuk putih di izar, tanpa bintik). Pilihan ke-3 di pemilih karakter (data-look=beard); kode ?g=mb (S.gq) ikut Jelajah -> Tawaf -> Sa'i. Pratinjau: scripts/sim-illus/?s=karakter|wajah. Semua ?v= -> 41 (chibi ikut).
- 2026-11-06 **Karakter & kain (DEPLOY, backup /tmp/umrah-code-before-karakter.tgz)**: pria polos disembunyikan (kelas `hidden` di tombol; ?g=m & kode tetap), pilihan = Laki-laki (preset beard, tanpa kumis; `look.mustache` opsional) / Perempuan. Kain mengikuti kaki (`createChibi.drape`): izar/gamis tidak digabung batchRigid (`userData.keep`), profil lathe dirapatkan (`densify`), tiap frame titik kain di bawah pinggang didorong KELUAR radial sampai di luar kapsul paha / bola dengkul / kapsul betis / kapsul telapak (3 ketinggian per titik, hitung di ruang badan karena skala z mesh). Uji raycast 'kaki di luar kain' (scratchpad js/poke.js): pria jalan 3,6% -> 0,1% (terburuk 9,7% -> 0,7%), lari 8,2% -> 0,9%; wanita jalan 7,4% -> 1,7%, lari 9,1% -> 1,8%. ?v=42. Pelajaran: dorong 'menjauh dari titik kaki' bisa menarik kain KE DALAM; dorong radial keluar saja.
- 2026-11-06 **Jendela tinggi & eskalator rooftop di dalam aula (DEPLOY, backup /tmp/umrah-code-before-eskalator-dalam.tgz)**: tekstur fasad = satu jendela lengkung tinggi per panel (1,6-14,2 m, palang tiap 2,6 m) menggantikan 3 tingkat. Eskalator naik dipindah dari pelataran ke DALAM aula, sejajar dinding di samping pintu utama (bukan King): ujung bawah ±6 m dari pintu, bordes atas menjauhi pintu, menembus atap lewat **sumur** (`WELL_L` 8 m: lubang di tutup atap+plafon, pagar di rooftop, sisi lubang dari roofB.sides). Menempel dinding (celah 0,2 m) bila tidak ada pintu lain di bentang 27 m; kalau ada pintu samping, digeser ke dalam dengan lorong 2,4 m (`r.tight`; stats '*'). Hasil 10 eskalator (dulu 14). Dinding luar utuh lagi (lowSpans dihapus). `upWalk(x,z,layer)`: di rooftop hanya bagian di dalam sumur yang dianggap permukaan eskalator (atap di atas sisa tanjakan = lantai atap); lapisan pemain berganti di 1 m bawah atap (naik & turun). Kamera saat naik (`ride`, kamera di sisi bawah tanjakan): sudut per jarak supaya kamera tepat di batas plafon (ROOF_Y-0,8), mendekat sampai garis pandang lewat sumur; uji baru di blockedAt: tembus lempeng atap di luar sumur, kamera di dalam lempeng, kamera di dalam blok eskalator. Saat turun (kamera di sisi atas): aturan udara terbuka. X-ray memakai tinggi garis pandang sebenarnya. Tiang aula menjauh 3 m dari eskalator + lorong bebas tiang 18 m di depan ujung bawah; jamaah duduk menjauh. Uji: rooftest (10 naik-turun OK, atap bersih), ride (film real-time, x-ray tidak salah nyala), e2e, crowd, zfight (1 pasangan gelap lama), 61 fps (aula dekat eskalator 179k tris/107 call). ?v=43. Skrip: scratchpad js/inside.js, js/ride.js.
- 2026-11-06 **Bangunan putih gading (DEPLOY, backup /tmp/umrah-code-before-putih.tgz)**: warna krem kekuningan (hue 25-60°, saturasi >=12%, terang 60-93%) di haram-kit/haram-world/tawaf/sai dikurangi saturasinya ke 30% dan dinaikkan terangnya +5,5% (skrip scratchpad whiten.py; emas/lampu/kayu/ornamen plafon Mas'a & gedung sekitar dikecualikan). Noise tekstur & bingkai lengkung netral. createSim (common.js): pantulan tanah hemisphere #cfcdc8 (dulu #cbbf9f), matahari #fffbf3 (dulu #fff4df), material stone/stoneDark/pillar/arch/dome putih gading (chibiKit untuk ilustrasi tetap warna asal). ?v=44.
- 2026-11-06 **Kotak pelataran -60% (DEPLOY, backup /tmp/umrah-code-before-kotak.tgz)**: di haram.js setelah memuat data, bangunan OSM kecil (bukan jembatan, h < 14 m: 91 kotak 1,2-3 m + 39 bangunan 9 m) dikurangi 130 -> 52: semua yang < 25 m dari dinding masjid dihapus, sisanya dijarangkan merata menurut sudut. Data JSON tidak diubah (mudah dikembalikan). ?v=45.
- 2026-11-06 **Half Million 3D (DEPLOY, backup /tmp/umrah-code-before-halfmillion.tgz)**: `buildHalfMillion` di haram-kit.js. Segi delapan memanjang 12x13 m, menghadap pusat masjid (l.dir; posisi OSM digeser 2 m kiri + 10 m depan agar tangga/teras bebas hotel, dihitung dari poligon gedung). Lantai dasar batu 0-4,6 (undakan 3 anak tangga, pintu kaca berbingkai, panel kaligrafi kanvas Arab), pita batu 4,6-6,2 (logo MCDC kanvas transparan), lantai 2 kaca gelap 6,2-10,2 (tekstur kusen berulang, tiang sudut) dengan tanda '½M' putih MeshBasic menyala, mahkota 10,2-11. Tangga kiri-kanan (20 anak tangga pejal, pagar miring 2 muka) dari z=4,5 ke z=-8,5 naik 6,2 m ke teras belakang (blok pejal, pagar). BISA DINAIKI: floorAt(x,z) lokal -> lmFloors di floorY; col: poligon badan, segmen pagar tangga & teras (h 7,6). landmarkAt untuk kerumunan pelataran & inBld (r 17). Titik lompat 18 m di depan menghadap bangunan. Pelajaran: panel muka +z harus p->q dari -x ke +x dengan nSign -1, kalau tidak tulisan tercermin. Uji: naik kanan 6,2 -> teras -> turun kiri 0; dari belakang tertahan pagar; crowd bersih; 61 fps. ?v=46.
- 2026-11-06 **Spot 'ORDER HERE' + tumbler ½M (DEPLOY, backup /tmp/umrah-code-before-tumbler.tgz)**: lingkaran pink berdenyut (CircleGeometry + kanvas, r 1,4 m) 9,5 m di depan undakan Half Million (`world.orderSpot`, label '☕ Order here' disembunyikan saat dekat: `hideNear`). haram.js `updateOrder`: berdiri di spot -> `makeTumbler()` (silinder pink + tutup + label '½M', ±0,42 satuan rig, posisi (0,-0.3,0.19) di `rig.wrists[0]`) + toast; disimpan sessionStorage `ukHalfM` (kembali setelah muat ulang). Gestur baru 'cup' (chibi.js `CUP_POSE`, bisa ditimpa `api.cupPose`; common.js meneruskan gestureKind 'cup'): saat diam di spot, atau diam >1,2 dtk di mana saja sambil membawa tumbler, karakter menghadap kamera (`sim.face(view.yaw)`) dan mengangkat tumbler di depan dada. Pelajaran: pose dengan lengan bawah ke belakang menyembunyikan tumbler di balik telapak; label DOM bisa menutupi objek kecil. Uji: order.js (spot -> tumbler+pose, tetap saat pergi, muat ulang), e2e bersih. ?v=47.
- 2026-11-06 **Revisi spot (DEPLOY, backup /tmp/umrah-code-before-tumbler2.tgz)**: lingkaran ORDER HERE hitam-putih; tumbler burgundy (#6b1f2a, label '½M' #e9cfd3); tumbler HANYA ada selama pemain di area spot (keluar -> dilepas), tanpa sessionStorage & tanpa pose diam di tempat lain. ?v=48.
- 2026-11-06 **Tumbler tegak + lingkaran memudar (DEPLOY, backup /tmp/umrah-code-before-tumbler3.tgz)**: pivot grup tumbler di titik genggam (0,18 dari dasar), posisi di telapak (0,-0.085,0.15); tiap frame quaternion dunia dipaksa tegak & menghadap arah karakter (`cup.quaternion = inv(parentWorldQuat) * rotY(facing)`). Lingkaran ORDER HERE memudar (opacity -> 0) saat pemain di spot, kembali saat pergi. CUP_POSE az -1.0 supaya tumbler tidak tertutup jenggot. ?v=49.

## 2026-11-10 — Halaman affiliate /perlengkapan + pop-up beranda (di luar open world)
- `config/perlengkapan.json`: 17 produk (pouch HP kalung paling atas, `featured`). Isi `url` dengan link affiliate Shopee; kosong → tombol ke pencarian Shopee (`search`). Dimuat ulang saat SIGHUP/restart.
- `app/services/perlengkapan.go`, `app/handlers/perlengkapan.go`, route `GET /perlengkapan`, template `perlengkapan.html`; ilustrasi SVG di `web/static/perlengkapan/` (generator: scratchpad perlengkapan_svg.py).
- Beacon `click_shopee` (pkg = key produk) di analytics.
- Pop-up di beranda (`home.html`): muncul 2,5 dtk, sekali per pengunjung (`localStorage uk_pl_pop_v1`), HP = banner bawah, desktop = kartu pojok kanan bawah, tombol X.
- Link tambahan: menu akun "Perlengkapan umrah", banner di /persiapan. Deploy: backup /tmp/umrah-code-before-perlengkapan.tgz.
