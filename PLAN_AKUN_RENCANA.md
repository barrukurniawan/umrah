# Rencana: Akun, Tabungan & Kalender Umroh ("Umrohku Saya")

Tujuan: Umrohku bukan cuma tempat membandingkan paket, tapi teman perjalanan dari **niat → nabung → berangkat**. Pengguna bisa menyimpan paket incaran, mencatat tabungan sampai target tercapai, menyusun rencana, dan memilih tanggal berangkat dengan melihat cuaca, musim, dan kalender Hijriah.

## 1. Prinsip

1. **Login opsional.** Semua fitur yang ada sekarang (cari, bandingkan, WA travel) tetap bisa dipakai tanpa akun. Tidak ada pop-up yang memaksa login.
2. **Satu tombol di kanan atas:** "Masuk" (ikon Google). Di HP cukup ikon orang. Teks "Ikhtiar untuk kepulangan" pindah ke bawah logo atau ke hero.
3. **Ajakan login muncul hanya saat dibutuhkan**, misalnya:
   - Klik ♡ di kartu paket → "Simpan paket ini? Masuk dengan Google supaya tersimpan di semua perangkat."
   - Buka menu Tabungan, Rencana, atau Kalender pribadi.
4. **Coba dulu tanpa login.** Paket yang di-♡ sebelum login disimpan sementara di browser. Begitu login, semuanya otomatis pindah ke akun, jadi tidak ada yang hilang.

## 2. Login dengan Google

- Alur: OAuth 2.0 / OpenID Connect di server Go, memakai `golang.org/x/oauth2` dan verifikasi `id_token`. Scope hanya `openid email profile`; tidak ada akses ke Drive, Gmail, atau kontak.
- Klik "Masuk" → halaman izin Google → kembali ke halaman semula (`?next=`) dalam keadaan login.
- Sesi: cookie HttpOnly + Secure, 30 hari, disimpan di tabel `sessions`. Ada tombol Keluar.
- Setelah login, menu avatar di kanan atas berisi:
  - **Tersimpan**
  - **Tabungan**
  - **Rencana**
  - **Kalender**
  - **Akun** (unduh data, hapus akun)
  - **Keluar**
- Login pertama: satu layar persetujuan singkat ("Kami menyimpan nama, email, dan data yang Anda isi"), link ke kebijakan privasi, lalu tombol Lanjut.
- Yang perlu kamu siapkan: OAuth Client ID baru di Google Cloud Console. Boleh di project yang sama dengan MarryMe, tapi sebaiknya client terpisah dengan redirect `https://umrohku.web.id/auth/google/callback`. Rahasianya disimpan di `.env` server, tidak pernah lewat chat.

## 3. Fitur

### 3a. ♡ Simpan paket & bandingkan

- Ikon ♡ di setiap kartu paket. Halaman **Tersimpan** berisi daftar paket incaran.
- **Bandingkan 2–3 paket** berdampingan: harga, durasi, maskapai, hotel dan jaraknya, DP, tanggal.
- **Badge "harga turun" / "jadwal habis"**: setiap kali crawler jalan, paket yang disimpan dicocokkan lagi.
- Penting secara teknis: ID paket berubah tiap restart, karena database dibangun ulang dari hasil crawl. Jadi paket disimpan dengan **kunci stabil** (travel + nama paket + tanggal berangkat) plus **salinan data saat disimpan**. Paket yang hilang dari travel tetap terlihat dengan label "sudah tidak tersedia".

### 3b. Tabungan Umroh 💰

- **Buat tujuan**: "Umroh bareng Ibu", target nominal (bisa diisi otomatis dari harga paket tersimpan × jumlah orang + uang saku), dan target tanggal.
- **Catat setoran**: nominal, tanggal, catatan ("gaji Oktober"), opsional sumbernya (tabungan bank, emas, arisan). Bisa diedit atau dihapus. Hanya pencatatan; tidak terhubung ke bank dan tidak ada uang yang dipegang Umrohku.
- **Progres**:
  - bar persen
  - sisa kekurangan
  - "perlu menabung **Rp X/bulan** supaya tercapai tanggal ...", dihitung dari sisa ÷ bulan tersisa
  - perkiraan tanggal tercapai berdasarkan rata-rata setoran
- **Penanda penting**: "✓ DP Rp 5 jt sudah cukup, bisa daftar sekarang" (diambil dari DP paket pilihan) dan tonggak 25/50/75/100%.
- **Tujuan tercapai 🎉**: layar selamat dengan konfetti, lalu tombol "Hubungi travel via WA" untuk paket pilihan. Tujuannya pindah ke arsip "Tercapai".
- Beberapa tujuan sekaligus boleh, misalnya untuk diri sendiri dan untuk orang tua.

### 3c. Rencana perjalanan 🧭

- Dari paket tersimpan → **"Jadikan rencana"**: tanggal berangkat, jumlah jamaah, dan hitung mundur.
- **Rincian biaya**:
  - harga paket
  - tambahan kamar (Triple/Double)
  - perlengkapan
  - uang saku
  - oleh-oleh

  Totalnya bisa langsung dijadikan target Tabungan.
- **Checklist persiapan** dengan tenggat otomatis dihitung mundur dari tanggal berangkat. Isinya bisa ditambah sendiri.
  - paspor (nama minimal 2 kata, masa berlaku ≥ 7 bulan)
  - vaksin meningitis dan polio
  - pelunasan
  - manasik
  - perlengkapan
  - dokumen mahram
- **Bagikan ke keluarga**: link hanya-baca, misalnya untuk menunjukkan rencana ke orang tua. Bisa dimatikan kapan saja.

### 3d. Kalender Umroh 📅 (bagian yang paling "wow")

Tampilan bulanan; tiap tanggal menampilkan:

| Info | Sumber |
|---|---|
| Tanggal Masehi + **Hijriah** | Kalender Umm al-Qura, dihitung lokal |
| **Suhu Makkah & Madinah** (min–maks °C) | ≤ 16 hari ke depan: **prakiraan** cuaca; lebih jauh: **rata-rata 10 tahun** untuk tanggal itu (diberi label jelas) |
| **Musim**, diwarnai sebagai tema kalender | 🔥 Panas ekstrem (Jun–Agu, 40–45°C) · 🌤 Hangat · 🍃 Sejuk/nyaman (Des–Feb, 18–30°C Makkah) · 🌧 Peluang hujan (Nov–Jan) |
| **Momen penting** | Ramadhan, 10 malam terakhir, Idul Fitri, **musim haji** (visa umroh biasanya ditutup), Maulid, libur sekolah dan cuti bersama Indonesia (biasanya harga naik) |
| **Paket tersedia** | Titik di tanggal yang punya keberangkatan; klik untuk melihat paketnya |
| **Rencana saya** | Tanggal berangkat, tenggat checklist, target tabungan |

- Klik tanggal → panel detail berisi:
  - suhu siang/malam di Makkah dan Madinah
  - peluang hujan
  - jam matahari terbit/terbenam
  - perkiraan keramaian
  - paket yang berangkat sekitar tanggal itu
  - tombol "Simpan ke rencana"
- **Tema**: warna sel mengikuti musim (oranye = panas, hijau = nyaman, biru = sejuk), plus pilihan tampilan Terang, Gelap, atau "Malam Madinah" (hijau tua + emas). Kalau yang kamu maksud "tema" itu hal lain, kabari.
- **Kalender bisa dibuka tanpa login.** Kalender umum (cuaca, musim, Hijriah, paket) adalah magnet trafik yang bagus untuk dibagikan di Threads. Data pribadi hanya tampil kalau login.

**Sumber cuaca:** Open-Meteo (tanpa API key). Prakiraan di-cache di server 3 jam. Rata-rata iklim dihitung **sekali** dari data historis 10 tahun lalu disimpan lokal (Makkah & Madinah × 366 hari), jadi tidak bergantung pada layanan luar setiap halaman dibuka. ⚠️ Open-Meteo gratis untuk non-komersial. Karena Umrohku punya partner promo, sebaiknya pakai paket berbayar mereka (murah), **atau** hanya pakai data iklim historis yang dihitung sekali (lisensi CC BY 4.0, cukup mencantumkan sumber) tanpa prakiraan harian. Perlu kamu putuskan.

## 4. Data & keamanan

- Database terpisah `data/users.db` (SQLite), supaya tidak ikut terhapus saat data paket dibangun ulang. Tabel:
  - `users`
  - `sessions`
  - `saved_packages` (kunci stabil + salinan data)
  - `savings_goals`
  - `savings_entries`
  - `plans`
  - `plan_tasks`
  - `share_links`
- Ikut **backup harian terenkripsi** ke GitHub seperti data lain.
- **UU PDP:** ini data pribadi sungguhan (nama, email, catatan keuangan). Yang disiapkan:
  - kebijakan privasi diperbarui
  - persetujuan saat login pertama
  - **Unduh data saya** (JSON)
  - **Hapus akun** (semua data terhapus permanen)
  - hanya kamu yang bisa melihat data pengguna lewat database, dan admin tidak menampilkan isi tabungan siapa pun
- Analitik tetap anonim. Akun **tidak** dihubungkan dengan cookie statistik.
- Admin hanya mendapat angka agregat: jumlah pengguna terdaftar, berapa yang membuat tabungan, dan paket yang paling sering disimpan. Angka terakhir ini menarik untuk ditunjukkan ke travel partner.

## 5. Tahapan & estimasi

| Fase | Isi | Estimasi |
|---|---|---|
| 1 | Login Google (tombol kanan atas, menu avatar), ♡ simpan paket + halaman Tersimpan, perpindahan data dari browser ke akun, persetujuan, hapus akun, kebijakan privasi diperbarui, backup | 1,5 hari |
| 2 | Tabungan: tujuan, setoran, progres, "Rp X/bulan", penanda DP, perayaan tercapai | 1 hari |
| 3 | Kalender: Hijriah, musim + tema warna, suhu (rata-rata iklim; prakiraan jika disetujui), momen penting, paket per tanggal | 2 hari |
| 4 | Rencana: dari paket tersimpan, rincian biaya ke target tabungan, checklist bertenggat, link berbagi | 1 hari |
| 5 (nanti) | Bandingkan paket, badge harga turun, pengingat email/WA (tenggat checklist, setoran bulanan) | 1 hari |

Total ±6,5 hari kerja. Setiap fase di branch sendiri, dites di lokal, lalu deploy setelah kamu setujui.

## 6. Yang perlu kamu putuskan

1. **OAuth Google:** buat client baru (saya pandu langkahnya) di project Google Cloud MarryMe atau project baru?
2. **Cuaca:** pakai prakiraan harian (berbayar untuk komersial), atau cukup rata-rata iklim historis (gratis)?
3. **"Tema" kalender:** apakah cukup warna musim + Terang/Gelap/"Malam Madinah", atau ada maksud lain (misalnya tema per bulan Hijriah atau foto Makkah/Madinah)?
4. **Urutan:** saran saya Fase 1 → 2 → 3 → 4. Tabungan paling cepat terasa manfaatnya, dan Kalender paling bagus untuk konten Threads.
