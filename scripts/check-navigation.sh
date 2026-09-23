#!/usr/bin/env bash
# check-navigation.sh — tegakkan batas sidebar (`51-UX.md` §2.1) pada model navigasi.
#
# Cara pakai (dari root repo):
#   bash scripts/check-navigation.sh
#
# Kenapa ada: aturan §2.1 adalah **"sidebar memuat modul saja"** — sub-navigasi
# (tab dan penyaring) hidup di halamannya sendiri. Aturan itu tidak dijaga apa pun
# di CI: test klien tidak berjalan di sana (tidak ada job frontend), dan
# pemeriksaan di peramban (`scripts/responsive-evidence.mjs`) hanya mencari item
# menu yang membawa **kueri** penyaring — ia tidak melihat path bersarang. Jadi
# `Reports > Audit` dapat berdiri sebagai entri sidebar untuk sebuah halaman anak
# tanpa satu pun pemeriksa menyebutnya, sementara diagram `51-UX.md` §2 menggambar
# tujuh butir dan prosa §2.1 mengatakan "tanpa sub-item". Kelas cacat: C-078
# (pemeriksa yang memeriksa lebih sedikit daripada yang tampak), C-079.
#
# Dua arah yang diperiksa, keduanya terhadap **sumbernya**:
#   1. Model (`frontend/src/config/navigation.ts`): tiap `path` bebas kueri;
#      entri sidebar maksimal satu segmen; halaman anak punya induk yang ada dan
#      letaknya di bawah induk itu; tidak ada path/label ganda.
#   2. Dokumen (`docs/design/51-UX.md` §2.1): himpunan menu sidebar sama dengan
#      himpunan baris tabel yang **bukan** `X > Y`, dan himpunan halaman anak
#      sama dengan baris yang berbentuk `X > Y` beserta induknya. Menambah menu
#      di kode tanpa baris di dokumen (atau sebaliknya) karena itu gagal.
#
# Batas yang disadari:
#   1. Ia membaca **teks** berkas TypeScript dan tabel Markdown, bukan mengimpor
#      modelnya: skrip pemeriksa di repo ini berjalan tanpa dependensi, dan
#      mengimpor `.ts` menuntut runner tambahan. Bentuk yang tidak dikenali
#      karena itu **gagal**, bukan dilewati — lihat butir penjaga di bawah.
#   2. Ia tidak memeriksa rute yang tidak punya entri di kedua daftar (mis. halaman
#      detail `/projects/:id`), karena halaman itu memang bukan halaman bernavigasi.
#
# Exit code: 1 bila ada FAIL, 0 bila seluruh batasan terpenuhi.

set -u

ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

NAV=frontend/src/config/navigation.ts
UX=docs/design/51-UX.md

fail=0
fail_msg() { printf 'FAIL  %s\n' "$1"; fail=$((fail + 1)); }
info() { printf 'info  %s\n' "$1"; }

for f in "$NAV" "$UX"; do
  [ -f "$f" ] || { printf 'FAIL  berkas yang dibutuhkan tidak ada: %s\n' "$f"; exit 1; }
done

# ---------------------------------------------------------------------------
# 1. Model navigasi
# ---------------------------------------------------------------------------
#
# Format yang dibaca: `export const <section>` diikuti objek-objek dengan
# `label:`, `path:`, dan (untuk halaman anak) `parent:` pada barisnya sendiri.
# Objek dianggap selesai pada baris `  },`.
model=$(awk '
  /^export const navigation/ { sec = "navigation"; next }
  /^export const subPages/    { sec = "subPages";    next }
  !sec { next }
  /^  \},/ {
    if (label != "" || path != "") print sec "\t" label "\t" path "\t" parent
    label = ""; path = ""; parent = ""; next
  }
  /^ *label: "/  { label  = $0; sub(/^ *label: "/,  "", label);  sub(/",?[[:space:]]*$/, "", label);  next }
  /^ *path: "/   { path   = $0; sub(/^ *path: "/,   "", path);   sub(/",?[[:space:]]*$/, "", path);   next }
  /^ *parent: "/ { parent = $0; sub(/^ *parent: "/, "", parent); sub(/",?[[:space:]]*$/, "", parent); next }
' "$NAV")

sidebar_paths=()
sidebar_labels=()
child_paths=()
child_parents=()
child_labels=()

while IFS=$'\t' read -r sec label path parent; do
  [ -n "${sec:-}" ] || continue
  if [ "$sec" = "navigation" ]; then
    sidebar_paths+=("$path")
    sidebar_labels+=("$label")
  else
    child_paths+=("$path")
    child_labels+=("$label")
    child_parents+=("$parent")
  fi
done <<< "$model"

# Penjaga bentuk: parser yang tidak mengenali berkasnya harus **berhenti di situ**,
# bukan meneruskan pemeriksaan dengan nilai kosong. Versi pertama skrip ini hanya
# menghitung jumlah menu, sehingga pada berkas yang kunci `label:`-nya berganti nama
# ia melaporkan 25 kegagalan yang semuanya menyesatkan ("path \"\" tidak diawali
# /" dan seterusnya) alih-alih satu sebab. Dua hal yang dijaga di sini: hasilnya
# tak boleh hampa, dan tak boleh ada objek tanpa label atau path.
# (Kelas C-055/C-075/C-078/C-080 — pemeriksa yang memeriksa lebih sedikit daripada
# yang tampak, atau berhenti memeriksa tanpa mengatakannya.)
if [ "${#sidebar_paths[@]}" -lt 5 ]; then
  fail_msg "$NAV: hanya ${#sidebar_paths[@]} menu sidebar terbaca — format berkasnya berubah dan parser ini tidak mengenalinya"
  printf 'FAIL  check-navigation: %d pelanggaran batas sidebar (parser berhenti lebih awal)\n' "$fail"
  exit 1
fi
unparsed=0
for i in "${!sidebar_paths[@]}"; do
  [ -n "${sidebar_labels[$i]}" ] && [ -n "${sidebar_paths[$i]}" ] || unparsed=$((unparsed + 1))
done
for i in "${!child_paths[@]}"; do
  [ -n "${child_labels[$i]}" ] && [ -n "${child_paths[$i]}" ] && [ -n "${child_parents[$i]}" ] \
    || unparsed=$((unparsed + 1))
done
if [ "$unparsed" -gt 0 ]; then
  fail_msg "$NAV: $unparsed objek tanpa label/path/induk yang terbaca — bentuk berkasnya berubah dan parser ini tidak mengenalinya; perbarui parsernya, jangan nilai hasil di bawahnya"
  printf 'FAIL  check-navigation: %d pelanggaran batas sidebar (parser berhenti lebih awal)\n' "$fail"
  exit 1
fi
if [ "${#child_paths[@]}" -eq 0 ]; then
  info "$NAV: belum ada halaman anak sama sekali (bukan kegagalan; aturannya tetap diperiksa)"
fi

has_segment() { # 1: nilai, 2: nilai, ... ; benar bila ada yang berisi
  local needle="$1"; shift
  local item
  for item in "$@"; do [ "$item" = "$needle" ] && return 0; done
  return 1
}

# Label sebuah path, diambil dari daftar mana pun yang memuatnya. Ditulis dengan
# penelusuran biasa, bukan nameref `local -n`, supaya skrip ini jalan di bash 3.2
# bawaan macOS seperti skrip pemeriksa lain di repo ini.
value_of() { # 1: path
  local needle="$1" i
  for i in "${!sidebar_paths[@]}"; do
    [ "$needle" = "${sidebar_paths[$i]}" ] && { printf '%s' "${sidebar_labels[$i]}"; return 0; }
  done
  for i in "${!child_paths[@]}"; do
    [ "$needle" = "${child_paths[$i]}" ] && { printf '%s' "${child_labels[$i]}"; return 0; }
  done
  return 1
}

# 1a. Tidak ada kueri/fragmen di path mana pun: penyaring adalah keadaan halaman,
#     bukan alamat halaman.
check_path_shape() {
  case "$1" in
    *"?"*|*"#"*) fail_msg "path \"$1\" membawa kueri/karakter fragmen — penyaring halaman bukan alamat halaman (§2.1)" ;;
  esac
  case "$1" in
    /*) ;;
    *) fail_msg "path \"$1\" tidak diawali \"/\"" ;;
  esac
}
for p in ${sidebar_paths[@]+"${sidebar_paths[@]}"}; do check_path_shape "$p"; done
for p in ${child_paths[@]+"${child_paths[@]}"}; do check_path_shape "$p"; done

# 1b. Entri sidebar = modul: maksimal satu segmen (`/`, `/projects`).
for p in "${sidebar_paths[@]}"; do
  segments=$(printf '%s' "$p" | tr -cd '/' | wc -c | tr -d ' ')
  [ "$segments" -le 1 ] || fail_msg "menu \"$p\" menunjuk sub-halaman — sidebar memuat modul saja (§2.1); daftarkan halaman anak di subPages dengan induk \"$(printf '%s' "$p" | cut -d/ -f1-2)\""
done

# 1c. Halaman anak: induknya ada, dan pathnya benar-benar di bawah induknya.
for i in "${!child_paths[@]}"; do
  p="${child_paths[$i]}"
  parent="${child_parents[$i]}"
  if [ -z "$parent" ]; then
    fail_msg "halaman anak \"$p\" tidak menyebut induknya (parent:)"
    continue
  fi
  if ! has_segment "$parent" ${sidebar_paths[@]+"${sidebar_paths[@]}"}; then
    fail_msg "halaman anak \"$p\" menyebut induk \"$parent\" yang tidak ada di sidebar"
    continue
  fi
  case "$p" in
    "$parent"/*) ;;
    *) fail_msg "halaman anak \"$p\" tidak berada di bawah induknya \"$parent\"" ;;
  esac
done

# 1d. Tidak ada path atau label yang dipakai dua kali.
check_duplicates() { # 1: label pemeriksaan, 2..: nilai
  local what="$1"; shift
  local seen="" item
  for item in "$@"; do
    [ -n "$item" ] || continue
    if has_segment "$item" ${seen[@]+"${seen[@]}"}; then
      fail_msg "$what \"$item\" dipakai lebih dari satu kali"
    fi
    seen+=("$item")
  done
}

check_duplicates "path menu" ${sidebar_paths[@]+"${sidebar_paths[@]}"} ${child_paths[@]+"${child_paths[@]}"}
check_duplicates "label menu" ${sidebar_labels[@]+"${sidebar_labels[@]}"}
check_duplicates "path halaman anak" ${child_paths[@]+"${child_paths[@]}"}

# 1e. Label sidebar tidak memakai gaya remah (`Reports > Audit`): label seperti itu
#     adalah tanda entri menu yang sebenarnya sebuah halaman anak.
for l in "${sidebar_labels[@]}"; do
  case "$l" in
    *">"*) fail_msg "label menu \"$l\" memakai gaya remah — halaman anak bukan entri sidebar (§2.1)" ;;
  esac
done

# ---------------------------------------------------------------------------
# 2. Dokumen `51-UX.md` §2.1
# ---------------------------------------------------------------------------

# Isi §2.1 sampai heading berikutnya, lalu baris tabelnya.
section=$(awk '/^### 2\.1 /{f=1; next} f && /^#/{exit} f {print}' "$UX")

# Aturan itu sendiri harus masih tertulis; kalau teksnya dilunakkan, pemeriksa ini
# juga harus diubah dengan sadar (bukan ikut mengendur tanpa ada yang tahu).
case "$section" in
  *"**modul saja**"*) ;;
  *) fail_msg "$UX: §2.1 tidak lagi menyatakan \"modul saja\" — aturan yang dijaga pemeriksa ini berubah; perbarui keduanya bersama" ;;
esac

doc_rows=$(printf '%s\n' "$section" | awk -F'|' '
  /^[[:space:]]*\|/ {
    label = $2; gsub(/^[[:space:]]+|[[:space:]]+$/, "", label)
    if (label == "" || label == "Menu" || label ~ /^-+$/) next
    print label
  }
')

doc_modules=()
doc_children=()
while IFS= read -r label; do
  [ -n "$label" ] || continue
  case "$label" in
    *">"*)
      doc_parent=$(printf '%s' "$label" | sed 's/>.*//' | sed 's/[[:space:]]*$//')
      doc_child=$(printf '%s' "$label" | sed 's/.*>//' | sed 's/^[[:space:]]*//')
      doc_children+=("$doc_parent>$doc_child")
      doc_modules+=("$doc_parent")
      ;;
    *) doc_modules+=("$label") ;;
  esac
done <<< "$doc_rows"

if [ "${#doc_modules[@]}" -lt 5 ]; then
  fail_msg "$UX: hanya ${#doc_modules[@]} baris menu terbaca di §2.1 — format tabelnya berubah dan parser ini tidak mengenalinya"
fi

# 2a. Himpunan menu di dokumen = himpunan menu di model, dua arah.
for l in "${doc_modules[@]}"; do
  # Baris `X > Y` menyumbang `X` ke daftar modul; ia sah walau tidak ada menu
  # terpisah, karena halaman anaknya yang berdiri sendiri.
  has_segment "$l" ${sidebar_labels[@]+"${sidebar_labels[@]}"} \
    || fail_msg "$UX §2.1 menyebut menu \"$l\" yang tidak ada di $NAV"
done
for l in "${sidebar_labels[@]}"; do
  has_segment "$l" ${doc_modules[@]+"${doc_modules[@]}"} \
    || fail_msg "$NAV memuat menu \"$l\" yang tidak punya baris di $UX §2.1"
done

# 2b. Himpunan halaman anak = baris `X > Y`, beserta induknya.
for pair in ${doc_children[@]+"${doc_children[@]}"}; do
  doc_parent="${pair%%>*}"
  doc_child="${pair##*>}"
  found=0
  for i in "${!child_labels[@]}"; do
    [ "${child_labels[$i]}" = "$doc_child" ] || continue
    parent_path="${child_parents[$i]}"
    parent_label=$(value_of "$parent_path" sidebar_paths || true)
    if [ "$parent_label" != "$doc_parent" ]; then
      fail_msg "$UX §2.1 menempatkan \"$doc_child\" di bawah \"$doc_parent\", tetapi model menaruhnya di bawah \"${parent_label:-?}\" ($parent_path)"
    fi
    found=1
  done
  [ "$found" -eq 1 ] || fail_msg "$UX §2.1 menyebut halaman anak \"$doc_parent > $doc_child\" yang tidak ada di subPages"
done
for i in "${!child_labels[@]}"; do
  l="${child_labels[$i]}"
  ok=0
  for pair in ${doc_children[@]+"${doc_children[@]}"}; do
    [ "${pair##*>}" = "$l" ] && ok=1
  done
  [ "$ok" -eq 1 ] || fail_msg "${child_paths[$i]} tidak punya baris di $UX §2.1"
done

# ---------------------------------------------------------------------------

if [ "$fail" -gt 0 ]; then
  printf 'FAIL  check-navigation: %d pelanggaran batas sidebar\n' "$fail"
  exit 1
fi

printf 'navigation OK — %d menu sidebar (tanpa kueri, satu segmen), %d halaman anak ber-induk, %d baris §2.1 cocok\n' \
  "${#sidebar_paths[@]}" "${#child_paths[@]}" "${#doc_modules[@]}"
