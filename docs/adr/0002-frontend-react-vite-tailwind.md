# ADR-0002 — Frontend React 18 + TypeScript + Vite + TailwindCSS

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-17
- **Dokumen terkait:** `docs/design/30-ARCHITECTURE.md` §2, `docs/design/50-FSD.md`, `docs/design/51-UX.md`

## Konteks

UI BWDCS adalah aplikasi internal data-heavy: banyak tabel, form, permission-dependent rendering, dan banyak state (loading, empty, error, aksi gagal). Tim butuh stack yang umum, mudah di-review, dan cepat di-build tanpa layanan eksternal.

## Keputusan

Frontend berupa SPA **React 18 + TypeScript**, di-build dengan **Vite**, di-style dengan **TailwindCSS**. Hasil build adalah aset statis yang dilayani oleh binary Go atau reverse proxy, tanpa dependensi CDN pihak ketiga.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Next.js / SSR | Server Node terpisah menambah komponen deployment; SPA sudah cukup karena aplikasi internal di belakang login |
| Vue / Svelte | Tidak ada keunggulan yang berarti untuk kebutuhan ini; React dipilih agar ekosistem tabel/form lebih matang |
| CSS-in-JS runtime | Biaya runtime dan fragmentasi style; Tailwind + token desain lebih mudah dijaga konsisten |

## Konsekuensi

- Positif: satu bahasa (TypeScript) untuk lapisan UI, build cepat, hasil statis mudah di-deploy.
- Negatif / risiko: Tailwind mudah dipakai berlebihan sehingga menghasilkan UI seragam/tanpa identitas.
- Mitigasi: arah visual diambil dari `DESIGN.md` dan disaring lewat antislop (`antislop.md`); dials ENERGY/RHYTHM/MOTION dijaga konsisten (`01-AGENT-WORKFRAME.md` §3.3 setelah arah desain final).

## Bukti / Referensi

`docs/design/00-README.md` tabel arsitektur, `docs/design/51-UX.md` §5 (component library).
