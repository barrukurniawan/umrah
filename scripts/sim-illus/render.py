#!/usr/bin/env python3
"""Render ilustrasi Niat & Tahallul ke web/static/sim/img/*.webp.
Jalankan dari root repo: python3 scripts/sim-illus/render.py  (butuh Google Chrome & Pillow)."""
import http.server, os, subprocess, sys, tempfile, threading, functools
from PIL import Image

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'
SCENES = sys.argv[1:] or ['pesawat', 'birali', 'aisyah', 'cukur-pria', 'cukur-wanita']
OUT = os.path.join(ROOT, 'web/static/sim/img')
os.makedirs(OUT, exist_ok=True)

class Quiet(http.server.SimpleHTTPRequestHandler):
    def log_message(self, *a): pass
srv = http.server.ThreadingHTTPServer(('127.0.0.1', 8099), functools.partial(Quiet, directory=ROOT))
threading.Thread(target=srv.serve_forever, daemon=True).start()
tmp = tempfile.mkdtemp()
for s in SCENES:
    png = os.path.join(tmp, s + '.png')
    subprocess.run([CHROME, '--headless=new', '--use-angle=swiftshader', '--enable-unsafe-swiftshader', '--hide-scrollbars',
                    '--virtual-time-budget=8000', '--window-size=960,600', f'--user-data-dir={tmp}/chr', f'--screenshot={png}',
                    f'http://127.0.0.1:8099/scripts/sim-illus/?s={s}'], capture_output=True)
    im = Image.open(png).convert('RGB').crop((0, 0, 960, 600)).resize((640, 400), Image.LANCZOS)
    dst = os.path.join(OUT, s + '.webp'); im.save(dst, 'WEBP', quality=82, method=6)
    print(dst, os.path.getsize(dst), 'byte')
srv.shutdown()
