# Makefile backend BWDCS.
#
# Target mengikuti `60-DEPLOYMENT.md` §3.1; aturan lain mengikuti §8.
# Konfigurasi dibaca dari `.env` di root repo (satu sumber daftar variabel:
# `60-DEPLOYMENT.md` §2.1). Setiap resep memuat berkas itu BILA ADA, tanpa
# menimpa environment yang sudah diset di shell.
#
# Catatan: `docker-build` memerlukan Dockerfile di direktori ini. Dockerfile
# belum dibuat (menunggu task kontainerisasi); target disediakan agar perintah
# dokumentasi tetap jalan begitu berkas itu ada.

SHELL := /bin/bash
GO    ?= go
GOOSE ?= goose

LOAD_ENV  = set -a; [ -f ../.env ] && . ../.env; set +a;
REQUIRE_GOOSE = command -v $(GOOSE) >/dev/null 2>&1 || { echo "goose tidak ditemukan di PATH; lihat 12-DEVELOPMENT-WORKFLOW.md §2 (T-011/T-012)"; exit 1; };
DB_DSN    = host=$${DB_HOST:-localhost} port=$${DB_PORT:-5432} user=$${DB_USER:-bwdcs} dbname=$${DB_NAME:-bwdcs} password=$${DB_PASSWORD} sslmode=disable

# DSN test diturunkan dari kredensial yang sama, tetapi menunjuk database
# TERPISAH (default `bwdcs_test`) supaya suite tidak pernah menyentuh database
# dev aplikasi (temuan C-038 / task T-036; kontrak: 70-TESTING.md §8.1).
# Nilai `TEST_DATABASE_URL` dari shell atau ../.env selalu menang.
# Menyembunyikan sandi saat DSN dicetak: ganti bagian user:password@.
MASK_DSN  = sed -e 's|://[^@]*@|://***@|'
DERIVE_TEST_DSN = export TEST_DATABASE_URL="$${TEST_DATABASE_URL:-postgres://$${DB_USER}:$${DB_PASSWORD}@$${DB_HOST:-localhost}:$${DB_PORT:-5432}/$${TEST_DB_NAME:-bwdcs_test}?sslmode=disable}";

.PHONY: build run test test-dsn vet fmt migrate-up migrate-down migrate-status docker-build docker-run

build: ## Kompilasi binary ke bin/bwdcs
	$(GO) build -o bin/bwdcs ./cmd/server

run: ## Jalankan server dari sumber (membaca ../.env)
	@$(LOAD_ENV) $(GO) run ./cmd/server

test: ## Jalankan seluruh test (integrasi memakai database terpisah dari database dev)
	@$(LOAD_ENV) \
	if [ -z "$$DB_PASSWORD" ]; then \
		echo "DB_PASSWORD kosong: isi ../.env (daftar variabel: 60-DEPLOYMENT.md §2.1)"; exit 1; \
	fi; \
	$(DERIVE_TEST_DSN) \
	if [ -z "$$TEST_DATABASE_URL" ]; then \
		echo "TEST_DATABASE_URL kosong dan tidak dapat diturunkan; lihat 70-TESTING.md §8.1"; exit 1; \
	fi; \
	test_db="$$(printf '%s' "$$TEST_DATABASE_URL" | sed -E 's|^[^/]*//[^/]*/||' | sed -E 's|\?.*$$||')"; \
	if [ "$$test_db" = "$$DB_NAME" ]; then \
		echo "TEST_DATABASE_URL menunjuk database DEV ('$$test_db'): suite akan merusak data dev dan gagal palsu (C-038)."; \
		echo "Arahkan ke database terpisah (mis. $${TEST_DB_NAME:-bwdcs_test}); lihat 70-TESTING.md §8.1."; exit 1; \
	fi; \
	echo "test → database '$$test_db' di $$(printf '%s' "$$TEST_DATABASE_URL" | $(MASK_DSN))"; \
	$(GO) test ./... -cover -p 1 -count=1

test-dsn: ## Cetak DSN database test yang akan dipakai (sandi disamarkan)
	@$(LOAD_ENV) $(DERIVE_TEST_DSN) printf '%s\n' "$$TEST_DATABASE_URL" | $(MASK_DSN)

vet: ## Analisis statis standar
	$(GO) vet ./...

fmt: ## Rapikan format seluruh berkas Go
	$(GO) fmt ./...

migrate-up: ## Terapkan migrasi yang belum jalan
	@$(LOAD_ENV) $(REQUIRE_GOOSE) $(GOOSE) -dir internal/migration postgres "$${DATABASE_URL:-$(DB_DSN)}" up

migrate-down: ## Batalkan satu migrasi terakhir
	@$(LOAD_ENV) $(REQUIRE_GOOSE) $(GOOSE) -dir internal/migration postgres "$${DATABASE_URL:-$(DB_DSN)}" down

migrate-status: ## Tampilkan status migrasi
	@$(LOAD_ENV) $(REQUIRE_GOOSE) $(GOOSE) -dir internal/migration postgres "$${DATABASE_URL:-$(DB_DSN)}" status

docker-build: ## Bangun image backend
	docker build -t bwdcs/backend:latest .

docker-run: ## Jalankan stack compose dari root repo
	docker compose -f ../docker-compose.yml up -d
