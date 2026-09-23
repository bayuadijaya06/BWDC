# P-043 — 2026-09-22 — Skill & aturan antislop pada frontend, laci menu di layar sempit, dan pengukur tata letak

**Sesi:** P-043 · **Model/agen:** z-ai/glm-5.3-flash (lanjutan worktree yang sama) · **Status:** selesai
**Task:** `T-056`, `T-057`, `T-058` — semuanya DONE · **Temuan:** C-067, C-068, C-069 (ditemukan **dan** ditutup di sesi ini)

---

## 1. Prompt User

> Terapkan skills dan rules antislop pada frontend yang sudah anda buat. Perbaiki juga tampilan menu pada
> saat resolusi layar lebih kecil. Lanjutkan penulisan kode sesuai dengan progress saat ini. Utamakan
> perbaikan dan pemenuhan gap sebelum melanjutkan progress.

---

## 2. Interpretasi & Scope

Tiga hal, dan urutannya mengikat:

1. **Menerapkan aturan antislop pada frontend** bukan pekerjaan menulis fitur baru, melainkan memeriksa
   apa yang sudah berdiri terhadap aturan yang sudah dipasang di P-042 (`antislop.md` + lima skill, ADR-0025).
   Yang paling relevan: `antislop-layoutmobile` (reflow, overflow, target sentuh, navigasi seluler) dan
   `antislop-ui` (keadaan kosong/memuat/gagal, angka karangan, kontrol mati).
2. **Menu pada resolusi kecil** dilaporkan sebagai tampilan yang belum benar. Saat diperiksa, panelnya hidup
   di dalam baris flex sehingga membuka menu **menyempitkan** halaman alih-alih menutupinya, tanpa latar
   penutup, tanpa kunci gulir, dan tanpa pengembalian fokus.
3. **Gap lebih dulu.** Verifikasi dijalankan sebelum menyentuh apa pun: `make test` (sembilan paket, 236
   test) hijau, frontend **tidak** hijau (satu error `tsc` + dua error `eslint` dari pekerjaan yang belum
   selesai di worktree), dan lima pemeriksa dokumen hijau. Jadi yang diperbaiki lebih dulu adalah error itu,
   lalu pekerjaan yang menggantung, lalu halaman berikutnya.

**Di luar scope:** menambah modul backend baru, menyentuh migrasi, menambah dependency, atau menambah
halaman bisnis berikutnya (Documents) — sesi ini menutup gap tata letak dan aturan, bukan membuka modul.

---

## 3. Rencana

1. Verifikasi menyeluruh lebih dulu (backend, frontend, lima pemeriksa dokumen) dan perbaiki yang merah.
2. Satukan perilaku lapisan modal (laci menu **dan** dialog) di satu hook; perbaiki laci agar menutupi
   konten, bukan mendorongnya.
3. Tetapkan tiga state lebar yang nyata (`< 768`, `768-1023`, `≥ 1024`) dan dua register target sentuh.
4. **Ukur** klaim tata letaknya di peramban sungguhan (jsdom tidak menghitung piksel), lalu buktikan
   pengukurnya bekerja dengan menyuntikkan cacat.
5. Selaraskan dokumen ke perilaku yang benar-benar berlaku (temuan), tambahkan penegakan mesin untuk R-02,
   lalu tutup ledger.

---

## 4. Aksi yang Dilakukan

**Perbaikan yang ditemukan lebih dulu (bukan diabaikan):**

- `frontend/src/components/layout/AppShell.tsx:83` gagal `tsc`: `panelRef` bertipe `RefObject<HTMLElement>`
  dipasang ke `<div>` → diganti `HTMLDivElement`.
- `eslint` menolak dua `setState` di dalam effect (`react-hooks/set-state-in-effect`): pembacaan media query
  disalin ke state, dan reset laci saat melewati titik henti. Yang pertama diganti `useSyncExternalStore`
  (cara resmi membaca sumber luar), yang kedua dipindah ke callback langganan (`useMediaQueryEnter`).
- Peringatan `react-hooks/exhaustive-deps` pada pembersih `useModalLayer` diperbaiki dengan membaca tujuan
  fokus **di dalam** effect.

**Lapisan modal bersama:** `hooks/useModalLayer.ts` baru — fokus awal `[data-autofocus]` → kontrol pertama →
wadah, jebakan Tab dua arah, Escape yang tidak merambat, kunci gulir, dan fokus yang **dikembalikan** ke
elemen pembuka. Dipakai `Dialog` (menggantikan salinan logika di sana) **dan** laci `AppShell`.

**Laci menu:** panel menjadi `fixed` dengan latar penutup ber-penanda `data-drawer-backdrop`, konten di
belakangnya `inert`, tombol berlabel **Menu** (bukan hamburger tanpa keterangan) dengan `aria-expanded` +
`aria-controls`, tombol **Tutup** di kepala panel, dan `useMediaQueryEnter` yang **melupakan** permintaan
membuka saat jendela melewati titik henti (kalau hanya disembunyikan, menu akan muncul sendiri beserta gulir
terkunci begitu jendela dipersempit lagi).

**Tiga state lebar + target sentuh:** `Header` (label tema pendek di layar sempit, nama akun dipotong),
`Sidebar` (`tap-target` pada semua tautan dan sub-tautan), `Button`/`Field`/`DataTable`/`States` memakai
utility `tap-target` yang sama; `51-UX.md` §8/§9 diselaraskan (termasuk alasan penyimpangan state tengah).

**Pengukur tata letak (`scripts/responsive-evidence.mjs`, baru):** menjalankan Chrome yang sudah terpasang
lewat protokol DevTools (`WebSocket` bawaan Node, tanpa dependensi baru), login lewat form yang sama dengan
pengguna, lalu mengukur per lebar: gulir mendatar halaman + elemen penyebabnya, ukuran tiap kontrol terhadap
ambang lebarnya sendiri (44px < 1024px, 36px ≥ 1024px), state sidebar, perilaku laci 375px (termasuk
pelebaran jendela **saat laci masih terbuka**), dan kedua tema. Tidak dijalankan di CI.

**Penegakan R-02:** `check-antislop-refs.sh` butir 8 memindai `frontend/src` tanpa komentar; lima teks
pengguna yang memuat em dash diganti kalimat/tanda yang menerangkan keadaan.

---

## 5. File yang Berubah

### Added

- `frontend/src/hooks/useMediaQuery.ts` — tiga titik henti bernama + pembacaan lewat `useSyncExternalStore`
  + `useMediaQueryEnter` untuk state yang hanya berlaku di layar sempit.
- `frontend/src/hooks/useModalLayer.ts` — perilaku lapisan modal yang dipakai bersama laci dan dialog.
- `frontend/src/components/layout/AppShell.test.tsx` (5 test), `frontend/src/components/common/tapTarget.test.tsx`
  (2 test) — perilaku laci (kunci gulir, `inert`, fokus, Escape, latar, pelebaran jendela) dan pemakaian
  `tap-target` pada kontrol antarmuka.
- `scripts/responsive-evidence.mjs` — pengukur tata letak di peramban sungguhan.

### Changed

- `frontend/src/components/layout/AppShell.tsx` — tiga state lebar, laci modal, `inert`, penanda latar,
  perbaikan tipe ref, pembacaan media query tanpa `setState` di effect.
- `frontend/src/components/layout/{Header,Sidebar}.tsx` — target sentuh, label pendek, nama akun dipotong,
  permintaan fokus untuk laci.
- `frontend/src/components/common/{Dialog,Button,Field,DataTable,States}.tsx` — `Dialog` memakai
  `useModalLayer`; kontrol memakai `tap-target`.
- `frontend/src/{styles/tokens.css,pages/Login/index.tsx,pages/Dashboard/index.tsx,pages/Projects/*}` —
  utility `tap-target`, pemakaian placeholder yang menerangkan keadaan, penyesuaian tata letak sempit.
- `frontend/src/utils/format.ts` — `EMPTY_VALUE`/`EMPTY_DATE` + `formatTimestamp` mengembalikan kalimat.
- `frontend/src/test/a11y.test.tsx` — axe atas laci yang **terbuka**.
- `docs/design/51-UX.md` §8/§9, `docs/design/70-TESTING.md` §3.14b (baru), `docs/design/01-AGENT-WORKFRAME.md`
  §3.2, `AGENTS.md`, `README.md` §4/§12.3, `.freebuff/run.md` (ditulis ulang: frontend + backend + cara
  membersihkan `login_attempts`), `scripts/check-antislop-refs.sh` (butir 8), `scripts/check-readme-facts.sh`
  (semua berkas `scripts/`, bukan hanya `*.sh`).
- Ledger: `docs/progress/{TASKS,OPEN-QUESTIONS,CHANGELOG,SESSION-LOG,STATE}.md`, `CONTINUE.md`,
  `docs/progress/audits/{AUDIT-001-…,README}.md`.

### Removed

- Logika modal yang terduplikasi di `Dialog` (digantikan `useModalLayer`).

---

## 6. Verifikasi (WAJIB)

```
cd backend && make test                     → ok, sembilan paket, 236 test
go vet ./... / gofmt -l .                   → bersih

cd frontend
  npm run typecheck                         → bersih (sebelumnya 1 error)
  npm run lint                              → bersih (sebelumnya 2 error)
  npm run test:run                          → 16 berkas, 137 test lulus
  npm run build                             → dist/ 397 kB js / 21,9 kB css

bash scripts/check-ledger.sh                → ledger OK — 0 peringatan, 236 test di backend
bash scripts/check-doc-links.sh              → BROKEN referensi dokumen: 0
bash scripts/check-readme-facts.sh           → readme-facts OK — 39 fakta diperiksa
bash scripts/check-api-contract.sh           → api-contract OK — 110 pemeriksaan, 55 endpoint
bash scripts/check-antislop-refs.sh          → antislop-refs OK — 8 pemeriksaan

node scripts/responsive-evidence.mjs         → responsive-evidence OK
```

**Bukti tata letak di peramban sungguhan** (halaman `/projects` **dengan satu project nyata**, sehingga
baris tabel ikut terukur):

| Lebar | Gulir mendatar | Kontrol | Di bawah ambang | State sidebar | Tema diukur |
|---|---|---|---|---|---|
| 375px | 0px | 10 | 0 | laci (menu tampil, sidebar tersembunyi) | terang + gelap |
| 768px | 0px | 38 | 0 | kolom kompak 192px | — |
| 1024px | 0px | 38 | 0 | kolom penuh 240px | — |
| 1440px | 0px | 38 | 0 | kolom penuh 240px | terang + gelap |

Laci 375px: `opened`, lebar panel 288px (lebih sempit dari viewport), menutupi seluruh tinggi, latar penutup
ada, gulir terkunci, konten `inert`, fokus di dalam panel, konten utama tetap > 90% viewport. Sesudah Escape:
tertutup, latar hilang, gulir bebas, `inert` dibersihkan, fokus kembali ke tombol Menu. Melebarkan jendela
saat laci terbuka lalu mempersempitnya lagi: laci tetap tertutup dan gulir tetap bebas.

**Gigi pengukur & test dibuktikan** dengan menyuntikkan cacat sementara, lalu memulihkannya — hasilnya
sekaligus menemukan **C-068**:

| Cacat | Hasil |
|---|---|
| `overflow-x-auto` dicabut dari `DataTable` | `FAIL`: gulir mendatar 240px @375px, 39px @768px + elemen penyebab |
| `tap-target` dicabut dari tautan sidebar | `FAIL`: 8 kontrol 175x19px @768px |
| `fixed` dicabut dari panel laci | `FAIL`: `contentNotSqueezed = false` |
| `inert` dicabut | `FAIL`: `contentInert = false` |
| `useMediaQueryEnter` dicabut | `FAIL`: `drawerStaysClosed = false`, `scrollStaysFree = false` |
| kunci gulir dicabut dari `useModalLayer` | `FAIL`: `scrollLocked = false` |
| em dash ditambahkan ke string UI | `FAIL` (butir 8 antislop); di komentar → lolos; sesudah `https://` → `FAIL` |
| nama berkas `.mjs` dihapus dari pohon README | `FAIL` (2 temuan `check-readme-facts.sh`) |

Dua cacat **lolos** pada percobaan pertama dan itu tercatat sebagai temuan, bukan disenyapkan: test
"menutup laci saat jendela dilebarkan" lulus tanpa `useMediaQueryEnter` (yang diuji hanya penyembunyian,
bukan melupakan permintaan) dan latar penutup diuji lewat posisi DOM sehingga `aria-hidden` yang diubah tetap
lulus; dua pemeriksaan di skrip pengukur juga lulus hampa karena mengukur tetangga panel sebagai "konten" dan
menutup laci sebelum melebarkan jendela. Ketiganya diperbaiki lebih dulu, lalu seluruh cacat disuntikkan ulang.

**Database dikembalikan persis ke baseline** sesudah bukti: `audit_logs` 43, `login_attempts` 0, `projects` 0,
`users` 1, versi goose 10; `bwdcs_test` kosong.

---

## 7. Hasil & Dampak

- **Tiga temuan lama kelas baru muncul dan langsung ditutup** (C-067, C-068, C-069). Yang paling penting
  pelajarannya: **klaim tata letak tidak dapat diperiksa oleh test tipe apa pun yang dimiliki repo ini**, dan
  selama tidak ada yang mengukurnya, "44px", "tanpa gulir mendatar", dan "laci menutupi konten" hanyalah
  kalimat yang terasa benar. Sekarang ketiganya angka.
- **Laci menu benar-benar menjadi lapisan modal**, dan perilakunya diuji di dua tingkat: jsdom (perilaku,
  fokus, kunci gulir) dan peramban sungguhan (geometri, ungkapan media, kedua tema).
- **Satu hook untuk dua lapisan** (`Dialog` + laci) menghapus duplikasi yang justru bagian paling mudah
  terlupakan (fokus kembali, jebakan Tab, kunci gulir).
- **R-02 kini ditegakkan mesin** pada teks yang dibaca pengguna, dan keputusan proyeknya dipersempit agar
  tidak menjanjikan lebih daripada yang diperiksa (**Q-025**). Ketegangan ambang 44px/36px dicatat **Q-026**.
- **Tidak ada halaman bisnis baru** dibuka: sesi ini menutup gap, dan halaman Documents tetap pekerjaan
  berikutnya dengan pola yang kini juga terukur tata letaknya.

### Delivery Gate (laporan wajib, `01-AGENT-WORKFRAME.md` §5.3)

Butir Gate-nya dibaca dari `antislop.md`; laporan bentuk ini yang ditetapkan dokumen proyek.

**Blok 1 — Hard Gate** (semua jawaban harus "tidak"):

- R-02 PASS: teks yang dibaca pengguna bebas em dash, **diperiksa mesin** (`check-antislop-refs.sh` butir 8):
  tiga placeholder `"—"` diganti `EMPTY_VALUE`/`EMPTY_DATE`, satu kalimat dialog dan satu pemisah role
  anggota ditulis ulang. Prosa/komentar lama tidak disapu dan itu dinyatakan (Q-025).
- R-03 PASS: 0px gulir mendatar pada 375/768/1024/1440px **dengan satu baris tabel nyata**; tabel 615px di
  viewport 375px bergulir **di dalam wadahnya**; target sentuh 0 kontrol di bawah ambang (10/38/38/38
  diperiksa); laci 375px menutupi (lebar 288px) alih-alih mendorong (konten tetap > 90% viewport).
- R-17/R-18 PASS: Dashboard tetap **tanpa angka** apa pun dan tidak ada testimoni/avatar karangan; halaman
  `/projects` menampilkan apa yang dikirim server (0 project = keadaan kosong, bukan contoh karangan).
- R-23 PASS: tidak ada aset yang dikarang (wordmark teks, inisial dari `username` nyata); batas yang tidak
  dapat dipenuhi (pemilih `Owner`, C-063/Q-024) ditampilkan di layar.
- R-24 PASS: setiap item menu punya rute; halaman yang belum dibangun menampilkan `ModulePending` dan
  menandai dirinya `belum` (tidak ada tautan mati).
- R-25 PASS: 31 test kontras WCAG 2.2 AA per pasangan token di kedua tema tetap hijau; tidak ada nilai warna
  yang ditulis di berkas komponen.
- R-26 PASS: setiap kontrol punya perilaku nyata (Menu membuka laci, Escape/Tutup/latar menutupnya, tema
  bersiklus, Arsipkan mengarsipkan, Buat project mengirim `POST /projects`).
- R-27 PASS: halaman data punya keadaan memuat, kosong, dan gagal yang dibedakan test; nilai kosong kini
  dijawab kalimat ("Belum diisi"/"Belum ditetapkan"/"Belum dicatat"), bukan tanda pisah.
- R-32 PASS: `useModalLayer` memindahkan fokus ke dalam laci, menjebak Tab, Escape menutup, dan fokus
  **kembali** ke tombol Menu (diukur di jsdom **dan** di peramban).
- R-34 PASS: kedua tema diukur di 375px dan 1440px (enam keadaan) — nol gulir mendatar di semuanya.
- R-35 PASS: halaman dibuka di dev server dan diukur mesin; enam cacat disuntikkan untuk membuktikan
  pengukurnya bekerja.
- R-13/R-14/R-19/R-22/R-28/R-33/R-36/R-37/R-38 PASS (tidak berlaku/tidak ada): tanpa glow, tanpa animasi
  berulang (MOTION 1), tanpa ilustrasi, tanpa FAQ, tanpa tambalan skrip luar, tanpa klaim keamanan atau
  kinerja, dan arah desain sudah ada (bukan draft tanpa arah).
- R-01/R-04/R-06/R-07/R-08/R-09/R-10/R-12: tidak berubah dari P-037/P-041 — tidak ada gradient dekoratif,
  ikon generik, font unduhan, grid latar, panah dekoratif pada tombol, badge dekoratif, glassmorphism, atau
  bayangan yang tersebar; state tengah sidebar **berlabel** justru karena R-04 melarang rail ikon generik.

**Blok 2 — Purpose-Gate:** tidak ada teknik baru yang dipakai sesi ini (tidak ada gradient, glow, glass,
ikon, atau animasi yang ditambahkan), sehingga tidak ada alasan baru yang perlu ditulis. Satu pengecualian
disengaja: **penanda `data-drawer-backdrop`** pada latar laci, yang ada supaya test jsdom dan pengukur tata
letak mencari elemen yang sama lewat namanya, bukan lewat posisinya (alasannya tertulis di komponen).

**Blok 3 — Liveliness** (semua "ya"): dial tetap **ENERGY 1 / RHYTHM 2 / MOTION 1** (`DESIGN.md`); keluaran
sesuai dial (tidak ada animasi, hanya keadaan hover/aktif); satu titik fokus per layar (judul halaman +
satu aksi utama); whitespace struktural (`gap-5`, `px-4 py-4` di sempit, `lg:px-6` di lebar); satu accent
(Signal) yang tetap muncul paling banyak sekali per layar; motif identitas (punggung rekam 3px + label
kanonik) dipakai ulang di tabel; Design Read halaman ditulis di log ini.

**Blok 4 — Craftsmanship & Quality Locks:** C-1..C-5 PASS (tidak ada data atau janji yang dikarang; angka
hanya dari server; keadaan tiga wujud ada; ukuran/radius/bayangan dari token); R-05/R-11/R-15/R-16/R-20/R-21/
R-29/R-30/R-31 PASS — satu radius (`rounded-control`/`rounded-panel`), tanpa teks buzzword, kontras badge
status diuji, hierarki mengikuti keputusan pengguna di tiap layar, dan alasan visual tetap tertulis di
`DESIGN.md` §2-§4 (sesi ini menambah alasan satu baris di `51-UX.md` §8/§9 untuk keputusan yang sebelumnya
tidak tertulis).

**Tidak ada butir `FAIL`** pada laporan di atas.

---

## 8. Update Ledger (Checklist Wajib)

- [x] `docs/progress/prompts/P-043-2026-09-22-antislop-frontend-dan-laci-menu.md` (berkas ini)
- [x] `CHANGELOG.md` — entri P-043
- [x] `STATE.md` — ringkasan sesi, `Titik masuk resume`, `Next action`, dan tabel §3 (hitungan frontend)
- [x] `TASKS.md` — `T-056`, `T-057`, `T-058` masuk `DONE` beserta buktinya
- [x] `OPEN-QUESTIONS.md` — **Q-025** (cakupan sapuan em dash) dan **Q-026** (36px desktop vs 44px)
- [x] `docs/progress/audits/` — **C-067**, **C-068**, **C-069** (`FIXED`); marker `audit-summary` dan prosa
      lima dokumen diselaraskan ke **69 / 67 / 0 / 2**
- [x] `CONTINUE.md` §0 dan `AGENTS.md` (perintah verifikasi + keadaan frontend)
- [x] `docs/design/70-TESTING.md` §3.14b + `51-UX.md` §8/§9 + `01-AGENT-WORKFRAME.md` §3.2 + `README.md`
      §4/§12.3 + `.freebuff/run.md`
- [x] `SESSION-LOG.md`

---

## 9. Next Action

1. **Halaman Documents** (`50-FSD.md` §4, `42-API.md` §4 — endpoint dan cakupannya sudah hidup) memakai pola
   P-041 yang kini **terukur tata letaknya**; sisa halaman tingkat dua: Tasks, lalu Approvals (sebagian
   menunggu `GET /workflows/instances`).
2. **Jalur backend** tetap sah dan belum tersentuh: modul **Workflow** (`43-WORKFLOW.md`, Phase 2,
   ADR-0015/0016) lalu admin/notification.
3. **Menunggu pemilik:** Q-019/C-050 (threading), Q-023 (lisensi), Q-024/C-063 (endpoint daftar pengguna),
   dan dua pertanyaan baru sesi ini — **Q-025** (sapuan em dash) dan **Q-026** (ambang sentuh desktop).
