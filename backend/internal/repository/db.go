// Package repository menangani akses data BWDCS (`40-TSD.md` §2.5).
//
// Aturan yang mengikat (ADR-0011 butir 3): repository menerima `DBTX`
// (interface berisi `Exec`, `Query`, `QueryRow`), bukan `*pgxpool.Pool`
// langsung, supaya service dapat menggabungkan beberapa operasi — termasuk
// penulisan audit log — ke dalam satu transaksi.
//
// Satu folder = satu package, tanpa subfolder per modul (ADR-0013): modul
// dipisahkan oleh nama berkas (`user_repository.go`, `token_revocation_repository.go`).
package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX adalah bagian pgx yang dibutuhkan repository. Dipenuhi oleh
// `*pgxpool.Pool` (produksi) maupun `pgx.Tx` (satu transaksi).
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// ErrDuplicate dikembalikan repository saat operasi menabrak constraint unik
// (mis. `projects (organization_id, code)` atau keanggotaan project ganda).
// Service memetakannya ke `409 CONFLICT` (`42-API.md` §12), bukan ke 500.
var ErrDuplicate = errors.New("data sudah ada")

// ErrNoUpdateFields dikembalikan saat permintaan pembaruan tidak membawa satu
// pun kolom yang dapat diubah. Service memetakannya ke `422 VALIDATION_ERROR`.
var ErrNoUpdateFields = errors.New("tidak ada field yang dapat diperbarui")

// ErrNotFound dikembalikan repository saat baris yang diminta tidak ada.
// Service memetakannya ke perilaku domain (mis. login gagal), bukan ke 404
// secara otomatis — pesan 404 untuk username yang tidak ada akan membocorkan
// daftar user.
var ErrNotFound = errors.New("baris tidak ditemukan")

// wrapNotFound menyeragamkan `pgx.ErrNoRows` menjadi ErrNotFound.
func wrapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
