#!/bin/bash
set -e

echo "================================="
echo "Deploy UmrohKu"
echo "================================="

cd /var/www/umrah

echo "Pull latest code..."
git restore crawler import 2>/dev/null || true
git pull

echo "Build binaries..."
go build -o umrah-server .
go build -o crawler ./cmd/crawler/
go build -o import ./cmd/import/

echo "Restart service..."
sudo systemctl restart umrah
sudo systemctl status umrah --no-pager

echo ""
echo "Deploy selesai!"
