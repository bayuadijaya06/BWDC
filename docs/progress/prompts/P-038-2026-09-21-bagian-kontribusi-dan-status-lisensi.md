# P-038 — 2026-09-21 — Bagian Kontribusi `README.md` + status lisensi yang ditunda (T-049, T-050, C-061)

| Field | Isi |
|---|---|
| ID | P-038 |
| Waktu mulai | 2026-09-21 (zona waktu lokal) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | pra-4 (dokumentasi repositori; tidak menyentuh fase implementasi) |
| Task terkait | `T-049` (**baru**, DONE — bagian Kontribusi), `T-050` (**baru**, BLOCKED — lisensi, menunggu **Q-023**); menutup temuan audit **C-061** |
| Status akhir | DONE (untuk yang dikerjakan) — satu butir eksplisit **ditunda pemilik**: berkas `LICENSE` |

---

## 1. Prompt User

> "Tambahkan berkas LICENSE dan bagian kontribusi pada README setelah pemilik proyek memutuskan lisensinya."

## 2. Interpretasi & Scope

- **Yang diminta:** (1) berkas `LICENSE`, (2) bagian kontribusi di `README.md` — keduanya **dengan syarat**: "setelah pemilik proyek memutuskan lisensinya".
- **Yang TIDAK termasuk (out of scope):** memilih lisensi atas nama pemilik; mengarang teks lisensi; membuat `CONTRIBUTING.md` terpisah (bagian kontribusi diminta ada **pada README**); mengubah `frontend/package.json` dengan nilai `license` tebakan.
- **Asumsi yang diambil:**
  1. Klausa "setelah pemilik proyek memutuskan lisensinya" adalah **prasyarat**, bukan basa-basi. Karena tidak ada keputusan lisensi di mana pun (dikonfirmasi lewat pemindaian `grep`), prasyarat itu **belum terpenuhi**, sehingga bertanya lebih dulu adalah satu-satunya jalur — `CONTINUE.md` §1 butir 4 melarang menebak keputusan user.
  2. Bagian kontribusi masuk README (sesuai kalimat permintaan), dan isinya **menunjuk** ke sumber aturan yang sudah ada (`AGENTS.md`, `01-AGENT-WORKFRAME`, `02-AGENT-PROGRESS-PROTOCOL`, `12-DEVELOPMENT-WORKFLOW`) alih-alih menyalin teksnya — pola yang sama dengan perbaikan **C-014** (satu sumber kebenaran). Aturan kontribusi sendiri **tidak** bergantung pada lisensi, jadi bagian ini dikerjakan tanpa menunggu keputusan.
- **Pertanyaan yang muncul (link ke `OPEN-QUESTIONS.md`):** **Q-023** (BARU) — jenis lisensi + apakah kontribusi pihak ketiga diterima. Ditandai **BLOCKING untuk distribusi, NON-BLOCKING untuk pengembangan**.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Pastikan status lisensi tidak pernah ditetapkan (bukan hanya "tidak saya temukan") | Dasar untuk bertanya, bukan menebak |
| 2 | Tanyakan keputusan lisensi ke pemilik lewat pilihan berganda | Keputusan tercatat, bukan diasumsikan |
| 3 | Kenali pemegang hak cipta | Nama yang akan ditulis di `LICENSE` kelak |
| 4 | Sapu klaim basi di `README.md` sebelum menambah bagian baru ke berkas itu | Tidak menumpuk bagian baru di atas klaim lama yang salah |
| 5 | Tulis §12 Kontribusi (enam subbagian) | Aturan kontribusi yang dapat diikuti agen **dan** manusia |
| 6 | Tulis §13 Lisensi sebagai tabel **status**, bukan teks lisensi | Niat "belum ditetapkan" terbaca, bukan hilang |
| 7 | Catat Q-023 + `T-050` + temuan **C-061** + ledger | Handoff jujur untuk sesi berikutnya |
| 8 | Verifikasi: kedua skrip dokumen + seluruh suite backend/frontend | Bukti, bukan klaim |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Pemindaian lisensi: `grep -rn "lisensi\|license\|LICENSE\|MIT\|Apache\|GPL"` atas `README.md`, `AGENTS.md`, `docs/`, `frontend/package.json` | Memastikan statusnya memang belum ada, bukan tidak ditemukan karena malas mencari | Satu-satunya kemunculan adalah **placeholder** di `README.md` §12 lama: "Belum ada berkas lisensi di repositori ini... **Jangan menganggapnya open source**". `frontend/package.json` tanpa field `license`; tidak ada `LICENSE`; tidak ada `CONTRIBUTING.md` |
| 2 | Tanyakan ke pemilik: keputusan lisensi (empat opsi) + pemegang hak cipta | Lisensi adalah keputusan hukum pemilik repo, bukan keputusan agen | Jawaban pemilik: **tunda** keputusan lisensi; pemegang hak cipta **BSA** |
| 3 | Sapu klaim basi `README.md` sebelum menulis bagian baru | Berkas yang paling sering dibaca orang luar juga yang paling mudah tertinggal | Enam klaim basi ketahuan (lihat **C-061**) |
| 4 | Hitung ulang jumlah route dari kode, bukan dari dokumen | Angka yang disalin dari ledger bisa mengulang kelas C-044/C-055 | `internal/handler/router.go` → **32** rute (1 health, 5 auth, 1 admin, 8 project, 7 document, 5 task, 5 comment) |
| 5 | Tambah §12 Kontribusi (12.1 persiapan, 12.2 alur satu perubahan, 12.3 verifikasi wajib, 12.4 sepuluh aturan + sumbernya, 12.5 konvensi commit, 12.6 kanal usulan) | Permintaan user; tabel aturan menunjuk sumber, bukan menyalinnya | §12 ada, 88 baris |
| 6 | Tambah §13 Lisensi sebagai tabel status + peringatan | Bila lisensi tidak ditulis, ketiadaannya harus terlihat — bukan hilang diam-diam | Status "belum ditetapkan", pemegang hak cipta **BSA**, field `package.json` belum diisi, rujukan Q-023 |
| 7 | Tandai ketiadaan `LICENSE` di pohon folder `README.md` §4 | Pohon yang tidak menyebut ketiadaannya menyiratkan kelalaian, bukan keputusan | Baris `LICENSE  # belum ada — lihat §13` |
| 8 | Catat **Q-023**, task **`T-049`** (DONE) dan **`T-050`** (BLOCKED), temuan **C-061** | Kewajiban progress protocol | Semuanya masuk ledger |
| 9 | Perbaiki `scripts/check-doc-links.sh` agar dokumen yang **sengaja tidak dibuat** tidak dihitung tautan rusak | Penyebutan `CONTRIBUTING.md` — yang memang tidak ada — membuat skrip gagal; memperbaiki dokumennya saja akan melahirkan larangan menulis nama berkas yang sah | Daftar `ABSENT_DOCS` + klasifikasi `ABSENT`, dengan **gigi tetap tajam**: dokumen hilang yang tidak terdaftar masih `BROKEN` (diuji dengan menyisipkan berkas `.md` sementara di `docs/design/` yang jelas tidak ada) |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `README.md` | Changed | §12 **Kontribusi** (baru, enam subbagian); §13 **Lisensi** (dari §12 lama, kini tabel status); §1/§3/§4 status dan pohon diselaraskan; jumlah route 30 → 32 | — (dokumentasi repo) |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | **Q-023** (baru): lisensi + kebijakan kontribusi pihak ketiga | — |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | Changed | **C-061** (baru): enam klaim basi README; marker audit 60 → **61** temuan | — |
| `docs/progress/audits/README.md` | Changed | Baris audit + jumlah FIXED/OPEN diselaraskan ke marker | — |
| `docs/progress/TASKS.md` | Changed | `T-049` DONE; `T-050` BLOCKED | — |
| `docs/progress/STATE.md`, `CONTINUE.md`, `AGENTS.md`, `docs/progress/CHANGELOG.md`, `docs/progress/SESSION-LOG.md` | Changed | Ledger diselaraskan ke kondisi sesudah sesi | — |
| `scripts/check-doc-links.sh` | Changed | Klasifikasi baru **`ABSENT`** untuk dokumen yang sengaja tidak dibuat (`ABSENT_DOCS`, kini hanya `CONTRIBUTING.md`) | — |
| `docs/progress/prompts/P-038-2026-09-21-bagian-kontribusi-dan-status-lisensi.md` | Added | Log sesi ini | — |

**Tidak dibuat:** `LICENSE` — sengaja, karena pemilik menunda keputusannya (Q-023). Ini satu-satunya bagian permintaan user yang tidak dikerjakan, dan alasannya tercatat, bukan disenyapkan.

## 6. Verifikasi

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `bash scripts/check-ledger.sh` | `ledger OK` | PASS |
| 2 | `bash scripts/check-doc-links.sh` | `BROKEN referensi dokumen: 0` | PASS |
| 3 | `grep -cE '^\| C-' docs/progress/audits/AUDIT-001-...md` vs marker | 61 baris = `total=61` | PASS |
| 4 | `grep -n "router\.\(GET\|POST\|PATCH\|DELETE\)" backend/internal/handler/router.go` | 32 rute | PASS (angka di README berasal dari kode) |
| 5 | `bash scripts/check-doc-links.sh` sesudah menyisipkan berkas `.md` sementara di `docs/design/` yang jelas tidak ada | Berkas uji itu tetap dilaporkan `BROKEN`, sementara `CONTRIBUTING.md` dilaporkan `ABSENT` | PASS — klasifikasi baru tidak melonggarkan pemeriksaan |
| 6 | `bash -n scripts/check-doc-links.sh` | `syntax OK` | PASS |

- [x] Perubahan dokumen dicek konsisten (referensi berkas ada) — kedua skrip dokumen hijau
- [x] Tidak ada perubahan kode, sehingga typecheck/test backend & frontend **tidak** terpengaruh (tidak dijalankan ulang; tidak ada berkas kode yang tersentuh sesi ini)
- [ ] Bukan pekerjaan UI — Delivery Gate antislop tidak berlaku

## 7. Hasil & Dampak

- **Selesai:** §12 Kontribusi; §13 Lisensi sebagai status; **C-061** ditutup (enam klaim basi README, termasuk `frontend/` yang masih disebut kosong dan `DESIGN.md` yang masih disebut placeholder); **Q-023** dan **T-050** dicatat.
- **Belum selesai / sisa:** **`T-050` — berkas `LICENSE`.** Menunggu pemilik memilih jenis lisensi dan menentukan apakah kontribusi pihak ketiga diterima. Tidak ada pekerjaan lain yang tersisa dari permintaan ini.
- **Risiko / utang:** tanpa lisensi, kontribusi pihak ketiga **tidak punya dasar hak** — bagian §12.6 menulisnya terus terang alih-alih menyembunyikannya di balik "hubungi maintainer". Selama itu tidak diputuskan, ini bukan utang teknis, melainkan keputusan yang memang tertunda.
- **Dampak ke dokumen desain:** tidak ada dokumen desain yang berubah. `README.md` bukan dokumen desain (tidak terdaftar di `00-README.md`), dan itulah akar **C-061** — kelas cacat yang sama dengan **C-060**, yaitu berkas yang tidak masuk daftar "dokumen yang diselaraskan" sehingga pembersihan melewatkannya.
- **Pelajaran yang dicatat sebagai aturan:** bila sebuah klaim muncul di beberapa dokumen, perbaikan harus **menyapu** semua pemuatnya (`grep` atas klaimnya), bukan mengandalkan daftar berkas dari sesi sebelumnya.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-049` DONE, `T-050` BLOCKED)
- [ ] `TRACEABILITY.md` — tidak disentuh: sesi ini tidak menyentuh requirement produk
- [x] `OPEN-QUESTIONS.md` diperbarui (**Q-023**)
- [ ] ADR — tidak ada keputusan arsitektur; menunda lisensi bukan keputusan teknis yang butuh ADR

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Pilih jenis lisensi (proprietary *all rights reserved*, MIT, atau Apache-2.0) dan tentukan apakah kontribusi pihak ketiga diterima — sesudahnya `T-050` dapat langsung dikerjakan | **pemilik proyek** |
| 2 | Halaman bisnis frontend (Projects lebih dulu) memakai endpoint yang sudah hidup — jalur terdekat untuk memanfaatkan kerangka P-037 | agen |
| 3 | Modul **Workflow** (Phase 2, `43-WORKFLOW.md`, ADR-0015/0016 `ACCEPTED`) — satu-satunya fase yang belum disentuh | agen |
| 4 | Jadikan job `ledger` di CI sebagai *required status check* GitHub | pemilik proyek (setelan repo) |
