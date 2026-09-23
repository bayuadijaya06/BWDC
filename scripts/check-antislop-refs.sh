#!/usr/bin/env bash
# check-antislop-refs.sh — memeriksa rujukan antislop terhadap sumbernya, tanpa jaringan.
#
# Cara pakai (dari root repo):
#   bash scripts/check-antislop-refs.sh
#
# Kenapa ada: `antislop.md` (core) adalah **sumber tunggal** daftar aturan, tier, dan Delivery
# Gate. Sebelum P-042 dua hal menyimpang tanpa ada yang menangkapnya (kelas cacat C-064/C-065):
# `AGENTS.md` mendaftarkan lima skill yang **tidak ada** di disk, dan berkas core di root ternyata
# **varian lama** yang berbeda dari yang dipegang salinan Gate di dokumen desain. Keduanya cacat
# yang hanya terlihat bila rujukan diperiksa mesin, bukan dibaca.
#
# Tujuh aturan:
#   1. Core ada dan daftar aturannya terbaca: `#### R-XX —` harus ada, berjumlah >= 30, dan
#      berurutan tanpa lubang. Skrip berhenti bila daftar itu tidak terbaca (tidak boleh lulus
#      secara hampa).
#   2. Setiap nomor aturan `R-XX` yang dirujuk dokumen proyek wajib ada di daftar itu.
#   3. Setiap path `skills/...` yang disebut `AGENTS.md` wajib ada di disk (kelas C-064), dan
#      setiap `skills/<nama>/SKILL.md` di disk wajib disebut di `AGENTS.md` atau `skills/README.md`.
#   4. `sha256` setiap berkas antislop wajib sama dengan tabel provenans di `skills/README.md`
#      (kelas C-065: berkas salinan menyimpang dari sumbernya diam-diam).
#   5. Isi `skills/antislop/SKILL.md` (setelah frontmatter dan baris pemisah `---` dibuang) wajib
#      **identik** dengan `antislop.md`, supaya tidak ada dua versi core.
#   6. Rentang aturan yang ditulis dokumen (`R-01..R-38`) wajib cocok dengan yang benar-benar ada:
#      menaikkan aturan upstream tanpa memperbarui klaim angka = gagal.
#   7. Tidak ada kalimat khas aturan/Gate upstream yang boleh muncul di dokumen proyek. Aturan
#      disalin dengan **menunjuk** (pola C-014), bukan dengan mengutip; kalimat di daftar sidik
#      jari di bawah adalah yang terbukti pernah tersalin.
#   8. R-02 pada teks yang dibaca pengguna: `frontend/src` tidak boleh memuat em dash di luar
#      komentar. Keputusan proyeknya di `01-AGENT-WORKFRAME.md` §3.2. Diperiksa mesin karena
#      em dash paling mudah masuk lewat teks pengganti seperti `?? "-"` dan kalimat penjelas
#      di layar, yang tidak pernah terlihat pada test.
#
# Batas yang disadari:
#   1. Yang diperiksa hanya **nomor** aturan dan **keberadaan serta keaslian berkas**. Apakah
#      keputusan proyek benar terhadap aturan itu tidak dapat diperiksa mesin; itu tugas Delivery
#      Gate dan tinjauan user.
#   2. Dokumen yang dipindai adalah `docs/**/*.md` dan berkas `*.md` di root selain `antislop.md`.
#      `skills/` dikecualikan karena isinya memang salinan upstream, dan `.freebuff/` dikecualikan
#      karena itu berkas jalannya agen, bukan dokumen proyek.
#   3. Pemeriksa ini **tidak** mengunduh apa pun dan tidak boleh diubah untuk mengunduh: aturan
#      upstream menetapkan skill disediakan user, bukan diambil agen (izin eksplisit user dicatat
#      di `skills/README.md` §1 dan ADR-0025).
#
# Kalau pemeriksa ini gagal: perbaiki dokumen, berkas skill, atau tabel provenansnya. **Jangan**
# melunakkan skripnya dan jangan memperbarui `sha256` tanpa benar-benar mengambil berkas baru dari
# tag rilis yang dicatat.
#
# Exit code: 1 bila ada FAIL, 0 bila semua rujukan sah.

set -u

ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

CORE=antislop.md
AGENTS=AGENTS.md
SKILLS_README=skills/README.md
SKILL_CORE=skills/antislop/SKILL.md

fail=0
checks=0

fail_msg() { printf 'FAIL  %s\n' "$1"; fail=$((fail + 1)); }
info() { printf 'info  %s\n' "$1"; }
ok() { checks=$((checks + 1)); }

for f in "$CORE" "$AGENTS" "$SKILLS_README" "$SKILL_CORE"; do
  [ -f "$f" ] || { printf 'FAIL  berkas yang dibutuhkan tidak ada: %s\n' "$f"; exit 1; }
done

# ---------------------------------------------------------------------------
# 1. Daftar aturan dari core
# ---------------------------------------------------------------------------

rules=$(grep -oE '^#### R-[0-9]{2}' "$CORE" | sed -E 's/^#### //' | sort -u)
rule_count=$(printf '%s\n' "$rules" | grep -c .)

if [ "$rule_count" -lt 30 ]; then
  printf 'FAIL  daftar aturan tidak terbaca di %s (%s aturan) — pola heading `#### R-XX —` mungkin berubah\n' \
    "$CORE" "$rule_count"
  exit 1
fi
info "daftar aturan: $rule_count aturan <- $CORE"

rule_min=$(printf '%s\n' "$rules" | head -1)
rule_max=$(printf '%s\n' "$rules" | tail -1)

# Berurutan tanpa lubang: R-01..R-NN
expected=$(printf '%s\n' "$rules" | awk 'BEGIN{ok=1} {n=substr($0,3)+0; want=NR; if (n!=want) ok=0} END{print ok}')
[ "$expected" = "1" ] || fail_msg "$CORE: nomor aturan tidak berurutan tanpa lubang ($rule_min..$rule_max)"

in_rules() { printf '%s\n' "$rules" | grep -qxF "$1"; }

# ---------------------------------------------------------------------------
# 2. Setiap R-XX yang dirujuk dokumen proyek ada di daftar
# ---------------------------------------------------------------------------

doc_files=$(ls *.md 2>/dev/null | grep -v '^antislop.md$'; find docs -type f -name '*.md' 2>/dev/null)

ref_re='(^|[^A-Za-z0-9-])R-[0-9]{2}([^0-9]|$)'
refs=$(
  for f in $doc_files; do
    [ -f "$f" ] || continue
    grep -nE "$ref_re" "$f" 2>/dev/null | while IFS= read -r row; do
      ln=${row%%:*}
      body=${row#*:}
      printf '%s\n' "$body" | grep -oE "$ref_re" | sed -E 's/.*(R-[0-9]{2}).*/\1/' | while IFS= read -r r; do
        printf '%s:%s:%s\n' "$f" "$ln" "$r"
      done
    done
  done
)
ref_count=$(printf '%s\n' "$refs" | grep -c .)

[ "$ref_count" -gt 0 ] || fail_msg "tidak ada satu pun rujukan R-XX yang terbaca di dokumen proyek — pola rujukan mungkin berubah"

while IFS=: read -r f ln r; do
  [ -n "${r:-}" ] || continue
  if ! in_rules "$r"; then
    fail_msg "$f:$ln merujuk aturan yang tidak ada di $CORE: $r"
  fi
done <<EOF
$refs
EOF
ok
info "rujukan aturan: $ref_count rujukan di $(printf '%s\n' "$refs" | cut -d: -f1 | sort -u | grep -c .) berkas"

# ---------------------------------------------------------------------------
# 3. Path skill: yang disebut AGENTS.md harus ada; yang ada harus disebut
# ---------------------------------------------------------------------------

declared_skills=$(
  sed -n '/<!-- antislop:start -->/,/<!-- antislop:end -->/p' "$AGENTS" |
    grep -oE 'skills/[A-Za-z0-9._/-]+' | sed -E 's/[.,)]$//' | sort -u
)
declared_count=$(printf '%s\n' "$declared_skills" | grep -c .)
[ "$declared_count" -gt 0 ] || fail_msg "$AGENTS: tidak ada path skill yang terbaca di blok antislop (kelas C-064)"

for p in $declared_skills; do
  [ -e "$p" ] || fail_msg "$AGENTS menyebut skill yang tidak ada di disk: $p"
done
ok

on_disk=$(find skills -type f -name 'SKILL.md' 2>/dev/null | sort)
for p in $on_disk; do
  named=0
  grep -qF "$p" "$AGENTS" "$SKILLS_README" 2>/dev/null && named=1
  dir=${p%/SKILL.md}
  grep -qF "$dir" "$AGENTS" 2>/dev/null && named=1
  [ "$named" = "1" ] || fail_msg "skill di disk tidak terdaftar di $AGENTS maupun $SKILLS_README: $p"
done
ok
info "path skill: $declared_count disebut di $AGENTS, $(printf '%s\n' "$on_disk" | grep -c .) SKILL.md di disk"

# ---------------------------------------------------------------------------
# 4. sha256 berkas antislop sama dengan tabel provenans
# ---------------------------------------------------------------------------

hash_rows=$(grep -nE '^\| `[^`]+`' "$SKILLS_README" | grep -E '[0-9a-f]{64}')
hash_count=$(printf '%s\n' "$hash_rows" | grep -c .)
[ "$hash_count" -gt 0 ] || fail_msg "$SKILLS_README: tabel provenans sha256 tidak terbaca"

while IFS= read -r row; do
  [ -n "$row" ] || continue
  ln=${row%%:*}
  body=${row#*:}
  path=$(printf '%s\n' "$body" | sed -E 's/^\| `([^`]+)`.*/\1/')
  want=$(printf '%s\n' "$body" | grep -oE '[0-9a-f]{64}' | head -1)
  if [ ! -f "$path" ]; then
    fail_msg "$SKILLS_README:$ln mencatat sha256 untuk berkas yang tidak ada: $path"
    continue
  fi
  got=$(shasum -a 256 "$path" | awk '{print $1}')
  if [ "$got" != "$want" ]; then
    fail_msg "$path sha256 berbeda dari tabel: tercatat ${want:0:12}…, di disk ${got:0:12}…"
  fi
done <<EOF
$hash_rows
EOF
ok
info "sha256: $hash_count berkas antislop diverifikasi terhadap $SKILLS_README §1"

# ---------------------------------------------------------------------------
# 5. Salinan core di skills/antislop tidak boleh menyimpang dari core di root
# ---------------------------------------------------------------------------

norm_body() { awk '/^---$/{n++; next} n>=2' "$1" | grep -v '^---$'; }
norm_core() { grep -v '^---$' "$1"; }

if cmp -s <(norm_body "$SKILL_CORE") <(norm_core "$CORE"); then
  :
else
  fail_msg "$SKILL_CORE berbeda dari $CORE (setelah frontmatter dibuang) — dua versi core beredar"
fi
ok

# ---------------------------------------------------------------------------
# 6. Rentang aturan yang diklaim dokumen cocok dengan yang ada
# ---------------------------------------------------------------------------

ranges=$(
  for f in $doc_files; do
    [ -f "$f" ] || continue
    grep -nE 'R-[0-9]{2}\.\.R-[0-9]{2}' "$f" 2>/dev/null | while IFS= read -r row; do
      ln=${row%%:*}
      body=${row#*:}
      printf '%s\n' "$body" | grep -oE 'R-[0-9]{2}\.\.R-[0-9]{2}' | while IFS= read -r rng; do
        printf '%s:%s:%s\n' "$f" "$ln" "$rng"
      done
    done
  done
)
range_count=$(printf '%s\n' "$ranges" | grep -c .)

while IFS=: read -r f ln rng; do
  [ -n "${rng:-}" ] || continue
  lo=${rng%%..*}
  hi=${rng##*..}
  if ! in_rules "$lo" || ! in_rules "$hi"; then
    fail_msg "$f:$ln mengklaim rentang aturan $rng yang ujungnya tidak ada di $CORE"
  elif [ "$lo" != "$rule_min" ] || [ "$hi" != "$rule_max" ]; then
    fail_msg "$f:$ln mengklaim rentang $rng, padahal daftar di $CORE adalah $rule_min..$rule_max"
  fi
done <<EOF
$ranges
EOF
ok
[ "$range_count" -eq 0 ] && info "rentang aturan: tidak ada klaim rentang eksplisit"
[ "$range_count" -gt 0 ] && info "rentang aturan: $range_count klaim rentang diperiksa"

# ---------------------------------------------------------------------------
# 7. Sidik jari aturan upstream tidak boleh tersalin ke dokumen proyek
# ---------------------------------------------------------------------------

fingerprints=(
  'Is there an em dash'
  'backed by concrete evidence'
  'Do not start until they answer'
  'First-Run Install Wizard'
  'This is a diagnostic scan, not a ban list'
)

for fp in "${fingerprints[@]}"; do
  hits=$(
    for f in $doc_files $AGENTS; do
      [ -f "$f" ] || continue
      grep -nF "$fp" "$f" 2>/dev/null | while IFS= read -r row; do
        printf '%s:%s\n' "$f" "${row%%:*}"
      done
    done
  )
  if [ -n "$hits" ]; then
    while IFS= read -r h; do
      fail_msg "$h menyalin kalimat aturan upstream (\"$fp\") — tunjuk ke $CORE, jangan mengutip"
    done <<EOF
$hits
EOF
  fi
done
ok
info "sidik jari aturan: ${#fingerprints[@]} kalimat khas diperiksa tidak tersalin"

# ---------------------------------------------------------------------------
# 8. R-02: teks yang dibaca pengguna bebas em dash (tanpa komentar kode)
# ---------------------------------------------------------------------------

ui_dir=frontend/src
ui_count=0
if [ -d "$ui_dir" ]; then
  # Komentar dibuang lebih dulu: `/* ... */` (mode slurp) lalu `//` yang berdiri
  # sebagai awal komentar (`^` atau didahului spasi), supaya `https://` di dalam
  # string tidak memotong barisnya. Sisa berkas itulah teks yang dibaca pengguna.
  ui_files=$(
    find "$ui_dir" -type f \( -name '*.tsx' -o -name '*.ts' \) \
      ! -name '*.test.tsx' ! -name '*.test.ts' | sort
  )
  for f in $ui_files; do
    ui_count=$((ui_count + 1))
    hits=$(perl -0777 -pe 's{/\*.*?\*/}{}gs' "$f" 2>/dev/null \
      | perl -pe 's{(^|\s)//.*$}{}' 2>/dev/null \
      | grep -n '—' | head -10)
    if [ -n "$hits" ]; then
      while IFS= read -r row; do
        fail_msg "$f:${row%%:*} teks yang dibaca pengguna memuat em dash (R-02)"
      done <<EOF
$hits
EOF
    fi
  done
fi
ok
info "R-02 em dash: $ui_count berkas teks UI diperiksa (komentar dikecualikan)"

# ---------------------------------------------------------------------------

if [ "$fail" -gt 0 ]; then
  printf 'FAIL  antislop-refs: %s masalah\n' "$fail"
  exit 1
fi

printf 'antislop-refs OK — %s aturan (%s..%s), %s rujukan, %s berkas skill, %s pemeriksaan\n' \
  "$rule_count" "$rule_min" "$rule_max" "$ref_count" "$declared_count" "$checks"
exit 0
