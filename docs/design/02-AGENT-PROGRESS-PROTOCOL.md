# 02-AGENT-PROGRESS-PROTOCOL — Protokol Dokumentasi Progress (WAJIB)

**Proyek:** BWDCS — Business Workflow & Document Control System
**Versi:** 1.0.0
**Tanggal:** 2026-09-17
**Status:** WAJIB diikuti setiap agen dan setiap sesi

---

## 1. Perintah Inti

> **Setiap prompt yang dijalankan dan setiap file yang dibuat, diubah, atau dihapus WAJIB dicatat** di direktori `docs/progress/` sebelum turn diakhiri.

Ini bukan kebiasaan yang boleh ditinggalkan saat sibuk. Pekerjaan yang tidak tercatat dianggap **belum selesai**, walaupun kodenya sudah jalan, karena agen berikutnya tidak dapat memverifikasi atau melanjutkannya.

Dokumen ini tidak menggantikan `01-AGENT-WORKFRAME.md` (aturan kerja & antislop) maupun `12-DEVELOPMENT-WORKFLOW.md` (alur teknis & Definition of Done). Ketiganya berjalan bersamaan.

---

## 2. Kapan Protokol Ini Aktif

| Peristiwa | Kewajiban |
|---|---|
| Sesi / prompt baru dimulai | Baca `STATE.md`, `SESSION-LOG.md`, `TASKS.md`, `OPEN-QUESTIONS.md` lebih dulu |
| Sebelum mengubah file | Tulis rencana singkat di log prompt (bagian 3) |
| Setiap file dibuat/diubah/dihapus | Catat di `CHANGELOG.md` dan di log prompt sesi tersebut |
| Ada keputusan teknis | Buat/perbarui ADR di `docs/adr/` |
| Ada requirement mulai dikerjakan | Perbarui `TRACEABILITY.md` |
| Ada pertanyaan yang butuh user | Catat di `OPEN-QUESTIONS.md`, tandai `BLOCKING`/`NON-BLOCKING` |
| Task berubah status | Perbarui `TASKS.md` |
| Turn akan diakhiri | Lengkapi log prompt + `STATE.md` + `CHANGELOG.md` |

---

## 3. Artefak & Tanggung Jawab

| Artefak | Isi | Sifat | Alasan |
|---|---|---|---|
| `docs/progress/STATE.md` | Snapshot kondisi sekarang: fase, modul, environment, task aktif, next action | Ditimpa (selalu kondisi terkini) | Titik masuk agen berikutnya |
| `docs/progress/SESSION-LOG.md` | Riwayat entri sesi/prompt, terbaru di atas | Append-only | Menjawab "apa yang terjadi" |
| `docs/progress/CHANGELOG.md` | Daftar file Added/Changed/Fixed/Removed per tanggal | Append-only | Menjawab "apa yang berubah dan kenapa" |
| `docs/progress/TASKS.md` | Task `T-###` dengan status & bukti | Status boleh berubah, task tidak dihapus | Menjawab "apa yang sedang dikerjakan" |
| `docs/progress/TRACEABILITY.md` | Requirement `FR-*`/`NFR-*` → dokumen → file → test → bukti | Diperbarui bertahap | Menjawab "requirement mana yang benar-benar tertutup" |
| `docs/progress/OPEN-QUESTIONS.md` | Pertanyaan pending, blocker, temuan inkonsistensi dokumen | Status boleh berubah | Mencegah agen menebak keputusan user |
| `docs/progress/prompts/P-###-*.md` | Log satu prompt: prompt, rencana, aksi, perubahan, verifikasi, next action | Append-only | Jejak audit kerja agen |
| `CONTINUE.md` | Titik masuk resume untuk agen/model apa pun + blok snapshot §0 | Blok §0 diperbarui setiap sesi; instruksi §1-§9 stabil | Menjembatani perbedaan agen/model antar sesi |

---

## 4. Alur Wajib per Prompt

```
1. RESUME
   Baca CONTINUE.md, ikuti urutan bacanya (dokumen aturan -> ledger -> dokumen modul)
   Baca STATE.md -> SESSION-LOG.md (20 entri terakhir) -> 3 log prompt terakhir -> TASKS.md -> OPEN-QUESTIONS.md
   Pastikan tidak melanjutkan pekerjaan yang sudah selesai atau menggandakan yang sedang jalan.
2. TENTUKAN ID
   Ambil nomor prompt berikutnya di docs/progress/prompts/ (P-001, P-002, ...).
   Buka task T-### terkait atau buat task baru di TASKS.md.
3. RENCANA
   Tulis bagian 1-3 template log prompt SEBELUM mengubah file.
4. KERJAKAN
   Ikuti dokumen desain yang relevan (lihat routing di AGENTS.md).
5. VERIFIKASI
   Jalankan typecheck/build/test dan/atau link check dokumen. Rekam command + output ringkas.
   Klaim tanpa bukti tidak boleh berstatus DONE.
6. CATAT (tidak boleh dilewati)
   - lengkapi log prompt di docs/progress/prompts/
   - tambah entri di SESSION-LOG.md dan CHANGELOG.md
   - perbarui TASKS.md, TRACEABILITY.md, OPEN-QUESTIONS.md sesuai kebutuhan
   - perbarui STATE.md agar mencerminkan kondisi terakhir
   - perbarui blok snapshot §0 di CONTINUE.md
   - buat/perbarui ADR bila ada keputusan arsitektur
7. LAPORKAN
   Ringkas ke user: apa yang berubah, bukti verifikasi, apa yang belum, next action.
```

---

## 5. Format Minimum Log Prompt

Salin `docs/progress/prompts/TEMPLATE.md`. Bagian yang **tidak boleh kosong**: Prompt User, Rencana, Aksi, File yang Berubah, Verifikasi, Status, Next Action.

Nama file: `P-<nomor 3 digit>-<YYYY-MM-DD>-<slug>.md`.

Nomor yang sudah dipakai **tidak boleh didaur ulang**, walaupun log-nya ternyata salah. Perbaikan dilakukan dengan entri baru.

---

## 6. Aturan Bukti (Evidence)

| Status | Syarat |
|---|---|
| `DONE` | Ada perubahan nyata + command verifikasi + output ringkas + ledger lengkap |
| `PARTIAL` | Perubahan ada, verifikasi belum lengkap, atau ledger belum lengkap |
| `BLOCKED` | Butuh keputusan user/akses eksternal; sudah tercatat di `OPEN-QUESTIONS.md` |
| `FAILED` | Sudah dicoba, gagal, dan penyebabnya dicatat |

Dilarang menulis klaim seperti "sudah saya test" tanpa perintah dan hasilnya. Untuk pekerjaan dokumen (seperti sesi fondasi ini), verifikasi yang sah adalah link check referensi file, pemeriksaan konsistensi istilah, dan pembacaan ulang file hasil edit.

### 6.1 Angka tidak ditulis dari ingatan (kelas cacat C-044/C-055/C-057)

Empat kali angka pada ledger salah karena disalin dari sesi sebelumnya alih-alih dihitung ulang dari berkasnya: hitungan temuan audit (C-044, C-055), hitungan test per berkas dan total suite (C-057), dan rujukan test yang sudah diganti namanya (C-057, C-058). Aturan yang berlaku sejak P-036:

1. **Hitung ulang, jangan salin.** Setiap angka pada `TASKS.md`, `STATE.md`, `CONTINUE.md`, `AGENTS.md`, `audits/README.md`, dan laporan audit dihitung dari sumbernya pada sesi itu (mis. `grep -cE '^\| C-'` untuk temuan, `grep -c '^func Test'` untuk test).
2. **Marker mesin untuk hitungan audit.** Setiap laporan audit memuat satu baris:

   ```
   <!-- audit-summary total=N fixed=N approved=N open=N rejected=N rinci=N -->
   ```

   Angka wajib sama dengan isi tabel tindak lanjut di berkas yang sama (`total` = jumlah baris `| C-### |`, `fixed`/`approved`/`open`/`rejected` = jumlah baris dengan status itu, `rinci` = jumlah judul `### C-###`). Salinan marker yang **sama persis** ada di `docs/progress/audits/README.md`, `AGENTS.md`, `STATE.md`, dan `CONTINUE.md`.
3. **Satu task, satu kolom.** Papan `TASKS.md` tidak boleh memuat ID yang sama di dua kolom status, baris `DONE` wajib bertanggal, dan baris di kolom lain tidak boleh mengklaim selesai. Kewajiban berulang (mis. menjaga `CONTINUE.md`) tetap butuh ID sendiri.
4. **Rujukan test wajib hidup.** Nama test pada dokumen status (dan ADR) harus benar-benar ada di `backend/`; nama lama hanya boleh dikutip sebagai riwayat, dengan penanda `<!-- ledger-check: skip: <alasan> -->` di baris yang sama.

Semuanya diperiksa `scripts/check-ledger.sh`, dijalankan sebelum menutup sesi dan otomatis di CI (`.github/workflows/ci.yml`). Keluarannya `ledger OK` (exit 0) atau daftar `FAIL` (exit 1) — angkanya dihitung dari kode dan ledger, bukan dari klaim.

### 6.2 Angka di README diperiksa terhadap repo (kelas cacat C-061/C-062)

Ledger bukan satu-satunya tempat angka bisa basi. `README.md` memuat **keadaan repo** yang berubah setiap kali kode tumbuh — jumlah route, versi dependensi, rentang migrasi, dan pernyataan bahwa sesuatu "belum ada" — dan ketika itu basi tidak ada yang menangkapnya: tidak ada test yang membacanya, dan `check-ledger.sh` hanya memeriksa ledger. Dua sesi berturut-turut menemukan kekeliruan seperti itu di berkas yang paling sering dibaca orang luar (jumlah route **30 vs 32**, `frontend/` disebut kosong padahal 51 berkas sudah berdiri, dan baris "Frontend (rencana) ... React 18 ... Belum diinisialisasi" yang bertahan sesudah itu).

Sejak **P-039** ada pemeriksanya: `bash scripts/check-readme-facts.sh`. Ia **tidak** membaca dokumen lain sebagai kebenaran, melainkan menghitung dari sumbernya:

| Fakta di README | Sumber kebenaran |
|---|---|
| jumlah route + rincian per modul + endpoint per modul | `backend/internal/handler/router.go` (setiap group route wajib punya ember di skrip; group baru yang belum terklasifikasi **menggagalkan** pemeriksaan) |
| versi Go, Gin, pgx, viper, goose | `backend/go.mod` |
| versi React, Vite, Tailwind, TypeScript | `frontend/package.json` |
| rentang migrasi | berkas di `backend/internal/migration/` |
| klaim "belum ada/belum diisi" atas sesuatu yang sudah ada | keberadaan path di repo |
| daftar berkas di pohon folder §4 (kelas **C-066**, sejak P-042) | isi `scripts/` dan setiap path `scripts/...`/`skills/...` yang ditulis README — **dua arah**: yang ada wajib disebut, yang disebut wajib ada |

Keluarannya `readme-facts OK — N fakta diperiksa` (exit 0) atau daftar `FAIL` dengan nomor baris (exit 1).

**Pelajaran C-066 (P-042):** klaim berbentuk **himpunan** tidak tertangkap pemeriksa yang hanya membandingkan **angka**. Pohon folder README menyebut dua dari lima skrip yang sudah berjalan di CI selama dua sesi, dan tidak ada yang menyadarinya — justru karena `check-readme-facts.sh` sudah dipercaya menjaga README. Setiap daftar di dokumen yang dapat dibandingkan dengan isi direktori harus punya pembandingnya, dan pembandingnya memeriksa **kedua arah**; angka yang diperiksa mesin tidak membuat himpunannya ikut terjaga.

### 6.3 Izin endpoint tidak ditulis dari ingatan (kelas cacat T-024/Q-016)

`T-024` ("anotasi izin pada setiap endpoint") dirawat sebagai **hitungan manual** selama enam sesi, dan angkanya pernah tidak dapat diperiksa silang (**C-055**). Lebih buruk: pasangan izin yang tidak ada di matriks pernah ditulis di kontrak — `document_version:read` di draf §4, padahal matriks §3.1.2 tidak memuatnya (Q-016) — dan itu hanya ketahuan karena ada manusia yang membandingkan dua dokumen.

Sejak **P-040** ada pemeriksanya: `bash scripts/check-api-contract.sh`. Ia membaca matriks §3.1.2 dari `44-SECURITY.md` sebagai sumber kebenaran, lalu menuntut tiga hal:

1. Setiap endpoint di `42-API.md` punya izin yang **terbaca** — baris `Izin:` di dalam bloknya, atau baris di tabel izin babnya (bentuk `| \`GET /documents\` | \`document:read\` |`).
2. Setiap pasangan `resource:action` pada baris `Izin:` dan pada tabel izin wajib ada di matriks.
3. Setiap `RequirePermission(deps.Permission, "<resource>", "<action>")` di `internal/handler/router.go` wajib memakai pasangan dari matriks yang sama — inilah yang menahan kode dan matriks menyimpang satu sama lain.

Pemeriksa ini **tidak** mengesahkan pasangan baru: menambah pasangan izin tetap menuntut ADR (ADR-0014). Yang dilakukannya adalah menahan pasangan karangan, dan mengganti hitungan manual dengan hitungan mesin — 55 endpoint, 48 beranotasi di bloknya dan 7 lewat tabel izin bab, 18 pasangan izin di router. Tulisannya harus **menghindari menuliskan pasangan yang tidak ada sebagai pasangan**: kalimat yang menjelaskan ketiadaan (mis. "matriks tidak memuat pasangan `update` untuk `comment`") ditulis tanpa token `resource:action`, karena pemeriksa membacanya sebagai klaim. **Klaim yang polanya hilang dari README juga dianggap gagal**, karena pemeriksa yang diam-diam berhenti memeriksa lebih berbahaya daripada pemeriksa yang berisik. Batasnya (hanya README, hanya pola tetap, hanya angka 1..10 dalam huruf) ditulis di kepala skrip. Kalau pemeriksanya gagal: perbaiki README atau sumbernya — **jangan** melunakkan skripnya.

### 6.4 Rujukan antislop dan berkas pihak ketiga (kelas cacat C-064/C-065)

Kelas cacat ketiga bukan angka yang basi, melainkan **rujukan yang menunjuk sesuatu yang tidak ada atau tidak sama**. `AGENTS.md` sudah lama mendaftarkan lima skill antislop (`skills/antislop-ui/SKILL.md` dan empat lainnya) padahal direktori `skills/` belum pernah dibuat (**C-064**), dan berkas core `antislop.md` di root ternyata **varian lama** yang berbeda dari salinan Delivery Gate di `01-AGENT-WORKFRAME.md` §5.3, sehingga dua versi core beredar di satu repo (**C-065**). Keduanya tidak terlihat dari membaca: yang pertama hanya muncul ketika agen mencoba membuka skill-nya, yang kedua hanya muncul ketika dua berkas dibandingkan mesin.

Sejak **P-042** ada pemeriksanya: `bash scripts/check-antislop-refs.sh`, tanpa jaringan. Enam aturannya:

| Diperiksa | Sumber kebenaran |
|---|---|
| daftar aturan (`R-01..R-38`) terbaca dan berurutan | heading `#### R-XX —` di `antislop.md` — skrip **berhenti** bila daftarnya tidak terbaca, supaya tidak lulus secara hampa |
| setiap `R-XX` yang dirujuk dokumen ada di daftar | `antislop.md` |
| setiap path `skills/...` di blok penunjuk `AGENTS.md` ada di disk, dan setiap berkas instruksi `skills/<nama>/SKILL.md` di disk terdaftar | disk |
| `sha256` setiap berkas antislop | tabel provenans `skills/README.md` §1 |
| salinan core di `skills/antislop/SKILL.md` (tanpa frontmatter) | `antislop.md` |
| kalimat khas aturan/Gate upstream tidak tersalin ke dokumen proyek | daftar sidik jari di skrip |

Keluarannya `antislop-refs OK — N aturan, M rujukan, K berkas skill` (exit 0) atau `FAIL` dengan nomor baris (exit 1). Tiga aturan yang mengikat penggunaannya: (a) **jangan melunakkan skripnya** untuk lulus; (b) **jangan memperbarui `sha256`** tanpa benar-benar mengambil berkas dari tag rilis yang dicatat; (c) **jangan menyalin aturan ke dokumen proyek** — dokumen hanya menunjuk ke `antislop.md`, karena salinan yang tidak diperiksa mesin selalu kalah dari sumbernya, dan itulah cara C-065 lahir. Skill dipin ke tag rilis dan cara menaikkannya ada di `skills/README.md` §2; keputusannya di **ADR-0025**.

Batas yang disadari: skrip ini tidak dapat menilai apakah **keputusan desain** proyek taat pada aturannya. Itu tugas Delivery Gate (`01-AGENT-WORKFRAME.md` §5.3) dan tinjauan user.

### 6.5 Batas sidebar (kelas cacat C-079)

Kelas cacat keempat bukan rujukan mati, melainkan **aturan desain yang tidak dijaga apa pun**. `51-UX.md` §2.1 menetapkan bahwa sidebar memuat **modul saja**: satu entri per modul, tanpa sub-item, dan setiap kueri penyaring hidup di halamannya sendiri. Aturan itu ditegakkan di klien oleh `frontend/src/config/navigation.ts`, tetapi **tidak satu pun pemeriksa menjaganya**: CI tidak punya job frontend sama sekali (tidak ada `npm test` di `.github/workflows/ci.yml`), sehingga test klien tidak pernah berjalan di sana, dan `scripts/responsive-evidence.mjs` hanya mencari item menu yang membawa **kueri** — path bersarang tidak dilihatnya. Jadi `Reports > Audit` dapat berdiri sebagai entri sidebar, menyimpang dari tabel dan diagram §2, tanpa ada yang gagal.

Sejak **P-051** ada pemeriksanya: `bash scripts/check-navigation.sh`, tanpa perkakas tambahan (tanpa Node, tanpa runner TypeScript). Ia memeriksa **dua arah**, keduanya terhadap sumbernya:

| Diperiksa | Sumber kebenaran |
|---|---|
| setiap `path` bebas kueri dan karakter fragmen | model navigasi |
| entri sidebar maksimal **satu segmen** (`/`, `/projects`) | aturan §2.1 |
| label menu tidak berbentuk remah (`X > Y`) | aturan §2.1 |
| halaman anak menyebut induknya, induknya ada, dan pathnya benar-benar di bawah induk itu | model navigasi |
| tidak ada path atau label menu yang dipakai dua kali | model navigasi |
| himpunan menu sidebar = himpunan baris §2.1 yang **bukan** `X > Y`, dan himpunan halaman anak = baris `X > Y` beserta induknya | tabel `51-UX.md` §2.1 |
| teks aturan §2.1 masih menyatakan "modul saja" | `51-UX.md` §2.1 |

Keluarannya `navigation OK — N menu sidebar, M halaman anak, K baris §2.1 cocok` (exit 0) atau `FAIL` (exit 1). Dua hal yang membedakannya dari pemeriksa lain di berkas ini. Pertama, ia **membaca teks sumbernya sendiri** (berkas TypeScript dan tabel Markdown) karena repo ini tidak punya job frontend di CI — konsekuensinya bentuk berkas yang tidak dikenali harus **gagal**, dan itu diuji: mengganti nama kunci `label:` menjadi `title:` membuat versi pertamanya melaporkan 25 kegagalan yang semuanya menyesatkan, sehingga penjaga bentuk ditambahkan dan kini pesannya menyebut sebabnya (**C-080**). Kedua, arah **dokumen → model** diperiksa sama kerasnya dengan **model → dokumen**, jadi menambah menu di kode tanpa baris di §2.1 gagal, dan sebaliknya juga. Kalau pemeriksanya gagal: perbaiki model navigasi atau tabel §2.1 — **jangan** melunakkan skripnya, dan **jangan** melonggarkan §2.1 untuk membuat skripnya tenang.

Batasnya: ia tidak memeriksa rute yang memang bukan halaman bernavigasi (mis. `/projects/:id`), dan ia tidak menilai apakah sebuah menu **seharusnya** ada — hanya apakah bentuknya taat aturan.

---

## 7. Anti-drift (Dokumen vs Kode)

Protokol ini juga alat anti-drift, bukan sekadar administrasi:

1. Jika implementasi berbeda dari dokumen desain, **ubah dokumennya di sesi yang sama** dan catat di `CHANGELOG.md`.
2. Jika keputusan berubah (misalnya ganti library), **ADR baru** wajib dibuat dan ADR lama diberi status `SUPERSEDED`.
3. Jika menemukan dokumen yang saling bertentangan, catat di `OPEN-QUESTIONS.md` §2 (temuan inkonsistensi) lalu selesaikan: perbaiki dokumen yang salah, atau ubah keputusan lewat ADR.
4. Requirement yang sudah diimplementasikan tetapi tidak muncul di `TRACEABILITY.md` dianggap belum tertutup.

---

## 8. Self-check Sebelum Turn Diakhiri

- [ ] Log prompt sesi ini ada dan lengkap
- [ ] Semua file yang disentuh tercatat di `CHANGELOG.md`
- [ ] `STATE.md` mencerminkan kondisi setelah perubahan
- [ ] `SESSION-LOG.md` punya entri baru
- [ ] Status task di `TASKS.md` sudah benar (dan bukti terisi bila `DONE`)
- [ ] Requirement yang disentuh diperbarui di `TRACEABILITY.md`
- [ ] `bash scripts/check-ledger.sh` → `ledger OK` (hitungan audit, papan kerja, dan rujukan test konsisten)
- [ ] `bash scripts/check-doc-links.sh` → `BROKEN referensi dokumen: 0`, `bash scripts/check-readme-facts.sh` → `readme-facts OK`, `bash scripts/check-api-contract.sh` → `api-contract OK`, `bash scripts/check-antislop-refs.sh` → `antislop-refs OK`, dan (bila menyentuh navigasi) `bash scripts/check-navigation.sh` → `navigation OK`
- [ ] Blok snapshot §0 di `CONTINUE.md` diperbarui
- [ ] Pertanyaan/blocker baru ada di `OPEN-QUESTIONS.md`
- [ ] Keputusan arsitektur baru ada di `docs/adr/`
- [ ] Bila UI: checklist `01-AGENT-WORKFRAME.md` §5.2 dan Delivery Gate §5.3 dijalankan

Jika salah satu tidak bisa dipenuhi, tulis alasannya di log prompt dan di `OPEN-QUESTIONS.md`. Melewatinya tanpa catatan adalah pelanggaran protokol.

Catatan lintas agen: karena agen dan model bisa berbeda antar sesi, log prompt dan `CONTINUE.md` harus dapat dibaca tanpa konteks percakapan apa pun. Larangan lengkapnya ada di `CONTINUE.md` §7.

---

## 9. Contoh Ringkas

```md
## P-014 — 2026-09-19 — Auth login handler

Aksi: implementasi POST /api/v1/auth/login, bcrypt cost 12, rate limit 5/15 menit.
File: backend/internal/handler/auth_handler.go (Added), backend/internal/service/auth_service.go (Changed),
      docs/progress/STATE.md (Changed), docs/progress/CHANGELOG.md (Changed).
Bukti: `go test ./internal/service/...` -> ok (3 test), `go build ./...` -> sukses,
       `curl -X POST /api/v1/auth/login` -> 401 untuk kredensial salah.
Requirement: FR-AUTH-01, FR-AUTH-05, FR-AUTH-06 -> TRACEABILITY.md diperbarui.
Status: DONE. Next: middleware RBAC (T-005 lanjutan).
```
