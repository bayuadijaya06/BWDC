// Package dto memuat payload request/response; satu berkas per modul
// (`40-TSD.md` §2.0). Struct di sini adalah batas sistem: bentuk JSON-nya
// mengikuti `42-API.md`, bukan nama kolom database.
package dto

import (
	"time"

	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
)

// LoginRequest adalah body `POST /auth/login` (`42-API.md` §2).
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LogoutRequest adalah body opsional `POST /auth/logout`.
type LogoutRequest struct {
	LogoutAll bool `json:"logout_all"`
}

// UserSummary adalah bentuk user pada response login (`42-API.md` §2).
type UserSummary struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Roles    []string  `json:"roles"`
}

// LoginResponse adalah response 200 `POST /auth/login`.
type LoginResponse struct {
	Token     string      `json:"token"`
	ExpiresAt time.Time   `json:"expires_at"`
	User      UserSummary `json:"user"`
}

// ProfileResponse adalah response 200 `GET /auth/me`: profil + daftar izin
// efektif, sehingga frontend tidak perlu menebak matriks `44-SECURITY.md` §3.1.
type ProfileResponse struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	IsActive       bool      `json:"is_active"`
	Roles          []string  `json:"roles"`
	Permissions    []string  `json:"permissions"`
}

// NewUserSummary memetakan model user ke ringkasan response.
func NewUserSummary(user *model.User) UserSummary {
	roles := user.Roles
	if roles == nil {
		roles = []string{}
	}
	return UserSummary{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Roles:    roles,
	}
}
