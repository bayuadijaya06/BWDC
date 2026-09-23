# P-049 — 2026-09-23 — Rentang tenggat jadi satu kendali, dan ukurannya menemukan cacat sendiri

| Field | Isi |
|---|---|
| ID | P-049 |
| Waktu mulai | 2026-09-23 (lanjutan sesi P-048 pada worktree yang sama) |
| Aktor | agen (Buffy, perkakas berkas + terminal + Preview) |
| Model / agen | z-ai/glm-5.3-flash (lanjutan) lalu deepseek/deepseek-v4-flash |
| Fase roadmap | 4 (UI) |
| Task terkait | `T-065` (baru, DONE di sesi ini) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Rapikan pengelompokan penyaring rentang tenggat di halaman Tasks supaya kedua batasnya tidak terpisah baris."

## 2. Interpretasi & Scope

- **Yang diminta:** kedua batas rentang tenggat (`Tenggat dari` dan `Tenggat sampai`) pada halaman
  Tasks berhenti berperilaku sebagai dua penyaring yang tidak berhubungan.
- **Yang TIDAK termasuk (out of scope):** menyentuh semantik rentangnya (batas **inklusif** RFC 3339
  tetap seperti `42-API.md` §6), menyentuh backend/perpindahan kueri, dan menambah penyaring baru.
- **Asumsi yang diambil:**
  - Kekurangannya sudah dinyatakan sendiri oleh sesi P-047 di `70-TESTING.md` §3.14e: baris penyaring
    **melipat** (`flex-wrap`), sehingga dua kolom terpisah dapat jatuh ke garis berbeda — batas awal di
    satu baris, batas akhir di baris berikutnya. Jadi yang diminta adalah menutup kekurangan yang
    tercatat, bukan menafsirkan keluhan baru.
  - "Supaya tidak terpisah baris" dibaca sebagai **hubungan**, bukan sekadar posisi: yang harus benar
    bukan hanya "sebaris pada 1440px", melainkan "tetap satu rentang pada lebar mana pun".
  - Karena rentang itu satu nilai yang dibentuk dua isian, kendalinya dijadikan **satu kelompok
    ber-peran `group` berlabel "Rentang tenggat"** — label yang terlihat adalah label **kelompok**,
    sedangkan tiap isian tetap bernama sendiri lewat `aria-label`.
- **Pertanyaan yang muncul:** tidak ada yang baru. Sesi ini tidak mengubah status `Q-016`/`C-063`.

## 3. Rencana

1. Ukur dulu bentuk lamanya, supaya "terpisah baris" berhenti menjadi dugaan.
2. Satukan kedua isian ke satu kelompok berlabel; tentukan perilakunya per lebar.
3. Kunci strukturnya dengan test, dan kunci perilakunya di peramban dengan skrip bukti.
4. Buktikan keduanya **punya gigi** dengan memasang kembali bentuk lamanya.
5. Selaraskan dokumen (`51-UX.md` §2.1, `70-TESTING.md` §3.14f) dan ledger.

## 4. Aksi yang Dilakukan

1. `scripts/responsive-evidence.mjs` bagian `navigation` diperluas: penyaring Task kini juga diukur
   per **kelompok** (`fromTop`, `toTop`, `sameLine`, `sameGroup`, `contained`) selain kesebarisan kolom
   yang sudah ada, dan pasangan itu diukur pada lebar **lebar** maupun **sempit** (gulir mendatar di
   halaman ini, bukan di halaman Projects seperti sapuan tema sebelumnya).
2. `frontend/src/pages/Tasks/index.tsx`: kedua isian berpindah ke satu `<div role="group" aria-labelledby>`
   berlabel "Rentang tenggat", `flex flex-col gap-1.5 sm:flex-row sm:items-center sm:gap-2`.
3. **Ukuran menemukan cacat yang baru saja dibuat:** pada 375px pasangan berdampingan itu menuntut
   ~400px dan halaman menggulir mendatar **71px**. Diperbaiki dengan **menumpuknya di dalam kelompok
   yang sama** pada layar sempit — bukan memotong lebar isian, bukan mengembalikannya ke dua kolom.
4. Pencarian isian di skrip bukti dipindah dari `aria-label` ke **`id`** yang stabil, sesudah
   percobaan pertama terbukti berhenti mengukur tepat pada bentuk yang harus ditangkapnya (lihat §6).
5. `frontend/src/pages/Tasks/Tasks.test.tsx`: satu test baru mengunci strukturnya.
6. `51-UX.md` §2.1 dan `70-TESTING.md` §3.14f diselaraskan.

## 5. File yang Berubah

### Changed

| File | Keterangan |
|---|---|
| `frontend/src/pages/Tasks/index.tsx` | Kedua batas rentang berada di satu kelompok berlabel "Rentang tenggat": berdampingan pada lebar lebar, menumpuk **di dalam kelompok yang sama** di layar sempit — P-049 |
| `frontend/src/pages/Tasks/Tasks.test.tsx` | Satu test baru: kedua batas di satu `role="group"` bernama "Rentang tenggat", barisnya ber-`flex` **tanpa** `flex-wrap`, tiga anak (dari, pemisah, sampai) — P-049 |
| `scripts/responsive-evidence.mjs` | Bagian `navigation` mengukur pasangan rentang per kelompok pada lebar lebar **dan** sempit (`sameLine`, `sameGroup`, `contained`, `overflowX`), plus tiga catatan kegagalan; pencarian isian lewat `id`, bukan `aria-label` — P-049 |
| `docs/design/51-UX.md` | §2.1: aturan "dua kendali yang membentuk satu nilai berdiri sebagai satu kelompok", beserta alasan mengapa pada lebar sempit yang dituntut **kelompoknya**, bukan garisnya — P-049 |
| `docs/design/70-TESTING.md` | §3.14f: pengukuran bentuk lama vs bentuk baru dan bukti gigi tiga butir — P-049 |
| `README.md` | Ringkasan frontend menyebut penyaring rentang tenggat sebagai satu kelompok berlabel — P-049 |

## 6. Verifikasi (WAJIB)

**Frontend:** `npm run typecheck` bersih, `npm run lint` bersih, **241 test / 24 berkas** hijau (naik
dari 240/24), `Tasks.test.tsx` 16 test hijau.

**Bukti peramban nyata (`node scripts/responsive-evidence.mjs` → `OK`):**

| Ukuran | 1440px | 375px |
|---|---|---|
| `fromTop` / `toTop` | `289` / `289` | `672` / `747` |
| `sameLine` | **true** | false (menumpuk — disengaja) |
| `sameGroup` | **true** | **true** |
| `contained` (melebar melewati kolomnya) | `0` | `0` |
| `overflowX` (gulir mendatar) | `0` | `0` |
| tinggi kontrol | 36px seragam | 44px seragam |
| garis pada baris penyaring | 2 | 6 |

**Gigi dibuktikan.** Bentuk lama (dua kolom terpisah) dipasang kembali → skrip bukti **FAIL dengan
tiga butir sekaligus**:

```
FAIL  responsive-evidence: 3 klaim tata letak tidak terbukti
  penyaring task: kedua batas rentang tidak berada di satu kelompok ber-label
  penyaring task: kedua batas rentang terpisah baris (atas 220px vs 289px) — satu batas dapat jatuh ke garis berikutnya saat baris penyaring melipat
  penyaring task @ 375px: kedua batas rentang tidak berada di satu kelompok ber-label
```

Angka `220px vs 289px` itu **dari bentuk lamanya**, bukan dari percobaan saya sendiri — angka yang
diduga P-047 kini terukur ulang, bukan dikutip. Bentuk lamanya dipulihkan, lalu skrip hijau lagi.

**Satu cacat pada alat ukurnya sendiri, ketahuan dari percobaan itu.** Percobaan pertama butir ini
mencari isiannya lewat `aria-label`. Pada bentuk lama, `aria-label` **tidak lagi** menjadi cara
pelabelannya, sehingga butir itu berbunyi "kedua batas rentang tenggat tidak ditemukan" — ia berhenti
**mengukur** tepat pada bentuk yang harus ditangkapnya, kelas yang sama dengan **C-075**. Pencariannya
dipindah ke **`id`** yang stabil, dan setelah itu cacatnya tertangkap karena alasan yang benar.
Pesan "tidak ditemukan" pun dikoreksi agar menyebut `id`, bukan `aria-label`, supaya pesan berikutnya
tidak menyesatkan.

**Tanpa perubahan backend, kontrak API, izin, atau skema** — sesi ini tidak menyentuh satu berkas pun
di `backend/`.

## 7. Hasil & Dampak

- Rentang tenggat di halaman Tasks kini **satu kendali**: kedua batasnya tidak dapat terpisah ke baris
  berbeda pada lebar lebar, dan tetap terbaca sebagai satu rentang pada lebar sempit.
- Aturannya **diturunkan dari sebabnya** dan dinyatakan di `51-UX.md` §2.1, sehingga berlaku juga untuk
  kendali lain yang membentuk satu nilai dari beberapa isian — bukan tambalan satu halaman.
- Klaim "tidak terpisah baris" berhenti menjadi klaim: tiga butir mesin menjaga hubungan maupun
  kesebarisannya, dan gigi ketiganya dibuktikan.
- Yang **tidak** dijanjikan: rentangnya belum punya jalan pintas (mis. "7 hari terakhir"), dan
  penyaring `Penanggung jawab` masih dibatasi `C-063` (`Q-024`).

## 8. Update Ledger (Checklist Wajib)

- [x] `docs/progress/CHANGELOG.md` — sesi P-049
- [x] `docs/progress/TASKS.md` — `T-065` (`DONE`)
- [x] `docs/progress/STATE.md` — "Diperbarui oleh" P-049 + hitungan test frontend
- [x] `docs/progress/SESSION-LOG.md` — ringkasan sesi
- [x] `CONTINUE.md` — posisi terakhir + task berikutnya
- [x] `README.md` — ringkasan frontend
- [x] `70-TESTING.md` §3.14f + `51-UX.md` §2.1
- [x] `TRACEABILITY.md` — catatan sesi ditambahkan (**tanpa** perubahan status): ini kepatuhan UI terhadap `50-FSD.md` §6.1/§6.2, bukan requirement baru, dan semantik rentangnya (Q-017, batas tertutup) tidak disentuh
- [x] `OPEN-QUESTIONS.md` — **tidak berubah** (`Q-016`/`Q-024` tetap terbuka)
- [x] Audit — **tidak ada temuan baru**

## 9. Next Action

- Modul **Approvals** di frontend (`50-FSD.md` §5.4) — kontraknya sudah hidup sejak P-048, jadi tidak
  ada penunggu. Modul backend berikutnya: **Notification** (`42-API.md` §8) atau **Audit** (§9).
- Dua pertanyaan pemilik yang masih terbuka: **Q-016** (`?view=mine` Documents) dan **Q-024**
  (endpoint daftar pengguna — `C-063`).
