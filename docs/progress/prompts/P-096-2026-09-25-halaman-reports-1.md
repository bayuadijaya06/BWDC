# P-096 — 2026-09-25 — Halaman Reports, tab Projects + Documents (T-104, bagian 1/2)

| Field | Isi |
|---|---|
| ID | P-096 |
| Waktu mulai | 2026-09-25 (zona waktu lokal) |
| Aktor | agen |
| Fase roadmap | 4 (halaman bisnis) |
| Task terkait | `T-104` (IN PROGRESS; tab Tasks menyusul P-097) |
| Status akhir | DONE (bagian 1/2; tab Tasks menyusul, T-104 tetap IN PROGRESS) |

---

## 1. Prompt User

> "Lanjutkan ke next task" — T-104 halaman Reports (`50-FSD.md` §10.6, `51-UX.md` §2.1).

## 2. Interpretasi & Scope

- Yang diminta: `/reports` 3 tab + daftar kompak + filter per-tab + Export CSV per tab.
- Yang TIDAK termasuk: tab Tasks (P-097, T-104 tetap IN PROGRESS); kontrak baru (kontrak §10 eksplisit: halaman memakai endpoint daftar yang ada); izin baru (`report:read`/`report:export` ada).
- Asumsi/keputusan bentuk (dicatat agar tidak dikira tebakan):
  - Tab `?tab=` di URL (berbagi seperti tab Documents/Tasks); filter per-tab di state lokal (kunci param `search`/`status` dipakai ketiga daftar — satu URL tidak muat tiga keadaan).
  - Kolom tabel = kolom halaman modul terkait (FSD melarang kolom berbeda): disalin dengan komentar penunjuk; drift dijaga test header.
  - Export mengirim subset yang diterima endpoint (`search`/`status`/`project_id`): tab Documents menyatakan terbuka penyaring yang tidak ikut (kategori/pemilik/rentang) — pola C-063, bukan disenyapkan.
  - Batasan server yang ditemukan (bukan scope sesi ini): kontrak §10 menyebut `date_from`/`date_to`, tetapi `parseReportExportQuery` mengabaikannya — dicatat di sini, tidak diperbaiki di sesi UI ini.
- Pertanyaan: tidak ada.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Baca §10/§10.6 + queries/services + pola 3 halaman + navigasi + evidence | tanpa tebakan bentuk |
| 2 | Log ini + T-104 IN PROGRESS SEBELUM mengubah file | rencana tercatat dulu |
| 3 | `services/reports.ts` (`exportReport` blob + nama dari disposition) + test | kontrak §10 apa adanya |
| 4 | `pages/Reports/index.tsx`: shell `?tab=` + ProjectsTab + DocumentsTab (filter, tabel, export, tautan daftar) | paritas filter per-tab |
| 5 | `navigation.ts` ready + route `/reports` + baris §2.1 + evidence `pages` | dua arah §2.1 |
| 6 | Test halaman (tab, daftar, export, gate, kosong/gagal) + `typecheck`/`lint`/`test:run`/`build` | bukti |
| 7 | Ledger (T-104 tetap IN PROGRESS) | sesi tertutup jujur |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Baca kontrak + pola + evidence | cegah tebakan bentuk | peta lengkap |
| 2 | `services/reports.ts` + test (export blob + nama disposition) | kontrak §10 apa adanya | 4 test hijau |
| 3 | `pages/Reports/index.tsx`: shell `?tab=` + ProjectsTab + DocumentsTab (filter/paritas, tabel, export, tautan daftar) | FSD §10.6 + §2.1 | `typecheck`/`lint` bersih |
| 4 | Navigasi ready + route `/reports` + baris §2.1 + evidence `pages` | dua arah §2.1 (C-079) | `navigation OK`, `check-navigation` hijau |
| 5 | Test halaman (6 test: tab, daftar, export, gate, kosong/gagal, axe) | bukti perilaku | hijau; gigi: export tanpa filter → FAIL |
| 6 | Perbaiki App.test pending-Reports → Administration | contoh pending harus yang belum dibangun | hijau, suite 363/36 |
| 7 | Bukti peramban: diagnosis `:8081` basi → server segar `:8089` + Vite `:5174` → ukur → 1 cacat → perbaiki → OK | klaim tata letak tanpa angka dilarang (R-35) | `responsive-evidence OK`: 20 layout + 30 tema + laci; DB dev dibersihkan (sisa +1 dijelaskan di bawah) |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `docs/progress/prompts/P-096-2026-09-25-halaman-reports-1.md` | Added | Log sesi ini | — |
| `frontend/src/services/reports.ts` | Added | `exportReport` blob + nama disposition | — (agregasi baca §10) |
| `frontend/src/services/reports.test.ts` | Added | 4 test kontrak export | — |
| `frontend/src/pages/Reports/index.tsx` | Added | Shell `?tab=` + 2 tab + export + tautan daftar | — |
| `frontend/src/pages/Reports/Reports.test.tsx` | Added | 6 test halaman | — |
| `frontend/src/config/navigation.ts` | Changed | Reports ready, task T-104 | — |
| `frontend/src/App.tsx` | Changed | route `/reports` | — |
| `frontend/src/App.test.tsx` | Changed | contoh pending → Administration | — |
| `docs/design/51-UX.md` | Changed | baris Reports tanpa "belum dibangun" | — |
| `scripts/responsive-evidence.mjs` | Changed | entri reports + catatan expectedRanges | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `npm run typecheck` + `npm run lint` + `vite build` | bersih, `dist/` | PASS |
| 2 | `npx vitest run` (2 berkas baru) | 10/10 hijau; gigi export → FAIL lalu hijau | PASS |
| 3 | `npm run test:run` (penuh) | 363/36 hijau (termasuk App.test yang diperbarui) | PASS |
| 4 | `check-navigation` / `check-antislop-refs` / `check-ledger` / `check-doc-links` | semuanya OK | PASS |
| 5 | `node scripts/responsive-evidence.mjs --url :5174` (backend segar `:8089`) | cacat tap-target 1 → perbaiki → `OK`: 20 layout + 30 tema + laci | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan (targeted + suite penuh + peramban)
- [x] Perubahan dokumen dicek konsisten
- [x] UI: halaman dibuka di dev server sungguhan + diukur mesin; Design Read di bawah

> **Design Read (Dials ENERGY 1 / RHYTHM 2 / MOTION 1):** halaman operasional
> padat — header + tab + filter + tabel + tombol export dalam satu alur vertikal
> tanpa hero; tab berupa tombol berbatas (bukan pill berwarna) dengan penanda
> `aria-selected`; satu-satunya aksen pada tombol Export primer; tanpa animasi
> (tabel memakai kerangka statis bila memuat). Honest boundary (catatan batas
> export, izin export) ditampilkan sebagai teks, bukan disembunyikan.

## 7. Hasil & Dampak

- Selesai (1/2): shell + 2 tab + export + navigasi + evidence OK. T-104 tetap IN PROGRESS untuk tab Tasks.
- Belum selesai / sisa: tab Tasks (+ test + evidence ulang); sapuan `date_from`/`date_to` kontrak §10 yang diabaikan handler (temuan samping, bukan scope sesi ini).
- Pelajaran alat: (a) `:8081` basi pra-P-086 mematikan sesi tiap reload — diagnosis via `POST /auth/refresh {}` (422 = lama, 401 = baru); server segar wajib dibuild dari kode sesi ini dan dijaga hidup selama startup; (b) jangan pipe `go test`/output panjang ke `head` (SIGPIPE membunuh runner); (c) DB dev: catat hitungan + id baris SEBELUM probe; 5 pasang milik sesi ini dihapus via jalur pemeliharaan, sisa +1 dijelaskan (baris 22:07/22:09 kepemilikan tak jelas — dibiarkan).
- Risiko / utang teknis: kolom tabel disalin dari halaman modul (drift dijaga test header); `expectedRanges: 0` tidak mengukur range tab Documents (dicatat di skrip).
- Dampak ke dokumen desain: baris §2.1 + entri evidence; tanpa ADR/kontrak/izin baru.

## 8. Update Ledger

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` — `T-104` IN PROGRESS (catatan dimutakhirkan)
- [x] `TRACEABILITY.md` — tidak ada FR baru (laporan = agregasi baca)
- [x] `OPEN-QUESTIONS.md` — tidak perlu
- [x] ADR — tidak perlu
- [x] `CONTINUE.md` §0 diperbarui

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | T-104 bagian 2/2: tab Tasks + evidence peramban + DONE | agen (sesi berikut) |
