# ADR-0016 — Arah rollback `request_revision`: kembali ke step sebelumnya

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-18 (diputuskan pada sesi P-014, menutup temuan `AUDIT-001` C-022)
- **Pengganti dari / digantikan oleh:** -
- **Dokumen terkait:** `docs/design/43-WORKFLOW.md` §4.5, `docs/design/20-SRS.md` FR-WF-09, `docs/design/42-API.md` §5, `docs/design/41-DATABASE.md` §2.4, ADR-0015 (guard transisi), ADR-0012 (overdue turunan)

## Konteks

Dua dokumen menetapkan perilaku berbeda untuk aksi `request_revision` pada satu step:

- `20-SRS.md` FR-WF-09 (prioritas **High**): "Action: Request Revision → dokumen Revision Required, workflow **kembali ke step sebelumnya**".
- `43-WORKFLOW.md` §4.5: "Optionally roll back to previous step or reset to step 1 … Configurable per workflow definition … **Default: reset to step 1**".

Perbedaannya terlihat langsung oleh user dan mengubah beban approval: dengan "step sebelumnya", reviewer yang meminta revisi bertanggung jawab me-review ulang hasilnya sendiri; dengan "step 1", dokumen yang revisinya selesai harus melewati **semua** approval dari awal. Karena FR-WF-09 adalah requirement High, implementasi tidak boleh memilih sendiri — dan sampai ADR ini, tidak ada dokumen yang berhak memutuskan.

Catatan teknis: keduanya dapat diimplementasikan pada skema MVP tanpa kolom baru — `current_step = current_step - 1` vs `current_step = 1` — karena `current_step` adalah integer per instance dan riwayat step hidup di `workflow_actions`. Yang hilang hanyalah keputusannya.

## Keputusan

**Aksi `request_revision` mengembalikan instance ke step sebelumnya (`current_step = current_step - 1`), sesuai FR-WF-09 apa adanya.** Dokumen berstatus `revision_required`, instance **tetap `running`**, dan `current_step_deadline` dihitung ulang untuk step tujuan dari `deadline_days` miliknya.

Aturan yang mengikat:

1. **Batas bawah: step 1.** Pada step 1, `current_step - 1` tidak ada; rollback tidak menurunkan `current_step` di bawah 1. Instance tetap di step 1 — status dokumen berubah `revision_required`, deadline step 1 dihitung ulang, pemilik step (step yang sama) diberi tahu lagi. Tidak ada status instance baru; "menunggu perbaikan owner" adalah kondisi `revision_required` pada dokumen, bukan status instance.
2. **Re-submit tidak membuat instance baru.** Setelah owner mengunggah versi baru, review dilanjutkan pada instance yang sama (`POST /workflows/submit` hanya untuk dokumen `draft` yang belum punya instance). Dua jalur re-entry tidak pernah ada bersamaan. Endpoint untuk re-entry ini adalah **tugas baru `T-028`**, bukan bagian ADR ini.
3. **Guard tidak berubah.** Transisi rollback tetap satu conditional UPDATE ADR-0015: `WHERE id AND version AND status = 'running' AND current_step = <step yang divalidasi>`; `version = version + 1`; `rowsAffected = 0` → rollback + `409 WORKFLOW_CONFLICT`.
4. **"Reset ke step 1" bukan perilaku bawaan dan tidak ditambahkan sekarang.** Bila kelak dibutuhkan per definisi, ia butuh kolom di `workflow_steps`/`workflow_definitions` dan ADR baru — bukan opsi diam-diam di kode.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Default "reset ke step 1" (bentuk lama `43-WORKFLOW.md` §4.5) | Menentang FR-WF-09 yang merupakan requirement High; owner dokumen yang revisinya kecil tetap harus melewati seluruh rantai approval lagi, sehingga beban approval membengkak tanpa nilai |
| Konfigurable per workflow definition (bentuk lama §4.5 juga) | Menambah permukaan desain (kolom + UI + validasi) untuk kebutuhan yang belum pernah diminta requirement mana pun; FR-WF-03 sudah menetapkan field step dan tidak memuat opsi ini |
| FR-WF-09 diubah menjadi "kembali ke step yang ditentukan definisi" | Mengubah requirement High tanpa permintaan pemilik requirement; alternatif ini hanya masuk akal bila kelak ada kasus nyata, dan saat itu pula dibuat ADR-nya |
| `request_revision` menyelesaikan instance (mis. status `revision_required` baru untuk instance) | Menambah nilai `status` instance baru di luar kanonik `running/completed/rejected` (sudah diperiksa konsisten di `AUDIT-001` §5), dan membuat re-entry butuh status ketiga yang tidak punya pemilik di dokumen mana pun |

## Konsekuensi

- Positif: satu perilaku untuk satu aksi di seluruh dokumen; beban approval proporsional (yang meminta revisi me-review ulang hasilnya); `workflow_instance` tetap punya tepat tiga nilai status kanonik; tidak ada perubahan skema.
- Negatif / risiko: rollback mencampur dua tanggung jawab dalam satu handler — memilih tujuan (baru) dan menghitung state (lama); step 1 punya kasus tepi yang berbeda dari step lain.
- Mitigasi: tujuan rollback ditetapkan **sebelum** validasi guard (urutan langkah `43-WORKFLOW.md` §4.2 tidak berubah), dan kasus tepi step 1 diuji eksplisit (`70-TESTING.md` §3.4).

## Bukti / Referensi

- Bukti kontradiksi: `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` C-022.
- Requirement: `20-SRS.md` FR-WF-09 (High).
- Guard yang dipertahankan: ADR-0015; deteksi overdue step tujuan: ADR-0012 + kolom `current_step_deadline`.
