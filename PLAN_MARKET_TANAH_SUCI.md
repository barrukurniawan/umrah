# Rencana: Market Tanah Suci ala GoFood (demo `startup.umrohku.web.id`)

Tujuan: jamaah yang sedang di Madinah atau Makkah bisa:
1. melihat toko oleh-oleh terdekat,
2. memilih barang (kurma, cokelat, parfum, abaya, sajadah, tasbih, kuliner Indonesia),
3. memesan untuk **diambil di toko**,
4. membayar lewat **QRIS dinamis** yang bisa dipindai aplikasi apa pun (DANA, ShopeePay, GoPay, OVO, LinkAja, m-banking),
5. lalu berjalan ke toko dengan **petunjuk arah Google Maps**.

Hanya aktif saat `DEMO_MODE=1`. Semua toko, harga, dan lokasi adalah **contoh**.

> **Pembaruan 24 Nov 2026 (malam):**
> - Beranda subdomain (`/`) kini **langsung market ini**, ala GrabFood versi ambil di toko. Pencarian paket umroh pindah ke `/paket`, dan `/market` diarahkan ke `/`.
> - **Web tidak memindai QR.**
>   - Tombol QR di tengah navigasi bawah dan ikon "Bayar di toko" dihapus. `/bayar` diarahkan ke beranda.
>   - Web hanya **menerbitkan QR tagihan**, seperti marketplace. Jamaah membayar dari aplikasi lain dengan memindai layar atau mengunduh QR (**Unduh QR**, PNG) lalu membukanya dari galeri.
> - Tidak ada saldo atau Top Up. Kartu "sumber dana/saldo" di Profil juga dihapus.
> - Navigasi bawah: **Beranda · Paket umroh · Riwayat · Profil**.

---

## 1. Prinsip (selaras proposal)

- **Tidak ada saldo dan tidak ada Top Up di umrohku.** Bar saldo ala GoPay sengaja tidak dipakai. Posisinya diisi **bar lokasi & kurs**: kota aktif, patokan masjid, kurs contoh 1 SAR ≈ Rp 4.300, pesanan aktif, dan bantuan.
- **Pembayaran memakai QRIS dinamis dari mitra PJP berizin BI (contoh).**
  - Dana tidak lewat rekening umrohku.
  - Jamaah membayar dari aplikasi miliknya sendiri.
  - Di demo, QR dibuat dalam format QRIS asli: TLV, CRC, nominal, nomor tagihan. Merchant dan penyelenggaranya fiktif, jadi aplikasi sungguhan akan menolaknya dan tidak ada uang yang berpindah.
- **Toko terkurasi.**
  - Nama toko diberi label "(contoh)".
  - Toko sungguhan dari Instagram (mis. toko oleh-oleh di sekitar Nabawi) **tidak dipakai** sebelum ada kerja sama tertulis.
- **Ikon baris pertama khusus KSA:** Bayar di toko, Paket oleh-oleh, PO gelombang, Kuliner RI.
- **Fitur Indonesia di baris kedua:** Cari paket, Kalender, Manasik 3D, Checklist. Dengan begitu Market tetap terlihat sebagai marketplace jamaah di Saudi.

## 2. Alur

| Langkah | Halaman | Isi |
|---|---|---|
| 1 | `/market` | Header hijau, pencarian, pilihan kota **Madinah / Makkah**, bar lokasi & kurs, 4 ikon KSA + 4 ikon persiapan, chip kategori, kartu "Terdekat dari Masjid Nabawi/Haram" (geser), daftar semua toko, produk paling dicari |
| 2 | `/market/toko/:slug` | Banner, rating, jam buka, jarak, **peta Google Maps + tombol Petunjuk arah** (rute jalan kaki), menu per kategori dengan tombol Tambah / − n +, bar keranjang melayang |
| 3 | `POST /market/toko/:slug/checkout` | Ringkasan, waktu ambil (secepatnya ±20 mnt / 1 jam lagi / setelah Isya), nama pengambil, catatan |
| 4 | `POST /market/pesan` | Pesanan dibuat: status `menunggu_bayar`, **QRIS dinamis** berlaku 15 menit |
| 5 | `/market/pesanan/:code` | **QR besar + nominal Rp & SAR + hitung mundur.** Pilihan bayar: pindai dari aplikasi lain, bayar dengan menu QRIS umrohku, atau tombol demo "simulasikan dibayar dari DANA/ShopeePay/…". Status diperbarui otomatis tiap 3 detik |
| 6 | (sama) | Setelah dibayar: **kode ambil 6 angka + QR kode ambil**, linimasa Dibayar → Disiapkan → Siap diambil → Diambil, tombol **Petunjuk arah ke toko** |
| 7 | `/toko/pesanan` (peran Toko) | Kasir melihat pesanan masuk, menekan "Siap diambil", lalu "Serahkan" dengan mencocokkan kode ambil |

### Lokasi
- Kota dipilih lewat `?kota=madinah|makkah` (disimpan di cookie). Bawaannya Madinah.
- Tombol **"Pakai lokasi saya"** meminta izin lokasi di browser. Tidak ada permintaan izin otomatis saat halaman dibuka.
  - Kalau posisi dekat salah satu kota, kota itu dipilih otomatis.
  - Jarak tiap toko lalu dihitung dari posisi jamaah dan daftar diurutkan ulang.
- Tanpa izin lokasi, jarak dihitung dari pusat Masjid Nabawi atau Masjidil Haram.
  - ≤ 2 km ditampilkan sebagai "menit jalan kaki".
  - Lebih jauh dari itu ditampilkan sebagai "menit naik mobil".

### Keterkaitan dengan QRIS umrohku
- QR pesanan menyimpan kode pesanan di tag 62 (bill number & reference).
- Kalau QR itu dibayar lewat menu QRIS umrohku (scan dari HP lain atau tombol "Bayar dengan QRIS umrohku"), pesanan **otomatis berstatus dibayar**.
- Struk QRIS juga menautkan kembali ke pesanan.
- Pesanan yang sudah dibayar masuk ke **Riwayat** (filter Belanja).

## 3. Data contoh
- **Madinah:** 6 toko.
  - Kurma Markaziyah, Oleh-oleh Quba
  - Galeri Sajadah Bab As-Salam, Butik Abaya Thaybah
  - Parfum & Bakhoor Al-Anshar, Dapur Nusantara
- **Makkah:** 5 toko.
  - Kurma Ajyad, Oleh-oleh Misfalah
  - Butik Abaya Jabal Omar, Parfum Ibrahim Khalil
  - Warung Nusantara Syisyah
- **Harga** dalam SAR, dengan perkiraan rupiah memakai kurs contoh.
- **Koordinat** hanya perkiraan di sekitar masjid (lokasi ilustrasi).
- **Gambar produk** berupa ilustrasi vektor per kategori, tanpa foto toko sungguhan.

## 4. Jalan ke produksi
1. PKS dengan toko lokal lewat badan usaha Saudi (lihat proposal KSA), serta verifikasi CR & lokasi.
2. QRIS dinamis diterbitkan **PJP/acquirer berizin**, lalu settlement ke toko dalam SAR. Untuk jamaah, rute pembayarannya QRIS lintas negara Indonesia–Saudi begitu tersedia, atau gateway berizin SAMA.
3. Katalog dan stok dikelola toko lewat portal toko. Foto produk dari toko sendiri.
4. Lokasi toko memakai koordinat terverifikasi. Petunjuk arah tetap lewat Google Maps (tanpa API berbayar).
5. Notifikasi "pesanan siap" dikirim lewat WhatsApp/push.

## 5. Kriteria selesai demo
- [x] Market rapi di 360/390 px dan desktop, tanpa scroll horizontal.
- [x] Ganti kota Madinah ↔ Makkah; jarak & urutan toko berubah; filter kategori & pencarian.
- [x] Toko: peta Google tampil, Petunjuk arah membuka Google Maps (jalan kaki).
- [x] Pesan sajadah → QR dinamis; terbaca pemindai `/bayar` (merchant & Rp 258.000 benar, CRC valid).
- [x] Bayar via QRIS umrohku → pesanan otomatis dibayar; struk QRIS menautkan ke pesanan; tampil di Riwayat (Belanja).
- [x] Tombol simulasi DANA/ShopeePay/GoPay/OVO → tab lain berubah status sendiri (polling 3 detik).
- [x] Peran Toko: pesanan masuk → Siap → Serahkan; kode ambil salah ditolak, kode benar diterima.
- [x] QR kedaluwarsa 15 menit → tidak bisa dibayar → Buat QR baru (tes `go test ./app/demo`).
- [ ] Uji di HP sungguhan: izin lokasi ("Pakai lokasi saya") hanya jalan di HTTPS.

## 6. File
- `app/demo/market.go` (+ `market_test.go`): kota, kategori, menu toko, jarak, pesanan, QRIS dinamis, kait ke `PayQR`.
- `app/demo/shop.go`: data toko diperluas (koordinat, rating, kategori, MCC).
- `app/handlers/demo_market.go`: rute `/market*` & `/toko/pesanan`.
- `web/templates/demo/market*.html`: beranda, toko, cek pesanan, pesanan + potongan status, daftar pesanan, ilustrasi kategori.
