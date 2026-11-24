# Rencana: Bayar QRIS + Navigasi Bawah (demo `startup.umrohku.web.id`)

Tujuan: jamaah bisa memindai **QRIS apa pun** dari umrohku, lalu membayar dalam alur seperti aplikasi dompet digital. Di demo, pemindaian dan pembacaan QR-nya **sungguhan**, sedangkan pembayarannya **simulasi** (tidak ada dana yang berpindah).

Hanya aktif dengan `DEMO_MODE=1`. Produksi umrohku.web.id tidak berubah.

---

## 1. Yang sudah dibangun (demo)

### Navigasi bawah
`partials/demo.html` → `demo_tabbar`, dengan 5 ikon garis bertema hijau:

| Tab | Route | Aktif juga di |
|---|---|---|
| Beranda | `/` | |
| Market | `/market` | `/tanah-suci`, `/mandiri`, `/perlengkapan`, `/travel/*` |
| **QRIS** (tombol bulat menonjol di tengah) | `/bayar` | |
| Riwayat | `/riwayat` | `/booking/*` |
| Profil | `/profil` | `/akun*`, `/tabungan`, `/tersimpan`, `/inbox` |

- Disembunyikan di halaman bisnis (`/portal`, `/admin`, `/mitra-pembiayaan`, `/toko`), di alur bayar, dan di simulasi 3D.
- Di ponsel: pengalih peran demo menjadi ikon saja, dan chat dukungan disembunyikan supaya bawah layar lega.

### Alur bayar
1. **`/bayar` (scan):** kamera belakang dengan bingkai pemindai, garis animasi, dan tombol senter bila didukung perangkat.
   - Pembacaan QR memakai `BarcodeDetector` bawaan browser bila ada (Chrome Android). Kalau tidak ada, memakai **jsQR 1.4.0** yang disimpan di server sendiri (`/static/js/vendor`).
   - Alternatif: **Dari galeri** (foto/screenshot QR), **QR contoh** (statis & dinamis, untuk demo tanpa kamera), atau tempel isi QR.
2. **`POST /bayar/decode`:** server membaca QR (`app/qris`) lalu menampilkan konfirmasi:
   - merchant, kota, kategori, NMID, statis/dinamis
   - nominal: diisi sendiri untuk QR statis, terkunci untuk QR dinamis
   - tip/biaya layanan (tag 55–57)
   - sumber dana
   - detail QR: penyelenggara, PAN tersamar, kriteria, MCC, referensi, terminal, CRC valid
3. **PIN 6 angka:** lembar keypad. Di demo, PIN apa pun diterima.
4. **`POST /bayar/qris`:** QR **di-decode ulang di server** (isi dari browser tidak dipercaya). Lalu dicek:
   - batas QRIS Rp 10 jt per transaksi,
   - saldo,
   - PIN.

   Setelah itu saldo contoh dipotong, transaksi dicatat, dan notifikasi terkirim.
5. **Struk `/riwayat/qris/:code`:** status berhasil, rincian, nomor referensi.
6. **`/riwayat`:** gabungan QRIS, DP/pelunasan umroh, oleh-oleh, dan voucher. Dikelompokkan per tanggal dan bisa disaring.
7. **`/profil`:** kartu akun, sumber dana, ringkasan, dan menu.

### API decode
`POST /api/qris/decode {"qr_content": "..."}`
- Nama field mengikuti `QrMobileDecodeResult` dari **`POST /qr/mobile/decode`** di core-payment-ultimate: `qr_type`, `qris_type`, `merchant_name`, `merchant_city`, `merchant_pan`, `acquirer_name`, `transaction_amount`, `service_charge_*`, `total_amount`, `transaction_limit`.
- Tujuannya supaya nanti frontend tinggal diarahkan ke layanan issuer sungguhan.

### Parser QRIS (`app/qris`)
Ditulis sendiri dari standar EMVCo MPM. Kode internal core-payment tidak disalin. Langkah validasinya mengikuti `ProcessDecodeQR`:
1. hapus baris baru,
2. cek bentuk EMV,
3. urai TLV,
4. Payload Format Indicator `01`,
5. CRC16-CCITT di tag 63,
6. Merchant Account Information tag 26–51 (tag 51 = NMID & kriteria),
7. statis/dinamis (tag 01),
8. nominal (54), tip (55–57), negara/kota/kode pos, data tambahan 62 (bill, store, reference, terminal).

Nama penyelenggara diambil dari GUID tag 26 (mis. `COM.GO-JEK.WWW` → GoPay) atau dari NNS 8 digit di PAN.

**Tes:**
- `go test ./app/qris`: vektor CRC standar, QR statis/dinamis, dan penolakan untuk CRC rusak, bukan EMV, tanpa merchant, serta TLV terpotong.
- Diverifikasi lokal dengan 12 contoh QRIS dari tes core-payment: yang valid terbaca benar (nama, kota, NMID, kriteria, MCC, tip), dan yang sengaja dirusak ditolak. Contoh itu tidak disalin ke repo ini.

### Prinsip yang dijaga
- **Umrohku tidak menyimpan saldo.** Sumber dana adalah rekening jamaah di bank syariah mitra (contoh). Transaksi digambarkan diproses oleh **mitra PJP berizin BI**.
- **Tidak ada panggilan ke switching/acquirer.** Tidak ada dana yang sampai ke merchant. Setiap layar menyebut "simulasi".

---

## 2. Jalan ke produksi (setelah demo)

| Langkah | Isi | Bergantung pada |
|---|---|---|
| 1. Model bisnis | umrohku sebagai **front-end** dari issuer berizin, bukan issuer sendiri. Dana dan saldo tetap di PJP/bank mitra. | Perjanjian dengan PJP/bank (mis. lewat layanan issuer seperti core-payment) |
| 2. Akun & KYC | Hubungkan akun umrohku ke akun dompet/rekening mitra (OAuth/binding + OTP). Basic vs premium mengikuti limit mitra. | API binding mitra |
| 3. Decode | Ganti `POST /bayar/decode` lokal menjadi panggilan `POST /qr/mobile/decode` milik issuer (JWT user). Parser lokal tetap dipakai untuk pra-validasi & tampilan cepat. | Kredensial issuer |
| 4. Bayar | `POST` payment issuer dengan PIN di sisi issuer (PIN tidak pernah lewat/disimpan umrohku), idempotency key = `qr_id`, lalu status via callback/polling. | API payment issuer |
| 5. Riwayat & struk | Sumber kebenaran = issuer (RRN, status, reversal/void). umrohku hanya menyimpan salinan untuk tampilan. | |
| 6. Keamanan | Rate limit scan/bayar, cek NMID diblokir (dilakukan issuer), log tanpa data sensitif, kamera hanya di HTTPS. | |
| 7. Kepatuhan | PSE Komdigi, UU PDP (data transaksi), S&K pembayaran, penanganan keluhan sesuai ketentuan BI. | PT & PSE |

## 3. Kriteria selesai demo
- [x] Kamera membaca QRIS asli, dan unggah gambar juga terbaca (diuji di Chrome dengan gambar QR dari payload QRIS asli).
- [x] QR bukan QRIS → layar "Ini bukan QRIS" beserta isinya; tautan bisa dibuka.
- [x] CRC rusak / format salah → pesan jelas, tidak bisa dibayar.
- [x] Saldo tidak cukup, melebihi Rp 10 jt, PIN salah format, dan POST tanpa Origin (CSRF) semuanya ditolak.
- [x] Navigasi bawah aktif sesuai halaman; tidak ada scroll horizontal di 390 px.
- [ ] Uji di iPhone (Safari) dan Android sungguhan di domain HTTPS. Izin kamera hanya berjalan di HTTPS.
