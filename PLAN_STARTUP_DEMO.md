# Rencana: Demo Startup `startup.umrohku.web.id`

Tujuan: ada satu situs demo yang **bisa diklik dari ujung ke ujung** untuk juri, investor, dan calon travel mitra (misalnya program kurasi PUAS di Saudi). Isinya umrohku.web.id ditambah semua fitur yang dijanjikan di proposal (Fase 1–4). Semua data bisnisnya dummy, dan tidak ada uang sungguhan yang bergerak.

Produksi `umrohku.web.id` **tidak berubah**. Fitur demo baru pindah ke produksi satu per satu kalau sudah siap dan legalnya beres.

---

## 1. Arsitektur: satu kode, dua instance

Kodenya tidak di-fork. Repo dan binary tetap satu. Di VPS ada **instance kedua** dengan konfigurasi berbeda.

| | Produksi | Demo |
|---|---|---|
| Domain | umrohku.web.id | startup.umrohku.web.id |
| Folder VPS | /var/www/umrah | /var/www/umrah-startup |
| systemd | `umrah` (port 3000) | `umrah-startup` (port 3001) |
| Env | – | `DEMO_MODE=1`, `APP_PORT=3001` |
| Database | data/*.db | data/*.db **milik folder demo sendiri** (users, analytics, plus `demo.db` baru) |
| Data paket | hasil crawl harian | **salinan** `output/all_*.json` produksi (disalin otomatis tiap malam) |
| Indeks Google | ya | **tidak** (`X-Robots-Tag: noindex` + robots.txt Disallow) |

Alasan memilih dua instance, bukan satu proses dengan routing per host:
- Data dummy tidak mungkin bocor ke produksi, karena file database-nya berbeda.
- Kalau demo crash atau di-reset, produksi tidak terpengaruh.
- Deploy cukup dengan rsync yang sama ke dua folder. Build demo dulu, produksi belakangan.

### Kode
- `app/demo/`: flag `demo.Enabled()`, seeder data dummy, reset harian, pengalih peran.
- Semua route Fase 1–4 didaftarkan di `main.go` dalam blok `if demo.Enabled() { ... }`. Fitur yang nanti naik ke produksi cukup dipindah keluar dari blok itu.
- Pembayaran memakai **interface** `payments.Gateway` dengan implementasi `MockGateway` (VA/QRIS palsu, tombol "Simulasikan pembayaran berhasil"). Nanti tinggal ditambah `XenditGateway`/`MidtransGateway` tanpa mengubah alur.
- Package baru: `app/portal` (travel), `app/booking`, `app/payments`, `app/leads`, `app/financing`, `app/marketplace` (oleh-oleh dan voucher).

### Ciri khas mode demo (wajib)
1. **Pita "MODE DEMO · data contoh, tidak ada transaksi sungguhan"** di bagian atas setiap halaman.
2. **Pengalih peran** di pojok kanan bawah: *Jamaah · Travel (Amanah Demo) · Admin umrohku · Bank/Pembiayaan*. Satu klik langsung masuk tanpa Google login dan tanpa password, supaya alur demo lancar di depan juri.
3. **Tombol "Reset demo"** (di menu admin) dan reset otomatis setiap malam pukul 03.00 WIB dari seed yang deterministik.
4. Nomor WA, email, dan notifikasi **tidak dikirim keluar**. Semuanya tampil di "Kotak masuk demo" (simulasi WA/email) supaya juri tetap bisa melihat pesannya.
5. Opsional: Basic Auth atau kode akses `?kode=PUAS2026` kalau demo hanya untuk orang tertentu.

### Data dummy: perhatian soal nama travel asli
Badge "PPIU Terverifikasi", laporan lead, tagihan, dan booking **tidak boleh** memakai nomor izin palsu atas nama travel asli (Hamdan, Rabbani, dan lain-lain), karena bisa menyesatkan dan merugikan travel tersebut. Dua cara aman:
- **Travel fiktif untuk alur bisnis**: misalnya "Amanah Demo Tour", "Baitullah Contoh Travel", "Safa Sampel Wisata" dengan logo generik. Merekalah yang punya portal, lead, booking, dan tagihan.
- **Paket travel asli** tetap tampil di pencarian (salinan data publik). Badge PPIU untuk travel asli hanya dipasang kalau nomornya **sungguh dicek** di data resmi Kemenhaj/Siskopatuh. Selain itu badge-nya "Belum diverifikasi".

---

## 2. Rencana per tahap

Ukuran: S = ≤1 hari, M = 2–3 hari, L = ±1 minggu (perkiraan kasar, untuk satu developer dibantu Claude).

### Tahap 0 · Fondasi demo (M)
- [ ] DNS: A record `startup` → 43.133.150.138 (**kamu yang tambah** di panel domain).
- [ ] nginx server block + sertifikat certbot, systemd `umrah-startup`, folder `/var/www/umrah-startup`.
- [ ] Cron malam: salin `output/all_*.json` produksi ke folder demo, lalu SIGHUP. Setelah itu reset seed demo.
- [ ] `app/demo`: flag, pita demo, noindex, pengalih peran, reset, kotak masuk demo.
- [ ] Seeder: 3 travel fiktif + ±20 paket, 50 jamaah dummy, ±500 lead 60 hari terakhir (mengikuti pola klik asli dari analitik), ±30 booking dalam berbagai status, tagihan 2 bulan.
- [ ] Halaman `/demo`: "Tur demo" berisi daftar alur yang bisa dicoba, dengan tombol "Mulai sebagai Jamaah / Travel / Admin".

### Tahap 1 · Fase 1: pembanding terverifikasi (L)
| # | Fitur | Isi demo | Ukuran |
|---|---|---|---|
| 7 | **Tombol "Mau cicil?"** | Di kartu paket. Modal kecil berisi pilihan *Cicil sebelum berangkat / Pembiayaan syariah / Cuma lihat-lihat* dan tenor 6/12/24 bln. Tercatat sebagai event `intent_cicil`, lalu grafiknya tampil di admin. | S · **juga layak naik ke produksi** |
| 1 | **Badge PPIU Terverifikasi** | Tabel `travel_licenses(no_izin, tgl_berlaku, status, dicek_at, sumber)`. Badge hijau di kartu dengan tooltip nomor izin dan tanggal cek. Admin punya tombol "Cek ulang" (demo: data dari tabel lokal). | M |
| 2 | **Portal travel** | `/portal`: login per akun travel (demo: lewat pengalih peran; produksi nanti email + password/magic link). Dashboard berisi klik, lead, dan konversi per paket. | L |
| 6 | **Kelola paket & seat** (menggantikan scraping) | Di portal: CRUD paket, tanggal keberangkatan, sisa seat, dan harga. Paket dari portal ditandai "Data langsung dari travel" dan diprioritaskan di atas hasil scraping. | (bagian dari #2) |
| 3 | **Lead terverifikasi + biaya per lead** | Setiap klik WA/website atau form "Minta dihubungi" menjadi baris `leads` (sumber, paket, waktu, status: baru/dihubungi/closing/tidak valid). Lead yang dobel atau dari bot otomatis tidak ditagih. Halaman **Tagihan** bulanan berisi jumlah lead valid × tarif dan PDF invoice. | M |
| 4 | **Promo Spesial berbayar** | Tabel `placements(travel, paket, rentang budget, mulai–selesai, harga)` diatur dari admin/portal. **Kode `promoTravels` (Alviandra) di recommendation.go pindah ke tabel ini.** Label "Promo Spesial" dan "Bersponsor" tetap tampil jujur. | M |
| 5 | **Langganan portal** | Paket Gratis / Pro / Premium (fitur dan harga per bulan). Halaman "Langganan" di portal berisi status, perpanjangan (Mock checkout), dan riwayat invoice. | S |

### Tahap 2 · Fase 2: booking, DP, cicilan (L+)
> Rincian lanjutan (DP ringan, simulasi cicilan, cashback, undian umroh berhadiah): lihat **[PLAN_CICILAN_PROMO_UNDIAN.md](PLAN_CICILAN_PROMO_UNDIAN.md)**. Isi bisnisnya ada di [docs/proposal/umrohku-proposal.html](docs/proposal/umrohku-proposal.html).

| # | Fitur | Isi demo |
|---|---|---|
| 8 | **Booking + DP** | Tombol "Pesan & bayar DP" di kartu paket (hanya untuk travel mitra). Form jamaah berisi nama sesuai paspor, jumlah orang, tipe kamar, dan pilihan DP (mulai Rp 500 rb). Halaman bayar menampilkan **VA/QRIS atas nama rekening penampungan travel** (teks jelas: "Dana masuk ke rekening PT Amanah Demo Tour, bukan ke umrohku"). Ada tombol "Simulasikan pembayaran". |
| 9 | **Konfirmasi otomatis** | Setelah dibayar, status booking berubah dan e-kuitansi dibuat. Pesan WA dan email muncul di Kotak masuk demo, baik di sisi jamaah maupun travel. Riwayat ada di `/akun/booking`. |
| 10 | **Voucher umrohku** | Kode voucher (misalnya `UMROHKU500`) memotong DP. Hanya berlaku lewat checkout umrohku. Admin bisa membuat voucher, mengatur kuota, dan melihat pemakaiannya. |
| 11 | **Komisi booking** | Setiap booking lunas DP mencatat komisi (% atau nominal per travel), terpisah dari DP. Muncul di Tagihan travel bersama biaya lead dan langganan. |
| 12 | **Cicil sebelum berangkat** | Fitur Tabungan yang sekarang ditambah "Hubungkan ke Tabungan Umroh Bank Syariah Mitra (demo)": nomor rekening virtual, autodebet bulanan, dan saldo yang "tersinkron". Tanpa margin, tanpa utang. Saat saldo ≥ DP, tombol "Gunakan untuk DP paket ini" muncul. |
| 13 | **Rujukan pembiayaan syariah** | Form pengajuan dengan **centang persetujuan berbagi data (UU PDP)** yang terpisah dan bisa dicabut. Pengajuan diteruskan ke 2–3 "mitra pembiayaan" fiktif. Peran *Bank/Pembiayaan* bisa menyetujui atau menolak. Status terlihat di akun jamaah, dan fee rujukan tercatat saat dana cair. |
| – | **Ulasan terverifikasi** | Hanya jamaah dengan booking berstatus "Sudah pulang" yang bisa memberi ulasan, dengan badge "Berangkat via umrohku". Seed berisi beberapa ulasan contoh yang ditandai *contoh*. |

### Tahap 3 · Fase 3: umrah mandiri (M, prototipe)
- [ ] `/mandiri`: panduan langkah (visa/Nusuk → tiket → hotel → transport → muthawwif) dengan checklist yang tersimpan di akun.
- [ ] Pemesanan berupa **prototipe yang bisa diklik** (katalog hotel/transport/muthawwif dummy, keranjang, checkout mock).
- [ ] Pita khusus: "Menunggu aturan turunan UU 14/2025 dan putusan MK. Belum dibuka untuk umum."

### Tahap 4 · Fase 4: layanan di Tanah Suci (M, prototipe)
- [ ] **Oleh-oleh dikirim ke rumah**: katalog kurma, air zamzam (kemasan resmi), sajadah, dan lain-lain dari pemasok dummy di Indonesia. Keranjang → QRIS mock → lacak pengiriman.
- [ ] **Voucher prabayar Tanah Suci**: beli dalam rupiah, lalu dapat kode QR. Ada peran "Merchant Makkah" yang memindai atau memasukkan kode untuk menukar. Saldo dan riwayat terlihat.
- [ ] **Pickup di toko Makkah/Madinah**: pilih toko mitra dan jadwal ambil, lalu status "Siap diambil" (pin di peta memakai data geometri Haram yang sudah ada).

### Tahap 5 · Halaman legal & korporat (S)
- [ ] `/syarat-ketentuan`: draf S&K, termasuk peran umrohku sebagai platform (bukan penyelenggara), dana tidak dipegang umrohku, kebijakan refund mengikuti travel, dan voucher. **Perlu ditinjau ahli hukum sebelum naik ke produksi.** Draf ini juga layak naik ke produksi.
- [ ] `/tentang`: badan usaha (PT, NIB, nomor PSE Komdigi) dengan status "dalam proses". Isian ditampilkan kosong atau "segera", **bukan nomor karangan**.
- [ ] `/mitra`: formulir pendaftaran travel mitra (nama PT, no. izin PPIU, PIC, kontak). Di demo, data masuk ke admin.
- [ ] `/investor`: ringkasan roadmap Fase 1–4 dengan link "Coba demonya" ke tiap alur. Halaman ini bisa jadi pembuka saat presentasi.

---

## 3. Alur presentasi (±7 menit)

1. **Jamaah**: cari paket Rp 25 jt → badge PPIU dan "Promo Spesial" → klik "Mau cicil?" → pesan, bayar DP dengan QRIS mock, pakai voucher → konfirmasi muncul di Kotak masuk demo.
2. **Travel (Amanah Demo)**: dashboard lead dan booking masuk secara real time → ubah sisa seat → lihat tagihan (lead + komisi + langganan).
3. **Admin umrohku**: atur posisi Promo Spesial, verifikasi PPIU, lihat pendapatan per sumber (CPL, komisi, langganan, fee rujukan).
4. **Pembiayaan**: setujui pengajuan → status jamaah berubah.
5. **Visi**: halaman Mandiri dan Tanah Suci (prototipe) → simulasi 3D Masjidil Haram yang sudah ada sebagai pembeda produk.

---

## 4. Yang bisa naik ke produksi lebih dulu
Fitur-fitur ini tidak bergantung pada izin atau payment gateway. Setelah teruji di demo, tinggal dipindahkan keluar dari blok `demo.Enabled()`:
- #7 "Mau cicil?" (pengukur minat)
- #4 Promo dari tabel (menghapus hardcode Alviandra)
- #3 laporan lead dari data klik yang sudah ada
- Halaman S&K dan formulir travel mitra

Booking/DP, rekening penampungan, pembiayaan, dan voucher Tanah Suci **tetap demo** sampai PT, PSE, perjanjian dengan travel, dan mitra bank/payment gateway beres.

## 5. Yang perlu dari kamu
1. Tambah DNS `startup.umrohku.web.id` → 43.133.150.138.
2. Izin SSH ke VPS untuk menyiapkan instance kedua (nginx, certbot, systemd).
3. Nama 3 travel fiktif dan 2–3 mitra pembiayaan/bank fiktif, atau pakai usulan di atas.
4. Tarif contoh untuk ditampilkan: biaya per lead, % komisi booking, harga langganan Pro/Premium.
5. Demo terbuka untuk umum (noindex saja) atau pakai kode akses?

## 6. Urutan kerja yang disarankan
Tahap 0 → #7 → #1 → #2/#6 → #3 → #4 → #5 → Tahap 5 (S&K, mitra, investor) → Tahap 2 → Tahap 3–4.
Setelah Tahap 0 + Tahap 1 + Tahap 5 (±2 minggu), demo sudah cukup untuk pengajuan **Gelombang 1 (batas 25 Nov 2026)**. Fase 2–4 menyusul sebelum pengumuman.

---

## 7. Status (23 Nov 2026) · branch `startup-demo`

Semua tahap di atas sudah bisa diklik di mode demo (`DEMO_MODE=1`), **live di https://startup.umrohku.web.id sejak 23 Nov 2026**. Cara deploy subdomain ada di [docs/DEPLOY_STARTUP_DEMO.md](docs/DEPLOY_STARTUP_DEMO.md).

| Bagian | Route utama | Kode |
|---|---|---|
| Fondasi: pita demo, pengalih peran, Kotak masuk, reset 03.00 WIB, noindex | `/demo`, `/inbox` | `app/demo/{demo,roles,inbox,seed}.go` |
| #1 PPIU, #6 data dari travel | kartu paket, `/travel/:slug`, `/admin/bisnis/mitra` | `app/demo/travels.go` |
| #2 portal, #3 lead + tagihan, #5 langganan | `/portal/*` | `app/handlers/demo_portal.go`, `app/demo/{leads,billing}.go` |
| #4 Promo Spesial dari tabel | `/admin/bisnis/promo` | `services.SetFeaturedSource` |
| #7 Mau cicil? + simulasi | lembar di kartu paket, `/admin/bisnis/cicil` | `app/handlers/demo_jamaah.go` |
| #8–#11 booking, DP, konfirmasi, promo, komisi | `/booking/*` | `app/demo/{booking,promo}.go`, `app/payments` |
| #12 tabungan ke bank syariah mitra | `/tabungan/:id` | `SavingsLink` |
| #13 rujukan pembiayaan + UU PDP | `/pembiayaan/ajukan`, `/akun/pembiayaan`, `/mitra-pembiayaan` | `app/demo/financing.go` |
| Undian yang bisa diaudit | `/undian`, `/admin/bisnis/undian` | `app/draws` (+ tes), `app/demo/draw.go` |
| Ulasan terverifikasi | `/booking/:id/ulasan`, `/travel/:slug#ulasan` | `Review` |
| Fase 3 mandiri, Fase 4 Tanah Suci, toko mitra | `/mandiri`, `/tanah-suci`, `/toko` | `app/demo/shop.go`, `app/handlers/demo_tanahsuci.go` |
| S&K, tentang, daftar mitra, roadmap | `/syarat-ketentuan`, `/tentang`, `/mitra`, `/investor` | `app/handlers/demo_public.go` |

Produksi tetap sama (sudah diuji tanpa `DEMO_MODE`):
- semua route demo 404,
- tidak ada pita demo,
- `output/all_zz_demo_portal.json` diabaikan,
- Promo Spesial Alviandra tetap jalan dari map lama.
