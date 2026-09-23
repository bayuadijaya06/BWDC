#!/usr/bin/env bash
# check-api-contract.sh — verifikasi anotasi izin endpoint terhadap matriks RBAC dan router.
#
# Cara pakai (dari root repo):
#   bash scripts/check-api-contract.sh
#
# Kenapa ada: `T-024` dijalankan berkali-kali sebagai **hitungan manual** — "45/55 endpoint punya
# baris izin" — dan angka itu pernah tidak dapat diperiksa silang (C-055). Lebih buruk lagi,
# pasangan izin yang tidak ada di matriks dapat ditulis di kontrak tanpa ada yang menangkapnya:
# `document_version:read` pernah muncul di draf §4 padahal matriks tidak memuatnya (Q-016).
# Karena itu yang diperiksa di sini bukan hanya "apakah izinnya ditulis", tetapi juga
# **apakah pasangan yang ditulis benar-benar ada di matriks** — di dokumen maupun di kode.
#
# Tiga aturan:
#   1. Setiap endpoint di `42-API.md` harus punya izin yang terbaca: baris `Izin:` di dalam
#      bloknya, **atau** baris di tabel izin babnya (bentuk `| \`GET /documents\` | \`document:read\` |`).
#   2. Setiap pasangan `resource:action` yang disebut pada baris `Izin:` dan pada tabel izin wajib
#      ada di matriks `44-SECURITY.md` §3.1.2. Pasangan yang dikarang = keadaan matriks berubah
#      tanpa ADR (ADR-0014).
#   3. Setiap `RequirePermission(deps.Permission, "...", "...")` di `internal/handler/router.go`
#      wajib memakai pasangan dari matriks yang sama — inilah yang menahan kode dan matriks
#      menyimpang satu sama lain.
#
# Batas yang disadari:
#   1. Yang diperiksa adalah **keberadaan dan keabsahan** anotasi, bukan kesesuaian route di
#      router dengan judul endpoint di kontrak. Endpoint yang belum diimplementasikan (mis.
#      modul Workflow) memang belum punya route, jadi memeriksa arah itu akan gagal selamanya.
#   2. Pasangan ber-placeholder (`<action>`) dan berpola wildcard (`report:*`) dilewati: keduanya
#      menjelaskan aturan, bukan menetapkan pasangan.
#   3. Cakupan data (§3.1.3) tidak diperiksa mesin — ia hidup di kueri, dan pemeriksanya adalah
#      test per modul, bukan skrip dokumen.
#
# Exit code: 1 bila ada FAIL, 0 bila semua anotasi sah.

set -u

ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

API=docs/design/42-API.md
SEC=docs/design/44-SECURITY.md
ROUTER=backend/internal/handler/router.go

fail=0
checks=0

fail_msg() { printf 'FAIL  %s\n' "$1"; fail=$((fail + 1)); }
info() { printf 'info  %s\n' "$1"; }

for f in "$API" "$SEC" "$ROUTER"; do
  [ -f "$f" ] || { printf 'FAIL  berkas yang dibutuhkan tidak ada: %s\n' "$f"; exit 1; }
done

# ---------------------------------------------------------------------------
# Sumber kebenaran: matriks izin (§3.1.2)
# ---------------------------------------------------------------------------

matrix_pair_re='^\| `[a-z_]+` \| `[a-z_]+` \|'
matrix_pairs=$(grep -E "$matrix_pair_re" "$SEC" | sed -E 's/^\| `([a-z_]+)` \| `([a-z_]+)` \|.*/\1:\2/' | sort -u)
matrix_count=$(printf '%s\n' "$matrix_pairs" | grep -c . )
[ "$matrix_count" -gt 0 ] || { printf 'FAIL  matriks izin tidak terbaca di %s — pola barisnya mungkin berubah\n' "$SEC"; exit 1; }
info "matriks izin: $matrix_count pasangan <- $SEC §3.1.2"

in_matrix() { printf '%s\n' "$matrix_pairs" | grep -qxF "$1"; }

# ---------------------------------------------------------------------------
# 1 & 2. Setiap endpoint punya izin yang terbaca, dan pasangannya ada di matriks
# ---------------------------------------------------------------------------

endpoints=$(grep -oE '^### (GET|POST|PATCH|PUT|DELETE) [^ ]+' "$API" | sed -E 's/^### //' | sort -u)
endpoint_count=$(printf '%s\n' "$endpoints" | grep -c .)

# a. Endpoint yang punya baris `Izin:` di dalam bloknya.
annotated=$(awk '
  /^### (GET|POST|PATCH|PUT|DELETE) /{ if (h != "" && found) print h; h=substr($0, 5); found=0; next }
  /^Izin:/{ found=1 }
  END{ if (h != "" && found) print h }
' "$API" | sort -u)
annotated_count=$(printf '%s\n' "$annotated" | grep -c .)

# b. Endpoint yang izinnya ada di tabel bab (bentuk `| \`GET /documents\` | \`document:read\` |`).
table_covered=$(grep -oE '^\| `(GET|POST|PATCH|PUT|DELETE) /[^`]*` \|' "$API" | sed -E 's/^\| `//; s/` \|$//' | sort -u)
table_covered_count=$(printf '%s\n' "$table_covered" | grep -c .)

checks=$((checks + endpoint_count))
while IFS= read -r ep; do
  [ -z "$ep" ] && continue
  if printf '%s\n' "$annotated" | grep -qxF "$ep"; then continue; fi
  if printf '%s\n' "$table_covered" | grep -qxF "$ep"; then continue; fi
  fail_msg "$API: endpoint \"$ep\" tidak punya izin yang terbaca — tambahkan baris \"Izin: ...\" di bloknya, atau baris di tabel izin babnya"
done <<< "$endpoints"

info "endpoint: $endpoint_count ($annotated_count dengan baris \`Izin:\` di bloknya, $table_covered_count lewat tabel izin bab)"

# c. Klaim ringkas di `AGENTS.md` ("anotasi `Izin:` per endpoint, kini N/M") diperiksa
#    terhadap angka yang baru saja dihitung. Sebelum butir ini, `T-024` masih ditulis
#    "40/51" di sana padahal sudah 48/55 — klaim angka yang tidak diperiksa (kelas C-075).
agents="$ROOT/AGENTS.md"
if [ -f "$agents" ]; then
  claim=$(grep -oE 'anotasi `Izin:` per endpoint, kini [0-9]+/[0-9]+' "$agents" | head -1)
  checks=$((checks + 1))
  if [ -z "$claim" ]; then
    fail_msg "$agents: klaim jumlah anotasi izin (\"anotasi \`Izin:\` per endpoint, kini N/M\") tidak ditemukan — perbarui pola di skrip ini"
  else
    claimed_izin=$(printf '%s' "$claim" | grep -oE '[0-9]+/[0-9]+' | cut -d/ -f1)
    claimed_total=$(printf '%s' "$claim" | grep -oE '[0-9]+/[0-9]+' | cut -d/ -f2)
    [ "$claimed_total" = "$endpoint_count" ] || fail_msg "$agents: jumlah endpoint ditulis $claimed_total, sebenarnya $endpoint_count (sumber: $API)"
    [ "$claimed_izin" = "$annotated_count" ] || fail_msg "$agents: endpoint beranotasi \`Izin:\` ditulis $claimed_izin, sebenarnya $annotated_count ($table_covered_count sisanya lewat tabel izin bab)"
  fi
fi

# Pasangan izin yang disebut pada baris `Izin:` dan pada baris tabel izin.
pair_sources=$( { grep '^Izin:' "$API"; grep -E '^\| `(GET|POST|PATCH|PUT|DELETE) /[^`]*` \|' "$API"; } )
pairs=$(printf '%s\n' "$pair_sources" | grep -oE '`[a-z_]+:[a-z_]+`' | tr -d '`' | sort -u)
pair_count=$(printf '%s\n' "$pairs" | grep -c .)
checks=$((checks + pair_count))

while IFS= read -r pair; do
  [ -z "$pair" ] && continue
  in_matrix "$pair" || fail_msg "$API: pasangan izin \"$pair\" tidak ada di matriks $SEC §3.1.2 — jangan mengarang pasangan tanpa ADR baru (ADR-0014)"
done <<< "$pairs"

info "pasangan izin pada anotasi & tabel izin: $pair_count diperiksa terhadap matriks"

# ---------------------------------------------------------------------------
# 3. Pasangan izin di router wajib berasal dari matriks yang sama
# ---------------------------------------------------------------------------

router_pairs=$(grep -oE 'RequirePermission\(deps\.Permission, "[^"]+", "[^"]+"\)' "$ROUTER" | sed -E 's/.*"([^"]+)", "([^"]+)"\).*/\1:\2/' | sort -u)
router_count=$(printf '%s\n' "$router_pairs" | grep -c .)

if [ "$router_count" -eq 0 ]; then
  fail_msg "$ROUTER: tidak ada pemanggilan RequirePermission yang terbaca — pola kodenya mungkin berubah, perbarui skrip ini"
fi

checks=$((checks + router_count))
while IFS= read -r pair; do
  [ -z "$pair" ] && continue
  in_matrix "$pair" || fail_msg "$ROUTER: route memakai pasangan izin \"$pair\" yang tidak ada di matriks $SEC §3.1.2"
done <<< "$router_pairs"

info "pasangan izin di router: $router_count diperiksa terhadap matriks"

# ---------------------------------------------------------------------------

echo "---"
if [ "$fail" -eq 0 ]; then
  echo "api-contract OK — $checks pemeriksaan, $endpoint_count endpoint"
  exit 0
fi
echo "api-contract GAGAL: $fail temuan dari $checks pemeriksaan — perbaiki kontrak, matriks, atau route-nya; jangan ubah skrip ini agar lulus"
exit 1
