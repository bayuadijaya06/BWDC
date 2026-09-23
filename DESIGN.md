# DESIGN.md — Arah Desain BWDCS

- **Status:** `TERISI` — **jalur 2 ADR-0007**: arah desain disusun **agen atas instruksi eksplisit user** ("Isi DESIGN.md, tentukan dengan rekomendasi anda sendiri berdasarkan best practices"), bukan identitas resmi pemilik produk.
- **Tanggal dibuat:** 2026-09-17 · **Diisi:** 2026-09-21 (sesi P-037, `T-006`)
- **Diisi oleh:** agen (Buffy) atas perintah user
- **Peringatan yang menyertai status ini (wajib dibaca sebelum menimpa atau mempercayai butir rasa):** arah yang ditulis agen cenderung jatuh ke selera default AI, yaitu hal yang justru disaring antislop. Butir **struktural** di bawah (dials, aturan aksesibilitas, pemisahan warna semantik, motif, aturan mono) adalah keputusan yang dipertanggungjawabkan dan boleh dipertahankan. Butir **rasa** (kepribadian, referensi, penamaan tema) adalah draf: user boleh menggantinya tanpa ADR baru, dan penggantian itu tidak membatalkan bukti kontras.
- **Konsekuensi status ini:** UI yang dibangun dari dokumen ini **bukan** lagi "draft without direction"; dial resmi berlaku (bukan dial sementara 1/1/1). Selama status masih `TERISI` dan bukan "dikonfirmasi pemilik produk", setiap UI tetap wajib lulus Delivery Gate antislop.
- **Sumber mekanis:** palet, skala, dan dial di bawah dikunci di `frontend/src/styles/tokens.css` dan diperiksa test (kontras WCAG AA per pasangan token). Bila dokumen ini dan token berbeda, **token yang berlaku** dan dokumen ini yang diperbaiki pada sesi yang sama.

> **Aturan untuk agen:** dilarang mengarang logo, avatar, angka, atau referensi visual (antislop R-18/R-23). BWDCS tidak punya berkas logo: identitasnya adalah **wordmark teks**, dan avatar pengguna adalah **inisial dari data nyata** (`username`), bukan foto stok. Isi hanya field di bawah ini, jangan menambah bagian promosi atau klaim.

---

## 1. Identitas & Kepribadian

| Field | Isi |
|---|---|
| Nama produk | BWDCS (Business Workflow & Document Control System) |
| Kategori | Internal business tool: manajemen proyek, dokumen, workflow approval, task, audit |
| Pengguna utama | Administrator, Manager, Contributor, Reviewer, Viewer (`44-SECURITY.md` §3.1) |
| Kepribadian | **"Ruang arsip yang bekerja"**: tenang, presisi, tegas pada status, tanpa hiasan. Bahasa visualnya kertas + tinta + nomor indeks, bukan dasbor pemasaran. Setiap baris adalah catatan yang bisa ditelusuri. |
| Yang harus dihindari | (1) kartu berderet seragam tanpa hierarki (R-14); (2) angka tanpa sumber, statistik tren, atau testimoni (R-17/R-18/R-36); (3) gradien, glow, glassmorphism (R-01/R-10/R-12/R-13); (4) ikon generik sparkle/star/robot (R-04); (5) biru-indigo bawaan framework dan tata letak klon Linear/Vercel/Stripe (R-30); (6) sudut pil di semua elemen (R-11); (7) skala tipografi gaya landing page (h1 raksasa) di dalam alat kerja |

**Tiga kata yang menjawab "apa ini" tanpa melihat logo:** tercatat, terurut, tertelusur.

## 2. Palette (maksimum 2-3 warna inti + 1 accent, R-29)

Tiga warna inti (Paper, Ink, Line) plus **satu** accent (Signal). Warna status di §2.4 adalah **semantik wajib** `50-FSD.md` §11 (ADR-0012), bukan warna brand: ia hanya muncul pada badge/pill/indikator status, tidak pernah sebagai warna aksi.

### 2.1 Core 1 — Paper (latar)

| Token | Hex | Peran | Reason (wajib, R-31) |
|---|---|---|---|
| `--paper-000` | `#FFFFFF` | Permukaan panel/tabel | Panels duduk di atas latar, bukan di atas bayangan: batas dipakai garis, bukan shadow (R-12) |
| `--paper-100` | `#F6F5F2` | Latar aplikasi | Netral **hangat**, bukan slate/biru-abu. Ruang arsip berwarna kertas; pilihan ini yang memisahkan BWDCS dari dasbor abu-abu-biru default (R-20/R-30) |
| `--paper-200` | `#EDEBE6` | Hover baris, area tenggelam | Satu langkah lebih gelap saja, supaya tabel tetap bisa dipindai tanpa garis tebal |

### 2.2 Core 2 — Ink (teks & aksi)

| Token | Hex | Peran | Reason |
|---|---|---|---|
| `--ink-900` | `#15181C` | Teks utama, tombol primer | Hampir hitam dengan cast dingin tipis; kontras **16,33:1** di atas `--paper-100`. Tombol primer memakai tinta, bukan warna jenuh, sehingga accent tetap satu-satunya suara berwarna |
| `--ink-700` | `#454B54` | Teks sekunder, judul kolom | Kontras **8,07:1**; hierarki teks dibuat dengan bobot/warna, bukan ukuran raksasa |
| `--ink-500` | `#63696F` | Teks bantu, metadata, placeholder | Kontras **5,09:1** (lulus AA 4,5:1). Batas bawah yang dipakai: apa pun yang lebih pucat dari ini dilarang untuk teks |
| `--ink-300` | `#C7CACF` | Garis pemisah baris/panel (`--line`) | Kontras 1,51:1 — **dekoratif**, sengaja tipis: pemisah tidak menyampaikan status, jadi tidak dikenai syarat 3:1 WCAG 1.4.11 |
| `--line-strong` | `#63696F` = nilai `--ink-500` | Batas kontrol form, garis luar tabel | Kontras 5,09:1 (>3:1) supaya batas input tetap terlihat (WCAG 1.4.11); sengaja memakai **satu** nilai yang sama dengan teks bantu, bukan nuansa baru |

### 2.3 Core 3 — Signal (accent tunggal)

| Token | Hex | Peran | Reason |
|---|---|---|---|
| `--signal-700` | `#0E5B63` | Tautan di dalam konten, teks aksen | Kontras **7,15:1** di paper-100 dan **7,79:1** di putih; aman untuk teks |
| `--signal-600` | `#11707A` | **Focus ring**, indikator nav aktif, penanda step workflow berjalan | Teal-cyan, bukan indigo/ungu: satu warna yang tidak dipakai slot status, sehingga "di mana saya sekarang" selalu terbaca sama. Kontras terhadap paper 5,32:1 (>3:1 untuk non-teks) |
| `--signal-100` | `#DCEDEF` | Latar tipis untuk baris aktif | Hanya pada elemen aktif, tidak pernah untuk dekorasi |

Aturan accent: muncul di **satu** tempat per layar sebagai penanda fokus/posisi. Bukan warna tombol, bukan warna badge status, bukan gradien.

### 2.4 Warna status (semantik 50-FSD §11, dipakai badge — bukan warna brand)

Setiap status memakai pasangan `surface` (latar badge) + `ink` (teks badge). Semua pasangan **≥ 5,4:1**, jadi lulus AA pada ukuran teks 12-13px:

| Status kanonik | Label UI | surface | ink | Kontras |
|---|---|---|---|---|
| `draft` | Draft | `#ECEBE8` | `#43464A` | 7,96:1 |
| `in_review` / `running` | In Review | `#E4EBF6` | `#1D4E89` | 7,00:1 |
| `revision_required` | Revision Required | `#FBEEDC` | `#8A5410` | 5,47:1 |
| `approved` / `completed` | Approved | `#E3F0E6` | `#1F6B3C` | 5,54:1 |
| `rejected` | Rejected | `#FBE7E5` | `#9B2C22` | 6,36:1 |
| `archived` | Archived | `#E2E1DE` | `#3A3B3D` | 8,57:1 |

Aturan: **warna tidak pernah satu-satunya pembawa makna** (WCAG 1.4.1). Badge selalu memuat label teks dari tabel `50-FSD.md` §11; status turunan (`overdue`) memakai label teks + penanda bentuk, bukan warna baru.

## 3. Tipografi

| Peran | Font | Reason (wajib, R-06) |
|---|---|---|
| Primary (UI) | system sans stack: `ui-sans-serif, -apple-system, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif` | ADR-0002 melarang aset CDN pihak ketiga, dan R-23 melarang mengarang aset. Font sistem dirender native (tanpa layout shift, tanpa lisensi, tanpa berkas binary di repo). Identitas tidak dibawa oleh typeface display, melainkan oleh **suara mono** dan garis indeks |
| Mono (ID, nomor dokumen, versi, audit, checksum) | system mono stack: `ui-monospace, "SF Mono", "JetBrains Mono", Menlo, Consolas, monospace` | Identitas entitas **adalah** nomor (`WEB-001`, `#a3f9`, `v1.2`). Mono membuat nomor rata kolom dan tidak pernah ambigu antara `1`/`l`/`I`; di tabel, angka memakai **tabular numerals** supaya digit rata per nilai |
| Skala ukuran | 12 / 13 / **14 (body)** / 16 / 20 / 26 px; bobot 400 / 500 / 600; line-height 1,45 body, 1,25 judul | Basis `51-UX.md` §4 dipertahankan (14px body untuk kepadatan alat kerja). Plafon 26px mencegah skala landing page masuk ke dalam aplikasi: judul halaman adalah **penanda**, bukan hero |
| Label kolom | 12px, bobot 500, huruf kapital kecil, `letter-spacing: 0.04em`, warna `--ink-700` | Teknik huruf kapital + tracking dipakai **hanya** pada judul kolom tabel, dengan alasan tertulis (R-06): himpunan labelnya sempit, harus menurun dari data di bawahnya, dan tidak boleh bersaing dengan nilai sel |

Aturan: **tidak ada font yang diunduh saat runtime**, tidak ada berkas font di repo, tidak ada `@import` dari domain pihak ketiga.

## 4. Mood & Kepadatan

- [x] Professional & trustworthy
- [x] Padat informasi, terasa seperti alat kerja (tool-like)
- [ ] Modern & clean *(bukan tujuan; "bersih" adalah dampak, bukan target)*
- [ ] Lainnya: `[TIDAK DIPAKAI]`
- **Kepadatan tabel & form:** baris tabel default **36px** (mode *compact*) dengan padding horizontal 12px; form memakai grid 8px, tinggi kontrol 36px, jarak antar-field 16px. Mode *nyaman* (44px) menyusul sebagai preferensi pengguna, bukan sebagai default kedua.
- **Aturan angka:** kolom angka/ID rata **kanan** dan memakai tabular numerals; kolom teks rata kiri; tanggal rata kiri. (Kesalahan yang paling sering merusak tabel adalah angka rata tengah.)
- **Aturan teks panjang:** satu baris + ellipsis, dengan `title` untuk nilai penuh; hanya satu kolom "deskripsi" yang boleh membungkus baris.

## 5. Dials (antislop Part 3)

| Dial | Nilai | Reason |
|---|---|---|
| ENERGY | **1** | Alat kerja, bukan situs pemasaran. Halaman dibuka 100 kali sehari: yang dibutuhkan kejelasan, bukan sapaan. Setara Linear/GOV.UK pada tangga dial |
| RHYTHM | **2** | Halaman daftar seragam (tabel berulang, dapat diprediksi) tetapi halaman detail berbeda: kepala rekam + dua kolom + timeline. Keseragaman penuh (1) menyembunyikan hierarki; variasi penuh (3) memperlambat pemindaian |
| MOTION | **1** | Hanya transisi status dan hover (120-160ms). Alat kerja yang beranimasi saat dipakai adalah gangguan, dan R-19 melarang animasi template tanpa tujuan UX |

Dials resmi sejak dokumen ini terisi. Nilai sementara 1/1/1 **tidak** berlaku lagi.

## 6. Identity Motif

| Field | Isi |
|---|---|
| Satu gestur/pattern yang berulang dan membuat UI ini "milik BWDCS" | **Punggung rekam (record spine):** garis vertikal tipis di tepi kiri setiap rekam yang warnanya mengikuti status kanonik, ditemani **setiap identitas dalam mono** (nomor dokumen, kode project, versi, id audit). Satu pandangan ke kiri tepi tabel = membaca status seluruh halaman |
| Di mana motif muncul | (1) tepi kiri baris tabel (3px), (2) kepala halaman detail rekam (4px), (3) tiap entri timeline versi/step (2px), (4) baris audit log, (5) kartu antrean Approvals. Motif **tidak** muncul di elemen yang tidak punya status (panel form, dialog, header) |

Aturan: motif ini dekoratif **dan** fungsional (ia menggandakan informasi status secara visual), jadi ia boleh ada tanpa melanggar larangan dekorasi tanpa tujuan (R-01/R-07). Bila suatu saat warna status tidak dibawa lagi oleh punggung, motifnya harus diganti, bukan dipertahankan sebagai hiasan.

## 7. Theme

| Field | Isi |
|---|---|
| Default | **Light** (`Paper`): ruang arsip dengan latar kertas. Bukan dark default, karena R-21 melarang dark sebagai default tanpa alasan brand, dan dokumen arsip dibaca di ruang terang |
| Toggle light/dark | **Ada dan wajib berfungsi keduanya** (R-21 & R-34). Preferensi awal mengikuti `prefers-color-scheme`, lalu preferensi pengguna menang. Dark = **"Malam arsip"**: `--paper-950 #14161A`, teks `--ink-100 #ECEDEF` (kontras 15,46:1), accent `--signal-400 #6FC6CE` (9,18:1), warna status versi gelap dengan kontras ≥ 9,4:1 |

Aturan: tidak ada warna yang ditulis langsung di komponen. Seluruh warna, radius, dan bayangan berasal dari token; itulah yang membuat mode gelap tidak "rusak sebagian".

## 8. References & Anti-references

| Field | Isi |
|---|---|
| Referensi rasa (bukan untuk ditiru) | (1) **Kartu katalog perpustakaan**: nomor indeks, judul, tahun, satu baris per rekam. (2) **Buku besar cetak**: kolom bergaris, angka rata kanan, judul kolom kecil. (3) **Checklist operasi ruang kendali**: status tanpa ambiguitas, tidak ada kalimat pemasaran. Ketiganya dipakai untuk *rasa keteraturan*, bukan untuk menyalin tampilannya |
| Produk yang sengaja TIDAK ditiru (R-30) | Linear, Vercel, Stripe, Notion, dan template dasbor admin bergaya glassmorphism/gradien. Termasuk "tiga kartu KPI + grafik tren" yang tidak dapat dipertanggungjawabkan sumbernya (R-38) |

---

## 9. Design Read (diisi saat build dimulai)

> "Reading this as: **internal document-control console** for **administrators, managers, and contributors who process records daily**, in a **ruled-ledger visual language (paper, ink, mono index numbers)**, dial **ENERGY 1 / RHYTHM 2 / MOTION 1**."

**Yang mengikuti Design Read ini pada build pertama (sesi P-037, `T-048`):** AppShell (header + sidebar + area konten), primitives (Button, Badge status, Field/Input, Panel, DataTable, EmptyState, ErrorState, Skeleton), halaman Login, dan halaman "modul belum dibangun" untuk setiap menu yang belum punya halaman nyata. Halaman bisnis (Projects, Documents, Tasks, Approvals, Reports, Administration) menyusul dan otomatis terikat pada dialog dan motif di dokumen ini.

**Bukti pemenuhan (bukan klaim):** kontras setiap pasangan token dihitung ulang oleh test `src/styles/tokens.contrast.test.ts`; dial dinyatakan di muka dan diperiksa di Delivery Gate; tidak ada berkas font, logo, atau ikon pihak ketiga di dalam repo.
