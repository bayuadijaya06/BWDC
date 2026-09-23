# P-046 — 2026-09-22 — Halaman Tasks di frontend dan probe server nyata 41 langkah

**Sesi:** P-046 · **Model/agen:** deepseek/deepseek-v4-flash (lanjutan worktree yang sama) · **Status:** selesai
**Task:** `T-062` — DONE · **Temuan baru:** tidak ada (seluruh perilaku yang diklaim halaman terbukti di server nyata)

---

## 1. Prompt User

1. "Bangun halaman Tasks di frontend dengan pola yang sudah terbukti di Projects dan Documents, termasuk penyaring tri-state overdue dan transisi statusnya."
2. (sesi sebelumnya, terputus) "Silakan lanjutan kembali, jangan lupa untuk selalu cross check dengan dokumen desain, dan dokumentasikan segala bentuk progress, gap dan temuan yang ada."

## 2. Interpretasi & Scope

Yang diminta: halaman bisnis ketiga di frontend mengikuti pola yang dua kali terbukti (Projects P-041, Documents P-044) — lapisan data `services` + `queries` dengan kunci terpusat, halaman daftar ber-penyaring yang hidup di URL, halaman detail dengan aksi transisi, dialog buat, test di setiap lapis, dan bukti di server nyata. Dua hal disebut eksplisit karena keduanya pernah menjadi sumber temuan: **penyaring tri-state overdue** (`42-API.md` §6 membedakan "tidak dikirim" dari `false`) dan **transisi status** sesuai tabel `42-API.md` §6 (Start/Complete/Reopen memakai endpoint dan izin yang berbeda — bukan satu tombol ubah-status).

**Di luar scope:** form ubah task (PATCH lengkap), komentar di detail task, activity log, dan modul Workflow — semuanya masih di antrean papan dan dinyatakan "belum dibangun" di layar beserta alasannya.

## 3. Rencana

1. Verifikasi keadaan disk: mayoritas berkas Tasks sudah tertulis sesi sebelumnya yang terputus; baca ulang, jangan karang ulang.
2. Jalankan verifikasi penuh frontend: typecheck, lint, test, build.
3. Jalankan probe HTTP nyata (`scripts/probe-task-module.py`, sudah ditulis sesi sebelumnya) terhadap server yang hidup di 8081.
4. Perbaiki apa pun yang gagal pada langkah 2–3.
5. Tulis ledger P-046 dan jalankan lima pemeriksa.

## 4. Aksi yang Dilakukan

Sesi dibuka dengan **memverifikasi kondisi disk, bukan mengulang pekerjaan**: seluruh berkas Tasks (`services/tasks.ts`, `queries/tasks.ts`, `pages/Tasks/{index,CreateTaskDialog,TaskDetail}.tsx`, tiga berkas test, `scripts/probe-task-module.py`) sudah ada dengan waktu modifikasi sesi terputus. Dibaca ulang kedua halaman utama dan lapisan kuerinya, lalu dikonfirmasi terhadap dokumen desain:

- **Tri-state overdue** diterapkan apa adanya: kontrol bernilai `"" | "true" | "false"` dengan tiga pilihan yang menerangkan ketiga keadaannya ("Tanpa penyaring overdue" / "Hanya yang overdue" / "Hanya yang belum overdue"), dan halaman detail menerangkan bahwa `false` memuat task tanpa tenggat sedangkan rentang tenggat tidak.
- **Tabel transisi** dirender di halaman detail (aksi → transisi → endpoint → izin), dan tombolnya dipilih menurut status berjalan — `Complete` tidak pernah ditawarkan pada task Open karena server membalas `409`.
- **Penanggung jawab** dipilih dari anggota project (batas C-063 dinyatakan di layar, bukan ditutupi).

Verifikasi frontend: `tsc --noEmit` dan `eslint .` bersih; **234 test / 24 berkas** hijau (naik dari 188/21); `vite build` 455,8 kB js / 23,5 kB css. Test mengunci perilaku yang diminta: `Tasks.test.tsx` punya test eksplisit "membedakan tiga keadaan penyaring overdue, bukan dua" yang memeriksa nilai kueri yang dikirim (`""` → `"false"` → `"true"` → `""`), dan `TaskDetail.test.tsx` menguji ketiga transisi, ketidakhadiran tombol yang tidak sah menurut status, pesan izin, dan penjelasan `409`.

**Probe HTTP nyata: 41/41 PASS** terhadap server yang hidup (dibangun dari kode sesi ini, `versi_skema 11`), dengan aktor kedua ber-role viewer. Seluruh klaim halaman kini punya bukti di sisi server — termasuk dua yang paling berisiko karangan: tri-state overdue dan tabel transisi. Database dev dikembalikan ke baseline oleh probe itu sendiri.

Tidak ada temuan baru: berbeda dari P-045, tidak ada satu pun langkah probe yang gagal, dan tidak ada klaim dokumen yang perlu ditepati.

## 5. File yang Berubah

### Added

| Berkas | Isi | Prompt |
|---|---|---|
| `frontend/src/services/tasks.ts` | Lapisan layanan modul task: `listTasks`, `fetchTask`, `createTask`, `updateTask` (PATCH), `completeTask` (endpoint sendiri), kosakata prioritas + label, `validateDueRange` (batas inklusif, rentang terbalik ditolak di klien) | P-045/P-046 |
| `frontend/src/queries/tasks.ts` | Kunci kueri terpusat + hook; mutasi detail **diambil ulang** karena `is_overdue`/`updated_at` dihitung server | P-045/P-046 |
| `frontend/src/pages/Tasks/index.tsx` | Task List §6.1: sub-halaman sebagai penyaring yang dapat dibagikan, penyaring tri-state overdue, rentang tenggat RFC 3339, paginasi dari `meta` | P-045/P-046 |
| `frontend/src/pages/Tasks/TaskDetail.tsx` | Task Detail §6.3: aksi transisi menurut status berjalan, tabel transisi endpoint+izin, bagian "belum dibangun" beserta alasannya | P-045/P-046 |
| `frontend/src/pages/Tasks/CreateTaskDialog.tsx` | Dialog buat task: project → anggota + dokumen project sebagai sumber pilihan | P-045/P-046 |
| `frontend/src/services/tasks.test.ts`, `frontend/src/pages/Tasks/Tasks.test.tsx`, `frontend/src/pages/Tasks/TaskDetail.test.tsx` | Test lapisan layanan, halaman daftar (termasuk tiga keadaan penyaring overdue), dan halaman detail (ketiga transisi + `409`) | P-045/P-046 |
| `scripts/probe-task-module.py` | Sesi probe HTTP nyata modul task (16 kelompok langkah, 41 asersi, aktor kedua viewer, pembersihan baseline otomatis) | P-045/P-046 |
| `docs/progress/prompts/P-046-2026-09-22-halaman-tasks-dan-probe-41-pass.md` | Log ini | P-046 |

### Changed

| Berkas | Perubahan | Prompt |
|---|---|---|
| `frontend/src/config/navigation.ts` | `Tasks` menjadi `ready` | P-045/P-046 |
| `frontend/src/App.tsx` | Rute `/tasks` + `/tasks/:id` terpasang | P-045/P-046 |
| `frontend/src/queries/projects.ts`, `frontend/src/queries/documents.ts` | Hook pendukung dialog buat task (anggota project, dokumen project) | P-045/P-046 |
| `frontend/src/components/layout/Sidebar.tsx`, `frontend/src/components/layout/AppShell.test.tsx` | Penyesuaian laci menu kecil; test yang menguncinya | P-045/P-046 |
| `frontend/src/App.test.tsx`, `README.md` | Rute & status halaman diperbarui | P-045/P-046 |
| `docs/progress/TASKS.md` | Baris `DONE` **`T-062`** | P-046 |
| `docs/progress/STATE.md`, `CONTINUE.md`, `AGENTS.md` | Paragraf sesi P-046, angka test frontend, keadaan halaman | P-046 |
| `docs/progress/CHANGELOG.md`, `docs/progress/SESSION-LOG.md` | Entri P-045/P-046 | P-045/P-046 |
| `docs/progress/TRACEABILITY.md` | Baris UI halaman Tasks terhadap FR modul task | P-046 |
| `docs/design/70-TESTING.md` | Bukti probe §6 (41 asersi) dan test halaman Tasks | P-046 |

## 6. Verifikasi (WAJIB)

**Frontend:** `npm run typecheck` bersih, `npm run lint` bersih, `npx vitest run` **234 test / 24 berkas** hijau, `npm run build` → 455,80 kB js / 23,52 kB css.

**Probe HTTP nyata** (server hidup di 8081, `versi_skema 11`, aktor kedua viewer) — **41/41 PASS**, ringkasan kelompok:

```
1-5.  login admin + /auth/me + project + 2 user uji (contributor, viewer) + anggota + dokumen PROBE062-001
6-7.  4 task (A overdue, B tidak, C overdue, D tanpa tenggat); task tanpa tenggat tidak pernah overdue
8.    tri-state overdue: tanpa penyaring total=4 | ?overdue=true total=2 (A,C) | ?overdue=false total=2 (B,D — tanpa tenggat ikut)
      | nilai di luar true/false -> 422 field=overdue
9.    rentang inklusif: kedua batas termasuk | due_to==due_from sah | hanya due_to / hanya due_from | terbalik -> 422 field=due_to
10.   halaman di luar rentang: baris=0 total=4 (tambalan C-048/T-043 terbukti)
11.   prioritas + penanggung jawab + status di luar kosakata -> 422
12.   Complete pada Open -> 409 | Start open->in_progress (PATCH) | Complete in_progress->completed (endpoint sendiri)
      | Complete kedua idempoten 200 | Reopen completed->open | PATCH status=completed -> 409 | pindah project -> 409
      | PATCH tanpa field -> 422 | POST dengan status -> 422 (task selalu lahir open)
13.   penanda overdue hilang saat selesai (True -> False)
14.   viewer non-anggota: total=0 | detail 404 | menulis 403 | sesudah jadi anggota 200
15.   jejak audit: TASK_CREATED x4, TASK_UPDATED x3, TASK_COMPLETED x2, dst.
16.   baseline pulih: users 1, projects 0, documents 0, tasks 0, audit_logs 43, login_attempts 0, skema 11
```

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan (234 frontend; backend tidak tersentuh sesi ini)
- [x] Perubahan dokumen dicek konsisten (lima pemeriksa, lihat §8)
- [x] Delivery Gate antislop (`01-AGENT-WORKFRAME.md` §5.3): tanpa gradient/glow/ikon generik/font CDN/em dash di string UI/warna langsung; dial tetap ENERGY 1 / RHYTHM 2 / MOTION 1; tidak ada angka karangan — seluruh angka di log ini dari perintah yang dijalankan sesi ini.

## 7. Hasil & Dampak

- **Selesai:** halaman bisnis ketiga berdiri penuh dengan pola yang sama dengan Projects dan Documents; seluruh perilaku yang diklaim halaman terbukti di server nyata (41/41).
- **Sisa:** form ubah task (PATCH lengkap), komentar, activity log — dinyatakan "belum dibangun" beserta alasannya di halaman detail.
- **Risiko / utang teknis:** tidak ada baru. Batas lama tetap: C-063 (pemilih pengguna, Q-024), C-050 (threading komentar, Q-019), T-050 (lisensi, Q-023), Q-025 (em dash prosa lama).
- **Dampak ke dokumen desain:** tidak ada — kontrak `42-API.md` §6 sudah benar; halaman mengikutinya apa adanya.

## 8. Update Ledger (Checklist Wajib)

- [x] `docs/progress/prompts/P-046-…md` (log ini)
- [x] `docs/progress/TASKS.md` — `T-062` di `DONE`
- [x] `docs/progress/STATE.md` — paragraf sesi, angka test frontend
- [x] `docs/progress/CHANGELOG.md` — entri P-046
- [x] `docs/progress/SESSION-LOG.md` — entri P-046
- [x] `docs/progress/TRACEABILITY.md` — baris UI halaman Tasks
- [x] `docs/design/70-TESTING.md` — bukti probe
- [x] `CONTINUE.md` (root) — blok §0 + pointer `P-047`
- [x] `AGENTS.md` — keadaan halaman frontend
- [x] `OPEN-QUESTIONS.md` — tidak ada pertanyaan baru
- [x] ADR — tidak ada keputusan arsitektur baru
- [x] Lima pemeriksa dijalankan dan hijau

## 9. Next Action

**Approvals** (`50-FSD.md` §9) adalah halaman berikutnya yang polanya sama, tetapi ia memakai `GET /workflows/instances` — dan itu berarti **modul Workflow** (`43-WORKFLOW.md`, Phase 2, ADR-0015/ADR-0016 `ACCEPTED`) harus dibangun lebih dulu di backend; sampai hari ini itu satu-satunya fase backend yang belum disentuh. Jalur alternatif yang tidak menunggu apa pun: menyambungkan `scripts/responsive-evidence.mjs` ke halaman Tasks/Documents agar klaim tata letaknya juga terukur, atau menyelesaikan Q-025.
