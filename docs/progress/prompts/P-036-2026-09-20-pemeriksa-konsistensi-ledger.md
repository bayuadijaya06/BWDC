# P-036 — 2026-09-20 — Pemeriksa konsistensi ledger + CI

| Field | Isi |
|---|---|
| ID | P-036 |
| Waktu mulai | 2026-09-20 (zona waktu lokal) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | 1 (pemeliharaan infrastruktur dokumentasi) |
| Task terkait | `T-047` (baru); menutup temuan audit **C-058** dan **C-059** |
| Status akhir | DONE |

---

## 1. Prompt User

> "Buat skrip pemeriksa konsistensi ledger (hitungan audit, status task, referensi test/fungsi yang tidak ada lagi) dan jalankan di CI, supaya kelas cacat seperti C-055 tidak terulang."

## 2. Interpretasi & Scope

- **Yang diminta:** satu skrip yang memeriksa ledger progress secara mekanis (hitungan audit, status task, rujukan test/fungsi mati), plus menjalankannya di CI sehingga kelas cacat C-044/C-055/C-057 tertahan sebelum masuk.
- **Yang TIDAK termasuk (out of scope):**
  - Memeriksa kontradiksi dokumen desain yang lebih luas (jumlah endpoint di `42-API.md`, matriks izin, kosakata status) — itu kelas audit lain dan butirnya berbeda-beda per dokumen.
  - Menjadikan pemeriksa ini **required status check** di GitHub: itu setelan repositori, bukan berkas; langkah manualnya dicatat di §7.
  - Memeriksa log historis append-only (`prompts/**`, `SESSION-LOG.md`, `CHANGELOG.md`) — di sana menyebut nama test lama justru **benar**.
- **Asumsi yang diambil:**
  - Sumber kebenaran hitungan test adalah kode (`grep -c '^func Test'`), bukan tabel mana pun.
  - Sumber kebenaran hitungan audit adalah tabel tindak lanjut di laporan audit; angka di prosa dikunci ke sana.
  - Nilai yang tertulis di dokumen yang tidak dapat diperiksa mesin tetap boleh ada, asalkan kalimatnya **menunjuk** ke sumber yang diperiksa (mis. menyebut `scripts/check-ledger.sh`) alih-alih menyalin angka.
- **Pertanyaan yang muncul:** tidak ada pertanyaan baru untuk user. Batasan yang disengaja dicatat di §7.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Recon: cari bentuk data nyata (marker, tabel, sitasi test) dan ukur cacat yang ada sekarang | Daftar wujud nyata + daftar cacat nyata |
| 2 | Tulis `scripts/check-ledger.sh` dengan lima kelompok pemeriksaan | Skrip lolos `bash -n`, keluar 0/1 |
| 3 | Jalankan dan perbaiki cacat yang ditemukan skrip | Ledger yang benar-benar konsisten |
| 4 | Tambah marker `audit-summary` ke lima dokumen | Angka audit terkunci mesin |
| 5 | Buktikan skrip punya gigi (suntikkan cacat, lihat gagal, kembalikan) | Bukti, bukan klaim |
| 6 | Dokumentasikan aturan (protokol §6.1, workflow §8, agent guide §7, AGENTS.md) + CI | Aturan yang dapat ditemukan agen berikutnya |
| 7 | Ledger: task, audit, STATE, CONTINUE, CHANGELOG, SESSION-LOG, log ini | Sesi ditutup rapi |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Mengukur wujud nyata: 57 baris `\| C-` vs 37 judul `### C-`; format marker belum ada; papan `TASKS.md` belum pernah diperiksa | Skrip harus membaca bentuk yang benar-benar ada, bukan bentuk yang dikira ada | Lima kelompok pemeriksaan dengan pola yang terverifikasi |
| 2 | Mengumpulkan daftar test hidup sekali (`path<TAB>TestName`) lalu mencocokkan sitasi dokumen | Rujukan mati persis cacat C-057 (nama diganti, berkas berganti nama) | Menemukan 5 rujukan mati di 3 berkas |
| 3 | Menulis `scripts/check-ledger.sh`; kegagalan diakumulasi ke **berkas**, bukan variabel | Sebagian pemeriksaan berjalan di pipeline (subshell) sehingga penghitung variabel hilang tanpa jejak | Hitungan `FAIL` akurat walau semua pemeriksaan berjalan paralel-pipa |
| 4 | Membuang isi backtick sebelum memecah sel tabel papan | Sel memuat pipe di dalam kode (`GET\|POST /comments`) yang menggeser kolom Selesai → satu temuan palsu pada `T-042` | Temuan palsu hilang; temuan asli tetap |
| 5 | Memeriksa angka audit di prosa lima dokumen (angka sebelum `FIXED`/`OPEN`, dan bentuk penjumlahan `a + b + c = d`) | Marker saja tidak cukup: yang dibaca manusia adalah kalimatnya | Dua baris prosa basi ditemukan dan dikoreksi |
| 6 | Menyuntikkan dua cacat (hitungan suite `236 → 213`, marker `fixed=55 → 54`) lalu menjalankan skrip | Membuktikan skrip menangkap, bukan lulus kosong | Dua `FAIL` dengan sebab spesifik, `exit=1`; lalu dikembalikan dan hijau |
| 7 | Menyuntikkan cacat ketiga di prosa (`57 + 0 + 2 = 59` → `56 + 0 + 2 = 59`) | Membuktikan pemeriksaan prosa juga bergigi | `FAIL AGENTS.md:57: penjumlahan audit di prosa tidak berjumlah` |
| 8 | Memperbaiki cacat nyata yang ditemukan (rujukan test mati, papan ganda, angka prosa basi) | Pemeriksa yang menemukan cacat wajib menutupnya di sesi yang sama | `C-058` dan `C-059` ditutup |
| 9 | Menambah `.github/workflows/ci.yml` (job `ledger` + job `backend`) | "Jalankan di CI": pemeriksa harus jalan tanpa diingat manusia | CI menjalankan kedua skrip dokumen dan suite backend dengan database test terpisah |
| 10 | Menulis aturan di `02-AGENT-PROGRESS-PROTOCOL.md` §6.1 dan menambahkannya ke checklist §8 | Tanpa aturan, skrip akan dilewati sesi berikutnya | Aturan dapat ditemukan dari protokol, workflow, agent guide, dan AGENTS.md |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `scripts/check-ledger.sh` | Added | Pemeriksa konsistensi ledger: hitungan audit (+ marker & prosa), papan kerja, rujukan test/task/temuan, hitungan test `STATE.md` §3 | NFR-MAINT (protokol progress) |
| `.github/workflows/ci.yml` | Added | CI: job `ledger` (dua skrip dokumen) + job `backend` (build, vet, gofmt, `make test` di atas PostgreSQL 16 `bwdcs_test`) | NFR-PORT-02 |
| `docs/design/02-AGENT-PROGRESS-PROTOCOL.md` | Changed | §6.1 baru: angka tidak ditulis dari ingatan, marker, satu task satu kolom, rujukan test hidup; checklist §8 ditambah baris `check-ledger.sh` | — |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Changed | §8: perintah dan arti `check-ledger.sh`, plus catatan CI | — |
| `docs/design/90-AGENT-GUIDE.md` | Changed | §7 quick reference: perintah `check-ledger.sh` | — |
| `AGENTS.md` | Changed | Marker `audit-summary` + status audit diperbarui (59/57), aturan menjalankan pemeriksa | — |
| `CONTINUE.md` | Changed | Marker `audit-summary`, baris audit, snapshot §0 (P-036), arahan resume menyebut pemeriksa | — |
| `docs/progress/STATE.md` | Changed | Marker, tanggal "Terakhir diperbarui", blok "Diperbarui oleh" (P-036), baris audit §1, dua baris §5 diberi penanda historis | — |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | Changed | Marker + penjelasannya, **C-058** dan **C-059** baru (FIXED), prosa §1 diperbarui | — |
| `docs/progress/audits/README.md` | Changed | Marker, hitungan 59/57, rincian `C-038..C-059` | — |
| `docs/progress/TASKS.md` | Changed | `T-044` pindah ke DONE, `T-006` hanya di BLOCKED, kewajiban berulang jadi `T-046`, `T-047` DONE, kutipan nama test lama diberi penanda skip | — |
| `docs/progress/TRACEABILITY.md` | Changed | Tiga rujukan test mati dikoreksi (FR-AUTH-02, FR-AUTH-06, FR-VER-06) | FR-AUTH-02/06, FR-VER-06 |
| `docs/adr/0022-login-attempts-dan-auto-lock.md` | Changed | Dua nama test di §Konsekuensi dikoreksi ke nama yang benar-benar ada | — |
| `docs/progress/CHANGELOG.md` | Changed | Entri sesi P-036 | — |
| `docs/progress/SESSION-LOG.md` | Changed | Entri sesi P-036 | — |
| `docs/progress/prompts/P-036-2026-09-20-pemeriksa-konsistensi-ledger.md` | Added | Log sesi ini | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `bash -n scripts/check-ledger.sh` | `SYNTAX OK` | PASS |
| 2 | `bash scripts/check-ledger.sh` (sebelum perbaikan) | 11 temuan, termasuk `T-044 masih di kolom TODO tetapi teksnya sudah mengklaim selesai`, `T-006/T-008 muncul di lebih dari satu kolom status`, dan 5 rujukan test mati | PASS (skrip menemukan cacat nyata) |
| 3 | `bash scripts/check-ledger.sh` (sesudah perbaikan) | `ledger OK — 0 peringatan, 236 test di backend`, `exit=0` | PASS |
| 4 | Sisipkan cacat hitungan suite & marker → jalankan skrip | `FAIL STATE.md §3: total suite ditulis 213 test, sebenarnya 236` dan `FAIL audits/README.md: marker audit-summary tidak sama dengan laporan audit`; `exit=1`. Dikembalikan → hijau | PASS (punya gigi) |
| 5 | Sisipkan cacat prosa (`56 + 0 + 2 = 59`) → jalankan skrip | `FAIL AGENTS.md:57: penjumlahan audit di prosa tidak berjumlah`; dikembalikan → hijau | PASS (punya gigi) |
| 6 | `bash scripts/check-doc-links.sh` | `BROKEN referensi dokumen: 0` | PASS |
| 7 | `cd backend && make test` | sembilan paket `ok` (bootstrap, config, handler, middleware, migration, model, filestorage, jwt, service); database test `bwdcs_test` | PASS |
| 8 | Hitung ulang angka test dari kode | `grep -cE '^func Test'` per berkas: jwt 17, middleware 12, service auth 22, user 4, handler auth 14, user 3, total 236 — sama dengan `STATE.md` §3 | PASS |

- [x] Typecheck / build dijalankan (`make test` mem-build; tidak ada kode produksi yang berubah)
- [x] Test relevan dijalankan (`make test`, sembilan paket hijau)
- [x] Perubahan dokumen dicek konsisten (`check-doc-links.sh` BROKEN 0, `check-ledger.sh` OK)
- [ ] Delivery Gate antislop — tidak berlaku (tidak ada UI)

## 7. Hasil & Dampak

- **Selesai:** `scripts/check-ledger.sh` dengan lima kelompok pemeriksaan; marker `audit-summary` di lima dokumen; CI dua job; aturan tertulis di protokol §6.1 + workflow §8 + agent guide §7 + `AGENTS.md`; temuan **C-058** (rujukan test mati di `TRACEABILITY.md` dan ADR-0022) dan **C-059** (papan `TASKS.md` bertentangan dengan dirinya sendiri) ditutup.
- **Belum selesai / sisa:** tidak ada sisa pekerjaan di skrip; dua hal bergantung pada setelan repositori, bukan berkas:
  1. jadikan job `ledger` **required status check** di GitHub agar pull request tidak dapat di-merge saat gagal;
  2. tidak ada `.pre-commit`/hook lokal — pemeriksa bergantung pada disiplin sesi + CI.
- **Risiko / utang teknis yang disengaja:**
  - Pemeriksaan **tidak mengenali** dua hal yang tetap bisa drift: (a) *jumlah* temuan audit yang ditulis di prosa (hanya bentuk penjumlahan `a + b + c = d` yang diperiksa) dan (b) hitungan yang bukan test, mis. jumlah endpoint (`T-024`) dan baris matriks. Keduanya butuh kalimat penunjuk ke sumber, bukan mesin.
  - Job `backend` di CI tidak dapat dijalankan di mesin ini (tidak ada runner GitHub Actions); langkah-langkahnya identik dengan yang sudah hijau lokal (`go build`, `go vet`, `gofmt`, `make test`), tetapi belum pernah dieksekusi di runner.
  - Baris historis ditandai manual (`historis`, `saat itu`, `waktu itu`) — penanda itu bisa dipakai untuk menyembunyikan angka basi. Karena itu jumlahnya sedikit dan tercatat di C-059/§6.1.
- **Dampak ke dokumen desain:** `02-AGENT-PROGRESS-PROTOCOL.md`, `12-DEVELOPMENT-WORKFLOW.md`, `90-AGENT-GUIDE.md` (aturan dan perintah baru), `AGENTS.md` (aturan agen). Tidak ada ADR baru: ini bukan keputusan arsitektur produk, melainkan cara verifikasi ledger.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-047` DONE; `T-044`/`T-006`/`T-008`→`T-046` dirapikan)
- [x] `TRACEABILITY.md` diperbarui (FR-AUTH-02/06, FR-VER-06)
- [x] `OPEN-QUESTIONS.md` — tidak ada pertanyaan baru
- [x] ADR — tidak ada keputusan arsitektur produk baru
- [x] Audit diperbarui (C-058, C-059 → `FIXED`; marker 59/57)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Jadikan job `ledger` required status check di GitHub (setelan repositori) | user |
| 1 | Kerjakan modul **Workflow** (`43-WORKFLOW.md`, Phase 2) — satu-satunya modul backend besar yang belum ada | agen |
| 2 | **UI** (`T-006`/`T-007`) menunggu Q-001 (mode antislop) dan Q-002 (`DESIGN.md`) | user |
| 3 | Utang: `T-034` selesai, `T-046` kewajiban berulang; sisa temuan audit hanya C-015 dan C-050 | agen/user |
