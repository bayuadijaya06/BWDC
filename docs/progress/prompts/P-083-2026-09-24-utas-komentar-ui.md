# P-083 — 2026-09-24 — Utas komentar UI di detail dokumen (T-097)

| Field | Isi |
|---|---|
| ID | P-083 |
| Waktu mulai | 2026-09-24 21:45 (WIB) |
| Aktor | agen (opencode / Muse Spark) |
| Model / agen | muse-spark |
| Fase roadmap | Phase 4 (komentar UI, lanjutan T-096) |
| Task terkait | `T-097` (UI; backend `T-096` P-082) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Lanjutkan CONTINUE.md" (×3) — Next action: T-097. Investigasi menemukan UI timeline **belum ada di mana pun** (bagian Comments = penanda belum dibangun; tanpa `services/comments.ts`); T-097 mengandaikannya. Diajukan 1 pertanyaan cakupan. Jawaban: **timeline minimal penuh di detail dokumen**.

## 2. Interpretasi & Scope

- Yang diminta: utas baca + tulis + balas + ubah/hapus + penanda, di DocumentDetail.
- Yang TIDAK termasuk: TaskDetail (tetap penanda; follow-up), bersarang visual (datar per ADR-0032), endpoint baru.
- Asumsi: tidak ada — cakupan ditanyakan eksplisit.
- Pertanyaan yang muncul: cakupan T-097, dijawab timeline penuh.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | service + query + test | Lapisan data berketentuan kontrak |
| 2 | CommentThread + kabel + test | UI hidup di detail dokumen |
| 3 | Browser nyata + responsive + ledger + 6 pemeriksa | Serah terima bersih |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `services/comments.ts` + `queries/comments.ts` (kunci per entitas) | Belum ada lapisan komentar di klien | 5 test service |
| 2 | `CommentThread.tsx`: datar + penanda dari daftar (+fallback), balas/tulis/ubah/hapus milik sendiri, hapus dua langkah, avatar inisial | FSD §7 display; hapus permanen → konfirmasi | 6 test incl. axe |
| 3 | Kabel di DocumentDetail; Comments keluar dari pendingSections | Penanda basi diganti UI nyata | 2 test detail diperbarui |
| 4 | Dua edit keliru diperbaiki sebelum lanjut (duplikat JSX, header test terpotong) | Baca ulang sesudah edit | Diverifikasi via baca baris |
| 5 | Misteri parent:None dipecahkan: port 5173 dipakai bersama sesi lain → vite basi → backend basi mengabaikan parent_id diam-diam | Tanpa bukti ini, kode benar 거의 disalahkan | Vite terisolasi :5174; curl + XHR membuktikan rantai |
| 6 | Skrip verifikasi `/tmp/verify-thread.mjs` (adhoc, tidak dikomit) | responsive-evidence tidak mengunjungi detail | verify-thread OK |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `frontend/src/services/comments.ts` | Added | tipe + CRUD + validasi | FR-CMT-01/03 |
| `frontend/src/services/comments.test.ts` | Added | 5 test | FR-CMT-01 |
| `frontend/src/queries/comments.ts` | Added | kunci + list + 3 mutasi | FR-CMT-01 |
| `frontend/src/pages/Documents/CommentThread.tsx` | Added | utas + penanda + balas/ubah/hapus | FR-CMT-01/03 |
| `frontend/src/pages/Documents/CommentThread.test.tsx` | Added | 6 test incl. axe | FR-CMT-01/03 |
| `frontend/src/pages/Documents/DocumentDetail.tsx` | Changed | kabel thread, Comments keluar pending | FR-CMT-03 |
| `frontend/src/pages/Documents/DocumentDetail.test.tsx` | Changed | mock komentar + sesuaikan 1 test | FR-CMT-03 |

> Daftar ini disalin ke `CHANGELOG.md`.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `cd frontend && npm run typecheck && npm run lint && npm run build` | bersih | PASS |
| 2 | `cd frontend && npm run test:run` | 34 berkas, 350 test (+11) | PASS |
| 3 | Server nyata :8089 + Chrome (`/tmp/verify-thread.mjs`) | login, utas live, balas UI, penanda Membalas; API: parent YA | PASS |
| 4 | `node scripts/responsive-evidence.mjs --url http://localhost:5174` | OK 16+24+laci | PASS |
| 5 | 6 pemeriksa | semua OK | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [x] Jika UI: Delivery Gate antislop dijalankan — primitives + token ada; tanpa em dash; avatar = inisial teks (tanpa gambar); hapus konfirmasi dua langkah; `aria-pressed` tidak dipakai (bukan toggle)

> Design Read: tidak ada keputusan visual baru; baris komentar mengikuti kepadatan teks yang ada.

## 7. Hasil & Dampak

- Selesai: T-097 DONE. Utas hidup di detail dokumen; TaskDetail tetap penanda (jujur).
- Belum selesai / sisa: T-098 cookie HttpOnly.
- Risiko / utang teknis: port dev bersama antar-sesi dapat mengarahkan verifikasi browser ke backend basi (ditemukan + diisolasi sesi ini; pelajaran: selalu pakai port unik + pastikan proxy target).
- Dampak ke dokumen desain: tidak ada (UI memakai kontrak §7 yang sudah ada).

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (T-097 DONE)
- [x] `TRACEABILITY.md` diperbarui (FR-CMT-01/03)
- [x] `OPEN-QUESTIONS.md` diperbarui (tidak perlu — jawaban sudah P-081)
- [x] ADR dibuat/diperbarui (tidak perlu — ADR-0032 sudah P-082)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| Tinggi | T-098 cookie HttpOnly (kontrak + ADR-0023 + backend + frontend + probe) | agen |
