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
//
// `refresh_token` + `refresh_expires_at` ikut dikirim sejak ADR-0023: itulah
// modal yang diperlukan `POST /auth/refresh`, dan tanpanya endpoint itu tidak
// dapat dipakai klien mana pun.
type LoginResponse struct {
	Token            string      `json:"token"`
	ExpiresAt        time.Time   `json:"expires_at"`
	RefreshToken     string      `json:"refresh_token"`
	RefreshExpiresAt time.Time   `json:"refresh_expires_at"`
	User             UserSummary `json:"user"`
}

// ChangePasswordRequest adalah body `POST /auth/change-password` (`42-API.md` §2).
//
// `binding` sengaja tidak dipakai: handler memakai `bindJSON` yang membaca nama
// field untuk pesan `422` (temuan C-045), dan keharusan-isi diperiksa eksplisit
// di handler supaya pesannya menyebut field. `confirm_password` pada FSD §2.2
// **tidak** ada di sini: mencocokkan dua ketikan adalah urusan form, bukan
// kontrak API — server hanya menerima password baru yang sudah pasti.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ChangePasswordResponse adalah response 200 `POST /auth/change-password`.
//
// Token baru ikut di response karena endpoint ini mencabut **seluruh** sesi
// user lewat `users.tokens_invalid_before` (ADR-0021 butir 3): tanpa token baru,
// perangkat yang baru saja mengganti password akan ter-logout sendiri.
type ChangePasswordResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// RefreshRequest adalah body `POST /auth/refresh` (`42-API.md` §2).
//
// Yang dikirim adalah **refresh token** dari login/refresh sebelumnya, bukan
// access token yang sedang dipakai header `Authorization`.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse adalah response 200 `POST /auth/refresh`: sepasang token baru.
//
// Bentuknya memuat kedua masa berlaku (`expires_at` untuk access token,
// `refresh_expires_at` untuk refresh token) supaya klien tahu kapan harus
// memperpanjang lagi tanpa membedah tokennya.
type RefreshResponse struct {
	Token            string    `json:"token"`
	ExpiresAt        time.Time `json:"expires_at"`
	RefreshToken     string    `json:"refresh_token"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

// LockedDetails adalah `details` pada `423 LOCKED` (`42-API.md` §12). Bentuknya
// **objek**, bukan daftar `{field, error}` seperti 422: yang dibutuhkan klien
// adalah lama tunggu, bukan nama field yang salah — tidak ada field yang salah
// pada permintaan yang ditolak karena akunnya terkunci.
type LockedDetails struct {
	RetryAfterSeconds int       `json:"retry_after_seconds"`
	LockedUntil       time.Time `json:"locked_until"`
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
