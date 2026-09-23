# ADR-0007 — Sumber arah desain (`DESIGN.md`)

- **Status:** ACCEPTED (2026-09-21, sesi P-037 — user memilih **jalur 2** dan memberi izin eksplisit ke agen)
- **Tanggal:** 2026-09-17 · **Diputuskan:** 2026-09-21
- **Dokumen terkait:** `DESIGN.md`, `docs/design/11-DESIGN-DIRECTION.md`, `antislop.md` R-37

## Konteks

antislop R-37 mewajibkan arah desain dimuat sebelum membangun UI. Saat ini `DESIGN.md` ada tetapi **belum diisi**: identitas, palet, tipografi, dan mood masih kosong. `11-DESIGN-DIRECTION.md` adalah kuesioner yang menampung jawaban user, `DESIGN.md` adalah artefak final yang dibaca agen saat membangun UI.

## Keputusan

**Jalur 2: agen mengisi `DESIGN.md` dengan peringatan eksplisit.** User memerintahkan "Isi DESIGN.md, tentukan dengan rekomendasi anda sendiri berdasarkan best practices", sehingga agen menyusun arah desain, menandai `DESIGN.md` sebagai `TERISI` (bukan "dikonfirmasi pemilik produk"), dan menuliskan di kepala dokumen bahwa butir rasa berstatus draf sedangkan butir struktural (dials, aturan kontras, pemisahan warna semantik, motif, aturan mono) dipertanggungjawabkan dan boleh dipertahankan.

Pembagian yang mengikat antara draf dan keputusan: penggantian butir **rasa** (kepribadian, referensi, penamaan tema) tidak menuntut ADR baru; perubahan butir **struktural** (dials, jumlah warna inti, aturan kontras, motif) menuntut ADR baru karena ia mengubah kontrak `DESIGN.md` yang dipakai seluruh halaman.

Tiga jalur yang tersedia (dokumentasi keputusan):

1. **User mengisi sendiri (rekomendasi).** Arah desain adalah identitas pemilik produk; hasil paling spesifik dan paling kecil kemungkinan terasa generik.
2. **Agen mengisi dengan peringatan.** Agen menyusun draf arah desain, disertai pernyataan eksplisit bahwa itu selera default AI dan hasilnya berstatus draft. Tidak boleh dipresentasikan sebagai final.
3. **Dilewati.** UI dibangun tanpa arah, wajib berlabel "draft without direction" dengan dials ENERGY 1 / RHYTHM 1 / MOTION 1, dan tidak dianggap deliverable.

Agen hanya boleh mengubah `DESIGN.md` setelah user memilih salah satu jalur di atas. Syarat itu sudah terpenuhi: izin diberikan pada P-037 untuk jalur 2.

Yang tetap dilarang meski jalur 2 dipilih: mengarang **aset** (logo, avatar, foto, ilustrasi) dan **angka** (statistik, testimoni, jumlah pengguna). `DESIGN.md` §1 menyatakan BWDCS tidak punya berkas logo, jadi identitasnya wordmark teks dan avatar adalah inisial dari `username` nyata (R-23).

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Agen mengisi `DESIGN.md` diam-diam lalu melanjutkan | Melanggar R-37 dan R-23 (mengarang aset/identitas tanpa instruksi) |
| Memakai template UI populer sebagai pengganti arah | Melanggar R-30 (kloning produk populer) |

## Konsekuensi

- Positif: larangan "desain steril tanpa arah" ditegakkan oleh dokumen, bukan oleh niat; Phase 4 terbuka dengan dial resmi (ENERGY 1 / RHYTHM 2 / MOTION 1) alih-alih dial sementara.
- Negatif / risiko: arah yang ditulis agen cenderung selera default AI, dan **setiap halaman yang dibangun di atasnya mewarisi risiko itu**. Bila user kemudian mengganti butir rasa, halaman-halaman itu perlu ditinjau ulang (bukan dibangun ulang: token dan struktur tetap).
- Mitigasi: (a) peringatan tercetak di kepala `DESIGN.md`, (b) butir rasa dan butir struktural dipisahkan supaya revisi rasa tidak membatalkan bukti kontras, (c) kontras token diuji mesin, (d) Delivery Gate tetap dijalankan per unit UI (ADR-0006).

## Bukti / Referensi

`DESIGN.md`, `docs/design/11-DESIGN-DIRECTION.md`, `docs/progress/OPEN-QUESTIONS.md` Q-002.
