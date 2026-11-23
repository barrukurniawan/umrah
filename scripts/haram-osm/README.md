# Denah Masjidil Haram dari OpenStreetMap

Sumber data open world "Jelajah Masjidil Haram" (lihat `PLAN_OPEN_WORLD.md`).

| Berkas | Isi |
|---|---|
| `query.txt` | Query Overpass (relasi masjid 1472531, Ka'bah, Hijr, Maqam, sudut-sudut, Safa/Marwah, jalur sa'i, pintu, menara, bangunan sekitar) |
| `fetch.py` | Ambil ulang data ke `raw/haram-raw.json` (butuh internet; jalankan hanya bila ingin memperbarui) |
| `raw/haram-raw.json` | Salinan data OSM mentah, di-commit supaya build bisa diulang tanpa jaringan |
| `extra.json` | Tempat yang belum ada di OSM (mis. Half Million), koordinat dari Google Maps |
| `build.py` | Mengolah raw + extra menjadi `web/static/sim/data/haram.json` |
| `cek.png` | Gambar cek denah (dibuat `build.py --check`) |

```sh
python3 scripts/haram-osm/fetch.py          # opsional: perbarui data OSM
python3 scripts/haram-osm/build.py --check  # tulis haram.json + cek.png
```

Koordinat `haram.json`: meter dunia, pusat Ka'bah (0,0), x = timur, z = selatan (sama dengan `tawaf.js`).
Jarak dikali `k` (0,5) kecuali radius `localR` (22 m asli) di sekitar Ka'bah yang tetap ukuran asli.
Tinggi bangunan `h` sudah dikali `k`.

Overpass menolak POST mentah (HTTP 406); `fetch.py` mengirim query form-encoded dengan User-Agent.

**Lisensi**: data © kontributor OpenStreetMap, ODbL 1.0. Halaman yang memakai data ini harus menampilkan
atribusi "© kontributor OpenStreetMap". `haram.json` adalah basis data turunan dan ikut dipublikasikan apa adanya.
