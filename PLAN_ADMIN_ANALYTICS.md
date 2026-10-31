# Rencana: Analitik + `/admin` Umrohku

Tujuan: melihat trafik umrohku.web.id (berapa yang buka, dari mana, apa yang dicari) dan **klik WA/website per travel**. Data klik ini juga bisa jadi bukti ke travel partner seperti Alviandra ("paket kalian dilihat X orang, Y orang klik WA").

Tanpa Google Analytics atau script pihak ketiga. Semua data disimpan di server sendiri, jadi ringan dan tidak perlu banner cookie.

## 1. Yang dicatat

| Event | Kapan | Data |
|---|---|---|
| `view` | halaman `/` dibuka (dicatat di server, middleware Fiber) | path, sumber (Threads/IG/WA/TikTok/Google/Langsung/…), perangkat (HP/desktop), pengunjung unik |
| `search` | klik "Cari Paket Umrah" (`POST /recommendations`) | budget, filter travel, bulan, urutan, jumlah hasil |
| `click_wa` / `click_web` | klik tombol WA / website di kartu paket | nama travel, nama paket, harga, posisi kartu (1, 2, 3, …) |
| `calc` | buka kalkulator cicilan | travel, paket |
| `chat` | kirim pesan ke chat AI (kalau nanti diaktifkan) | hanya jumlah, tanpa isi pesan |

- **Klik** dikirim lewat `navigator.sendBeacon('/t', …)` dari script kecil (±1 KB) yang ditambahkan ke template. Tidak memperlambat halaman, dan link WA tetap langsung terbuka.
- **Pengunjung unik**: hash harian dari IP + User-Agent + garam acak yang diganti tiap hari. IP asli tidak pernah disimpan.
- **Bot** (Googlebot, crawler, preview WhatsApp, dll.) disaring berdasarkan User-Agent.
- **Sumber**: logika `classifySource` disalin dari MarryMe (utm_source → referrer). Link yang kamu bagikan di Threads sebaiknya pakai `?utm_source=threads`, karena aplikasi Threads sering tidak mengirim referrer.

## 2. Penyimpanan

- File SQLite terpisah: `data/analytics.db`. Database paket dibangun ulang dari hasil crawler setiap restart, jadi analitik **tidak boleh** satu file dengan data paket.
- Satu tabel `events(id, ts, type, visitor, source, device, path, travel, package, price, position, budget, meta_json)` dengan index pada `ts` dan `type`.
- Retensi: event mentah disimpan 13 bulan. Ukurannya kecil (±100 byte/event).
- Ikut backup harian off-site (terenkripsi `age`, ke repo `marryme-backups`, folder `umrohku/`).

## 3. Halaman `/admin`

Login: satu password dari env `ADMIN_PASSWORD_HASH` (bcrypt) dengan cookie sesi HttpOnly 7 hari, dibatasi 5 percobaan per 15 menit. Tampilan meniru admin MarryMe: terang, bersih, aksen hijau Umrohku, dan nyaman dibuka di HP.

Menu:
1. **Ringkasan**: pengunjung unik dan pencarian untuk hari ini / 7 hari / 30 hari, grafik harian, perbandingan dengan periode sebelumnya.
2. **Sumber trafik**: Threads, Instagram, WhatsApp, TikTok, Google, Langsung, Lainnya (bar + persen), plus HP vs desktop.
3. **Travel & paket**: per travel tampil (jumlah tampil di hasil, klik WA, klik website, CTR), juga 10 paket yang paling banyak diklik WA. Ada filter tanggal, cocok untuk laporan ke partner (misalnya Alviandra).
4. **Pencarian**: sebaran budget yang dicari (20–25 jt, 25–30 jt, …), travel yang paling sering dipilih di filter, bulan keberangkatan favorit, dan pencarian yang hasilnya 0 (tanda ada paket yang kurang).
5. **Live**: kunjungan dalam 30 menit terakhir.

Tidak ada edit data di admin (hanya baca). Pengelolaan partner promo (Alviandra) tetap lewat file/kode seperti sekarang. Itu bisa jadi tahap berikutnya.

## 4. Implementasi (Go/Fiber, tanpa dependensi berat)

- `app/analytics/` (baru): store SQLite, middleware view, handler beacon `/t`, deteksi bot dan sumber, agregasi query.
- `app/handlers/admin.go` (baru) + `web/templates/admin/*.html`: halaman server-rendered memakai Tailwind dan HTMX yang sudah ada; grafik pakai SVG sederhana tanpa library.
- Ubah `main.go` (route `/admin`, `/t`, middleware) dan `recommendations.html` (atribut `data-track` di tombol WA/website/kalkulator + script beacon).
- Nomor urut kartu (posisi) disertakan supaya terlihat apakah posisi teratas (featured) memang mendapat klik lebih banyak.

## 5. Langkah & verifikasi

1. Kerjakan di branch baru `admin_analytics` (dari `new/design`). Branch ini sekaligus akan memuat commit perubahan crawler dan promo yang sekarang belum di-commit.
2. Tes lokal: buka halaman, cari, klik WA, lalu cek angka di `/admin` dan pastikan bot tidak terhitung.
3. Deploy ke VPS setelah kamu setujui: build, restart `umrah.service`, tambah `ADMIN_PASSWORD_HASH` di `/var/www/umrah/.env`, lalu tambahkan `analytics.db` ke backup off-site.

Estimasi: ±1 hari kerja.

Catatan: data baru mulai terkumpul sejak fitur ini aktif. Trafik viral yang sudah lewat tidak bisa dihitung mundur, karena dulu tidak ada pencatatan. Log nginx di server baru hanya ada sejak 31 Okt.
