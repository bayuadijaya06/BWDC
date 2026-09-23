# ADR-0025 — Skill antislop terpasang di repo, dipin ke tag rilis, dan aturan hanya bersumber dari `antislop.md`

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-22
- **Dokumen terkait:** ADR-0006 (**butir 2 konsekuensinya diamandemen oleh ADR ini**; mode `during` tetap), `antislop.md`, `skills/README.md`, `docs/design/01-AGENT-WORKFRAME.md` §3.2/§5.3, `AGENTS.md` (blok penunjuk), `scripts/check-antislop-refs.sh`, Q-003

## Konteks

`antislop.md` adalah **core**: satu berkas berisi 38 aturan (R-01..R-38) dalam tiga tier, dial, dan
Delivery Gate. Sistemnya juga menyediakan **skill** opsional, satu folder per bidang kerja
(nama seperti `antislop-ui`, `-copywriting`, `-human`, `-layoutmobile`, `-code`), masing-masing sebuah berkas instruksi `skills/<nama>/SKILL.md`
yang merujuk aturan inti lewat nomor, bukan menyalinnya.

Keadaan sebelum sesi ini tidak konsisten, dan keduanya adalah cacat yang tidak terlihat dari membaca:

1. **`AGENTS.md` mendaftarkan lima skill yang tidak ada di disk.** Blok penunjuk antislop sudah
   menyebut `skills/antislop-ui/SKILL.md` dan empat lainnya sejak P-037, padahal direktori
   `skills/` belum pernah dibuat (kelas **C-064**). Entry file menyatakan sesuatu yang tidak benar,
   dan agen berikutnya akan mencoba membacanya.
2. **Core di root bukan berkas yang sama dengan core yang dipegang dokumen.** `antislop.md` di root
   adalah **varian lama** (686 baris, Delivery Gate tanpa penanda `[ ]`, tanpa butir *scope* pada
   R-02), sementara `01-AGENT-WORKFRAME.md` §5.3 memuat salinan Gate yang **lebih baru**. Dua versi
   core beredar di satu repo, dan tidak ada yang tahu mana yang berlaku (kelas **C-065**).

Q-003 sudah mencatat masalah ini sebagai pertanyaan terbuka, dan aturan inti antislop sendiri
menetapkan skill **disediakan user, bukan diunduh agen** ("an agent that downloads one at runtime is
fetching its own next prompt: do not do it"). Karena itu keputusan ini bukan keputusan agen.

**Yang diputuskan user pada sesi ini:** menyebutkan sumbernya
(`https://github.com/miqdadbadjuber/anti-slop`), meminta kelima skill diterapkan dengan benar, dan
**memberi izin eksplisit** kepada agen untuk mengunduhnya langsung dari repositori itu.

## Keputusan

**Kelima skill dipasang, isinya apa adanya, dan dipin ke satu tag rilis; aturan tetap hanya bersumber
dari `antislop.md`.**

| Aspek | Keputusan |
|---|---|
| Yang dipasang | `antislop.md` (core, di root) + lima skill (`antislop`, `antislop-ui`, `antislop-copywriting`, `antislop-human` + `contrast-check.py`, `antislop-layoutmobile`, `antislop-code`) + salinan `LICENSE` (MIT) |
| Sumber | repo `miqdadbadjuber/anti-slop`, **tag rilis `v3.2.12`** — bukan `main` |
| Keaslian | disalin **byte-identik**; `sha256` setiap berkas dicatat di `skills/README.md` §1 dan diperiksa mesin |
| Cara memperbarui | §2 `skills/README.md`: naikkan **semua** berkas bersamaan dan perbarui tabel provenansnya; jangan mencampur core lama dengan skill baru, karena skill merujuk nomor aturan inti |
| Sumber aturan | **`antislop.md` saja.** Daftar aturan, tier, dial, dan butir Delivery Gate **dilarang disalin** ke dokumen proyek; dokumen hanya **menunjuk** (pola C-014) |
| Izin unduh | **satu kali, untuk sesi ini**, dan tercatat di sini. Ia **bukan** aturan tetap: agen mana pun setelah ini **tidak boleh** mengunduh ulang atas inisiatif sendiri, termasuk "sekadar menyegarkan" |
| Pemeriksa | `bash scripts/check-antislop-refs.sh` (tanpa jaringan) menahan enam kelas cacat: rujukan `R-XX` yang tidak ada di core, path skill yang disebut tapi tidak ada di disk, berkas skill di disk yang tidak terdaftar, `sha256` yang menyimpang dari tabel, salinan core di `skills/antislop/` yang berbeda dari core di root, dan kalimat khas aturan upstream yang tersalin ke dokumen proyek |

**Amandemen ADR-0006.** ADR-0006 tetap `ACCEPTED` untuk keputusan intinya (mode **`during`**), tetapi
**butir 2 konsekuensinya** ("Filter yang tersedia adalah `antislop.md` core saja: `skills/antislop-*/SKILL.md`
belum ada di repo dan agen dilarang mengunduhnya") **tidak lagi benar** dan digantikan oleh keputusan di
atas. Isi ADR-0006 tidak disunting, sesuai aturan `docs/adr/README.md` §2 butir 4; perubahan dicatat
lewat ADR baru ini dan baris indeksnya diberi penanda.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Hanya memakai core, tanpa skill | Justru inilah keadaan yang melahirkan **C-064**: entry file menjanjikan lima skill yang tidak ada. User juga meminta skill-nya diterapkan dengan benar, dan skill-lah yang memperluas core ke bidang kerja tertentu. Core tetap lengkap tanpa skill, tetapi membiarkan pointer yang salah lebih buruk daripada memasangnya |
| Membiarkan `AGENTS.md` menyebut skill yang tidak ada | Melanggar aturan yang sudah tertulis di dokumen itu sendiri (jangan menuliskan yang tidak benar) dan akan menyesatkan agen berikutnya |
| Mengunduh dari `main` | Rilis dapat berubah kapan saja tanpa jejak; `sha256` yang dicatat akan basi dan pemeriksa tidak dapat membedakan "diperbarui" dari "menyimpang". Pin ke tag |
| Menyalin berkas skill lalu menyuntingnya agar cocok dengan proyek | Skill yang disunting bukan lagi skill pihak ketiga: provenans hilang, dan setiap pembaruan upstream menjadi penggabungan manual. Keputusan proyek yang menyimpang ditulis di dokumen proyek (`01-AGENT-WORKFRAME.md` §3.2), bukan di dalam berkas salinan |
| Menyimpan seluruh repo upstream (`guide.md`, `rules/`, `contrast-mcp.py`, berkas plugin) | Semuanya alat distribusi, bukan aturan. Menambah berkas yang tidak dibaca siapa pun dan mengaburkan mana yang mengikat |
| Menyunting isi ADR-0006 supaya butir 2 sesuai kenyataan | `docs/adr/README.md` §2 butir 4 melarang menyunting ADR yang sudah `ACCEPTED`; perubahan lewat ADR baru membuat jejaknya terbaca |
| Membiarkan salinan Gate dan daftar aturan di dokumen desain | Terbukti menyimpang dalam satu sesi (**C-065**). Salinan yang tidak diperiksa mesin selalu kalah dari sumbernya |
| Menjadikan `check-antislop-refs.sh` dapat mengunduh untuk "menyegarkan" | Bertabrakan langsung dengan aturan inti: agen yang mengunduh saat berjalan mengambil prompt berikutnya sendiri. Pemeriksa hanya membandingkan dengan tabel `sha256` yang dipegang manusia |

## Konsekuensi

- Positif: entry file (`AGENTS.md`) kini **benar**; agen mana pun dapat membaca skill yang sesuai
  dengan pekerjaannya tanpa mencari sumber lain.
- Positif: hanya ada **satu** core di repo, dan salinan di `skills/antislop/SKILL.md` diperiksa
  **identik** dengan `antislop.md` (setelah frontmatter dibuang), sehingga dua versi core tidak dapat
  lahir kembali tanpa suara.
- Positif: `sha256` yang dicatat membuat keberatan "ini benar-benar berkas upstream" dapat diperiksa
  siapa pun, dan menaikkan versi menjadi langkah sadar, bukan efek samping.
- Positif: kontras token proyek dapat diperiksa ulang dengan alat **upstream**, bukan hanya test sendiri.
  `python3 skills/antislop-human/contrast-check.py` dijalankan atas enam pasangan yang angkanya sudah
  dikomentari di `tokens.css`, dan hasilnya **sama persis**: `#63696f` di atas `#f6f5f2` = **5,09**, `#0e5b63` di atas `#f6f5f2` = **7,15**,
  `#ecedef` di atas `#14161a` = **15,46**, `#9aa1a8` di atas `#14161a` = **6,93**, `#7fd1d9` di atas `#14161a` = **10,37**, `#7a828a` di atas `#14161a` = **4,65** — semuanya `PASS`.
  `--selftest` alat itu juga lulus (`8 reference pairs OK`), sehingga tabel acuan di dalamnya dihitung
  ulang dari rumus, bukan dipercaya.
- Negatif / risiko: repo kini memuat berkas pihak ketiga yang harus diperbarui manual. Mitigasi:
  pembaruannya satu perintah per berkas (§2 `skills/README.md`) dan **tidak wajib** setiap saat;
  memakai versi lama yang konsisten lebih baik daripada campuran.
- Negatif / risiko: core memuat satu berkas ~681 baris dan lima skill tambahan, sehingga ada biaya
  konteks untuk membacanya. Mitigasi: `AGENTS.md` menunjuk skill **per jenis pekerjaan**, jadi sesi
  backend tidak membacanya sama sekali.
- Risiko yang diterima: keputusan proyek yang menyimpang dari core (mis. ambang kontras **WCAG 2.2 AA**
  yang lebih ketat) hidup di `01-AGENT-WORKFRAME.md` §3.2 sebagai **keputusan proyek**, bukan sebagai
  teks aturan. Pembaca harus membuka dua tempat, dan itu memang disengaja: lebih baik daripada dua
  daftar aturan yang saling menyimpang.

## Bukti / Referensi

- Berkas terpasang beserta `sha256` dan cara memperbarui: `skills/README.md` §1/§2.
- `bash scripts/check-antislop-refs.sh` → `antislop-refs OK — 38 aturan (R-01..R-38), 93 rujukan, 7 berkas skill, 7 pemeriksaan`.
- Gigi pemeriksa dibuktikan dengan **enam cacat disuntikkan sementara**, semuanya tertangkap: (1) satu
  rujukan ke nomor aturan yang tidak ada, (2) satu path skill hantu, (3) `sha256` berkas skill UI yang
  diubah, (4) salinan core yang diubah, (5) kalimat Gate yang disalin ke `01-AGENT-WORKFRAME.md`, dan
  (6) satu klaim rentang aturan yang ujungnya berhenti sebelum aturan terakhir — lalu semuanya
  dipulihkan dan pemeriksa kembali hijau. Cacat (1) dan (6) **sengaja ditulis tanpa tokennya di
  dokumen ini**, karena pemeriksa membaca token semacam itu sebagai klaim, bukan sebagai contoh: itu
  pelajaran yang sama dengan §6.3 protokol progress tentang pasangan izin yang tidak ada.
- Pemeriksa kontras upstream dijalankan atas token proyek (10 pasangan, kedua tema) dan angkanya sama
  dengan `tokens.css`.
- Temuan terkait: **C-064** (skill didaftarkan tanpa ada di disk) dan **C-065** (core menyimpang dari
  salinan Gate-nya sendiri; daftar aturan disalin ke dokumen desain).
- Log sesi: `docs/progress/prompts/P-042-2026-09-22-skill-antislop-terpasang-dan-dipin.md`.
