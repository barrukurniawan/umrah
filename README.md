# Umrohku

Rekomendasi travel agent umroh berbasis web. Cukup geser budget, pilih kebutuhan, langsung dapat rekomendasi paket umroh terbaik.

## Tech Stack

- **Backend**: Go + Fiber
- **Frontend**: HTMX + TailwindCSS
- **Database**: SQLite + GORM
- **Crawler**: Colly + goquery

## Build

```bash
go build .                     # web app → ./umrah
go build -o crawler ./cmd/crawler/  # crawler
go build -o import ./cmd/import/    # import tool
```

## Menjalankan

```bash
./umrah
# Buka http://localhost:3000
```

## Dua DB, Switch via Env `DB_PATH`

| DB | Data | Cara Aktifkan |
|---|---|---|
| `data/umrah.db` | Data dummy (default, auto-seed) | `./umrah` |
| `data/crawled.db` | Data hasil crawler (import JSON) | `DB_PATH=data/crawled.db ./umrah` |

Untuk reset data dummy: `rm data/umrah.db && ./umrah` (auto-seed ulang).

## Workflow Crawler → Import → Web

```bash
# 1. Jalankan crawler — scrape semua website di config/sites.json
./crawler

# 2. Import hasil crawler ke DB khusus
DB_PATH=data/crawled.db ./import output/all_*.json

# 3. Jalankan web app dengan data crawler
DB_PATH=data/crawled.db ./umrah
# Buka http://localhost:3000 — isinya data real dari Hamdan Tour & Taiba Medina
```

Balik ke dummy: jalankan `./umrah` tanpa env (default `data/umrah.db`).

## Menambah Website Target Crawler

1. Tambah entry di `config/sites.json`:
```json
{ "name": "Travel XYZ", "url": "https://...", "parser": "xyz" }
```
2. Buat parser baru di `app/crawlers/xyz.go` — implementasi `Parser` interface
3. Daftarkan di `cmd/crawler/main.go` bagian `switch`

## Struktur Project

```
├── main.go                       # Web app entry point
├── cmd/
│   ├── crawler/main.go           # Crawler runner
│   └── import/main.go            # Import JSON → SQLite
├── config/
│   └── sites.json                # Daftar website target crawler
├── app/
│   ├── handlers/handler.go       # HTTP handlers
│   ├── models/travel.go          # GORM models
│   ├── repositories/database.go  # DB init + seed data
│   ├── services/recommendation.go # Scoring engine
│   └── crawlers/                 # Crawler parsers
├── web/templates/                # HTMX partials
├── output/                       # Hasil crawl JSON (gitignored)
└── data/                         # SQLite files (gitignored)
```

## Fitur

- Slider budget 20jt – 50jt
- Pilih siapa (Sendiri / Pasangan / Keluarga / Lansia)
- Pilih prioritas (Dekat Haram / Ramah Anak-Lansia / Full Aktivitas)
- Scoring engine — 10 rekomendasi teratas
- Detail paket: hotel, maskapai, kuota, tipe kamar, pembimbing
- HTMX: hasil tanpa reload
- Month filter: filter keberangkatan per bulan

## Lisensi

MIT
