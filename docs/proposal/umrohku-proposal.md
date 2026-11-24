<!--
Dokumen: Proposal Pendanaan Pre-Seed Umrohku (versi 18 / final, 24 November 2026)
Sumber: konversi utuh dari halaman HTML https://claude.ai/artifact/6ZcYnkkyj3kiAicwVdJUPB
Isi: sampul, profil founder, bagian 01–14, lampiran draf LOI, dan sumber data.
Catatan untuk agent pembaca:
- Bagian yang semula berupa grafik, kartu, atau tabel sudah diubah menjadi daftar dan tabel Markdown.
- Rujukan [Gambar: ...](img/...) adalah tangkapan layar produk/kanal; teks alt menjelaskan isinya.
- Label `[Mengikat]` / `[Tidak mengikat]` di lampiran LOI menandai sifat hukum tiap pasal.
-->

**PROPOSAL PENDANAAN PRE-SEED · NOVEMBER 2026**

# Umrohku

**Agen umroh digital dengan checkout, cicilan syariah, dan travel terkurasi**

Agen umroh digital berskala nasional. Jamaah membandingkan paket travel PPIU berizin, membayar lewat checkout umrohku, dan dananya diteruskan ke rekening resmi travel dengan komisi terpotong otomatis.

**Rancangan alur dana (pra-PKS)**

- **Checkout di umrohku** — VA bank dulu; pembiayaan di Fase 2
- **Dana di PJP berizin BI** — Ditahan 1–3 hari kerja, bukan di umrohku
- **Cair ke rekening PPIU** — Hanya rekening penampungan resmi travel
- **Komisi terpotong otomatis** — 4–5% lewat split payment

Status November 2026: belum ada PKS dengan penyedia jasa pembayaran, lembaga pembiayaan, maupun travel. Umrohku tidak menyimpan dana jamaah.

- **Cari umroh by budget**
- **Cari umroh by kalender**
- **Cicil syariah**
- **Nabung umroh**
- **Simulasi umroh 3D**
- **Book & DP umroh**
- **Konsultasi AI personal**
- **Travel terkurasi**

**Kanal resmi**

- Live: umrohku.web.id — https://umrohku.web.id
- Pitching: startup.umrohku.web.id — https://startup.umrohku.web.id
- Instagram: @umrohku.web.id — https://www.instagram.com/umrohku.web.id
- Threads: @umrohku.web.id — https://www.threads.net/@umrohku.web.id
- TikTok: @umrohku.web.id — https://www.tiktok.com/@umrohku.web.id
- YouTube: @umrohku.official — https://www.youtube.com/@umrohku.official
- Owner: Barru Kurniawan — https://www.linkedin.com/in/barru-kurniawan/

---

**PROFIL FOUNDER**

## Barru Kurniawan

*Founder umrohku · Backend Developer*

Software Engineer dengan 9+ tahun pengalaman.

Jakarta Universitas Gadjah Mada 939 pengikut LinkedIn

### Riwayat kerja

Jun 2021 – sekarang

**Back End Developer**

Financial Technology Startup

REST API dengan Golang dan Python untuk kartu kredit & debit, QRIS lokal & cross border, split payment, dan cicilan.

Jan – Jun 2021

**Python Developer**

SigmaTech

Ditempatkan di startup fintech yang sama. REST API dengan Flask dan CherryPy.

Jun 2019 – Jan 2021

**Software Developer**

Gigel.id

Backend marketplace sewa perlengkapan bayi. REST API dengan Flask.

Agt 2018 – Jun 2019

**Software Developer**

WE+ (PT Kita Indonesia Plus)

Backend Django, serta otomasi pengujian web dan Android (Selenium, Robot Framework, Appium).

Sep 2017 – Jul 2018

**Software Engineer**

Blipcom

Fullstack web dengan CherryPy, JavaScript, dan MongoDB.

### Yang dibangun sendiri

2026 – sekarang

**Founder**

umrohku.web.id

Data 508 paket, simulasi manasik 3D, dasbor analitik, dan konten video, dikerjakan seorang diri.

2021 – 2024

**Founder**

Jago QA (PT Institusi Jago Indonesia)

E-learning Software Quality Assurance. Non-aktif secara operasional; badan usaha aktif dan tertib lapor pajak nihil.

### Nilai yang dibawa

- **Spesialisasi:** kartu kredit & debit, QRIS lokal & cross border, split payment, dan cicilan
- **Disiplin QA:** setiap alur uang diuji sebelum dipakai jamaah
- **Produksi konten sendiri,** sehingga biaya akuisisi rendah

**Komitmen full-time di umrohku begitu pendanaan Tahap 1 cair.** Rencana produk tidak memuat e-wallet, sehingga tidak bersinggungan dengan lini bisnis pemberi kerja saat ini. Co-founder bisnis dari industri umroh sedang dicari.

Keahlian: Golang, Python, RESTful API, MongoDB, Sistem pembayaran, QRIS & cross-border, Split payment, Otomasi QA

Riwayat lengkap: [linkedin.com/in/barru-kurniawan](https://www.linkedin.com/in/barru-kurniawan/)

---

**01 · RINGKASAN**

## Agen umroh digital yang membuat jamaah aman memilih travel

- **Masalah** — Jamaah menyetor puluhan juta ke travel tanpa perlindungan: dananya tidak diawasi OJK dan tidak diasuransikan. Jaminannya hanya kepercayaan pada rekam jejak travel.
- **Solusi** — Agen digital untuk banyak travel PPIU: checkout dengan banyak cara bayar, dana diteruskan ke rekening resmi travel, dan travel terkurasi.
- **Model bisnis** — Komisi agen 4–5% per jamaah lunas, terpotong otomatis saat checkout, plus fee rujukan pembiayaan.
- **Pendanaan** — Rp1,5 M dalam dua tahap berbasis milestone. Saham biasa dengan dividen, bukan SAFE.

- **508** — paket aktif, diperbarui 2× sehari
- **461** — pencarian paket dalam 30 hari
- **78** — klik ke WhatsApp & situs travel
- **±185 rb** — tayangan video di YouTube & TikTok

Analitik internal umrohku.web.id, 26 Okt–24 Nov 2026; YouTube Studio per 24 Nov 2026.

---

**02 · MASALAH**

## Uang besar, informasi minim, perlindungan nol

- **Rawan penipuan** — 244 kasus haji dan umrah, kerugian Rp247,65 miliar. Juni 2026: 85 mitra Hanania Group melapor ke Polda Metro Jaya.
- **Dana disetor penuh di depan** — Paket mulai ±Rp23,5 juta per orang, umumnya lunas berbulan-bulan sebelum berangkat.
- **Cicilan syariah terbatas** — Hanya 7 multifinance aktif membiayai umroh; jamaah harus mencari sendiri.
- **Sulit membandingkan** — Harga, hotel, dan maskapai tersebar di brosur dan WhatsApp dengan format berbeda.

---

**03 · SOLUSI & PRODUK**

## Fokus pada tiga hal: bandingkan, bayar, biayai

- **Bandingkan** Cari by budget & kalender Badge PPIU dan skor travel Simulasi manasik 3D
- **Bayar** Checkout di umrohku, mulai dari VA bank Dana diteruskan ke rekening resmi PPIU Concierge sampai visa terbit
- **Biayai** Tabungan di bank syariah atas nama jamaah (Fase 2) Atau ajukan ke beberapa mitra syariah sekaligus Dana pembiayaan cair ke travel

> **Travel tidak menunggu uang.** Jamaah yang memakai pembiayaan cukup membayar DP sekitar 20%; sisanya dicairkan mitra syariah ke travel, lalu jamaah mengangsur setelah pulang.

### Sudah live di umrohku.web.id

- [Gambar: Cari sesuai budget: geser budget Rp20–50 jt, paket langsung muncul](img/g1-budget.jpg) — **01 · Cari sesuai budget** Geser budget Rp20–50 jt, pilih satu travel atau semua.
- [Gambar: Bandingkan detail: jarak hotel, direct atau transit, DP, tanggal berangkat](img/g2-bandingkan.jpg) — **02 · Bandingkan detail** Jarak hotel ke Haram, direct atau transit, DP, dan tanggal.
- [Gambar: Kalender keberangkatan Desember 2026 dengan penanda cuaca dan libur sekolah](img/g3-kalender.jpg) — **03 · Kalender keberangkatan** Penanda cuaca nyaman atau sejuk, dan libur sekolah.
- [Gambar: Satu tanggal menampilkan semua paket, suhu Makkah dan Madinah, tanggal Hijriah](img/g4-tanggal.jpg) — **Satu tanggal, semua paket** Suhu Makkah & Madinah, tanggal Hijriah, simpan paket.
- [Gambar: Paket tersimpan dengan insight cuaca dan keramaian bulan keberangkatan](img/g5-tersimpan.jpg) — **Paket tersimpan** Login Google, insight cuaca, jadikan target tabungan.
- [Gambar: Tabungan umroh: target Rp91,5 jt untuk 3 orang termasuk uang saku](img/g6-tabungan.jpg) — **Kalkulator target tabungan** Pelacak target, bukan rekening bank. Target dihitung per orang, termasuk uang saku.
- [Gambar: Checklist persiapan 22 langkah dengan tanggal berangkat](img/g7-checklist.jpg) — **Checklist persiapan** 22 langkah, disesuaikan untuk laki-laki atau perempuan.
- [Gambar: Panduan paspor: berlaku minimal 6 bulan, contoh halaman data, tips M-Paspor](img/g8-paspor.jpg) — **Panduan paspor** Contoh halaman data dan cara buat lewat M-Paspor.
- [Gambar: Dokumen keluarga: KTP, KK, akta lahir atau buku nikah, dengan contoh tiap dokumen](img/g9-dokumen.jpg) — **Dokumen keluarga** KTP, KK, dan buku nikah, dengan contoh tiap dokumen.
Rancangan checkout, pembiayaan, dan portal travel dapat dicoba di [startup.umrohku.web.id](https://startup.umrohku.web.id/demo).

---

**04 · CHECKOUT & ALUR DANA**

## Menguasai jalur pembayaran, bukan menahan uang

> **Pelajaran dari marketplace dan OTA besar:** kekuatannya ada di checkout, verifikasi cepat, dan potongan komisi otomatis. Dana penjual hanya ditahan beberapa hari, bukan berbulan-bulan.

- **Checkout** — Fase 1: virtual account bank. Fase 2: pembiayaan syariah dan cicilan kartu.
- **Dana di PJP** — Masuk sub-rekening penyedia jasa pembayaran berizin BI, bukan rekening umrohku.
- **Verifikasi dua lapis** — DP: cair 1–3 hari kerja setelah kuitansi resmi PT dan bukti booking seat. Pelunasan: cair setelah bukti pendaftaran Siskopatuh.
- **Split otomatis** — 95–96% ke rekening penampungan PPIU, 4–5% komisi ke umrohku.
- **Ulasan terverifikasi** — Hanya jamaah yang checkout dapat menilai travel.

- **Anti rekening pribadi** — Tujuan pencairan dikunci ke rekening penampungan atas nama PT PPIU, bukan rekening agen atau perorangan.
- **Komisi tidak perlu ditagih** — Terpotong oleh sistem saat dana dicairkan. Tidak ada invoice yang bisa "terlupa".
- **Arus kas travel aman** — Dana diterima travel dalam hitungan hari, sehingga modal kerja tiket, hotel, dan visa tidak terganggu.
- **Sesuai keahlian founder** — Kartu, split payment, dan cicilan adalah pekerjaan harian founder sejak 2021.

### Skema biaya: menjaga komisi Rp1,35 jt tidak tergerus

| Cara bayar | Tarif publik gateway | Pada paket Rp30 jt | Kebijakan umrohku |
|---|---|---|---|
| **Virtual account bank** | ±Rp4.000 per transaksi | ±Rp12 rb (3 kali bayar) | Utama Ditanggung umrohku; di bawah 1% komisi |
| **Pembiayaan syariah** | Tanpa biaya gateway | +Rp360–480 rb fee rujukan | Didorong Dana cair penuh ke travel |
| **QRIS** | 0,7% | Rp210 rb bila untuk seluruh paket | Terbatas Hanya untuk DP kecil |
| **Kartu kredit** | 2,9% + Rp2.000 | ±Rp872 rb, setara 65% komisi | Bersyarat Aktif hanya jika pembagian biaya disepakati dalam PKS |
| **Cicilan kartu** | Tarif kartu + biaya cicilan bank | Belum dipastikan | Menunggu Dibuka setelah ada penawaran resmi bank |

> **Aturan margin:** biaya transaksi yang ditanggung umrohku dibatasi 10% dari komisi. Kartu kredit tidak pernah ditanggung sendiri, dan larangan surcharge kartu tetap dipatuhi.

- **Bertahap** — Fase 0 masih manual: jamaah membayar langsung ke rekening PPIU. Checkout dan split payment dibangun di Fase 1 setelah ada bukti transaksi.
- **Yang tidak kami janjikan** — Jaminan dana kembali, asuransi penggelapan, dan penahanan dana sampai hari keberangkatan. Itu butuh izin PPIU sendiri atau rekening terblokir di bank.

Tarif gateway dari daftar harga publik Xendit dan Midtrans, belum termasuk PPN, dan dapat dinegosiasikan. Biaya split payment dan cicilan kartu belum terverifikasi. Rancangan ini perlu dikonfirmasi dengan PJP mitra dan konsultan hukum.

---

**05 · CELAH PASAR**

## Travel sudah membayar agen. Umrohku menjadi agen terbesarnya.

- **Program mitra travel** — Travel rutin beriklan "ayo jadi mitra kami": bawa jamaah, dapat komisi Rp1–1,5 jt per orang.
- **Tour leader pengajian** — Bawa 10 jamaah, berangkat gratis. Setara ±Rp3 jt per jamaah, sekitar 10% harga paket.
- **Umrohku: agen digital** — Satu kanal untuk ratusan jamaah dari konten dan pencarian, dengan data dan pembiayaan.

| Kanal penjualan | Biaya travel per jamaah | Skala |
|---|---|---|
| **Agen perorangan** | Rp1–1,5 jt | Beberapa jamaah per musim |
| **Tour leader pengajian** | ±Rp3 jt (1 kursi gratis per 10 jamaah) | Satu rombongan |
| **Umrohku** | Rp1,2–1,5 jt (4–5%) | Ratusan hingga ribuan per tahun, terukur per klik |

> **Bukan biaya baru bagi travel.** Komisi 4–5% berasal dari anggaran pemasaran yang sudah ada, dialihkan ke kanal yang lebih besar dan lebih terukur. Karena itu angka ini realistis.

---

**06 · ANTI-BYPASS**

## Kenapa jamaah membayar di umrohku, bukan langsung ke travel

- **Cara bayar, bertahap** — **Fase 1:** checkout virtual account yang rapi dan tercatat. **Mulai Fase 2:** pembiayaan syariah dan cicilan kartu, yang tidak dimiliki kebanyakan travel.
- **Pendampingan concierge** — Status dokumen, visa, dan keberangkatan dipantau satu kontak. Hanya untuk jamaah yang mendaftar lewat umrohku.
- **Harga lebih untung** — Harga travel sama, ditambah potongan langsung dari komisi umrohku.
- **Ulasan terverifikasi** — Hanya jamaah yang checkout di umrohku yang dapat memberi nilai dan laporan ketepatan berangkat.

> **Pengunci transaksi per fase.** Fase 1 mengandalkan potongan harga resmi, pendampingan, dan checkout yang tercatat. Pembiayaan syariah dan cicilan kartu menjadi pengunci utama mulai Fase 2. Komisi terpotong otomatis saat checkout; pada masa uji coba dihitung dari laporan jamaah berkode booking, sesuai LOI Pasal 3.

---

**07 · PEMBIAYAAN SYARIAH**

## Berangkat dulu, angsur kemudian, tanpa riba

- **Ijarah multijasa** — Lembaga menyewa jasa travel, lalu menyewakannya ke jamaah dengan ujrah tetap yang disepakati di awal.
- **Murabahah** — Lembaga membeli paket, lalu menjualnya ke jamaah dengan margin tetap. Angsuran tidak berubah sepanjang tenor.

### Simulasi: paket Rp30 jt, DP 20% Rp6 jt, dibiayai Rp24 jt

| Tenor | Bank syariah · ±0,75–0,95%/bln | Multifinance syariah · ±1,5–1,7%/bln |
|---|---|---|
| **12 bulan** | ±Rp2,19 jt/bln | ±Rp2,39 jt/bln |
| **24 bulan** | ±Rp1,19 jt/bln | ±Rp1,39 jt/bln |
| **36 bulan** | ±Rp856 rb/bln | ±Rp1,07 jt/bln |

### Peran umrohku

- Menampilkan beberapa penawaran berdampingan
- Satu pengajuan, persetujuan data UU PDP
- Dana cair langsung ke travel; risiko kredit di mitra

### Contoh produk di pasar

- BSI Mitraguna Umroh
- CIMB Niaga Syariah X-Tra Dana iB
- Amitra (FIFGROUP)

> **Risiko yang kami hitung:** pengajuan tanpa agunan banyak ditolak di pengecekan SLIK. Proyeksi hanya memakai asumsi 10% jamaah lunas lewat pembiayaan, dan angka itu diuji lebih dulu lewat tes pengajuan ke satu lembaga.

Estimasi ilustratif dari skema yang beredar, belum termasuk biaya admin dan asuransi jiwa. Contoh produk bukan mitra; kerja sama belum ada.

---

**08 · PASAR & PESAING**

## Pasar besar, pembiayaan masih kosong

- **1,6 jt** — jamaah umroh Indonesia, 2025 Siskopatuh
- **±Rp48 T** — nilai pasar per tahun asumsi paket Rp30 jt
- **Rp32,17 T** — piutang multifinance syariah OJK, Sep 2026
- **7** — multifinance aktif membiayai umroh OJK

- **Haji makin jauh** — Antrean haji reguler puluhan tahun; keluarga beralih menabung untuk umroh.
- **Regulasi bergerak** — UU 14/2025 mengakui umrah mandiri; Saudi meluncurkan Nusuk Umrah.
- **Pembiayaan belum tergarap** — OJK menilai pemain pembiayaan umroh masih sedikit dibanding potensinya.

| Pemain | Fokus | Pembeda umrohku |
|---|---|---|
| **Situs & WhatsApp travel** | Paket sendiri | Perbandingan lintas travel dan rekam jejak |
| **Marketplace umroh** (Pergiumroh, Waqara) | Katalog paket | Pembiayaan dan pendampingan yang mengunci transaksi |
| **Travel dengan cicilan** (Wipatour) | Cicilan paket sendiri | Banyak travel, banyak mitra pembiayaan |
| **Fintech umroh** (UmrahCash) | Tabungan & pembiayaan | Calon mitra di sisi pembiayaan |
| **Nusuk Umrah** (Saudi) | Visa & layanan langsung | Panduan berbahasa Indonesia lewat PPIU |

> **Dua penyakit marketplace umroh: bypass dan repeat rendah.** Umrohku menjawab bypass dengan pembiayaan dan pendampingan yang hanya ada di platform, dan menjawab repeat rendah dengan CAC rendah dari konten.

---

**09 · TRAKSI & VALIDASI**

## Minat sudah terukur. Tahap berikutnya: transaksi pertama.

- **461** — pencarian paket dalam 30 hari situs berumur 1 bulan
- **78** — klik ke travel 23 WhatsApp · 55 situs
- **±185 rb** — tayangan video organik YouTube + TikTok
- **Rp0** — biaya iklan seluruh trafik organik

- **Hitungan jujur:** klik adalah minat, bukan pembelian. Dengan asumsi 1–2% klik menjadi jamaah lunas, trafik hari ini setara 0–1 jamaah per bulan.
- **Jalur ke 25 jamaah per bulan (target Tahun 1):** ±1.700 klik per bulan, atau sekitar 20× trafik sekarang.
- **Tiga pengungkit:** konten (satu video sudah 124 rb tayangan), SEO (Google baru 1% trafik), dan pendampingan concierge untuk menaikkan konversi.

### Bukti: dasbor admin umrohku.web.id, 26 Okt–24 Nov 2026

- **847** — pengunjung unik
- **8.791** — halaman dibuka
- **461** — pencarian paket
- **23** — klik WhatsApp travel
- **55** — klik situs travel

#### Budget yang dicari

- < Rp25 jt 56 · 12%
- Rp25–29 jt 217 · 47%
- Rp30–34 jt 59 · 13%
- Rp35–39 jt 54 · 12%
- ≥ Rp40 jt 75 · 16%

Hampir separuh pencarian ada di segmen yang paling butuh cicilan.

#### Sumber pengunjung

- Langsung 593 · 83%
- Threads 50 · 7%
- Facebook 28 · 4%
- Instagram 18 · 3%
- Google 7 · 1%

"Langsung" mencakup klik dari aplikasi media sosial tanpa referrer. Angka pengunjung belum disaring dari bot dan kunjungan pengembang; log dibersihkan sebelum uji tuntas.

| Travel (5 teratas, belum bermitra) | Tampil | Klik WA | Klik situs | CTR |
|---|---|---|---|---|
| **PT Labbaika Cipta Imani** | 388 | 7 | 10 | 4,4% |
| **Umrah Bisa** | 625 | 6 | 9 | 2,4% |
| **Ventour Travel** | 597 | 3 | 4 | 1,2% |
| **Al Hijaz** | 186 | 2 | 8 | 5,4% |
| **Rabbani Tour** | 316 | 1 | 6 | 2,2% |

Lima travel ini sudah menerima klik dari umrohku, sehingga menjadi target LOI pertama. 63 pencarian (14%) tanpa hasil menunjukkan permintaan yang belum terlayani.

> **Catatan data paket.** Pada tahap uji coba MVP, 508 paket dihimpun dari publikasi terbuka untuk memvalidasi minat pencarian. Listing tanpa izin atau LOI akan dibersihkan saat masuk fase komersial.

### Mesin akuisisi: konten organik

- **±185 rb** — total tayangan video YouTube + TikTok
- **124 rb** — tayangan Short terbaik "Umrah itu berapa lama?"
- **±155 rb** — tayangan 7 YouTube Shorts 461 subscriber
- **±30 rb** — tayangan 6 video TikTok 2.390 suka

- [Gambar: Halaman Shorts @umrohku.official: 461 subscriber, tayangan 1,2 rb hingga 124 rb](img/25-shorts.jpg) — **YouTube @umrohku.official** 7 Shorts, satu menembus 124 rb tayangan.
- [Gambar: Profil TikTok @umrohku.web.id: 2.390 suka, video terbaik 20,9 rb tayangan](img/26-tiktok.jpg) — **TikTok @umrohku.web.id** Video terbaik 20,9 rb tayangan.
Seluruhnya organik, tanpa iklan, diproduksi sendiri oleh founder. Tangkapan layar 24 November 2026.

### Sudah berjalan di production

| Fitur live | Yang dikerjakan | Manfaat bisnis |
|---|---|---|
| **Pembanding paket** | 508 paket dari 18 travel, diperbarui 2× sehari | Sumber klik minat ke travel: 78 per bulan |
| **Cari by budget & kalender** | Rekomendasi sesuai dana dan bulan berangkat | Data permintaan pasar yang tidak dimiliki travel |
| **Simulasi manasik 3D** | Masjidil Haram, tawaf, dan sa'i | Menarik jamaah sejak tahap niat; pembeda dari katalog biasa |
| **Checklist & perlengkapan** | 22 langkah persiapan, 24 produk afiliasi | Alasan kembali ke situs; pendapatan afiliasi sudah aktif |
| **Akun & kalkulator tabungan** | Login Google, paket tersimpan, pelacak target tabungan (bukan rekening bank) | Basis pengguna terdaftar untuk booking dan pembiayaan |
| **Dasbor analitik per travel** | Tampil, klik WA, klik situs, dan CTR | Bukti nilai untuk negosiasi komisi dalam PKS |

- [Gambar: Kartu rekomendasi paket](img/02-rekomendasi.jpg) — `Live` **Rekomendasi paket** Harga, DP, maskapai, jarak hotel.
- [Gambar: Kalender keberangkatan](img/03-kalender.jpg) — `Live` **Kalender keberangkatan** Jadwal dan harga termurah.
- [Gambar: Simulasi 3D Masjidil Haram](img/04-simulasi-haram.jpg) — `Live` **Masjidil Haram 3D** 126 pintu dan menara.
- [Gambar: Checklist persiapan umroh](img/08-persiapan.jpg) — `Live` **Checklist persiapan** 22 langkah, tersimpan di akun.

### Prasyarat sebelum dana investor cair · 60 hari, biaya sendiri

- **5 LOI atau PKS bertanda tangan** dari direktur travel PPIU, dimulai dari lima travel di atas (draf LOI terlampir)
- **5–10 transaksi concierge lunas,** dengan bukti komisi masuk ke rekening umrohku
- **Tes pengajuan pembiayaan** ke satu lembaga syariah: berapa yang mendaftar, berapa yang lolos SLIK
- **Badan hukum siap:** KBLI 63122 ditambahkan pada PT Institusi Jago Indonesia, PSE Komdigi terdaftar, merek dicek di DJKI kelas 35, 36, 39, dan 42

Angka konversi 1–2% adalah asumsi untuk trafik dingin pada produk Rp30 jt, dan akan diganti data nyata dari transaksi concierge.

---

**10 · UNIT ECONOMICS**

## Laba kotor ±Rp1 jt per jamaah, dua jamaah per transaksi

- Komisi agen 4–5% dari paket Rp30 jt; setara komisi agen hari ini +Rp1,2–1,5 jt
- Fee rujukan pembiayaan 1,5–2% pokok; hanya 10% jamaah +±Rp40 rb
- Potongan harga untuk jamaah −Rp250 rb
- Biaya virtual account & split ±3 kali bayar per jamaah; kartu kredit tidak ditanggung −±Rp15 rb
- Laba kotor per jamaah Rp1,0–1,3 jt

- **Efek keluarga** — Rata-rata 2 jamaah per transaksi: laba kotor Rp2,0–2,6 jt.
- **CAC lewat konten** — Target di bawah Rp300 rb per transaksi; dibuktikan di Tahap 1.
- **≥6× CAC** — Dari satu transaksi, tanpa bergantung pada pembelian ulang.

### Lapisan berikutnya, belum masuk proyeksi

- **SaaS untuk PPIU · Tahun 2+** — Booking engine dan CRM berlangganan untuk travel mitra: pendapatan berulang dengan margin tinggi.
- **Rujukan modal kerja travel · Tahun 3+** — Data kinerja keberangkatan menjadi dasar bank syariah menalangi tiket dan hotel travel bereputasi baik.

Asumsi internal, divalidasi lewat transaksi concierge. Umroh jarang diulang (1× per 3–5 tahun), sehingga hitungan tidak memakai repeat order. Risiko kredit sepenuhnya di lembaga pembiayaan.

---

**11 · PROYEKSI 3 TAHUN**

## Impas di tahun kedua, dari komisi saja

| Metrik | Tahun 1 · Validasi | Tahun 2 · Tumbuh | Tahun 3 |
|---|---|---|---|
| **Jamaah lunas** | 300 (±25/bln) | 1.200 (±100/bln) | 5.000 (±415/bln) |
| **Pangsa pasar** | 0,02% | 0,08% | 0,31% |
| **GMV** | Rp9 M | Rp36 M | Rp150 M |
| **Pendapatan** (±Rp1,4 jt/jamaah) | Rp420 jt | Rp1,68 M | Rp7,0 M |
| **Biaya operasional** | Rp900 jt | Rp1,5 M | Rp5,4 M |
| EBITDA | −Rp480 jt | +Rp180 jt | +Rp1,6 M |

### Titik kontrol bulan 18

- **600 jamaah lunas kumulatif:** 300 di Tahun 1, 300 di paruh pertama Tahun 2
- **30 travel** dengan PKS
- **Paruh kedua Tahun 2 (±150/bln)** ditopang musim puncak umroh dan mulai aktifnya pembiayaan syariah

### Anggaran 18 bulan = Rp1,5 M

- Tahun 1: Rp900 jt
- Paruh pertama Tahun 2: Rp600 jt
- Sama dengan dana yang diminta; perusahaan hidup 18 bulan meski pendapatan nol

Pendapatan dicatat bruto (komisi dan fee rujukan). Potongan harga Rp250 rb per jamaah dicatat sebagai biaya promosi di dalam biaya operasional: Rp75 jt, Rp300 jt, dan Rp1,25 M per tahun. SaaS dan rujukan modal kerja belum dihitung.

---

**12 · ROADMAP**

## Buktikan dulu, baru besarkan

Fase 0 · 60 hari · biaya sendiri

### Bukti transaksi

- 5 LOI travel dan 5–10 transaksi concierge lunas
- Tes tingkat persetujuan pembiayaan
- Berjalan di bawah PT Institusi Jago Indonesia (tambah KBLI 63122), PSE, dan cek merek
- Bersihkan listing travel tanpa izin; logo diganti nama teks

Fase 1 · Bulan 0–6 · Tahap 1

### Kanal agen digital

- Pendirian PT baru; LOI, domain, dan kode dialihkan (novasi)
- PKS dengan 15 travel PPIU, kode booking, komisi per jamaah lunas
- Checkout virtual account, verifikasi dua lapis, dan split payment lewat PJP berizin BI
- Skor rekam jejak travel dan layanan concierge
- Target 100 jamaah lunas

Fase 2 · Bulan 6–18 · Tahap 2

### Pembiayaan & skala

- Kerja sama rujukan dengan 1 bank syariah dan 1 multifinance
- Penjajakan tabungan jamaah dan escrow bersama bank syariah
- 30 travel mitra, 600 jamaah lunas kumulatif

Fase 3 · Tahun 2–3

### Lapisan fintech

- SaaS booking engine untuk PPIU
- Rujukan modal kerja travel ke bank syariah
- Pendaftaran ITSK di OJK (POJK 3/2024) setelah volume memadai

Fase ekspansi · Tahun 3+

- Wallet & QRIS di Saudi lewat mitra PJP berizin BI
- Layanan Tanah Suci: voucher riyal dan oleh-oleh
- Undian umroh, setelah GMV stabil di atas Rp2 M per bulan
- Panduan umrah mandiri: afiliasi hotel dan tiket, bukan rombongan

---

**13 · LEGAL & KEPATUHAN**

## Peta izin: umrohku platform, bukan penyelenggara umroh

| Area | Dasar | Posisi umrohku | Status |
|---|---|---|---|
| **Badan usaha** | KBLI 63122, PSE Komdigi | Fase 0 memakai PT Institusi Jago Indonesia. PT baru didirikan saat Tahap 1, lalu LOI, domain, dan kode dialihkan sebagai syarat pencairan | Pra-pendanaan |
| **Penjualan paket** | UU 8/2019 jo. UU 14/2025 | PPMSE dan mitra pemasaran; PKS tertulis dengan setiap PPIU; tidak mengorganisir rombongan | Fase 1 |
| **Alur uang** | PP 38/2021 | Dana diterima dan diteruskan oleh PJP berizin BI ke rekening penampungan PPIU; komisi terpotong lewat split payment. Umrohku tidak menyimpan dana di rekening sendiri | Fase 1 |
| **Pendaftaran OJK** | POJK 3/2024 (ITSK) | Dikaji setelah volume transaksi memadai; prosesnya panjang dan butuh audit keamanan informasi | Tahun 2–3 |
| **Tabungan & cicilan** | OJK; POJK 3/2024 (ITSK) | Umrohku hanya merujuk ke lembaga berizin; tidak menghimpun dana dan tidak memberi pinjaman | Fase 2 |
| **Data pribadi** | UU 27/2022 (PDP) | Persetujuan eksplisit sebelum data dikirim ke mitra, dapat dicabut | Sudah di demo |
| **Wallet & QRIS** | PBI 23/6/PBI/2021 | Ditunda. Izin PJP kategori 1 butuh modal Rp15 M; nanti lewat mitra. Cashback diberikan sebagai potongan harga, bukan saldo | Tahun 3+ |
| **Undian** | Permensos 3/2024; PP 132/2000 | Ditunda. PPh final 25% atas nilai hadiah; anggaran dialihkan ke potongan harga pasti | Tahun 3+ |
| **Merek** | DJKI | Cek dan daftarkan "Umrohku" di kelas 35, 36, 39, 42 | Pra-pendanaan |

---

**14 · PENDANAAN**

## Rp1,5 M dalam dua tahap, dengan dividen sejak perusahaan laba

- **Fase 0 · Rp0 dari investor** — 60 hari biaya sendiri: 5 LOI, 5–10 transaksi lunas, tes pembiayaan.
- **Tahap 1 · Rp450 jt** — Cair setelah Fase 0 terbukti dan PT baru berdiri. Target 6 bulan: 15 PKS dan 100 jamaah lunas.
- **Tahap 2 · Rp1,05 M** — Cair setelah target Tahap 1 tercapai. Target bulan 18: 30 PKS dan 600 jamaah lunas.

> **Risiko investor dibatasi.** Jika Fase 0 gagal, tidak ada dana yang keluar. Jika Tahap 1 gagal, eksposur maksimal Rp450 jt.

### Struktur: bisnis agen yang sehat, bukan taruhan valuasi

### Saham biasa dengan dividen

- Saham biasa PT, valuasi pre-money Rp2,5–3,5 M, ditetapkan setelah Fase 0 terbukti
- Dividen tunai 30–50% dari laba bersih setelah pajak, mulai Tahun 2
- Harga Tahap 2 mengikuti capaian Tahap 1

### Apa yang didapat investor

- **Pengaman:** dividen Tahun 3 setara ±7–12% per tahun atas dana Rp1,5 M, dengan porsi saham ±30%
- **Imbal hasil utama:** kenaikan nilai saat lapisan SaaS dan rujukan modal kerja berjalan, atau lewat sinergi strategis

### Investor yang kami cari: angel strategis

- **Pemilik travel PPIU**, sebagai investor minoritas pasif
- **Pengusaha muslim** yang memahami bisnis umroh
- **Eksekutif perbankan syariah** dengan akses ke bank dan lembaga pembiayaan

> **Aturan netralitas.** Pemilik travel hanya dapat masuk sebagai investor minoritas pasif, maksimal 10–15%, tanpa akses ke data lead dan transaksi travel lain. Co-founder bisnis harus figur independen: eks eksekutif bank syariah, eks manajer OTA, atau eks pengurus asosiasi yang tidak lagi memiliki travel aktif.

**PENGGUNAAN DANA · 18 BULAN**

**Rp1,5 M**

- Tim inti 3 orang 50% · Rp750 jt
- Konten, promo & akuisisi 20% · Rp300 jt
- Legal, PT, PKS & merek 10% · Rp150 jt
- Teknologi & infrastruktur 8% · Rp120 jt
- Cadangan 12% · Rp180 jt

### Tahap 1 · Rp450 jt untuk 6 bulan

| Tim 3 orang (6 × Rp41,7 jt) | Rp250 jt |
|---|---|
| Konten, promo & akuisisi | Rp80 jt |
| Legal: PT baru, PKS, merek | Rp60 jt |
| Teknologi & infrastruktur | Rp30 jt |
| Cadangan | Rp30 jt |
| Total | Rp450 jt |

Anggaran tim sudah termasuk THR dan BPJS. Engineering dikerjakan founder.

### Tim inti: 3 orang

- **Barru Kurniawan** · CEO, produk & teknologi. Full-time sejak Tahap 1 cair
- **Co-founder bisnis** · kemitraan travel dan bank syariah. Sedang dicari dari industri umroh; ekuitas dan komisi, bukan sekadar gaji
- **Konten & operasional** · produksi video dan concierge jamaah

### Imbal hasil, dihitung apa adanya

- EBITDA Tahun 3 ±Rp1,6 M; laba bersih setelah pajak ±Rp1,2 M
- Dividen 30–50% dari laba bersih: ±Rp370–620 jt per tahun
- Nilai usaha Tahun 3 pada 6–8× EBITDA: ±Rp10–13 M, dari komisi saja
- SaaS dan rujukan modal kerja menjadi tambahan, belum dihitung

---

**LAMPIRAN · DRAF LOI TRAVEL**

## Dokumen yang akan ditandatangani travel

Berbahasa Indonesia, dengan pemisahan tegas antara ketentuan yang mengikat dan yang tidak. Belum ada travel yang menandatangani.

---

**SURAT MINAT KERJA SAMA (LETTER OF INTENT)**

Nomor: ________/LOI/UMROHKU/________/2026

Pada hari ini, ________, tanggal ________, bertempat di ________, yang bertanda tangan di bawah ini:

- Pihak Pertama: **PT Institusi Jago Indonesia**, pengelola platform umrohku.web.id ("Umrohku")
- Diwakili oleh: Barru Kurniawan, selaku Direktur
- Alamat: ________

- Pihak Kedua: **PT ________**, Penyelenggara Perjalanan Ibadah Umrah (PPIU)
- Nomor izin PPIU: ________
- Diwakili oleh: ________, selaku ________
- Alamat: ________

Pihak Pertama dan Pihak Kedua, bersama-sama disebut "Para Pihak", dengan ini menyatakan minat untuk menjajaki kerja sama dengan pokok-pokok sebagai berikut.

#### Pasal 1 Maksud dan Tujuan `[Tidak mengikat]`

Surat ini mencatat kesepahaman awal Para Pihak untuk bekerja sama memasarkan paket umrah Pihak Kedua melalui platform digital Pihak Pertama, sebagai dasar penyusunan Perjanjian Kerja Sama ("PKS").

#### Pasal 2 Pokok-Pokok Rencana Kerja Sama `[Tidak mengikat]`

1. Pihak Pertama menampilkan dan memasarkan paket umrah Pihak Kedua kepada calon jamaah.
2. Setiap jamaah dari Pihak Pertama dicatat dengan kode booking unik yang disepakati Para Pihak.
3. Pihak Kedua membayar imbal jasa pemasaran sebesar ________% (rencana 4–5%) dari harga paket untuk setiap jamaah yang lunas, berdasarkan tagihan Pihak Pertama.
4. Pembayaran jamaah diterima langsung di rekening penampungan Pihak Kedua. Pihak Pertama tidak menerima, menyimpan, maupun mengelola dana jamaah. Apabila kelak tersedia fitur pembayaran melalui penyedia jasa pembayaran berizin Bank Indonesia, tujuan pencairannya tetap rekening penampungan Pihak Kedua.
5. Pihak Pertama mendampingi jamaah dari pendaftaran sampai visa terbit, dan Pihak Kedua memberikan informasi status dokumen dan keberangkatan yang diperlukan.
6. Pihak Kedua tetap menjadi penyelenggara perjalanan dan bertanggung jawab penuh atas keberangkatan, visa, dan layanan jamaah sesuai peraturan perundang-undangan.

#### Pasal 3 Masa Uji Coba dan Imbal Jasa `[Mengikat]`

1. Para Pihak menjalankan uji coba selama 60 (enam puluh) hari sejak surat ini ditandatangani.
2. Untuk setiap jamaah berkode booking Pihak Pertama yang melunasi paket selama masa uji coba, Pihak Kedua wajib membayar imbal jasa pemasaran sebesar ________% dari harga paket.
3. Pihak Kedua wajib melaporkan secara tertulis setiap pelunasan jamaah berkode booking Pihak Pertama paling lambat 3 (tiga) hari kerja setelah pelunasan diterima.
4. Imbal jasa dibayar paling lambat 7 (tujuh) hari kerja setelah tagihan Pihak Pertama diterima, ke rekening atas nama Pihak Pertama.
5. Kewajiban pada angka 2 sampai dengan 4 tetap berlaku untuk jamaah yang mendaftar selama masa uji coba dan melunasi setelahnya.

#### Pasal 4 Penggunaan Data dan Merek `[Mengikat]`

Selama surat ini berlaku, Pihak Kedua mengizinkan Pihak Pertama menampilkan nama, logo, serta informasi paket dan harga Pihak Kedua di platform Pihak Pertama. Pihak Pertama wajib menampilkannya secara akurat dan menghapusnya paling lambat 7 (tujuh) hari kerja setelah diminta secara tertulis.

#### Pasal 5 Kerahasiaan `[Mengikat]`

Para Pihak wajib menjaga kerahasiaan informasi bisnis yang diterima dari pihak lainnya, termasuk data jamaah, harga khusus, dan besaran imbal jasa, selama surat ini berlaku dan 12 (dua belas) bulan setelahnya. Data pribadi jamaah diproses sesuai Undang-Undang Nomor 27 Tahun 2022 tentang Pelindungan Data Pribadi.

#### Pasal 6 Sifat Surat `[Mengikat]`

1. Kecuali Pasal 3 sampai dengan Pasal 8, surat ini merupakan pernyataan minat dan tidak menimbulkan kewajiban hukum bagi Para Pihak untuk menandatangani PKS.
2. Kerja sama ini tidak eksklusif. Masing-masing pihak bebas bekerja sama dengan pihak lain.
3. Masing-masing pihak menanggung biayanya sendiri sehubungan dengan surat ini.
4. Pihak Pertama dapat mengalihkan hak dan kewajibannya dalam surat ini kepada badan hukum baru pengelola platform Umrohku dengan persetujuan tertulis Pihak Kedua, yang tidak akan ditahan tanpa alasan yang wajar.

#### Pasal 7 Jangka Waktu `[Mengikat]`

Surat ini berlaku selama 6 (enam) bulan sejak ditandatangani dan berakhir lebih awal apabila PKS ditandatangani atau salah satu pihak mengakhirinya dengan pemberitahuan tertulis 14 (empat belas) hari sebelumnya.

#### Pasal 8 Hukum yang Berlaku dan Penyelesaian Perselisihan `[Mengikat]`

Surat ini tunduk pada hukum Republik Indonesia dan dilaksanakan dengan itikad baik. Perselisihan diselesaikan secara musyawarah untuk mufakat dalam 30 (tiga puluh) hari; apabila tidak tercapai, melalui Pengadilan Negeri ________.

Demikian surat ini dibuat dalam rangkap 2 (dua), masing-masing bermeterai cukup dan mempunyai kekuatan yang sama.

- **Pihak Pertama** PT Institusi Jago Indonesia meterai, tanda tangan & cap **Barru Kurniawan** Direktur
- **Pihak Kedua** PT ________ meterai, tanda tangan & cap **________** Direktur

Draf disusun mengikuti praktik dokumen prakontrak di Indonesia: rencana kerja sama jangka panjang dinyatakan tidak mengikat, sedangkan imbal jasa masa uji coba, kerahasiaan, jangka waktu, dan pilihan hukum dinyatakan mengikat (Pasal 1320 dan 1338 KUHPerdata). Perlu ditinjau konsultan hukum sebelum digunakan.

---

**SUMBER DATA**

- Data produk & traksi: analitik internal umrohku.web.id 26 Okt–24 Nov 2026; YouTube Studio @umrohku.official per 24 November 2026. Unit economics dan proyeksi: asumsi internal.
- [Infobank: 1,6 juta jamaah umrah 2025](https://infobanknews.com/bsi-genggam-pangsa-83-persen-jemaah-haji-indonesia)
- [Sindonews: 244 kasus haji dan umrah, kerugian Rp247,65 miliar](https://www.rctiplus.com/news/detail/sindonews/5502797/kemenhaj-catat-244-kasus-haji-dan-umrah--kerugian-mencapai-rp247-65-miliar)
- [Republika: 85 mitra Hanania Group laporkan dugaan penipuan umrah (Juni 2026)](https://news.republika.co.id/berita/tfzoa2370/85-mitra-hanania-group-laporkan-dugaan-penipuan-umrah-ke-polda-metro-jaya)
- [Kompas: umrah mandiri diperbolehkan (UU 14/2025)](https://cahaya.kompas.com/aktual/25K02135332790/umrah-mandiri-diperbolehkan-menhaj-tegaskan-peran-ppiu-tak-tergantikan)
- [Kemenhaj Saudi: peluncuran Nusuk Umrah](https://haj.gov.sa/en/Media-Center/Ministry-News/Ministry-of-Hajj-and-Umrah-launches-Nusuk-Umrah-service-to-enable-pilgrims-from-outside-the-Kingdom)
- [Kontan: 7 perusahaan aktif membiayai umrah](https://keuangan.kontan.co.id/news/peningkatan-pelaksanaan-ibadah-umrah-bisa-jadi-potensi-bagi-multifinance-syariah)
- [Kontan: piutang multifinance syariah Rp32,17 T](https://keuangan.kontan.co.id/news/tumbuh-791-piutang-pembiayaan-syariah-multifinance-rp-3217-t-per-september-2026)
- [Umrohkit: simulasi pembiayaan umroh Amitra](https://www.umrohkit.co.id/pembiayaan-umroh-amitra/)
- [Sadar Travel: syarat dan simulasi pembiayaan umroh syariah](https://sadartravel.com/articles/panduan-pengajuan-pembiayaan-umroh-amitra-syariah-syarat-simulasi-tips)
- [Hukumonline: perbedaan perjanjian dengan MoU dan kekuatan mengikatnya](https://www.hukumonline.com/klinik/a/perbedaan-antara-perjanjian-dengan-mou-lt514689463d4b2/)
- [Baker McKenzie: dokumen pendahuluan dalam transaksi di Indonesia](https://resourcehub.bakermckenzie.com/en/resources/global-private-ma-guide-limited/asia-pacific/indonesia/topics/preliminary-documents)
- [Xendit: daftar harga virtual account, kartu, dan QRIS](https://www.xendit.co/en/pricing)
- [Midtrans: biaya per transaksi, belum termasuk PPN](https://docs.midtrans.com/docs/how-much-does-midtrans-charge-for-its-payment-service)
- [PP 38/2021: rekening penampungan PPIU](https://peraturan.go.id/files/pp38-2021bt.pdf)
- [SmartLegal: modal PJP kategori izin 1 Rp15 miliar](https://smartlegal.id/perizinan/2021/08/15/ingin-mendirikan-usaha-penyedia-jasa-pembayaran-ketahui-dulu-perizinannya/)
- [Hukumonline: PPh final 25% atas hadiah undian](https://www.hukumonline.com/klinik/a/pemungutan-pph-atas-hadiah-undian-lt584127d3d0149/)
