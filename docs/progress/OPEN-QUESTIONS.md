# OPEN-QUESTIONS — Pertanyaan & Keputusan Pending

**Protokol:** `docs/design/02-AGENT-PROGRESS-PROTOCOL.md`
**Aturan:** pertanyaan yang butuh keputusan user tidak boleh ditebak. Catat di sini, tandai sebagai `BLOCKING` atau `NON-BLOCKING`, dan tulis asumsi sementara di kolom "Asumsi sementara" agar pekerjaan lain tetap bisa jalan.

---

## 1. Pertanyaan Terbuka

### Q-001 — Mode penggunaan antislop (BLOCKING untuk UI)

- **Pertanyaan:** antislop dipakai **during** (aturan diterapkan sambil membangun) atau **after** (audit setelah selesai)?
- **Kenapa penting:** `antislop.md` melarang memulai UI work sebelum pertanyaan ini dijawab. Mode menentukan apakah ada audit bertingkat prioritas di akhir.
- **Rekomendasi:** `during`, karena UI BWDCS dibangun dari nol dan lebih murah mencegah slop daripada memperbaikinya.
- **Asumsi sementara:** `during`.
- **Terkait:** ADR-0006, `01-AGENT-WORKFRAME.md` §3.1, task `T-007`.

### Q-002 — Sumber arah desain (BLOCKING untuk UI)

- **Pertanyaan:** siapa yang mengisi arah desain di `DESIGN.md`?
  - (a) User mengisi sendiri (paling baik: identitas, palet, tipografi, mood).
  - (b) Agen mengisi **dengan peringatan eksplisit** bahwa hasilnya cenderung selera default AI, dan hasilnya berlabel draft.
  - (c) Dilewati: semua UI berstatus "draft without direction", dials ENERGY 1 / RHYTHM 1 / MOTION 1, tidak boleh dianggap deliverable.
- **Kenapa penting:** antislop R-37. Tanpa arah, hasil UI cenderung steril dan tidak punya identitas.
- **Rekomendasi:** (a). Jika tidak memungkinkan, (c) lebih jujur daripada (b) yang diam-diam dianggap final.
- **Asumsi sementara:** belum ada; UI belum boleh dimulai.
- **Terkait:** ADR-0007, `DESIGN.md`, task `T-006`.

### Q-003 — Skill antislop belum tersedia (NON-BLOCKING)

- **Temuan:** `AGENTS.md` menunjuk `skills/antislop-ui/SKILL.md`, `skills/antislop-copywriting/SKILL.md`, `skills/antislop-human/SKILL.md`, `skills/antislop-layoutmobile/SKILL.md`, `skills/antislop-code/SKILL.md`, tetapi direktori `skills/` tidak ada di repo.
- **Aturan yang berlaku:** agen tidak boleh mengunduh berkas skill (`skills/<nama>/SKILL.md`) dari jaringan, dan tidak boleh mengarang isinya.
- **Aksi yang diminta:** user menyediakan folder `skills/<nama>/SKILL.md` dari rilis antislop yang sama dengan `antislop.md` versi ini.
- **Sampai itu terjadi:** filter UI hanya memakai `antislop.md` (core). Catat di log prompt bahwa skill tidak dipakai.

### Q-010 — Temuan audit mana yang diperbaiki? → **RESOLVED sebagian** (2026-09-18, sesi P-008 hingga P-017)

- **Konteks:** `AUDIT-001` menemukan 20 kontradiksi (10 S1, 7 S2, 3 S3) dengan bukti `file:line` dan usul resolusi.
- **Keputusan user:** C-001 (lapisan audit log), C-002 (struktur folder backend), C-003 (status kanonik vs label & aturan overdue) diperbaiki lebih dulu, lalu C-017 (matriks permission RBAC).
- **Hasil P-008:** C-001, C-002, C-003 `FIXED` lewat ADR-0011/0012/0013 (`T-018`/`T-019`/`T-020`); C-019 ikut tertutup sebagai efek samping C-003.
- **Hasil P-009:** C-017 `FIXED` lewat ADR-0014 (`T-021`); C-008 (akses audit log) ikut tertutup karena matriks permission tidak boleh ambigu.
- **Hasil P-010:** C-014 `FIXED` (`T-022`) — sumber tunggal konfigurasi runtime; dua duplikasi sekelas ikut dibersihkan.
- **Hasil P-011:** C-011 dan C-013 `FIXED` (`T-023`) — satu jalur untuk definisi workflow, dua endpoint yang hilang dilengkapi, dan `42-API.md` ditetapkan sebagai sumber tunggal daftar endpoint.
- **Hasil P-012:** C-012 `FIXED` (`T-025`) — enam requirement tanpa kontrak endpoint kini punya endpoint, izin, dan kode error.
- **Hasil P-013:** C-005 `FIXED` (`T-026`, ADR-0015) — optimistic locking transisi `workflow_instances` kini punya kolom `version` dan pola guard yang mengikat, plus kolom `current_step_deadline` yang juga hilang dari skema (C-021) dan izin route aksi yang salah pasang (C-023). Saat mengerjakan, ditemukan satu kontradiksi baru: **C-022**.
- **Hasil P-014:** C-022 `FIXED` (`T-027`, ADR-0016) — keputusan user mengikuti rekomendasi: FR-WF-09 diberlakukan apa adanya, `request_revision` kembali ke **step sebelumnya** (batas bawah step 1, instance tetap `running`, re-submit melanjutkan instance yang sama). "Reset ke step 1" ditolak sebagai default maupun opsi konfigurasi; bila kelak dibutuhkan, ia butuh ADR baru.
- **Hasil P-017:** C-025 `FIXED` (`T-028`) — kontrak re-submit setelah revisi ditetapkan (`POST /workflows/instances/:id/resubmit`), "satu aksi per step" berlaku per siklus, dan jeda revisi menolak semua aksi. Ditemukan saat mengerjakan kontrak itu, bukan temuan lama.
- **Hasil P-016:** C-016 `FIXED` (`T-030`, ADR-0017) — format nomor dokumen `{PROJECT_CODE}-{NNN}` dibangkitkan server secara atomik, immutable, tanpa penomoran manual; `projects.code` menjadi permanen. Bagian usul audit "apakah boleh diisi manual" **ditolak** dengan alasan tertulis (dua sumber penomoran = duplikat `409` yang dapat dipicu klien). Sekaligus ditemukan dan ditutup **C-024** (requirement ID hantu `FR-DOC-08`).
- **Hasil P-015:** C-018 `FIXED` (`T-029`) — tiga halaman nav tanpa spec kini punya bagian FSD: Approvals (`50-FSD.md` §5.4, sebagai **view** workflow instance sesuai usul resolusi audit), Reports (§10.6), Administration > Workflows (§10.7). Endpoint daftar `GET /workflows/instances` (dengan `scope=assigned_to_me` untuk antrean) ditambahkan ke `42-API.md` §5 sebagai prasyarat halaman antrean.
- **Hasil P-020:** `T-004` dikerjakan (migrasi `001`-`009` + seed `008` + bootstrap ADR-0010). Saat menjalankannya, **empat temuan baru** muncul dan semuanya ditutup di sesi yang sama: **C-029** (FK `documents.workflow_instance_id` menunjuk tabel yang belum ada saat migrasi `004`), **C-030** (`system_settings` tidak punya rumah di daftar migrasi), **C-031** (badan fungsi PL/pgSQL terpotong pengurai goose), dan **C-032** (ADR-0013 butir 1 tidak dapat dipenuhi bersamaan dengan migrasi-saat-startup — ditutup lewat **ADR-0018**). Tiga yang pertama membuat `goose up` gagal atau skema tidak lengkap. Jumlah temuan menjadi 32 (25 FIXED / 7 OPEN).
- **Hasil P-019:** C-020 `FIXED` (`T-032`) — cuplikan `EXECUTE FUNCTION raise_exception(...)` diganti trigger PL/pgSQL yang benar-benar jalan (fungsi `prevent_audit_modification()` + dua trigger: `UPDATE`/`DELETE` row-level dan `TRUNCATE` statement-level, tolak SQLSTATE `23001`), **diikat ke migrasi `007`** karena sebelumnya tidak ada migrasi yang memasangnya. "Immutable" pada akhirnya berarti sesuatu. Saat mengerjakan, ditemukan **C-028** (butuh keputusan Anda — lihat Q-012) sehingga jumlah temuan menjadi 28.
- **Hasil P-026 (lanjutan, 2026-09-19) — seluruh sisa temuan diputuskan.** User meminta pertanyaan terbuka **dijawab** memakai best practice (§3), bukan diserahkan ulang sebagai pilihan tanpa data. Hasilnya: **C-006, C-007, C-010, C-028 `FIXED`** (penyelarasan dokumen; C-028 + ADR-0020), **C-004, C-009, C-033, C-035 `APPROVED`** lewat **ADR-0019/0021/0022** dengan tugas implementasi `T-039`/`T-040`/`T-041`, dan hanya **C-015 yang tetap OPEN** karena ia keputusan rasa/identitas pemilik produk (Q-002), bukan pilihan teknis. Sebelumnya (P-021–P-025): C-038, C-043, C-044, C-045, C-046, C-047 ditutup. Rinciannya di tabel tindak lanjut `AUDIT-001`.
- **Keputusan user untuk C-022 sudah dijawab (P-014):** mengikuti rekomendasi agen, FR-WF-09 diberlakukan apa adanya — rollback ke step sebelumnya (ADR-0016).
- **Disarankan berikutnya (P-026):** kerjakan implementasi keputusan yang sudah `ACCEPTED` — **`T-039`** (arsip dokumen, ADR-0019), **`T-040`** (pencabutan sesi + `T-034`, ADR-0021), **`T-041`** (login_attempts + auto-lock, ADR-0022) — lalu modul berikutnya (**Comment**, Q-018). Satu-satunya temuan yang masih menunggu Anda adalah **C-015** (arah desain, Q-002).
- **Temuan yang butuh ADR baru bila diperbaiki:** tidak ada lagi. C-004, C-009, C-028, C-033, dan C-035 kini punya ADR (`ADR-0019`/`0020`/`0021`/`0022`); C-006, C-007, dan C-010 cukup diselaraskan. ADR sebelumnya: C-005/C-021/C-023 → ADR-0015, C-022 → ADR-0016, C-016 → ADR-0017, C-032 → ADR-0018, C-017 → ADR-0014, C-001/C-002/C-003 → ADR-0011/0012/0013.
- **Terkait:** `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md`, task `T-016`-`T-021`.

### Q-011 — Perilaku selama jeda `revision_required` (NON-BLOCKING)

Ditemukan saat menetapkan kontrak re-submit (P-017, temuan C-025). Dua hal belum diputuskan dan **tidak** menghalangi pekerjaan yang sudah selesai:

1. **Notifikasi overdue saat jeda.** Selama dokumen `revision_required`, instance tetap `running` dan `current_step_deadline` (ditetapkan saat rollback — ADR-0016) terus berjalan, sehingga rumus turunan di `50-FSD.md` §11.4 dapat menandai step "terlambat" padahal tidak ada seorang pun yang boleh bertindak (aksi ditolak `409` — `43-WORKFLOW.md` §4.6 butir 4). Mengubahnya berarti mengubah rumus yang dikunci ADR-0012 ("tidak boleh diubah tanpa ADR baru").
   - **Rekomendasi:** kecualikan instance yang dokumennya `revision_required` dari deteksi overdue, lewat ADR baru — penambahan satu klausa pada rumus, bukan kolom baru.
2. **Unggah versi baru saat `in_review`.** Kontrak saat ini tidak melarang owner mengunggah versi baru sementara review sedang berjalan; berkas yang sedang dinilai akan berubah tanpa membatalkan siklus. Apakah unggahan saat `in_review` ditolak (hanya boleh saat `draft` dan `revision_required`) atau tetap diizinkan adalah keputusan produk.
   - **Rekomendasi:** tolak saat `in_review` → `409`, karena keputusan reviewer harus selalu mengacu pada berkas yang tidak berubah (FR-VER-03).

- **Asumsi sementara:** tidak ada perubahan yang dilakukan sekarang; kedua perilaku berjalan seperti kontrak P-017 (jeda menolak aksi, deadline tetap berjalan, unggahan belum dibatasi status).
- **Terkait:** `43-WORKFLOW.md` §4.6, `42-API.md` §4/§5, ADR-0012, ADR-0016, temuan C-025, task `T-028`.

### Q-012 — Masa simpan (retention) audit log → **RESOLVED: opsi (b) dengan lantai 12 bulan, lewat ADR-0020** (2026-09-19, P-026)

**Keputusan:** retensi audit adalah **operasi pemeliharaan operator**, bukan kontrol UI: lantai **12 bulan** sebagai konstanta kebijakan, pemangkasan lewat jalur `bwdcs.audit_maintenance` dengan prosedur bernomor di `60-DEPLOYMENT.md` §6.4, trigger append-only tidak pernah dilepas, dan jejak pemangkasan ditulis ke log aplikasi. Butir "Audit log retention" **dihapus** dari `50-FSD.md` §10.5; tidak ada kunci `system_settings` baru (itu justru akan mengulang cacat C-028). Sumber pertimbangan: riset di §3 di bawah (SOC 2/ISO 27001, 1–7 tahun praktik industri). Task: tidak ada — yang dijanjikan adalah prosedur, bukan fitur.

Ditemukan saat memperbaiki C-020 (sesi P-019, temuan C-028). Trigger append-only-lah yang membuat tabrakan ini nyata: audit log kini **benar-benar** tidak dapat diedit/dihapus, sementara `50-FSD.md` §10.5 mencantumkan "Audit log retention" sebagai setting di `/admin/settings` tanpa perilaku, requirement (`20-SRS.md` §3.11), kunci `system_settings` (`41-DATABASE.md` §2.6), maupun endpoint/field apa pun (`42-API.md` §11).

- **Opsi A — buang butir itu dari MVP** (rekomendasi, paling sesuai FR-AUDIT-03 dan paling murah): `50-FSD.md` §10.5 hanya memuat setting yang benar-benar dipakai (`auth.max_login_attempts`, `auth.lockout_duration_minutes`, `file.max_upload_mb`, `app.name`), dan tidak ada halaman mati.
- **Opsi B — tetapkan retensi sebagai operasi pemeliharaan terjadwal lewat ADR baru:** menentukan masa simpan, siapa/apa yang menjalankannya, dan bagaimana operasi itu sendiri diaudit; secara teknis memakai jalur `bwdcs.audit_maintenance` yang sudah disediakan `44-SECURITY.md` §6 (`SET LOCAL`, per transaksi).
- **Dampak bila tidak dijawab:** halaman Settings (Phase 5) berisiko memuat kontrol tanpa perilaku, atau — bila ditafsirkan sebagai purge — agen menulis `DELETE` yang langsung ditolak SQLSTATE `23001` oleh trigger. Tidak menghalangi `T-004` maupun modul Phase 1-2.
- **Asumsi sementara:** tidak ada perubahan yang dilakukan sekarang; butir di `50-FSD.md` §10.5 dibiarkan apa adanya sambil diberi temuan C-028.
- **Terkait:** `50-FSD.md` §10.5, `44-SECURITY.md` §6/§8, `41-DATABASE.md` §2.6/§4, `70-TESTING.md` §4.3, FR-AUDIT-03, temuan C-020/C-028, task `T-032`.

### Q-013 — Mekanisme pencabutan "seluruh token aktif user" → **RESOLVED: kolom per user `users.tokens_invalid_before`, lewat ADR-0021** (2026-09-19, P-026)

**Keputusan: opsi (B), bukan opsi (A) yang dulu lebih disarankan.** Satu kolom di `users`, satu perbandingan di middleware (`iat < tokens_invalid_before` → `401 TOKEN_REVOKED`), tanpa tabel sesi yang tumbuh. Opsi (A) (`session_id` + daftar sesi) **tidak** dibatalkan sebagai kemungkinan kelak — ia dapat dibangun di atas kolom ini tanpa membatalkannya — tetapi MVP tidak membutuhkan *daftar* sesi, hanya kemampuan mematikannya. Keterbatasan yang diterima dengan sadar: `logout_all` juga mematikan sesi di perangkat lain, dan `change-password` menyelesaikannya dengan **menerbitkan token baru** untuk request yang sedang berjalan (bukan dengan `keepJTI`, yang tidak dinyatakan oleh penanda per user). Sumber pertimbangan: §3 di bawah (denylist `jti` vs penanda per user). Task: **`T-040`** (menyertai `T-034`); sampai selesai, `501 NOT_IMPLEMENTED` tetap benar.

Ditemukan saat mengerjakan `T-005` (P-021, temuan **C-033**). ADR-0009 butir 3 dan `42-API.md` §2 menjanjikan lebih dari yang dapat dilakukan skema: tabel `token_revocations` hanya memuat `jti` yang **sudah** dicabut, dan sistem tidak menyimpan daftar sesi/token aktif. Tiga janji yang bergantung padanya: `logout_all: true`, "cabut seluruh token lain" pada `POST /auth/change-password` (FR-AUTH-09), dan hal yang sama pada reset password Administrator (FR-AUTH-08).

- **Opsi A — `session_id` di dalam token dan di `token_revocations`** (rekomendasi): satu login = satu sesi yang dicabut utuh, `logout_all` mencabut semua sesi user, dan `change-password` mencabut semua sesi kecuali yang sedang dipakai (`keepJTI` menjadi `keepSession`). Menuntut ADR baru + migrasi kolom.
- **Opsi B — kolom `users.tokens_valid_after`**: token dengan `iat` lebih lama ditolak. Paling murah (satu kolom, satu perbandingan), tetapi `logout_all` **juga** mematikan sesi di perangkat lain yang belum melakukan apa-apa, dan "kecuali sesi yang sedang dipakai" pada `change-password` tidak dapat dinyatakan tanpa mekanisme tambahan.
- **Opsi C — turunkan janji**: logout mencabut token yang dipakai saja; `logout_all` dan pencabutan sesi lain dibuang dari MVP, `42-API.md` §2 dan ADR-0009 disesuaikan. Tidak ada perubahan skema.

- **Asumsi sementara yang sudah dijalankan:** `logout_all: true` dibalas `501 NOT_IMPLEMENTED` — bukan 200 dengan efek sebagian (`42-API.md` §2/§12). `change-password` dan `refresh` belum didaftarkan sebagai route (kontraknya tetap tertulis sebagai rencana).
- **Dampak bila tidak dijawab:** FR-AUTH-08/FR-AUTH-09 tetap `TODO`, dan dua baris kontrak di `42-API.md` §2 tetap tidak dapat dijalankan. Tidak menghalangi modul lain.
- **Terkait:** ADR-0009, `42-API.md` §2/§12, `41-DATABASE.md` §2.1, `40-TSD.md` §2.4/§5.2.1, `internal/repository/token_revocation_repository.go`, temuan C-033, task `T-005`/`T-034`.

### Q-014 — Apakah login **gagal** harus masuk audit log? → **RESOLVED: tidak ke `audit_logs`, tetapi ke tabel `login_attempts`, lewat ADR-0022** (2026-09-19, P-026)

**Keputusan: opsi (B) diperluas.** `actor_id` tetap `NOT NULL` — `audit_logs` bermakna "tindakan aktor yang terautentikasi", dan membuatnya nullable akan melemahkan FK tabel inti untuk seluruh modul. Percobaan login (berhasil **dan** gagal) masuk tabel `login_attempts` yang **tidak** ber-FK pada `username_attempted`, justru supaya percobaan atas username yang tidak ada tetap tercatat. FR-AUDIT-01 ditafsirkan tegas: "login" = login **berhasil**; itu **keputusan**, bukan kekurangan, dan kini tertulis di `44-SECURITY.md` §2.3. Tabel yang sama menjadi penghitung auto-lock **C-009**, jadi satu keputusan menutup dua temuan. Task: **`T-041`**.

Ditemukan saat mengerjakan `T-005` (P-021, temuan **C-035**). FR-AUDIT-01 mewajibkan "login" tercatat, tetapi `audit_logs.actor_id` bersifat `NOT NULL REFERENCES users(id)`: percobaan dengan username yang tidak ada tidak punya baris user untuk dirujuk, sehingga percobaan brute force tidak meninggalkan jejak di audit log yang append-only.

- **Opsi A — `actor_id` nullable untuk entri sistem** (rekomendasi): login gagal dicatat dengan `actor_id = NULL`, username di `metadata`; menuntut migrasi + keputusan siapa yang boleh melihat entri tanpa aktor.
- **Opsi B — domain audit terpisah untuk autentikasi** (tabel sendiri, mis. `authentication_events`): jejak tetap append-only tanpa mengubah makna `audit_logs`.
- **Opsi C — turunkan FR-AUDIT-01**: "login" berarti login **berhasil**; kegagalan cukup di log aplikasi (keadaan sekarang).

- **Asumsi sementara:** tidak ada perubahan skema. Kegagalan login dicatat di log aplikasi (username + IP + correlation id) — perilaku yang paling dekat dengan opsi C, tetapi FR-AUDIT-01 **belum** diubah dan batas itu dinyatakan eksplisit di `44-SECURITY.md` §2.3 supaya tidak dikira sudah tertutup.
- **Terkait:** FR-AUDIT-01, `44-SECURITY.md` §2.3/§6, `41-DATABASE.md` §2.5, temuan C-035.

### Q-015 — Tiga kontrak modul project yang diputuskan agen saat implementasi (NON-BLOCKING)

Ditemukan saat mengerjakan `T-035` (P-022). Ketiganya **bukan** pertanyaan yang menghalangi
pekerjaan — implementasinya sudah berjalan dan terdokumentasi — tetapi keputusannya diambil agen
karena dokumen desain belum mengaturnya. Silakan konfirmasi atau koreksi; semuanya reversibel
dalam satu perubahan kecil.

**(1) Owner project selalu menjadi anggota.** `POST /projects` dan pemindahan `owner_id` lewat
`PATCH` menambahkan owner ke `project_members` berrole `owner` di transaksi yang sama, dan
menghapus owner dari anggota ditolak `409 CONFLICT`.

- Alasannya: cakupan `44-SECURITY.md` §3.1.3 membaca `project_members`. Tanpa ini, orang yang baru
  membuat project (atau menerima pemindahan kepemilikan) tidak dapat melihat project-nya sendiri.
- Alternatif yang ditolak: membiarkan `owner_id` di luar keanggotaan, atau memakai `projects.owner_id`
  sebagai jalan masuk tambahan di kueri cakupan (menambah cabang kedua di setiap kueri).
- Yang mungkin Anda inginkan berbeda: kepemilikan pindah **dan** keanggotaan lama dicabut otomatis.
  Sekarang owner lama tetap anggota (role-nya tidak diubah).

**(2) Pelanggaran cakupan dibalas `404 NOT_FOUND`, bukan `403`.** Berlaku untuk project yang bukan
milik anggota maupun milik organisasi lain.

- Alasannya: `403` membedakan "ada tetapi bukan milik Anda" dari "tidak ada", sehingga keberadaan
  resource antartenant dapat dipetakan dari luar.
- Alternatif: `403 FORBIDDEN` dengan pesan "anda bukan anggota project ini" (lebih jelas bagi UI,
  lebih membocorkan informasi). Bila Anda memilih ini, perubahannya ada di satu fungsi pemetaan error.

**(3) Batas pagination `limit` 1–100** (di luar rentang → `422`).

- Sebelumnya `42-API.md` §1 hanya menulis `?page=1&limit=20` tanpa batas atas, sehingga `limit=100000`
  akan diteruskan ke kueri.
- Alternatif: potong diam-diam ke 100 (klien tidak pernah tahu permintaannya tidak dipenuhi).

- **Asumsi sementara:** ketiganya berlaku seperti yang diimplementasikan pada P-022 dan sudah ditulis
  di `42-API.md` §1/§3/§12, `44-SECURITY.md` §3.1.3, dan `50-FSD.md` §3.
- **Terkait:** FR-PROJ-01..07, `T-035`, temuan yang tidak diangkat: keempatnya bukan kontradiksi dokumen.

### Q-016 — Kontrak modul document yang diputuskan agen saat implementasi (NON-BLOCKING)

Ditemukan saat mengerjakan `T-037` (P-023). Sama seperti Q-015: implementasinya sudah berjalan dan
terdokumentasi, tetapi dokumen desain belum mengaturnya, sehingga keputusannya diambil agen agar
pekerjaan tidak berhenti. Semuanya reversibel dalam perubahan kecil.

**(1) Aturan versi berikutnya (FR-VER-02) dibuat deterministik.**

| Keadaan saat unggah | Versi |
|---|---|
| Belum ada versi | `1.0` |
| Unggahan biasa | minor naik (`1.0` → `1.1`) |
| Dokumen berstatus `revision_required` | major naik (`1.1` → `2.0`) |

- Alasannya: `50-FSD.md` §4.2 menulis "next minor or major **based on revision note**", dan teks
  catatan revisi tidak dapat dinilai mesin. Status dokumen sudah menandai bahwa unggahan ini adalah
  jawaban atas permintaan revisi (ADR-0016), jadi ia dipakai sebagai penentunya — tanpa menambah
  field jenis versi di `42-API.md` §4.
- Alternatif yang ditolak: klien mengirim `version_type` (menambah input kontrak dan membuka peluang
  lompatan versi sewenang-wenang); atau selalu minor sehingga `2.0` tidak pernah ada.
- Yang mungkin Anda inginkan berbeda: major hanya bila status `rejected` (bukan `revision_required`),
  atau penomoran major mengikuti jumlah siklus review.

**(2) Daftar versi memakai izin `document:read`, bukan pasangan baru `document_version:read`.**

- Matriks `44-SECURITY.md` §3.1.2 hanya memuat `document_version:upload` dan `:download`; menambah
  pasangan baru berarti menambah baris di matriks **dan** seed migrasi `008` (104 baris).
- Alternatif: menambah `document_version:read` (Y untuk semua role) lewat ADR baru.

**(3) Validasi berkas dibalas `422 VALIDATION_ERROR`, bukan `413 Payload Too Large`.** Batas 100 MB,
MIME dari magic bytes, dan ekstensi dari daftar tertutup `44-SECURITY.md` §4.2; semuanya dengan
`details[].field = "file"`. `42-API.md` §12 tidak memuat `413`, dan menambahkannya berarti menambah
kode status baru untuk satu kasus.

**(4) `owner_id` dokumen adalah pembuat request.** Kontrak `42-API.md` §4 tidak memuat field owner,
dan `50-FSD.md` §4.2 hanya menampilkan Owner sebagai metadata. Dokumen karena itu tidak dapat dibuat
atas nama orang lain (berbeda dari project yang menerima `owner_id`).

**(5) Hapus dokumen = hapus (kaskade baris + berkas), dan ditolak `409` bila workflow berjalan.**
Syarat "no workflow running" di `50-FSD.md` §4.3 ditegakkan dengan memeriksa `workflow_instances`
berstatus `running`. Semantik hapus-vs-arsip dokumen sendiri masih temuan terbuka **C-004** — endpoint
ini mengikuti kontrak yang ada sekarang, bukan memutuskan C-004.

**(6) `entity_id` entri audit dokumen adalah nomor dokumen, bukan UUID.** Mengikuti contoh yang sudah
ada di `42-API.md` §9 (`entity: "document"`, `entity_id: "WEB-001"`). Modul project memakai UUID;
perbedaan ini disengaja agar mengikuti contoh dokumen yang tertulis, dan `document_id` tetap dikirim
di `metadata`.

**(7) Project berstatus `archived` masih menerima dokumen baru.** Tidak ada dokumen desain yang
melarangnya (`FR-DOC-*`, `42-API.md`, `50-FSD.md`, ADR-0017 semuanya diam); komentar kode yang dulu
menjanjikan aturan itu sudah dikoreksi (temuan **C-042**). Response dokumen karena itu mengirim
`project_archived: true/false` supaya UI dapat memutuskan sendiri. **Bila Anda ingin dokumen baru
ditolak di project arsip, itu keputusan Anda** — perubahannya satu pemeriksaan di service (dan satu
kode error baru, mis. `409`).

**(8) Filter `category`, `owner`, dan rentang tanggal belum ada di `GET /documents`.** `50-FSD.md`
§4.1 menyebutkannya untuk UI, tetapi `42-API.md` §4 hanya memuat `project_id`, `status`, `page`,
`limit`, `search`; menambah parameter dilakukan di `42-API.md` lebih dulu, bukan di kode saja.

- **Asumsi sementara:** kedelapan butir berlaku seperti yang diimplementasikan pada P-023 dan sudah
  ditulis di `42-API.md` §4, `44-SECURITY.md` §3.1.3/§4.2, `50-FSD.md` §4.2/§4.3, dan `40-TSD.md` §2.3-§2.6.
- **Terkait:** FR-DOC-01..07, FR-VER-01..06, `T-037`, temuan C-040/C-042 (koreksi yang menyertainya).

### Q-017 — Kontrak modul task yang diputuskan agen, plus dua keputusan yang benar-benar menunggu Anda (NON-BLOCKING)

Ditemukan saat mengerjakan `T-038` (P-025). Butir (1)-(9) berjalan dan sudah tertulis di `42-API.md` §6,
`50-FSD.md` §6, dan `44-SECURITY.md` §3.1.3; semuanya reversibel dalam perubahan kecil. Butir **(10)**
dan **(11)** belum diimplementasikan dan **tidak akan** dikerjakan agen tanpa keputusan Anda (§
"Keputusan yang diminta" di bawah), karena keduanya menambah permukaan kontrak.

**(1) Task selalu lahir `open`; `status` dari klien ditolak `422`, bukan diabaikan.** FR-TASK-03
menetapkan tiga nilai status, tetapi tidak mengatakan apa yang harus terjadi bila klien mengirimnya.
Menolak lebih jujur daripada menerima-lalu-membuang, dan jalan menuju `completed` adalah endpoint
ber-izin berbeda.

**(2) `assignee_id` dan `due_date` wajib pada `POST /tasks`.** `50-FSD.md` §6.2 menandai keduanya
"Required" pada form, sedangkan kolom database (`41-DATABASE.md` §2.5) keduanya nullable. Kontrak
mengikuti FSD: tanpa assignee, penanda overdue (FR-TASK-06) tidak punya pemilik yang jelas.
- Alternatif yang ditolak: membuat keduanya opsional seperti kolomnya — akan memberi task tanpa
  tenggat, padahal halaman Overdue adalah inti kebutuhan (FR-TASK-06).

**(3) `priority` opsional dengan default `medium`.** Nilainya diambil dari default kolom di skema,
bukan angka yang dikarang handler. Alternatif: wajib seperti form FSD (menolak request yang sah
menurut kolomnya) — ditolak karena menghukum klien yang tidak peduli prioritas.

**(4) `PATCH /tasks/:id` tidak dapat memindahkan task antar-project → `409`.** Memindahkan task
mengubah cakupan datanya (siapa yang boleh membacanya berubah) dan kontrak §6 tidak memuat operasi itu.

**(5) `in_progress` → `completed` **ditolak** di `PATCH`; wajib lewat `POST /tasks/:id/complete`.**
Matriks `44-SECURITY.md` §3.1.2 memisahkan `task:update` (Contributor punya) dari `task:complete`
(Contributor punya juga, tetapi Manager+ berbeda konteks), dan satu jalan pintas akan membuat pemisahan
itu tidak berarti. `50-FSD.md` §6.3 memang menyebut tiga aksi terpisah, bukan satu endpoint ubah status.

**(6) `Complete` dari `open` → `409`; `complete` ulang → `200` idempoten tanpa audit baru.** Mengizinkan
`open` → `completed` berarti melewati status `in_progress` yang tidak pernah ada di jejak audit.
Alternatif: membuat `Complete` langsung melompat (lebih ramah, lebih sulit diaudit).

**(7) Izin `task:assign` diperiksa di handler, hanya bila body `PATCH` memuat `assignee_id`.** Ini
route **kedua** yang izinnya bergantung isi body (yang pertama `POST /workflows/instances/:id/actions`),
dan `40-TSD.md` §6 aturan 3 mewajibkan alasannya tertulis — sudah ditulis di `42-API.md` §6. Tanpa itu,
Contributor (punya `task:update`, tidak punya `task:assign`) dapat memindahkan penugasan orang lain.

**(8) Dua penyaring server-side ditambahkan: `?priority=` dan `?overdue=` (tri-state).** `50-FSD.md`
§6.1 memuatnya untuk UI, `42-API.md` §6 hanya memuat tiga penyaring. `?overdue=` bernilai `true` (hanya
overdue), `false` (hanya yang belum overdue), atau tidak dikirim (tidak disaring) — "false = jangan
saring" akan mengembalikan tepat yang tidak diminta. Penyaringan terjadi di `WHERE`, sebelum
`LIMIT`/`OFFSET`, supaya sub-halaman Overdue tidak salah karena dipotong lebih dulu.

**(9) `is_overdue` disajikan sebagai field read-only bernama `is_overdue` pada daftar dan detail.**
`50-FSD.md` §11.4 mengizinkan "field read-only (`is_overdue`) atau filter kueri (`?overdue=true`)" —
keduanya dipakai, dengan rumus yang sama (`model.IsTaskOverdue`, diikat test). Nama field ditetapkan
agen karena dokumen hanya menyebut "field read-only".

**(10) Penyaring rentang tanggal (`due_from`/`due_to`) → DIPUTUSKAN 2026-09-19 (P-026) memakai opsi (a), dengan dasar riset di §4.** `50-FSD.md` §6.1
memuat "Due date range", tidak ada dokumen lain yang menetapkan semantiknya (temuan **C-046**).
Pilihannya:

- **(a) ✓ DIPILIH** — dua parameter RFC 3339, diimplementasikan sebagai interval **setengah terbuka** `[due_from, due_to)`: batas bawah inklusif, batas atas eksklusif. Alasan pilihan ini (bukan inklusif-inklusif seperti usulan awal): rentang bersebelahan (mis. per bulan) tidak tumpang tindih dan tidak melewatkan baris, dan konvensi itu yang dipakai API publik besar — Stripe memakai `created[gte]` + `created[lt]`. Zona waktu eksplisit dari klien, jadi tidak ada tafsir diam-diam.
- (b) Ditolak: parameter tanggal `YYYY-MM-DD` menuntut server menetapkan satu zona waktu tetap dan menafsirkannya sendiri — satu konfigurasi baru plus satu tafsir implisit, padahal klien sudah tahu zona waktunya.
- (c) Ditolak: menyaring di klien atas halaman ber-paginasi menghasilkan hasil yang salah (persis alasan `?overdue=` dibuat server-side).

**(11) Atribusi error untuk UUID yang tidak sah di body (temuan C-045) → DIPUTUSKAN 2026-09-19 (P-026) memakai opsi (a).**
`PATCH /tasks/:id` dengan `{"document_id":"bukan-uuid"}` dijawab `422` ber-`field: "body"` dan pesan
"harus JSON objek yang sah", padahal JSON-nya sah — pesannya menyesatkan dan field-nya tidak disebut.
Penyebabnya `bindJSON` bersama (`internal/handler/project_handler.go`). Pilihannya:

- **(a) ✓ DIPILIH, diperluas** — `bindJSON` bersama kini membedakan tiga sebab: tipe salah (nama field dari `json.UnmarshalTypeError`), JSON rusak (`json.SyntaxError`/`json.Valid` → `body`), dan JSON sah dengan nilai yang tidak dapat diurai. Untuk sebab ketiga, nama field dicari lewat `undecodableField` (reflection atas field yang ada di body) dan pesannya dipilih dari bentuk tipe. Jadi atribusi field **tidak** lagi terbatas pada `UnmarshalTypeError`, seperti yang dulu saya duga pada usulan awal.
- (b) Ditolak untuk sekarang: memindahkan UUID body menjadi `string` menyentuh tiga modul (DTO, handler, test) dan mengubah bentuk input yang sudah berjalan, sementara hasil yang diinginkan sudah tercapai tanpa itu.
- (c) Ditolak: pesannya salah, bukan hanya kurang lengkap.
- **Dasar best practice:** `uuid.UUID` dan `time.Time` adalah `json.Unmarshaler` kustom yang mengembalikan error tanpa nama field (diperiksa langsung pada Go 1.22 di repo ini: `uuid.invalidLengthError`, bukan `UnmarshalTypeError`), dan `encoding/json` memang hanya mengisi `UnmarshalTypeError.Field`. Panduan umumnya: jangan menyerahkan pelaporan field kepada `encoding/json` — validasi eksplisit (bentuk (b)) atau petakan sendiri seperti dilakukan di sini. Kontrak pemetaannya dicatat di `42-API.md` §12.

- **Status:** butir (1)-(9) berlaku seperti yang diimplementasikan pada P-025; butir (10) dan (11)
  **diputuskan dan dikerjakan pada P-026** memakai rekomendasi (a) masing-masing, dengan dasar riset di §4.
  Keduanya tetap reversibel: rentang tanggal dapat diubah menjadi inklusif-inklusif atau `YYYY-MM-DD`,
  dan atribusi error dapat diganti pendekatan (b), tanpa menyentuh skema.
- **Terkait:** FR-TASK-01..07, FR-AUDIT-01, `T-038`, temuan C-045/C-046 (keduanya FIXED pada P-026), `70-TESTING.md` §3.10/§3.11.

### Q-018 — Urutan fase berikutnya: Comment (menutup Phase 3) atau kembali ke Phase 2 Workflow? (NON-BLOCKING)

- **Konteks:** `80-ROADMAP.md` menempatkan Workflow di **Phase 2** dan Task & Comment di **Phase 3**,
  sedangkan Task sudah dikerjakan lebih awal (P-025/P-026). Sekarang tersisa dua kandidat berikutnya.
- **Pertanyaan:** lanjut menutup Phase 3 dengan **Comment** dulu, atau kembali ke urutan roadmap dan
  mengerjakan **Workflow** (Phase 2)?
- **Rekomendasi:** **Comment dulu.** Alasannya: tabel `comments` sudah terpasang sejak migrasi `007`,
  kontrak `42-API.md` §7 sudah ada, cakupannya mengikuti entitas yang boleh dibaca (`44-SECURITY.md`
  §3.1.3) sehingga tidak menuntut keputusan baru, dan menutup Phase 3 berarti satu fase benar-benar
  selesai — lebih jujur daripada meninggalkan dua fase setengah jalan. Workflow (Phase 2) menuntut
  lebih banyak: `43-WORKFLOW.md` + ADR-0015/ADR-0016 sudah terkunci, tetapi engine-nya menyentuh
  `documents.status`, optimistic locking, dan notifikasi.
- **Asumsi sementara:** Comment dikerjakan lebih dulu; bila Anda ingin urutan roadmap dipegang ketat,
  cukup katakan dan Workflow dikerjakan lebih dulu (tidak ada perubahan kode yang terbuang).
- **Terkait:** `80-ROADMAP.md` §1/§3, `TASKS.md` backlog fase 2/3, temuan C-047.

### Q-009 — Izin penyiapan toolchain → **RESOLVED: DIIZINKAN** (2026-09-18, sesi P-018)

- **Temuan awal T-001 sebagian keliru dan sudah dikoreksi (P-005):** yang berjalan di port `5432` bukan PostgreSQL 14.6 Homebrew, melainkan **PostgreSQL 16.10 dari Postgres.app**. Jadi **tidak perlu memasang PostgreSQL sama sekali**. Yang rusak hanya binary server formulasi `postgresql@14` Homebrew (`/usr/local/Cellar`), yang memang tidak akan dipakai.
- **Yang tersisa dibutuhkan:**
  1. `PATH` shell memuat `/usr/local/go/bin`, `$HOME/go/bin`, dan `/Applications/Postgres.app/Contents/Versions/16/bin` (`T-011`).
  2. `goose` dipasang lewat `go install github.com/pressly/goose/v3/cmd/goose@latest` → menulis ke `~/go/bin` dan **butuh jaringan** (`T-012`).
  3. Role + database `bwdcs` dibuat pada instance PostgreSQL 16 yang sudah berjalan (`T-013`). Bersifat aditif: tidak menyentuh database proyek lain (`finmo`, `glid_gateway`, `restaurant`, `wms`) dan tidak menjalankan server baru.
- **Kenapa butuh izin:** ketiganya menulis di luar direktori proyek (profil shell, `~/go/bin`, dan instance database bersama).
- **Alternatif tanpa mengubah home:** memakai path absolut untuk `go`, serta `go run github.com/pressly/goose/v3/cmd/goose@latest` setiap kali migrasi dijalankan (tetap butuh jaringan sekali untuk mengunduh modul).
- **Dampak bila tidak dijawab:** `T-004` (migrasi) dan seluruh pekerjaan backend tidak dapat dijalankan atau diuji.
- **Keputusan user (2026-09-18, P-018):** **izin diberikan.** Agen menjalankan `T-011` (PATH), `T-013` (role + database `bwdcs`), dan `T-012` (`goose`) tanpa perlu bertanya lagi. Batas yang tetap berlaku: tidak menginstal server database baru, tidak menyentuh database proyek lain (`finmo`, `glid_gateway`, `restaurant`, `wms`), dan tidak menjalankan `sudo`.
- **Hasil:** `T-011`/`T-012`/`T-013` DONE pada P-018 — lihat `docs/progress/prompts/P-018-*.md` dan `STATE.md` §2.
- **Terkait:** task `T-011`, `T-012`, `T-013`, `T-014`.

### Q-006 — Strategi invalidasi token saat logout → **RESOLVED** (2026-09-17, sesi P-004)

- **Keputusan:** opsi paling aman yang tidak menambah dependensi runtime: **daftar revokasi `jti` di PostgreSQL**, diperiksa di middleware auth dengan cache in-memory TTL maksimum 30 detik.
- **Alasan:** logout berlaku seketika (memenuhi FR-AUTH-04), tetap benar bila kelak berjalan multi-instance, dan tidak memerlukan Redis. Opsi Redis ditolak karena menambah dependensi wajib; access token pendek saja ditolak karena token lama tetap valid sampai `exp`.
- **Konsekuensi:** tabel `token_revocations` (migrasi `009`), revokasi juga dijalankan saat password diubah/direset dan akun dinonaktifkan, ada jendela maksimum 30 detik sebelum revokasi terlihat oleh instance lain (dicatat di `44-SECURITY.md` §2.2).
- **Bukti:** ADR-0009 (ACCEPTED), `41-DATABASE.md` §2.1 & §4, `44-SECURITY.md` §2.2/§2.3, `40-TSD.md` §2.4 & §5.2.1, `42-API.md` §2.
- **Membuka:** `T-004` (migrasi `009`) dan `T-005` (bagian logout).

### Q-007 — Bootstrap organisasi & admin pertama → **RESOLVED** (2026-09-17, sesi P-004)

- **Keputusan:** opsi paling aman: **bootstrap otomatis dari environment variable saat startup, idempotent, hanya berjalan bila tabel `users` masih kosong**, dijalankan dalam satu transaksi setelah migrasi dan sebelum HTTP server melayani request.
- **Pengaman wajib:** `ADMIN_PASSWORD` minimal 12 karakter dan bukan nilai contoh (`changeme`, `admin`, `password`, `admin123`); bila gagal, startup berhenti dengan pesan yang menyebut variabelnya. Log peringatan meminta operator mengganti password setelah login pertama.
- **Alasan:** instalasi baru langsung dapat dipakai dan dapat diuji otomatis, tanpa kredensial tetap yang tertanam di repositori (opsi migrasi seed ditolak karena mudah terbawa ke produksi).
- **Bukti:** ADR-0010 (ACCEPTED), `41-DATABASE.md` §4.1, `40-TSD.md` §2.7, `60-DEPLOYMENT.md` §2 & §4.2 (termasuk penambahan `ADMIN_ORG_NAME` dan `ADMIN_ORG_CODE`).
- **Membuka:** `T-004` dan bagian login pada `T-005`.

### Q-008 — Kebijakan file upload untuk dokumen kantor (NON-BLOCKING, menyentuh Phase 1)

> Catatan: masih terbuka. Tidak menghalangi Phase 0.

- **Temuan:** `44-SECURITY.md` §4.2 membatasi MIME dan ekstensi ke pdf, txt, csv, xls/xlsx, jpg, jpeg, png. Untuk sistem document control, ketiadaan `.doc`, `.docx`, dan `.pptx` kemungkinan menghambat pemakaian nyata.
- **Opsi:**
  - **A. Tambahkan `.doc`, `.docx`, `.ppt`, `.pptx`** (rekomendasi) dengan validasi magic bytes yang sesuai, plus `application/vnd.openxmlformats-officedocument.*`.
  - **B. Pertahankan daftar sekarang** dan minta pengguna mengonversi ke PDF.
  - **C. Daftar dapat dikonfigurasi admin** lewat `system_settings` (`41-DATABASE.md` sudah punya tabel setelan) sebagai pekerjaan lanjutan.
- **Dampak bila tidak dijawab:** baru terasa di Phase 1 (document module), tidak menghalangi Phase 0.
- **Terkait:** `44-SECURITY.md` §4.2, `50-FSD.md` §4.2, task Phase 1.

### Q-004 — Repositori git belum diinisialisasi → **RESOLVED: DIIZINKAN** (2026-09-18, sesi P-018)

> Mencakup izin `git init` + pembuatan `.gitignore` (task `T-002a`, wajib memuat `.env`) + `.editorconfig`.

- **Temuan:** `git status` gagal (not a git repository). Semua konvensi commit/branch di `12-DEVELOPMENT-WORKFLOW.md` belum bisa dijalankan.
- **Aksi yang diminta:** konfirmasi bahwa `git init` beserta pembuatan `.gitignore`, `.editorconfig`, dan branch default boleh dilakukan sebagai bagian `T-002`.
- **Keputusan user (2026-09-18, P-018):** **izin diberikan.** Agen menjalankan `git init` dengan branch default `main`, membuat `.gitignore` + `.editorconfig`, dan menyusun struktur folder `backend/`, `frontend/`, `scripts/`.
- **Batas yang tetap berlaku:** agen **tidak** membuat commit atas inisiatif sendiri (commit hanya bila user memintanya), tidak mengubah git config, dan tidak menyentuh remote.
- **Hasil:** `T-002`/`T-002a` DONE pada P-018 — lihat `docs/progress/prompts/P-018-*.md`.

### Q-005 — Keputusan duplikat di `01-AGENT-WORKFRAME.md` §7 dan `90-AGENT-GUIDE.md` §5 (RESOLVED)

> Catatan: Q-006 sampai Q-008 di atas muncul dari pemeriksaan pra-kerja Phase 0 (log prompt P-003).

- **Temuan:** dua decision log dengan isi berbeda dan status berbeda (satu ada status `Pending`, satu tidak).
- **Resolusi:** diarahkan ke sumber tunggal `docs/adr/` (lihat `docs/adr/README.md`). Kedua section lama sekarang hanya menunjuk ke sana.

---

## 2. Temuan Inkonsistensi Dokumen

| # | Temuan | Status | Tindakan |
|---|---|---|---|
| 1 | `20-SRS.md` §2.4 menyebut "Self-hosted only" dan §4.5 "Deployment: Docker Compose", sedangkan `00-README.md` + `01-AGENT-WORKFRAME.md` §2.2 + ADR-0004 menyatakan mekanisme deployment tidak dipaksa | RESOLVED | SRS diselaraskan ke ADR-0004 |
| 2 | `20-SRS.md` §2.1 memuat karakter non-Latin yang tercecer di tengah kalimat | RESOLVED | Teks diperbaiki |
| 3 | `80-ROADMAP.md` §3 menampilkan semua fase 100% padahal belum ada kode | RESOLVED | Diganti tabel status nyata + pointer `docs/progress/STATE.md` |
| 4 | Tidak ada daftar environment variable tunggal (tersebar di `60-DEPLOYMENT.md` §2 dan `90-AGENT-GUIDE.md` §7) | RESOLVED | Sumber tunggal ditetapkan `60-DEPLOYMENT.md` §2.1 dengan cermin `.env.example`. Salinan usang di `90-AGENT-GUIDE.md` §7 dan blok YAML compose kedua di `30-ARCHITECTURE.md` §5.1 diganti penunjuk pada sesi P-010 (temuan C-014) |
| 6 | `01-AGENT-WORKFRAME.md` §2.3 dan `30-ARCHITECTURE.md` §6 menulis pilihan library sebagai "Chi/Gin" dan "SQLX/pgx", sedangkan `40-TSD.md` §1 sudah spesifik (`gin`, `pgx/v5`) | RESOLVED | Dikunci lewat ADR-0008, kedua dokumen diarahkan ke ADR tersebut |
| 7 | `44-SECURITY.md` §2 mewajibkan invalidasi token saat logout tetapi tidak ada mekanismenya; Redis bersifat opsional | RESOLVED | Q-006 + ADR-0009 (ACCEPTED) + tabel `token_revocations` |
| 8 | Bootstrap organisasi & admin pertama belum punya mekanisme meski env `ADMIN_*` ada di `60-DEPLOYMENT.md` §2 | RESOLVED | Q-007 + ADR-0010 (ACCEPTED), env `ADMIN_ORG_*` ditambahkan |
| 9 | Port dev frontend tidak pernah disebut di dokumen mana pun | RESOLVED | Konvensi 5173 + proxy Vite `/api` ditulis di `12-DEVELOPMENT-WORKFLOW.md` §7.1 |
| 5 | Requirement `FR-*`/`NFR-*` tidak pernah dipetakan ke kode | RESOLVED | Dibuat `TRACEABILITY.md` + wajib mencantumkan ID requirement di commit message |

---

## 3. Riset Best Practice untuk Sembilan Temuan yang Masih OPEN (2026-09-19, P-026)

Diminta oleh user: "jika masih ada pertanyaan terbuka, cari best practice yang ada dan jadikan pertimbangan
untuk menjawab". Riset di bawah dipakai untuk **menjawab**, bukan sekadar menyarankan: (a) dua butir yang
dapat diputuskan secara teknis (Q-017 butir 10 dan 11) dikerjakan di P-026, lalu (b) **sembilan temuan yang
tersisa diputuskan pada lanjutan sesi yang sama** — C-006/C-007/C-010/C-028 `FIXED`, dan
C-004/C-009/C-033/C-035 `APPROVED` lewat ADR-0019/0020/0021/0022, dengan C-015 tetap milik user karena ia
bukan pilihan teknis. Kolom "Rekomendasi" karena itu menjadi **dasar keputusan yang dapat diperiksa ulang**,
termasuk dua tempat di mana keputusan akhirnya **menyimpang** dari rekomendasi (C-009 dan C-028) —
selisihnya dicatat di akhir bagian ini. Sumbernya praktik industri & panduan keamanan yang dapat diperiksa ulang, bukan preferensi agen.

| Temuan | Pertanyaan | Best practice yang ditemukan | Rekomendasi |
|---|---|---|---|
| **C-004** | Dokumen: hapus atau arsip? | Sistem yang menyimpan jejak audit memakai **soft delete/arsip sebagai default**; penghapusan permanen disediakan hanya sebagai operasi eksplisit ber-izin (permintaan penghapusan data, mis. hak penghapusan UU 27/2022 PDP) dan ia sendiri harus tercatat. Versi immutable (FR-VER-03) bertabrakan langsung dengan kaskade hapus. | **Arsip sebagai default**, menyamakan dengan project (`POST /projects/:id/archive`): `DELETE /documents/:id` diganti arsip + kolom status/`archived_at`; penghapusan permanen (bila kelak dituntut hukum) dibuat endpoint terpisah khusus Administrator dan ber-audit. Butuh ADR karena mengubah kontrak §4 |
| **C-006** | "Reviewer" role kelima atau bukan? | Pisahkan **role sistem** (izin, organisasi) dari **peran fungsional proses** (siapa yang bertanggung jawab pada step berjalan). Peran fungsional diturunkan dari keadaan data, bukan disimpan sebagai role baru — pola yang sama dengan ADR-0012 (turunan, bukan nilai baru). | Perbaiki kata di `10-BRD.md`, `20-SRS.md`, dan `51-UX.md`: Reviewer = peran fungsional (user yang menjadi responsible step), **bukan** role sistem. Tidak ada role kelima di matriks |
| **C-007** | Hierarki role | Jangan memberi peringkat lintas cakupan: role sistem (organisasi) dan role project (keanggotaan) adalah dua ruang berbeda; membandingkannya menghasilkan aturan yang tidak dapat diuji. Peringkat hanya berguna bila dipakai keputusan izin. | Pisahkan dua hierarki di `44-SECURITY.md`/`20-SRS.md`, dan letakkan **Owner** di puncak hierarki **project** (ia tidak punya arti di tingkat sistem). Tanpa ADR karena hanya menyelaraskan dua dokumen |
| **C-009** | Auto-lock akun | OWASP WSTG menyarankan ambang **5–10** percobaan gagal; CIS: **≤10**; praktik yang disarankan adalah **lock sementara + backoff**, bukan lock permanen, supaya penyerang tidak dapat mengunci akun orang lain (denial of service). Counter harus disimpan di server, dan admin dapat membuka lebih awal. | Implementasikan janji yang sudah ada di `44-SECURITY.md` §2: ambang dari `system_settings` (`auth.max_login_attempts`, `auth.lockout_duration_minutes` — **kolomnya sudah ada**, jadi ini menutup janji, bukan menambah fitur baru): 10 kegagalan dalam 15 menit → terkunci 15 menit, otomatis terbuka, admin dapat membuka lebih awal. Butuh ADR + migrasi kolom (`locked_until` atau tabel `login_attempts` pada C-035) |
| **C-010** | Penugasan step ke user tertentu | Untuk MVP, otorisasi **berbasis role** menjangkau hampir seluruh kebutuhan alur persetujuan dan lebih mudah diuji; penugasan per user menambah kolom nullable yang mengubah arti `responsible_role` (dua sumber kebenaran) dan biasanya baru dibutuhkan saat pola pergantian tanggung jawab (delegasi/cuti) muncul. | Tandai `FR-WF-03` sebagai **role-based untuk MVP** di `20-SRS.md` dan `43-WORKFLOW.md` §5 tetap "Future"; `responsible_user_id` ditunda sampai ada kebutuhan delegasi yang nyata |
| **C-015** | Arah desain (`DESIGN.md`) | — (keputusan rasa/identitas, bukan teknis) | **Tetap milik Anda** (Q-002). Riset tidak dapat menggantikannya |
| **C-028** / **Q-012** | Retensi audit log | Tidak ada satu angka universal: auditor SOC 2 umumnya mengharapkan **≥12 bulan**; ISO 27001 (A.8.15) tidak menetapkan angka tetapi menuntut log dipelihara untuk investigasi — 12 bulan adalah praktik umum; audit/security log di industri lazim disimpan **1–7 tahun**; penyedia besar memberi pilihan 7 hari–7 tahun (Microsoft Purview). | Opsi **(b)**: retensi sebagai **operasi pemeliharaan terjadwal** lewat ADR, dengan **lantai 12 bulan**, angka dapat dikonfigurasi di `system_settings`, pemadaman memakai jalur `bwdcs.audit_maintenance` (jangan melepas trigger), dan penghapusan itu sendiri dicatat di log aplikasi. Append-only berarti "tidak dapat diubah", bukan "tidak dapat dipangkas kebijakan yang sah" |
| **C-033** / **Q-013** | Pencabutan seluruh sesi | Konsensus: access token pendek + rotasi refresh token; pencabutan segera seluruh sesi biasanya lewat **denylist `jti`** (TTL = sisa umur token) atau **penanda per user** (`token_version`/`tokens_invalid_before`) — yang terakhir lebih murah karena tidak menumbuhkan tabel dan langsung mencakup semua token yang sudah terbit. | Tambah **`users.tokens_invalid_before TIMESTAMPTZ`** (migrasi `010`); middleware menolak token dengan `iat < tokens_invalid_before` (satu baris, cache TTL yang sudah ada tetap dipakai). `token_revocations` tetap untuk jti tunggal. Ini menghidupkan `logout_all`, `change-password`, reset oleh admin, dan penonaktifan akun sekaligus — ketiga `reason` itu sudah ada di skema |
| **C-035** / **Q-014** | Audit login **gagal** | OWASP menempatkan percobaan autentikasi gagal sebagai **security event yang wajib dicatat** (tanpa kredensial). Namun `audit_logs` dimaknai sebagai "tindakan aktor yang terautentikasi" — membuat `actor_id` nullable melemahkan FK itu untuk seluruh tabel. Praktik umum: pisahkan **telemetry keamanan** (volume besar, retensi pendek) dari **audit kepatuhan** (volume kecil, retensi panjang). | Buat tabel **`login_attempts`** terpisah (`username_diupayakan`, `ip`, `user_agent`, `berhasil`, `correlation_id`, `created_at`, retensi mis. 90 hari) alih-alih membuat `actor_id` nullable. Tabel yang sama menjadi penghitung untuk auto-lock **C-009**, jadi satu keputusan menutup dua temuan |

**Hasil pemakaian riset ini (2026-09-19, lanjutan P-026).** Sembilan temuan di atas **dijawab memakai kolom Rekomendasi**, dengan dua koreksi yang diambil sadar saat memutuskan:

| Temuan | Keputusan akhir | Berbeda dari rekomendasi? |
|---|---|---|
| C-004 | ADR-0019: arsip default, izin arsip `document:update`, penghapusan permanen ditunda | Sesuai |
| C-006 / C-007 / C-010 | Penyelarasan dokumen (peran fungsional; dua hierarki terpisah; `responsible_user_id` ditunda) | Sesuai |
| C-009 | ADR-0022: ambang tetap dari `system_settings` yang **sudah ada** (default 5, bukan 10 baru), lock **sementara** | **Ya** — ambang diubah tidak naik ke 10, supaya FR-AUTH-06 yang sudah dikontrak tidak berubah demi selera |
| C-028 | ADR-0020: lantai 12 bulan sebagai **konstanta kebijakan**, **bukan** kunci `system_settings` | **Ya** — rekomendasi menyebut "dapat dikonfigurasi di `system_settings`"; itu akan menambah kunci yang tidak dibaca siapa pun, yaitu cacat yang sedang ditutup |
| C-033 | ADR-0021: `users.tokens_invalid_before` (opsi B), bukan `session_id` (opsi A) | **Ya** — Opsi A lebih tepat secara semantik tetapi menuntut tabel sesi; ia tetap mungkin dibangun di atas kolom ini, dan MVP tidak membutuhkannya |
| C-035 | ADR-0022: tabel `login_attempts`, `actor_id` tetap `NOT NULL` | Sesuai (opsi B) |
| C-015 | Tetap OPEN — milik user | Sesuai (riset memang tidak dapat menjawabnya) |

**Sumber utama** (diperiksa 2026-09-19): OWASP *Web Security Testing Guide* — Testing for Weak Lock Out Mechanism (ambang 5–10) dan OWASP *Authentication Cheat Sheet*; CIS Windows Benchmark (lockout threshold ≤10); SOC 2/ISO 27001 retention guidance (testrig/optro/konfirmity/stellans ringkasan praktik: ≥12 bulan, 1–7 tahun); Microsoft Purview audit log retention policies (opsi 7 hari–7 tahun); Stripe API (interval setengah terbuka `created[gte]`/`created[lt]`); Google AIP-160 (filtering); SuperTokens/OneUptime (pola pencabutan JWT: denylist `jti` vs penanda per user); UU 27/2022 (hak penghapusan data).

**Yang tidak dijawab riset:** urutan fase (Q-018) dan arah desain (C-015/Q-002) — keduanya keputusan pemilik produk, bukan pilihan teknis.
