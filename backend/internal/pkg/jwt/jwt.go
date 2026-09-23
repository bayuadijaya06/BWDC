// Package jwt menerbitkan dan memvalidasi token BWDCS.
//
// Kontrak: docs/design/40-TSD.md §5.2, docs/design/44-SECURITY.md §2.2, dan
// ADR-0023 untuk bentuk token refresh.
//
// Ada **dua** jenis token, dan klaim `typ` yang membedakannya: access token
// (masa berlaku `JWT_EXPIRY`, dipakai middleware) dan refresh token (7 hari,
// hanya ditukar di `POST /auth/refresh`). Keduanya memakai pemeriksaan yang sama
// di sini; yang berbeda hanya tipe yang diterima.
//
// Setiap token memuat klaim `jti` (UUID unik per token) sebagai kunci revokasi —
// tanpa `jti`, mekanisme ADR-0009 tidak dapat dijalankan sama sekali.
//
// Algoritma terkunci pada HS256, dan tanda tangan hanya diterima bila klaim
// `iss`, `exp`, dan `iat` ada serta sah. Token tanpa `jti` ditolak walaupun
// tanda tangannya benar: token seperti itu tidak dapat dicabut saat logout.
package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Issuer adalah nilai klaim `iss`. Dikunci di sini (bukan environment variable)
// karena hanya ada satu penerbit token di sistem ini — `44-SECURITY.md` §2.2.
const Issuer = "bwdcs"

// minSecretLength mengikuti `60-DEPLOYMENT.md` §2.1 (JWT_SECRET minimal 32 karakter).
const minSecretLength = 32

// ErrInvalidToken menandai token yang tidak sah (tanda tangan, issuer, klaim,
// atau bentuknya salah). Middleware memetakannya ke 401 `UNAUTHORIZED`.
var ErrInvalidToken = errors.New("token tidak sah")

// ErrExpiredToken menandai token yang tanda tangannya benar tetapi sudah lewat
// `exp`. Dipisahkan dari ErrInvalidToken supaya dapat dibedakan di log dan test.
var ErrExpiredToken = errors.New("token kedaluwarsa")

// TokenType adalah isi klaim `typ`, dan ia yang membuat access token tidak dapat
// ditukar dengan refresh token (ADR-0023).
//
// Klaim ini **wajib**: token tanpa `typ` ditolak. Sebelum ADR-0023, klaim itu
// tidak ada, sehingga token yang terbit sebelum perubahan itu tidak lagi sah —
// konsekuensi yang disengaja (sistem belum berproduksi; yang dibutuhkan hanya
// login ulang) demi satu invarian yang dapat diperiksa: setiap token menyatakan
// sendiri jenisnya.
type TokenType string

const (
	// TypeAccess adalah token bearer yang dipakai middleware untuk endpoint terproteksi.
	TypeAccess TokenType = "access"
	// TypeRefresh hanya dapat ditukar di `POST /auth/refresh`, tidak pernah di endpoint lain.
	TypeRefresh TokenType = "refresh"
)

// RefreshExpiry adalah masa berlaku refresh token: **7 hari**, sesuai
// `44-SECURITY.md` §2.2. Ia konstan di kode, bukan kunci konfigurasi baru:
// angkanya sudah ditetapkan dokumen kebijakan, dan menambah variabel runtime
// untuknya berarti menambah kunci yang harus dijaga di tiga berkas untuk nilai
// yang tidak pernah berbeda (alasan yang sama dengan ADR-0020 butir 2).
// Konsekuensinya disadari: mengubahnya butuh perubahan kode, bukan `.env`.
const RefreshExpiry = 7 * 24 * time.Hour

// Config adalah pengaturan penerbitan token.
type Config struct {
	Secret string
	Expiry time.Duration
}

// Claims adalah klaim yang dibawa token. `jti` berasal dari
// `RegisteredClaims.ID`, dan `Type` (klaim `typ`) membedakan access dari refresh.
type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	OrgID    uuid.UUID `json:"org_id"`
	Type     TokenType `json:"typ"`
	jwt.RegisteredClaims
}

// Token adalah hasil penerbitan: nilai siap kirim plus metadata yang dibutuhkan
// pemanggil (masa berlaku untuk response, `jti` untuk revokasi).
type Token struct {
	Value     string
	JTI       uuid.UUID
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// Service menerbitkan dan memvalidasi token. Aman dipakai bersamaan.
type Service struct {
	secret []byte
	expiry time.Duration
	parser *jwt.Parser
	now    func() time.Time
}

// New memvalidasi konfigurasi lalu membuat service.
func New(cfg Config) (*Service, error) {
	if len(cfg.Secret) < minSecretLength {
		return nil, fmt.Errorf("JWT_SECRET minimal %d karakter, saat ini %d", minSecretLength, len(cfg.Secret))
	}
	if cfg.Expiry <= 0 {
		return nil, fmt.Errorf("JWT_EXPIRY harus durasi positif, saat ini %s", cfg.Expiry)
	}

	return &Service{
		secret: []byte(cfg.Secret),
		expiry: cfg.Expiry,
		parser: jwt.NewParser(
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			jwt.WithIssuer(Issuer),
			jwt.WithExpirationRequired(),
			jwt.WithIssuedAt(),
		),
		now: time.Now,
	}, nil
}

// Expiry mengembalikan masa berlaku token yang dikonfigurasi.
func (s *Service) Expiry() time.Duration { return s.expiry }

// Generate menerbitkan **access token** baru untuk user. `jti` selalu UUID v4
// baru, sehingga dua token untuk user yang sama tidak pernah berbagi kunci
// revokasi.
func (s *Service) Generate(userID, orgID uuid.UUID, username string) (Token, error) {
	return s.issue(userID, orgID, username, TypeAccess, s.expiry)
}

// GenerateRefresh menerbitkan **refresh token** untuk user: token bertipe
// `refresh` dengan masa berlaku `RefreshExpiry` (ADR-0023).
//
// Ia tidak punya jalur pemeriksaan sendiri: pencabutannya dinilai dengan cara
// yang sama seperti access token (denylist `jti` atau
// `iat < users.tokens_invalid_before`), sehingga `logout_all` dan
// `change-password` otomatis membuat refresh token lama tidak berguna.
func (s *Service) GenerateRefresh(userID, orgID uuid.UUID, username string) (Token, error) {
	return s.issue(userID, orgID, username, TypeRefresh, RefreshExpiry)
}

// issue menerbitkan satu token dengan tipe dan masa berlaku yang diminta.
// Hanya dua tipe yang sah, dan keduanya selalu melewati fungsi ini supaya klaim
// `typ` tidak pernah lupa dipasang.
func (s *Service) issue(userID, orgID uuid.UUID, username string, typ TokenType, lifetime time.Duration) (Token, error) {
	issuedAt := s.now()
	expiresAt := issuedAt.Add(lifetime)
	jti := uuid.New()

	claims := Claims{
		UserID:   userID,
		Username: username,
		OrgID:    orgID,
		Type:     typ,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti.String(),
			Issuer:    Issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return Token{}, fmt.Errorf("tandatangani token: %w", err)
	}

	return Token{Value: signed, JTI: jti, IssuedAt: issuedAt, ExpiresAt: expiresAt}, nil
}

// Validate memeriksa tanda tangan, `iss`, `exp`, bentuk `jti`, dan **tipe**
// token, lalu mengembalikan klaimnya. Hanya access token yang diterima: refresh
// token yang dikirim sebagai bearer token ditolak, bukan diperlakukan sebagai
// access token berumur panjang (ADR-0023).
func (s *Service) Validate(tokenString string) (*Claims, error) {
	return s.validate(tokenString, TypeAccess)
}

// ValidateRefresh memeriksa token dengan aturan yang sama, tetapi menuntut tipe
// `refresh` — sehingga access token tidak dapat ditukar di `POST /auth/refresh`.
func (s *Service) ValidateRefresh(tokenString string) (*Claims, error) {
	return s.validate(tokenString, TypeRefresh)
}

// validate adalah inti pemeriksaan bersama. Pemisahannya disengaja: access dan
// refresh token harus lulus pemeriksaan yang **sama** (`jti`, `user_id`, `iat`),
// dan yang berbeda hanya tipe yang diterima.
func (s *Service) validate(tokenString string, want TokenType) (*Claims, error) {
	claims := &Claims{}

	if _, err := s.parser.ParseWithClaims(tokenString, claims, func(*jwt.Token) (any, error) {
		return s.secret, nil
	}); err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if claims.Type != want {
		return nil, fmt.Errorf("%w: tipe token %q, diharapkan %q (ADR-0023)", ErrInvalidToken, claims.Type, want)
	}
	if claims.ID == "" {
		return nil, fmt.Errorf("%w: klaim jti tidak ada sehingga token tidak dapat dicabut (ADR-0009)", ErrInvalidToken)
	}
	if _, err := uuid.Parse(claims.ID); err != nil {
		return nil, fmt.Errorf("%w: klaim jti bukan UUID", ErrInvalidToken)
	}
	if claims.UserID == uuid.Nil {
		return nil, fmt.Errorf("%w: klaim user_id kosong", ErrInvalidToken)
	}
	if claims.IssuedAt == nil {
		// `iat` bukan sekadar metadata: ADR-0021 membandingkannya dengan
		// `users.tokens_invalid_before` untuk mencabut seluruh sesi. Token tanpa
		// `iat` tidak dapat dinilai, jadi ditolak di sini — bukan dilewatkan lalu
		// dianggap sah oleh middleware.
		return nil, fmt.Errorf("%w: klaim iat tidak ada sehingga sesi tidak dapat dinilai (ADR-0021)", ErrInvalidToken)
	}

	return claims, nil
}

// JTI mengembalikan `jti` klaim sebagai UUID.
func (c *Claims) JTI() (uuid.UUID, error) {
	jti, err := uuid.Parse(c.ID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: klaim jti bukan UUID", ErrInvalidToken)
	}
	return jti, nil
}
