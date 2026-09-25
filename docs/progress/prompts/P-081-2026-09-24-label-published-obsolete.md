# P-081 — 2026-09-24 — Jawaban Q-019/Q-021/published + label turunan (T-099)

| Field | Isi |
|---|---|
| ID | P-081 |
| Waktu mulai | 2026-09-24 21:15 (WIB) |
| Aktor | agen (opencode / Muse Spark) |
| Model / agen | muse-spark |
| Fase roadmap | Phase 4/5 (sisa tanya-jawab) |
| Task terkait | `T-099` DONE; `T-096`/`T-097`/`T-098` TODO baru |
| Status akhir | DONE |

---

## 1. Prompt User

> "Kalau ada pertanyaan silakan sampaikan" — diverifikasi pertanyaan terbuka nyata (Q-019 threading, Q-021 storage, published/obsolete; Q-025/Q-026 dan Q-022 dibiarkan sebagai catatan teks), diajukan 3 opsi via question tool. Jawaban: **threading implementasi (datar)**, **cookie HttpOnly**, **label turunan UI**.

## 2. Interpretasi & Scope

- Yang diminta: rekam 3 keputusan + kerjakan unit terkecil yang tuntas (label turunan).
- Yang TIDAK termasuk: backend/UI threading (T-096/T-097 sesudahnya); migrasi cookie (T-098 sesudahnya).
- Asumsi: tidak ada — ketiga keputusan ditanyakan eksplisit.
- Pertanyaan yang muncul: ketiganya, dijawab sekaligus.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Rekam jawaban + buat task T-096..T-099 | Jejak keputusan utuh |
| 2 | `documentLifecycle()` + header detail + test | 339 test hijau |
| 3 | Ledger + 6 pemeriksa | Serah terima bersih |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Q-019/Q-021 jawaban + jawaban published terekam; T-096/097/098 TODO, T-099 DONE | Satu prompt, satu keputusan per baris | Tidak ada pertanyaan menggantung dari sesi ini |
| 2 | Helper murni + `now` disuntik; nada ulang; tanggal rusak = absen | ADR-0012 utuh; tanpa tebak zona waktu; tanpa 500 klien | 6 + 1 test |
| 3 | Satu edit keliru di test (header terpotong) langsung dikembalikan sebelum lanjut | Edit tanpa baca ulang berisiko | Diverifikasi via baca baris |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `frontend/src/types/status.ts` | Changed | `documentLifecycle()` + tipe | — (label baca) |
| `frontend/src/types/status.test.ts` | Changed | 6 test turunan | — |
| `frontend/src/pages/Documents/DocumentDetail.tsx` | Changed | header pakai override + fallback | — |
| `frontend/src/pages/Documents/DocumentDetail.test.tsx` | Changed | 1 test header Published | — |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | jawaban Q-019/Q-021 | Q-019/Q-021 |
| `docs/progress/TASKS.md` | Changed | T-096/097/098 TODO, T-099 DONE | — |

> Daftar ini disalin ke `CHANGELOG.md`.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `cd frontend && npm run typecheck && npm run lint && npm run build` | bersih | PASS |
| 2 | `cd frontend && npm run test:run` | 32 berkas, 339 test (+7) | PASS |
| 3 | 6 pemeriksa | semua OK | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [x] Jika UI: Delivery Gate antislop dijalankan — label "Published"/"Obsolete" tanpa em dash; nada dari token; tanpa ukuran/posisi baru (tata letak tak disentuh → responsive-evidence tidak wajib)

## 7. Hasil & Dampak

- Selesai: 3 keputusan terekam + T-099 DONE.
- Belum selesai / sisa: T-096 (threading backend + migrasi 013 + ADR), T-097 (threading UI), T-098 (cookie HttpOnly + amandemen ADR-0023).
- Risiko / utang teknis: tidak ada.
- Dampak ke dokumen desain: tidak ada (label baca tidak masuk kontrak).

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (T-096..T-099)
- [x] `TRACEABILITY.md` diperbarui (tidak perlu — tanpa requirement kode baru)
- [x] `OPEN-QUESTIONS.md` diperbarui (3 jawaban)
- [x] ADR dibuat/diperbarui (tidak perlu — menyusul di T-096/T-098)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| Tinggi | T-096 threading backend (migrasi 013 + ADR + API) | agen |
| Sedang | T-097 threading UI (sesudah T-096) | agen |
| Sedang | T-098 cookie HttpOnly (kontrak + ADR-0023 + probe) | agen |
