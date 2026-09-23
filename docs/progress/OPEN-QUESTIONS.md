# OPEN-QUESTIONS — Pertanyaan & Keputusan Pending

**Protokol:** `docs/design/02-AGENT-PROGRESS-PROTOCOL.md`
**Aturan:** pertanyaan yang butuh keputusan user tidak boleh ditebak. Catat di sini, tandai sebagai `BLOCKING` atau `NON-BLOCKING`, dan tulis asumsi sementara di kolom "Asumsi sementara" agar pekerjaan lain tetap bisa jalan.

---

## 1. Pertanyaan Terbuka

### Q-001 — Mode penggunaan antislop → **RESOLVED: `during`** (2026-09-21, P-037)

- **Keputusan user:** **`during`** — aturan antislop diterapkan **sambil** membangun, bukan sebagai audit di akhir.
- **Konsekuensi:** filter antislop dipakai pada setiap sesi UI, dan Delivery Gate dijalankan **sebelum** serah terima, bukan sesudahnya. ADR-0006 dinaikkan dari `PROPOSED` menjadi **`ACCEPTED`** pada sesi yang sama.
- **Catatan jujur:** direktori `skills/antislop-*/SKILL.md` tetap belum ada (Q-003), jadi filter yang benar-benar dipakai sesi ini adalah `antislop.md` (core) + `DESIGN.md`. Itu dicatat di log prompt `P-037` §7.
- **Terkait:** ADR-0006, `01-AGENT-WORKFRAME.md` §3.1, task `T-007` (DONE).

### Q-002 — Sumber arah desain → **RESOLVED: jalur (b)/(2) — agen mengisi atas izin eksplisit user** (2026-09-21, P-037)

- **Keputusan user:** user **menyuruh agen menyusun arah desain** ("isi `DESIGN.md`, tentukan dengan rekomendasi Anda berdasarkan best practice"). Itu **jalur 2** ADR-0007, bukan (a) yang tadinya direkomendasikan dan bukan pula (c).
- **Konsekuensi:** `DESIGN.md` berstatus `TERISI` dengan peringatan yang menyertai statusnya: butir **struktural** (dials, aturan aksesibilitas, pemisahan warna status, aturan mono) adalah keputusan yang dipertanggungjawabkan; butir **rasa** (kepribadian, referensi, penamaan tema) adalah **draf agen** yang boleh diganti user tanpa ADR baru. Status "draft without direction" (R-37) gugur, dan dial sementara 1/1/1 tidak dipakai lagi. ADR-0007 `ACCEPTED`.
- **Yang membuat keputusan ini tidak bisa diklaim sepihak:** palet dan skala dikunci di `frontend/src/styles/tokens.css` dan **dihitung ulang** test kontras (31 test WCAG 2.2 AA); mengubahnya tanpa mengubah test akan gagal.
- **Terkait:** ADR-0006/0007/0024, `DESIGN.md`, `11-DESIGN-DIRECTION.md` (kini `SUPERSEDED`), task `T-006` (DONE), temuan C-015 + C-060.

### Q-003 — Skill antislop belum tersedia → **RESOLVED: skill terpasang, dipin ke tag rilis** (2026-09-22, P-042)

- **Temuan (semula):** `AGENTS.md` menunjuk `skills/antislop-ui/SKILL.md`, `skills/antislop-copywriting/SKILL.md`, `skills/antislop-human/SKILL.md`, `skills/antislop-layoutmobile/SKILL.md`, `skills/antislop-code/SKILL.md`, tetapi direktori `skills/` tidak ada di repo — entry file menyatakan sesuatu yang tidak benar (temuan **C-064**).
- **Aturan yang berlaku:** agen tidak boleh mengunduh berkas skill (`skills/<nama>/SKILL.md`) dari jaringan, dan tidak boleh mengarang isinya.
- **Keputusan user (2026-09-22):** user menyebutkan sumbernya (`https://github.com/miqdadbadjuber/anti-slop`), meminta skill-nya diterapkan dengan benar, dan **memberi izin eksplisit** kepada agen untuk mengunduhnya langsung dari repo itu. Agen memasang **kelima** skill + core + `contrast-check.py` + salinan `LICENSE`, semuanya byte-identik dari **tag rilis `v3.2.12`** (bukan `main`), dengan `sha256` dicatat di `skills/README.md` §1.
- **Konsekuensi yang mengikat:** izin itu **satu kali, untuk sesi P-042** — bukan aturan tetap. Sesi berikutnya **tidak boleh** mengunduh ulang atas inisiatif sendiri, termasuk "sekadar menyegarkan"; pembaruan mengikuti `skills/README.md` §2 atas permintaan user. Filter UI kini tersedia dalam bentuk lengkap (core + lima skill), dan `bash scripts/check-antislop-refs.sh` memeriksa rujukan serta keaslian berkasnya.
- **Sekaligus ditemukan dan ditutup:** **C-065** — `antislop.md` di root ternyata varian lama yang berbeda dari salinan Delivery Gate di dokumen desain, dan daftar aturannya sudah disalin ke `01-AGENT-WORKFRAME.md` §3.2. Salinan itu kini dihapus (dokumen hanya menunjuk), dan salinan core di `skills/antislop/SKILL.md` diperiksa identik dengan core di root.
- **Terkait:** ADR-0025, ADR-0006 (butir 2 konsekuensinya diamandemen), `skills/README.md`, `AGENTS.md` (blok penunjuk), temuan C-064/C-065, task `T-054`.

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

**Keputusan: opsi (B), bukan opsi (A) yang dulu lebih disarankan.** Satu kolom di `users`, satu perbandingan di middleware (`iat < tokens_invalid_before` → `401 TOKEN_REVOKED`), tanpa tabel sesi yang tumbuh. Opsi (A) (`session_id` + daftar sesi) **tidak** dibatalkan sebagai kemungkinan kelak — ia dapat dibangun di atas kolom ini tanpa membatalkannya — tetapi MVP tidak membutuhkan *daftar* sesi, hanya kemampuan mematikannya. Keterbatasan yang diterima dengan sadar: `logout_all` juga mematikan sesi di perangkat lain, dan `change-password` menyelesaikannya dengan **menerbitkan token baru** untuk request yang sedang berjalan (bukan dengan `keepJTI`, yang tidak dinyatakan oleh penanda per user). Sumber pertimbangan: §3 di bawah (denylist `jti` vs penanda per user). Task: **`T-040`** (menyertai `T-034`) — **DONE pada P-030**: `logout_all` mencabut seluruh sesi lewat `users.tokens_invalid_before` dan `501 NOT_IMPLEMENTED` tidak ada lagi.

Ditemukan saat mengerjakan `T-005` (P-021, temuan **C-033**). ADR-0009 butir 3 dan `42-API.md` §2 menjanjikan lebih dari yang dapat dilakukan skema: tabel `token_revocations` hanya memuat `jti` yang **sudah** dicabut, dan sistem tidak menyimpan daftar sesi/token aktif. Tiga janji yang bergantung padanya: `logout_all: true`, "cabut seluruh token lain" pada `POST /auth/change-password` (FR-AUTH-09), dan hal yang sama pada reset password Administrator (FR-AUTH-08).

- **Opsi A — `session_id` di dalam token dan di `token_revocations`** (rekomendasi): satu login = satu sesi yang dicabut utuh, `logout_all` mencabut semua sesi user, dan `change-password` mencabut semua sesi kecuali yang sedang dipakai (`keepJTI` menjadi `keepSession`). Menuntut ADR baru + migrasi kolom.
- **Opsi B — kolom `users.tokens_valid_after`**: token dengan `iat` lebih lama ditolak. Paling murah (satu kolom, satu perbandingan), tetapi `logout_all` **juga** mematikan sesi di perangkat lain yang belum melakukan apa-apa, dan "kecuali sesi yang sedang dipakai" pada `change-password` tidak dapat dinyatakan tanpa mekanisme tambahan.
- **Opsi C — turunkan janji**: logout mencabut token yang dipakai saja; `logout_all` dan pencabutan sesi lain dibuang dari MVP, `42-API.md` §2 dan ADR-0009 disesuaikan. Tidak ada perubahan skema.

- **Status per P-030:** `logout_all: true` → `200` — seluruh sesi user dicabut lewat `users.tokens_invalid_before` (`42-API.md` §2/§12; `501 NOT_IMPLEMENTED` dihapus). `change-password` dan `refresh` **belum** didaftarkan sebagai route (`T-034`), tetapi mekanisme pencabutan sesinya sudah jalan.
- **Status per P-034:** `POST /auth/change-password` **sudah berjalan** — `users.tokens_invalid_before` disetel, lalu token pengganti diterbitkan sesudah commit; token lama dan token perangkat lain `401 TOKEN_REVOKED`, sesi user lain tidak tersentuh (`70-TESTING.md` §3.12c). Yang **belum** hanya `POST /auth/refresh`, dan itu bukan lagi soal mekanisme pencabutan melainkan bentuk tokennya: **Q-020** (task `T-045`).
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

> **Catatan P-047 (tempat penyaring `Milik saya`):** sub-item sidebar yang dahulu menjadi jalan masuk
> `?view=mine` dipindahkan ke **baris tab di halaman Documents** (`51-UX.md` §2.1). Deep link-nya tetap
> berjalan seperti sebelumnya, dan karena tabnya kini terlihat di halaman, alasan butir (8) tetap terbaca
> oleh siapa pun — bukan hanya oleh orang yang menghafal URL-nya. Status pertanyaan ini **tidak berubah**.

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

- **(a) ✓ DIPILIH agen pada P-026, lalu DIKOREKSI USER pada 2026-09-19 (P-028)** — dua parameter RFC 3339. Semula diimplementasikan sebagai interval **setengah terbuka** `[due_from, due_to)`: batas bawah inklusif, batas atas eksklusif, dengan alasan rentang bersebelahan tidak tumpang tindih dan konvensi API publik besar (Stripe `created[gte]` + `created[lt]`). **Semantik yang berlaku sekarang: keduanya inklusif** `[due_from, due_to]`, karena user meminta batas inklusif dan pilihan inklusif-inklusif memang dinyatakan reversibel pada butir Status di bawah. Akibatnya: task yang `due_date`-nya tepat sama dengan salah satu batas ikut terpilih, `due_to == due_from` sah (satu instan), hanya rentang terbalik yang ditolak `422`, dan rentang bersebelahan dapat tumpang tindih — klien yang ingin tidak tumpang tindih harus mengirim batas atas satu satuan sebelum batas bawah berikutnya. Perubahan ini tidak menyentuh skema; kontraknya di `42-API.md` §6, testnya `TestTaskListDueRangeFilterIsInclusiveBothEnds`, bukti server nyata di `prompts/P-028-*.md` §6.
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

- **Status:** butir (1)-(9) berlaku seperti yang diimplementasikan pada P-025. Butir (11) **diputuskan dan
  dikerjakan pada P-026** memakai rekomendasi (a) dan **masih berlaku apa adanya** (diverifikasi ulang
  pada P-028: `422` menamai `document_id`/`title`/`due_date`/`body` dan `owner_id` pada `POST /projects`).
  Butir (10) diputuskan P-026 dengan opsi (a) **setengah terbuka**, lalu **dikoreksi user pada 2026-09-19
  (P-028)** menjadi **kedua batas inklusif** — itulah yang berlaku sekarang. Keduanya tetap reversibel
  tanpa menyentuh skema (dan tanpa migrasi): `YYYY-MM-DD` maupun pengembalian ke setengah terbuka masih
  mungkin, begitu pula pendekatan (b) untuk atribusi error.
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
- **Hasil P-027:** **Comment sudah dikerjakan** (`T-042`) dan Phase 3 tertutup, persis seperti rekomendasi.
  Pertanyaan yang tersisa hanya urutan **sesudahnya**: Workflow (Phase 2) atau modul admin/notification.
- **Terkait:** `80-ROADMAP.md` §1/§3, `TASKS.md` backlog fase 2/3, temuan C-047.

### Q-019 — Apakah balasan komentar ber-thread (`Reply`) perlu dihidupkan? (NON-BLOCKING)

- **Konteks:** `50-FSD.md` §7 mencantumkan field "Reply (optional, threaded)", sedangkan tabel `comments`
  (`41-DATABASE.md` §2.5, migrasi `007`) tidak punya kolom induk (`parent_id`/`reply_to_id`). Tabel itu
  **sudah terpasang** pada database yang berjalan, jadi menambah kolom berarti migrasi baru; dan karena
  ini mengubah bentuk data, `CONTINUE.md`/`AGENTS.md` mewajibkan **ADR** untuk itu.
  Ditemukan saat menulis test modul komentar (temuan **C-050**).
- **Pertanyaan:** apakah threading masuk MVP, ditunda, atau dihapus dari FSD?
- **Rekomendasi: TUNDA.** Alasannya: (1) tidak ada `FR-CMT-*` yang menuntutnya — `FR-CMT-03` hanya
  meminta "timeline"; (2) threading bukan sekadar satu kolom: bentuk respons harus diputuskan
  (datar ber-`parent_id` vs bersarang rekursif), urutan tampilan perlu aturan (kronologis vs per cabang),
  paginasi menjadi lebih rumit (satu halaman bisa memuat separuh cabang), dan notifikasi
  `COMMENT_MENTION`/balasan menuntut modul Notification yang belum dibangun; (3) menetapkan bentuk API
  sebelum ada pemakai nyata berisiko membekukan pilihan yang salah. Alternatif "harga tetap" yang bisa
  dipilih nanti: kolom `parent_id UUID NULL REFERENCES comments(id)` + tampilan datar dengan penanda
  "membalas…" — migrasi kecil, tetapi tetap butuh ADR karena mengubah skema.
- **Asumsi sementara:** balasan ditulis sebagai komentar biasa pada entitas yang sama dan ditampilkan
  dalam satu timeline datar; `50-FSD.md` §7 sudah menyatakannya sebagai **belum didukung** sehingga tidak
  ada janji dokumen yang menggantung. Perilaku itu dikunci test
  (`TestCommentThreadingIsNotSupported`, yang gagal bila kelak kolom induk muncul).
- **Terkait:** `50-FSD.md` §7/§8.1, `41-DATABASE.md` §2.5, `42-API.md` §7, temuan C-050, `70-TESTING.md` §3.13.

### Q-021 — Refresh token di klien: `sessionStorage` atau cookie `HttpOnly`? (NON-BLOCKING)

- **Konteks:** `42-API.md` §2/ADR-0023 mengembalikan refresh token **di body respons**, dan backend tidak memasang cookie. `frontend/src/services/session.ts` karena itu menyimpan refresh token di `sessionStorage` (`localStorage` ditolak supaya sesi berakhir saat tab ditutup), sedangkan **access token hanya di memori** agar tidak dapat dibaca dari penyimpanan. Praktik yang dituju (OWASP) adalah refresh token di cookie `HttpOnly` + `Secure` + `SameSite`, dengan access token tetap di memori.
- **Kenapa belum dikerjakan:** memindahkannya **mengubah kontrak backend** (`Set-Cookie` di `login`/`refresh`, `credentials: true`, aturan CORS dengan origin eksplisit, dan `/auth/refresh` menerima token dari cookie alih-alih body). Itu keputusan yang menyentuh ADR-0023 dan `42-API.md`, jadi tidak dikerjakan sendiri oleh agen pada sesi scaffold.
- **Risiko bila dibiarkan:** XSS yang berhasil berjalan di halaman dapat membaca refresh token dari `sessionStorage` dan mempertahankan sesi lebih lama. Dampaknya dibatasi klaim `typ` (refresh token **tidak** dapat dipakai sebagai bearer — sudah dibuktikan test) dan umur 7 hari, tetapi tetap lebih lemah daripada cookie `HttpOnly`.
- **Asumsi sementara:** `sessionStorage` sebagai jembatan, terisolasi di **satu berkas** (`services/session.ts`) supaya perpindahan ke cookie hanya menyentuh berkas itu plus `http.ts` (`withCredentials: false`).
- **Terkait:** ADR-0009/0021/0023, `42-API.md` §2, `44-SECURITY.md` §2.2, `frontend/src/services/session.ts`.

### Q-022 — Konvensi frontend yang diputuskan agen saat scaffold (NON-BLOCKING)

- **Konteks:** membangun kerangka frontend (`T-048`, P-037) menuntut keputusan yang belum ada di dokumen desain. Semuanya diputuskan agen dan dicatat di sini supaya dapat dikoreksi user, bukan ditemukan belakangan sebagai penyimpangan.
- **Keputusan yang diambil (beserta alasannya singkat):**
  1. **Tailwind v4 CSS-first, tanpa `tailwind.config.js`** — token hidup di `src/styles/tokens.css` (`@theme` + lapisan semantik). Satu sumber token untuk utility dan CSS biasa. Dikunci **ADR-0024**.
  2. **React Router 7 library mode** (bukan framework mode) — SPA tanpa SSR, sesuai ADR-0002.
  3. **`VITE_API_BASE_URL` default relatif `/api/v1`** — memakai proxy dev server Vite, jadi tidak perlu CORS dibuka lebar; produksi juga relatif karena web server yang mem-proxy.
  4. **`withCredentials: false`** sampai Q-021 selesai.
  5. **Penukaran refresh di klien bersifat single-flight** — banyak permintaan yang gagal bersama hanya memicu **satu** penukaran, karena ADR-0023 tidak mencabut refresh token lama saat rotasi sehingga penukaran berulang tidak menambah keamanan.
  6. **Zustand hanya untuk keadaan klien** (sesi, tema). Hasil API **tidak** disimpan di sana; pustaka server-state (**TanStack Query**) akan ditambahkan bersama halaman data pertama, dan itu disebut di `30-ARCHITECTURE.md` §2.1.
  7. **Halaman menu yang belum dibangun tampil sebagai `ModulePending`** yang menyebut apa yang belum ada — bukan tabel kosong atau angka contoh. Aturan yang sama untuk Dashboard (R-18/R-36).
  8. **Motif "punggung rekam" (`DESIGN.md` §6) dijalankan lewat token, bukan CSS ad-hoc** — garis status pada baris tabel memakai pasangan `--color-status-*-ink`.
- **Risiko bila salah:** semuanya konvensi internal yang murah dibalik; yang paling mahal adalah (1) dan (6) karena menyentuh banyak berkas.
- **Terkait:** `30-ARCHITECTURE.md` §2.1/§2.2, ADR-0024, `60-DEPLOYMENT.md` §3.2, `frontend/src/services/*`.

### Q-023 — Lisensi proyek dan kebijakan kontribusi pihak ketiga (BLOCKING untuk distribusi, NON-BLOCKING untuk pengembangan)

- **Konteks:** `README.md` §13 menyatakan status lisensi **belum ditetapkan**, dan repositori ini memang
tidak punya berkas `LICENSE`, tidak punya field `license` di `frontend/package.json`, dan tidak punya
`CONTRIBUTING.md`. Sesi P-038 meminta pemilik proyek menetapkan lisensinya sebelum berkas `LICENSE`
dibuat — dan pemilik memilih **menunda keputusan itu** sambil tetap menjawab siapa **pemegang hak cipta**-nya.
- **Yang sudah dijawab pemilik (2026-09-21, P-038):**
  - **Butir (b) — pemegang hak cipta: `BSA`** (pribadi pemilik proyek). Ini yang akan ditulis pada baris
    hak cipta berkas `LICENSE` begitu jenis lisensinya ditetapkan; tercatat juga di `README.md` §13.
- **Yang masih menunggu keputusan pemilik:**
  - **Butir (a) — jenis lisensi.** Belum diputuskan. Pilihan yang sudah disiapkan agen: proprietary
    *all rights reserved* (paling konsisten dengan alat internal perusahaan, dan menegaskan keadaan
    bawaan yang sudah berlaku), **MIT** (paling ringkas dan umum untuk proyek publik), atau **Apache-2.0**
    (permisif plus pemberian lisensi paten eksplisit). Sampai diputuskan, **tidak ada** yang boleh
    menganggap proyek ini open source, dan agen tidak boleh membuat berkas `LICENSE` sendiri.
  - **Butir (c) — apakah kontribusi pihak ketiga diterima**, dan lewat kanal apa (pelacak isu, surel,
    atau pull request). Saat ini jalur satu-satunya adalah ledger: entri di `TASKS.md` untuk usulan kerja
    dan `OPEN-QUESTIONS.md` untuk hal yang butuh keputusan pemilik.
- **Konsekuensi bila ditunda terus:** tidak ada perubahan pada pengembangan (build, test, dan migrasi
  tidak bergantung pada lisensi). Yang tertahan hanya **distribusi dan penerimaan kontribusi luar**:
  tanpa lisensi, pihak ketiga tidak punya hak apa pun atas kode ini — termasuk hak untuk ikut
  menyumbang dengan aman secara hukum. Karena itu butir (a) dan (c) sebaiknya dijawab bersamaan.
- **Asumsi sementara:** pengembangan berjalan seolah proyek ini privat (module Go-nya pun sudah privat:
  `bwdcs/backend`, tanpa domain), `frontend/package.json` tetap `"private": true` tanpa field `license`,
  dan tidak ada berkas `LICENSE` yang dibuat. Tidak ada kode yang bergantung pada asumsi ini.
- **Terkait:** `README.md` §12.6/§13, ADR-0004 (mekanisme deployment fleksibel), `docs/progress/TASKS.md` **T-050**.

### Q-025 — Apakah prosa dokumen dan komentar kode lama juga disapu em dash? (NON-BLOCKING)

- **Konteks:** aturan **R-02** melarang em dash pada teks yang ditulis agen, dan keputusan proyeknya di
  `01-AGENT-WORKFRAME.md` §3.2 semula berbunyi "semua teks UI **dan dokumentasi baru** bebas em dash" —
  tetapi tidak ada pemeriksa yang membaca keputusan itu, sehingga tiga teks di layar memang masih
  memuatnya (temuan **C-069**, ditutup P-043). Sejak P-043 yang diperiksa mesin adalah **teks yang dibaca
  pengguna** (`frontend/src`, tanpa komentar), dan baris §3.2 sudah dipersempit agar tidak menjanjikan
  lebih dari yang ditegakkan.
- **Yang belum diputuskan:** prosa di `docs/**` dan komentar kode **lama** masih memakai em dash sebagai
  tanda pisah dalam kalimat Indonesia (ratusan kemunculan, tersebar di hampir seluruh berkas). Menyapunya
  berarti menyunting prosa teknis dalam jumlah besar: risikonya bukan pada alat (penggantian mekanis
  mudah) melainkan pada **makna** — banyak kalimat memakai tanda itu untuk sisipan yang tidak setara
  dengan koma, sehingga penggantian buta menghasilkan kalimat yang salah baca.
- **Pilihan:**
  - **(a) Biarkan pada cakupan sekarang** (hanya teks yang dibaca pengguna yang diperiksa). Paling murah,
    dan yang benar-benar dibaca pengguna akhir sudah bersih. Risikonya: aturan upstream lebih luas
    daripada yang ditegakkan, sehingga selisihnya harus tetap dinyatakan di §3.2 — seperti sekarang.
  - **(b) Sapu juga dokumentasi baru ke depan** (berlaku untuk berkas yang dibuat sesudah aturan ini),
    tanpa menyentuh prosa lama. Menambah disiplin tanpa gelombang suntingan besar; menuntut pemeriksa
    yang tahu "berkas baru", dan itu tidak dapat diperiksa mesin tanpa daftar tambahan.
  - **(c) Sapu seluruh repo sekarang**, termasuk prosa dan komentar lama, dengan pemeriksa yang diperluas
    ke `docs/**`. Konsisten dengan aturan upstream, tetapi diff-nya besar (menyentuh hampir setiap
    dokumen) dan setiap kalimat perlu dibaca ulang untuk menjaga maknanya.
- **Rekomendasi agen:** **(a) sekarang, (b) saat berkas baru ditulis** — karena teks yang benar-benar
  sampai ke pengguna sudah dijaga mesin, sedangkan (c) paling baik dikerjakan bersamaan dengan sesi yang
  memang menyentuh dokumen itu, bukan sebagai gelombang tersendiri yang menenggelamkan perubahan lain.
- **Keputusan Anda:** _(belum dijawab)_

### Q-026 — Target sentuh di desktop: pertahankan 36px (padat) atau naikkan ke 44px? (NON-BLOCKING)

- **Konteks:** `DESIGN.md` §4 menetapkan kepadatan alat kerja: baris tabel dan kontrol 36px di desktop,
  44px di layar sentuh (`tap-target` di `tokens.css` membalik nilainya pada `min-width: 64rem`). Keputusan
  itu beralasan (alat kerja yang padat menampilkan lebih banyak baris dalam satu layar), tetapi aturan
  kerajinan layar sentuh menulis ambang **44px tanpa kualifikasi**, dan Delivery Gate menanyakannya
  sebagai satu butir yang harus dijawab "ya". Sejak P-043 ambangnya **diukur mesin** per lebar
  (`scripts/responsive-evidence.mjs`: 44px < 1024px, 36px ≥ 1024px) dan keputusannya dinyatakan di
  `51-UX.md` §9, jadi tidak ada lagi klaim yang tidak diperiksa — yang tersisa hanya ketegangan antara
  dua aturan itu sendiri.
- **Pilihan:**
  - **(a) Pertahankan dua register** (44px sentuh, 36px desktop) dengan alasan tertulis seperti sekarang.
    Rekomendasi agen: halaman ini alat kerja internal, sasarannya kursor di desktop, dan mode sentuhnya
    sudah memenuhi syarat.
  - **(b) Naikkan semuanya ke 44px**, termasuk desktop. Paling aman terhadap aturan, dengan biaya:
    baris tabel dan kontrol menjadi lebih tinggi, informasi per layar berkurang sekitar seperlima.
  - **(c) Naikkan hanya kontrol yang sering dipakai** (tombol aksi, item navigasi) sementara baris tabel
    tetap 36px. Menuntut daftar kontrol yang "sering dipakai" — dan daftar seperti itu tidak dapat
    diperiksa mesin, jadi ia menambah aturan yang harus diingat manusia.
- **Rekomendasi agen:** **(a)**, karena alasan kepadatannya memang tertulis dan kini diukur, dan karena
  (b) mengubah tata letak demi memenuhi ambang yang jelas-jelas dimaksudkan untuk layar sentuh.
- **Keputusan Anda:** _(belum dijawab)_

### Q-024 — Bagaimana klien memilih **pengguna** (field `Owner` project) tanpa endpoint daftar pengguna? (NON-BLOCKING)

- **Konteks:** `50-FSD.md` §3.2 mencantumkan field `Owner` sebagai **wajib** dan bertipe "dropdown / User
  select", plus aturan server "`Owner` yang dipilih langsung menjadi anggota project dengan role Owner".
  Halaman **Projects** dibangun pada P-041 dan tidak dapat memenuhinya: **tidak ada satu pun endpoint**
  yang dapat menyebutkan daftar pengguna — `42-API.md` §3 tidak memuatnya, `GET /admin/users` (§11)
  belum diimplementasikan, dan izin `user:read` menurut matriks `44-SECURITY.md` §3.1.2 hanya dimiliki
  **Administrator**, sementara halaman Administrasi juga belum ada. Ditemukan saat **membangun**
  halamannya, bukan dari membaca kontrak — dicatat sebagai temuan **C-063**.
- **Yang dilakukan sementara (tidak menunggu keputusan):** dialog membuat project menampilkan pemilik
  sebagai teks tetap **(Anda)**, halaman detail menyebut batas yang sama, dan test mengunci perilaku itu
  supaya tidak berubah menjadi dropdown kosong atau daftar pengguna karangan. Artinya **owner selalu
  pembuat project**; perilaku itu sah menurut `POST /projects` (server menambahkan pemilik sebagai
  anggota `Owner` dalam transaksi yang sama), hanya tidak memenuhi "User select" di FSD.
- **Pilihan yang disiapkan agen:**
  - **(a) Tambah satu endpoint baca daftar/pencarian pengguna** (`GET /users?q=&limit=`, terpaginasi,
    tanpa detail sensitif) dan tetapkan izinnya lewat ADR — memakai ulang `user:read` (Administrator)
    berarti pemilih pengguna hanya jalan bagi Administrator, atau menambah pasangan baru
    (`user:list`, semua role internal) berarti matriks `44-SECURITY.md` §3.1.2 berubah dan itu selalu
    butuh ADR (ADR-0014). **Rekomendasi agen**, karena `Owner` di FSD bukan hiasan: tanpanya project
    hanya dapat dibuat oleh dan untuk pembuatnya, Administrator tidak dapat menyerahkan kepemilikan,
    dan halaman **Members** (`42-API.md` §3) tidak dapat dibangun sama sekali.
  - **(b) Turunkan FSD §3.2 untuk MVP** menjadi "pemilik selalu pembuat project", lalu hapus janji
    "User select" beserta kalimat aturan servernya — murah, tetapi menghapus kemampuan menyerahkan
    kepemilikan dari dokumen, dan itu keputusan produk.
  - **(c) Gabungkan dengan modul Administration** (`42-API.md` §11): kerjakan `GET /admin/users` beserta
    halaman Administrasi lebih dulu, lalu pemilih pengguna di halaman Projects memakai endpoint itu
    (hanya untuk Administrator). Paling lengkap, paling lama, dan tetap menyisakan masalah (b).
- **Dampak bila ditunda:** **tidak menghalangi** halaman Projects, Documents, Tasks, Approvals, maupun
  Reports — hanya form anggota project dan pemilihan owner di halaman Projects yang tertahan, dan
  keduanya sudah dinyatakan terbuka di layar (C-063), bukan disenyapkan.
- **Terkait:** temuan **C-063**, `50-FSD.md` §3.2, `42-API.md` §3 dan §11, `44-SECURITY.md` §3.1.2 (ADR-0014),
  `frontend/src/pages/Projects/CreateProjectDialog.tsx` + `ProjectDetail.tsx`.

### Q-020 — Bentuk token `POST /auth/refresh` → **RESOLVED: JWT bertanda `typ` tanpa penyimpanan di server, lewat ADR-0023** (2026-09-20, P-034)

**Keputusan: opsi (A).** Refresh token adalah JWT kedua dari penerbit yang sama, bertanda klaim
`typ: refresh` (`access` untuk token akses), berumur **7 hari** sebagai konstanta kode `jwt.RefreshExpiry`,
**tidak disimpan di server**, dan tanpa migrasi apa pun. Klaim `typ` **wajib**: token tanpa `typ` ditolak,
refresh token tidak pernah diterima middleware, dan access token tidak pernah diterima endpoint refresh.
Login menerbitkan keduanya; setiap penukaran mengembalikan sepasang token baru sehingga jendela 7 hari
bergulir. Pencabutannya memakai jalur yang **sama** (ADR-0021 butir 6), sehingga `logout_all` dan
`change-password` otomatis mematikan refresh token lama. **Batas yang diterima sadar:** token lama tidak
dicabut saat rotasi karena bentuknya stateless, jadi pemakaian ulang tidak dapat dideteksi; opsi (B)
yang lebih kuat (token buram ber-rotasi di tabel sendiri, pola OWASP) **tidak** dibatalkan sebagai
kemungkinan kelak, dan `TestRefreshKeepsPreviousRefreshTokenValid` sengaja ditulis untuk gagal lebih dulu
bila mekanismenya kelak diganti. Opsi (C) (menghapus endpoint dari kontrak) ditolak karena user memilih
agar kontrak di `42-API.md` §2 benar-benar berjalan. **Hasil:** route hidup, ADR-0023 `ACCEPTED`,
task **`T-045`** `DONE`, dan `POST /auth/login` kini juga mengembalikan `refresh_token`.

- **Konteks:** `42-API.md` §2 sudah memuat `POST /auth/refresh` sejak lama, dan `40-TSD.md` §2.4
  mencantumkan `Refresh(refreshToken string) (*AuthToken, error)` pada sketsa `AuthService`. Yang
  **belum** ada adalah bentuk tokennya: tidak ada tabel maupun kolom untuk menyimpannya, tidak ada
  requirement `FR-AUTH-*` yang menuntutnya, dan `44-SECURITY.md` §2.2 hanya menyebut "Refresh token:
  optional, 7 hari expiration" tanpa mekanisme. Karena itu `42-API.md` §2 memuat larangan eksplisit:
  **jangan mengarang bentuk refresh token di kode sebelum ada keputusan**. Ditemukan saat mengerjakan
  `T-034` (P-034), yang akhirnya hanya dapat menyelesaikan `change-password`.
- **Yang sudah diputuskan, dan mengikat apa pun jawabannya nanti:** pemeriksaan pencabutan refresh
  memakai jalur yang **sama** dengan endpoint terproteksi lain — token ditolak bila `jti`-nya ada di
  `token_revocations` **atau** `iat`-nya lebih tua daripada `users.tokens_invalid_before` (ADR-0009
  butir 7 + ADR-0021 butir 6). Refresh tidak boleh punya jalur pemeriksaan sendiri. Konsekuensinya:
  `logout_all` dan `change-password` otomatis membuat refresh token lama tidak berguna.
- **Pertanyaan:** token refresh disimpan bagaimana, dan berapa masa berlakunya?
- **Opsi A — JWT bertanda `typ` (tanpa migrasi), *rekomendasi*.** Refresh token adalah JWT kedua dengan
  klaim pembeda (`typ: refresh`), masa berlaku 7 hari (sesuai `44-SECURITY.md` §2.2), dan divalidasi
  lewat pemeriksaan pencabutan yang sudah ada. Kelebihannya: nol perubahan skema, nol tabel baru yang
  harus dibersihkan, dan sejalan dengan ADR-0021 butir 6 yang memang menyebut refresh "memakai kolom
  yang sama sebagai pemeriksaan tunggal". Batasannya jujur: rotasi tidak dapat **dideteksi** (tidak ada
  penyimpanan), jadi refresh token yang dicuri tetap sah sampai `exp` kecuali `jti`-nya dicabut
  eksplisit atau sesinya dimatikan lewat `tokens_invalid_before`.
- **Opsi B — token buram (opaque) tersimpan ter-hash, dengan rotasi + deteksi pemakaian ulang.** Tabel
  baru `refresh_tokens` (`user_id`, `token_hash`, `expires_at`, `revoked_at`, `replaced_by`), token
  dikirim sekali dan diganti tiap kali dipakai; pemakaian token yang sudah diganti dianggap pencurian dan
  mematikan seluruh keluarga sesi. Ini pola yang direkomendasikan OWASP untuk refresh token. Harganya:
  migrasi `011`, ADR baru, dan satu tabel yang harus dirawat (pembersihan berkala seperti
  `token_revocations`).
- **Opsi C — turunkan janji: `POST /auth/refresh` dihapus dari kontrak.** Argumentasinya kuat dan
  sederhana: token akses berlaku 24 jam (`JWT_EXPIRY`), tidak ada `FR-AUTH-*` yang menuntut refresh, dan
  menambah kredensial berumur panjang justru memperbesar permukaan serangan untuk sistem internal
  berjumlah pengguna kecil. Kalau dipilih, `42-API.md` §2 dan sketsa `40-TSD.md` §2.4 dibersihkan, dan
  `T-045` ditutup sebagai `CANCELLED`.
- **Asumsi sementara:** route `POST /auth/refresh` **tidak didaftarkan** dan kodenya tidak ditulis;
  `42-API.md` §2 menyatakan itu apa adanya beserta alasannya, `40-TSD.md` §2.4 menandai `Refresh` belum
  ada, dan `T-045` berada di papan `BLOCKED`. Tidak ada perilaku yang menggantung: token akses 24 jam
  tetap berlaku, dan pencabutannya sudah jalan penuh.
- **Dampak bila tidak dijawab:** satu-satunya efeknya `T-045` tetap terbuka. Tidak menghalangi modul
  lain, tidak menghalangi Workflow, dan tidak membuat `change-password` setengah jalan. **Answer ini sudah
  diberikan pada P-034**, jadi tidak ada lagi yang menggantung di sini.
- **Terkait:** `42-API.md` §2, `44-SECURITY.md` §2.2, `40-TSD.md` §2.4/§6, ADR-0009 butir 7,
  ADR-0021 butir 6, `70-TESTING.md` §3.12c, `TASKS.md` `T-034`/`T-045`.

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


### Q-DASH-01 — Department / organisasi unit untuk filter & chart (NON-BLOCKING, Phase 5)

- **Konteks:** `Dashboard.md` §1.7/§2.7/§6 meminta `Documents by Department`, `Workflow Volume by Department`, filter `Department`. Skema `41-DATABASE.md` hanya punya `organizations` (tenant) dan `projects` — tidak ada hierarki department. MVP dashboard (`52-DASHBOARD-ANALYTICS.md` §4.1) menahan chart itu sampai struktur didefinisikan.
- **Pertanyaan:** apakah department = subset `projects` (mis. `projects.department` ENUM), tabel baru `departments` + `project_department_id`, atau `users.department` + `documents.owner.department`? Masing-masing menambah DDL berbeda.
- **Rekomendasi agen:** tunda sampai kebutuhan organisasi nyata ada; untuk MVP gunakan `project` sebagai proxy department, jangan menambah kolom nullable yang tidak jelas dipakai.
- **Dampak bila ditunda:** chart by Department di Dashboard.md akan kosong di MVP, tetapi 8 chart MVP lain tetap hidup.
- **Terkait:** `Dashboard.md` §1.7, `52-DASHBOARD-ANALYTICS.md` §4.1, task `T-074`.

### Q-DASH-02 — Definisi SLA Compliance (On Time / Late / Overdue) (NON-BLOCKING)

- **Konteks:** KPI `SLA Compliance Rate` dan chart `SLA Compliance`/`SLA Trend` (`Dashboard.md` §1.6, §2.4) menulis tiga bucket tanpa rumus. `workflow_steps.deadline_days` dan `workflow_instances.current_step_deadline` sudah ada (ADR-0015), tetapi kapan sebuah workflow dianggap Late? `completed_at > deadline` pertama? `current_step_deadline` lewat saat masih `running`? Butuh ADR.
- **Opsi:** (a) `On Time = completed_at <= max(step deadines) OR still running && deadline not passed`, (b) per-step Late, (c) definisi tenant-specific via `system_settings`.
- **Rekomendasi:** (a) sederhana + dapat dihitung dari kolom yang ada — `completed_at` vs `current_step_deadline` + `workflow_actions` — tanpa `stage_history` baru.
- **Terkait:** `52-DASHBOARD-ANALYTICS.md` §2.1, task `T-071` (ADR-0026 akan mengikatnya).

### Q-DASH-03 — `review_due_at` / `expiry_at` / `published_at` untuk Document Control (NON-BLOCKING)

- **Konteks:** `Dashboard.md` §3 `Review Due / Overdue`, `Document Expiry / Review Calendar`, `Obsolete Documents` meminta field `review_due_at`, `expiry_at`, `published_at` pada `documents` yang tidak ada di `41-DATABASE.md` §2.3 (hanya `created_at`/`updated_at`/`archived_at`). MVP dashboard menggantikan "Due for Review" dengan "Revised This Month" dari `document_versions` — sisa masuk backlog.
- **Pertanyaan:** apakah review due dihitung `created_at + N hari` (tanpa kolom) atau disimpan eksplisit? Apakah `Published` = `approved + published_at` terisi?
- **Rekomendasi:** tunda; masuk Phase 5 dengan migrasi `012` bila keputusan ada — jangan meniru `updated_at + 30 hari` tanpa keputusan.

### Q-DASH-04 — `workflow_stage_history` presisi untuk Average Time per Stage (NON-BLOCKING)

- **Konteks:** `Dashboard.md` §2.2 `Average Time per Workflow Stage` butuh durasi per stage. Sumber yang ada: `workflow_instances` (satu `current_step`, satu deadline) + `workflow_actions` (aksi). Durasi stage presisi butuh `stage_started_at`/`stage_completed_at` per langkah, yang tidak ada. MVP memakai selisih dua aksi berturut (`52-*` §3.1) — estimasi, bukan ukuran.
- **Rekomendasi:** terima estimasi untuk MVP; bila presisi diminta, buat tabel `workflow_stage_transitions` baru (Phase 5) — jangan menambah trigger history tanpa ADR.

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
