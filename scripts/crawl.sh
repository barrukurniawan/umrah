#!/bin/bash
set -e

echo "================================="
echo "Crawl & Import UmrohKu"
echo "================================="

cd /var/www/umrah

echo "Running crawler..."
go build -o crawler ./cmd/crawler/
./crawler

echo ""
echo "Importing to crawled.db..."
go build -o import ./cmd/import/
rm -f data/crawled.db
DB_PATH=data/crawled.db ./import output/all_*.json

echo ""
echo "Restarting service..."
go build -o umrah-server .
sudo systemctl restart umrah
sudo systemctl status umrah --no-pager

echo ""
echo "Selesai!"
