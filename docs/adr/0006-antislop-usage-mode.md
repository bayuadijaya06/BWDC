# ADR-0006 — Mode penggunaan antislop untuk BWDCS

- **Status:** ACCEPTED (2026-09-21, sesi P-037 — user memilih **during**)
- **Tanggal:** 2026-09-17 · **Diputuskan:** 2026-09-21
- **Dokumen terkait:** `antislop.md`, `docs/design/01-AGENT-WORKFRAME.md` §3

## Konteks

`antislop.md` mewajibkan pertanyaan mode dijawab sebelum UI work dimulai: **during** (aturan diterapkan sambil membangun) atau **after** (audit bertingkat prioritas setelah selesai). UI BWDCS belum dibangun sama sekali, sehingga keputusan ini menentukan cara kerja Phase 4.

## Keputusan

**Mode `during`.** Aturan antislop diterapkan sambil membangun: setiap unit UI (shell, primitives, halaman) harus lulus Delivery Gate sebelum dianggap selesai, dan hasilnya dilaporkan sebagai daftar PASS/FAIL dengan bukti konkret per butir. Tidak ada audit `after` terpisah di `anti-slop/`, karena mode ini mencegah slop sebelum masuk, bukan menagihnya di akhir.

Konsekuensi operasional yang mengikat:

1. Setiap sesi UI menutup dengan **laporan Delivery Gate** (Blok 1-4 antislop) di log prompt, bukan dengan ringkasan bebas.
2. Filter yang tersedia adalah `antislop.md` **core** saja: `skills/antislop-*/SKILL.md` belum ada di repo dan agen dilarang mengunduhnya (Q-003).
3. Pemeriksa otomatis tetap berlaku: kontras token diuji `frontend/src/styles/tokens.contrast.test.ts`, dan `bash scripts/check-ledger.sh` menjaga ledger.

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

- Positif: keputusan eksplisit dan tercatat; slop dicegah di titik termurah (saat menulis), bukan diperbaiki setelah 12 halaman jadi.
- Negatif / risiko: setiap unit UI menambah laporan wajib; bila gate dilewati, mode ini kehilangan seluruh nilainya, karena tidak ada audit `after` yang mencegahnya.
- Mitigasi: gate dijalankan **dan** dilaporkan per unit kecil (bukan sekali di akhir seluruh Phase 4), sehingga melewatinya terlihat jelas di log prompt hari itu.

## Bukti / Referensi

`docs/design/01-AGENT-WORKFRAME.md` §3.1, `docs/progress/OPEN-QUESTIONS.md` Q-001.
