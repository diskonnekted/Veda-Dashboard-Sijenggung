# Dasbor Geospasial Desa Sijenggung Banjarnegara
**VEDA - Visual Economic Data Analytics**

Aplikasi pemetaan status kesejahteraan dan potensi wilayah Desa Sijenggung, Banjarnegara. Sistem ini membantu pemerintah desa dalam pengambilan keputusan berbasis data spasial dan statistik.

## Fitur Utama
- **Peta Sebaran Kesejahteraan**: Visualisasi tingkat kesejahteraan penduduk (Desil 1-4) dengan marker warna-warni.
- **Layer Tematik Lengkap**:
  - **Fasilitas Desa**: Kantor desa, tanah kas desa.
  - **Kesehatan**: Posyandu, Puskesmas, Bidan, Apotik.
  - **Kebencanaan**: Rawan Longsor, Rawan Banjir, EWS, Titik Kejadian Bencana.
  - **Infrastruktur**: Jalan, Jembatan, Irigasi.
  - **Potensi Ekonomi**: UMKM, Wisata, Pertanian.
- **Analitik Data (VEDA Analytics)**:
  - Statistik kemiskinan per dusun.
  - Analisis *Exclusion Error* bantuan sosial (PKH/BPNT).
  - Korelasi tingkat pendidikan vs kesejahteraan.
  - Rekomendasi *Action Plan* strategis.
- **Editor Spasial**: Fitur untuk memperbarui koordinat penduduk dan data spasial secara langsung.

## Teknologi
- **Backend**: Golang (Gin Framework) - High Performance & Low Latency.
- **Frontend**: HTML5, Tailwind CSS, Leaflet.js (Peta Interaktif), Chart.js (Visualisasi Data).
- **Database**: Excel-based Database (Mudah diedit oleh perangkat desa) + GeoJSON.

## Cara Menjalankan (Lokal)

1.  Pastikan Go terinstall.
2.  Jalankan perintah berikut:

```bash
go mod tidy
go run .
```

3.  Buka browser di `http://localhost:8080`.

## Cara Menjalankan (Server/CloudPanel)
Lihat panduan lengkap di [DEPLOY_CLOUDPANEL.md](DEPLOY_CLOUDPANEL.md).

## Struktur Data
Aplikasi membaca data utama dari file Excel di folder `data/`:
- `penduduk_04_03_2026.xlsx`: Data master penduduk.
- `pkh-sijenggung.xlsx`: Data penerima PKH.
- `bpnt-sijenggung.xlsx`: Data penerima BPNT.
- `tanah-sijenggung.xlsx`: Data kepemilikan aset tanah.

---
**Desa Sijenggung, Banjarnegara**
*Menuju Desa Cerdas Berbasis Data*
