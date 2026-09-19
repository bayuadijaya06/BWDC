# ADR-0011 — Lapisan penulisan audit log: di service, di dalam transaksi yang sama

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-18 (diputuskan pada sesi P-008, menutup temuan `AUDIT-001` C-001)
- **Pengganti dari / digantikan oleh:** -
- **Dokumen terkait:** `docs/design/40-TSD.md` §2.4/§2.6, `docs/design/90-AGENT-GUIDE.md` §3.1, `docs/design/44-SECURITY.md` §6, `docs/design/12-DEVELOPMENT-WORKFLOW.md` §5, `docs/design/20-SRS.md` FR-AUDIT-01..03

## Konteks

Sebelum keputusan ini, tiga dokumen menunjukkan tempat penulisan audit log yang berbeda-beda:

- `40-TSD.md` §2.6 mencontohkan **handler** memanggil `audit.Log` (`// 3. Call audit.Log`) dan menyimpan `audit *service.AuditService` di struct handler.
- `90-AGENT-GUIDE.md` §3.1 mencontohkan **service** memanggil `s.audit.Log(...)`.
- Definition of Done di `12-DEVELOPMENT-WORKFLOW.md` §5 menuntut audit log "di dalam transaksi yang sama", sesuatu yang **mustahil** jika audit ditulis di handler, karena transaksi dimiliki service.
- `IDEA.md` bagian 13 dan `30-ARCHITECTURE.md` §3.3 menempatkan audit di business layer (`service` boleh memanggil `audit.Service`).

Kontradiksi ini bukan kosmetik: audit di handler tidak ikut ter-rollback ketika transaksi gagal, sehingga bisa lahir entri audit untuk perubahan data yang sebenarnya batal. Karena audit trail bersifat append-only (FR-AUDIT-03), entri palsu tidak bisa dihapus dan justru merusak nilai bukti sistem. Pola ini akan disalin ke setiap modul (document, workflow, task, admin), sehingga harus ditetapkan sebelum `T-003` menulis kode pertama.

## Keputusan

**Semua penulisan audit log dilakukan di layer `service`, di dalam transaksi yang sama dengan perubahan datanya. Handler tidak pernah memanggil `AuditService`.**

Detail yang mengikat:

1. **Handler** hanya melakukan: parse & validasi input, memanggil service, memetakan error ke status HTTP, dan menyusun response. Tidak ada pemanggilan audit di handler.
2. **Service** membuka transaksi (`pgx.Tx`), menjalankan perubahan data **dan** `AuditService.Log`, lalu commit. Bila salah satu langkah gagal, seluruh transaksi di-rollback sehingga tidak ada entri audit untuk perubahan yang batal.
3. **`AuditService.Log` menerima `pgx.Tx`** (bukan `*pgxpool.Pool`), supaya penulisan audit ikut transaksi pemanggil. Service lain yang memanggil `AuditService` wajib sudah berada di dalam transaksi.
4. **Repository menerima `pgx.Tx`/`DBTX`** (interface berisi `Exec`, `Query`, `QueryRow`), bukan pool langsung, agar service dapat menggabungkan beberapa operasi dan audit ke dalam satu transaksi.
5. **Aksi read-only yang tetap wajib dicatat** (mis. unduh dokumen, lihat per FR-AUDIT-01) tetap ditulis service; karena tidak ada perubahan data yang harus dilindungi, transaksi singkat tersendiri boleh dipakai.
6. **Middleware tidak menulis audit domain.** Middleware hanya menulis log aplikasi terstruktur (slog) untuk `request completed`, dan tidak boleh membuat entri `audit_logs`.
7. **Kegagalan menulis audit membatalkan transaksi**, bukan diabaikan. Aksi kritis tanpa jejak audit lebih berbahaya daripada request yang gagal dan dapat diulang.
8. Penulisan audit **tidak asynchronous** pada MVP. Queue/worker dapat ditambahkan kelak untuk **notifikasi**, bukan untuk audit.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Audit di handler (seperti contoh lama `40-TSD.md` §2.6) | Tidak ikut rollback saat transaksi gagal; handler harus tahu detail audit; pola berulang di setiap handler; bertentangan dengan DoD yang menuntut transaksi yang sama |
| Audit di repository | Repository menjadi tahu aktor, aksi, dan konteks bisnis; melanggar batas layer `handler → service → repository` |
| Audit setelah commit (fire-and-forget / queue) | Entri audit bisa hilang bila proses berhenti; FR-AUDIT-01/03 tidak terjamin; tidak ada rollback yang bisa memperbaikinya |
| Middleware audit generik (berdasarkan method + path) | Tidak tahu `entity_id`, nama aksi bisnis, atau metadata yang bermakna; menghasilkan entri yang tampak lengkap tetapi tidak dapat dipakai menelusuri keputusan |
| Audit in-memory lalu di-flush berkala | Kehilangan entri saat crash; tidak append-only dalam arti yang dijanjikan |

## Konsekuensi

- Positif: satu pola untuk semua modul; audit selalu konsisten dengan data (atomik); mudah diuji (unit test service dapat memeriksa entri audit dalam transaksi yang sama); handler tetap tipis.
- Negatif / risiko: `service` menjadi tempat yang wajib diperiksa setiap review; transaksi menjadi sedikit lebih panjang (satu INSERT tambahan); kesalahan cara pemakaian `pgx.Tx` (mis. memanggil pool di tengah transaksi) dapat memecah atomicity.
- Mitigasi: butir 3 dan 4 di atas mengunci signature (`pgx.Tx` masuk ke audit dan repository); daftar aksi kritis di `44-SECURITY.md` §6 dipakai sebagai checklist test integration per modul; DoD `12-DEVELOPMENT-WORKFLOW.md` §5 memuat butir audit bertransaksi.

## Bukti / Referensi

- Temuan: `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` C-001.
- Dokumen yang diselaraskan pada sesi P-008: `40-TSD.md` §2.4 (aturan transaksi + audit) dan §2.6 (contoh handler tanpa audit), `90-AGENT-GUIDE.md` §3.1 (contoh service memakai transaksi), `12-DEVELOPMENT-WORKFLOW.md` §5 (butir DoD diperjelas).
- Requirement terkait: `20-SRS.md` FR-AUDIT-01, FR-AUDIT-02, FR-AUDIT-03; `41-DATABASE.md` §2.5 (DDL `audit_logs`).
