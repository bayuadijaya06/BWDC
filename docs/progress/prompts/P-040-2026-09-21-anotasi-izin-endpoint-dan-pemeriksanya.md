# P-040 — 2026-09-21 — Anotasi izin 10 endpoint + pemeriksa kontrak izin (T-024, T-052)

| Field | Isi |
|---|---|
| ID | P-040 |
| Waktu mulai | 2026-09-21 (zona waktu lokal) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | 1 (dokumentasi kontrak + instrumentasi; tidak menyentuh kode produksi) |
| Task terkait | `T-024` (**DONE** — gap 10 endpoint ditutup), `T-052` (**baru**, DONE — pemeriksanya) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Lanjutkan sesuai dengan progress, baca kembali CONTINUE.md dan dokumen progress lain, utamakan penyelesaian gap terlebih dahulu dan bug fixing, jika sudah clear lanjutkan ke tahap berikutnya. Jika seluruh dokumen sudah selesai, segera lanjutkan ke project sesungguhnya, baik frontend maupun backend."

## 2. Interpretasi & Scope

- **Yang diminta:** (1) resume dari `CONTINUE.md` + ledger, (2) tutup gap dan perbaiki bug lebih dulu, (3) sesudah bersih, lanjut ke tahap berikutnya — frontend maupun backend.
- **Yang TIDAK termasuk (out of scope) pada sesi ini:** memulai modul Workflow atau halaman bisnis frontend. Keduanya adalah "tahap berikutnya", tetapi sesi ini memilih menutup **satu** gap yang masih terbuka dan dapat diperiksa mesin lebih dulu (bagian (2) permintaan user) — aturan repo: satu prompt = satu unit yang dapat diverifikasi.
- **Asumsi yang diambil:**
  1. "Gap" berarti apa yang ledger sendiri catat belum selesai. Papan `TASKS.md` hanya memuat **satu** task yang terblokir (lisensi, `T-050`, menunggu Q-023 — milik pemilik) dan **satu** temuan `OPEN` (C-050, keputusan produk). Yang benar-benar **dapat** dikerjakan agen tanpa keputusan siapa pun adalah **`T-024`** — anotasi izin 10 endpoint — sebab keduanya sudah diputuskan di dokumen (matriks ADR-0014).
  2. Verifikasi menyeluruh dijalankan lebih dulu; bila ada yang merah, itu bug dan menang atas pekerjaan baru. Hasilnya hijau (bagian 4 butir 1), jadi tidak ada bug yang perlu diperbaiki sebelum gap.
  3. Memulai modul Workflow atau halaman bisnis **bukan** pekerjaan satu sesi; memilihnya berarti meninggalkan dua hal di atas setengah jalan. Karena itu gap ditutup dulu (permintaan eksplisit user), lalu pilihan tahap berikutnya diserahkan kepada user di akhir sesi.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Jalankan seluruh verifikasi yang ada (backend, frontend, empat pemeriksa dokumen) | Tahu apakah ada bug/gap nyata sebelum menambah pekerjaan |
| 2 | Petakan endpoint tanpa anotasi izin | Daftar yang dapat diverifikasi, bukan dari ingatan |
| 3 | Tulis izin 10 endpoint dari matriks §3.1.2 | Anotasi berasal dari sumber, bukan dikarang |
| 4 | Buat pemeriksa `check-api-contract.sh` (tiga aturan) | Angka T-024 tidak lagi dirawat manual; pasangan karangan tertahan |
| 5 | Buktikan pemeriksa berpunya gigi | Bukti, bukan klaim |
| 6 | Sambungkan ke CI + dokumen aturan | Tidak bergantung ingatan sesi berikutnya |
| 7 | Ledger + laporan, lalu tanyakan tahap berikutnya | Handoff dan keputusan pemilik |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Verifikasi menyeluruh: `go build`/`vet`/`gofmt`, `make test`, `npm run typecheck`/`lint`/`test:run`/`build`, empat pemeriksa dokumen | Permintaan user: gap/bug dulu. Klaim "hijau" di dokumen tidak boleh dipercaya tanpa dijalankan | **Semua hijau** — sembilan paket backend `ok` (236 test), frontend 88 test / 10 berkas, build 336 kB js, `ledger OK`, `BROKEN: 0`, `readme-facts OK`. Tidak ada bug yang menunggu |
| 2 | Memetakan endpoint tanpa baris `Izin:` dari berkasnya (`awk` per blok endpoint) | Angka T-024 (45/55) adalah hitungan manual dan pernah sala | **10 endpoint**: `GET|POST /workflows/definitions`, tiga endpoint notifications, dan lima endpoint admin (`GET|POST /admin/users`, `PATCH /admin/users/:id`, `GET /admin/roles`, `PATCH /admin/settings/:key`) |
| 3 | Menulis izin kesepuluhnya dari matriks §3.1.2, termasuk alasan per pasangan yang tidak jelas (`workflow_definition:manage` menggabungkan create+update; `notification:*` semua role tetapi **hanya baris miliknya**; `setting:read` tidak dipakai route mana pun) | Anotasi harus dapat diperiksa silang ke sumbernya; yang tidak jelas harus dijelaskan supaya tidak "diperbaiki" keliru di sesi berikutnya | `Izin:` bertambah dari 38 → **48 baris**; total **55 endpoint = 48 + 7 lewat tabel §4** |
| 4 | Membuat `scripts/check-api-contract.sh` (110 pemeriksaan) | T-024 dirawat manual selama enam sesi; pasangan izin karangan (`document_version:read`) pernah lolos ke draf §4 (Q-016) | Tiga aturan: endpoint punya izin terbaca; pasangan di anotasi/tabel ada di matriks (44 pasangan); pasangan di `RequirePermission` router (18 pasangan) ada di matriks |
| 5 | Menjalankan pemeriksa itu terhadap kontrak yang baru saja dirapikan | Pemeriksa yang langsung hijau belum membuktikan apa pun kecuali bahwa ia bisa membaca | **1 temuan nyata**: kalimat penjelas "matriks tidak punya `comment:update`" terbaca sebagai klaim pasangan. Kalimatnya ditulis ulang tanpa token pasangan — dan aturan menulisnya dicatat di §6.3 protokol |
| 6 | Menyuntikkan tiga cacat sementara: hapus satu baris `Izin:`; tambahkan `document_version:read` pada anotasi; ubah `task:complete` → `task:finish` di salinan `router.go` | Bukti gigi | Ketiganya `FAIL` dengan nama berkas dan pasangannya; semuanya lalu dipulihkan (`api-contract OK`) |
| 7 | Menyambungkan ke CI (langkah keempat job `ledger`) dan ke dokumen aturan | Pemeriksa yang tidak dijalankan mesin akan terlupakan | `AGENTS.md`, `README.md` §12.3, `12-DEVELOPMENT-WORKFLOW.md` §8, `02-AGENT-PROGRESS-PROTOCOL.md` §6.3 + checklist §8 |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `docs/design/42-API.md` §5/§8/§11 | Changed | 10 baris `Izin:` ditambahkan (workflow definitions, notifications, admin users/roles/settings); satu butir prosa `PATCH /admin/users/:id` diangkat menjadi baris `Izin:`; satu kalimat penjelas `comment:update` ditulis ulang agar tidak terbaca sebagai klaim pasangan | FR-ROLE-03 |
| `scripts/check-api-contract.sh` | Added | Pemeriksa anotasi izin terhadap matriks RBAC dan router (110 pemeriksaan) | — |
| `.github/workflows/ci.yml` | Changed | Langkah keempat job `ledger` + komentar kepala | — |
| `AGENTS.md` | Changed | Baris kewajiban penutup sesi + paragraf bahwa izin endpoint tidak ditulis dari ingatan | — |
| `README.md` §12.3 | Changed | Perintah verifikasi baru | — |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` §8 | Changed | Perintah + paragraf penjelasan | — |
| `docs/design/02-AGENT-PROGRESS-PROTOCOL.md` §6.3/§8 | Changed | Aturan baru: izin endpoint dari matriks, tiga aturan pemeriksa, dan cara menulis kalimat tentang pasangan yang **tidak** ada | — |
| `docs/progress/{TASKS,STATE,CHANGELOG,SESSION-LOG}.md`, `CONTINUE.md` | Changed | `T-024` DONE (dipindah dari TODO), `T-052` DONE, ringkasan sesi, blok snapshot | FR-ROLE-03 |
| `docs/progress/prompts/P-040-2026-09-21-anotasi-izin-endpoint-dan-pemeriksanya.md` | Added | Log sesi ini | — |

## 6. Verifikasi

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `cd backend && go build ./... && go vet ./... && gofmt -l .` | tanpa keluaran | PASS |
| 2 | `cd backend && make test` | sembilan paket `ok` (236 test, coverage 63–84%) | PASS |
| 3 | `cd frontend && npm run typecheck && npm run lint && npm run test:run && npm run build` | bersih, **88 test**, build 336 kB js / 19,5 kB css | PASS |
| 4 | `bash scripts/check-api-contract.sh` | `api-contract OK — 110 pemeriksaan, 55 endpoint` (44 pasangan matriks, 18 pasangan router) | PASS |
| 5 | Tiga cacat disuntikkan sementara (hapus satu `Izin:`; `document_version:read`; `task:finish`) | Ketiganya `FAIL`; sesudah dipulihkan `api-contract OK` | PASS (punya gigi) |
| 6 | `bash scripts/check-ledger.sh` | `ledger OK — 0 peringatan, 236 test di backend` | PASS |
| 7 | `bash scripts/check-doc-links.sh` | `BROKEN referensi dokumen: 0` (562 `PLANNED`, 5 `ABSENT`) | PASS |
| 8 | `bash scripts/check-readme-facts.sh` | `readme-facts OK — 25 fakta diperiksa` | PASS |
| 9 | `ruby -ryaml` atas `.github/workflows/ci.yml` | empat langkah job `ledger` terbaca | PASS |

- [x] Build/typecheck/test dijalankan untuk kedua stack
- [x] Perubahan dokumen dicek konsisten (empat pemeriksa hijau)
- [x] Pemeriksa baru dibuktikan berpunya gigi
- [ ] Bukan pekerjaan UI — Delivery Gate antislop tidak berlaku

## 7. Hasil & Dampak

- **Selesai:** gap `T-024` ditutup (10 endpoint) **dan** mekanisme rawatnya diganti: dari hitungan manual menjadi `scripts/check-api-contract.sh` (`T-052`) yang menahan tiga kelas cacat sekaligus — endpoint tanpa izin, pasangan izin yang tidak ada di matriks, dan route yang memakai pasangan di luar matriks.
- **Belum selesai / sisa:** tahap berikutnya belum dimulai (lihat §9). Satu temuan audit tetap `OPEN` (C-050, keputusan produk) dan satu task tetap `BLOCKED` (T-050, lisensi — menunggu pemilik).
- **Risiko / utang:** pemeriksa ini **tidak** membandingkan judul endpoint di kontrak dengan route di `router.go`, karena endpoint yang belum diimplementasikan (Workflow) memang belum punya route; arah itu akan gagal selamanya sampai modulnya ada. Dicatat sebagai batas di kepala skrip, bukan disembunyikan.
- **Dampak ke dokumen desain:** `44-SECURITY.md` **tidak** diubah — matriksnya tetap satu-satunya sumber dan tidak ada pasangan baru yang ditambahkan. `42-API.md` hanya menerima anotasi yang diturunkan dari matriks itu.
- **Pelajaran:** selama enam sesi, "45/55 endpoint" ditulis ulang dari ingatan dan pernah salah (C-055). Yang mengakhirinya bukan ketelitian tambahan, melainkan memindahkan angka ke mesin — pola yang sama dengan P-036 (ledger) dan P-039 (README).

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-024` dipindah ke DONE, `T-052` baru DONE)
- [x] `TRACEABILITY.md` diperbarui untuk `FR-ROLE-03` (bukti: pemeriksa kontrak izin)
- [x] `OPEN-QUESTIONS.md` — tidak ada pertanyaan baru
- [ ] ADR — tidak ada keputusan arsitektur baru; ADR-0014 tetap sumbernya

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Pilih tahap berikutnya (ditanyakan ke user di akhir sesi): halaman bisnis frontend (Projects) **atau** modul Workflow backend (Phase 2) | pemilik proyek |
| 2 | Putuskan lisensi (Q-023) supaya `T-050` dapat dikerjakan | pemilik proyek |
| 3 | Putuskan threading komentar (Q-019/C-050) | pemilik proyek |
