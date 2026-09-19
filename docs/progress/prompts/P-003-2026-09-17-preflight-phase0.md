# P-003 — 2026-09-17 — Pre-flight Phase 0 (Pemeriksaan Kesiapan Sebelum Kerja)

| Field | Isi |
|---|---|
| ID | P-003 |
| Waktu mulai | 2026-09-17 |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | pra-Phase 0 |
| Task terkait | prasyarat `T-001`-`T-008` |
| Status akhir | DONE |

---

## 1. Prompt User

> "Ada yang harus dikonfirmasi ke saya atau harus anda tambahkan sebelum lanjut ke pengerjaan?"

## 2. Interpretasi & Scope

- Yang diminta: periksa kesiapan dokumen sebelum implementasi dimulai; pisahkan antara (a) keputusan yang butuh jawaban user, dan (b) hal yang masih harus ditambahkan agen.
- Yang TIDAK termasuk: memulai implementasi Phase 0, menginisialisasi git, mengubah keputusan arsitektur yang sudah `ACCEPTED`.
- Asumsi: keputusan yang sudah tertulis spesifik di `40-TSD.md` dianggap final dan cukup dikunci dalam ADR, bukan ditanyakan ulang ke user.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Pindai dokumen untuk ambiguitas yang menghalangi Phase 0 (library, port, seed, security) | Daftar temuan |
| 2 | Pisahkan: bisa diselesaikan agen vs butuh user | Daftar aksi & daftar pertanyaan |
| 3 | Kunci pilihan teknis yang sudah final di TSD ke dalam ADR + perbaiki dokumen yang masih ambigu | ADR-0008, dokumen selaras |
| 4 | Catat keputusan yang belum ada sebagai pertanyaan/ADR `PROPOSED` | Q-006, Q-007, Q-008, ADR-0009 |
| 5 | Tambahkan checklist pra-kerja ke alur kerja | `12-DEVELOPMENT-WORKFLOW.md` §3.1 |
| 6 | Verifikasi | `BROKEN: 0` |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Pindai `40-TSD`, `30-ARCHITECTURE`, `41-DATABASE`, `44-SECURITY`, `60-DEPLOYMENT`, `80-ROADMAP` | Mencari keputusan yang belum final | 4 temuan nyata (lihat §7) |
| 2 | Membuat ADR-0008 (Gin, pgx langsung, Viper, goose, jwt/v5) | `01-AGENT-WORKFRAME` §2.3 dan `30-ARCHITECTURE` §6 masih menulis "Chi/Gin" dan "SQLX/pgx", artinya belum diputuskan bagi agen baru | Stack terkunci, ADR-0001 dipersempit resmi |
| 3 | Memperbaiki `01-AGENT-WORKFRAME.md` §2.3 dan `30-ARCHITECTURE.md` §6 | Menghapus pilihan ganda yang menyesatkan | Dokumen selaras dengan ADR-0008 |
| 4 | Membuat ADR-0009 (PROPOSED) + Q-006 | `44-SECURITY` mewajibkan invalidasi token saat logout, tetapi mekanismenya tidak ada dan Redis opsional | Keputusan jelas, tidak ditebak |
| 5 | Menambahkan Q-007 (bootstrap organisasi & admin pertama) | `41-DATABASE` hanya menyediakan seed role; env `ADMIN_*` sudah ada tetapi tanpa mekanisme | Blocker `T-004`/`T-005` tercatat |
| 6 | Menambahkan Q-008 (whitelist `.doc/.docx/.pptx`) | Daftar di `44-SECURITY` §4.2 tidak memuat dokumen kantor | Non-blocking, untuk Phase 1 |
| 7 | Menambahkan `12-DEVELOPMENT-WORKFLOW.md` §3.1 (pre-flight checklist) dan §7.1 (port dev 8080/5432/5173 + proxy Vite) | Prasyarat kerja harus terlihat sebelum langkah 0 | Checklist sembilan item |
| 8 | Menandai `T-004` dan `T-005` sebagai BLOCKED dengan blocker spesifik | Agar agen berikutnya tahu bagian mana yang bisa dikerjakan lebih dulu | Papan TASKS akurat |

## 5. File yang Berubah

| File | Jenis | Ringkasan | Requirement terkait |
|---|---|---|---|
| `docs/adr/0008-backend-library-lockin.md` | Added | Lock-in stack backend, mengoreksi istilah di ADR-0001 | NFR-MAIN-01 |
| `docs/adr/0009-logout-token-invalidation.md` | Added | Empat opsi invalidasi token, PROPOSED | FR-AUTH-04 |
| `docs/adr/README.md` | Changed | Index ADR-0008, ADR-0009 | — |
| `docs/design/01-AGENT-WORKFRAME.md` | Changed | Stack backend dikunci ke ADR-0008 | NFR-MAIN-01 |
| `docs/design/30-ARCHITECTURE.md` | Changed | Baris Web Framework & Backend framework dikunci | — |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Changed | §3.1 pre-flight checklist, §7.1 port & dev environment | — |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | Q-006, Q-007, Q-008 + temuan inkonsistensi #6-#9 | — |
| `docs/progress/TASKS.md` | Changed | `T-004` & `T-005` BLOCKED dengan blocker spesifik | — |

## 6. Verifikasi

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `bash scripts/check-doc-links.sh` | `BROKEN referensi dokumen: 0`, `exit=0` | PASS |
| 2 | `grep -n -i "chi\|gin\|viper\|sqlx\|pgx" docs/design/*.md` | Tidak ada lagi pilihan ganda "Chi/Gin" atau "SQLX/pgx"; tugas tersisa hanya di ADR-0008 | PASS |
| 3 | `grep -rn "Q-00[678]" docs/progress docs/adr` | Q-006/Q-007/Q-008 tertaut ke ADR-0009, TASKS, dan pre-flight checklist | PASS (keputusan tidak hilang di antara dokumen) |

- [x] Pemeriksaan referensi dokumen dijalankan
- [x] Tidak ada kode yang berubah, sehingga build/test tidak berlaku pada sesi ini
- [ ] Delivery Gate antislop: tidak dijalankan (tidak ada UI yang dibangun)

## 7. Hasil & Dampak

- Selesai: 4 temuan pra-kerja ditindaklanjuti, 3 di antaranya menjadi pertanyaan eksplisit dan 1 (pilihan library) langsung dikunci; checklist pra-kerja ditambahkan sebagai §3.1.
- Belum selesai: keputusan user (Q-001, Q-002, Q-004, Q-006, Q-007) dan penyediaan skill antislop (Q-003).
- Risiko / utang teknis: `T-004`/`T-005` terblokir; sisa auth (login, middleware, RBAC) tetap dapat dikerjakan untuk menjaga alur.
- Dampak ke dokumen desain: `30-ARCHITECTURE.md` §6 dan `01-AGENT-WORKFRAME.md` §2.3 tidak lagi memuat pilihan ganda library.

## 8. Update Ledger

- [x] `STATE.md`
- [x] `SESSION-LOG.md`
- [x] `CHANGELOG.md`
- [x] `TASKS.md`
- [x] `TRACEABILITY.md` (tidak ada requirement yang mulai dikerjakan)
- [x] `OPEN-QUESTIONS.md`
- [x] ADR (0008 ACCEPTED, 0009 PROPOSED)
- [x] `CONTINUE.md` §0

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Jawab Q-006 dan Q-007 (membuka `T-004`/`T-005`), plus Q-001/Q-002 (membuka Phase 4) | User |
| 2 | Beri izin `git init` (Q-004) agar `T-002` bisa jalan | User |
| 3 | Kerjakan `T-001` (verifikasi tooling) lalu `T-002` (struktur repo) | Agen |
