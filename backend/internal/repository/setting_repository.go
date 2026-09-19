package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SettingRepository membaca `system_settings` (`41-DATABASE.md` §2.6).
type SettingRepository struct {
	db DBTX
}

// NewSettingRepository membuat repository di atas pool atau transaksi.
func NewSettingRepository(db DBTX) *SettingRepository {
	return &SettingRepository{db: db}
}

// AuthPolicy adalah kebijakan login yang dibaca dari `system_settings`.
//
// Nilainya berasal dari seed migrasi `002` (kunci `auth.max_login_attempts`
// dan `auth.lockout_duration_minutes`), bukan dari environment variable —
// daftar environment variable tetap satu sumber di `60-DEPLOYMENT.md` §2.1.
type AuthPolicy struct {
	MaxLoginAttempts int
	LockoutDuration  time.Duration
}

// DefaultAuthPolicy dipakai bila barisnya tidak ada (mis. database di-seed
// versi lama). Angkanya sengaja sama dengan seed `002` dan `44-SECURITY.md` §2.3.
var DefaultAuthPolicy = AuthPolicy{MaxLoginAttempts: 5, LockoutDuration: 15 * time.Minute}

// AuthPolicy membaca kebijakan login. Baris yang hilang memakai default
// (bukan error), tetapi nilai yang tidak dapat dibaca adalah error: kebijakan
// keamanan tidak boleh diam-diam jatuh ke nilai lain.
func (r *SettingRepository) AuthPolicy(ctx context.Context) (AuthPolicy, error) {
	rows, err := r.db.Query(ctx, `
		SELECT key, value
		FROM system_settings
		WHERE key IN ('auth.max_login_attempts', 'auth.lockout_duration_minutes')`)
	if err != nil {
		return AuthPolicy{}, fmt.Errorf("baca kebijakan login: %w", err)
	}
	defer rows.Close()

	policy := DefaultAuthPolicy
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return AuthPolicy{}, fmt.Errorf("scan system_settings: %w", err)
		}

		switch key {
		case "auth.max_login_attempts":
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || n < 1 {
				return AuthPolicy{}, fmt.Errorf("system_settings %s = %q bukan jumlah percobaan yang sah", key, value)
			}
			policy.MaxLoginAttempts = n
		case "auth.lockout_duration_minutes":
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || n < 1 {
				return AuthPolicy{}, fmt.Errorf("system_settings %s = %q bukan menit yang sah", key, value)
			}
			policy.LockoutDuration = time.Duration(n) * time.Minute
		}
	}
	if err := rows.Err(); err != nil {
		return AuthPolicy{}, fmt.Errorf("iterasi system_settings: %w", err)
	}

	return policy, nil
}
