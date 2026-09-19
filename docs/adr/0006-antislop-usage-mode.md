# ADR-0006 — Mode penggunaan antislop untuk BWDCS

- **Status:** PROPOSED (menunggu keputusan user, lihat `docs/progress/OPEN-QUESTIONS.md` Q-001)
- **Tanggal:** 2026-09-17
- **Dokumen terkait:** `antislop.md`, `docs/design/01-AGENT-WORKFRAME.md` §3

## Konteks

`antislop.md` mewajibkan pertanyaan mode dijawab sebelum UI work dimulai: **during** (aturan diterapkan sambil membangun) atau **after** (audit bertingkat prioritas setelah selesai). UI BWDCS belum dibangun sama sekali, sehingga keputusan ini menentukan cara kerja Phase 4.

## Keputusan

**Belum diputuskan.** Agen tidak boleh memilih sendiri. Sampai dijawab, UI work dilarang dimulai (aturan antislop) dan semua task UI berstatus `BLOCKED`.

## Opsi

| Opsi | Konsekuensi |
|---|---|
| **during** (rekomendasi) | Slop dicegah sejak awal, Delivery Gate dijalankan per modul. Biaya: setiap halaman UI harus lulus gate sebelum dianggap selesai |
| **after** | Pembangunan lebih cepat, tetapi menghasilkan temuan audit bertingkat di `anti-slop/audit-001-YYYY-MM-DD.md` yang harus diperbaiki ulang. Risiko: refactor UI besar di akhir |

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Menganggap "draft dulu, audit nanti" sebagai default | Sama dengan memilih `after` tanpa persetujuan user |
| Memakai antislop hanya pada halaman yang "tampak penting" | Melanggar sifat filter yang berlaku menyeluruh |

## Konsekuensi

- Positif: keputusan eksplisit dan tercatat, tidak ada mode yang dipilih diam-diam.
- Negatif: Phase 4 tertahan sampai user menjawab.
- Mitigasi: pekerjaan non-UI (Phase 0-3) tetap berjalan tanpa terpengaruh.

## Bukti / Referensi

`docs/design/01-AGENT-WORKFRAME.md` §3.1, `docs/progress/OPEN-QUESTIONS.md` Q-001.
