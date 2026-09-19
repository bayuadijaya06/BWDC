# ADR-0004 — Mekanisme deployment fleksibel (Docker opsional)

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-17
- **Dokumen terkait:** `docs/design/60-DEPLOYMENT.md`, `docs/design/01-AGENT-WORKFRAME.md` §2.2
- **Catatan:** ADR ini menyelesaikan inkonsistensi lama: `20-SRS.md` §2.4 pernah menyatakan "Self-hosted only" dan §4.5 "Deployment: Docker Compose", yang bertentangan dengan `00-README.md` dan `01-AGENT-WORKFRAME.md` §2.2.

## Konteks

Pengguna BWDCS bisa menjalankan aplikasi di laptop untuk development, VPS kecil, atau infrastruktur perusahaan yang sudah punya standar sendiri. Memaksa satu mekanisme deployment menutup beberapa jalur adopsi tanpa manfaat teknis.

## Keputusan

Artefak yang diwajibkan hanya: **satu binary Go** (backend), **satu build aset statis** (frontend), **koneksi PostgreSQL**, dan **storage file yang dapat diakses proses**. Mekanisme menjalankannya bebas: langsung di host, systemd, Docker Compose, atau orkestrasi container. **Docker Compose disediakan sebagai referensi resmi**, bukan syarat.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Docker Compose wajib | Menutup deployment tanpa Docker dan tidak menambah jaminan apa pun untuk MVP |
| Kubernetes-first | Kompleksitas tidak sebanding dengan target 50 concurrent users |
| Paket SaaS terkelola | Melanggar prinsip inti: tidak ada dependensi layanan cloud wajib |

## Konsekuensi

- Positif: jalur adopsi luas, kode tidak terikat asumsi container.
- Negatif / risiko: konfigurasi lingkungan bervariasi, lebih mudah terjadi masalah "jalan di laptop saya".
- Mitigasi: daftar environment variable punya satu sumber tunggal (`60-DEPLOYMENT.md` §2), `.env.example` harus mengikuti, health check wajib (`60-DEPLOYMENT.md` §5).

## Bukti / Referensi

`docs/design/60-DEPLOYMENT.md` §1 dan §2, `IDEA.md` bagian 15 & 16.
