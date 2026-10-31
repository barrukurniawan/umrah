#!/bin/bash
# Perbarui database lokasi DB-IP Lite (gratis, CC BY 4.0) untuk analitik Umrohku, lalu restart service.
# Dipasang di VPS: /usr/local/bin/umrah-geoip-update.sh, cron bulanan (tanggal 3, 04:30 WIB).
set -euo pipefail
DEST=/var/www/umrah/data/dbip-city-lite.mmdb
TMP=$(mktemp)
trap 'rm -f "$TMP" "$TMP.gz"' EXIT
for m in "$(date +%Y-%m)" "$(date -d '-1 month' +%Y-%m)"; do
  if curl -fsSL -o "$TMP.gz" "https://download.db-ip.com/free/dbip-city-lite-$m.mmdb.gz"; then
    gunzip -c "$TMP.gz" > "$TMP"
    [ "$(stat -c %s "$TMP")" -gt 50000000 ] || { echo "$(date '+%F %T') file $m terlalu kecil, batal"; exit 1; }
    install -m 644 -o ubuntu -g ubuntu "$TMP" "$DEST"
    systemctl restart umrah
    echo "$(date '+%F %T') GeoIP diperbarui ke $m"
    exit 0
  fi
done
echo "$(date '+%F %T') gagal mengunduh GeoIP"
exit 1
