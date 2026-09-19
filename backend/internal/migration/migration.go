// Package migration menjalankan migrasi skema BWDCS.
//
// Sumber skema: docs/design/41-DATABASE.md (§2 DDL, §4 strategi migrasi).
// Berkas `.sql` di direktori ini di-embed ke binary dan dijalankan dengan
// `goose` sebagai library, sesuai "Migrasi dijalankan saat startup aplikasi"
// (`41-DATABASE.md` §4) dan urutan startup ADR-0010 butir 1.
//
// # Kenapa goose dipin ke v3.24.1
//
// Toolchain mesin kerja masih Go 1.22.5 (STATE.md §2), sementara goose
// >= v3.24.2 menaikkan direktif `go` di go.mod-nya ke 1.23.0 atau lebih
// (v3.28.0: 1.26.0). Pin ini sebabnya sama dengan pin `jackc/pgx/v5` v5.7.4:
// menahan kenaikan toolchain yang belum diputuskan. **CLI `goose` dipasang pada
// versi yang sama** supaya migrasi manual (`make migrate-up`) dan migrasi saat
// startup memakai satu versi yang identik — dua versi berbeda akan membuat
// `goose status` dan perilaku startup dapat berbeda tanpa terlihat.
//
// Berkas SQL tetap sumber kebenarannya (bukan kode Go); paket ini hanya
// menjalankannya.
package migration

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"strings"

	// Driver database/sql untuk goose. Aplikasi sendiri memakai pgxpool
	// (`jackc/pgx/v5`), sementara goose bekerja di atas `database/sql`.
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var files embed.FS

// dir adalah direktori di dalam embed.FS. Berkas SQL berada di akar direktori
// paket ini, sehingga goose membaca "." — bukan "internal/migration".
const dir = "."

// Up menerapkan semua migrasi yang belum dijalankan pada database tujuan DSN.
//
// Aman dipanggil berulang: goose mencatat versi yang sudah diterapkan di tabel
// `goose_db_version` dan melewatinya. Migrasi yang gagal membatalkan
// transaksinya, sehingga database tidak tertinggal di keadaan separuh jalan.
func Up(ctx context.Context, dsn string, logger *slog.Logger) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("buka koneksi migrasi: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database tidak dapat dihubungi untuk migrasi: %w", err)
	}

	goose.SetBaseFS(files)
	goose.SetLogger(&slogLogger{logger: logger})
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialek goose: %w", err)
	}

	if err := goose.UpContext(ctx, db, dir); err != nil {
		return fmt.Errorf("jalankan migrasi: %w", err)
	}

	version, err := goose.GetDBVersionContext(ctx, db)
	if err != nil {
		return fmt.Errorf("baca versi skema setelah migrasi: %w", err)
	}
	logger.Info("migrasi selesai", "versi_skema", version)

	return nil
}

// slogLogger mengarahkan output goose ke slog JSON (`40-TSD.md` §2.2) supaya
// log migrasi dapat dibaca mesin seperti log aplikasi lainnya.
type slogLogger struct {
	logger *slog.Logger
}

func (l *slogLogger) Printf(format string, v ...interface{}) {
	l.logger.Info(strings.TrimSpace(fmt.Sprintf(format, v...)))
}

func (l *slogLogger) Fatalf(format string, v ...interface{}) {
	// Tidak memanggil os.Exit: goose tetap mengembalikan error, dan pemanggil
	// (main) yang memutuskan berhenti dengan pesan yang seragam.
	l.logger.Error(strings.TrimSpace(fmt.Sprintf(format, v...)))
}
