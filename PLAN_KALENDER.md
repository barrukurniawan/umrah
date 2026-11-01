# Rencana: Mode Kalender

Status: rencana + mockup. Belum ada kode di aplikasi.
Mockup interaktif (data paket asli 6 bulan, Nov 2026 – Apr 2027): buka `docs/mockup-kalender.html` di browser.

## 1. Tujuan

Sekarang orang mencari umroh dari budget. Banyak juga yang mulai dari tanggal. Misalnya "saya cuti 20 Desember" atau "mau berangkat waktu libur sekolah".
Mode Kalender menjawab tiga pertanyaan sekaligus dalam satu layar.

1. Tanggal berapa saja ada keberangkatan, dan berapa travel.
2. Cuacanya seperti apa di tanggal itu.
3. Paket apa saja yang berangkat, kalau tanggalnya diklik.

## 2. Tampilan

### 2a. Tombol geser Budget / Kalender

- Ada di atas kartu pencarian, di tengah. Bentuknya pil dua sisi: **💰 Budget** dan **📅 Kalender**.
- Default tetap **Budget**, supaya pengguna lama tidak bingung.
- Pilihan terakhir diingat di browser. Link bisa dibagikan dengan `?mode=kalender&bulan=2026-12`.

### 2b. Kalender bulanan (satu halaman penuh)

Kepala kalender:
- Tombol ‹ › untuk pindah bulan, plus strip **6 bulan ke depan** ("Nov 26 · 83 jadwal", "Des 26 · 202 jadwal", …) supaya bisa loncat langsung dan langsung kelihatan bulan mana yang ramai.
- Nama bulan Masehi besar, di bawahnya bulan Hijriah ("Jumadilakhir – Rajab 1448 H").
- Filter **budget** (Semua, ≤ 25 jt, ≤ 30 jt, …) dan **travel**. Angka di kalender langsung ikut berubah.
- Ringkasan satu baris. Contoh dari data asli: "202 keberangkatan dari 11 travel di 31 tanggal. Mulai 24,5 jt."

Setiap kotak tanggal berisi:

| Bagian | Isi |
|---|---|
| Kiri atas | Tanggal Masehi, tebal. Abu-abu bila tidak ada keberangkatan |
| Kanan atas | Tanggal Hijriah kecil ("21 Rajab") |
| Tengah | Ikon cuaca + suhu siang/malam Makkah ("☀️ 36°/24°") |
| Bawah | Pil hijau **"5 travel"** + "mulai 23,5 jt" |
| Garis tipis di dasar kotak | Momen. Ungu = libur sekolah, hijau = Ramadhan, kuning = 10 malam terakhir, biru = Idul Fitri. Musim haji diarsir abu dengan tulisan "Visa umroh biasanya ditutup" |
| Latar | Gradasi tipis sesuai cuaca (lihat 2d) |

- Hari yang sudah lewat dibuat pudar. Hari ini diberi cincin hijau.
- Kotak yang punya keberangkatan bisa diklik dan sedikit terangkat saat disentuh kursor.

### 2c. Pop-up detail tanggal

Klik tanggal → pop-up di tengah (desktop) atau bottom sheet dari bawah (HP).

1. **Judul.** "Rabu, 30 Desember 2026", di bawahnya "21 Rajab 1448 H". Di atasnya label kecil "8 travel · 13 paket berangkat".
2. **Kartu cuaca.** Suhu siang/malam Makkah dan Madinah, musim, keramaian, peluang hujan, momen (misalnya "Libur sekolah akhir semester").
3. **Daftar paket dikelompokkan per travel**, travel termurah di atas. Tiap paket menampilkan nama, harga, durasi + tanggal pulang, maskapai (+ Direct), hotel + bintang, sisa seat.
4. Tombol **Lihat paket** (membuka kartu lengkap seperti di mode Budget) dan **♡ Simpan**.

Tutup dengan ×, klik area gelap, atau tombol Esc.

### 2d. Tema cuaca

Warna latar mengikuti suhu siang Makkah hari itu. Sengaja tipis supaya angka tetap mudah dibaca.

| Suhu siang Makkah | Tema | Ikon | Kira-kira bulan |
|---|---|---|---|
| < 33° | Sejuk (biru muda) | ❄️ | Des–Feb |
| 33–37° | Nyaman (hijau muda) | 🌤️ | Mar, Nov |
| 38–41° | Hangat (kuning muda) | ☀️ | Apr, Okt |
| 42–44° | Panas (oranye muda) | 🔥 | Mei, Agu–Sep |
| ≥ 45° | Sangat panas (merah muda) | 🥵 | Jun–Jul |

Legenda kecil ada di atas grid.

### 2e. HP

- Grid tetap 7 kolom. Isi kotak diringkas jadi tanggal + bulatan hijau berisi jumlah travel.
- Suhu dan harga tampil di pop-up saja, supaya kotak tidak sesak.
- Pop-up menjadi bottom sheet setinggi 88% layar.

## 3. Data

### 3a. Keberangkatan

Sumber sama dengan mode Budget (`detail_packages`), dengan aturan yang sama:
- tanggal yang sudah lewat tidak dihitung,
- tanggal yang dinyatakan penuh oleh travel tidak dihitung,
- "jumlah travel" = jumlah travel berbeda di tanggal itu, bukan jumlah paket,
- "mulai" = harga termurah di tanggal itu (harga per tanggal bila ada).

Data dihitung di server per bulan, sekali per permintaan. Ukurannya kecil (± 200 baris per bulan), jadi tidak perlu cache.

### 3b. Suhu harian

- Pakai tabel iklim bulanan yang sudah ada di `app/services/insight.go`.
- Suhu per hari = interpolasi garis lurus antar pertengahan bulan (tanggal 15). Hasilnya halus dari hari ke hari, tidak loncat di pergantian bulan.
- Selalu diberi label **"rata-rata iklim, bukan prakiraan"**.
- Tanpa layanan luar dan tanpa biaya.

Opsional di tahap berikutnya: prakiraan nyata untuk 16 hari ke depan dari Open-Meteo. Gratisnya hanya untuk non-komersial. Karena ada partner promo, pilihannya bayar paket mereka (murah) atau tetap pakai rata-rata iklim saja.

### 3c. Tanggal Hijriah

Dihitung di browser dengan `Intl.DateTimeFormat('id-u-ca-islamic-umalqura')`. Ini bawaan semua browser modern (Chrome, Safari, Firefox), jadi tidak perlu library atau tabel di server. Diberi catatan "perkiraan Umm al-Qura (±1 hari)".

### 3d. Momen

Pakai daftar `events` di `insight.go` (Ramadhan, 10 malam terakhir, musim haji, libur sekolah).
- Musim haji ditampilkan beda. Kotak diberi arsir abu dan tulisan "visa umroh biasanya ditutup".
- Daftar ini perlu ditambah setahun sekali (perkiraan tanggal Hijriah + kalender pendidikan).

## 4. Teknis

| Bagian | Rencana |
|---|---|
| Rute | `GET /kalender?bulan=2026-12&budget=&travels=` → potongan HTML grid (HTMX, sama seperti `/recommendations`) |
| | `GET /kalender/hari?tanggal=2026-12-30&budget=&travels=` → potongan HTML pop-up |
| Service | `services.CalendarMonth(month, budget, travels)` dan `services.CalendarDay(date, …)`. Memakai ulang filter yang sama dengan `GetRecommendations` supaya hasilnya konsisten |
| Template | `calendar.html` (grid) dan `calendar_day.html` (pop-up). Pop-up memakai partial kartu paket yang sudah ada |
| Tombol geser | Di `home.html`. Menukar isi kartu pencarian tanpa reload |
| Analitik | Event baru `cal_open`, `cal_month`, `cal_day` ke `/t`, supaya kelihatan di admin seberapa dipakai |
| Tes | Agregasi per tanggal (penuh dan lewat tidak dihitung), interpolasi suhu, batas tema |

Perkiraan beban: satu query per bulan ke SQLite lokal. Tidak menambah infrastruktur.

## 5. Tahapan

| Tahap | Isi | Perkiraan |
|---|---|---|
| K1 | Service + rute + grid bulanan, tombol geser, filter budget/travel | ½ hari |
| K2 | Pop-up detail + "Lihat paket" + ♡ Simpan, bottom sheet di HP | ½ hari |
| K3 | Tema cuaca, Hijriah, momen (libur sekolah, Ramadhan, musim haji), legenda | ¼ hari |
| K4 | Analitik, tes, cek di HP & desktop, deploy | ¼ hari |

Total sekitar 1,5 hari kerja. Bisa dikerjakan dalam satu sesi.

## 6. Yang perlu kamu putuskan

1. **Alviandra di mode Kalender.** Di mode Budget dia hanya muncul di budget 28–29 jt. Di Kalender, apakah (a) ikut aturan yang sama, hanya muncul kalau filter budget 28–29 jt, atau (b) selalu muncul di tanggal 15 Jan 2027 dengan badge 🔥?
2. **Suhu kota mana di kotak tanggal.** Usulan saya Makkah (paling lama di sana). Madinah tampil di pop-up.
3. **Prakiraan cuaca nyata** untuk 16 hari ke depan. Pakai Open-Meteo berbayar, atau cukup rata-rata iklim.
4. **Default saat halaman dibuka.** Tetap Budget (usulan saya), atau Kalender.
