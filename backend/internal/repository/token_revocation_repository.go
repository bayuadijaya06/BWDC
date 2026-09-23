package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// DefaultRevocationCacheTTL adalah TTL cache in-memory untuk pengecekan
// revokasi. `44-SECURITY.md` §2.2 menetapkan maksimum 30 detik: cukup untuk
// menghindari satu query per request, dan tetap membatasi jendela saat
// instance lain belum melihat revokasi.
const DefaultRevocationCacheTTL = 30 * time.Second

// Reason* adalah alasan revokasi yang dipakai sistem. Daftarnya mengikuti tabel
// di `44-SECURITY.md` §2.2; kolom `token_revocations.reason` menyimpannya.
const (
	ReasonLogout             = "logout"
	ReasonLogoutAll          = "logout_all"
	ReasonPasswordChanged    = "password_changed"
	ReasonAdminReset         = "admin_reset"
	ReasonAccountDeactivated = "account_deactivated"
)

// sessionVerdict adalah hasil pengecekan satu token: sudah tidak sah atau belum.
// `userID` ikut disimpan supaya pencabutan seluruh sesi satu user
// (`RevokeAllForUser`) dapat membuang SELURUH entri milik user itu sekaligus —
// tanpa itu, instance yang sama masih menganggap token perangkat lain sah
// sampai TTL habis, padahal ADR-0021 dimaksudkan mencabutnya.
type sessionVerdict struct {
	revoked   bool
	userID    uuid.UUID
	expiresAt time.Time
}

// revocationCache adalah cache hasil pengecekan sesi. Dimiliki bersama oleh
// repository dan salinan yang terikat transaksi (`WithTx`) agar invalidasi
// seketika saat logout juga terlihat oleh pemeriksa lain di instance yang sama.
type revocationCache struct {
	mu      sync.Mutex
	ttl     time.Duration
	now     func() time.Time
	entries map[uuid.UUID]sessionVerdict
}

func newRevocationCache(ttl time.Duration) *revocationCache {
	return &revocationCache{
		ttl:     ttl,
		now:     time.Now,
		entries: make(map[uuid.UUID]sessionVerdict),
	}
}

func (c *revocationCache) load(jti uuid.UUID) (bool, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[jti]
	if !ok {
		return false, false
	}
	if c.now().After(entry.expiresAt) {
		delete(c.entries, jti)
		return false, false
	}
	return entry.revoked, true
}

func (c *revocationCache) store(jti, userID uuid.UUID, revoked bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[jti] = sessionVerdict{revoked: revoked, userID: userID, expiresAt: c.now().Add(c.ttl)}
}

// invalidateUser membuang seluruh entri milik satu user. Dipanggil sesudah
// `tokens_invalid_before` user itu disetel ulang: seluruh tokennya yang mungkin
// sudah tercatat "sah" di cache harus dinilai ulang.
func (c *revocationCache) invalidateUser(userID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for jti, entry := range c.entries {
		if entry.userID == userID {
			delete(c.entries, jti)
		}
	}
}

func (c *revocationCache) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[uuid.UUID]sessionVerdict)
}

// RevocationRepository menyimpan `jti` yang dicabut di tabel `token_revocations`
// (ADR-0009) dan membungkus pengecekan sesi dengan cache in-memory ber-TTL.
//
// Dua mekanisme pencabutan hidup di sini karena keduanya menjawab pertanyaan
// yang sama dari middleware — "apakah sesi token ini masih sah":
//
//   - `Revoke`: satu token tertentu, lewat tabel `token_revocations` (ADR-0009);
//   - `RevokeAllForUser`: seluruh token satu user, lewat kolom
//     `users.tokens_invalid_before` (ADR-0021).
type RevocationRepository struct {
	db    DBTX
	cache *revocationCache
}

// NewRevocationRepository membuat repository di atas pool atau transaksi.
// `ttl` <= 0 jatuh ke DefaultRevocationCacheTTL.
func NewRevocationRepository(db DBTX, ttl time.Duration) *RevocationRepository {
	if ttl <= 0 {
		ttl = DefaultRevocationCacheTTL
	}
	return &RevocationRepository{db: db, cache: newRevocationCache(ttl)}
}

// WithTx mengembalikan repository yang terikat pada satu transaksi, sehingga
// revokasi dan entri audit log-nya commit bersama (ADR-0011 butir 1). Cache
// dibagi dengan repository asal supaya invalidasi seketika tetap berlaku.
func (r *RevocationRepository) WithTx(tx pgx.Tx) *RevocationRepository {
	return &RevocationRepository{db: tx, cache: r.cache}
}

// Revoke mencatat `jti` sebagai dicabut.
//
// Idempotent (`ON CONFLICT DO NOTHING`): memanggil ulang dengan `jti` yang sama
// tetap berhasil — kontrak logout di `42-API.md` §2 minta perilaku itu supaya
// request yang terulang (mis. jaringan putus setelah commit) tidak jadi error.
func (r *RevocationRepository) Revoke(ctx context.Context, jti, userID uuid.UUID, reason string, expiresAt time.Time) error {
	if _, err := r.db.Exec(ctx, `
		INSERT INTO token_revocations (jti, user_id, reason, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (jti) DO NOTHING`, jti, userID, reason, expiresAt); err != nil {
		return fmt.Errorf("catat revokasi token: %w", err)
	}

	// Invalidasi seketika di instance ini; instance lain menunggu TTL (maksimum 30 detik).
	r.cache.store(jti, userID, true)
	return nil
}

// RevokeAllForUser mencabut SELURUH token satu user (ADR-0021): kolom
// `users.tokens_invalid_before` disetel ke detik berjalan, sehingga setiap token
// dengan `iat` lebih tua tidak sah lagi — termasuk token yang terbit **sebelum**
// kolom ini ada.
//
// Nilainya **dipotong ke detik** (`date_trunc('second', NOW())`) dan itu
// disengaja: `iat` JWT berpresisi detik, sehingga `NOW()` mentah akan menolak
// token yang diterbitkan pada detik yang sama — termasuk token hasil login
// ulang tepat sesudah logout (pengguna ter-logout sendiri). Konsekuensi yang
// diterima: token lain yang terbit pada detik yang sama ikut selamat, dengan
// jendela maksimum satu detik (catatan "presisi satu detik" di ADR-0021).
func (r *RevocationRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE users
		SET tokens_invalid_before = date_trunc('second', NOW()), updated_at = NOW()
		WHERE id = $1`, userID)
	if err != nil {
		return fmt.Errorf("setel tokens_invalid_before: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}

	r.cache.invalidateUser(userID)
	return nil
}

// SessionRevoked menjawab apakah sesi sebuah token sudah tidak sah, memakai
// **satu** kueri untuk kedua sebab (ADR-0009 + ADR-0021) dan cache ber-TTL agar
// tidak menambah satu round-trip per request (`40-TSD.md` §5.2.1).
//
// Dua sebab itu sengaja tidak dibedakan di sini maupun di response: klien hanya
// perlu tahu bahwa ia harus login lagi (ADR-0021 butir 5).
//
// User yang barisnya sudah tidak ada dianggap tercabut — gagal-tertutup, karena
// token milik user yang tidak ada tidak mungkin sah.
func (r *RevocationRepository) SessionRevoked(ctx context.Context, jti, userID uuid.UUID, issuedAt time.Time) (bool, error) {
	if revoked, ok := r.cache.load(jti); ok {
		return revoked, nil
	}

	var invalidBefore time.Time
	var revoked bool
	err := r.db.QueryRow(ctx, `
		SELECT u.tokens_invalid_before,
		       EXISTS (SELECT 1 FROM token_revocations tr WHERE tr.jti = $1)
		FROM users u
		WHERE u.id = $2`, jti, userID).Scan(&invalidBefore, &revoked)
	if errors.Is(err, pgx.ErrNoRows) {
		r.cache.store(jti, userID, true)
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("periksa sesi token: %w", err)
	}

	if !revoked && issuedAt.Before(invalidBefore) {
		revoked = true
	}

	r.cache.store(jti, userID, revoked)
	return revoked, nil
}

// CleanupExpired menghapus baris yang sudah lewat `expires_at` (ADR-0009 butir 5).
// Dijalankan saat startup dan berkala dari `cmd/server/main.go`.
//
// `users.tokens_invalid_before` tidak perlu dibersihkan siapa pun: ia satu kolom
// per user, bukan tabel yang tumbuh (ADR-0021 butir 1).
func (r *RevocationRepository) CleanupExpired(ctx context.Context) (int64, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM token_revocations WHERE expires_at < NOW()`)
	if err != nil {
		return 0, fmt.Errorf("bersihkan token_revocations: %w", err)
	}
	return tag.RowsAffected(), nil
}

// ResetCache mengosongkan cache. Dipakai test supaya tidak bergantung pada TTL.
func (r *RevocationRepository) ResetCache() { r.cache.reset() }
