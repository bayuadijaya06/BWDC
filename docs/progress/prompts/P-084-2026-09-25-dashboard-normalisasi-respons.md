# P-084 — 2026-09-25 — Dashboard tahan backend lama + C-083

| Field | Isi |
|---|---|
| ID | P-084 |
| Waktu mulai | 2026-09-25 11:00 (WIB) |
| Aktor | agen (opencode / Muse Spark) |
| Model / agen | muse-spark |
| Fase roadmap | Phase 4 (bugfix laporan pengguna) |
| Task terkait | `T-100` |
| Status akhir | DONE |

---

## 1. Prompt User

> Paste stack `TypeError: charts.byDepartment is undefined` (DashboardPage) + "perbaiki ini dulu". "React 18" di stack adalah label overlay generik — repo dan node_modules React 19.3.0.

## 2. Interpretasi & Scope

- Yang diminta: hilangkan layar putih saat respons tak lengkap.
- Yang TIDAK termasuk: mematikan/mengganti backend basi `:8081` (milik supervisor sesi lain, me-restart sendiri — P-076); migrasi data; endpoint baru.
- Asumsi: tidak ada — akar dipastikan via bukti (binary 23 Sep pra-P-076).
- Pertanyaan yang muncul: tidak ada.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Normalisasi di `fetchDashboard` + test regresi | Halaman tak pernah baca field hilang |
| 2 | Buktikan akar lingkungan (binary basi) tanpa menyentuhnya | Diagnosis, bukan tuduhan |
| 3 | C-083 + T-100 + ledger + 6 pemeriksa | Serah terima bersih |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `normalizeDashboardData` (KPI→0, chart→[], funnel objek) dipakai `fetchDashboard` | Satu tempat; panel kosong yang sudah ada yang tampil | Crash tertutup untuk kombinasi backend lama/baru mana pun |
| 2 | Test pass-through lama ditulis ulang + regresi service + regresi halaman | Test lama mengunci perilaku yang salah | 352 hijau |
| 3 | `backend/bin/bwdcs` 23 Sep + PID supervisor → akar tanpa sentuhan | Membangun ulang binary orang lain bukan wewenang sesi ini | Petunjuk rebuild untuk pengguna di §9 |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `frontend/src/services/analytics.ts` | Changed | `normalizeDashboardData` + pakai | FR-DASH-02 |
| `frontend/src/services/analytics.test.ts` | Changed | tulis ulang + regresi warisan | FR-DASH-02 |
| `frontend/src/pages/Dashboard/Dashboard.test.tsx` | Changed | regresi halaman payload lama | FR-DASH-02 |
| `docs/progress/audits/AUDIT-001-*.md` | Changed | baris C-083 + 83/82/1 | C-083 |
| `docs/progress/audits/README.md` | Changed | marker + prosa 83/82/1 | C-083 |
| `AGENTS.md` | Changed | marker + angka 83/82/1 | C-083 |
| `CONTINUE.md` | Changed | marker + angka + §0 P-084 | C-083 |
| `docs/progress/STATE.md` | Changed | marker + angka + §0 P-084 | C-083 |
| `docs/progress/TASKS.md` | Changed | `T-100` DONE | T-100 |

> Daftar ini disalin ke `CHANGELOG.md`.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `cd frontend && npm run typecheck && npm run lint && npm run build` | bersih | PASS |
| 2 | `cd frontend && npm run test:run` | 34 berkas, 352 test (+2) | PASS |
| 3 | Diagnosis: `ls -la backend/bin/bwdcs` (23 Sep) + `lsof :8081` (PID supervisor sesi lain) | akar terbukti tanpa sentuhan | PASS |
| 4 | 6 pemeriksa | semua OK | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [x] Jika UI: Delivery Gate antislop dijalankan — tanpa teks/kolom/warna baru (logika ketahanan murni)

## 7. Hasil & Dampak

- Selesai: T-100 DONE; C-083 FIXED (83/82/1).
- Belum selesai / sisa: T-098 cookie HttpOnly.
- Risiko / utang teknis: backend `:8081` pengguna tetap basi sampai dibangun ulang dari sumber kini (migrasi 012–013 ikut startup).
- Dampak ke dokumen desain: tidak ada (kontrak §13 tak berubah).

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (T-100 DONE)
- [x] `TRACEABILITY.md` diperbarui (FR-DASH-02 bukti P-084)
- [x] `OPEN-QUESTIONS.md` diperbarui (tidak perlu — tanpa pertanyaan baru)
- [x] ADR dibuat/diperbarui (tidak perlu)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| Tinggi | T-098 cookie HttpOnly (kontrak + ADR-0023 + backend + frontend + probe) | agen |
| Pengguna | Bangun ulang backend dari sumber kini (`cd backend && make build`, restart biner `:8081`) agar migrasi 012–013 dan endpoint Phase 5 hidup | pemilik |
