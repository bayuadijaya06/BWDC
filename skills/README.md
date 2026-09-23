# skills/ — Skill antislop yang Dipasang di Repositori Ini

Direktori ini memuat **skill antislop dari pihak ketiga**, apa adanya. Isinya bukan tulisan proyek ini:
setiap berkas di sini disalin **byte-identik** dari rilis resminya, dan keasliannya dapat diperiksa
dengan `shasum -a 256` terhadap tabel di bawah.

**Sumber tunggal aturan:** `antislop.md` (core) di root repo. Skill di folder ini **memperluas** core
untuk satu bidang kerja dan selalu merujuk aturan inti **lewat nomor** (R-XX), tidak menyalinnya —
karena itu tidak ada daftar aturan kedua yang perlu dirawat. Daftar aturan, tier, dial, dan Delivery
Gate hanya sah bila berasal dari `antislop.md`; salinan di dokumen proyek dilarang (temuan C-065).

---

## 1. Provenans

| Berkas | Path upstream | sha256 |
|---|---|---|
| `antislop.md` (root repo, bukan folder ini) | `antislop.md` | `5ac257c57acd4ccab24133be263b9541f15cdb652079909661c5e5a60e99e2ee` |
| `skills/antislop/SKILL.md` | `skills/antislop/SKILL.md` | `ccfc99bcad12d6cd313fcd27b8eb8381b7ba60832abe199ce4e862b27ff0af43` |
| `skills/antislop-ui/SKILL.md` | `skills/antislop-ui/SKILL.md` | `47d9892771253f544f7eb2e40af35fd5c31407dcd8c048183bbd1de5a00a4bbe` |
| `skills/antislop-copywriting/SKILL.md` | `skills/antislop-copywriting/SKILL.md` | `2434a6fb0603dea1e62d4c1968382def47af779bd0e43b2130edd49df41ba444` |
| `skills/antislop-human/SKILL.md` | `skills/antislop-human/SKILL.md` | `5bc7c8e56cc79a5154eabf32469fd9f95621ac0b0faa73acd5816d56aafd5868` |
| `skills/antislop-human/contrast-check.py` | `skills/antislop-human/contrast-check.py` | `619cdf3da8b27f87d8e4049b4acecfe1a559b234ab2aa9cf09447c28745c3075` |
| `skills/antislop-layoutmobile/SKILL.md` | `skills/antislop-layoutmobile/SKILL.md` | `8613395a5109f637837f5c24f0ce5e58a96df1c2567a9f7666fcaecc47ff39cd` |
| `skills/antislop-code/SKILL.md` | `skills/antislop-code/SKILL.md` | `004933009d352710dcbea7817101d5c188474a442e3ece26a6e0c18d3223898f` |
| `skills/LICENSE-antislop` | `LICENSE` | `9fd83fb1fda52ca0094b96e2ed5a3d6c42a81b95a6c6326553f5bbe570f443de` |

| Field | Nilai |
|---|---|
| Repositori | `https://github.com/miqdadbadjuber/anti-slop` |
| Rilis yang dipin | tag **`v3.2.12`** (isi `main` pada 2026-09-22 **sama persis** dengan tag ini — diperiksa, bukan diasumsikan) |
| Diunduh pada | 2026-09-22 (sesi P-042) |
| Diunduh oleh | **agen**, atas **izin eksplisit user** — user memilih "Agen unduh langsung dari repo itu sekarang". Aturan inti antislop sendiri menyatakan skill disediakan **user**, bukan diunduh agen; izin itu dicatat karena menyimpang dari aturan itu (ADR-0025) |
| Lisensi | **MIT** (berkas salinan: `LICENSE-antislop`) — menyalin isinya ke repo ini sah selama pemberitahuan hak cipta dipertahankan |

Tidak disalin: `guide.md`, `rules/`, `contrast-mcp.py`, dan berkas plugin per-agent (`.claude-plugin/`,
`.kimi-plugin/`, `.cursor/`). Semuanya alat distribusi, bukan aturan; menyimpannya di sini akan
menambah berkas yang tidak dibaca siapa pun.

## 2. Cara Memperbarui (satu perintah per berkas)

```bash
TAG=v3.2.12   # ganti ke tag rilis baru, JANGAN pakai `main`
BASE="https://raw.githubusercontent.com/miqdadbadjuber/anti-slop/$TAG"
curl -fsSL -o antislop.md                             "$BASE/antislop.md"
curl -fsSL -o skills/antislop/SKILL.md                "$BASE/skills/antislop/SKILL.md"
curl -fsSL -o skills/antislop-ui/SKILL.md             "$BASE/skills/antislop-ui/SKILL.md"
curl -fsSL -o skills/antislop-copywriting/SKILL.md    "$BASE/skills/antislop-copywriting/SKILL.md"
curl -fsSL -o skills/antislop-human/SKILL.md          "$BASE/skills/antislop-human/SKILL.md"
curl -fsSL -o skills/antislop-human/contrast-check.py "$BASE/skills/antislop-human/contrast-check.py"
curl -fsSL -o skills/antislop-layoutmobile/SKILL.md   "$BASE/skills/antislop-layoutmobile/SKILL.md"
curl -fsSL -o skills/antislop-code/SKILL.md           "$BASE/skills/antislop-code/SKILL.md"
curl -fsSL -o skills/LICENSE-antislop                 "$BASE/LICENSE"
shasum -a 256 antislop.md skills/antislop*/SKILL.md skills/antislop-human/contrast-check.py skills/LICENSE-antislop
```

Menaikkan tag berarti **menaikkan semua berkas bersamaan** (`antislop.md` + lima skill + berkas
pendamping), lalu memperbarui tabel §1 di berkas ini dan baris versi di ADR-0025. Aturan upstream:
skill baru tidak boleh tercampur dengan core lama, karena skill merujuk nomor aturan inti.

## 3. Pemeriksa Otomatis

`bash scripts/check-antislop-refs.sh` memeriksa, tanpa jaringan:

1. Setiap nomor aturan `R-XX` yang **dirujuk dokumen** benar-benar ada di `antislop.md` (sumber tunggal).
2. Setiap path skill yang disebut `AGENTS.md` benar-benar ada di disk (kelas cacat C-064: pointer
   block pernah mendaftarkan lima skill yang tidak ada).
3. `sha256` setiap berkas antislop **sama dengan yang dicatat** di tabel §1 di atas (kelas C-065:
   berkas yang disalin diam-diam menyimpang dari sumbernya).
4. Isi `skills/antislop/SKILL.md` (tanpa frontmatter) setara dengan `antislop.md`, sehingga dua
   salinan core tidak dapat berbeda diam-diam.

Berkas ini **diperbarui pada 2026-09-22** karena pemeriksa itu semula menemukan dua cacat nyata:
`antislop.md` di root ternyata **varian lama** (686 baris, tanpa penanda `[ ]` pada Delivery Gate dan
tanpa butir *scope* R-02 yang ditambahkan upstream), dan `AGENTS.md` mendaftarkan lima skill yang tidak
ada. Root `antislop.md` diganti dengan berkas pristine dari tag `v3.2.12` pada sesi yang sama.
