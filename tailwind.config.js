// Tailwind dibangun sekali menjadi web/static/css/app.css (lihat scripts/build-css.sh),
// menggantikan cdn.tailwindcss.com yang menyusun CSS di browser setiap halaman dibuka.
// Kelas juga dipakai di Go (mis. toneColor/eventBar di main.go) dan di JS simulasi.
module.exports = {
  content: [
    './web/templates/**/*.html',
    './web/static/sim/**/*.js',
    './main.go',
    './app/**/*.go',
  ],
  theme: { extend: {} },
  plugins: [],
};
