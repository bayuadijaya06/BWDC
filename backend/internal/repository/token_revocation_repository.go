package repository

import (
	"context"
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

// revocationCache adalah cache hasil pengecekan `jti`. Dimiliki bersama oleh
// repository dan salinan yang terikat transaksi (`WithTx`) agar invalidasi
// seketika saat logout juga terlihat oleh pemeriksa lain di instance yang sama.
type revocationCache struct {
	mu      sync.Mutex
	ttl     time.Duration
	now     func() time.Time
	entries map[uuid.UUID]cacheEntry
}

type cacheEntry struct {
	revoked   bool
	expiresAt time.Time
}

func newRevocationCache(ttl time.Duration) *revocationCache {
	return &revocationCache{
		ttl:     ttl,
		now:     time.Now,
		entries: make(map[uuid.UUID]cacheEntry),
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

func (c *revocationCache) store(jti uuid.UUID, revoked bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[jti] = cacheEntry{revoked: revoked, expiresAt: c.now().Add(c.ttl)}
}

func (c *revocationCache) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[uuid.UUID]cacheEntry)
}

// RevocationRepository menyimpan `jti` yang dicabut di tabel `token_revocations`
// (ADR-0009) dan membungkus pengecekannya dengan cache in-memory ber-TTL.
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
	r.cache.store(jti, true)
	return nil
}

// IsRevoked menjawab apakah `jti` sudah dicabut, memakai cache ber-TTL agar
// tidak menambah satu query per request (`40-TSD.md` §5.2.1).
func (r *RevocationRepository) IsRevoked(ctx context.Context, jti uuid.UUID) (bool, error) {
	if revoked, ok := r.cache.load(jti); ok {
		return revoked, nil
	}

	var revoked bool
	if err := r.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM token_revocations WHERE jti = $1)`, jti,
	).Scan(&revoked); err != nil {
		return false, fmt.Errorf("periksa revokasi token: %w", err)
	}

	r.cache.store(jti, revoked)
	return revoked, nil
}

// CleanupExpired menghapus baris yang sudah lewat `expires_at` (ADR-0009 butir 5).
// Dijalankan saat startup dan berkala dari `cmd/server/main.go`.
func (r *RevocationRepository) CleanupExpired(ctx context.Context) (int64, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM token_revocations WHERE expires_at < NOW()`)
	if err != nil {
		return 0, fmt.Errorf("bersihkan token_revocations: %w", err)
	}
	return tag.RowsAffected(), nil
}

// ResetCache mengosongkan cache. Dipakai test supaya tidak bergantung pada TTL.
func (r *RevocationRepository) ResetCache() { r.cache.reset() }
