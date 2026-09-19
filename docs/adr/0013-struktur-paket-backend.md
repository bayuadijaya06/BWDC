# ADR-0013 — Struktur paket backend: satu package per folder, tanpa subfolder per modul

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-18 (diputuskan pada sesi P-008, menutup temuan `AUDIT-001` C-002)
- **Pengganti dari / digantikan oleh:** -
- **Dokumen terkait:** `docs/design/40-TSD.md` §2.0, `docs/design/30-ARCHITECTURE.md` §3.2, `docs/design/01-AGENT-WORKFRAME.md` §6, `docs/design/90-AGENT-GUIDE.md` §2.1, `docs/design/70-TESTING.md` §2.1

## Konteks

Struktur folder backend punya **tiga versi berbeda** yang semuanya tampak resmi:

- `30-ARCHITECTURE.md` §3.2: subfolder per modul (`internal/handler/auth/`, `internal/service/document/`, `internal/repository/postgres/`, `internal/pkg/audit/`, `internal/pkg/notification/`).
- `01-AGENT-WORKFRAME.md` §6 dan `90-AGENT-GUIDE.md` §2.1: flat (`internal/handler/`, `internal/service/`, `internal/repository/`), dengan perbedaan `pkg/` ada di satu dokumen dan tidak di dokumen lain.
- `40-TSD.md` §2: flat, dengan contoh kode `package handler`, `package service`, `package repository`, dan satu-satunya yang memuat `internal/bootstrap` (ADR-0010).

Selain itu `30-ARCHITECTURE.md` menempatkan `audit` dan `notification` **dua kali**: sebagai `internal/pkg/audit` dan sebagai `internal/service/audit` — padahal keduanya adalah layanan domain yang menulis data. Tanpa keputusan tunggal, agen berikutnya akan membuat import path yang berbeda dari agen sebelumnya, dan `go build` akan mulai bergantung pada konvensi masing-masing orang.

## Keputusan

**`40-TSD.md` §2 adalah satu-satunya definisi struktur folder backend.** Strukturnya flat: **satu folder = satu package Go**, nama package sama dengan nama folder. Tidak ada subfolder per modul di bawah `handler/`, `service/`, atau `repository/`.

Detail yang mengikat:

1. `internal/` berisi tepat: `bootstrap/`, `config/`, `middleware/`, `model/`, `dto/`, `repository/`, `service/`, `handler/`, `migration/` (berkas `.sql`, bukan paket Go), dan `pkg/`.
2. **Penamaan berkas:** `<modul>_<peran>.go` — mis. `document_handler.go`, `document_service.go`, `document_repository.go`, `workflow_service.go`; test `<nama>_test.go` di folder yang sama dengan package `_test`.
3. **`model/`** satu berkas per entitas (`document.go`, `workflow.go`, `task.go`); **`dto/`** satu berkas per modul (`document_dto.go`).
4. **`internal/pkg/` hanya berisi infrastruktur tanpa aturan domain**: `jwt/`, `filestorage/`, `response/`. Dilarang menaruh layanan domain di sini. Karena itu `pkg/audit/` dan `pkg/notification/` **dihapus** dari rencana; audit adalah `service/audit_service.go` (ADR-0011) dan notifikasi adalah `service/notification_service.go`.
5. **`internal/repository/`** memuat interface dan implementasi `pgx` di package yang sama (sesuai contoh `40-TSD.md` §2.5). Tidak ada subfolder `postgres/` maupun `interfaces/`.
6. **Batas dependensi tetap satu arah:** `handler → service → repository → model`. Service boleh memanggil service lain (mis. `audit`, `notification`) — inilah alasan struktur flat dipilih.
7. **`internal/bootstrap/`** (ADR-0010) tetap menjadi bagian struktur, dipanggil dari `cmd/server/main.go` setelah migrasi dan sebelum HTTP server melayani request.
8. Dokumen lain **tidak boleh menyalin ulang pohon ini**; mereka menyebut path folder lalu menaut ke `40-TSD.md` §2.0.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Subfolder per modul (`internal/service/document/`) seperti `30-ARCHITECTURE.md` §3.2 | 11 modul × 3 layer = 33 paket kecil dengan satu-dua berkas; memunculkan risiko **import cycle** antar-service (document ↔ workflow ↔ notification saling memanggil, dan memecahnya butuh interface tambahan); sebagian besar dokumen lain (`40-TSD.md` §2, `01-AGENT-WORKFRAME.md` §6, `02-AGENT-PROGRESS-PROTOCOL.md`, `TRACEABILITY.md`) sudah memakai path flat, sehingga perpindahan ini justru menambah churn |
| `internal/pkg/audit` + `internal/pkg/notification` untuk layanan domain | Layanan domain menulis tabel dan mengikuti aturan bisnis; menaruhnya di `pkg` membuat lapisan tidak lagi dapat ditebak dan bertentangan dengan ADR-0011 (audit ditulis service, dalam transaksi) |
| `internal/repository/postgres/` + `internal/repository/interfaces/` | Memisahkan interface dari satu-satunya implementasinya tanpa manfaat; `40-TSD.md` §2.5 sudah mencontohkan keduanya dalam satu package dan memudahkan mock test |
| Mempertahankan tiga versi dan "menyerahkan ke agen berikutnya" | Persis kondisi yang menimbulkan temuan C-002; setiap sesi akan menghasilkan import path yang berbeda |

## Konsekuensi

- Positif: satu sumber kebenaran yang dapat dirujuk; import path dapat ditebak; tidak ada risiko import cycle antar modul; penambahan modul = menambah berkas, bukan struktur.
- Negatif / risiko: `internal/service/` dan `internal/handler/` akan berisi banyak berkas dan tumbuh seiring jumlah modul; disiplin penamaan berkas menjadi penting karena nama package tidak lagi memisahkan modul.
- Mitigasi: aturan penamaan berkas (`<modul>_<peran>.go`) dan target maksimum "satu berkas per peran per modul"; bila sebuah berkas melewati ~500 baris, pecah **di dalam package yang sama** dengan sufiks peran (`document_service_upload.go`), bukan dengan membuat subfolder baru tanpa ADR.

## Bukti / Referensi

- Temuan: `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` C-002.
- Dokumen yang diselaraskan pada sesi P-008: `40-TSD.md` §2.0 (pohon kanonik baru + aturan penamaan), `30-ARCHITECTURE.md` §3.2 (pohon diganti tautan), `01-AGENT-WORKFRAME.md` §6, `90-AGENT-GUIDE.md` §2.1, `70-TESTING.md` §2.1/§3.1 (path test), `12-DEVELOPMENT-WORKFLOW.md` §3 langkah 1, `02-AGENT-PROGRESS-PROTOCOL.md` contoh entri.
- ADR terkait: ADR-0008 (lock-in library backend), ADR-0010 (`internal/bootstrap`), ADR-0011 (audit di service).
