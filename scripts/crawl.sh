#!/bin/bash
# Crawl otomatis Umrohku (cron 05:00 & 17:00 WIB, lihat scripts/umrah-crawl.cron):
#   1. build & jalankan crawler 14 travel -> output/all_<waktu>.json
#   2. rapikan snapshot: simpan 4 snapshot terbaru (cadangan bila satu travel gagal),
#      sisanya & file per-travel dipindah ke output/old (dihapus setelah 30 hari)
#   3. kirim SIGHUP ke service: data dimuat ulang tanpa restart (tanpa jeda / error 502)
# Travel yang gagal/kosong di crawl ini tetap memakai hasil snapshot sebelumnya.
set -uo pipefail
cd /var/www/umrah || exit 1
export PATH=$PATH:/usr/local/go/bin

exec 9>/tmp/umrah-crawl.lock
flock -n 9 || { echo "$(date '+%F %T') crawl lain masih berjalan, lewati"; exit 0; }

echo "== $(date '+%F %T') mulai crawl"
go build -o crawler-bin ./cmd/crawler || { echo "build crawler gagal"; exit 1; }
timeout 40m ./crawler-bin -out output 2>&1 | grep -E "paket ditemukan|error crawling|selesai" || true

mkdir -p output/old
ls -1t output/all_2*.json 2>/dev/null | tail -n +5 | xargs -r mv -t output/old/
find output -maxdepth 1 -name '2*_*.json' -exec mv -t output/old/ {} +
find output/old -name '*.json' -mtime +30 -delete

sudo -n systemctl kill -s HUP umrah && echo "== $(date '+%F %T') data dimuat ulang (tanpa restart)"
