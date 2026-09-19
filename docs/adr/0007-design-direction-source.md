# ADR-0007 — Sumber arah desain (`DESIGN.md`)

- **Status:** PROPOSED (menunggu keputusan user, lihat `docs/progress/OPEN-QUESTIONS.md` Q-002)
- **Tanggal:** 2026-09-17
- **Dokumen terkait:** `DESIGN.md`, `docs/design/11-DESIGN-DIRECTION.md`, `antislop.md` R-37

## Konteks

antislop R-37 mewajibkan arah desain dimuat sebelum membangun UI. Saat ini `DESIGN.md` ada tetapi **belum diisi**: identitas, palet, tipografi, dan mood masih kosong. `11-DESIGN-DIRECTION.md` adalah kuesioner yang menampung jawaban user, `DESIGN.md` adalah artefak final yang dibaca agen saat membangun UI.

## Keputusan

**Belum diputuskan.** Tiga jalur yang tersedia:

1. **User mengisi sendiri (rekomendasi).** Arah desain adalah identitas pemilik produk; hasil paling spesifik dan paling kecil kemungkinan terasa generik.
2. **Agen mengisi dengan peringatan.** Agen menyusun draf arah desain, disertai pernyataan eksplisit bahwa itu selera default AI dan hasilnya berstatus draft. Tidak boleh dipresentasikan sebagai final.
3. **Dilewati.** UI dibangun tanpa arah, wajib berlabel "draft without direction" dengan dials ENERGY 1 / RHYTHM 1 / MOTION 1, dan tidak dianggap deliverable.

Agen hanya boleh mengubah `DESIGN.md` setelah user memilih salah satu jalur di atas.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Agen mengisi `DESIGN.md` diam-diam lalu melanjutkan | Melanggar R-37 dan R-23 (mengarang aset/identitas tanpa instruksi) |
| Memakai template UI populer sebagai pengganti arah | Melanggar R-30 (kloning produk populer) |

## Konsekuensi

- Positif: larangan "desain steril tanpa arah" ditegakkan oleh dokumen, bukan oleh niat.
- Negatif: seluruh Phase 4 tertahan sampai ada arah.
- Mitigasi: pekerjaan backend dan penyiapan data tidak terpengaruh; dial sementara di `01-AGENT-WORKFRAME.md` §3.3 tetap boleh dipakai untuk perencanaan, tidak untuk build.

## Bukti / Referensi

`DESIGN.md`, `docs/design/11-DESIGN-DIRECTION.md`, `docs/progress/OPEN-QUESTIONS.md` Q-002.
