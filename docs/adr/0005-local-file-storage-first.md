# ADR-0005 — Abstraksi storage dengan implementasi filesystem lokal lebih dulu

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-17
- **Dokumen terkait:** `docs/design/30-ARCHITECTURE.md`, `docs/design/41-DATABASE.md` §2, `IDEA.md` bagian 10

## Konteks

Binary file dokumen tidak boleh disimpan di dalam PostgreSQL. Pengguna MVP sebagian besar self-hosted tanpa object storage. Di sisi lain, kebutuhan masa depan (S3-compatible) sudah diketahui, sehingga penyimpanan tidak boleh di-hardcode ke satu implementasi.

## Keputusan

Definisikan interface `FileStorage` (put, get, delete, metadata) sejak awal. Implementasi untuk MVP adalah **filesystem lokal** dengan skema path per organisasi/proyek/dokumen/versi. Metadata file (nama asli, MIME, ukuran, checksum, path/key) disimpan di PostgreSQL. Implementasi S3-compatible dibuat kemudian tanpa mengubah service pemanggil.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Object storage wajib sejak MVP | Menambah dependensi infrastruktur yang bertentangan dengan prinsip self-hosted |
| Menyimpan file sebagai `bytea` di database | Membengkakkan database dan backup, buruk untuk file 100 MB (NFR-PERF-04) |
| Path file di-hardcode di service | Membuat migrasi storage nanti menjadi refactor besar |

## Konsekuensi

- Positif: MVP bisa jalan dengan direktori lokal; jalur ke S3 tetap terbuka.
- Negatif / risiko: filesystem lokal menyulitkan deployment multi-node, dan versi imutabel (ADR konsisten dengan `00-README.md` keputusan #2) harus dijaga dari penimpaan file.
- Mitigasi: versi dokumen ditulis ke path baru (tidak menimpa), checksum disimpan dan diverifikasi saat download, backup wajib mencakup direktori storage (`60-DEPLOYMENT.md` §6).

## Bukti / Referensi

`docs/design/30-ARCHITECTURE.md` (abstraksi storage), `docs/design/60-DEPLOYMENT.md` §6.
