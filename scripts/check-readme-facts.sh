#!/usr/bin/env bash
# check-readme-facts.sh — verifikasi angka & versi di `README.md` terhadap repo.
#
# Cara pakai (dari root repo):
#   bash scripts/check-readme-facts.sh
#
# Kenapa ada: `README.md` adalah berkas yang pertama dibaca orang luar, dan klaim di dalamnya
# pernah basi tanpa ada yang menangkapnya — jumlah route ditulis 30 padahal router memuat 32,
# `frontend/` masih disebut kosong padahal kerangkanya sudah berdiri (temuan C-061), dan baris
# "Frontend (rencana) ... Belum diinisialisasi" bertahan sesudah itu. Angka seperti ini tidak
# dijaga test dan tidak dijaga `check-ledger.sh` (yang menjaga *ledger*), jadi pemeriksanya
# ditaruh di sini. Kelas cacat: C-060, C-061.
#
# Cara kerja: setiap fakta punya **sumber kebenaran** di repo (bukan dokumen lain), lalu klaim
# di README dibandingkan dengan nilai yang dihitung dari sumber itu. Tidak menulis apa pun.
#
# Batas yang disadari:
#   1. Hanya `README.md` yang diperiksa. Dokumen lain memuat versi yang sama, tetapi sebagian
#      **sengaja** historis — mis. ADR-0002 menyebut "React 18" dan `30-ARCHITECTURE.md` §2.1
#      menjelaskan bahwa angkanya diputuskan ulang di ADR-0024. Memindai seluruh dokumen akan
#      menandai riwayat yang benar sebagai basi. Untuk menambah berkas, tambahkan pola klaimnya
#      di sini dengan berkas targetnya — jangan menyalin seluruh skrip.
#   2. Hanya klaim ber-pola tetap yang dapat diperiksa. Angka di prosa bebas (mis. "dua puluh
#      tabel") tidak diperiksa; kalimatnya harus diubah lebih dulu menjadi pola yang terbaca mesin.
#   3. Angka dalam huruf hanya dipetakan 1..10 (Indonesia), karena hanya itu yang dipakai README.
#   4. Klaim yang polanya hilang dari README dianggap **gagal**, bukan dilewati: pemeriksa yang
#      diam-diam berhenti memeriksa lebih berbahaya daripada pemeriksa yang berisik.
#
# Exit code: 1 bila ada FAIL, 0 bila semua fakta cocok.

set -u

ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

README=README.md
ROUTER=backend/internal/handler/router.go
GOMOD=backend/go.mod
PKG=frontend/package.json
MIGDIR=backend/internal/migration

fail=0
checks=0

fail_msg() { printf 'FAIL  %s\n' "$1"; fail=$((fail + 1)); }
info() { printf 'info  %s\n' "$1"; }

for f in "$README" "$ROUTER" "$GOMOD" "$PKG"; do
  [ -f "$f" ] || { printf 'FAIL  berkas yang dibutuhkan tidak ada: %s\n' "$f"; exit 1; }
done

# ---------------------------------------------------------------------------
# Sumber kebenaran
# ---------------------------------------------------------------------------

ROUTE_RE='^[[:space:]]+[A-Za-z0-9_]+\.(GET|POST|PATCH|PUT|DELETE)\('

route_total() { grep -cE "$ROUTE_RE" "$ROUTER"; }
route_receiver() { grep -cE "^[[:space:]]+$1\.(GET|POST|PATCH|PUT|DELETE)\(" "$ROUTER"; }

gomod_go() { grep -E '^go ' "$GOMOD" | awk '{print $2}'; }
gomod_dep() { grep -E "^[[:space:]]+$1 v" "$GOMOD" | head -1 | awk '{print $2}' | tr -d 'v'; }
pkg_major() { grep -E "^[[:space:]]+\"$1\":" "$PKG" | head -1 | grep -oE '[0-9]+' | head -1; }
pkg_minor() { grep -E "^[[:space:]]+\"$1\":" "$PKG" | head -1 | grep -oE '[0-9]+\.[0-9]+' | head -1; }

mig_first() { ls "$MIGDIR" | grep -oE '^[0-9]{3}' | sort -n | head -1; }
mig_last() { ls "$MIGDIR" | grep -oE '^[0-9]{3}' | sort -n | tail -1; }
mig_count() { ls "$MIGDIR" | grep -cE '^[0-9]{3}_'; }

# ---------------------------------------------------------------------------
# 1. Jumlah route dan rinciannya — sumber: `$ROUTER`
# ---------------------------------------------------------------------------

n_health=$(route_receiver r)
n_auth=$(route_receiver auth)
n_admin=$(route_receiver admin)
n_project=$(route_receiver projects)
n_document=$(route_receiver documents)
n_task=$(route_receiver tasks)
n_comment=$(route_receiver comments)
n_workflow=$(route_receiver workflows)
n_analytics=$(route_receiver analytics)
n_notification=$(route_receiver notifications)
n_audit=$(route_receiver audit)
n_total=$(route_total)

# Setiap penerima route harus punya ember di atas. Tanpa pemeriksaan ini sebuah group baru akan
# menambah total tanpa masuk rincian, dan README akan tampak cocok padahal tidak.
known_receivers="r auth admin projects documents tasks comments workflows analytics notifications audit"
unclassified=$(grep -oE "$ROUTE_RE" "$ROUTER" | sed -E 's/^[[:space:]]+([A-Za-z0-9_]+)\..*/\1/' | sort -u | grep -vxE "$(printf '%s' "$known_receivers" | tr ' ' '|')" || true)
if [ -n "$unclassified" ]; then
  fail_msg "$ROUTER: group route belum terklasifikasi di skrip ini ($(printf '%s' "$unclassified" | tr '\n' ' ')) — tambahkan embernya di sini dan di README, lalu perbarui jumlahnya"
fi

sum_parts=$((n_health + n_auth + n_admin + n_project + n_document + n_task + n_comment + n_workflow + n_analytics + n_notification + n_audit))
[ "$sum_parts" -eq "$n_total" ] || fail_msg "$ROUTER: rincian route berjumlah $sum_parts, total tercatat $n_total — ada route yang lolos dari pengklasifikasian"

info "route: $n_total ($n_health health, $n_auth auth, $n_admin admin, $n_project project, $n_document document, $n_task task, $n_comment comment, $n_workflow workflow, $n_analytics analytics, $n_notification notifications, $n_audit audit) <- $ROUTER"

hit=$(grep -nE 'router: \*\*[0-9]+ route\*\*' "$README" | head -1)
checks=$((checks + 1))
if [ -z "$hit" ]; then
  fail_msg "$README: klaim jumlah route (\"router: **N route**\") tidak ditemukan — perbarui pola di skrip ini"
else
  no=${hit%%:*}
  claimed_total=$(printf '%s' "${hit#*:}" | grep -oE '\*\*[0-9]+ route\*\*' | grep -oE '[0-9]+' | head -1)
  [ "$claimed_total" = "$n_total" ] || fail_msg "$README:$no: jumlah route ditulis $claimed_total, sebenarnya $n_total (sumber: $ROUTER)"
fi

actual_parts="(1 health, $n_auth auth, $n_admin admin, $n_project project, $n_document document, $n_task task, $n_comment comment, $n_workflow workflow, $n_analytics analytics, $n_notification notifications, $n_audit audit)"
hit=$(grep -nE '\(1 health, [0-9]+ auth, [0-9]+ admin, [0-9]+ project, [0-9]+ document, [0-9]+ task, [0-9]+ comment, [0-9]+ workflow, [0-9]+ analytics, [0-9]+ notifications, [0-9]+ audit\)' "$README" | head -1)
checks=$((checks + 1))
if [ -z "$hit" ]; then
  fail_msg "$README: rincian route per modul tidak ditemukan — perbarui pola di skrip ini"
else
  no=${hit%%:*}
  claimed_parts=$(printf '%s' "${hit#*:}" | grep -oE '\(1 health, [0-9]+ auth, [0-9]+ admin, [0-9]+ project, [0-9]+ document, [0-9]+ task, [0-9]+ comment, [0-9]+ workflow, [0-9]+ analytics, [0-9]+ notifications, [0-9]+ audit\)' | head -1)
  [ "$claimed_parts" = "$actual_parts" ] || fail_msg "$README:$no: rincian route ditulis \"$claimed_parts\", sebenarnya \"$actual_parts\" (sumber: $ROUTER)"
fi

# ---------------------------------------------------------------------------
# 2. Jumlah endpoint per modul di tabel status — sumber: `$ROUTER`
# ---------------------------------------------------------------------------

# <label baris>;<penerima route>;<pola nilai di baris itu>
MODULE_ROWS=(
  "Modul Project;projects;endpoint"
  "Modul Document;documents;endpoint"
  "Modul Task;tasks;endpoint"
  "Modul Comment;comments;endpoint"
  "Modul Workflow;workflows;endpoint"
)

for row in "${MODULE_ROWS[@]}"; do
  IFS=';' read -r label receiver unit <<< "$row"
  hit=$(grep -nE "^\| $label \|" "$README" | head -1)
  checks=$((checks + 1))
  if [ -z "$hit" ]; then
    fail_msg "$README: baris \"$label\" tidak ditemukan — perbarui pola di skrip ini"
    continue
  fi
  no=${hit%%:*}; text=${hit#*:}
  claimed=$(printf '%s' "$text" | grep -oE "[0-9]+ $unit" | head -1 | grep -oE '[0-9]+')
  actual=$(route_receiver "$receiver")
  if [ -z "$claimed" ]; then
    fail_msg "$README:$no: baris $label tidak lagi menyebut \"N $unit\" dengan angka — pakai angka agar dapat diperiksa"
  elif [ "$claimed" != "$actual" ]; then
    fail_msg "$README:$no: $label ditulis $claimed $unit, sebenarnya $actual (sumber: $ROUTER)"
  fi
done

# Jumlah endpoint auth ditulis dalam huruf ("Lima endpoint auth"). Peta 1..10 sudah cukup.
word_number() {
  case "$(printf '%s' "$1" | tr 'A-Z' 'a-z')" in
    satu) echo 1 ;; dua) echo 2 ;; tiga) echo 3 ;; empat) echo 4 ;; lima) echo 5 ;;
    enam) echo 6 ;; tujuh) echo 7 ;; delapan) echo 8 ;; sembilan) echo 9 ;; sepuluh) echo 10 ;;
    *) echo "" ;;
  esac
}
auth_row=$(grep -nE '^\| Auth \+ RBAC \|' "$README" | head -1)
checks=$((checks + 1))
if [ -z "$auth_row" ]; then
  fail_msg "$README: baris \"Auth + RBAC\" tidak ditemukan — perbarui pola di skrip ini"
else
  no=${auth_row%%:*}; text=${auth_row#*:}
  claimed_word=$(printf '%s' "$text" | grep -oE '[A-Za-z]+ endpoint auth' | head -1 | grep -oE '^[A-Za-z]+')
  claimed=$(word_number "$claimed_word")
  if [ -z "$claimed" ]; then
    fail_msg "$README:$no: jumlah endpoint auth tidak lagi ditulis sebagai angka huruf yang dikenali skrip ini"
  elif [ "$claimed" != "$n_auth" ]; then
    fail_msg "$README:$no: endpoint auth ditulis \"$claimed_word\" ($claimed), sebenarnya $n_auth (sumber: $ROUTER)"
  fi
fi

# ---------------------------------------------------------------------------
# 3. Versi dependensi — sumber: `$GOMOD` dan `$PKG`
# ---------------------------------------------------------------------------

# check_mentions <label> <pola klaim> <nilai nyata> <sumber> [exact|prefix]
check_mentions() {
  local label=$1 pat=$2 actual=$3 src=$4 mode=${5:-exact} hits=0 hit no text v
  while IFS= read -r hit; do
    [ -z "$hit" ] && continue
    no=${hit%%:*}; text=${hit#*:}
    # `tail -1`: pola klaim boleh memuat angka di depan (mis. `pgx/v5` v5.7.4), dan yang dibandingkan
    # adalah versi di ujung pola.
    v=$(printf '%s' "$text" | grep -oE "$pat" | head -1 | grep -oE '[0-9]+(\.[0-9]+)*' | tail -1)
    [ -z "$v" ] && continue
    hits=$((hits + 1)); checks=$((checks + 1))
    if [ "$mode" = prefix ]; then
      case "$actual" in
        "$v"*) : ;;
        *) fail_msg "$README:$no: $label ditulis \"$v\", sebenarnya \"$actual\" (sumber: $src)" ;;
      esac
    elif [ "$v" != "$actual" ]; then
      fail_msg "$README:$no: $label ditulis \"$v\", sebenarnya \"$actual\" (sumber: $src)"
    fi
  done < <(grep -nE "$pat" "$README")
  if [ "$hits" -eq 0 ]; then
    fail_msg "$README: tidak ada klaim $label yang dapat diperiksa — polanya mungkin berubah, perbarui skrip ini"
  fi
}

# Backend: nilai nyata dibaca dari `go.mod`.
go_actual=$(gomod_go)
gin_actual=$(gomod_dep 'github.com/gin-gonic/gin')
pgx_actual=$(gomod_dep 'github.com/jackc/pgx/v5')
viper_actual=$(gomod_dep 'github.com/spf13/viper')
goose_actual=$(gomod_dep 'github.com/pressly/goose/v3')

info "versi backend: go $go_actual, gin $gin_actual, pgx $pgx_actual, viper $viper_actual, goose $goose_actual <- $GOMOD"
check_mentions "versi Go" 'Go [0-9]+\.[0-9]+' "$go_actual" "$GOMOD" prefix
check_mentions "versi Gin" 'Gin v[0-9]+\.[0-9]+\.[0-9]+' "$gin_actual" "$GOMOD"
check_mentions "versi pgx" 'pgx/v5` v[0-9]+\.[0-9]+\.[0-9]+' "$pgx_actual" "$GOMOD"
check_mentions "versi viper" 'viper v[0-9]+\.[0-9]+\.[0-9]+' "$viper_actual" "$GOMOD"
check_mentions "versi goose" 'goose v[0-9]+\.[0-9]+\.[0-9]+' "$goose_actual" "$GOMOD"

# Frontend: yang dibandingkan adalah angka **major**, karena itu yang ditulis README.
react_actual=$(pkg_major react)
vite_actual=$(pkg_major vite)
tailwind_actual=$(pkg_major tailwindcss)
typescript_actual=$(pkg_minor typescript)

info "versi frontend: react $react_actual, vite $vite_actual, tailwind v$tailwind_actual, typescript $typescript_actual <- $PKG"
check_mentions "versi React" 'React [0-9]+' "$react_actual" "$PKG"
check_mentions "versi Vite" 'Vite [0-9]+' "$vite_actual" "$PKG"
check_mentions "versi Tailwind" 'Tailwind v[0-9]+' "$tailwind_actual" "$PKG"
check_mentions "versi TypeScript" 'TypeScript [0-9]+\.[0-9]+' "$typescript_actual" "$PKG"

# ---------------------------------------------------------------------------
# 4. Rentang migrasi — sumber: berkas di `$MIGDIR`
# ---------------------------------------------------------------------------

first_actual=$(mig_first)
last_actual=$(mig_last)
count_actual=$(mig_count)
info "migrasi: $first_actual..$last_actual ($count_actual berkas) <- $MIGDIR/"

expected_count=$((10#$last_actual - 10#$first_actual + 1))
[ "$count_actual" -eq "$expected_count" ] || fail_msg "$MIGDIR: ada nomor migrasi yang bolong — $first_actual..$last_actual seharusnya $expected_count berkas, ditemukan $count_actual"

hit=$(grep -nE 'migrasi `[0-9]{3}` sampai `[0-9]{3}`' "$README" | head -1)
checks=$((checks + 1))
if [ -z "$hit" ]; then
  fail_msg "$README: rentang migrasi (\"migrasi \`001\` sampai \`010\`\") tidak ditemukan — perbarui pola di skrip ini"
else
  no=${hit%%:*}
  mig_claim=$(printf '%s' "${hit#*:}" | grep -oE 'migrasi `[0-9]{3}` sampai `[0-9]{3}`' | grep -oE '[0-9]{3}')
  claimed_first=$(printf '%s' "$mig_claim" | head -1)
  claimed_last=$(printf '%s' "$mig_claim" | tail -1)
  [ "$claimed_first" = "$first_actual" ] || fail_msg "$README:$no: migrasi pertama ditulis $claimed_first, sebenarnya $first_actual (sumber: $MIGDIR/)"
  [ "$claimed_last" = "$last_actual" ] || fail_msg "$README:$no: migrasi terakhir ditulis $claimed_last, sebenarnya $last_actual (sumber: $MIGDIR/)"
fi

# ---------------------------------------------------------------------------
# 5. Klaim "belum ada" atas sesuatu yang sudah ada — kelas C-060/C-061
# ---------------------------------------------------------------------------

# <path yang harus sudah ada>;<pola nama pada baris>;<pola klaim tidak-ada>
ABSENCE_CHECKS=(
  "frontend/src;frontend;belum diinisialisasi|masih kosong|belum diisi|belum ada"
)

absence_checked=0
for entry in "${ABSENCE_CHECKS[@]}"; do
  IFS=';' read -r path namepat claimpat <<< "$entry"
  [ -e "$path" ] || continue
  absence_checked=$((absence_checked + 1))
  while IFS= read -r hit; do
    [ -z "$hit" ] && continue
    no=${hit%%:*}; text=${hit#*:}
    printf '%s' "$text" | grep -qiE "$namepat" || continue
    printf '%s' "$text" | grep -qiE "$claimpat" || continue
    checks=$((checks + 1))
    fail_msg "$README:$no: klaim \"$claimpat\" tentang \"$namepat\", padahal $path sudah ada"
  done < <(grep -nE "$claimpat" "$README" || true)
done
info "klaim \"tidak ada\" yang dapat diperiksa: $absence_checked (path: ${ABSENCE_CHECKS[*]%%;*})"

# ---------------------------------------------------------------------------
# 6. Daftar berkas di pohon folder — kelas C-066
# ---------------------------------------------------------------------------

# Pohon §4 README adalah **klaim** tentang isi repo, dan daftar yang tidak pernah
# diperiksa akan tertinggal setiap kali skrip atau direktori baru lahir (C-066:
# lima skrip sudah berdiri, README masih menyebut dua). Berkas yang tidak disebut
# sama sekali adalah berkas yang tidak akan ditemukan pembaca berikutnya.
# Semua berkas di `scripts/`, bukan hanya `*.sh`: pemeriksa berbasis Node (mis.
# `responsive-evidence.mjs`) adalah kelas yang sama persis, dan batas `*.sh`
# membuatnya tidak pernah diperiksa (pola C-066 dengan wajah berbeda).
for f in scripts/*; do
  [ -f "$f" ] || continue
  checks=$((checks + 1))
  grep -qF "$(basename "$f")" "$README" ||
    fail_msg "$README: berkas $f tidak pernah disebut (pohon §4 atau perintah §12.3) — perbarui daftarnya"
done

# Arah sebaliknya: setiap path `scripts/...` dan `skills/...` yang ditulis README
# wajib benar-benar ada di disk.
while IFS= read -r p; do
  [ -n "$p" ] || continue
  checks=$((checks + 1))
  [ -e "$p" ] || fail_msg "$README menunjuk path yang tidak ada: $p"
done < <(grep -oE '(scripts|skills)/[A-Za-z0-9._/-]+' "$README" | sed -E 's/[.,);]+$//' | sort -u)

script_count=$(find scripts -maxdepth 1 -type f | wc -l | tr -d ' ')
info "pohon folder: $script_count berkas di scripts/ semuanya disebut, path scripts//skills/ di README diperiksa"

# ---------------------------------------------------------------------------

echo "---"
if [ "$fail" -eq 0 ]; then
  echo "readme-facts OK — $checks fakta diperiksa"
  exit 0
fi
echo "readme-facts GAGAL: $fail temuan dari $checks fakta — perbaiki README atau sumbernya, jangan ubah skrip ini agar lulus"
exit 1
