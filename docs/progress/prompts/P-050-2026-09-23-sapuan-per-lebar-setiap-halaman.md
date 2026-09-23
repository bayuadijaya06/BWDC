# P-050 — 2026-09-23 — Sapuan tata letak per lebar untuk setiap halaman, dan tiga temuan yang lahir darinya

| Field | Isi |
|---|---|
| ID | P-050 |
| Waktu mulai | 2026-09-23 (lanjutan sesi P-049 pada worktree yang sama) |
| Aktor | agen (Buffy, perkakas berkas + terminal + Preview) |
| Model / agen | deepseek/deepseek-v4-flash |
| Fase roadmap | 4 (UI) |
| Task terkait | `T-066` (baru, DONE di sesi ini) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Jalankan pengukuran tata letak per lebar untuk halaman Tasks dan Documents, bukan hanya Projects."

## 2. Interpretasi & Scope

- **Yang diminta:** pengukuran tata letak per lebar dijalankan untuk halaman **Tasks** dan
  **Documents**, tidak hanya **Projects**.
- **Yang TIDAK termasuk (out of scope):** menambah halaman baru, menyentuh backend/kontrak/izin/skema,
  dan mengubah semantik penyaring mana pun.
- **Asumsi yang diambil:**
  - Permintaannya bukan "tambahkan dua baris ke tabel di dokumen", melainkan "buat sapuannya benar".
    Sapuan per lebar sejak P-043 hanya mengunjungi `/projects`; halaman lain diukur **hanya di dua
    ujung** lebar, sehingga cacat yang hanya muncul di lebar tengah tidak terlihat dari keduanya.
    Karena itu yang diubah adalah **cakupan sapuan**: setiap halaman × setiap lebar.
  - Menjalankan pengukuran berarti **menangani apa yang ditemukannya**. Itu bukan perluasan lingkup
    yang dikarang: hasil pengukuran adalah temuan, dan `docs/design/02-AGENT-PROGRESS-PROTOCOL.md`
    mewajibkan temuan dicatat **dan** ditutup pada sesi yang sama bila tidak menunggu keputusan siapa pun.
- **Pertanyaan yang muncul:** tidak ada yang baru. `Q-016`/`Q-024` tetap terbuka dan tidak tersentuh.

## 3. Rencana

1. Perluas sapuan per lebar dan per tema ke **setiap** halaman yang berdiri.
2. Jalankan pemeriksaan struktural (sidebar, baris penyaring Task, bilah tab Documents) di **setiap**
   lebar, bukan hanya di dua ujung.
3. Tambahkan stres data: sisipkan pilihan sengaja panjang ke setiap `select` penyaring supaya aturannya
   tidak bergantung pada data yang kebetulan ada.
4. Jalankan, baca hasilnya, dan pisahkan cacat nyata dari cacat alat ukurnya.
5. Tutup keduanya, buktikan gigi pemeriksa baru dengan memasang kembali cacatnya, lalu selaraskan
   dokumen dan ledger.

## 4. Aksi yang Dilakukan

1. `scripts/responsive-evidence.mjs`: sapuan per lebar dan per tema kini berjalan untuk
   `projects`/`tasks`/`documents` (15 pengukuran tata letak, 18 pengukuran tema), dan pada **setiap**
   lebar ikut diperiksa sidebar, bilah tab Documents, dan baris penyaring Task.
2. Pemeriksaan rentang tenggat diberi ambang dari kelas `sm:` elemennya (640px) alih-alih daftar lebar
   yang dirawat tangan, dan cacat "kolom melanjutkan di bawah kontrolnya" dibatasi pada kolom berisi
   **satu** kontrol — kendali majemuk (dua isian dalam satu kelompok) kini dilaporkan terpisah sebagai
   `liftedComposite` supaya pengecualiannya terbaca.
3. Probe **pilihan sengaja panjang** ditambahkan ke setiap `select` penyaring di ketiga halaman, dan
   yang dilaporkan adalah **pertambahan** lebar halaman, bukan luas sesudahnya.
4. `resize()` tidak lagi mengandalkan jeda tetap: ia menunggu tata letak berhenti berubah.
5. Kelima cacat alat ukur ditutup (cakupan, argumen berspasi, balapan jeda, kendali majemuk, tuduhan
   salah alamat) beserta satu fase hampa yang kini dilewati dengan alasan tertulis.
6. Tiga cacat antarmuka ditutup: `min-w-0` pada setiap kolom ber-`select` di ketiga halaman, dan
   `tap-target` menetapkan **kedua sisi** kotak sentuh.

## 5. File yang Berubah

### Added

| File | Keterangan |
|---|---|
| `frontend/src/styles/tap-target.test.ts` | 3 test membaca `tokens.css`: utility menetapkan **keduanya** (`min-height` **dan** `min-width`) pada kedua rentang lebar, dan tidak mengunci `width`/`height` tetap — P-050 |

### Changed

| File | Keterangan |
|---|---|
| `scripts/responsive-evidence.mjs` | Sapuan per lebar & tema untuk **setiap** halaman; pemeriksaan sidebar/tab/penyaring di setiap lebar; probe pilihan panjang; penantian tata letak yang tenang; argumen berspasi diterima; laporan yang tidak lagi menuduh salah alamat — P-050 |
| `frontend/src/styles/tokens.css` | `@utility tap-target` menetapkan `min-width` juga (44px, 36px di ≥1024px) — P-050 |
| `frontend/src/pages/Projects/index.tsx` | `min-w-0` pada kolom penyaring ber-`select` + alasan mengapa aturannya berlaku untuk semua kolom serupa — P-050 |
| `frontend/src/pages/Documents/index.tsx` | Idem untuk dua kolom penyaringnya — P-050 |
| `frontend/src/pages/Tasks/index.tsx` | Idem untuk lima kolom penyaringnya; kolom rentang tenggat sengaja **tidak** diberi `min-w-0`, beserta alasannya — P-050 |
| `frontend/src/pages/Projects/Projects.test.tsx`, `frontend/src/pages/Documents/Documents.test.tsx`, `frontend/src/pages/Tasks/Tasks.test.tsx` | Satu test per halaman yang mengunci `min-w-0` pada setiap kolom ber-`select`, dengan penjaga hampa — P-050 |
| `docs/design/51-UX.md` | §2.1: aturan kolom penyaring yang dapat menyusut dan ambang sentuh yang berupa kotak, beserta pengukuran yang melatarbelakanginya — P-050 |
| `docs/design/70-TESTING.md` | §3.14b (catatan cakupan baru), §3.14f (tuntutan rentang kini diukur di setiap lebar), dan **§3.14g** (baru) — P-050 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | Tiga temuan baru: **C-076**, **C-077**, **C-078** (`FIXED`); hitungan **78/76/0/2** — P-050 |
| `docs/progress/audits/README.md`, `AGENTS.md`, `docs/progress/STATE.md`, `CONTINUE.md` | Marker `audit-summary` dan prosa hitungannya diselaraskan ke 78/76/0/2 — P-050 |

## 6. Verifikasi (WAJIB)

**Frontend:** `tsc --noEmit` + `eslint .` bersih, **247 test / 25 berkas** hijau (naik dari 241/24),
`vite build` 455,5 kB js / 23,4 kB css.

**Bukti peramban nyata** (`node scripts/responsive-evidence.mjs --widths 375,640,768,1024,1440` →
`OK: halaman projects/tasks/documents × lebar 375/640/768/1024/1440px = 15 pengukuran tata letak,
ambang 44/36px, 18 pengukuran tema, laci 375px membereskan dirinya`):

| Ukuran | Hasil terukur |
|---|---|
| Gulir mendatar (`overflowX`) 3 halaman × 5 lebar | **0 px** di kelimanya belas |
| Kontrol di bawah ambang lebarnya sendiri | **0** di setiap halaman × lebar |
| State sidebar (laci ≤767px, kolom ≥768px) | sesuai `51-UX.md` §8 di kelima lebar |
| Stres 8 `select` penyaring (pertambahan lebar) | **0 px** di semua halaman × lebar |
| Rentang tenggat: `sameLine` / `sameGroup` | 375px `false`/`true`; 640-1440px `true`/`true`; `contained = 0` |
| Bilah tab Documents: garis / dapat digulir / tak terjangkau | 375px 2 garis, ≥640px 1 garis; tidak dapat digulir; **0** tak terjangkau |
| Tema terang/gelap | 18 pengukuran, **0** gulir mendatar |

**Dua cacat antarmuka yang ditemukan dan ditutup** (keduanya hanya muncul di halaman yang tadinya tidak
disapu, dan keduanya terukur, bukan diduga):

1. `#penyaring-project-dokumen` **403px** pada viewport 375px → halaman menggulir mendatar **44px**.
   Sebabnya min-width otomatis item flex = min-content anaknya, dan min-content `<select>` ditentukan
   teks pilihan terpanjang — **data pengguna**. Halaman **Projects** menyimpan cacat yang sama meski
   pilihannya pendek: probe stres mengukurnya **65px**.
2. Tab sub-halaman **"Tim"** terukur **43x44px** pada 375px dan 768px — ambang sentuh hanya dipasang
   pada `min-height`, sehingga satu label pendek cukup untuk membuat kotaknya tidak memenuhi ambangnya.

**Gigi dibuktikan** dengan memasang kembali dua cacat sekaligus (`min-w-0` kolom Project Documents dan
`min-width` pada `tap-target`) → skrip bukti **FAIL dengan enam butir**, termasuk
`halaman tasks @ 375px: 1 kontrol di bawah 44px (Tim=43x44)`,
`halaman documents @ 375px: gulir mendatar 44px (… #penyaring-project-dokumen …)`, dan
`pilihan penyaring yang panjang melebarkan halaman 21px`. Keduanya dipulihkan, lalu hijau.

**Lima pemeriksa hijau:** `ledger OK — 0 peringatan, 268 test di backend`, `BROKEN referensi dokumen: 0`,
`readme-facts OK — 44 fakta`, `api-contract OK — 115 pemeriksaan, 55 endpoint`,
`antislop-refs OK — 38 aturan … 8 pemeriksaan`. **Tanpa perubahan backend, kontrak API, izin, atau
skema.**

**Catatan lingkungan (bukan cacat kode).** Laporan di atas dijalankan dengan `ADMIN_PASSWORD` dari
lingkungan, bukan dari `.env`, karena `.env` di checkout ini diubah pada **2026-09-23 14:05** menjadi
nilai yang **tidak cocok** dengan password admin di database: login `.env` sekarang menjawab
`401 INVALID_CREDENTIALS`, sedangkan nilai `.env` **saat sesi dimulai** masih menjawab `200`. Baris
`users` tidak berubah sejak 2026-09-19 22:43 (dan tidak terkunci), jadi yang bergeser adalah berkasnya,
bukan databasenya. Skrip bukti memang membaca `process.env.ADMIN_PASSWORD` lebih dulu daripada `.env`,
dan memakai variabel itu membuat pengukuran tetap berjalan **tanpa menyentuh kredensial siapa pun**.
Pemilik perlu memutuskan mana yang benar (menyelaraskan `.env` dengan database, atau mengganti password
lewat `POST /auth/change-password`); agen tidak mengubah kredensial atas inisiatif sendiri.

## 7. Hasil & Dampak

- Klaim tata letak kini diukur **per halaman**, bukan per halaman pertama. Cakupan itu sendiri menjadi
  temuan **C-078**, karena ia yang membuat C-076 bertahan dua sesi.
- Aturan "panjang pilihan tidak menentukan lebar halaman" kini **diuji tanpa bergantung pada data**:
  probe menyisipkan pilihan sengaja panjang ke setiap `select` penyaring.
- Ambang sentuh berlaku sebagai **kotak**; aturannya berlaku atau tidak lagi tergantung panjang kata.
- Tiga aturan baru dipegang mesin sekaligus test klien: kolom penyaring dapat menyusut (`min-w-0`),
  utility `tap-target` menetapkan kedua sisi, dan rentang tenggat sebaris dari ambang `sm:` ke atas.
- Yang **tidak** dijanjikan: halaman **Approvals/Reports/Administration** belum dibangun sehingga belum
  ikut tersapu; saat dibangun, ia wajib masuk daftar halaman di skrip bukti.

## 8. Update Ledger (Checklist Wajib)

- [x] `docs/progress/CHANGELOG.md` — sesi P-050
- [x] `docs/progress/TASKS.md` — `T-066` (`DONE`)
- [x] `docs/progress/STATE.md` — "Diperbarui oleh" P-050, hitungan test frontend, baris frontend §3
- [x] `docs/progress/SESSION-LOG.md` — ringkasan sesi
- [x] `CONTINUE.md` — posisi terakhir, task berikutnya, dan fakta lingkungan kredensial
- [x] Audit — **C-076**, **C-077**, **C-078** (`FIXED`), hitungan **78/76/0/2** di enam berkas
- [x] `70-TESTING.md` §3.14b/§3.14f/§3.14g + `51-UX.md` §2.1
- [x] `TRACEABILITY.md` — **tidak berubah** (tidak ada requirement baru; ini kepatuhan UI dan alat ukur)
- [x] `OPEN-QUESTIONS.md` — **tidak berubah**

## 9. Next Action

- Modul **Approvals** di frontend (`50-FSD.md` §5.4; kontraknya hidup sejak P-048) — dan saat halaman
  itu berdiri, ia **wajib** ditambahkan ke daftar halaman `scripts/responsive-evidence.mjs` supaya
  sapuan per lebarnya tidak terlewat seperti C-078.
- Dua keputusan pemilik yang masih terbuka: **Q-016** (`?view=mine` Documents) dan **Q-024** (endpoint
  daftar pengguna, `C-063`), ditambah **Q-023** (lisensi).
