#!/usr/bin/env bash
# check-doc-links.sh — verifikasi bahwa setiap referensi file di dalam dokumen benar-benar ada.
#
# Cara pakai (dari root repo):
#   bash scripts/check-doc-links.sh
#
# Cara kerja:
#   1. Ambil semua path di dalam backtick yang berakhiran ekstensi umum dari semua file .md.
#   2. Uji keberadaan path relatif terhadap dokumen yang menyebutnya, lalu ke root dan folder dokumen.
#   3. Laporkan BROKEN (referensi dokumen hilang), PLANNED (file kode/aset yang memang belum dibuat),
#      dan ABSENT (dokumen yang memang sengaja tidak ada, lihat ABSENT_DOCS).
#   4. Referensi berpola placeholder (<...>, ..., NNNN) dilewati karena bukan path nyata.
#
# Exit code: 1 bila ada BROKEN. PLANNED dan ABSENT tidak dianggap gagal, tetapi harus disadari.

set -u

# Dokumentasi pihak ketiga yang menyebut file konfigurasi tool lain, bukan path proyek ini.
IGNORE_FILES='antislop.md'
# Catatan: ekstensi `.js`/`.jsx`/`.mjs`/`.cjs` ikut diklasifikasikan sebagai berkas kode (PLANNED),
# bukan BROKEN. Sebelum P-037 daftarnya hanya memuat `.ts`/`.tsx`, sehingga setiap penyebutan
# berkas JS di dalam backtick — termasuk yang memang **sengaja tidak ada**, seperti
# `tailwind.config.js` yang dilarang ADR-0024 — dilaporkan BROKEN dan membuat skrip ini gagal
# padahal tidak ada tautan dokumen yang rusak. Kalau EXT_PATTERN memuat sebuah ekstensi,
# ekstensi itu harus punya klasifikasi di sini.
# Dokumen yang **sengaja tidak ada**. Menyebutnya bukan tautan rusak, melainkan keadaan yang memang
# disengaja dan dijelaskan di dokumen yang menyebutnya (contoh: berkas kontribusi terpisah tidak
# dibuat karena aturannya ada di `README.md` §12). Daftar ini harus tetap **pendek**: setiap
# penambahan berarti ada dokumen yang dijanjikan tetapi tidak pernah dibuat.
ABSENT_DOCS='CONTRIBUTING.md guide.md'
# `guide.md` masuk daftar ini pada **P-042**: itu panduan upstream antislop yang **sengaja tidak
# disalin** ke repo (alat distribusi, bukan aturan) dan disebut justru untuk menyatakan ketiadaannya
# (`skills/README.md` §1). Menyebutnya bukan tautan rusak.

# Isi `skills/` selain `README.md`-nya adalah salinan **byte-identik** dari repo pihak ketiga
# (`sha256` diperiksa `scripts/check-antislop-refs.sh`), dan rujukan di dalamnya menunjuk tata letak
# repo **upstream** — mis. `CLAUDE.md`/`GEMINI.md` yang memang tidak ada di sini. Memeriksanya sebagai
# tautan dokumen proyek akan selalu gagal, dan memperbaikinya berarti menyunting berkas yang
# keasliannya dikunci `sha256`. Karena itu hanya `skills/README.md` (tulisan proyek ini) yang
# diperiksa di sini.

EXT_PATTERN='\.(md|go|ts|tsx|js|json|yml|yaml|sh|sql|toml|mod)'

ROOTS=("." "docs/design" "docs/progress" "docs/progress/prompts" "docs/adr")

broken=0
planned=0
absent=0

resolve() {
  local src_dir="$1" ref="$2" r
  for r in "$src_dir/$ref" "$ref" "${ROOTS[@]/%//$ref}"; do
    [ -e "$r" ] && return 0
  done
  return 1
}

is_planned() {
  case "$1" in
    skills/*|anti-slop/*|backend/*|frontend/*|cmd/*|scripts/*|prompts/P-*-...md|NNNN-*.md|docs/...md) return 0 ;;
    *\.go|*\.ts|*\.tsx|*\.js|*\.jsx|*\.mjs|*\.cjs|*\.sh|*\.yml|*\.yaml|*\.sql|*\.json|*\.mod|*\.toml) return 0 ;;
    docker-compose.yml|go.mod|package.json|vite.config.ts|Makefile) return 0 ;;
    *) return 1 ;;
  esac
}

PATTERN='`[A-Za-z0-9_./<>-]+'"$EXT_PATTERN"'`'

is_placeholder() {
  case "$1" in
    *'<'*|*'...'*|NNNN-*) return 0 ;;
    *) return 1 ;;
  esac
}

while IFS= read -r md; do
  src_dir=$(dirname "$md")
  case " $IGNORE_FILES " in *" $(basename "$md") "*) continue ;; esac
  case "$md" in
    ./skills/*|skills/*) [ "${md#./}" = "skills/README.md" ] || continue ;;
  esac
  while IFS= read -r ref; do
    [ -z "$ref" ] && continue
    is_placeholder "$ref" && continue
    resolve "$src_dir" "$ref" && continue
    case " $ABSENT_DOCS " in
      *" $(basename "$ref") "*) echo "ABSENT (sengaja tidak dibuat): $ref   <- $md"; absent=$((absent + 1)); continue ;;
    esac
    if is_planned "$ref"; then
      echo "PLANNED (belum dibuat): $ref   <- $md"
      planned=$((planned + 1))
    else
      echo "BROKEN: $ref   <- $md"
      broken=$((broken + 1))
    fi
  done < <(grep -oE "$PATTERN" "$md" | tr -d '`' | sort -u)
done < <(find . -name '*.md' -not -path './.git/*' -not -path '*/node_modules/*' | sort)

echo "---"
echo "BROKEN referensi dokumen: $broken"
echo "PLANNED (file kode/aset belum dibuat): $planned"
echo "ABSENT (dokumen yang sengaja tidak dibuat): $absent"
[ "$broken" -eq 0 ] || exit 1
