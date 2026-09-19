// Package model memuat struct domain; satu berkas per entitas (`40-TSD.md` §2.3).
//
// Struct di sini merepresentasikan tabel database apa adanya. `PasswordHash`
// tidak pernah ikut ke response (tag `json:"-"`).
package model

import (
	"time"

	"github.com/google/uuid"
)

// User merepresentasikan tabel `users` (`41-DATABASE.md` §2.1).
type User struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	PasswordHash   string    `json:"-"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Roles diisi hanya bila repository diminta mengambilnya (`Roles`).
	Roles []string `json:"roles,omitempty"`
}

// Permission adalah satu baris `role_permissions`: pasangan resource + action
// dari kosakata tertutup `44-SECURITY.md` §3.1.1 (ADR-0014).
type Permission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// String mengembalikan bentuk `resource:action` yang dipakai response profil
// dan pesan log.
func (p Permission) String() string { return p.Resource + ":" + p.Action }
