## Deploy ke CloudPanel (Node.js Site)

Anda menggunakan site tipe **Node.js** di CloudPanel dengan port **3000**. Ini adalah cara paling mudah karena CloudPanel otomatis menjalankan aplikasi via `npm start` dan mengelola reverse proxy.

### 1) Siapkan file di lokal (Windows)

1. **Build Binary Go untuk Linux**:
   Di folder proyek `Veda-Dashboard-Pondokrejo`, jalankan PowerShell:
   ```powershell
   $env:GOOS="linux"
   $env:GOARCH="amd64"
   $env:CGO_ENABLED="0"
   go build -o veda-dashboard .
   ```
   *(Hasilnya: file binary `veda-dashboard`)*

2. **Pastikan `package.json` ada**:
   Saya sudah membuatkan `package.json` yang isinya:
   ```json
   {
     "name": "veda-dashboard-pondokrejo",
     "scripts": {
       "start": "chmod +x ./veda-dashboard && ./veda-dashboard"
     }
   }
   ```
   Ini trik agar CloudPanel mengira ini aplikasi Node.js, padahal menjalankan binary Go.

### 2) Upload ke Server

Masuk ke File Manager CloudPanel atau via SFTP/SSH ke folder root aplikasi Node.js Anda (biasanya di `/home/clasnet-geospasial/htdocs/geospasial.clasnet.my.id/`).

Upload file & folder berikut:
- Binary `veda-dashboard`
- File `package.json`
- Folder `templates/`
- Folder `layers/`
- Folder `img/`
- Folder `data/` (isi file Excel: `penduduk_04_03_2026.xlsx`, dll.)
- File aset: `veda-logo.png`, `clasnet-logo.png`, `sijenggung.geojson`, dll.

**Struktur folder di server harus seperti ini:**
```
/home/clasnet-geospasial/htdocs/geospasial.clasnet.my.id/
├── veda-dashboard       (binary)
├── package.json
├── templates/
├── layers/
├── data/                (file Excel di sini)
├── img/
└── sijenggung.geojson
```

### 3) Konfigurasi Environment di CloudPanel

1. Buka CloudPanel > Sites > `geospasial.clasnet.my.id`.
2. Masuk ke tab **Node.js Settings** (atau **App Settings**).
3. Pastikan **App Port** diset ke `3000`.
4. Tambahkan **Environment Variables**:
   - `HOST`: `127.0.0.1`
   - `PORT`: `3000`
   - `DATA_DIR`: `./data`  *(agar aplikasi baca Excel dari folder data di sebelah binary)*
   - `GIN_MODE`: `release`

### 4) Restart Aplikasi

1. Klik tombol **Restart** di CloudPanel.
2. CloudPanel akan menjalankan `npm start`, yang akan mengeksekusi `./veda-dashboard` di port 3000.
3. Cek website `https://geospasial.clasnet.my.id`.

### Troubleshooting (SSH)

Jika website error 502/500, cek logs via SSH:

```bash
# Masuk folder
cd /home/clasnet-geospasial/htdocs/geospasial.clasnet.my.id/

# Cek apakah binary bisa jalan manual
HOST=127.0.0.1 PORT=3000 DATA_DIR=./data ./veda-dashboard
```

Jika permission denied:
```bash
chmod +x veda-dashboard
```
Lalu restart lagi dari CloudPanel.


