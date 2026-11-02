#!/bin/bash
# Bangun ulang CSS Tailwind setiap kali kelas di template/JS/Go berubah, lalu commit app.css.
# Butuh Node.js (npx). Server tidak perlu Node: cukup app.css yang sudah jadi.
set -euo pipefail
cd "$(dirname "$0")/.."
npx -y tailwindcss@3.4.17 -c tailwind.config.js -i web/static/css/tailwind.src.css -o web/static/css/app.css --minify
echo "web/static/css/app.css: $(wc -c < web/static/css/app.css) byte"
