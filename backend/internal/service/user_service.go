// Berkas ini memuat bagian Administration > Users dari `42-API.md` §11.
//
// Sampai modul administrasi lain dikerjakan, di sinilah satu-satunya endpoint
// `/admin/*` yang hidup: pembukaan lock akun lebih awal (**ADR-0022** butir 5),
// yang menutup klausa "unlocked by admin" di `44-SECURITY.md` §2.3.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"bwdcs/backend/internal/repository"
)

// ErrUserNotFound dipakai endpoint administrasi user saat sasaran tidak ada.
// Handler memetakannya ke `404 NOT_FOUND` (`42-API.md` §12).
var ErrUserNotFound = errors.New("user tidak ditemukan")

// UserService menjalankan aksi Administrator atas akun user.
type UserService struct {
	pool   *pgxpool.Pool
	users  *repository.UserRepository
	logger *slog.Logger
}

// NewUserService merakit service administrasi user.
func NewUserService(pool *pgxpool.Pool, users *repository.UserRepository, logger *slog.Logger) *UserService {
	return &UserService{pool: pool, users: users, logger: logger}
}

// Unlock membuka lock akun lebih awal (`POST /admin/users/:id/unlock`,
// `42-API.md` §11, ADR-0022 butir 5).
//
// Dua hal yang diputuskan di sini, dan keduanya terlihat dari luar:
//
//   - **Idempoten.** Akun yang memang tidak terkunci dijawab sukses tanpa entri
//     audit kedua (`ClearLock` melaporkan apa yang benar-benar berubah). Yang
//     idempoten adalah *hasilya*: "akun ini tidak terkunci" — bukan "lock
//     dilepas tepat satu kali".
//   - **Hitungan percobaan gagal tidak dihapus.** `login_attempts` adalah
//     telemetri keamanan, dan ADR-0022 butir 7 hanya mengizinkan pemangkasan
//     lewat retensi; menghapus barisnya di sini akan menghilangkan jejak
//     investigasi yang justru alasan tabel itu ada. Konsekuensinya: satu
//     kegagalan **berikutnya** di dalam jendela 15 menit akan mengunci lagi
//     (perilaku backoff), sedangkan password yang benar langsung diterima
//     karena lock hanya diperiksa sebagai keadaan akun.
func (s *UserService) Unlock(ctx context.Context, actorID, userID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mulai transaksi unlock: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	cleared, err := s.users.WithTx(tx).ClearLock(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	if !cleared {
		// Tidak ada yang berubah: tidak ada yang perlu di-commit maupun diaudit.
		return nil
	}

	if err := NewAuditService(tx).Log(ctx, actorID, ActionUserUnlocked, "user", userID.String(),
		"Administrator membuka lock akun lebih awal", map[string]any{
			"unlocked_user_id": userID.String(),
		}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit unlock: %w", err)
	}

	s.logger.Info("lock akun dibuka administrator",
		"actor_id", actorID.String(), "user_id", userID.String())
	return nil
}
