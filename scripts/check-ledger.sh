#!/usr/bin/env bash
# check-ledger.sh — pemeriksa konsistensi ledger progres (docs/progress/**).
#
# Memeriksa kelas cacat yang pernah lolos ke dokumen, bukan salah ketik biasa:
#
#   C-044 / C-055 / C-057  hitungan audit dan hitungan test yang tidak cocok
#                          dengan berkasnya sendiri (angka disalin dari sesi
#                          sebelumnya, bukan dihitung ulang)
#   C-057                  rujukan test/fungsi yang sudah tidak ada lagi
#   (papan kerja)          satu task muncul di dua kolom status, atau task yang
#                          buktinya sudah "DONE" masih duduk di TODO
#
# Aturan yang diperiksa (lihat docs/design/02-AGENT-PROGRESS-PROTOCOL.md §6):
#
#   1. Setiap laporan audit memuat marker mesin:
#         <!-- audit-summary total=N fixed=N approved=N open=N rejected=N rinci=N -->
#      Angkanya wajib sama dengan tabel tindak lanjut di berkas yang sama, dan
#      marker itu disalin apa adanya ke audits/README.md, AGENTS.md, STATE.md,
#      serta CONTINUE.md. Baris prosa di lima berkas itu yang menyebut
#      `AUDIT-001` **dan** memuat rincian status ikut diperiksa: angka pertama
#      sebelum `FIXED`/`OPEN` harus sama dengan markernya, dan bentuk
#      "a + b + c = d" harus benar-benar berjumlah. (Jumlah temuan di prosa
#      hanya terbaca lewat penjumlahan itu — kalimatnya bebas, jadi tidak
#      diurai; baris bersejarah ditandai `historis`/`saat itu`/`waktu itu`
#      dan sengaja dilewati.)
#   2. Rujukan test bernama (`TestXxx`, atau `path::TestXxx`) pada dokumen status
#      wajib menunjuk test yang benar-benar ada di backend/.
#   3. Hitungan test di STATE.md §3 wajib sama dengan `grep -c '^func Test'`.
#   4. ID yang dirujuk (`T-###`, `C-###`) wajib ada di papan kerja / tabel audit.
#
# Yang TIDAK diperiksa (sengaja): log historis append-only. `SESSION-LOG.md`,
# `CHANGELOG.md`, `docs/progress/prompts/**`, dan catatan bukti di
# `docs/progress/audits/**` mengutip keadaan masa lalu — termasuk nama test yang
# sudah diganti. Baris mana pun dapat dikecualikan dengan menambahkan komentar
# `<!-- ledger-check: skip -->` di baris itu.
#
# Keluar dengan kode 1 bila ada FAIL. Dipanggil `make check-ledger` dan CI.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

QUIET="${QUIET:-0}"
TMP="$(mktemp -d 2>/dev/null || mktemp -d -t ledger)"
trap 'rm -rf "$TMP"' EXIT

# Kegagalan ditulis ke berkas, bukan hanya ke penghitung variabel: sebagian
# pemeriksaan berjalan di dalam pipeline (subshell), dan penghitung variabel
# tidak akan kembali ke shell induk.
FAILS="$TMP/fails"; WARNS="$TMP/warns"
: > "$FAILS"; : > "$WARNS"
fail() { printf '%s\n' "$1" >> "$FAILS"; }
warn() { printf '%s\n' "$1" >> "$WARNS"; }
info() { [ "$QUIET" = "1" ] || printf '%s\n' "$1"; }
n_of() { [ -f "$1" ] && wc -l < "$1" | tr -d ' ' || printf '0'; }
# Laporan dicetak di akhir, tersortir dan tanpa duplikat: satu baris dokumen yang
# mengutip nama yang sama dua kali tidak perlu dua baris FAIL.
show() { while IFS= read -r m; do printf '%-5s %s\n' "$1" "$m"; done < "$2"; }

BACKEND="backend"
AUDIT_DIR="docs/progress/audits"
STATE="docs/progress/STATE.md"
TASKS="docs/progress/TASKS.md"
TRACE="docs/progress/TRACEABILITY.md"
QFILE="docs/progress/OPEN-QUESTIONS.md"

# --------------------------------------------------------------- inventarisasi
# Daftar test yang benar-benar ada di backend: "path<TAB>TestName".
: > "$TMP/tests.tsv"
for f in $(find "$BACKEND" -name '*_test.go' -type f | sort); do
  grep -oE '^func Test[A-Za-z0-9_]+' "$f" 2>/dev/null | sed 's/^func //' | while IFS= read -r t; do
    printf '%s\t%s\n' "$f" "$t"
  done >> "$TMP/tests.tsv"
done
TOTAL_TESTS="$(wc -l < "$TMP/tests.tsv" | tr -d ' ')"

has_test() { awk -F'\t' -v n="$1" '$2 == n { found = 1 } END { exit(found ? 0 : 1) }' "$TMP/tests.tsv"; }

count_tests_in() { grep -cE '^func Test' "$1" 2>/dev/null | tr -d ' '; }

# Menerjemahkan path relatif dokumen ("internal/service/x_test.go") ke path repo.
resolve_test_file() {
  if [ -f "$1" ]; then printf '%s' "$1"; return 0; fi
  if [ -f "$BACKEND/$1" ]; then printf '%s' "$BACKEND/$1"; return 0; fi
  local hits
  hits="$(find "$BACKEND" -name "$(basename "$1")" -type f | wc -l | tr -d ' ')"
  if [ "$hits" = "1" ]; then find "$BACKEND" -name "$(basename "$1")" -type f; return 0; fi
  return 1
}

# ------------------------------------------------------------- 1. hitungan audit
MARKER_REF=""
for A in "$AUDIT_DIR"/AUDIT-*.md; do
  [ -f "$A" ] || continue
  # Marker asli selalu memuat `total=`; sebutan marker di dalam prosa (mis. saat
  # dokumen menjelaskan konvensinya) tidak boleh dianggap marker.
  M="$(grep -oE '<!-- audit-summary [^>]*-->' "$A" | grep 'total=' | head -1 || true)"
  if [ -z "$M" ]; then
    fail "$A: tidak ada marker <!-- audit-summary ... --> (wajib; protokol §6)"
    continue
  fi
  if [ -z "$MARKER_REF" ]; then
    MARKER_REF="$(printf '%s' "$M" | sed 's/<!-- audit-summary //; s/ -->//; s/  */ /g; s/^ //; s/ $//')"
  fi

  val() { printf '%s' "$M" | grep -oE " $1=[0-9]+" | head -1 | cut -d= -f2; }
  for k in total fixed approved open rejected rinci; do
    [ -n "$(val "$k")" ] || fail "$A: marker audit-summary kehilangan kunci '$k'"
  done

  ROWS="$(grep -cE '^\| C-[0-9]+ \|' "$A" | tr -d ' ')"
  HEAD="$(grep -cE '^### C-[0-9]+' "$A" | tr -d ' ')"
  C_FIXED="$(grep -cE '^\| C-[0-9]+ \| \*\*FIXED\*\* \|' "$A" | tr -d ' ')"
  C_APPROVED="$(grep -cE '^\| C-[0-9]+ \| \*\*APPROVED\*\* \|' "$A" | tr -d ' ')"
  C_OPEN="$(grep -cE '^\| C-[0-9]+ \| \*\*OPEN\*\* \|' "$A" | tr -d ' ')"
  C_REJECTED="$(grep -cE '^\| C-[0-9]+ \| \*\*REJECTED\*\* \|' "$A" | tr -d ' ')"
  known=$((C_FIXED + C_APPROVED + C_OPEN + C_REJECTED))

  [ "$(val total)" = "$ROWS" ] || fail "$A: marker total=$(val total), tabel berisi $ROWS baris C-"
  [ "$(val fixed)" = "$C_FIXED" ] || fail "$A: marker fixed=$(val fixed), tabel berisi $C_FIXED baris **FIXED**"
  [ "$(val approved)" = "$C_APPROVED" ] || fail "$A: marker approved=$(val approved), tabel berisi $C_APPROVED baris **APPROVED**"
  [ "$(val open)" = "$C_OPEN" ] || fail "$A: marker open=$(val open), tabel berisi $C_OPEN baris **OPEN**"
  [ "$(val rejected)" = "$C_REJECTED" ] || fail "$A: marker rejected=$(val rejected), tabel berisi $C_REJECTED baris **REJECTED**"
  [ "$(val total)" = "$known" ] || fail "$A: total=$(val total) tidak sama dengan jumlah seluruh status ($known)"
  [ "$(val rinci)" = "$HEAD" ] || fail "$A: marker rinci=$(val rinci), jumlah judul '### C-' ada $HEAD"

  # status yang bukan salah satu dari empat kata itu
  awk -v F="$A" -F'|' '
    /^\| C-[0-9]+ \|/ {
      st = $3; gsub(/^[ \t]+|[ \t]+$/, "", st)
      if (st !~ /^\*\*(FIXED|APPROVED|OPEN|REJECTED)\*\*$/)
        printf "FAIL|%s:%d: status baris audit harus **FIXED**/**APPROVED**/**OPEN**/**REJECTED** (ditemukan: %s)\n", F, NR, st
    }' "$A" | while IFS= read -r m; do fail "${m#FAIL|}"; done

  # id ganda di tabel
  awk -F'|' '/^\| C-[0-9]+ \|/ { gsub(/ /,"",$2); print $2 }' "$A" | sort | uniq -d > "$TMP/dup"
  while IFS= read -r id; do
    [ -n "$id" ] || continue
    fail "$A: id temuan $id muncul lebih dari sekali di tabel tindak lanjut"
  done < "$TMP/dup"

  # bagian rinci wajib punya barisnya di tabel
  awk -F'|' '/^\| C-[0-9]+ \|/ { gsub(/ /,"",$2); print $2 }' "$A" | sort -u > "$TMP/rows.ids"
  grep -oE '^### C-[0-9]+' "$A" | sed 's/^### //' | sort -u > "$TMP/head.ids"
  comm -13 "$TMP/rows.ids" "$TMP/head.ids" | while IFS= read -r id; do
    [ -n "$id" ] || continue
    fail "$A: bagian rinci $id tidak punya baris di tabel tindak lanjut"
  done

  # Ringkasan jumlah temuan di dalam laporan audit **itu sendiri** wajib sama
  # dengan markernya. Aturan prosa di §2a hanya membaca baris yang menyebut
  # `AUDIT-001`, sedangkan paragraf §1 berada di berkas audit dan tidak menyebut
  # namanya sendiri — sehingga ia satu-satunya tempat yang menyatakan total
  # secara keseluruhan dan satu-satunya yang tidak diperiksa (temuan **C-081**:
  # prosa tetap menulis 75 sementara marker sudah 78 selama tiga sesi).
  # Pola ditulis **di dalam** program awk, bukan lewat `-v`: `-v` memproses escape
  # sehingga `\*` menjadi `*` dan `$0 ~ pat` tidak pernah cocok — versi pertama
  # aturan ini karena itu melaporkan OK sambil tidak membandingkan apa pun
  # (kelas C-080, kali kedua di sesi yang sama). Keduanya diikat ke **awal baris**
  # supaya kutipan angka lama di dalam baris tabel temuan tidak ikut dinilai.
  # Setiap tempat ringkasan diperiksa **terpisah**: pagar yang hanya menuntut
  # "salah satu ada" akan lolos ketika satu tempat berubah bentuk, dan tempat
  # yang berubah itu lalu berhenti diperiksa tanpa suara — kelas yang sama
  # dengan butir gigi ke-3 di `check-navigation.sh` (C-080).
  for SITE in '^\*\*[0-9]+ temuan\*\* secara keseluruhan|paragraf §1' '^> Ringkasan: \*\*[0-9]+ temuan\*\*|baris "> Ringkasan: **N temuan**"'; do
    SITE_PAT="${SITE%%|*}"
    SITE_NAME="${SITE##*|}"
    if [ "$(grep -cE "$SITE_PAT" "$A" | tr -d ' ')" = "0" ]; then
      fail "$A: $SITE_NAME tidak lagi cocok pola yang diperiksa ('**N temuan**' hilang atau berubah bentuk) — perbarui skripnya, jangan biarkan pemeriksaannya mati"
    fi
  done
  awk -v F="$A" -v total="$(val total)" '
    /^\*\*[0-9]+ temuan\*\* secara keseluruhan/ || /^> Ringkasan: \*\*[0-9]+ temuan\*\*/ {
      if (match($0, /\*\*[0-9]+ temuan\*\*/)) {
        m = substr($0, RSTART, RLENGTH); gsub(/[^0-9]/, "", m)
        if (m != total) printf "FAIL|%s:%d: ringkasan temuan di dalam laporan audit menulis %s, marker audit-summary bilang %s\n", F, NR, m, total
      } else {
        printf "FAIL|%s:%d: kalimat ringkasan temuan tidak lagi cocok pola yang diperiksa (C-081)\n", F, NR
      }
    }' "$A" | while IFS= read -r m; do fail "${m#FAIL|}"; done
done

# --------------------------- 2a. angka audit di prosa lima dokumen
if [ -n "$MARKER_REF" ]; then
  M_TOTAL="$(printf '%s' "$MARKER_REF" | grep -oE ' total=[0-9]+' | cut -d= -f2)"
  M_FIXED="$(printf '%s' "$MARKER_REF" | grep -oE ' fixed=[0-9]+' | cut -d= -f2)"
  M_OPEN="$(printf '%s' "$MARKER_REF" | grep -oE ' open=[0-9]+' | cut -d= -f2)"
  for D in $(find "$AUDIT_DIR" -name 'AUDIT-*.md' | sort) "$AUDIT_DIR/README.md" AGENTS.md "$STATE" CONTINUE.md; do
    [ -f "$D" ] || continue
    awk -v F="$D" -v total="$M_TOTAL" -v fixed="$M_FIXED" -v open="$M_OPEN" '
      /AUDIT-001/ && /FIXED/ {
        # baris temuan di laporan audit = catatan peristiwa (angka "saat itu"), bukan klaim keadaan kini
        if ($0 ~ /^\| C-[0-9]+ \|/) next
        if ($0 ~ /historis/ || $0 ~ /saat itu/ || $0 ~ /waktu itu/ || $0 ~ /ledger-check: skip/) next
        # angka pertama sebelum FIXED
        i = index($0, "FIXED")
        head = substr($0, 1, i - 1)
        n = 0
        while (match(head, /[0-9]+/)) { n = substr(head, RSTART, RLENGTH) + 0; head = substr(head, RSTART + RLENGTH) }
        if (n != fixed) printf "FAIL|%s:%d: prosa menulis %d FIXED, marker audit-summary bilang %s\n", F, NR, n, fixed
        # angka pertama sebelum OPEN, kecuali OPEN-QUESTIONS
        rest = $0
        while (match(rest, /[0-9]+[ \t*]*OPEN/)) {
          seg = substr(rest, RSTART, RLENGTH)
          after = substr(rest, RSTART + RLENGTH, 1)
          if (after != "-") { m = seg; sub(/[^0-9].*$/, "", m); m += 0
            if (m != open) printf "FAIL|%s:%d: prosa menulis %d OPEN, marker audit-summary bilang %s\n", F, NR, m, open
            break }
          rest = substr(rest, RSTART + RLENGTH)
        }
        # bentuk "a + b + c = d" harus berjumlah
        if (match($0, /[0-9]+ \+ [0-9]+ \+ [0-9]+ = [0-9]+/)) {
          s = substr($0, RSTART, RLENGTH)
          split(s, p, "[ +=]+")
          if (p[1] + p[2] + p[3] != p[4])
            printf "FAIL|%s:%d: penjumlahan audit di prosa tidak berjumlah: %s\n", F, NR, s
        }
      }' "$D" | while IFS= read -r m; do fail "${m#FAIL|}"; done
  done
fi

# ------------------------------------------- 2. marker audit disalin seragam
if [ -n "$MARKER_REF" ]; then
  for D in "$AUDIT_DIR/README.md" AGENTS.md "$STATE" CONTINUE.md; do
    if [ ! -f "$D" ]; then fail "$D: berkas tidak ada (marker audit-summary tidak dapat diperiksa)"; continue; fi
    M="$(grep -oE '<!-- audit-summary [^>]*-->' "$D" | grep 'total=' | head -1 || true)"
    if [ -z "$M" ]; then
      fail "$D: tidak memuat marker <!-- audit-summary ... --> seperti laporan audit"
      continue
    fi
    M="$(printf '%s' "$M" | sed 's/<!-- audit-summary //; s/ -->//; s/  */ /g; s/^ //; s/ $//')"
    [ "$M" = "$MARKER_REF" ] || fail "$D: marker audit-summary tidak sama dengan laporan audit ('$M' vs '$MARKER_REF')"
  done
fi

# ------------------------------------------------------------ 3. papan kerja task
if [ ! -f "$TASKS" ]; then
  fail "$TASKS: berkas tidak ada"
else
  # Spasi kode dibuang lebih dulu: sel tabel sering memuat pipe di dalam
  # backtick (`GET|POST /comments`), dan pipe itu akan menggeser kolom Selesai.
  awk '
    /^### (TODO|IN PROGRESS|BLOCKED|DONE)[ \t]*$/ { sec = $0; sub(/^### /, "", sec); next }
    /^\| *\**T-[0-9]+[a-z]?\** *\|/ {
      line = $0; gsub(/`[^`]*`/, "", line)
      id = line
      sub(/^\| */, "", id); sub(/ *\|.*$/, "", id); gsub(/\*/, "", id)
      if (sec != "") printf "%s\t%s\t%d\t%s\n", id, sec, NR, line
    }' "$TASKS" > "$TMP/board.tsv"

  cut -f1 "$TMP/board.tsv" | sort | uniq -d > "$TMP/dup.ids"
  while IFS= read -r id; do
    [ -n "$id" ] || continue
    secs="$(awk -F'\t' -v i="$id" '$1 == i { printf "%s ", $2 }' "$TMP/board.tsv")"
    lines="$(awk -F'\t' -v i="$id" '$1 == i { printf "%d ", $3 }' "$TMP/board.tsv")"
    fail "$TASKS: $id muncul di lebih dari satu kolom status ($secs| baris $lines) — satu task, satu kolom"
  done < "$TMP/dup.ids"

  awk -F'\t' '{ print $1"\t"$2 }' "$TMP/board.tsv" | sort | uniq -d | while IFS= read -r k; do
    [ -n "$k" ] || continue
    warn "$TASKS: $(printf '%s' "$k" | tr '\t' ' ') tercatat dua kali di kolom yang sama"
  done

  awk -F'\t' -v F="$TASKS" '
    $2 == "DONE" {
      n = split($4, c, "|")
      if (c[4] !~ /[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]/)
        printf "FAIL|%s:%d: baris DONE %s tanpa tanggal selesai YYYY-MM-DD\n", F, $3, $1
    }
    $2 != "DONE" {
      if ($0 ~ /\*\*DONE [0-9][0-9][0-9][0-9]-/)
        printf "FAIL|%s:%d: %s masih di kolom %s tetapi teksnya sudah mengklaim selesai (\"DONE ...\") — pindahkan ke kolom DONE\n", F, $3, $1, $2
    }' "$TMP/board.tsv" | while IFS= read -r m; do fail "${m#FAIL|}"; done
fi

# --------------------------------------------- 4. rujukan test / task / temuan
LIVE_DOCS="$STATE $TASKS $TRACE $QFILE CONTINUE.md AGENTS.md README.md"
LIVE_DOCS="$LIVE_DOCS $(find docs/design docs/adr -name '*.md' | sort | tr '\n' ' ')"
LIVE_DOCS="$LIVE_DOCS $(find "$AUDIT_DIR" -name '*.md' | sort | tr '\n' ' ')"
EXISTING=""
for f in $LIVE_DOCS; do [ -f "$f" ] && EXISTING="$EXISTING $f"; done

# 4a. sitasi berpath: `path_test.go::TestXxx` (klaim: berkas dan testnya ada)
grep -rnoE '[A-Za-z0-9_/.+-]+\.[A-Za-z0-9_]+_test\.go::Test[A-Za-z0-9_]+' $EXISTING 2>/dev/null \
| while IFS=: read -r f l ref; do
    p="${ref%%::*}"; n="${ref##*::}"
    if sed -n "${l}p" "$f" | grep -q 'ledger-check: skip'; then continue; fi
    if ! real="$(resolve_test_file "$p")"; then
      printf 'FAIL|%s:%d: sitasi test menunjuk berkas yang tidak ada: %s\n' "$f" "$l" "$p"
      continue
    fi
    grep -qE "^func $n\$" "$real" || printf 'FAIL|%s:%d: %s tidak ada di %s\n' "$f" "$l" "$n" "$real"
  done | while IFS= read -r m; do fail "${m#FAIL|}"; done

# 4b. nama test tanpa path, hanya pada dokumen status + ADR (log historis dikecualikan)
BARE_DOCS="$STATE $TASKS $TRACE $QFILE CONTINUE.md AGENTS.md README.md $(find docs/adr -name '*.md' | sort | tr '\n' ' ')"
BARE_EXISTING=""
for f in $BARE_DOCS; do [ -f "$f" ] && BARE_EXISTING="$BARE_EXISTING $f"; done

grep -rnoE '\bTest[A-Z][A-Za-z0-9_]+' $BARE_EXISTING 2>/dev/null \
| while IFS=: read -r f l n; do
    # `TestXxx` dan kerabatnya adalah placeholder di dokumen, bukan nama test
    case "$n" in Test*_|TestXxx|TestYyy|TestZzz|TestFoo|TestBar|TestName|TestSomething) continue ;; esac
    line="$(sed -n "${l}p" "$f")"
    case "$line" in *'ledger-check: skip'*) continue ;; esac
    case "$line" in
      *'**TODO**'*|*'**BLOCKED**'*|*'**OPEN**'*|*'**IN PROGRESS**'*|*'direncanakan'*|*'rencana test'*|*'belum ada'*|*'tidak ada di'*) continue ;;
    esac
    has_test "$n" || printf 'FAIL|%s:%d: rujukan test %s tidak ditemukan di backend/\n' "$f" "$l" "$n"
  done | while IFS= read -r m; do fail "${m#FAIL|}"; done

# 4c. ID task yang dirujuk wajib ada di papan
cut -f1 "$TMP/board.tsv" | sort -u > "$TMP/board.ids"
grep -rhoE '\bT-[0-9][0-9][0-9][a-z]?\b' $EXISTING 2>/dev/null | sort -u \
| while IFS= read -r id; do
    grep -qx "$id" "$TMP/board.ids" || printf 'FAIL|rujukan task %s tidak ada di %s\n' "$id" "$TASKS"
  done | while IFS= read -r m; do fail "${m#FAIL|}"; done

# 4d. ID temuan yang dirujuk wajib ada di tabel tindak lanjut
if [ -s "$TMP/rows.ids" ]; then
  grep -rhoE '\bC-[0-9][0-9][0-9]\b' $EXISTING 2>/dev/null | sort -u \
  | while IFS= read -r id; do
      grep -qx "$id" "$TMP/rows.ids" || printf 'FAIL|rujukan temuan %s tidak ada di tabel tindak lanjut audit\n' "$id"
    done | while IFS= read -r m; do fail "${m#FAIL|}"; done
fi

# ---------------------------------------- 5. hitungan test di STATE.md §3
if [ -f "$STATE" ]; then
  sed -n '/^## 3\. Modul/,/^## 4\./p' "$STATE" > "$TMP/state3.md"

  # Tiga bentuk penulisan klaim per berkas, semuanya diperiksa. Bentuk ketiga
  # (`(N test: …)`) dahulu TIDAK tercakup pola lama, sehingga dua klaim yang
  # ditulis dengan bentuk itu lolos tanpa diperiksa (temuan C-075) — pola yang
  # tidak mengenali klaim adalah cara paling sunyi mematikan pemeriksaan.
  grep -oE '`[A-Za-z0-9_./-]+_test\.go` \(\*\*[0-9]+\*\*[^)]*\)|`[A-Za-z0-9_./-]+_test\.go` \([0-9]+ test[^)]*\)|`[A-Za-z0-9_./-]+_test\.go` \([0-9]+\)' "$TMP/state3.md" 2>/dev/null \
  | sed -E 's/`([^`]+)` \(\*\*([0-9]+)\*\*.*/\1|\2/; s/`([^`]+)` \(([0-9]+) test.*/\1|\2/; s/`([^`]+)` \(([0-9]+)\)/\1|\2/' > "$TMP/state3.counts"
  while IFS='|' read -r p n; do
    [ -n "$p" ] || continue
    if ! real="$(resolve_test_file "$p")"; then
      printf 'FAIL|%s §3: hitungan test pada berkas yang tidak ada: %s\n' "$STATE" "$p"
      continue
    fi
    have="$(count_tests_in "$real")"
    [ "$have" = "$n" ] || printf 'FAIL|%s §3: %s ditulis %s test, sebenarnya %s\n' "$STATE" "$p" "$n" "$have"
  done < "$TMP/state3.counts" | while IFS= read -r m; do fail "${m#FAIL|}"; done

  grep -oE '`[A-Za-z0-9_./-]+` \*\*[0-9]+\*\* test' "$TMP/state3.md" 2>/dev/null \
  | sed -E 's/`([^`]+)` \*\*([0-9]+)\*\* test/\1|\2/' > "$TMP/state3.dirs"
  while IFS='|' read -r d n; do
    [ -n "$d" ] || continue
    dir="$d"; [ -d "$dir" ] || dir="$BACKEND/$d"
    if [ ! -d "$dir" ]; then
      printf 'FAIL|%s §3: hitungan test pada direktori yang tidak ada: %s\n' "$STATE" "$d"
      continue
    fi
    have="$(grep -hE '^func Test' $(find "$dir" -name '*_test.go' -type f | sort) 2>/dev/null | wc -l | tr -d ' ')"
    [ "$have" = "$n" ] || printf 'FAIL|%s §3: %s ditulis %s test, sebenarnya %s\n' "$STATE" "$d" "$n" "$have"
  done < "$TMP/state3.dirs" | while IFS= read -r m; do fail "${m#FAIL|}"; done

  grep -oE '\*\*total suite [0-9]+ test\*\*' "$TMP/state3.md" 2>/dev/null | grep -oE '[0-9]+' > "$TMP/state3.total"
  while IFS= read -r n; do
    [ -n "$n" ] || continue
    [ "$n" = "$TOTAL_TESTS" ] || printf 'FAIL|%s §3: total suite ditulis %s test, sebenarnya %s\n' "$STATE" "$n" "$TOTAL_TESTS"
  done < "$TMP/state3.total" | while IFS= read -r m; do fail "${m#FAIL|}"; done

  info "info  total suite yang dihitung skrip: $TOTAL_TESTS test"
fi

sort -u "$FAILS" > "$TMP/fails.u"
sort -u "$WARNS" > "$TMP/warns.u"
if [ -s "$TMP/warns.u" ] || [ -s "$TMP/fails.u" ]; then echo; fi
[ -s "$TMP/warns.u" ] && show WARN "$TMP/warns.u"
[ -s "$TMP/fails.u" ] && show FAIL "$TMP/fails.u"
NF="$(n_of "$TMP/fails.u")"
NW="$(n_of "$TMP/warns.u")"
echo
if [ "$NF" -eq 0 ]; then
  printf 'ledger OK — %d peringatan, %s test di backend\n' "$NW" "$TOTAL_TESTS"
  exit 0
fi
printf 'ledger GAGAL: %d temuan, %d peringatan\n' "$NF" "$NW"
exit 1
