# Peta Sebaran Penduduk Pondokrejo

Aplikasi pemetaan status kesejahteraan penduduk Kalurahan Pondokrejo, Sleman, Yogyakarta.

## Fitur
- **Peta Interaktif**: Menggunakan OpenStreetMap dan Leaflet.js.
- **Marker Cluster**: Mengelompokkan marker untuk performa yang lebih baik.
- **Status Kesejahteraan**: Marker berwarna berdasarkan tingkat kesejahteraan (Desil).
  - Merah: Desil 1 (Sangat Miskin)
  - Oranye: Desil 2 (Miskin)
  - Kuning: Desil 3 (Hampir Miskin)
  - Hijau: Desil 4+ (Mampu)
- **Detail Penduduk**: Popup card menampilkan informasi Kepala Keluarga, Alamat, dan Anggota Keluarga beserta bantuan yang diterima.

## Teknologi
- **Backend**: Golang (Gin Framework)
- **Frontend**: HTML, Tailwind CSS, Leaflet.js
- **Data**: Excel (.xlsx) parsing

## Cara Menjalankan

1.  Pastikan Go terinstall.
2.  Jalankan perintah berikut:

```bash
go mod tidy
go run .
```

3.  Buka browser di `http://localhost:8080`.

## Struktur Data
Aplikasi membaca file `1 KK_ART Pondokrejo.xlsx` dan memetakan kolom-kolom berikut:
- Koordinat (Kolom 44)
- NO KK (Kolom 2)
- ID Desil (Kolom 38)
- Bantuan Sosial (BPNT, PKH, BLT, dll)
