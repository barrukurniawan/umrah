# Rencana: Cicilan, DP Ringan, Cashback, dan Undian Umroh (demo `startup.umrohku.web.id`)

Dokumen ini **melanjutkan** [PLAN_STARTUP_DEMO.md](PLAN_STARTUP_DEMO.md). Ia memperdalam Tahap 2 (#8 Booking + DP, #10 Voucher, #11 Komisi, #12 Cicil sebelum berangkat, #13 Rujukan pembiayaan) dan menambah dua fitur baru: **cashback** dan **undian umroh berhadiah**.

- Rujukan isi bisnis: [docs/proposal/umrohku-proposal.html](docs/proposal/umrohku-proposal.html), terutama bagian "Kenapa bayar lewat umrohku, bukan langsung ke travel".
- Semua aturan demo di PLAN_STARTUP_DEMO tetap berlaku:
  - fitur dibungkus `demo.Enabled()`
  - pita "MODE DEMO"
  - data dummy
  - travel fiktif
  - **tidak ada uang sungguhan yang bergerak**
  - notifikasi hanya ke Kotak masuk demo

---

## 0. Nilai jual yang harus terasa di demo

Harga paket **sama** dengan harga resmi travel. Jamaah yang membayar lewat umrohku mendapat empat keuntungan yang tidak ada bila membayar langsung ke travel:

| # | Keuntungan | Inti |
|---|---|---|
| 1 | **Bisa dicicil secara syariah** | Jalur utama: cicil sebelum berangkat, tanpa utang. Jalur kedua: berangkat dulu lewat mitra pembiayaan syariah. |
| 2 | **DP lebih ringan** | DP minimal di umrohku lebih kecil dari DP normal travel, sesuai perjanjian dengan travel mitra. |
| 3 | **Cashback & diskon** | Dibiayai dari komisi umrohku, **bukan** memotong harga travel. |
| 4 | **Undian umroh berhadiah** | Setiap booking otomatis ikut. Tanpa biaya ikut dan tanpa kenaikan harga. |

Posisi produk: "tiket.com untuk umroh". Umrohku perantara; layanan ibadah dan perjalanan tetap dijalankan travel PPIU. Dana jamaah selalu masuk ke rekening penampungan travel (PP 38/2021), **tidak pernah** ke rekening umrohku.

---

## 1. Prinsip yang tidak boleh dilanggar

1. **Umrohku tidak menampung dana jamaah.** Semua DP dan pelunasan diarahkan ke VA/QRIS atas nama rekening penampungan travel. Hal ini berlaku juga untuk mock.
2. **Umrohku tidak memberi pinjaman.** Pembiayaan hanya diteruskan (rujukan) ke mitra berizin OJK.
3. **Tidak ada "saldo umrohku" yang bisa ditarik atau dibelanjakan bebas.** Saldo seperti itu menyerupai uang elektronik yang butuh izin BI.
   - Cashback diwujudkan sebagai **potongan tagihan pelunasan** ke travel. Selisihnya ditanggung umrohku lewat pengurangan komisi di Tagihan travel.
4. **Undian bebas maysir:**
   - Peserta tidak membayar apa pun untuk ikut.
   - Harga paket tidak dinaikkan.
   - Tidak bisa membeli kupon tambahan.
   - Hadiah dibiayai anggaran promosi umrohku.
5. **Tidak ada nomor izin karangan.** Izin undian Kemensos, PPIU, dan PSE ditampilkan sebagai "dalam proses" atau `[isi]`. Jangan memakai nomor palsu.
6. **Tidak memakai nama mitra keuangan asli** (Kredivo, Bank Aladin Syariah, FIF Amitra, Adira, BSI, dan lain-lain) di demo, sampai ada perjanjian tertulis.
   - Pakai nama fiktif: **"Mitra Syariah Amanah (contoh)"**, **"Mitra Syariah Barokah (contoh)"**, **"Bank Syariah Contoh"**.
7. **Harga coret jujur.** Dilarang menampilkan "harga normal" yang dinaikkan supaya diskon terlihat besar.
   - Pembanding DP hanya boleh memakai `dp_normal_travel` yang tercatat di perjanjian (tabel `travel_terms`).

---

## 2. Data (GORM, SQLite `data/demo.db`)

Model baru diletakkan di package yang sudah direncanakan: `app/booking`, `app/payments`, `app/financing`, ditambah **`app/promo`** dan **`app/draws`** yang baru. Semua `AutoMigrate` hanya dijalankan saat `demo.Enabled()`.

```go
// app/booking: ketentuan per travel mitra (hasil perjanjian)
type TravelTerms struct {
    ID              uint   `gorm:"primaryKey"`
    TravelSlug      string `gorm:"size:80;uniqueIndex"`
    DPMinUmrohku    int64  // DP minimal lewat umrohku, per orang (mis. 500_000)
    DPNormalTravel  int64  // DP normal bila bayar langsung ke travel (mis. 3_000_000)
    PelunasanHMinus int    // pelunasan paling lambat H-xx keberangkatan (mis. 45)
    DPHangusRule    string `gorm:"size:300"` // teks aturan pembatalan, ditampilkan di checkout
    KomisiPersen    float64 // komisi umrohku dari travel
    AgreementRef    string `gorm:"size:120"` // "[isi no. perjanjian]" di demo
}

// app/promo: diskon & cashback
type PromoRule struct {
    ID          uint   `gorm:"primaryKey"`
    Code        string `gorm:"size:40;uniqueIndex"` // kosong = otomatis berlaku
    Kind        string `gorm:"size:30"` // "diskon_dp" | "cashback_pelunasan"
    Amount      int64   // nominal tetap, atau
    Percent     float64 // persen dari harga paket
    Cap         int64   // batas maksimum potongan
    TravelSlug  string `gorm:"size:80"` // kosong = semua travel mitra
    MinPrice    int64
    Quota       int
    Used        int
    StartsAt    time.Time
    EndsAt      time.Time
    FundedBy    string `gorm:"size:20"` // selalu "umrohku" untuk fitur ini
}

type PromoUse struct {
    ID         uint `gorm:"primaryKey"`
    PromoID    uint `gorm:"index"`
    BookingID  uint `gorm:"index"`
    Kind       string `gorm:"size:30"`
    Amount     int64      // nilai potongan yang diberikan ke jamaah
    Status     string `gorm:"size:20"` // "dijanjikan" | "diterapkan" | "batal"
    AppliedAt  *time.Time // cashback: saat tagihan pelunasan dibuat
}

// app/draws: undian
type Draw struct {
    ID            uint   `gorm:"primaryKey"`
    Title         string `gorm:"size:150"` // "Undian Umroh Gratis Periode Jan–Mar 2027"
    PrizeDesc     string `gorm:"size:300"` // "2 paket umroh 9 hari, travel mitra pilihan"
    Winners       int
    StartsAt      time.Time // booking berDP lunas di rentang ini ikut
    EndsAt        time.Time
    DrawAt        time.Time
    PermitNo      string `gorm:"size:120"` // "dalam proses" di demo
    Terms         string `gorm:"type:text"`
    Status        string `gorm:"size:20"` // draft|terbuka|ditutup|diundi
    EntriesHash   string `gorm:"size:64"` // sha256 daftar entri, dipublikasi saat ditutup
    Seed          string `gorm:"size:128"` // diisi saat pengundian (lihat §4.5)
}

type DrawEntry struct {
    ID        uint   `gorm:"primaryKey"`
    DrawID    uint   `gorm:"index"`
    BookingID uint   `gorm:"uniqueIndex:ux_draw_booking"` // 1 booking = 1 entri per jamaah
    UserID    uint   `gorm:"index"`
    Pilgrim   int    // urutan jamaah dalam booking (1..n)
    Code      string `gorm:"size:16;uniqueIndex"` // "UMR-7K2Q-91"
    Void      bool   // true bila booking dibatalkan sebelum pengundian
    CreatedAt time.Time
}
```

Catatan:
- `uniqueIndex` pada `DrawEntry` sebaiknya gabungan `(DrawID, BookingID, Pilgrim)`, karena **satu entri per jamaah**, bukan per rupiah.
- Model booking (`Booking`, `Payment`, `FinancingApplication`) mengikuti #8–#13 di PLAN_STARTUP_DEMO. Booking perlu kolom `DPAmount`, `PromoTotal`, `CashbackPromised`, `Status` (`menunggu_dp` | `dp_lunas` | `lunas` | `batal` | `berangkat` | `pulang`).

---

## 3. Fitur & layar

Ukuran: S = ≤1 hari, M = 2–3 hari, L = ±1 minggu.

### 3.1 Halaman `/keuntungan` (S)
- Isi persis empat keuntungan di §0, dengan kotak 2×2 seperti di proposal.
- Ada tombol "Coba pesan paket" (ke pencarian) dan "Lihat undian yang berjalan" (ke `/undian`).
- Penutup: "Harga paket sama dengan harga resmi travel. Dana langsung ke rekening penampungan travel."
- Tautkan dari `/investor` dan dari tur `/demo`.

### 3.2 Simulasi cicilan di kartu paket (M)
Fitur ini memperluas modal **#7 "Mau cicil?"**: setelah memilih jalur, tampilkan simulasi.
- **Cicil sebelum berangkat (tanpa utang):**
  - Input: target tanggal berangkat (default: tanggal paket), jumlah orang, setoran awal.
  - Output: setoran per bulan = (harga × orang − setoran awal) ÷ jumlah bulan sampai H-`PelunasanHMinus`.
  - Tombol **"Buat rencana tabungan"** membuat `SavingsGoal` (sudah ada di `app/accounts/savings.go`, route `/tabungan`) dengan `PackageKey`, `PackagePrice`, dan `DownPayment` terisi. Tidak perlu model baru.
  - Label: "Tanpa margin, tanpa utang".
- **Berangkat dulu (pembiayaan mitra):**
  - Tampilkan pilihan tenor 6/12/24 bulan dan **estimasi** angsuran dari 2–3 mitra fiktif.
  - Angsuran dihitung dari `config/financing_demo.json` (ujrah/biaya per tenor per mitra) dan diberi label besar **"Contoh perhitungan, bukan penawaran. Angka resmi dari mitra setelah pengajuan."**
  - Tombol "Ajukan" masuk ke alur #13.
- Event analitik: `intent_cicil` (sudah direncanakan), ditambah `simulasi_cicil` dengan properti `jalur` dan `tenor`.

### 3.3 DP ringan (M), memperluas #8
- Di kartu paket travel mitra tampilkan chip: **"DP di umrohku Rp500 rb · di travel Rp3 jt"**. Angkanya dari `TravelTerms`. Chip hanya muncul bila `DPMinUmrohku < DPNormalTravel`.
- Form booking: pilihan DP minimal `DPMinUmrohku` sampai lunas.
- Teks wajib di bawah tombol bayar:
  - Batas pelunasan: "paling lambat H-45".
  - `DPHangusRule`.
  - "Dana masuk ke rekening penampungan PT … atas nama jamaah".
- Booking `dp_lunas` mengunci seat (kurangi sisa seat di portal travel).
- Booking yang lewat batas pelunasan → status `batal` otomatis (job harian demo), entri undiannya `Void=true`.

### 3.4 Cashback & diskon (M), memperluas #10 Voucher
- Dua jenis promo:
  - `diskon_dp`: memotong DP saat checkout (seperti voucher #10).
  - `cashback_pelunasan`: **dijanjikan** saat DP, **diterapkan** sebagai potongan pada tagihan pelunasan.
- Checkout menampilkan rincian:
  ```
  Harga paket (resmi travel)      Rp 27.500.000
  Diskon DP umrohku                 − Rp 250.000
  Cashback pelunasan (dijanjikan)   − Rp 500.000
  Ditanggung oleh umrohku, harga travel tidak berubah.
  ```
- **Tagihan travel** (Tahap 1 #3/#5 dan #11): komisi booking dikurangi `PromoUse.Amount` milik booking itu, sehingga terlihat "Komisi kotor − promo ditanggung umrohku = komisi bersih". Bila promo lebih besar dari komisi, selisihnya tercatat sebagai "biaya promosi umrohku" di admin.
- Admin `/admin/promo`:
  - CRUD `PromoRule`, kuota, dan masa berlaku.
  - Grafik "biaya promo vs komisi" per bulan.
- Validasi:
  - Satu booking maksimal satu `diskon_dp` dan satu `cashback_pelunasan`.
  - Kuota habis atau kedaluwarsa → tolak dengan pesan jelas.
  - Promo tidak bisa dicairkan menjadi uang.

### 3.5 Undian umroh berhadiah (L)
**Sisi jamaah:**
- `/undian`: daftar undian. Kartu berisi judul, hadiah, periode, jumlah entri, status, dan nomor izin ("dalam proses").
- `/undian/:id`:
  - Syarat & ketentuan.
  - Penegasan "Gratis. Tidak perlu membeli kupon. Setiap jamaah dalam booking yang DP-nya lunas di periode ini otomatis mendapat 1 kode."
  - Hash daftar entri (setelah ditutup) dan hasil pengundian (nama disamarkan: "B***u K., Bandung").
- `/akun/undian`: kode undian milik pengguna.
- Kotak masuk demo menerima pesan "Kode undianmu: UMR-7K2Q-91" setelah DP lunas.

**Aturan entri:**
- Dibuat otomatis saat booking berstatus `dp_lunas` dan `paid_at` di rentang `StartsAt–EndsAt`.
- Jumlahnya = jumlah jamaah dalam booking.
- Booking `batal` sebelum `DrawAt` → `Void=true`, tidak ikut diundi.
- Tidak ada cara lain mendapat entri (tidak ada beli kupon atau poin).

**Admin `/admin/undian`:**
- Buat undian (draft) → Buka → Tutup.
- Saat ditutup: hitung `EntriesHash` = sha256 dari daftar `Code` aktif yang diurutkan, lalu tampilkan publik.
- Tombol **Undi** (§4.5).
- Ekspor CSV berita acara: daftar entri, hash, seed, dan pemenang.

**Pengundian yang bisa diaudit (§4.5):**
1. Saat ditutup, `EntriesHash` dipublikasikan di `/undian/:id`.
2. Saat mengundi, admin memasukkan **seed saksi** (mis. angka yang dibacakan saksi atau notaris), dan sistem menggabungkannya: `seed = sha256(EntriesHash + seed_saksi)`.
3. Pemenang ditentukan dengan mengurutkan entri aktif berdasarkan `sha256(seed + Code)`, lalu mengambil N teratas.
4. Halaman publik menampilkan seed dan algoritmanya, sehingga siapa pun bisa menghitung ulang.
5. Unit test: hasilnya deterministik untuk input yang sama, entri `Void` tidak pernah menang, dan satu kode tidak menang dua kali.

**Teks kepatuhan** (tampilkan di S&K undian):
- Penyelenggara: PT umrohku `[isi]`.
- Izin undian gratis berhadiah Kemensos: `[dalam proses]`.
- Hadiah dibiayai penyelenggara.
- Pajak hadiah ditanggung `[isi]`.
- Hadiah tidak dapat diuangkan `[atau isi kebijakan]`.
- Mekanisme ditinjau dewan syariah atau ustadz pendamping `[isi]`.

**Hadiah haji:** jangan dibuat di demo. Cukup satu baris di `/investor` dan di roadmap: "Hadiah haji khusus menyusul setelah skala bisnis memungkinkan."

### 3.6 Peran demo & seed (S)
- Seed:
  - `TravelTerms` untuk 3 travel fiktif, dengan DP umrohku 500 rb / 750 rb / 1 jt dan DP normal 3–5 jt.
  - 3 `PromoRule` (1 diskon DP, 1 cashback, 1 kode khusus).
  - 1 undian berstatus `terbuka` dengan ±40 entri dari booking dummy.
  - 1 undian lampau berstatus `diundi` lengkap dengan pemenang.
- Pengalih peran tetap: Jamaah · Travel · Admin · Bank/Pembiayaan.

---

## 4. Urutan kerja

1. Model + migrasi (`app/promo`, `app/draws`, kolom tambahan booking, `TravelTerms`), seed demo.
2. §3.3 DP ringan, karena bergantung pada #8 Booking + DP yang sudah direncanakan.
3. §3.4 Cashback & diskon, termasuk pemotongan komisi di Tagihan.
4. §3.2 Simulasi cicilan + sambungan ke `/tabungan`.
5. §3.5 Undian: entri otomatis → admin buka/tutup → pengundian yang bisa diaudit → halaman publik.
6. §3.1 Halaman `/keuntungan`, lalu tambahkan langkahnya ke tur `/demo` dan alur presentasi:
   > Jamaah: cari paket → chip "DP di umrohku Rp500 rb" → simulasi cicil → pesan dengan promo → DP mock lunas → kode undian masuk Kotak masuk demo → Admin: tutup & undi → pemenang tampil.

Ukuran total kira-kira **2–3 minggu** untuk satu developer.

## 5. Kriteria selesai (cek sebelum dianggap beres)
- [ ] Tidak ada satu pun alur yang membuat dana "masuk ke umrohku". Semua halaman bayar menyebut rekening penampungan travel.
- [ ] Harga paket di checkout = harga di kartu = harga travel. Potongan selalu diberi label "ditanggung umrohku".
- [ ] Undian:
  - Tidak ada tombol atau jalan untuk membeli kupon.
  - Entri `Void` tidak bisa menang.
  - Hasil bisa dihitung ulang dari seed yang dipublikasikan.
  - Unit test lulus.
- [ ] Tidak ada nama lembaga keuangan asli dan tidak ada nomor izin karangan.
- [ ] Semua route baru hanya terdaftar dalam `demo.Enabled()`, dan produksi `umrohku.web.id` tidak berubah.
- [ ] Tampilan ponsel 390 px tanpa scroll horizontal.
- [ ] `go test ./...` lulus.

## 6. Yang masih menunggu pemilik (jangan dikarang agent)
- Link sumber berita kasus 52 jamaah ke Thailand (untuk proposal, bukan untuk app).
- Nominal DP ringan yang disepakati travel sungguhan.
- Mitra pembiayaan & bank syariah yang produknya sudah dikonfirmasi.
- Izin Kemensos untuk undian, serta besaran anggaran hadiah per periode.
- Peninjau syariah untuk mekanisme undian dan cashback.
