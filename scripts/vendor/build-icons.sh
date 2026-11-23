#!/usr/bin/env bash
# Bangun ulang web/static/js/vendor/lucide-icons.js (subset ikon Lucide yang dipakai).
# Tambah ikon baru: impor di scripts/vendor/lucide-entry.js (nama PascalCase, mis. data-lucide="map-pin" -> MapPin),
# lalu jalankan skrip ini. Versi dikunci supaya tampilan ikon tidak berubah diam-diam.
set -euo pipefail
cd "$(dirname "$0")/../.."
TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
cp scripts/vendor/lucide-entry.js "$TMP/entry.js"
(cd "$TMP" && npm init -y >/dev/null && npm i --no-audit --no-fund lucide@1.68.0 esbuild@0.24.2 >/dev/null &&
  npx esbuild entry.js --bundle --minify --format=iife --target=es2017 --legal-comments=none \
    --banner:js="/*! Lucide v1.68.0 (subset ikon Umrohku) | ISC License | https://lucide.dev/license */" --outfile=out.js)
cp "$TMP/out.js" web/static/js/vendor/lucide-icons.js
echo "ok: web/static/js/vendor/lucide-icons.js"
