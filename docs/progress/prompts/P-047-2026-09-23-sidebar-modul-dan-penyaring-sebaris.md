# P-047 — 2026-09-23 — Sidebar kembali ke modul saja, penyaring Task kembali sebaris

| Field | Isi |
|---|---|
| ID | P-047 |
| Waktu mulai | 2026-09-23 (lanjutan sesi P-046 pada worktree yang sama) |
| Aktor | agen (Buffy, perkakas berkas + terminal + Preview) |
| Model / agen | deepseek/deepseek-v4-flash |
| Fase roadmap | 4 (UI) |
| Task terkait | `T-063` (baru, DONE di sesi ini) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Pada modul Tasks, perbaiki filter Penanggung Jawab, tingginya beda dengan filter lain karena ada text di bawah selection. Optimize juga menu sidebar, terlalu banyak item padahal hanya filter atau pindah tab pada halaman sesungguhnya. Lanjutkan ke progress berikutnya."

## 2. Interpretasi & Scope

- **Yang diminta:**
  1. Penyaring **Penanggung jawab** di halaman Tasks disamakan tingginya dengan penyaring lain.
  2. **Sidebar** dirampingkan: item yang isinya hanya tab atau penyaring halaman tidak berdiri sebagai menu.
  3. Melanjutkan progress berikutnya setelah keduanya beres.
- **Yang TIDAK termasuk (out of scope):** menambah halaman baru (Approvals/Reports/Administration tetap `ModulePending`), menyentuh backend (tidak ada perubahan kode server pada sesi ini), dan menghapus kemampuan penyaring demi memendekkan menu.
- **Asumsi yang diambil:**
  - Keluhan (2) dibaca sebagai **redundansi**, bukan larangan punya tab: yang salah adalah sub-item sidebar, bukan sub-navigasinya. Karena itu sub-nav dipindah ke halaman, tidak dimatikan.
  - "Text di bawah selection" pada (1) adalah catatan sumber pilihan penanggung jawab (`<p>` di dalam kolom), bukan pesan galat.
  - Konsekuensi urutan: memindahkan sub-nav Documents mengharuskan baris tab dibuat lebih dulu, kalau tidak penyaring `?view=mine` kehilangan satu-satunya jalan masuk dari antarmuka dan penjelasan Q-016 ikut tenggelam.
- **Pertanyaan yang muncul:** tidak ada yang baru. Penyaring `Milik saya` pada Documents tetap belum dapat dijalankan karena kontraknya belum memuat parameter pemilik (**Q-016**, temuan **C-063**) — sesi ini **tidak** mengubah status itu, hanya memastikan alasannya tetap terbaca dari tab, bukan hanya dari URL yang harus dihafal.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Baca ulang disk sebelum mengubah apa pun (sesi sebelumnya terputus setelah menghentikan turn) | Tidak ada pekerjaan yang diulang; kondisi nyata diketahui dulu |
| 2 | Pindahkan catatan penanggung jawab keluar dari kolomnya | Seluruh kontrol baris penyaring berbagi satu tepi bawah |
| 3 | Buang `subItems` dari `config/navigation.ts`; tambah `requiresExactMatch` | Model navigasi kembali ke satu entri per modul |
| 4 | Sederhanakan `Sidebar.tsx` | Satu tautan per modul, tanpa sub-menu |
| 5 | Beri baris tab pada halaman **Documents** | Sub-halaman tetap dapat dibuka, termasuk `Milik saya` |
| 6 | Perbarui test yang mengunci ketiga perilaku itu | Perubahan tidak dapat dibatalkan diam-diam |
| 7 | Selaraskan `51-UX.md` §2.1 dengan keputusan | Dokumen desain bukan lagi menyebut kolom `Sub-items` |
| 8 | Perluas `scripts/responsive-evidence.mjs` dengan bagian `navigation` | Ketiga klaim diukur di peramban sungguhan, bukan diklaim |
| 9 | Ledger + lima pemeriksa | Sesi tertutup rapi |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Membaca ulang `pages/Tasks/index.tsx`, `config/navigation.ts`, `components/layout/Sidebar.tsx`, `pages/Documents/index.tsx`, `pages/ModulePending.tsx`, `AppShell.test.tsx`, `navigation.test.ts`, dan `scripts/responsive-evidence.mjs` | Turn sebelumnya dihentikan sebelum bertindak; berkas harus dilihat apa adanya | Dua sebab terdentifikasi: kolom lebih tinggi (bukan warna/ukuran), dan model navigasi yang mendaftarkan `subItems` |
| 2 | `Tasks/index.tsx`: `<p>` catatan dipindah dari kolom Penanggung jawab ke blok catatan di bawah baris penyaring | Menghapus penyebab kolom lebih tinggi tanpa menghapus informasinya | Ketujuh kontrol pertama berbagi `bottom 256`; tidak ada yang terangkat |
| 3 | `navigation.ts`: `subItems`/`NavSubItem` dibuang; `requiresExactMatch(item)` ditambahkan | Delapan menu menghasilkan lima belas tautan; aturan pencocokan awalan membuat dua menu sama-sama aktif untuk `/reports` dan `/reports/audit` | Menu = modul; `end` diturunkan dari daftar path, bukan ditulis tangan |
| 4 | `Sidebar.tsx`: `sameFilters`, `useLocation`, `Link`, dan blok `<ul>` dihapus; `end={requiresExactMatch(item)}` | Satu tautan per modul; tidak ada pencocokan kueri lagi di sidebar | Sidebar 8 tautan, 0 di antaranya berkueri |
| 5 | `Documents/index.tsx`: model `subPages` + baris tab (`Semua`, `Milik saya`, `Pending Review`, `Revision Required`, `Approved`) | Sub-navigasi harus tinggal di halaman yang memiliki daftarnya, dan `Milik saya` harus tetap dapat dibuka | Tab memetakan label → status kanonik `42-API.md` §4; alasan Q-016 tampil saat tabnya dibuka |
| 6 | Test diperbarui/ditambah: `AppShell.test.tsx` (describe `menu sidebar`), `navigation.test.ts` (describe `bentuk menu`), `Documents.test.tsx` (2 test tab), `Tasks.test.tsx` (teks catatan yang berpindah) | Mengunci perilaku baru, bukan menyisakan test yang menuntut sub-item | 240 test hijau |
| 7 | `51-UX.md` §2.1 ditulis ulang | Dokumen desain harus menyatakan keputusan ini, bukan versi lama | Kolom `Sub-items` → kolom halaman + sub-navigasinya, dengan alasan P-047 |
| 8 | `scripts/responsive-evidence.mjs` diperluas: bagian `navigation` (sidebar, baris penyaring, tab Documents) | Tiga klaim ini tidak dapat diperiksa jsdom (jsdom tidak menghitung tata letak) | Skrip mengukur dan **menggagalkan** sesi bila klaimnya tidak terbukti |
| 9 | Gigi dibuktikan dua kali: cacat (1) dipasang kembali, dan aturan awalan lama dipasang kembali | Test/skrip yang selalu hijau tidak membuktikan apa pun | Skrip **FAIL** pada `kolom "Penanggung jawab" melanjutkan 23px di bawah kontrolnya`; test AppShell **gagal** pada `Reports` yang ikut aktif |
| 10 | Dokumen desain lain diselaraskan + ledger P-047 | Kewajiban protokol | `42-API.md`/`44-SECURITY.md` tidak berubah (tidak ada kontrak yang bergerak); ledger diperbarui |

## 5. File yang Berubah

### Added

| File | Ringkasan perubahan | Requirement terkait |
|---|---|---|
| `docs/progress/prompts/P-047-2026-09-23-sidebar-modul-dan-penyaring-sebaris.md` | Log sesi ini | `02-AGENT-PROGRESS-PROTOCOL.md` |

### Changed

| File | Ringkasan perubahan | Requirement terkait |
|---|---|---|
| `frontend/src/pages/Tasks/index.tsx` | Catatan sumber pilihan penanggung jawab pindah dari kolom penyaring ke blok catatan di bawah baris | R-03, R-36, `51-UX.md` §9 |
| `frontend/src/config/navigation.ts` | `subItems` dibuang; `requiresExactMatch` diturunkan dari daftar path; komentar model navigasi ditulis ulang | R-14, `51-UX.md` §2.1 |
| `frontend/src/components/layout/Sidebar.tsx` | Satu tautan per modul; `end` dihitung; pencocokan kueri dihapus | R-14, R-24, R-03 |
| `frontend/src/pages/Documents/index.tsx` | Baris tab sub-halaman + komentar sumbernya | `50-FSD.md` §4.1, `42-API.md` §4 |
| `frontend/src/pages/Projects/index.tsx` | Komentar: Projects tidak butuh baris tab (List = halaman, Create = tombol) | `51-UX.md` §2.1 |
| `frontend/src/components/layout/AppShell.test.tsx` | Describe `penanda menu aktif` diganti `menu sidebar` (bentuk menu + tepat satu menu aktif + path bersarang) | R-14, R-32 |
| `frontend/src/config/navigation.test.ts` | Describe `bentuk menu`: tanpa `subItems`, aturan pencocokan persis, tidak ada dua menu saling menutupi | R-14 |
| `frontend/src/pages/Documents/Documents.test.tsx` | Dua test baru: tab memetakan ke status kanonik; `Milik saya` dapat dibuka dari tab dan alasannya terbaca | R-24, R-36, Q-016 |
| `frontend/src/pages/Tasks/Tasks.test.tsx` | Teks catatan penyaring yang berpindah tempat | R-36 |
| `scripts/responsive-evidence.mjs` | Bagian `navigation`: sidebar (tautan berkueri/sub-halaman/menu aktif), baris penyaring Task (kolom melanjutkan di bawah kontrolnya, kesebarisan per garis), tab Documents (dibuka lewat klik sungguhan) | R-35, R-03, `70-TESTING.md` §3.14b |
| `docs/design/51-UX.md` | §2.1 ditulis ulang: sidebar modul saja, sub-navigasi di halaman, catatan alasan P-047 | `51-UX.md` §2.1 |
| `docs/design/70-TESTING.md` | §3.14b diperluas dengan bagian `navigation` skrip bukti | `70-TESTING.md` |
| `docs/progress/TASKS.md` | Baris `T-063` (DONE) | `02-AGENT-PROGRESS-PROTOCOL.md` |
| `docs/progress/STATE.md`, `CONTINUE.md`, `SESSION-LOG.md`, `CHANGELOG.md`, `AGENTS.md`, `README.md` | Ledger sesi P-047 dan angka frontend (240 test / 24 berkas) | `02-AGENT-PROGRESS-PROTOCOL.md` |
| `.freebuff/run.md` | Baseline `versi skema` dikoreksi ke 11; bagian skrip bukti menyebut bagian `navigation` | `60-DEPLOYMENT.md`, `70-TESTING.md` §3.14b |

> Daftar ini disalin ke `CHANGELOG.md`.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `npm run typecheck` | bersih (`tsc --noEmit -p tsconfig.json`) | PASS |
| 2 | `npm run lint` | bersih (`eslint .`) | PASS |
| 3 | `npm run test:run` | **240 test / 24 berkas** hijau | PASS |
| 4 | `npm run build` | 173 modul; 455,3 kB js / 23,3 kB css (gzip 136,2 / 5,7) | PASS |
| 5 | Gigi test menu: `end={item.path === "/"}` dipasang kembali | `AppShell.test.tsx` **gagal** pada `Reports` yang ikut `aria-current="page"` | PASS (gigi terbukti) |
| 6 | Gigi ukuran: `<p>` catatan dipasang kembali di dalam kolom | `responsive-evidence` **FAIL**: `kolom "Penanggung jawab" melanjutkan 23px di bawah kontrolnya` | PASS (gigi terbukti) |
| 7 | `node scripts/responsive-evidence.mjs` | OK: lebar 375/768/1024/1440px, ambang 44/36px, 6 pengukuran tema, laci 375px membereskan dirinya; sidebar 8 modul / 0 tautan berkueri / 0 sub-tautan / 1 menu aktif; baris penyaring 2 garis dengan 0 kontrol terangkat dan tinggi seragam 36px; tab Documents 5 tautan → klik `Milik saya` memindahkan penanda + alasan Q-016 terbaca; gulir mendatar 0 di keempat lebar | PASS |
| 8 | Lima pemeriksa (`check-ledger`, `check-doc-links`, `check-readme-facts`, `check-api-contract`, `check-antislop-refs`) | lihat ringkasan di `70-TESTING.md` §3.14e | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [x] Jika UI: Delivery Gate antislop dijalankan (`01-AGENT-WORKFRAME.md` §5.3)

## 7. Hasil & Dampak

- **Selesai:** kedua keluhan tertutup dengan sebab yang disebut, bukan gejala yang ditambal. Sidebar kembali menjadi daftar modul (8 tautan) dan sub-navigasi tinggal di halaman yang memilikinya; baris penyaring Task kembali sebaris; `51-UX.md` §2.1 menyatakan keputusan barunya; ketiga klaim diukur mesin di peramban sungguhan dan **dapat menggagalkan sesi** bila rusak lagi.
- **Belum selesai / sisa:** tidak ada dalam lingkup sesi ini. Halaman Approvals/Reports/Administration tetap `ModulePending` (Approvals menunggu modul Workflow, Phase 2).
- **Risiko / utang teknis:**
  - Baris penyaring Task kini melipat menjadi **dua garis** pada 1440px (delapan kontrol). Melipat adalah perilaku `flex-wrap` yang disengaja, tetapi urutan pada garis kedua (`Tenggat sampai`, `Terapkan rentang`) menempatkan satu batas rentang terpisah dari batas pertamanya. Bila keluhan serupa muncul lagi, yang perlu ditata adalah **pengelompokan rentang** (satu kolom berisi dua isian), bukan tingginya.
  - Penyaring `Milik saya` pada Documents masih belum dapat dijalankan (**Q-016**); tabnya kini justru membuat batas itu lebih mudah ditemukan pembaca, bukan lebih tersembunyi.
- **Dampak ke dokumen desain:** `51-UX.md` §2.1 (ditulis ulang) dan `70-TESTING.md` §3.14b (cakupan skrip bukti). Kontrak API tidak berubah sama sekali: tidak ada endpoint, parameter, atau izin yang bergerak pada sesi ini.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-063` ditambahkan sebagai DONE)
- [x] `TRACEABILITY.md` diperbarui (baris `FR-TASK`/`FR-DOC` menyebut baris tab dan aturan menu)
- [x] `OPEN-QUESTIONS.md` diperbarui (Q-016 tidak berubah status; catatan bahwa tab Documents kini menjadi jalan masuknya)
- [ ] ADR dibuat/diperbarui — **tidak perlu**: tidak ada keputusan arsitektur baru, hanya penempatan kendali yang sudah diputuskan arah desainnya di `51-UX.md` §2.1

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Modul **Workflow** di backend (`42-API.md` §5, `43-WORKFLOW.md`, ADR-0015/0016) — halaman Approvals menunggu `GET /workflows/instances`-nya | agen |
| 2 | Halaman **Approvals** di frontend setelah endpointnya ada | agen |
| 3 | Bila diinginkan: tata ulang pengelompokan penyaring rentang tenggat Task menjadi satu kolom dua isian | pemilik proyek |
