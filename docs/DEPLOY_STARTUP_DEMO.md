# Deploy demo `startup.umrohku.web.id`

Branch `startup-demo`. Binary-nya sama dengan umrohku.web.id. Bedanya hanya env `DEMO_MODE=1`, folder, port, dan database. Tanpa `DEMO_MODE`, semua route demo **tidak terdaftar**, tidak ada pita demo, dan paket travel fiktif tidak dimuat. Ini sudah dicek: produksi tetap sama persis.

## Sekali saja (setup)

1. **DNS** (di panel domain): A record `startup` → `43.133.150.138`.
2. **Folder di VPS:**
   ```bash
   sudo mkdir -p /var/www/umrah-startup && sudo chown ubuntu:ubuntu /var/www/umrah-startup
   ```
3. **Kirim kode** dari folder worktree `~/Documents/umrah-startup`:
   ```bash
   rsync -az main.go go.mod go.sum app cmd config web scripts ubuntu@43.133.150.138:/var/www/umrah-startup/
   ```
   Lalu build di VPS:
   ```bash
   cd /var/www/umrah-startup && PATH=$PATH:/usr/local/go/bin go build -o umrah-server .
   ```
4. **Data paket:** salin hasil crawl produksi, lalu jadwalkan cron setiap malam. Berkas `all_zz_demo_portal.json` milik demo tidak ikut tertimpa.
   ```bash
   mkdir -p /var/www/umrah-startup/output /var/www/umrah-startup/data
   rsync -a /var/www/umrah/output/all_*.json /var/www/umrah-startup/output/
   # crontab -e (user ubuntu): 02:30 WIB = 19:30 UTC
   30 19 * * * rsync -a /var/www/umrah/output/all_*.json /var/www/umrah-startup/output/ && sudo systemctl kill -s HUP umrah-startup
   ```
   Opsional: salin juga `data/dbip-city-lite.mmdb` supaya analitik lokasi aktif.
5. **systemd** `/etc/systemd/system/umrah-startup.service`:
   ```ini
   [Unit]
   Description=umrohku demo startup
   After=network.target

   [Service]
   User=ubuntu
   WorkingDirectory=/var/www/umrah-startup
   Environment=DEMO_MODE=1
   Environment=APP_PORT=3001
   ExecStart=/var/www/umrah-startup/umrah-server
   Restart=always

   [Install]
   WantedBy=multi-user.target
   ```
   ```bash
   sudo systemctl daemon-reload && sudo systemctl enable --now umrah-startup
   ```
   **Jangan** salin `.env` produksi. Demo tidak memakai login Google, password admin, atau kunci API.
6. **nginx** `/etc/nginx/sites-available/startup.umrohku.web.id`. Salin dari server block umrohku.web.id, lalu ganti `server_name startup.umrohku.web.id` dan `proxy_pass http://127.0.0.1:3001`. Setelah itu:
   ```bash
   sudo ln -s /etc/nginx/sites-available/startup.umrohku.web.id /etc/nginx/sites-enabled/
   sudo nginx -t && sudo systemctl reload nginx
   sudo certbot --nginx -d startup.umrohku.web.id
   ```

## Update berikutnya
1. Build CSS: `./scripts/build-css.sh`.
2. Kirim kode: `rsync -az ...` ke `/var/www/umrah-startup/`.
3. Build di VPS, lalu `sudo systemctl restart umrah-startup`.

Data demo (`data/demo.db`) kembali ke kondisi awal otomatis setiap pukul 03.00 WIB. Admin juga bisa mereset kapan saja: **Konsol admin → Pendapatan → Reset data demo**.

## Pengaturan
- `config/startup_demo.json`:
  - Tarif contoh: biaya per lead, komisi, harga langganan, kurs SAR.
  - Mitra pembiayaan fiktif.
  - `access_code`: isi kalau demo hanya untuk orang tertentu. Pengunjung lalu membuka `https://startup.umrohku.web.id/?kode=KODE` satu kali.
- Semua halaman dikirim dengan `X-Robots-Tag: noindex`, dan `robots.txt` berisi `Disallow: /`.

## Yang perlu dikerjakan sebelum fitur demo naik ke produksi
Lihat bagian 4 di PLAN_STARTUP_DEMO.md. Booking/DP, pembiayaan, undian, dan voucher Tanah Suci baru boleh live setelah:
- PT dan PSE terdaftar,
- ada perjanjian dengan travel, payment gateway/bank, dan mitra OJK,
- izin undian dari Kemensos terbit.
