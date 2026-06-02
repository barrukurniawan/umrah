# Umrohku

Rekomendasi travel agent umroh berbasis web. Cukup geser budget, pilih kebutuhan, langsung dapat rekomendasi paket umroh terbaik.

## Tech Stack

- **Backend**: Go + Fiber
- **Frontend**: HTMX + TailwindCSS
- **Database**: SQLite + GORM

## Struktur Project

```
├── main.go                     # Entry point
├── app/
│   ├── handlers/handler.go     # HTTP handlers
│   ├── models/travel.go        # GORM models (Travel, Package, DetailPackage)
│   ├── repositories/database.go # DB init, migration, dan seed data
│   └── services/recommendation.go # Scoring engine rekomendasi
├── web/
│   ├── templates/
│   │   ├── home.html            # Landing page
│   │   └── recommendations.html # Hasil rekomendasi (HTMX partial)
│   └── static/
├── data/
│   └── umrah.db                 # SQLite (auto-generated)
└── .env
```

## Menjalankan (Development)

```bash
go run main.go
```

Buka `http://localhost:3000`

## Build

```bash
go build -o umrah-app .
./umrah-app
```

## Fitur

- Slider budget 20jt – 50jt
- Pilih siapa berangkat (Sendiri / Pasangan / Keluarga / Lansia)
- Pilih prioritas (Hotel Dekat Haram / Ramah Anak & Lansia / Full Aktivitas)
- Scoring engine: mencocokkan budget, profil, dan prioritas
- 10 rekomendasi teratas diurutkan berdasarkan skor
- Detail paket: hotel, maskapai, kuota, tipe kamar, add-on, pembimbing
- HTMX: hasil pencarian muncul tanpa reload halaman

## Lisensi

MIT
