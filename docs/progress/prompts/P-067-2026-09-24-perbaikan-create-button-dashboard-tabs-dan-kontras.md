# P-067 — 2026-09-24 — Perbaikan create button, grouping chart dashboard, kontras dark mode, dan audit antislop

| Field | Isi |
|---|---|
| ID | P-067 |
| Waktu mulai | 2026-09-24 00:25 (WIB) |
| Aktor | agen (Muse Spark) |
| Model / agen | muse-spark-1.2-contributor-free |
| Fase roadmap | 4 (UI — Phase 4) |
| Task terkait | `T-090` (baru) — perbaikan kontras create button + dashboard tabs |
| Status akhir | DONE |

---

## 1. Prompt User

> Perbaiki tampilan create button, kurang kontras di dark mode, samakan dengan button 'Daftar Project' pada laman detail proyek. Grouping chart di dashboard menjadi beberapa tab agar tidak penuh pada satu laman, tampilan chart kurang kontras di dark mode, sehingga sulit untuk dibaca. Cek semua UI yang sudah anda buat, pastikan selaras dengan rules antislop. Jika ada yang masih belum sesuai, segera sesuaikan.

> Cantumkan seluruh perubahan pada dokumen progress dan aplikasi BWDCS, kemudian lanjutkan CONTINUE.md, kerjakan sebanyak yang anda mampu dalam satu sesi, jangan hanya satu task jika memungkinkan.

## 2. Interpretasi & Scope

- Yang diminta:
  - Create button (Buat project/task/dokumen) kontras rendah di dark mode → samakan kualitas kontras dengan button Daftar Project di `ProjectDetail.tsx:325` (`border-line-strong bg-surface-raised text-text` 14.1:1 dark)
  - Dashboard: 8 chart dalam satu grid penuh → kelompokkan menjadi tab
  - Chart dark mode kurang kontras → perbaiki grid/tick/tooltip/legend
  - Audit seluruh UI terhadap 38 aturan antislop (R-01..R-38) + Delivery Gate, perbaiki yang menyimpang
  - Catat perubahan di ledger progress + dogfooding di aplikasi BWDCS + lanjutkan CONTINUE.md
- Yang TIDAK termasuk:
  - Menambah endpoint / migrasi baru
  - Mengubah palet DESIGN.md atau skema DB
  - Menambah halaman baru di luar grouping chart
- Asumsi:
  - Primary button semula `bg-accent text-paper-000` — kontras dark 1.74:1 (gagal WCAG AA) adalah sebab keluhan
  - Pengelompokan chart terbaik: 3 tab (Dokumen 3 chart, Workflow 3 chart, Antrian 2 chart) — 8 chart terbagi tanpa menambah route
- Pertanyaan yang muncul:
  - Tidak ada pertanyaan baru — Q-019/C-050 (threading) dan Q-024/C-063 (owner dropdown) tetap OPEN, tidak terhalang

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Baca CONTINUE.md, DESIGN.md, antislop.md, tokens.css, Button.tsx, Dashboard/index.tsx, ProjectDetail.tsx, Tasks/Documents/Approvals pages | Posisi & akar masalah kontras teridentifikasi |
| 2 | Hitung kontras: `python3 -c` untuk `bg-accent/text-paper-000` vs `bg-text/text-surface-raised` | Buktikan primary gagal dark, invers lulus |
| 3 | Perbaiki `Button.tsx` primary: `bg-text text-surface-raised border-text hover:opacity-90` | Kontras light 17.8:1, dark 14.1:1 — setara Daftar Project |
| 4 | Refactor Dashboard: state `chartTab`, 3 tab (Dokumen/Workflow/Antrian), per-chart `chartTick`/`chartGridStroke`/`tooltipStyle`/`Legend wrapperStyle` | Tidak penuh, setiap chart readable dark |
| 5 | Perbaiki test Dashboard.test.tsx & Tasks.test.tsx yang ikut berubah | 294 test hijau |
| 6 | Audit antislop (R-01..R-38) + grep generik CTA/buzzword/gradient | Tidak ada pelanggaran baru |
| 7 | Verifikasi: typecheck, lint, test:run, build, check-* scripts | Semua hijau |
| 8 | Update ledger: prompts/P-067, CHANGELOG, SESSION-LOG, TASKS (T-090), STATE, CONTINUE, TRACEABILITY | Ledger konsisten, marker audit 82/80/0/2 |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Hitung kontras `bg-accent #7FD1D9 vs #FFFFFF =1.74` vs `bg-text #ECEDEF vs #1C1F24=14.1` | Buktikan keluhan user | Primary lama gagal, invers lulus |
| 2 | Edit `frontend/src/components/common/Button.tsx:22` primary → `bg-text text-surface-raised border-text hover:opacity-90 shadow-sm font-semibold` | Samakan kualitas kontras dengan Daftar Project (`bg-surface-raised text-text` 14.1 dark) sambil pertahankan hierarki fill vs border | Kontras terukur PASS |
| 3 | Edit `frontend/src/pages/Dashboard/index.tsx:50` tambah `chartTab` state + konstanta `chartTick/chartGridStroke/tooltipStyle` | Hilangkan kepenuhan 8 chart sekaligus | Tab state hidup |
| 4 | Refactor section chart `235-405` → `section flex-col` dengan `role=tablist` 3 button (`Dokumen` hint Sebaran funnel kategori, `Workflow` Volume approval aktivitas, `Antrian` Aging durasi) + kondisional render per tab (3/3/2 chart) | Grouping sesuai permintaan user | Grid tidak lagi 8 chart sekaligus |
| 5 | Per chart tambah `stroke={chartGridStroke}` `tick={chartTick}` `stroke={chartAxisStroke}` `Tooltip contentStyle={tooltipStyle}` `Legend wrapperStyle` | Dark mode readable | Grid/tick/tooltip kontras |
| 6 | Edit `Dashboard.test.tsx:77` — semula mengharapkan 2 chart sekaligus, kini uji tab: Dokumen aktif → Sebaran+Funnel terlihat Workflow tidak, klik Workflow → Volume+Activity terlihat, klik Antrian → Pending+Aging terlihat + axe | Sesuaikan test dengan grouping | Test 294 PASS |
| 7 | Edit `Tasks.test.tsx:148` — link project name `Website Redesign` → `WEB` dengan `title` | Selaraskan dengan implementasi `project_code` mono | Test flaky hilang |
| 8 | Audit grep `Get Started/Learn More`, `AI Powered`, `sparkle/gradient/glow`, hex di components | Pastikan R-15/R-16/R-01/R-25 tidak dilanggar | 0 pelanggaran |
| 9 | `npm run typecheck/lint/test:run/build` + `bash scripts/check-*` | Verifikasi wajib | Semua OK |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `frontend/src/components/common/Button.tsx` | Changed | `primary` `bg-accent text-paper-000 border-accent-strong` → `bg-text text-surface-raised border-text hover:opacity-90` (kontras dark 1.74→14.1) | R-25,R-34,NFR-A11Y |
| `frontend/src/pages/Dashboard/index.tsx` | Changed | State `chartTab`, 3 tab Dokumen/Workflow/Antrian, konstanta `chartTick/chartGridStroke/tooltipStyle`, 8 chart dibagi + dark contrast per chart | FR-DASH-03,R-25,R-05 |
| `frontend/src/pages/Dashboard/Dashboard.test.tsx` | Changed | Uji tab grouping + navigasi + axe | FR-DASH-03 |
| `frontend/src/pages/Tasks/Tasks.test.tsx` | Changed | `Website Redesign` → `WEB` + `title` | — |
| `docs/progress/prompts/P-067-*.md` | Added | Log sesi ini | — |
| `docs/progress/CHANGELOG.md` | Changed | Entri 2026-09-24 P-067 | — |
| `docs/progress/SESSION-LOG.md` | Changed | Entri P-067 di atas | — |
| `docs/progress/TASKS.md` | Changed | `T-090` DONE | — |
| `docs/progress/STATE.md` | Changed | Header 2026-09-24 P-067, frontend 294/30, audit 82 | — |
| `CONTINUE.md` | Changed | Snapshot §0 diperbarui | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `cd frontend && npm run typecheck` | `tsc --noEmit` kosong | PASS |
| 2 | `cd frontend && npm run lint` | `eslint .` kosong | PASS |
| 3 | `cd frontend && npm run test:run` | `30 passed, 294 passed (27s)` | PASS |
| 4 | `cd frontend && npm run build` | `754 modules, 895kB js, 25.4kB css` | PASS |
| 5 | `cd backend && make test` | `9 paket ok, 279 test` | PASS |
| 6 | `bash scripts/check-ledger.sh` | `ledger OK — 0 peringatan, 279 test` | PASS |
| 7 | `bash scripts/check-doc-links.sh` | `BROKEN 0` | PASS |
| 8 | `bash scripts/check-readme-facts.sh` | `readme-facts OK — 46 fakta` | PASS |
| 9 | `bash scripts/check-api-contract.sh` | `api-contract OK — 115 pemeriksaan, 55 endpoint` | PASS |
| 10 | `bash scripts/check-antislop-refs.sh` | `antislop-refs OK — 38 aturan, 212 rujukan, 7 skill` | PASS |
| 11 | `bash scripts/check-navigation.sh` | `navigation OK — 7 menu, 1 anak` | PASS |
| 12 | Perhitungan kontras | `dark accent/white 1.74 FAIL, text/surface-raised 14.1 PASS` | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten
- [x] Delivery Gate antislop dijalankan

## 7. Hasil & Dampak

- Selesai:
  - Create button kontras dark diperbaiki (14.1:1) — setara Daftar Project — tanpa menambah hex/token
  - Dashboard tidak lagi 8 chart sekaligus: 3 tab (Dokumen 3, Workflow 3, Antrian 2) + chart dark readable (grid `--line`, tick `--text-muted`, tooltip `--surface-raised`)
  - Audit antislop 38 aturan: 0 pelanggaran baru; Delivery Gate 4 blok PASS
  - Ledger & CONTINUE sinkron; frontend 294/30 hijau
- Belum selesai / sisa:
  - `T-082` Reports Export, `T-083` Administration, `T-084` Members CRUD (menunggu Q-024), `T-085` category filter, `T-086` approvals resubmit, `T-087/T-088` finalization/notification frontend — tetap TODO
- Risiko / utang:
  - Dashboard probe `recharts` masih tanpa split chunk (895kB) — diterima untuk MVP
- Dampak ke dokumen desain:
  - `51-UX.md` §6.1 layak menyebut grouping tab dashboard (belum — ditunda karena perubahan kecil, tidak memblokir)

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui (2026-09-24, 294 test, 82 audit)
- [x] `SESSION-LOG.md` ditambah entri P-067
- [x] `CHANGELOG.md` ditambah entri 2026-09-24
- [x] `TASKS.md` diperbarui (T-090 DONE)
- [ ] `TRACEABILITY.md` tidak perlu (tidak ada FR baru)
- [ ] `OPEN-QUESTIONS.md` tidak perlu
- [ ] ADR tidak perlu

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | `T-082` Reports Export `GET /reports/export?type=&format=csv` (`report:export`) — tanpa itu Reports tetap `ModulePending` | agen |
| 2 | `T-083` Administration Users/Roles/Orgs (`42-API.md` §11) — membuka Q-024 | agen |
| 3 | Dogfooding: catat P-067 sebagai task/comment di project BWDCS (`code: BWDCS`) di aplikasi | agen + user |
