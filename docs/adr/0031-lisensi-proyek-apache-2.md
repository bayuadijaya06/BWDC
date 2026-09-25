# ADR-0031 — Lisensi proyek: Apache-2.0

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-24
- **Pengganti dari / digantikan oleh:** -
- **Dokumen terkait:** `README.md` §13, `frontend/package.json`, Q-023

## Konteks

Pemilik proyek (BSA) menentukan pemegang hak cipta. Q-023 menanyakan jenis lisensi dan apakah kontribusi pihak ketiga diterima.

## Keputusan

Pilih **Apache-2.0** dengan pemegang hak cipta **BSA**. Lisensi ini standar de-facto untuk backend enterprise Go (Gin, pgx, Kubernetes ekosistem). Memberi kebebasan penggunaan dan modifikasi, plus perlindungan paten eksplisit. Kontribusi pihak ketiga diterima dengan konvensi `sign-off` pada commit.

Buat berkas `LICENSE` di root repo berisi teks lengkap Apache-2.0. Isi field `license` di `frontend/package.json` = `"Apache-2.0"`. Perbarui `README.md` §13 dengan penunjuk ke `LICENSE`.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| BSD-3 / Proprietary | Terlalu restriktif; menghambat kontribusi internal dan ecosystem. |
| MIT | Tidak ada perlindungan paten; kurang standar untuk enterprise backend. |
| GPL-3.0 | Copyleft memaksa semua turunan open source; tidak sesuai model bisnis BSA. |

## Konsekuensi

- **Positif:** standar industri; perlindungan paten; tidak menghalangi kontribusi.
- **Negatif:** tidak ada yang baru (lisensi sudah standar).
- **Mitigasi:** berkas `LICENSE` ada di root; field `package.json` diselaraskan.

## Bukti / Referensi

- `README.md` §13 — kontributor guidelines + penunjuk lisensi.
- `frontend/package.json` — field `license`.
- Q-023 di `OPEN-QUESTIONS.md`.
