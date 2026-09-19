// Package jwt menerbitkan dan memvalidasi access token BWDCS.
//
// Kontrak: docs/design/40-TSD.md §5.2 dan docs/design/44-SECURITY.md §2.2.
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

// Config adalah pengaturan penerbitan token.
type Config struct {
	Secret string
	Expiry time.Duration
}

// Claims adalah klaim yang dibawa access token. `jti` berasal dari
// `RegisteredClaims.ID`.
type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	OrgID    uuid.UUID `json:"org_id"`
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

// Generate menerbitkan token baru untuk user. `jti` selalu UUID v4 baru,
// sehingga dua token untuk user yang sama tidak pernah berbagi kunci revokasi.
func (s *Service) Generate(userID, orgID uuid.UUID, username string) (Token, error) {
	issuedAt := s.now()
	expiresAt := issuedAt.Add(s.expiry)
	jti := uuid.New()

	claims := Claims{
		UserID:   userID,
		Username: username,
		OrgID:    orgID,
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

// Validate memeriksa tanda tangan, `iss`, `exp`, dan bentuk `jti`, lalu
// mengembalikan klaimnya.
func (s *Service) Validate(tokenString string) (*Claims, error) {
	claims := &Claims{}

	if _, err := s.parser.ParseWithClaims(tokenString, claims, func(*jwt.Token) (any, error) {
		return s.secret, nil
	}); err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
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
