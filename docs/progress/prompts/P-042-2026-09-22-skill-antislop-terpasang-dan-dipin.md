# P-042 — 2026-09-22 — Skill antislop terpasang dipin ke tag rilis, daftar aturan tidak lagi disalin

| Field | Isi |
|---|---|
| ID | P-042 |
| Waktu mulai | 2026-09-22 (zona waktu lokal) |
| Aktor | agen |
| Model / agen | Buffy |
| Fase roadmap | pra-fase / lintas fase (sistem kerja, bukan modul) |
| Task terkait | `T-054`, `T-055` |
| Status akhir | DONE |

---

## 1. Prompt User

> Terkait antislop skills yang ada pada `OPEN-QUESTIONS.md`, saya mengambil dari repository
> `https://github.com/miqdadbadjuber/anti-slop`; saya ambil satu file dan letakkan di root
> (`antislop.md`). Coba akses direktori tersebut, dan coba terapkan dengan proper pada project ini,
> terutama jika memang harus ada skills yang terdaftar. Jika sudah, sesuaikan dokumen terkait desain
> dengan rules dari repository tersebut.

## 2. Interpretasi & Scope

- **Yang diminta:** memeriksa repo upstream, menerapkan sistem antislopnya dengan benar (termasuk
  skill yang sudah didaftarkan `AGENTS.md`), dan menyelaraskan dokumen desain dengan aturannya.
- **Yang TIDAK termasuk:** mengubah keputusan mode (`during` tetap, ADR-0006), menyentuh kode
  backend/frontend, dan menyusun ulang `DESIGN.md` (arahan desain tetap milik P-037).
- **Asumsi yang diambil:** "terapkan dengan proper" berarti **menghilangkan ketidakbenaran**, bukan
  menambah bacaan. Karena itu pekerjaan utamanya menjadi tiga: memasang berkas yang dijanjikan,
  menyelaraskan berkas core dengan rilis upstream, dan **menghapus salinan** aturan dari dokumen
  desain.
- **Pertanyaan yang muncul:** Q-003 (skill tidak tersedia) — **dijawab user pada sesi ini** dengan
  memilih agar agen mengunduhnya langsung dari repo itu dan memasang **kelima** skill.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Periksa repo upstream, tentukan rilis terakhir | tahu tag yang dipakai, bukan `main` |
| 2 | Unduh 5 skill + core + lisensi dari tag itu, catat `sha256` | berkas byte-identik dengan sumbernya |
| 3 | Bandingkan `antislop.md` lokal dengan upstream pada tag yang sama | tahu apakah core di root benar |
| 4 | Tulis `skills/README.md` (provenans, cara memperbarui, dan apa yang diperiksa mesin) | salinan yang keasliannya dapat diperiksa ulang |
| 5 | Perbaiki blok penunjuk `AGENTS.md` supaya sesuai kenyataan | entry file tidak lagi menjanjikan berkas hantu |
| 6 | Hapus salinan aturan/Gate di `01-AGENT-WORKFRAME.md`, ganti penunjuk + keputusan proyek | satu sumber aturan |
| 7 | ADR-0025 (pin, lisensi, amandemen butir 2 ADR-0006) | keputusan tercatat, bukan tersirat |
| 8 | `scripts/check-antislop-refs.sh` + CI + daftar verifikasi dokumen | kelas cacatnya tertahan mesin |
| 9 | Bukti: pemeriksa kontras upstream + lima pemeriksa dokumen | bukan klaim |
| 10 | Ledger: temuan, log, `CHANGELOG`, `TASKS`, `STATE`, `SESSION-LOG`, `CONTINUE`, `OPEN-QUESTIONS` | sesi dapat dilanjutkan agen lain |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Baca rilis upstream (`v3.2.12`) dan bandingkan dengan `antislop.md` lokal | sumber aturan harus satu | core lokal ternyata **varian lama** — **C-065** |
| 2 | Unduh `antislop.md` + lima skill + `contrast-check.py` + `LICENSE` dari **tag** `v3.2.12` | pin supaya dapat diaudit; aturan inti melarang agen mengambil skill saat berjalan | 9 berkas, semua `sha256` dicatat |
| 3 | Tulis `skills/README.md` | salinan tanpa provenans tidak dapat dibedakan dari karangan | tabel `sha256`, cara naik versi, daftar apa yang diperiksa |
| 4 | Perbaiki blok penunjuk `AGENTS.md` (path nyata, versi dipin, aturan anti-drift) | entry file pernah mendaftarkan lima skill yang tidak ada — **C-064** | pointer sesuai disk |
| 5 | Tulis ADR-0025 | keputusan ini menyentuh berkas pihak ketiga, lisensi, dan satu ADR lama | `ACCEPTED`; butir 2 ADR-0006 diamandemen, indeks ADR diberi penanda |
| 6 | Tulis `scripts/check-antislop-refs.sh` (7 pemeriksaan) | kelas C-064/C-065 hanya tertangkap mesin | `antislop-refs OK` |
| 7 | Hapus tabel 23 aturan dan blok 4 Gate di `01-AGENT-WORKFRAME.md` §3.2/§5.3, ganti keputusan proyek + cara melaporkan Gate | salinan aturan = sumber kebenaran kedua (pola C-014) | dokumen menunjuk, tidak mengutip |
| 8 | Tambah §6.4 di `02-AGENT-PROGRESS-PROTOCOL.md` dan perintahnya di README §12.3, `12-DEVELOPMENT-WORKFLOW.md` §8, `90-AGENT-GUIDE.md`, `AGENTS.md`, CI | pemeriksa yang tidak dijalankan tidak ada gunanya | lima pemeriksa di CI |
| 9 | Jalankan pemeriksa kontras **upstream** atas token proyek | membuktikan alat pihak ketiga cocok dengan angka yang sudah dipegang test sendiri | enam pasangan berkomentar di `tokens.css`, angkanya sama persis |
| 10 | Perbarui pohon folder README dan tambah §6 di `check-readme-facts.sh` | menemukan **C-066**: README menyebut 2 dari 5 skrip yang sudah jalan di CI, dan pemeriksa angka tidak dapat menangkapnya | daftar himpunan diperiksa dua arah, 25 → 37 fakta |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `skills/README.md` | Added | Provenans (sumber, tag `v3.2.12`, `sha256` 9 berkas, lisensi MIT), cara memperbarui, daftar yang diperiksa mesin | NON-BLOCKING sistem kerja |
| `skills/antislop/SKILL.md`, `skills/antislop-ui/SKILL.md`, `skills/antislop-copywriting/SKILL.md`, `skills/antislop-human/SKILL.md`, `skills/antislop-layoutmobile/SKILL.md`, `skills/antislop-code/SKILL.md`, `skills/antislop-human/contrast-check.py`, `skills/LICENSE-antislop` | Added | Salinan byte-identik dari tag rilis | — |
| `antislop.md` | Changed | Diganti berkas pristine dari tag `v3.2.12` (varian lama → rilis) | — |
| `AGENTS.md` | Changed | Blok penunjuk sesuai disk (path nyata, versi dipin), aturan anti-drift `R-XX`, perintah pemeriksa kelima, angka audit 65/63/2 | — |
| `scripts/check-antislop-refs.sh` | Added | 7 pemeriksaan: daftar aturan, rujukan `R-XX`, path skill dua arah, `sha256` vs tabel, salinan core, klaim rentang, sidik jari aturan tersalin | — |
| `.github/workflows/ci.yml` | Changed | Langkah kelima di job `ledger` | — |
| `docs/design/01-AGENT-WORKFRAME.md` | Changed | §3.2 dan §5.3: salinan daftar aturan & blok Gate **dihapus**, diganti penunjuk + keputusan proyek + bentuk laporan Gate; pohon §6 memuat `skills/` dan `scripts/` yang sebenarnya | — |
| `docs/design/02-AGENT-PROGRESS-PROTOCOL.md` | Changed | §6.4 baru (kelas cacat C-064/C-065) + checklist §8 | — |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md`, `docs/design/90-AGENT-GUIDE.md`, `README.md` | Changed | Perintah `check-antislop-refs.sh` masuk daftar verifikasi; penjelasan apa yang ditahannya | — |
| `docs/adr/0025-skill-antislop-terpasang-dan-dipin.md` | Added | Keputusan pin, lisensi, amandemen butir 2 ADR-0006, larangan menyalin aturan | — |
| `docs/adr/README.md` | Changed | Indeks ADR-0025; status ADR-0006 diberi penanda amandemen | — |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | Q-003 → `RESOLVED` | — |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md`, `docs/progress/audits/README.md` | Changed | **C-064** dan **C-065** ditambahkan (keduanya `FIXED`), marker audit `65/63/0/2/0/37` | — |
| `README.md` §4/§12.3/§13 | Changed | Pohon folder memuat `skills/` + kelima skrip (temuan **C-066**); `check-antislop-refs.sh` masuk §12.3; §13 mencatat bahwa `skills/LICENSE-antislop` adalah salinan lisensi MIT pihak ketiga, **bukan** lisensi proyek yang masih tertunda (Q-023) | — |
| `scripts/check-readme-facts.sh` | Changed | **§6** baru: daftar berkas di pohon README dibandingkan dua arah dengan isi `scripts/` dan `skills/` (37 fakta) | — |
| `scripts/check-doc-links.sh` | Changed | `guide.md` → `ABSENT_DOCS`; isi `skills/` selain `README.md` dilewati (berkas pihak ketiga ber-`sha256`) | — |
| `docs/progress/TASKS.md` | Changed | `T-054` dan `T-055` `DONE` | — |
| `docs/progress/STATE.md`, `docs/progress/SESSION-LOG.md`, `docs/progress/CHANGELOG.md`, `CONTINUE.md` | Changed | Ledger sesi ini | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `bash scripts/check-antislop-refs.sh` | `antislop-refs OK — 38 aturan (R-01..R-38), 93 rujukan, 7 berkas skill, 7 pemeriksaan` | PASS |
| 2 | `bash scripts/check-ledger.sh` | `ledger OK — 0 peringatan, 236 test di backend` | PASS |
| 3 | `python3 skills/antislop-human/contrast-check.py` atas enam pasangan token yang angkanya sudah dikomentari di `tokens.css` | `#63696f` di `#f6f5f2` = **5,09**; `#0e5b63` di `#f6f5f2` = **7,15**; `#ecedef` di `#14161a` = **15,46**; `#9aa1a8` di `#14161a` = **6,93**; `#7fd1d9` di `#14161a` = **10,37**; `#7a828a` di `#14161a` = **4,65** — semuanya `PASS`, angkanya sama dengan komentar di berkas itu; `--selftest` alat juga lulus (`8 reference pairs OK`) | PASS |
| 4 | Enam cacat disuntikkan sementara (nomor aturan yang tidak ada, path skill hantu, `sha256` diubah, salinan core diubah, kalimat Gate disalin, klaim rentang aturan yang ujungnya berhenti sebelum yang terakhir) | keenamnya `FAIL` beserta nomor baris, lalu dipulihkan dan hijau kembali | PASS |
| 5 | `bash scripts/check-readme-facts.sh` setelah **§6** ditambahkan | `readme-facts OK — 37 fakta diperiksa` (25 → 36); dua cacat disuntikkan sementara (satu baris skrip dihapus dari pohon; satu path `skills/...` hantu) keduanya `FAIL`, lalu dipulihkan | PASS |
| 6 | `bash scripts/check-doc-links.sh`, `bash scripts/check-api-contract.sh`, `bash scripts/check-ledger.sh` | `BROKEN: 0`; `api-contract OK — 110 pemeriksaan, 55 endpoint`; `ledger OK — 0 peringatan, 236 test` | PASS |

- [x] Typecheck / build dijalankan — tidak ada kode yang berubah, tetapi keduanya dijalankan sebagai pagar: `make test` hijau (sembilan paket, 236 test) dan frontend `tsc --noEmit` + `eslint .` bersih + `129 test / 14 berkas`
- [x] Test relevan dijalankan — tidak ada test baru (perubahan dokumen + skrip); suite backend tetap 236 test lewat `check-ledger`
- [x] Perubahan dokumen dicek konsisten (referensi file ada) — `check-doc-links.sh` → `BROKEN: 0` (dua klasifikasi baru ditambahkan di sesi ini: `guide.md` masuk daftar `ABSENT_DOCS` karena memang sengaja tidak disalin, dan isi `skills/` selain `README.md` dilewati karena berkas pihak ketiga yang rujukannya menunjuk tata letak repo upstream)
- [ ] Jika UI: Delivery Gate antislop dijalankan — **tidak berlaku**: tidak ada UI yang dibangun di sesi ini

## 7. Hasil & Dampak

- **Selesai:** kelima skill terpasang dan dapat diperiksa keasliannya; core di root kembali satu versi (dan identik dengan salinan di `skills/antislop`); dua salinan daftar aturan di dokumen desain dihapus; satu keputusan baru (ADR-0025) dan satu pemeriksa baru; Q-003 ditutup.
- **Belum selesai:** tidak ada. Pekerjaan halaman berikutnya tidak tersentuh, sesuai scope.
- **Risiko / utang:** salinan pihak ketiga harus diperbarui manual bila upstream bergerak. Itu **disengaja** dan tercatat (pin + tabel `sha256`): versi lama yang konsisten lebih baik daripada campuran core lama dengan skill baru.
- **Temuan ketiga, kelas baru untuk tabel audit: C-066 (`FIXED`).** Pohon folder `README.md` §4 menyebut **dua dari lima** skrip yang sudah berjalan di CI — `check-readme-facts.sh` tidak menangkapnya karena ia membandingkan **angka & versi**, bukan **daftar**. Ketahuan justru ketika pohon itu dibuka untuk menambahkan `skills/`. Perbaikannya dua arah: daftar diperbarui **dan** pemeriksanya diperluas (§6 `check-readme-facts.sh`), sehingga himpunan yang basi tertangkap seperti angka yang basi.
- **Dampak ke dokumen desain:** `01-AGENT-WORKFRAME.md` §3.2/§5.3/§6, `02-AGENT-PROGRESS-PROTOCOL.md` §6.4/§8, `12-DEVELOPMENT-WORKFLOW.md` §8, `90-AGENT-GUIDE.md`, `README.md` §12.3. Tidak ada dokumen desain lain yang menyebut daftar aturan (diperiksa `grep`).

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-054` DONE)
- [x] `TRACEABILITY.md` — tidak ada requirement produk yang disentuh
- [x] `OPEN-QUESTIONS.md` diperbarui (Q-003 `RESOLVED`)
- [x] ADR dibuat (ADR-0025)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Lanjutkan halaman bisnis berikutnya (**Documents**) dengan pola P-041 | agen |
| 2 | Jawab Q-024 (endpoint daftar pengguna untuk pemilih `Owner`) — menahan anggota project | user |
| 3 | Naikkan pin antislop bila upstream merilis versi baru (ikuti `skills/README.md` §2) | user/agen atas izin |
