# P-098 — 2026-09-25 — Halaman Administration, Users + Roles (T-105, bagian 1/2)

| Field | Isi |
|---|---|
| ID | P-098 |
| Waktu mulai | 2026-09-25 (zona waktu lokal) |
| Aktor | agen |
| Fase roadmap | 4 (halaman bisnis) |
| Task terkait | `T-105` (IN PROGRESS; Orgs/Categories/Settings menyusul P-099) |
| Status akhir | DONE (bagian 1/2; T-105 tetap IN PROGRESS) |

---

## 1. Prompt User

> "Lanjutkan sesuai CONTINUE.md" — next action: T-105 halaman Administration.

## 2. Interpretasi & Scope

- Yang diminta: `/admin` + Users (penuh via T-102) + Roles (daftar; matriks tanpa endpoint) + navigasi.
- Yang TIDAK termasuk: Orgs/Categories/Settings (P-099); kontrak/izin baru; matriks permission UI (tanpa endpoint — dinyatakan terbuka); `PATCH /admin/settings/:key` (terkontrak tanpa route — ditemukan, tidak dikerjakan di sesi UI ini).
- Asumsi/keputusan bentuk:
  - Tab `?section=` di URL (users/roles dulu; organizations/categories/settings ditambah P-099).
  - Reset password = dialog berisi password sementara pilihan admin (kontrak: `new_password` dari pemanggil; Q-027 tetap: tanpa bell).
  - Tombol per izin (`user:create`/`user:update`/`user_role:manage`); halaman butuh `user:read`.
- Pertanyaan: tidak ada.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Baca FSD §10 + lapisan admin + route settings | tanpa tebakan endpoint |
| 2 | Log ini + T-105 IN PROGRESS SEBELUM mengubah file | rencana tercatat dulu |
| 3 | `services/admin.ts` (+test) dan `queries/admin.ts` (hooks + invalidasi) | kontrak §11 apa adanya |
| 4 | `pages/Administration/index.tsx`: shell + UsersTab + RolesTab | FSD §10.1–§10.2 |
| 5 | `navigation.ts` ready + route `/admin` + baris §2.1 + evidence `pages` | dua arah §2.1 |
| 6 | Test halaman + `typecheck`/`lint`/`test:run`/`build` | bukti |
| 7 | Ledger (T-105 tetap IN PROGRESS) | sesi tertutup jujur |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Baca FSD + lapisan + router | cegah karangan endpoint | peta lengkap; settings tanpa route |
| 2 | `services/admin.ts` (+test) + `queries/admin.ts` (hooks) | kontrak §11 apa adanya | 4 + 0 test hijau |
| 3 | `pages/Administration/index.tsx` (shell + Users + Roles) + navigasi + route + baris §2.1 + evidence | FSD §10.1–§10.2, dua arah §2.1 | `typecheck`/`lint` bersih |
| 4 | Test halaman (7 test) | bukti perilaku + gate | hijau; gigi gate-Role → FAIL lalu hijau |
| 5 | Diagnosis halaman blank di peramban: `:8081` basi → server segar → tetap blank → CDP: `TypeError row.roles.join` (dua lapis: key kapital + null) | klaim tanpa angka dilarang; observasi langsung | akar ditemukan |
| 6 | Perbaiki: tag JSON `UserListItem` + slice non-nil (users/roles/orgs) + guard `?? []` klien + test regresi | kontrak §11 + skew backend lama | hijau; gigi tag-dicabut → FAIL |
| 7 | Server segar + evidence ulang + bersihkan jejak + ledger | serah terima bersih | `responsive-evidence OK` (24 + 36 + laci); DB 231/100 (catatan di §7) |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `docs/progress/prompts/P-098-2026-09-25-halaman-administration-1.md` | Added | Log sesi ini | — |
| `frontend/src/services/admin.ts` | Changed | create/update/setRoles/reset + test | FR-ROLE-04, FR-AUTH-07/08 |
| `frontend/src/services/admin.test.ts` | Added | 4 test lapisan data | — |
| `frontend/src/queries/admin.ts` | Changed | hooks + invalidasi | — |
| `frontend/src/pages/Administration/index.tsx` | Added | Shell + UsersTab + RolesTab | — |
| `frontend/src/pages/Administration/Administration.test.tsx` | Added | 7 test halaman | — |
| `frontend/src/config/navigation.ts` | Changed | Administration ready | — |
| `frontend/src/App.tsx` + `App.test.tsx` | Changed | route `/admin`; contoh pending → Audit | — |
| `docs/design/51-UX.md` | Changed | baris Administration | — |
| `scripts/responsive-evidence.mjs` | Changed | entri administration | — |
| `backend/internal/service/user_service.go` | Changed | tag JSON `UserListItem` | FR-ROLE-04, FR-AUTH-07/08 |
| `backend/internal/repository/user_repository.go` | Changed | `Roles()` non-nil | — |
| `backend/internal/handler/user_handler_test.go` | Changed | test regresi casing + null | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `npm run typecheck` + `npm run lint` + `vite build` | bersih | PASS |
| 2 | `npx vitest run` (3 berkas baru) | 11/11 hijau; gigi gate-Role → FAIL lalu hijau | PASS |
| 3 | `npm run test:run` (penuh) | 376/38 hijau (termasuk App.test yang diperbarui) | PASS |
| 4 | server segar `:8089` + Vite `:5174` + `responsive-evidence` | blank → diagnosis → perbaiki → `OK`: 24 layout + 36 tema + laci | PASS |
| 5 | `make test` (pascaperbaikan backend) + 6 pemeriksa | 9 paket **313 test**; `ledger OK 313` (+1 test regresi; STATE diselaraskan) | PASS | |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan (targeted + suite penuh + peramban)
- [x] Perubahan dokumen dicek konsisten
- [x] UI: halaman dibuka di dev server sungguhan + diukur mesin; Design Read di bawah

> **Design Read (dials sama):** kelanjutan Reports — header + tab + filter/tabel + dialog; tombol destructive (reset) memakai `primary` di dialog konfirmasi dengan deskripsi akibatnya yang eksplisit (bukan merah besar); batas (matriks tanpa endpoint, tanpa bell) sebagai teks.

## 7. Hasil & Dampak

- Selesai (1/2): shell + Users + Roles + navigasi + evidence OK. T-105 tetap IN PROGRESS (Orgs/Categories/Settings).
- Belum selesai / sisa: bagian 2/2; `PATCH /admin/settings/:key` tanpa route (temuan samping).
- Temuan penting sesi ini: amplop `GET /admin/users` berkunci kapital + `roles` bisa null — halaman admin blank total di peramban (ditemukan via CDP exception, bukan via test yang selalu mock bentuk benar). Pelajaran: mock test tidak menangkap ketidakcocokan amplop; test regresi kini mengunci bentuk mentah respons.
- DB dev: 231/100 (+5/+5 vs baseline P-096 226/95). Dihapus yang pasti milik sesi ini (1 pasang evidence +dinilai; P-096 menghapus 5). Sisa tak teratribusi (00:18/00:21/00:25) DIBIARKAN — menghapus baris sesi lain lebih buruk daripada angka tak bulat. Pelajaran diulang: catat hitungan + id SEBELUM probe, bukan sesudah.
- Risiko / utang teknis: kolom `UserListItem` kini lowercase (kontrak §11 tidak mengeja casing — klien + `model.User` sudah lowercase).
- Dampak ke dokumen desain: baris §2.1 + entri evidence; tanpa ADR/migrasi/izin baru.

## 8. Update Ledger

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` — `T-105` IN PROGRESS (catatan dimutakhirkan)
- [x] `TRACEABILITY.md` — FR-ROLE-04, FR-AUTH-07/08 (UI)
- [x] `OPEN-QUESTIONS.md` — tidak perlu
- [x] ADR — tidak perlu
- [x] `CONTINUE.md` §0 diperbarui

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | T-105 bagian 2/2: Orgs + Categories + Settings + evidence + DONE | agen (sesi berikut) |
