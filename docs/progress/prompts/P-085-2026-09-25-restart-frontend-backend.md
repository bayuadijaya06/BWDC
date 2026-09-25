# P-085 — 2026-09-25 — Restart frontend + backend (binary segar)

| Field | Isi |
|---|---|
| ID | P-085 |
| Waktu mulai | 2026-09-25 11:15 (WIB) |
| Aktor | agen (opencode / Muse Spark) |
| Model / agen | muse-spark |
| Fase roadmap | — (operasional) |
| Task terkait | — (minta eksplisit pemilik, bukan task papan) |
| Status akhir | DONE |

---

## 1. Prompt User

> "tolong kill dulu frontend dan backend server, dan restart ulang"

## 2. Interpretasi & Scope

- Yang diminta: matikan kedua server, nyalakan lagi dengan kode kini.
- Yang TIDAK termasuk: perubahan kode; menyentuh layanan sesi lain (wms, vite operator lain); migrasi manual (startup otomatis).
- Asumsi: binary supervisor di-restart memakai berkas yang sama → rebuild in-place sebelum kill.

## 3. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Kill vite Business-Dev basi (73574, 85952); wms (52128) tidak disentuh | Hanya milik proyek ini | :5173 bebas |
| 2 | `make build` → `backend/bin/bwdcs` segar | Supervisor me-restart binary yang sama | Binary 25 Sep 11:21 |
| 3 | Kill `:8081` (12286 → respawn 2052 → kill → 2271) | Supervisor me-restart otomatis | Sehat + kode baru |
| 4 | Verifikasi: login → dashboard 13 KPI/11 chart + `byDepartment`; `goose_db_version` 13 | Bukti, bukan asumsi restart | PASS |
| 5 | `npm run dev` baru; proxy `/api` → 401 yang benar | Frontend hidup + tersambung backend | :5173 200 |

## 4. Bukti

- `lsof :8081` = biner baru; `curl /health` healthy; dashboard 13/11; goose v13.
- `:5173` 200; `/api/v1/auth/me` 401 (proxy bekerja).
- Catatan: satu vite operator lain (2074, flag eksplisit) masih berbagi :5173 — pra-ada, dua-duanya proxy ke backend segar yang sama, jadi konsisten.

## 5. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| Tinggi | T-098 cookie HttpOnly (kontrak + ADR-0023 + backend + frontend + probe) | agen |
