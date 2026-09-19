#!/usr/bin/env bash
# check-doc-links.sh — verifikasi bahwa setiap referensi file di dalam dokumen benar-benar ada.
#
# Cara pakai (dari root repo):
#   bash scripts/check-doc-links.sh
#
# Cara kerja:
#   1. Ambil semua path di dalam backtick yang berakhiran ekstensi umum dari semua file .md.
#   2. Uji keberadaan path relatif terhadap dokumen yang menyebutnya, lalu ke root dan folder dokumen.
#   3. Laporkan BROKEN (referensi dokumen hilang) dan PLANNED (file kode/aset yang memang belum dibuat).
#   4. Referensi berpola placeholder (<...>, ..., NNNN) dilewati karena bukan path nyata.
#
# Exit code: 1 bila ada BROKEN. PLANNED tidak dianggap gagal, tetapi harus disadari.

set -u

# Dokumentasi pihak ketiga yang menyebut file konfigurasi tool lain, bukan path proyek ini.
IGNORE_FILES='antislop.md'
EXT_PATTERN='\.(md|go|ts|tsx|js|json|yml|yaml|sh|sql|toml|mod)'

ROOTS=("." "docs/design" "docs/progress" "docs/progress/prompts" "docs/adr")

broken=0
planned=0

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
    *\.go|*\.ts|*\.tsx|*\.sh|*\.yml|*\.yaml|*\.sql|*\.json|*\.mod|*\.toml) return 0 ;;
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
  while IFS= read -r ref; do
    [ -z "$ref" ] && continue
    is_placeholder "$ref" && continue
    resolve "$src_dir" "$ref" && continue
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
[ "$broken" -eq 0 ] || exit 1
