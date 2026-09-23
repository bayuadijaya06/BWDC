# P-066 — 2026-09-23 — Polish filter, tabel Tasks, dan tombol Create

| Field | Isi |
|---|---|
| ID | P-066 |
| Waktu mulai | 2026-09-23 23:50 (Asia/Jakarta) |
| Aktor | agen |
| Model / agen | muse-spark-1.2-contributor-free |
| Fase roadmap | 4 — Frontend polish |
| Task terkait | `T-089` — filter/tabel/tombol |
| Status akhir | DONE |

---

## 1. Prompt User

> "Upgrade tampilan filter untuk seluruh menu, terasa terlalu blending dengan halaman, seperti kurang kontras bahwa itu adalah filter, panjang dari tiap field juga berbeda-beda, memberikan kesan kurang rapi, upgrade juga tampilan tabel pada menu Tasks, terasa terlalu penuh. Tombol untuk create pada tiap menu juga kurang kontras, seperti teks biasa, upgrade juga."

## 2. Interpretasi & Scope

- Filter di Projects/Documents/Tasks blending dengan `bg-paper-100`, field `select` lebar acak (data pengguna), tabel Tasks 6 kolom `150-190px` terlalu penuh, tombol Create `primary` terasa seperti teks biasa.
- Yang TIDAK termasuk: menambah endpoint, mengubah skema, menambah task baru di luar `T-089`.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | `Button.tsx` `primary` `bg-accent text-paper-000 border-accent-strong shadow-sm font-semibold` | Create kontras di light/dark |
| 2 | Filter `flex flex-wrap gap-3 rounded-panel border border-line bg-surface-raised p-3 shadow-sm` + kolom `flex-1 min-w-[140px] max-w-[200px] min-w-0` | Kontras + lebar konsisten |
| 3 | Tasks `DataTable` `density="comfortable"` + kolom `due_date 140px`, `assignee 120px`, `project 130px` | Tidak penuh |
| 4 | `responsive-evidence` 16×4 widths OK | Tidak breaking |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `frontend/src/components/common/Button.tsx` `primary: "bg-accent text-paper-000 hover:bg-accent-strong shadow-sm font-semibold border border-accent-strong"` | `bg-ink-900` blending di dark (`#15181c` di `#14161a`), `accent` adaptif light `#0e5b63`→`#7fd1d9` | Create kontras |
| 2 | `frontend/src/pages/{Tasks,Documents,Projects}/index.tsx` `className="flex flex-wrap items-end gap-3 rounded-panel border border-line bg-surface-raised p-3 shadow-sm"` + `className="flex flex-col gap-1.5 min-w-0 flex-1 min-w-[140px] max-w-[200px]"` | Filter `bg-paper-100` → `bg-surface-raised` + lebar konsisten `140-200px` | Rapi |
| 3 | `frontend/src/pages/Tasks/index.tsx` `taskColumns` `due_date 190→140`, `assignee 150→120`, `project 190→130` + `DataTable density="comfortable"` (`h-11` 44px) | 6 kolom `150-190` → `h-9` terlalu penuh | `h-11` airy |
| 4 | `frontend/src/pages/Dashboard/index.tsx` `statusColors` `#hex` → `var(--color-status-*)` + `frontend/src/pages/Projects/ProjectDetail.tsx` ` — ` → ` - ` | `tokens.contrast.test.ts` `antislop-refs` | `antislop-refs OK` |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `frontend/src/components/common/Button.tsx` | Changed | `primary` `bg-accent` `shadow-sm` | `DESIGN.md` §2 |
| `frontend/src/pages/Tasks/index.tsx` | Changed | Filter `bg-surface-raised` + kolom `flex-1` + tabel `density comfortable` | `51-UX.md` §6.1 |
| `frontend/src/pages/Documents/index.tsx` | Changed | Filter `bg-surface-raised` + kolom `flex-1` | `51-UX.md` §6.1 |
| `frontend/src/pages/Projects/index.tsx` | Changed | Filter `bg-surface-raised` + kolom `flex-1` | `51-UX.md` §6.1 |
| `frontend/src/pages/Dashboard/index.tsx` | Changed | ` - ` untuk R-02 + `var(--color-status-*)` | R-02 |
| `frontend/src/pages/Projects/ProjectDetail.tsx` | Changed | ` - ` untuk R-02 | R-02 |
| `frontend/vite.config.ts` | Changed | `optimizeDeps: { include: ["recharts"] }` + `rm -rf .vite` | `T-073` |
| `docs/progress/TASKS.md` | Changed | `T-089` TODO → DONE | `T-089` |
| `docs/progress/prompts/P-066-...md` | Added | Log prompt ini | — |

> Daftar ini disalin ke `CHANGELOG.md` §2026-09-23 P-066.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `npm run typecheck` | OK | PASS |
| 2 | `npm run lint` | OK | PASS |
| 3 | `npm run test:run` | 30 files, **294 test** PASS | PASS |
| 4 | `npm run build` | 893kB (recharts) | PASS |
| 5 | `ADMIN_PASSWORD=Admin123! node scripts/responsive-evidence.mjs` | 16×4 widths OK, 24 tema OK, laci OK | PASS |
| 6 | `bash scripts/check-antislop-refs.sh` | `antislop-refs OK` | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten
- [x] Jika UI: `Button` `bg-accent`, filter `bg-surface-raised`, Tasks `h-11`

## 7. Hasil & Dampak

- Selesai: **Filter** kini `rounded-panel border bg-surface-raised p-3 shadow-sm` — tidak blending dengan `bg-paper-100`; field `flex-1 min-w-[140px] max-w-[200px] min-w-0` — lebar konsisten, tidak acak karena `select` terpanjang. **Tabel Tasks** `density comfortable` (`h-11`) + kolom sempit (`due 140`, `assignee 120`, `project 130`) — tidak penuh. **Tombol Create** `bg-accent text-paper-000 border-accent-strong shadow-sm font-semibold` — kontras di light (`#0e5b63`/`#ffffff` 7:1) dan dark (`#7fd1d9`/`#1c1f24`).
- Belum selesai / sisa: `T-080`..`T-083` masih TODO, `T-074` Phase 5.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` belum diperbarui (hanya polish — STATE tetap pada P-065 `279 test`)
- [x] `SESSION-LOG.md` ditambah entri P-066
- [x] `CHANGELOG.md` ditambah entri P-066
- [x] `TASKS.md` diperbarui — `T-089` DONE
- [x] `TRACEABILITY.md` tidak berubah
- [x] `OPEN-QUESTIONS.md` tidak berubah
- [x] ADR tidak ada

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | `T-080` Workflow `?project_id` (sudah `T-080` DONE P-064) → `T-081` Activity `?project_id` (DONE P-065) → `T-082` Reports / `T-083` Admin | agen |
