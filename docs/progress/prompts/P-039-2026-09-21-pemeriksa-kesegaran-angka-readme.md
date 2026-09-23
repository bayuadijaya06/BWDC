# P-039 — 2026-09-21 — Pemeriksa kesegaran angka & versi `README.md` (T-051, C-062)

| Field | Isi |
|---|---|
| ID | P-039 |
| Waktu mulai | 2026-09-21 (zona waktu lokal) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | pra-4 (instrumentasi dokumen; tidak menyentuh kode produksi) |
| Task terkait | `T-051` (**baru**, DONE); menemukan sekaligus menutup temuan audit **C-062** |
| Status akhir | DONE |

---

## 1. Prompt User

> "Buat skrip yang memeriksa kesegaran angka di README (jumlah route, versi dependensi) terhadap repo agar tidak diam-diam basi."

## 2. Interpretasi & Scope

- **Yang diminta:** sebuah pemeriksa yang membandingkan **angka/versi di `README.md`** dengan **repo**, supaya tidak basi tanpa terlihat. Dua contoh yang disebut user: jumlah route dan versi dependensi.
- **Yang TIDAK termasuk (out of scope):** menulis ulang angka README dengan tangan sebagai satu-satunya tindakan (itu perbaikan sesaat, bukan pencegahan); memindai **seluruh** dokumen (ADR-0002 dan `30-ARCHITECTURE.md` §2.1 **sengaja** menyebut "React 18" sebagai riwayat yang dijelaskan ADR-0024 — memindai semuanya akan menandai sejarah yang benar sebagai basi); mengubah kode produksi.
- **Asumsi yang diambil:**
  1. "Terhadap repo" berarti **sumber kebenaran dihitung**, bukan dokumen lain dibandingkan dengan dokumen lain — `router.go`, `go.mod`, `package.json`, dan berkas migrasi. Membandingkan dokumen dengan dokumen hanya memindahkan masalah.
  2. Pemeriksa yang **berhenti memeriksa tanpa suara** lebih berbahaya daripada pemeriksa yang berisik: klaim yang polanya hilang dari README dianggap **gagal**, bukan dilewati.
  3. Klasifikasi agen-untuk-setiap-group-route diperlukan; tanpa itu sebuah group baru menambah total tanpa masuk rincian dan README akan tampak cocok padahal tidak.
- **Pertanyaan yang muncul (link ke `OPEN-QUESTIONS.md`):** tidak ada pertanyaan baru — pekerjaan ini adalah instrumentasi, bukan keputusan produk.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Recon: klaim angka/versi apa saja yang ada di README, dan dari mana masing-masing dapat dihitung | Daftar fakta + sumbernya, bukan tebakan |
| 2 | Tulis `scripts/check-readme-facts.sh` dengan pola klaim ber-anchor dan sumber kebenaran eksplisit | Pemeriksa yang dapat dibaca agen, bukan regex gelap |
| 3 | Tambahkan pagar: setiap group route wajib punya ember; klaim yang hilang = gagal | Pemeriksa tidak bisa "lulus" karena berhenti memeriksa |
| 4 | Jalankan terhadap README yang ada | Mengetahui apakah README sekarang jujur |
| 5 | Perbaiki apa pun yang ditemukan | README jujur **dan** C-062 tercatat |
| 6 | Buktikan berpunya gigi: suntik cacat sementara, lihat FAIL, pulihkan | Bukti, bukan klaim |
| 7 | Sambungkan ke CI + dokumen aturan (README §12.3, 12-DEVELOPMENT-WORKFLOW §8, 02-PROGRESS §6.2/§8, 90-AGENT-GUIDE, AGENTS.md) | Tidak bergantung ingatan siapa pun |
| 8 | Ledger: C-062, `T-051`, marker 61 → 62, CHANGELOG, SESSION-LOG, STATE, CONTINUE | Handoff untuk sesi berikutnya |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Menghitung route per "ember" dari `router.go` berdasarkan variabel penerima (`r`, `auth`, `projects`, `documents`, `tasks`, `comments`, `admin`) | Rincian README ("1 health, 5 auth, …") harus punya sumber tunggal; menghitung dengan `grep` atas pola path akan rapuh terhadap rename group | `32` route = 1+5+1+8+7+5+5, cocok dengan README |
| 2 | Menambah pagar **ember wajib**: setiap penerima route harus ada di daftar skrip | Group baru (`notifications`) akan menambah total tanpa masuk rincian — dan README akan tampak benar | Dua kali gagal bila group baru muncul (belum terklasifikasi **dan** rincian ≠ total) |
| 3 | Menambah pagar **klaim hilang = gagal** | Pemeriksa yang berhenti memeriksa tanpa suara adalah cacat yang tidak terlihat | Setiap fakta yang polanya lenyap dari README memunculkan `FAIL` dengan perintah perbarui skripnya |
| 4 | Menjalankan pemeriksa terhadap README | Mengetahui keadaan sebenarnya sebelum mengklaim apa pun | **3 temuan**: dua bug di skrip sendiri (penggabungan receiver, ekstraksi versi pgx) dan **satu cacat nyata di README** — baris §2 "Frontend (rencana) … React 18 … Belum diinisialisasi" |
| 5 | Memperbaiki README §2 dan menambah pemeriksaan versi TypeScript | Baris itu menyebut versi **dan** status yang salah; menambahkan versi TypeScript ke baris berarti klaimnya ikut diperiksa | Baris menjadi `React 19 + TypeScript 5.9 + Vite 8 + Tailwind v4`; `readme-facts OK — 25 fakta` |
| 6 | Menyuntikkan lima cacat sementara ke README, menjalankan, memulihkan | Pemeriksa tanpa bukti gigi hanya menambah kepercayaan palsu | Lima-limanya `FAIL` dengan nomor baris; README dipulihkan dan kembali `OK` |
| 7 | Menguji pagar ember route dengan **salinan** `router.go` (+1 group) dan salinan skrip | Menguji tanpa menyentuh berkas proyek — tidak ada risiko meninggalkan kerusakan di kode | Kedua pagar menyala; berkas probe dihapus |
| 8 | Menyambungkan ke CI dan lima dokumen aturan | Aturan yang tidak dijalankan mesin akan terlupakan pada sesi berikutnya | `.github/workflows/ci.yml` + §6.2 baru di protokol progress |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `scripts/check-readme-facts.sh` | Added | Pemeriksa 25 fakta README terhadap repo; exit 0/1; batas dan alasannya ditulis di kepala berkas | — |
| `README.md` §2/§12.3 | Changed | Baris "Frontend (rencana)…" diganti keadaan sebenarnya (**C-062**); perintah verifikasi menambah pemeriksa baru | — |
| `.github/workflows/ci.yml` | Changed | Langkah ketiga di job `ledger` + komentar kepala yang diperluas | — |
| `AGENTS.md` | Changed | Baris "sebelum menutup sesi" + paragraf yang menjelaskan pemeriksa README dan larangan melunakkannya | — |
| `docs/design/02-AGENT-PROGRESS-PROTOCOL.md` §6.2/§8 | Changed | Aturan baru §6.2 (tabel fakta → sumber) + butir checklist | — |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` §8 | Changed | Perintah + paragraf penjelasan | — |
| `docs/design/90-AGENT-GUIDE.md` | Changed | Perintah pada blok verifikasi | — |
| `docs/progress/README.md` | Changed | Butir 7 menunjuk pemeriksa kelas serupa di luar ledger | — |
| `docs/progress/audits/AUDIT-001-...md`, `audits/README.md`, `TASKS.md`, `STATE.md`, `CONTINUE.md`, `CHANGELOG.md`, `SESSION-LOG.md` | Changed | C-062 FIXED, marker 62 temuan, `T-051` DONE, ringkasan sesi | — |
| `docs/progress/prompts/P-039-2026-09-21-pemeriksa-kesegaran-angka-readme.md` | Added | Log sesi ini | — |

## 6. Verifikasi

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `bash scripts/check-readme-facts.sh` | `readme-facts OK — 25 fakta diperiksa` | PASS |
| 2 | `bash -n scripts/check-readme-facts.sh` | `syntax OK` | PASS |
| 3 | Lima cacat disuntikkan sementara ke `README.md` (route 32→31; Project 8→9 endpoint; React 19→18; migrasi `010`→`009`; kalimat "frontend/ masih kosong") | Kelimanya `FAIL` dengan nomor baris; setelah dipulihkan kembali `OK — 25 fakta` | PASS (punya gigi) |
| 4 | Salinan sementara `router.go` +1 group `notifications`, dijalankan dengan salinan skrip | `FAIL … belum terklasifikasi (notifications)` **dan** `FAIL … rincian route berjumlah 32, total tercatat 33` | PASS (pagar ember bekerja) |
| 5 | `bash scripts/check-ledger.sh` | `ledger OK — 0 peringatan, 236 test di backend` | PASS |
| 6 | `bash scripts/check-doc-links.sh` | `BROKEN referensi dokumen: 0` | PASS |

- [x] Pemeriksa dijalankan (bukan hanya ditulis) dan bukti gigi direkam
- [x] Tidak ada berkas kode produksi yang disentuh — suite backend/frontend tidak terpengaruh
- [x] Perubahan dokumen dicek konsisten: `check-doc-links.sh` + `check-ledger.sh` hijau
- [ ] Bukan pekerjaan UI — Delivery Gate antislop tidak berlaku

## 7. Hasil & Dampak

- **Selesai:** `scripts/check-readme-facts.sh` (25 fakta, tersambung ke CI), **C-062** ditutup, aturan §6.2 ditambahkan ke protokol progress, dan lima dokumen aturan menunjuk perintah barunya.
- **Belum selesai / sisa:** fakta yang belum dapat diperiksa karena tidak ber-pola tetap (mis. klaim dalam prosa bebas) dan dokumen **lain** yang memuat versi sama — keduanya dicatat sebagai batas di kepala skrip, bukan disembunyikan.
- **Risiko / utang:** pemeriksa yang terlalu spesifik bisa "mati" ketika README ditulis ulang; itulah sebabnya klaim yang hilang dijadikan **gagal**, bukan dilewati. Risiko sebaliknya (pemeriksa terlalu longgar) dijaga pagar ember route.
- **Dampak ke dokumen desain:** `02-AGENT-PROGRESS-PROTOCOL.md` mendapat §6.2 baru; `12-DEVELOPMENT-WORKFLOW.md` §8, `90-AGENT-GUIDE.md`, dan `AGENTS.md` menambah perintahnya. `README.md` diperbaiki di baris yang salah.
- **Pelajaran:** C-060 → C-061 → C-062 adalah **rantai yang sama**: klaim yang dapat dihitung dari kode, dijaga proses manual. C-061 memperbaiki gejalanya dengan `grep` yang lebih luas; C-062 membuktikan bahwa pendekatan itu tidak cukup, karena cacatnya tidak memuat kata kunci yang dicari. Yang menutup rantai ini adalah **pemeriksa yang berjalan sendiri**, bukan semangat untuk lebih teliti.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui (marker audit, ringkasan sesi, tabel file)
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-051` DONE)
- [ ] `TRACEABILITY.md` — tidak disentuh: tidak ada requirement produk yang berubah
- [x] `OPEN-QUESTIONS.md` — tidak ada pertanyaan baru (diperiksa, tidak diubah)
- [ ] ADR — tidak ada keputusan arsitektur; ini tooling dokumen yang sumbernya sudah ditetapkan di protokol

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Halaman bisnis frontend (Projects lebih dulu) memakai endpoint yang sudah hidup — jalur terdekat untuk memanfaatkan kerangka P-037 | agen |
| 2 | Modul **Workflow** (Phase 2, `43-WORKFLOW.md`, ADR-0015/0016 `ACCEPTED`) | agen |
| 3 | Putuskan **lisensi** (Q-023) supaya `T-050` dapat dikerjakan | pemilik proyek |
| 4 | Jadikan job `ledger` di CI sebagai *required status check* GitHub | pemilik proyek |
