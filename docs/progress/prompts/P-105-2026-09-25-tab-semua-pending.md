# P-105 — 2026-09-25 — Tab Semua Pending di Approvals (T-109, bug laporan pengguna)

| Field | Isi |
|---|---|
| ID | P-105 |
| Waktu mulai | 2026-09-25 15:30 (WIB) |
| Aktor | agen (opencode / Muse Spark) |
| Model / agen | muse-spark |
| Fase roadmap | Phase 4 (bugfix laporan pengguna) |
| Task terkait | `T-109` |
| Status akhir | DONE |

---

## 1. Prompt User

> Paste panel "Bagian lain" DocumentDetail + laporan: "Dokumen yang diajukan untuk approval tidak muncul di halam approvals (semua status)". Plus: Activity (T-108 ada) dan Related Tasks (butuh keputusan kontrak — tidak dibuatkan task).

## 2. Interpretasi & Scope

- Yang diminta: dokumen submit harus terlihat di Approvals.
- Yang TIDAK termasuk: endpoint/scope baru (kontrak cukup); mengubah arti tab Pending; Related Tasks (keputusan kontrak); Activity (T-108).
- Asumsi: tidak ada — FSD §5.4 sudah menuntut tab All Pending; yang hilang hanya implementasinya.
- Pertanyaan yang muncul: tidak ada.

## 3. Diagnosis (bukti, bukan tebakan)

| # | Langkah | Hasil |
|---|---|---|
| 1 | Submit live `:8089` → 201 | Instance running tercipta |
| 2 | `?status=running&scope=assigned_to_me` → total 0; tanpa scope → total 3 | Backend benar; UI tak pernah menanyakan tanpa scope kecuali Approved/Rejected |
| 3 | `50-FSD.md` §5.4 dibaca | Tab "All Pending" (`status=running`, Admin/Manager) memang dispec tapi tak dibangun |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Tab keempat `Semua Pending` + gate role + fallback URL crafting | FSD §5.4; pencocokan di daftar terlihat | Tanpa kecualikan revisi (visibilitas, bukan antrean tindak) |
| 2 | 2 test tab + gate | Perilaku gate harus terkunci | 393 hijau |
| 3 | Chrome nyata (`verify-apv OK`) + cleanup proyek uji + responsive | Bukti ujung-ke-ujung | 28 layout + 42 tema OK |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `frontend/src/pages/Approvals/index.tsx` | Changed | tab + gate + empty state + note | 50-FSD §5.4 |
| `frontend/src/pages/Approvals/Approvals.test.tsx` | Changed | 2 test tab/gate | 50-FSD §5.4 |
| `docs/progress/TASKS.md` | Changed | `T-109` DONE | T-109 |
| `docs/progress/TRACEABILITY.md` | Changed | UI-APPROVALS bukti P-105 | UI-APPROVALS |

> Daftar ini disalin ke `CHANGELOG.md`.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `cd frontend && npm run typecheck && npm run lint && npm run build` | bersih | PASS |
| 2 | `cd frontend && npm run test:run` | 39 berkas, 393 test (+2) | PASS |
| 3 | Live `:8089` + Chrome | submit 201; Semua Pending tampil; Pending tak bocor; cleanup 200 | PASS |
| 4 | `responsive-evidence --url :5174` | OK 28 + 42 + laci | PASS |
| 5 | 6 pemeriksa | semua OK (di bawah) | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [x] Jika UI: Delivery Gate antislop dijalankan — tanpa teks/kolom/warna/ukuran baru (tab + gate + empty state dari primitives)

## 7. Hasil & Dampak

- Selesai: T-109 DONE. Catatan untuk non-Admin/Manager: submitter Contributor tetap tak melihat kirimannya di tab mana pun — butuh scope baru (keputusan kontrak), dilaporkan tanpa task.
- Belum selesai / sisa: T-107, T-108, T-014 opsional.
- Risiko / utang teknis: tidak ada.
- Dampak ke dokumen desain: tidak ada (FSD §5.4 sudah menuntutnya).

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (T-109 DONE)
- [x] `TRACEABILITY.md` diperbarui (UI-APPROVALS)
- [x] `OPEN-QUESTIONS.md` diperbarui (tidak perlu)
- [x] ADR dibuat/diperbarui (tidak perlu)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| Tinggi | Kerjakan T-107 (utas di TaskDetail) | agen |
| Sedang | Kerjakan T-108 (activity per dokumen) | agen |
