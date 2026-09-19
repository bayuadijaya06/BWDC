# CONTINUE.md — Titik Masuk untuk Melanjutkan Pekerjaan

> **Baca file ini PERTAMA** sebelum menyentuh file apa pun. Dokumen ini dirancang untuk **agen atau model apa pun**: kamu tidak diasumsikan punya riwayat percakapan, memori sesi sebelumnya, atau akses ke tool yang sama dengan agen terdahulu.
>
> Ditulis juga sebagai `continue.md`. Nama resmi: **`CONTINUE.md`**.

**Terakhir diperbarui:** 2026-09-19 oleh P-026
**Prompt terakhir yang dijalankan:** `docs/progress/prompts/P-026-2026-09-19-perbaikan-c045-c046-dan-riset-best-practice.md`
**Prompt berikutnya:** `P-027` (nomor tertinggi di `docs/progress/prompts/` + 1)

---

## 0. Blok Snapshot (5 baris, selaras dengan `docs/progress/STATE.md`)

| Item | Nilai |
|---|---|
| Posisi | **Phase 0 selesai; Phase 1 berjalan.** `git init` + struktur repo + skeleton backend + migrasi `001`-`009` + bootstrap admin + modul auth + **modul Project (`T-035`, P-022)** + **modul Document (`T-037`, P-023)** + **database test terpisah (`T-036`, P-024)** + **modul Task (`T-038`, P-025)**: **20 endpoint** hidup (8 project §3 + 7 document §4 + 5 task §6), cakupan data diterapkan **di dalam kueri** (`internal/service/scope.go`: `systemScope` + `projectScopePredicate` untuk project/document, `taskScope` + `taskReadPredicate`/`taskWritePredicate` untuk task — modul pertama dengan cakupan **baca ≠ tulis**, `44-SECURITY.md` §3.1.3), dan penomoran dokumen `{PROJECT_CODE}-{NNN}` dibangkitkan di dalam transaksi (ADR-0017). Next: modul **Comment** (`42-API.md` §7) |
| Task terakhir selesai | P-026 menutup tiga temuan sekaligus: **C-045** (`bindJSON` kini menamai field yang bermasalah untuk UUID/waktu/tipe/JSON rusak di body — berlaku untuk project, document, task; kontraknya di `42-API.md` §12), **C-046** (penyaring `?due_from=`/`?due_to=` sebagai interval **setengah terbuka** `[from, to)` ber-batas RFC 3339, melengkapi kelima penyaring `50-FSD.md` §6.1), dan **C-047** (tabel status fase `80-ROADMAP.md` tertinggal; label fase modul Task diselaraskan ke Phase 3 sesuai roadmap). Riset best practice untuk sembilan temuan yang masih OPEN ada di `OPEN-QUESTIONS.md` §3. Sebelumnya: `T-038` modul Task (lima endpoint `42-API.md` §6: `GET|POST /tasks`, `GET|PATCH /tasks/:id`, `POST /tasks/:id/complete`; `taskScope` dengan dua predikat — baca mengikuti keanggotaan project atau seluruh organisasi untuk Manager/Administrator, tulis hanya task milik Contributor; overdue tetap turunan ADR-0012 dan menjadi penyaring tri-state `?overdue=` di `WHERE`; `?priority=` ditambahkan karena `50-FSD.md` §6.1; test: 3 model + 12 service + 10 handler lewat HTTP; bukti server nyata 201/403/404/409/422 + tujuh entri audit task; dua temuan baru C-045/C-046 dan keputusan Q-017). Sebelumnya: `T-036` database test terpisah `bwdcs_test` (P-024). Sebelumnya: `T-037` modul Document (7 endpoint `42-API.md` §4: metadata, unggah versi, daftar versi, unduh ber-audit, hapus berkaskade; `document_number` `{PROJECT_CODE}-{NNN}` di dalam transaksi; cakupan sama dengan project; test 5 penomoran + 9 service + 1 unit batas + 11 HTTP; bukti HTTP pada server nyata `DOC-UJI-001`/`DOC-UJI-002`, versi `1.0`→`1.1`, unduhan identik, non-anggota `404`, enam entri audit). Sebelumnya: `T-035` modul Project (CRUD project + anggota + cakupan data anggota di kueri; 14 test service + 10 test handler; bukti HTTP pada server nyata: 201/403/404/409/422 sesuai matriks dan cakupan; audit 5 aksi di transaksi yang sama). Sebelumnya: `T-005` modul auth (login, JWT ber-`jti`, middleware `RequirePermission`, rate limit FR-AUTH-06, logout lewat `token_revocations` ADR-0009) + `T-033` test config hermetis pada P-021, sekaligus lima temuan baru **C-033..C-037**. Sebelumnya: `T-004` migrasi `001`-`009` + seed role `008` + bootstrap admin (ADR-0010), **ADR-0018**, temuan C-029..C-032 pada P-020. Sebelumnya P-019: `T-032` trigger append-only `audit_logs` (temuan C-020). Sebelumnya P-018: `T-003` backend skeleton + `T-002`/`T-002a` (git + struktur + `.gitignore`) + `T-011`/`T-012`/`T-013` (PATH, goose, role+database `bwdcs`; CLI goose sejak P-020 dipin ke v3.24.1 — ADR-0018) + `T-031` (temuan C-026/C-027). Sebelumnya: `T-028` kontrak endpoint re-submit (P-017), `T-030` format nomor dokumen ADR-0017 (P-016), `T-029` spec FSD tiga halaman nav (P-015), `T-027` arah rollback `request_revision` ADR-0016 (P-014), `T-026` optimistic locking (P-013), `T-025` (P-012), `T-023`/`T-024` (P-011), `T-022` (P-010), `T-021` (P-009), `T-018`-`T-020` (P-008), `T-016`, `T-015`, `T-001`, `T-000`, `T-008`, dan pre-flight (P-003) |
| Task aktif | tidak ada (sesi P-025 sudah ditutup rapi). Berikutnya: **modul Comment** (Phase 1), lalu **Workflow**; selipan `T-024` (anotasi izin endpoint, kini **40/51**). `T-034` (`/auth/refresh` + `change-password`) menunggu Q-013 |
| Blocker | **UI**: `T-006`/`T-007` menunggu Q-001 (mode antislop) dan Q-002 (`DESIGN.md`). Backend tidak terblokir: modul Phase 1 dapat mulai. Non-blocking tapi menunggu keputusan: **Q-013** (mekanisme pencabutan sesi → `logout_all`/`change-password`/`refresh`), **Q-014** (audit login gagal), **Q-015** (tiga kontrak modul project yang diputuskan agen: owner selalu anggota; pelanggaran cakupan → 404 bukan 403; batas `limit` 1–100), **Q-016** (delapan kontrak modul document: aturan versi major sesudah `revision_required`; daftar versi memakai `document:read`; berkas tidak sah → `422` bukan `413`; owner = pembuat; hapus = kaskade + ditolak saat workflow berjalan; `entity_id` audit = nomor dokumen; project arsip **belum** ditolak; filter kategori/owner/tanggal belum ada), dan sisa temuan lama (Q-010/Q-012) |
| Audit terbuka | `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md`: **47 temuan — 38 FIXED, 9 OPEN** (C-039..C-042 ditutup P-023; **C-038, C-043, C-044** P-024; **C-045, C-046, C-047** P-026). OPEN: C-004 (hapus vs arsip dokumen), C-006 (label role "Reviewer"), C-007 (hierarki role sistem vs project), C-009 (auto-lock akun), C-010 (penugasan step ke user), C-015 (dials desain), C-028 (retensi audit), **C-033** (pencabutan sesi — Q-013), **C-035** (audit login gagal — Q-014). Semuanya menunggu keputusan Anda; tidak ada lagi yang menunggu izin, dan **setiap butir kini punya rekomendasi berdasar best practice** di `OPEN-QUESTIONS.md` §3 |
| Next action | **Modul Comment** (`42-API.md` §7, `50-FSD.md` §7) — dengan asumsi yang dinyatakan di **Q-018** (alternatif: kembali ke Workflow Phase 2 lebih dulu). Tabel `comments` sudah terpasang sejak migrasi `007`, cakupannya mengikuti entitas yang boleh dibaca dan edit/hapus hanya milik sendiri (`44-SECURITY.md` §3.1.3) — tidak ada keputusan baru yang dibutuhkan. Setelah itu **Workflow** (`43-WORKFLOW.md`, ADR-0015/ADR-0016) memakai kolom `documents.status`/`workflow_instance_id` dan handler unggah versi yang sudah ada. Selipan murah: `T-024` (anotasi izin, kini 40/51 endpoint). Checklist: `docs/design/12-DEVELOPMENT-WORKFLOW.md` §3 |

> Blok ini hanya ringkasan. **Detail selalu dari `docs/progress/STATE.md`.** Bila keduanya berbeda, `STATE.md` yang benar dan blok ini yang harus diperbaiki.

---

## 1. Aturan Dasar

1. **Kamu tidak punya memori percakapan sebelumnya.** Jangan berasumsi tahu apa yang sudah dikerjakan; semuanya harus dibaca dari dokumen.
2. **Ledger progress adalah sumber kebenaran status**: `docs/progress/`. Dokumen desain adalah sumber kebenaran spesifikasi. Repo ini adalah sumber kebenaran kondisi nyata.
3. **Jangan menyentuh file sebelum langkah 2 (§2) dan §3 selesai.** Tidak ada pengecualian, termasuk untuk perbaikan "kecil".
4. **Jangan menebak keputusan user.** Kalau ada yang ambigu atau blocking, catat di `docs/progress/OPEN-QUESTIONS.md` dan tanyakan.
5. **Tidak ada akses jaringan tanpa izin user.** Jangan mengunduh skill, template, dependensi baru, atau dokumen dari internet atas inisiatif sendiri. Kalau butuh aset, minta user menyediakannya. Izin eksplisit yang tercatat tetap berlaku untuk pekerjaan yang disetujui (contoh: Q-009/P-018 untuk memasang `goose` dan menarik modul Go `ADR-0008`).
6. **Bukti sebelum klaim.** Status `DONE` hanya sah bila ada perintah yang dijalankan dan ringkasan hasilnya.
7. **Setiap pekerjaan wajib meninggalkan catatan** (protokol: `docs/design/02-AGENT-PROGRESS-PROTOCOL.md`).

---

## 2. Langkah Wajib Sebelum Menyentuh File (urutan tetap)

### 2.1 Dokumen aturan (baca semuanya, urut)

| # | File | Yang kamu cari |
|---|---|---|
| 1 | `AGENTS.md` | Routing task → dokumen, kewajiban update progress |
| 2 | `docs/design/01-AGENT-WORKFRAME.md` | Prinsip kerja, konvensi, checklist kualitas, Delivery Gate |
| 3 | `docs/design/02-AGENT-PROGRESS-PROTOCOL.md` | Kewajiban pencatatan: kapan, apa, format minimum |
| 4 | `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Bootstrap, konvensi git, Definition of Done, perintah verifikasi |

### 2.2 Resume payload (kondisi terkini)

| # | File | Yang kamu cari |
|---|---|---|
| 5 | `docs/progress/STATE.md` | Fase, status modul, environment, next action |
| 6 | `docs/progress/SESSION-LOG.md` | 20 entri terakhir: apa yang baru terjadi |
| 7 | `docs/progress/prompts/` | **3 log prompt terakhir** (`P-###`) untuk detail aksi, file, dan bukti |
| 8 | `docs/progress/TASKS.md` | Task `TODO` / `IN PROGRESS` / `BLOCKED` dan urutannya |
| 9 | `docs/progress/OPEN-QUESTIONS.md` | Yang memblokir dan pertanyaan yang menunggu user |
| 10 | `docs/progress/TRACEABILITY.md` | Requirement mana yang sudah benar-benar tertutup |
| 11 | `docs/progress/CHANGELOG.md` | Perubahan file terakhir dan alasannya |

### 2.2a Fakta lingkungan yang sudah terverifikasi (jangan diperiksa ulang)

| Hal | Fakta |
|---|---|
| PostgreSQL | **16.10 (Postgres.app)** berjalan di `5432`; client 16 sudah di `PATH` (blok `BWDCS toolchain` di `~/.zshrc`). **Role dan database `bwdcs` sudah ada** di instance itu; jangan menyentuh database proyek lain (`finmo`, `glid_gateway`, `restaurant`, `wms`) |
| Go | 1.22.5 di `/usr/local/go/bin/go` (build amd64), sudah di `PATH`. **Batas toolchain:** `go.mod` memakai `go 1.22.5`, jadi ketergantungan yang menuntut Go ≥ 1.23 dipin turun (`jackc/pgx/v5` v5.7.4 **dan** `pressly/goose/v3` v3.24.1) |
| Port | `8080` dipakai `wms-backend` (proyek lain) -> backend dev memakai `8081`; `5173` bebas |
| goose | **v3.24.1** di `~/go/bin/goose`, sudah di `PATH` — sama dengan library yang menjalankan migrasi saat startup (ADR-0018 butir 3) |
| Database | Skema **sudah terpasang**: versi goose **9**, 22 tabel, 104 baris `role_permissions`, 2 trigger append-only, organisasi + admin pertama dari bootstrap |

Detail lengkap: `docs/progress/STATE.md` §2.

### 2.3 Dokumen desain sesuai task

Ambil baris yang sesuai dari tabel routing di `AGENTS.md` (mis. Document module → `40-TSD.md`, `41-DATABASE.md`, `42-API.md`, `50-FSD.md`). Untuk task UI tambahkan `DESIGN.md` dan `51-UX.md`.

Aturan khusus:

- **UI apa pun:** berhenti dulu. Cek `DESIGN.md`. Selama masih berstatus belum diisi, UI hanya boleh dibangun sebagai "draft without direction" dan **tidak boleh dianggap deliverable** (antislop R-37, ADR-0007).
- **Perubahan skema:** baca `docs/adr/0003-postgresql-goose.md` (migrasi wajib satu commit dengan kode).
- **Deployment/config:** `docs/adr/0004-flexible-deployment.md` dan `60-DEPLOYMENT.md` §2 (sumber tunggal environment variable).
- **Storage/file:** `docs/adr/0005-local-file-storage-first.md`.
- **Menulis kode backend apa pun:** ikuti `docs/adr/0011-audit-log-layer.md` (audit hanya di service, satu transaksi) dan `docs/adr/0013-struktur-paket-backend.md` (struktur flat dari `40-TSD.md` §2.0).
- **Status & label di API/UI:** ikuti `docs/adr/0012-status-kanonik-dan-overdue-turunan.md` dan tabel `50-FSD.md` §11.
- **Izin/RBAC & migrasi `008`:** ikuti matriks di `44-SECURITY.md` §3.1 (ADR-0014) — jangan mengarang daftar permission; `resource`/`action` hanya boleh dari kosakata tertutup di sana.
- **Konfigurasi runtime apa pun:** sumbernya hanya `60-DEPLOYMENT.md` §2.1 dan `.env.example` (+ `docker-compose.yml`). Jangan menyalin daftar variabel atau YAML compose ke dokumen lain (temuan C-014).
- **Endpoint API apa pun:** sumber tunggalnya `42-API.md` (13 bab; §9 Audit, §10 Reports, §11 Administration, §12 Error, §13 Swagger). `40-TSD.md` §6 hanya contoh pemasangan route — jangan menyalin daftar endpoint ke sana atau ke dokumen lain (temuan C-011/C-013).
- **Transisi state workflow (approve/reject/request revision):** ikuti `43-WORKFLOW.md` §6 dan ADR-0015. Guard `version` + `status = 'running'` + `current_step` wajib ada di `UPDATE`; `rowsAffected = 0` → rollback transaksi + `409 WORKFLOW_CONFLICT`. Jangan menulis `UPDATE workflow_instances` tanpa guard, dan jangan menambahkan retry otomatis (temuan C-005/C-021/C-023).

### 2.3a Audit yang sedang terbuka (baca sebelum menulis kode)

`docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` memuat 23 kontradiksi antar dokumen. Tiga belas sudah **FIXED** — P-008: C-001 (lapisan audit), C-002 (struktur folder), C-003 (status kanonik/overdue), C-019 (label navigasi); P-009: C-017 (matriks permission), C-008 (akses audit log); P-010: C-014 (konfigurasi runtime); P-011: C-011/C-013 (daftar endpoint); P-012: C-012 (requirement tanpa endpoint); P-013: C-005 (optimistic locking workflow instance), C-021 (`current_step_deadline`), C-023 (izin route aksi workflow). Keputusannya kini mengikat lewat ADR-0011 sampai ADR-0015 — **jangan dibuka kembali tanpa ADR baru**.

Sepuluh temuan **masih OPEN**. Tiga yang paling dekat ke pekerjaan berikutnya:

| ID | Ringkas |
|---|---|
| **C-022** | **Butuh keputusan user:** FR-WF-09 menuntut request revision kembali ke **step sebelumnya**, `43-WORKFLOW.md` §4.5 menetapkan default **reset ke step 1**. Jangan tetapkan salah satu tanpa jawaban |
| C-016 | Format nomor dokumen belum ditetapkan, padahal `document_number` unik per project dan divalidasi |
| C-018 | Halaman di navigasi belum punya spec FSD: Approvals, Reports, dan Administration > Workflows (cakupan diperluas pada P-012) |

Jangan menetapkan pola untuk temuan yang belum diputuskan. Bila user belum memilih, kerjakan hal lain dan tanyakan.

### 2.4 Keputusan yang sudah diambil

Baca `docs/adr/README.md` dan ADR yang relevan: `0001`-`0005` dan `0008`-`0015` berstatus `ACCEPTED` dan **tidak boleh diubah isinya**; hanya `0006` dan `0007` yang masih `PROPOSED` (keduanya soal UI/arah desain).

**Transisi workflow instance (ADR-0015):** semua perubahan state `workflow_instances` — termasuk approve, reject, dan request revision — hanya lewat satu conditional `UPDATE` yang memuat `id`, `version`, `status = 'running'`, dan `current_step` yang sudah divalidasi. `rowsAffected = 0` berarti transaksi **dibatalkan** dan API membalas `409 WORKFLOW_CONFLICT`; tidak ada retry otomatis, dan `version` tidak pernah diisi klien. `current_step_deadline` adalah kolom yang diisi saat submit/step maju, sehingga overdue step tetap turunan (`43-WORKFLOW.md` §6/§7).

Sebelum mulai menulis kode, periksa juga **checklist pra-kerja** di `docs/design/12-DEVELOPMENT-WORKFLOW.md` §3.1: sembilan prasyarat yang harus hijau atau sudah dicatat sebagai pertanyaan.

**Berhenti dan bertanya** bila kamu menemukan pertanyaan yang belum ada jawabannya di `OPEN-QUESTIONS.md`.

---

## 3. Rekonstruksi Posisi (menentukan "sudah sampai mana")

Sebelum melanjutkan, kamu harus bisa menjawab lima pertanyaan ini. Kalau tidak bisa, baca lebih dalam atau tanyakan ke user.

1. Fase roadmap mana yang sedang berjalan, dan apa exit criteria-nya?
2. Task apa yang terakhir selesai, dan **apa buktinya**?
3. Apakah ada pekerjaan yang sudah dimulai tetapi belum selesai (kode setengah jalan, migrasi belum jalan, test gagal)?
4. Apa yang memblokir, dan siapa yang bisa membuka blokir itu (user atau kamu)?
5. Apa perintah verifikasi yang tersedia di lingkungan ini (mis. `go build`, `bash scripts/check-doc-links.sh`)?

Aturan penilaian status:

| Yang kamu temukan | Kesimpulan |
|---|---|
| Status `DONE` + perintah + output | Selesai, jangan diulang |
| Status `DONE` tanpa bukti | Turunkan ke `PARTIAL` (catat di log prompt), lalu verifikasi ulang |
| Status `IN PROGRESS` | Lanjutkan pekerjaan itu, jangan buka pekerjaan baru |
| Status `BLOCKED` karena user | Kerjakan hal lain yang tidak terblokir, dan ingatkan user |
| Status `BLOCKED` karena teknis | Blocker itu yang jadi pekerjaan berikutnya |
| Task tidak ada di `TASKS.md` tapi filenya ada | Jangan berasumsi; catat temuan sebagai task baru dan laporkan |

---

## 4. Memilih Pekerjaan Berikutnya (urutan wajib)

Kerjakan **satu** item ini saja, dari atas:

1. **Blocker user** yang menghalangi pekerjaan lain: laporkan ke user di awal turn.
2. **Task `BLOCKED` yang bukan karena user**: selesaikan blocker teknisnya lebih dulu.
3. **Task `IN PROGRESS`**: teruskan sampai `DONE`. Jangan meninggalkan pekerjaan setengah jadi untuk membuka tugas baru.
4. **Task `TODO` sesuai urutan** di `docs/progress/TASKS.md` §1, mengikuti urutan dependency di `docs/design/12-DEVELOPMENT-WORKFLOW.md` §3 (Phase 0 langkah 0 → 7, lalu Phase 1 → 5).
5. **Tidak ada task yang bisa dikerjakan**: usulkan penambahan task ke user. Jangan mengarang ruang lingkup sendiri.

Batas yang tidak boleh dilewati:

- UI (Phase 4 dan semua halaman frontend) menunggu `DESIGN.md` terisi dan mode antislop dipilih (Q-001, Q-002).
- Jangan mengerjakan requirement dari fase yang lebih jauh sebelum fase saat ini memenuhi exit criteria-nya (`docs/design/80-ROADMAP.md`).
- Satu prompt = satu unit pekerjaan yang bisa divertifikasi. Jangan menggabungkan lima modul dalam satu sesi.

---

## 5. Cara Mengerjakan (ringkas, detail di dokumen aturan)

```
READ    -> §2 di atas + dokumen modul
PLAN    -> tulis rencana di log prompt baru SEBELUM mengubah file
BUILD   -> ikuti konvensi (01-AGENT-WORKFRAME §4.3, 90-AGENT-GUIDE §3)
TEST    -> jalankan verifikasi (12-DEVELOPMENT-WORKFLOW §8)
REVIEW  -> Definition of Done (12-DEVELOPMENT-WORKFLOW §5) + checklist antislop bila UI
REPORT  -> update ledger (02-AGENT-PROGRESS-PROTOCOL §4 langkah 6) + update §0 di file ini
```

Nomor prompt berikutnya: lihat nomor tertinggi di `docs/progress/prompts/`, lalu +1. **Jangan mendaur ulang nomor.**

---

## 6. Wajib Sebelum Turn Ditutup

- [ ] Log prompt baru ada di `docs/progress/prompts/P-###-<tanggal>-<slug>.md` (isi lengkap, bukan kerangka)
- [ ] `docs/progress/CHANGELOG.md` memuat semua file yang diubah/dibuat/dihapus pada sesi ini
- [ ] `docs/progress/STATE.md` mencerminkan kondisi setelah perubahan
- [ ] `docs/progress/SESSION-LOG.md` punya entri baru (terbaru di atas)
- [ ] `docs/progress/TASKS.md` status task benar, bukti terisi bila `DONE`
- [ ] `docs/progress/TRACEABILITY.md` diperbarui bila menyentuh `FR-*`/`NFR-*`
- [ ] `docs/progress/OPEN-QUESTIONS.md` memuat pertanyaan/blocker baru
- [ ] `docs/adr/` dibuat/diperbarui bila ada keputusan arsitektur
- [ ] **§0 di file ini (`CONTINUE.md`) diperbarui** supaya agen berikutnya punya ringkasan yang benar
- [ ] Verifikasi dijalankan, hasilnya direkam (mis. `bash scripts/check-doc-links.sh`, `go build ./...`, `cd backend && make test` — **bukan** `go test` telanjang: tanpa `TEST_DATABASE_URL` test integrasi di-skip)

Format laporan ke user (maksimal 6 baris):

```
Posisi   : <fase> - <task>
Selesai  : <apa> (bukti: <perintah> -> <hasil>)
Berubah  : <file utama>
Blocker  : <ada/tidak>
Next     : <task berikutnya sesuai urutan>
```

---

## 7. Aturan Khusus untuk Agen/Model yang Berbeda

1. **Jangan mengandalkan gaya atau keputusan agen sebelumnya dari ingatan.** Semua yang perlu diketahui ada di dokumen. Kalau tidak ada di dokumen, berarti belum diputuskan.
2. **Tulis log prompt untuk pembaca yang bukan kamu**: kalimat utuh, path lengkap, sebutkan alasan, hindari singkatan pribadi. Log prompt adalah satu-satunya jembatan antar model.
3. **Jangan mengubah isi ADR yang sudah `ACCEPTED`.** Ketidaksepakatan dinyatakan lewat ADR baru yang menandai ADR lama `SUPERSEDED`.
4. **Jangan menghapus riwayat** di `SESSION-LOG.md`, `CHANGELOG.md`, `prompts/`, atau bagian DONE di `TASKS.md`. Koreksi ditulis sebagai entri baru.
5. **Jangan menaikkan status tanpa bisa memverifikasi.** Kalau tool verifikasi tidak tersedia di lingkunganmu, tulis status `PARTIAL` dan catat tool apa yang tidak ada.
6. **Jangan menambah dependensi, folder, atau file baru di luar rencana** tanpa mencatatnya di ADR atau `OPEN-QUESTIONS.md`.
7. **Bahasa:** dokumen dan komentar memakai Bahasa Indonesia; istilah teknis dibiarkan apa adanya. Jangan menerjemahkan nama file, path, atau identifier kode.
8. **Kalau dokumen desain bertentangan dengan kode:** kode dianggap belum selesai kecuali ada ADR yang menyatakan sebaliknya. Perbaiki salah satunya di sesi yang sama dan catat di `CHANGELOG.md`.
9. **Kalau kamu menemukan pekerjaan lama yang salah:** perbaiki, catat sebagai entri koreksi, dan sebutkan log prompt yang dikoreksi. Jangan diamkan.

---

## 8. Larangan Eksplisit

- Memulai UI sebelum `DESIGN.md` terisi dan mode antislop dipilih.
- Memilih mode antislop atau mengisi `DESIGN.md` tanpa persetujuan user.
- Mengarang identitas desain, logo, angka, testimoni, atau klaim keamanan.
- Mengunduh apa pun dari jaringan.
- Menjalankan `git commit`, `git push`, `git init`, atau instalasi paket tanpa permintaan/perizinan user.
- Mengklaim `DONE` tanpa bukti verifikasi.
- Menulis kode baru sambil meninggalkan kerangka log prompt kosong.

---

## 9. Template Prompt untuk User

Gunakan salah satu kalimat ini saat membuka sesi baru dengan agen/model apa pun:

**Aman (lihat dulu, baru kerjakan):**
> "Baca `CONTINUE.md` di root repo. Ikuti langkah §2 dan §3, lalu laporkan posisi terakhir, blocker, dan next action. Tunggu konfirmasi saya sebelum mengubah file."

**Langsung lanjut:**
> "Baca `CONTINUE.md`, lahap dokumen dan log progress sesuai instruksinya, lalu kerjakan next action sesuai urutan sampai selesai. Update ledger progress wajib."

**Fokus satu task:**
> "Baca `CONTINUE.md`, lalu kerjakan task `T-00X` saja sampai status DONE beserta buktinya. Update ledger dan `CONTINUE.md` §0."

**Setelah pindah model/agen:**
> "Kamu agen baru tanpa memori sesi sebelumnya. Baca `CONTINUE.md`, jelaskan posisi proyek dan apa yang akan kamu kerjakan sebelum menyentuh file."
