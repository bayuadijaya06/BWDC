# ADR-0012 — Status kanonik vs label tampilan, dan "overdue" sebagai turunan

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-18 (diputuskan pada sesi P-008, menutup temuan `AUDIT-001` C-003)
- **Pengganti dari / digantikan oleh:** -
- **Dokumen terkait:** `docs/design/41-DATABASE.md` §2.2/§2.3/§2.5, `docs/design/50-FSD.md` §11, `docs/design/51-UX.md` §2.1, `docs/design/43-WORKFLOW.md` §7, `docs/design/20-SRS.md` FR-TASK-03/06, `IDEA.md` bagian 2.G

## Konteks

Tidak ada satu pun dokumen yang memetakan nilai status kanonik (yang diizinkan `CHECK` constraint) ke label yang tampil di UI. Akibatnya muncul dua masalah nyata:

1. **"Overdue" diperlakukan sebagai status.** `IDEA.md` bagian 2.G menulis `Task = Overdue`, `51-UX.md` menampilkannya sebagai sub-menu, dan `50-FSD.md` §6.1 sebagai sub-halaman — sementara `41-DATABASE.md` §2.5 hanya mengizinkan `open`, `in_progress`, `completed`. `FR-TASK-06` meminta penandaan otomatis, yang mudah disalahartikan sebagai izin menambah nilai status.
2. **Tidak ada aturan pemetaan label**, sehingga badge status bisa menampilkan nilai mentah (`in_review`) atau label yang berbeda antar halaman (`Pending` vs `Pending Review`).

Konsekuensi bila dibiarkan: ada risiko nyata menambahkan `'overdue'` (atau `'pending'`) ke `CHECK` constraint, yang membuat status saling eksklusif padahal overdue adalah kondisi yang bisa terjadi pada task `open` **maupun** `in_progress`, dan menuntut job agar bisa kembali ke nilai sebelumnya.

## Keputusan

**Kolom `status` di database dan API hanya memuat nilai kanonik (snake_case) yang diizinkan `CHECK` constraint. Label manusia hanya ada di frontend. Kondisi "overdue" (dan "pending") adalah turunan yang dihitung, tidak pernah disimpan.**

Detail yang mengikat:

1. **Nilai kanonik** (sumber: DDL `41-DATABASE.md`):
   - `documents.status`: `draft`, `in_review`, `revision_required`, `approved`, `rejected`
   - `tasks.status`: `open`, `in_progress`, `completed`
   - `projects.status`: `active`, `archived`
   - `workflow_instances.status`: `running`, `completed`, `rejected`
2. **Pemetaan kanonik → label → warna** ditulis satu kali di `50-FSD.md` §11. Dokumen dan kode lain menaut ke sana; dilarang mendefinisikan ulang label di tempat lain.
3. **Nilai turunan tidak pernah masuk kolom status dan tidak pernah masuk `CHECK` constraint.** Daftar terlarang: `overdue`, `pending`, `pending_review`, `revision` (sebagai nilai kolom), `late`.
4. **Rumus turunan** (dihitung di server, dihitung ulang setiap request):
   - Task overdue: `due_date IS NOT NULL AND due_date < NOW() AND status <> 'completed'`
   - Step workflow terlambat: `workflow_instances.status = 'running' AND current_step_deadline < NOW()`
   - "Pending approvals": instance `running` atau dokumen `in_review` (kueri agregat, bukan kolom)
5. **`FR-TASK-06` dipenuhi tanpa mengubah status**: penanda/pill "Overdue" pada baris task, notifikasi `TASK_OVERDUE` (FR-NOTIF-01), sub-halaman/filter "Overdue", dan widget dashboard "Overdue Tasks". Job harian yang ada di `80-ROADMAP.md` Phase 3 **hanya mengirim notifikasi**, bukan mengubah status task.
6. **API mengekspos turunan sebagai field read-only terpisah**, mis. `is_overdue` (boolean) pada response task, atau filter kueri `?overdue=true`. Klien tidak boleh mengirim nilai ini; server mengabaikan/menolaknya sebagai input.
7. Badge status **selalu memakai label dari tabel pemetaan**, tidak pernah nilai mentah kolom.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Menambahkan `'overdue'` ke `CHECK (status ...)` | Overdue bisa terjadi pada `open` **dan** `in_progress`, sehingga bukan nilai yang saling eksklusif; butuh job untuk masuk dan keluar dari nilai itu; "kapan overdue berakhir" menjadi ambigu dan data bisa basi |
| Menyimpan kolom `is_overdue` yang diperbarui cron | Dua sumber kebenaran (`due_date` + `status` vs kolom flag); nilai bisa basi di antara dua jalannya cron; perhitungannya sudah murah dan deterministik |
| Menyimpan label tampilan di database | Mengunci bahasa & istilah tampilan di skema; setiap perubahan label menjadi migrasi; melanggar pemisahan data vs presentasi |
| Mengirim nilai kanonik ke UI dan memformatnya di frontend dengan `replace('_', ' ')` | Menghasilkan label yang salah (`In review` vs `In Review`, `Revision required` vs `Revision Required`) dan menyebar aturan tampilan ke setiap komponen |

## Konsekuensi

- Positif: hanya ada satu sumber nilai (DDL) dan satu sumber label (`50-FSD.md` §11); tidak perlu job penentu status; tidak ada data basi; filter "Overdue" selalu akurat tanpa sinkronisasi.
- Negatif / risiko: setiap kueri daftar task harus memuat kondisi overdue (indeks parsial `idx_tasks_due_date ... WHERE status != 'completed'` sudah menutupi ini); perhitungan dilakukan berulang alih-alih dibaca dari kolom.
- Mitigasi: `41-DATABASE.md` §2.5 memuat komentar tegas pada kolom `status` dan merujuk ke `50-FSD.md` §11; `20-SRS.md` FR-TASK-06 diberi klausa "turunan"; `70-TESTING.md` mewajibkan test yang memastikan `'overdue'` ditolak constraint dan `is_overdue` benar pada batas tanggal.

## Bukti / Referensi

- Temuan: `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` C-003 (dan C-019: sub-menu "Pending" vs label "Pending Review").
- Dokumen yang diselaraskan pada sesi P-008: `50-FSD.md` §11 (tabel pemetaan baru) dan §6.1, `51-UX.md` §2.1, `43-WORKFLOW.md` §7, `20-SRS.md` FR-TASK-06, `41-DATABASE.md` §2.5, `IDEA.md` bagian 2.G, `70-TESTING.md` §2.1, `80-ROADMAP.md` Phase 3.
- Requirement terkait: `20-SRS.md` FR-TASK-03, FR-TASK-06, FR-NOTIF-01, FR-DASH-01.
