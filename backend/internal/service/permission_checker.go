package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"bwdcs/backend/internal/repository"
)

// PermissionChecker menjawab "boleh atau tidak" berdasarkan tabel
// `role_permissions` — sumbernya matriks `44-SECURITY.md` §3.1, di-seed
// migrasi `008` (ADR-0014).
//
// Yang diperiksa HANYA izin. *Baris mana* yang boleh disentuh (cakupan project,
// task milik siapa) bukan urusan pemeriksa ini: `44-SECURITY.md` §3.1.3
// mewajibkan cakupan diterapkan di kueri/service pemilik datanya. Menggabungkan
// keduanya di sini akan membuat izin dan cakupan bercampur — kelas masalah C-008.
type PermissionChecker struct {
	users *repository.UserRepository
}

// NewPermissionChecker membuat pemeriksa izin di atas repository user.
func NewPermissionChecker(users *repository.UserRepository) *PermissionChecker {
	return &PermissionChecker{users: users}
}

// HasPermission mengembalikan true bila user memiliki pasangan (resource, action).
//
// Tidak ada bypass "administrator selalu boleh": matriks sudah memberikan
// seluruh 44 izin kepada Administrator, dan `70-TESTING.md` §4.1 melarang bypass
// di kode dijadikan bukti pemenuhan FR-ROLE-03.
func (p *PermissionChecker) HasPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	allowed, err := p.users.HasPermission(ctx, userID, resource, action)
	if err != nil {
		return false, fmt.Errorf("periksa izin %s:%s: %w", resource, action, err)
	}
	return allowed, nil
}

// Permissions mengembalikan seluruh pasangan izin efektif user, dipakai
// `GET /auth/me` (`42-API.md` §2) supaya frontend dapat menyembunyikan menu
// tanpa menebak matriks.
func (p *PermissionChecker) Permissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	perms, err := p.users.Permissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(perms))
	for _, perm := range perms {
		out = append(out, perm.String())
	}
	return out, nil
}
