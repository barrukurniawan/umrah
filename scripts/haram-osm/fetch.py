#!/usr/bin/env python3
"""Ambil data OSM sekitar Masjidil Haram ke raw/haram-raw.json (sekali saja; hasilnya di-commit).

Pakai:  python3 scripts/haram-osm/fetch.py
Overpass menolak POST mentah (406), jadi query dikirim form-encoded dengan User-Agent.
"""
import json, pathlib, sys, urllib.parse, urllib.request

HERE = pathlib.Path(__file__).resolve().parent
MIRRORS = ["https://overpass-api.de/api/interpreter", "https://overpass.kumi.systems/api/interpreter"]

def main():
    q = (HERE / "query.txt").read_text()
    body = urllib.parse.urlencode({"data": q}).encode()
    for url in MIRRORS:
        try:
            req = urllib.request.Request(url, data=body, headers={"User-Agent": "umrohku-sim/1.0 (haram open world)"})
            raw = urllib.request.urlopen(req, timeout=180).read()
            data = json.loads(raw)
            break
        except Exception as e:  # coba mirror berikutnya
            print("gagal", url, e, file=sys.stderr)
    else:
        sys.exit("semua mirror Overpass gagal")
    out = HERE / "raw" / "haram-raw.json"
    out.parent.mkdir(exist_ok=True)
    out.write_text(json.dumps(data, ensure_ascii=False, separators=(",", ":")))
    print(out, len(data["elements"]), "elemen", f"{out.stat().st_size/1024:.0f} KB", data.get("osm3s", {}).get("timestamp_osm_base"))

if __name__ == "__main__":
    main()
