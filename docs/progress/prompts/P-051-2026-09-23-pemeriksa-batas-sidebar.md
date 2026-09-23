# P-051 — 2026-09-23 — Penegak mesin untuk batas sidebar, dan tiga cacat yang lahir dari membuatnya

| Field | Isi |
|---|---|
| ID | P-051 |
| Waktu mulai | 2026-09-23 (lanjutan sesi P-050 pada worktree yang sama) |
| Aktor | agen (Buffy, perkakas berkas + terminal) |
| Model / agen | deepseek/deepseek-v4-flash (dimulai sebagai z-ai/glm-5.3-flash untuk pembacaan dokumen) |
| Fase roadmap | 4 (UI) — pekerjaan infrastruktur aturan, bukan halaman baru |
| Task terkait | `T-067` (halaman anak di sub-navigasi, DONE), `T-068` (pemeriksa batas sidebar, DONE) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Buat pemeriksa yang menolak item sidebar baru yang membawa kueri penyaring atau menunjuk
> sub-halaman, supaya aturan 51-UX.md §2.1 tidak dapat dilanggar diam-diam."

## 2. Interpretasi & Scope

- **Yang diminta:** penegak **mesin** untuk aturan `51-UX.md` §2.1 (sidebar memuat modul saja).
- **Yang TIDAK termasuk (out of scope):** menambah menu baru, mengubah izin, menyentuh backend,
  kontrak, atau skema. Tidak ada halaman baru yang dibangun sesi ini.
- **Asumsi yang diambil:**
  - Permintaannya bukan "tambahkan satu test klien lagi". Test klien **tidak berjalan di CI**:
    `.github/workflows/ci.yml` hanya punya job `ledger` (skrip dokumen) dan job `backend`
    (`go build`/`vet`/`gofmt`/`make test`) — tidak ada `npm test`. Jadi pemeriksanya harus berdiri
    sebagai **skrip shell** yang menghitung dari berkasnya sendiri, seperti lima pemeriksa lain.
  - Karena skrip itu **membaca teks** model navigasi (repo ini tidak punya runner TypeScript untuk
    skrip pemeriksa), bentuk berkas yang tidak dikenali wajib **gagal** — bukan dilewati. Itu bukan
    detail teknis, melainkan inti kepercayaannya.
  - **Satu keputusan diminta ke user lewat pilihan berganda** sebelum menulis pemeriksanya: halaman
    **Audit** (di bawah Reports) sudah punya rute dan izin, tetapi tidak punya jalan masuk kalau ia
    tidak boleh menjadi entri sidebar. Pilihan yang diambil: daftarkan sebagai **halaman anak**
    (`subPages`, kolom `parent`) yang muncul di baris sub-navigasi modul induknya — bukan
    mengembalikannya ke sidebar, dan bukan membiarkannya tanpa jalan masuk. Itulah `T-067`.
- **Pertanyaan yang muncul:** tidak ada pertanyaan baru yang menunggu keputusan. `Q-016`, `Q-023`,
  `Q-024` tetap terbuka dan tidak tersentuh.

## 3. Rencana

1. Baca model navigasi, tabel §2.1, dan daftar pemeriksa/CI untuk tahu apa yang sudah dijaga mesin.
2. Beri halaman anak jalan masuk lewat baris sub-navigasi (`T-067`) supaya aturannya dapat ditegakkan
   tanpa mengorbankan halaman.
3. Tulis `scripts/check-navigation.sh` yang memeriksa **model → dokumen** dan **dokumen → model**.
4. Buktikan giginya dengan menyuntikkan cacat secara sementara — termasuk cacat yang **bukan**
   pelanggaran aturan (kerusakan bentuk berkas).
5. Sambungkan ke CI dan dokumen, catat temuan, tutup ledger, jalankan keenam pemeriksa.

## 4. Aksi yang Dilakukan

1. **Model navigasi (`T-067`).** `frontend/src/config/navigation.ts` memisahkan `navigation`
   (sidebar: modul saja, satu segmen, tanpa kueri) dan `subPages` (halaman bernavigasi yang **bukan**
   menu, dengan `parent`), plus `visibleSubPages()` dan `subNavFamily()`.
2. **Rute & baris sub-navigasi.** `App.tsx` membangun rute pending dari **kedua** daftar sehingga
   `/reports/audit` hidup walau modul induknya masih halaman penjelasan (R-24), dan
   `ModulePending.tsx` merender baris sub-navigasi keluarganya — bentuk yang sama dengan tab
   Documents/Tasks, hanya tautannya menunjuk **path lain**, bukan kueri penyaring.
3. **Pemeriksa (`T-068`).** `scripts/check-navigation.sh` menolak entri sidebar berkueri/berfragmen,
   entri sidebar lebih dari satu segmen, label bergaya remah, halaman anak tanpa induk/berinduk
   hantu/di luar induknya, path atau label ganda, dan himpunan menu yang menyimpang dari tabel §2.1
   (dua arah); ia juga menuntut teks §2.1 masih menyatakan "modul saja".
4. **Penjaga bentuk.** Menghitung jumlah menu **tidak cukup**: kunci `label:` yang berganti nama
   membuat parser memancarkan tujuh objek berisi kosong, penjaga hampa tidak menyala, dan hasilnya 25
   kegagalan menyesatkan. Kini objek tanpa label/path/induk membuat skrip **berhenti** dengan satu
   sebab yang benar.
5. **Sambungan.** CI (langkah keenam job `ledger` + komentar kepala), `README.md` §4/§12.3,
   `02-AGENT-PROGRESS-PROTOCOL.md` §6.5/§8, `12-DEVELOPMENT-WORKFLOW.md` §8, `51-UX.md` §2.1,
   `AGENTS.md`, dan `70-TESTING.md` §3.14h.
6. **`check-ledger.sh` diperluas** dengan aturan **C-081**: ringkasan jumlah temuan **di dalam**
   laporan audit wajib sama dengan marker `audit-summary`.
7. **Anggaran waktu suite dinaikkan** sesudah flakiness terukur (**C-082**): `asyncUtilTimeout: 5000`
   di `src/test/setup.ts` dan `testTimeout: 20000` di `frontend/vite.config.ts`.

## 5. Bukti

| Pemeriksaan | Hasil |
|---|---|
| `bash scripts/check-navigation.sh` | `navigation OK — 7 menu sidebar (tanpa kueri, satu segmen), 1 halaman anak ber-induk, 8 baris §2.1 cocok` |
| Gigi pemeriksa (7 cacat disuntikkan sementara) | ketujuhnya tertangkap (rinciannya `70-TESTING.md` §3.14h); berkas dipulihkan **byte-identik** (`diff -q`) |
| Gigi aturan ledger baru (C-081) | empat cacat disuntikkan sementara, keempatnya tertangkap: §1 → `75` (`FAIL … :28: menulis 75, marker bilang 82`), `> Ringkasan:` → `78` (`FAIL … :470`), §1 kehilangan penanda tebal (`FAIL … paragraf §1 tidak lagi cocok pola`), dan baris `> Ringkasan:` berubah bentuk (`FAIL … tidak lagi cocok pola`) — berkas dipulihkan byte-identik |
| Frontend | `tsc --noEmit` + `eslint .` bersih, **254 test / 25 berkas** hijau **tiga kali berturut-turut** pada worker bawaan, `vite build` 456,31 kB js / 23,43 kB css |
| Enam pemeriksa | `ledger OK — 0 peringatan, 268 test di backend`, `BROKEN referensi dokumen: 0`, `readme-facts OK — 46 fakta`, `api-contract OK — 115 pemeriksaan`, `antislop-refs OK — 38 aturan, 184 rujukan, 7 berkas skill`, `navigation OK` |
| Backend | **tidak disentuh** (tanpa perubahan kode, kontrak, izin, atau skema; versi goose tetap 11) |

## 6. Temuan

Empat temuan, semuanya lahir dari pekerjaan sesi ini dan semuanya sudah `FIXED`:

- **C-079 — aturan §2.1 tidak dijaga pemeriksa apa pun.** CI tidak punya job frontend, sehingga test
  klien tidak pernah berjalan di sana, dan `scripts/responsive-evidence.mjs` hanya mencari item menu
  yang membawa **kueri** — path bersarang luput. Halaman anak karena itu dapat masuk sidebar tanpa ada
  yang gagal. Ditutup `T-068`.
- **C-080 — penegak barunya sendiri melaporkan kegagalan untuk sebab yang salah.** Penjaga hampa yang
  hanya menghitung jumlah menu tidak menyala saat bentuk berkasnya berubah. Ditutup penjaga bentuk.
- **C-081 — total temuan di dalam laporan audit sendiri tertinggal tiga sesi.** Aturan prosa
  `check-ledger.sh` hanya membaca baris yang menyebut `AUDIT-001`, sedangkan paragraf §1 ada di berkas
  audit itu sendiri; ia menulis **75** padahal marker sudah **78** sejak P-050. Ditutup dengan aturan
  baru di `check-ledger.sh`, dan giginya dibuktikan dengan empat cacat. **Versi pertama aturannya
  sendiri mengulang cacat C-080 di sesi yang sama:** regexnya dikirim ke `awk` lewat `-v`, dan `-v`
  memproses escape sehingga `\*` menjadi `*` — polanya tidak pernah cocok, perbandingannya tidak
  berjalan sekali pun, dan skripnya melaporkan **OK**. Ketahuan justru karena giginya diuji (cacat
  disuntikkan, tidak ada kegagalan yang muncul); polanya kini ditulis di dalam program `awk`, dan kedua
  tempat ringkasan diperiksa **terpisah** supaya satu tempat yang berubah bentuk tidak berhenti
  diperiksa tanpa suara.
- **C-082 — suite frontend gagal berpindah-pindah antar-berkas pada kode yang sama.** Tiga dari lima
  kali `npm run test:run` gagal, file yang gagal berbeda tiap kali, sementara `--maxWorkers=1` dan
  `=2` hijau penuh pada kode yang sama: yang bermasalah adalah **jendela tunggu bawaan** (1000ms)
  sebagai klaim tentang kecepatan mesin, bukan aplikasi. Ditutup dengan menaikkan anggaran waktu
  **tanpa melonggarkan satu asersi pun**; tiga kali jalannya hijau berturut-turut.

## 7. Dokumen & Berkas yang Disentuh

| Berkas | Perubahan |
|---|---|
| `scripts/check-navigation.sh` | **Baru** — pemeriksa batas sidebar (dua arah, penjaga bentuk) |
| `frontend/src/config/navigation.ts`, `App.tsx`, `pages/ModulePending.tsx` | Daftar `subPages` + baris sub-navigasi keluarga (T-067) |
| `frontend/src/config/navigation.test.ts`, `components/layout/AppShell.test.tsx` | Test model navigasi & sidebar |
| `frontend/src/test/setup.ts`, `frontend/vite.config.ts` | Anggaran waktu suite (C-082) |
| `scripts/check-ledger.sh` | Aturan baru: ringkasan §1 laporan audit vs marker (C-081) |
| `.github/workflows/ci.yml` | Langkah keenam job `ledger` + komentar kepala |
| `README.md` | Pohon §4 + perintah verifikasi §12.3 |
| `docs/design/02-AGENT-PROGRESS-PROTOCOL.md` | §6.5 baru + checklist §8 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Perintah + paragraf pemeriksa baru |
| `docs/design/51-UX.md` | §2.1: batas itu dijaga mesin |
| `docs/design/70-TESTING.md` | §3.14h (bukti pemeriksa + anggaran waktu) |
| `AGENTS.md` | Blok aturan batas sidebar + daftar pemeriksa + angka audit |
| `docs/progress/*`, `CONTINUE.md` | TASKS (`T-067`/`T-068`), STATE, SESSION-LOG, CHANGELOG, audit (C-079..C-082), TRACEABILITY, snapshot |

## 8. Self-check Penutup

- [x] Log prompt sesi ini ada dan lengkap
- [x] Semua berkas yang disentuh tercatat di `CHANGELOG.md`
- [x] `STATE.md` mencerminkan kondisi setelah perubahan
- [x] `SESSION-LOG.md` punya entri baru
- [x] Status task di `TASKS.md` benar (`T-067`, `T-068` `DONE` bertanggal dan ber-bukti)
- [x] `TRACEABILITY.md` diperbarui untuk requirement yang disentuh
- [x] `bash scripts/check-ledger.sh` → `ledger OK`
- [x] Lima pemeriksa lain → `BROKEN: 0`, `readme-facts OK`, `api-contract OK`, `antislop-refs OK`, `navigation OK`
- [x] Blok snapshot §0 `CONTINUE.md` diperbarui
- [x] Tidak ada keputusan arsitektur baru (tidak ada ADR baru yang dibutuhkan: aturannya sudah ada di §2.1)
- [x] Checklist UI `01-AGENT-WORKFRAME.md` §5.2 tidak berubah perilakunya (tidak ada halaman baru dibangun)

## 9. Next Action

**Halaman Approvals (`50-FSD.md` §5.4)** — satu-satunya halaman bernavigasi yang endpointnya sudah hidup
(`GET /workflows/instances`, modul Workflow selesai P-048) tetapi halamannya belum dibangun. Dua hal yang
mengikat saat itu dikerjakan: (a) halaman itu **wajib** ditambahkan ke daftar `pages` di
`scripts/responsive-evidence.mjs` supaya sapuan per lebarnya tidak terlewat seperti C-078, dan (b) baris
§2.1 untuk modul itu sudah ada, jadi `bash scripts/check-navigation.sh` akan menuntut entri barunya tetap
berupa **modul** (satu segmen, tanpa kueri) sementara tab antreannya hidup di halamannya sendiri.
