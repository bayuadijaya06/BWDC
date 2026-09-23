package repository

import (
	"context"
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// LoginAttempt adalah satu baris `login_attempts` (`41-DATABASE.md` §2.1).
//
// Ini **telemetri keamanan**, bukan audit kepatuhan: baris di sini bertahan
// lintas restart dan lintas instance (inti ADR-0022), sementara `audit_logs`
// tetap bermakna "tindakan aktor yang terautentikasi" dan karena itu tidak
// menerima percobaan yang gagal (temuan C-035).
type LoginAttempt struct {
	// UsernameAttempted adalah input mentah dari klien. Ia **tidak** ber-FK ke
	// `users`: percobaan atas username yang tidak ada justru yang paling perlu
	// tercatat, dan itulah yang tidak dapat dilakukan `audit_logs`.
	UsernameAttempted string

	// UserID diisi hanya bila user-nya ada; `nil` untuk username tak dikenal.
	UserID *uuid.UUID

	// IPAddress `nil` bila alamat klien tidak dapat diurai.
	IPAddress *netip.Addr

	UserAgent     string
	Succeeded     bool
	CorrelationID string
}

// LoginAttemptRepository menulis dan menghitung percobaan login.
type LoginAttemptRepository struct {
	db DBTX
}

// NewLoginAttemptRepository membuat repository di atas pool atau transaksi.
func NewLoginAttemptRepository(db DBTX) *LoginAttemptRepository {
	return &LoginAttemptRepository{db: db}
}

// WithTx mengembalikan repository yang terikat pada satu transaksi.
func (r *LoginAttemptRepository) WithTx(tx pgx.Tx) *LoginAttemptRepository {
	return &LoginAttemptRepository{db: tx}
}

// Record menulis satu percobaan login — berhasil maupun gagal (ADR-0022 butir 1).
//
// Kegagalan menulis telemetri TIDAK boleh menggagalkan login: pemanggil
// mencatatnya di log aplikasi dan melanjutkan, karena menolak login hanya
// karena tabel telemetri bermasalah berarti mematikan sistem tanpa alasan
// keamanan.
func (r *LoginAttemptRepository) Record(ctx context.Context, attempt LoginAttempt) error {
	if _, err := r.db.Exec(ctx, `
		INSERT INTO login_attempts (username_attempted, user_id, ip_address, user_agent, succeeded, correlation_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		attempt.UsernameAttempted, attempt.UserID, attempt.IPAddress,
		nullIfEmpty(attempt.UserAgent), attempt.Succeeded, nullIfEmpty(attempt.CorrelationID),
	); err != nil {
		return fmt.Errorf("catat percobaan login: %w", err)
	}
	return nil
}

// CountRecentFailures menghitung percobaan GAGAL untuk satu username di dalam
// jendela `window` (FR-AUTH-06; ADR-0022 butir 3).
//
// Perbandingannya memakai jam DATABASE, sama dengan sumber `created_at` dan
// `users.locked_until` — memakai jam proses aplikasi akan membuat hitungan dan
// lock memakai dua jam yang berbeda.
//
// Hitungannya **per username yang dicoba**, bukan per user: username yang tidak
// ada pun ikut dihitung, sehingga menebak lewat daftar username tidak membuat
// ambangnya lebih longgar.
func (r *LoginAttemptRepository) CountRecentFailures(ctx context.Context, username string, window time.Duration) (int, error) {
	var count int
	if err := r.db.QueryRow(ctx, `
		SELECT count(*)
		FROM login_attempts
		WHERE username_attempted = $1
		  AND succeeded = false
		  AND created_at > NOW() - make_interval(secs => $2)`,
		username, window.Seconds(),
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("hitung percobaan login gagal: %w", err)
	}
	return count, nil
}

// nullIfEmpty mengubah string kosong menjadi NULL, supaya kolom opsional tidak
// berisi string kosong yang menyamar sebagai data.
func nullIfEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
