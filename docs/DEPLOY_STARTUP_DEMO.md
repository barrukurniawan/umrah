# Deploy demo `startup.umrohku.web.id`

**Status: LIVE sejak 23 Nov 2026.**

Branch `startup-demo`, di worktree lokal `~/Documents/umrah-startup`. Binary-nya sama dengan umrohku.web.id. Bedanya hanya env `DEMO_MODE=1`, folder, port, dan database. Tanpa `DEMO_MODE`, semua route demo tidak terdaftar dan paket travel fiktif tidak dimuat. Ini sudah dicek: di umrohku.web.id, `/demo` 404.

## Yang terpasang di VPS (`ubuntu@43.133.150.138`)

| Bagian | Lokasi |
|---|---|
| DNS | A `startup.umrohku.web.id` → 43.133.150.138 (Sumopod) |
| Kode & data | `/var/www/umrah-startup` (`data/` chmod 700) |
| Service | `umrah-startup.service`: `DEMO_MODE=1`, `APP_PORT=3010`, port 3010 diblokir dari internet lewat iptables seperti produksi |
| nginx | `/opt/umrah-nginx/zz-umrohku-startup.conf`, disalin ke container `wedding-summer-web-1` oleh compose override yang sudah ada. Berkas wajib diawali `zz-` supaya terbaca setelah `umrohku.conf`, karena format log `umrohku_anon` didefinisikan di sana. |
| Sertifikat | `/etc/letsencrypt/live/startup.umrohku.web.id` (webroot, ecdsa). Perpanjangan otomatis memakai hook `reload-nginx.sh` yang sudah ada. |
| Data paket | `/etc/cron.d/umrah-startup-data` jam 05:40 & 17:40: salin `all_*.json` produksi, lalu HUP. `all_zz_demo_portal.json` milik demo tidak tersentuh. |
| Reset data demo | Otomatis di dalam aplikasi setiap pukul 03.00 WIB, atau lewat Konsol admin → Reset |

Port 3001 sengaja tidak dipakai karena sudah dipakai API marryme di Docker.

## Update kode

1. Di lokal (worktree `~/Documents/umrah-startup`):
   ```bash
   ./scripts/build-css.sh
   rsync -az main.go go.mod go.sum app cmd config web scripts ubuntu@43.133.150.138:/var/www/umrah-startup/
   ```
2. Di VPS:
   ```bash
   cd /var/www/umrah-startup && PATH=$PATH:/usr/local/go/bin go build -o umrah-server . && sudo systemctl restart umrah-startup
   ```

Rsync tanpa `--delete`: berkas yang dihapus di lokal harus dihapus manual di VPS. Template hanya dimuat saat start, jadi setelah mengubah template service harus di-restart.

## Ubah nginx

Edit `/opt/umrah-nginx/zz-umrohku-startup.conf`, lalu:
```bash
docker exec wedding-summer-web-1 cp /etc/nginx/umrah/zz-umrohku-startup.conf /etc/nginx/conf.d/
docker exec wedding-summer-web-1 nginx -t && docker exec wedding-summer-web-1 nginx -s reload
```
Jangan reload kalau `nginx -t` gagal: container yang sama melayani marryme.web.id dan umrohku.web.id.

## Pengaturan
- `config/startup_demo.json`:
  - Tarif contoh: biaya per lead, komisi, harga langganan, kurs SAR.
  - Mitra pembiayaan fiktif.
  - `access_code`: isi kalau demo hanya untuk orang tertentu. Pengunjung lalu membuka `https://startup.umrohku.web.id/?kode=KODE` satu kali.
- Setelah mengubah config: `sudo systemctl restart umrah-startup`.
- Semua halaman dikirim dengan `X-Robots-Tag: noindex`, dan `robots.txt` berisi `Disallow: /`.

## Lepas demo
```bash
sudo systemctl disable --now umrah-startup
sudo rm /etc/cron.d/umrah-startup-data
```
Lalu hapus vhost `zz-umrohku-startup.conf` dari `/opt/umrah-nginx` dan dari `conf.d` di container, jalankan `nginx -t`, lalu reload.

## Sebelum fitur demo naik ke produksi
Lihat bagian 4 di PLAN_STARTUP_DEMO.md. Booking/DP, pembiayaan, undian, dan voucher Tanah Suci baru boleh live setelah:
- PT dan PSE terdaftar,
- ada perjanjian dengan travel, payment gateway/bank, dan mitra OJK,
- izin undian dari Kemensos terbit.
