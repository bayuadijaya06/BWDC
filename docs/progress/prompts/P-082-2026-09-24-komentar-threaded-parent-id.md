# P-082 — 2026-09-24 — Threading komentar backend (T-096, C-050 FIXED)

| Field | Isi |
|---|---|
| ID | P-082 |
| Waktu mulai | 2026-09-24 21:30 (WIB) |
| Aktor | agen (opencode / Muse Spark) |
| Model / agen | muse-spark |
| Fase roadmap | Phase 3 (komentar, lanjutan T-042) |
| Task terkait | `T-096` (backend; UI `T-097` sesudahnya) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Lanjutkan CONTINUE.md" — Next action P-082: T-096 threading backend (keputusan P-081: datar + `parent_id`).

## 2. Interpretasi & Scope

- Yang diminta: kolom + ADR + API berbalas + test; daftar tetap datar.
- Yang TIDAK termasuk: UI thread (T-097); bersarang rekursif (ditolak ADR-0032); izin baru (matriks utuh).
- Asumsi: tidak ada — bentuk datar diputuskan pemilik P-081.
- Pertanyaan yang muncul: tidak ada.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Migrasi 013 + ADR-0032 + kontrak §7 | Keputusan tertulis dulu |
| 2 | Model/repo/service/DTO/handler | API berbalas |
| 3 | Ganti test pengunci + docs skema + C-050 FIXED | 296 test hijau |
| 4 | Server nyata + ledger + 6 pemeriksa | Serah terima bersih |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `parent_id` FK CASCADE + indeks parsial; `FindOnEntity`; `CountReplies` rekursif | Induk-entitas-sama tanpa join cakupan; audit jujur semua tingkat | Kueri minimal |
| 2 | Validasi satu pesan (hilang = beda entitas); `parent_id` immutable (PATCH hanya content) | Anti-pemetaan entitas; kontrak eksplisit | 422 konsisten |
| 3 | Audit `parent_id` + `deleted_reply_count`; tabel ADR README dilengkapi 0027-0032 (tertunggak P-076) | Tabel berhenti di 0026 | Lengkap |
| 4 | Typo path edit (`posindomnia`) langsung gagal terlihat — diulang dengan path benar | Edit tool memvalidasi keberadaan | Tidak ada sisa |
| 5 | C-050 OPEN → FIXED + marker 81/1 di 4 berkas + prosa §1/ringkasan | Angka dari tabel, bukan ingatan (C-044/055) | Checker yang menilai |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/migration/013_comment_parent_id.sql` | Added | kolom + indeks + down | FR-CMT-01 |
| `docs/adr/0032-komentar-threaded-parent-id-datar.md` | Added | keputusan + 4 alternatif ditolak | Q-019 |
| `backend/internal/model/comment.go` | Changed | `ParentID` | FR-CMT-01 |
| `backend/internal/repository/comment_repository.go` | Changed | select/scan/create + `FindOnEntity` + `CountReplies` | FR-CMT-01 |
| `backend/internal/service/comment_service.go` | Changed | input + validasi + audit + hapus hitung | FR-CMT-01 |
| `backend/internal/dto/comment_dto.go` | Changed | request/response/mapping | FR-CMT-01 |
| `backend/internal/handler/comment_handler.go` | Changed | teruskan + nil-UUID + mapping 422 | FR-CMT-01 |
| `backend/internal/service/comment_service_test.go` | Changed | 3 test ganti pengunci | FR-CMT-01 |
| `backend/internal/handler/comment_handler_test.go` | Changed | `TestCommentReplyOverHTTP` + payload | FR-CMT-01 |
| `docs/design/42-API.md` | Changed | §7 POST/GET/DELETE threading | FR-CMT-01 |
| `docs/design/50-FSD.md` | Changed | §7 didukung ADR-0032 | FR-CMT-01 |
| `docs/design/41-DATABASE.md` | Changed | §2.5 kolom + §4 baris 013 | — |
| `docs/adr/README.md` | Changed | tabel dilengkapi 0027-0032 | — |
| `docs/progress/audits/AUDIT-001-*.md` | Changed | C-050 FIXED + marker/prosa 81/1 | C-050 |
| `README.md` | Changed | migrasi 001–013 | — |

> Daftar ini disalin ke `CHANGELOG.md`.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `cd backend && go build ./... && go vet ./... && make test` | 9 paket ok | PASS |
| 2 | `grep -rh "^func Test" backend/internal --include="*_test.go" \| wc -l` | 296 (+3 net: −1 +4) | PASS |
| 3 | Server nyata :8089 (migrasi v13) | balas 201 + parent; asing 404-entitas; hapus induk → total 0; cleanup 200 | PASS |
| 4 | 6 pemeriksa | semua OK | PASS |

- [x] Typecheck / build dijalankan (backend; tanpa frontend — tidak diubah)
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [x] Jika UI: tidak menyentuh UI

## 7. Hasil & Dampak

- Selesai: T-096 DONE; C-050 FIXED (81/1 — OPEN teknis terakhir tertutup, tersisa C-063 produk).
- Belum selesai / sisa: T-097 UI thread; T-098 cookie.
- Risiko / utang teknis: `deleted_reply_count` dihitung sebelum DELETE (selisih balapan diterima sadar, tercatat di ADR).
- Dampak ke dokumen desain: `42-API.md` §7, `50-FSD.md` §7, `41-DATABASE.md` §2.5/§4 — sudah diperbarui.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (T-096 DONE)
- [x] `TRACEABILITY.md` diperbarui (FR-CMT-01)
- [x] `OPEN-QUESTIONS.md` diperbarui (tidak perlu — jawaban sudah P-081)
- [x] ADR dibuat/diperbarui (ADR-0032 baru + tabel README)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| Tinggi | T-097 threading UI (penanda + balas dari komentar) | agen |
| Sedang | T-098 cookie HttpOnly | agen |
