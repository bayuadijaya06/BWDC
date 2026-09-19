package service

import (
	"sync"
	"time"
)

// LoginGuard membatasi percobaan login gagal: FR-AUTH-06 (5 gagal / 15 menit)
// dengan angka yang dibaca dari `system_settings` (`44-SECURITY.md` §2.3).
//
// Penghitungnya hidup **di memori proses**, mengikuti `44-SECURITY.md` §2.3
// yang memang menggambarkan `getRecentFailedAttempts` sebagai keadaan runtime,
// bukan tabel. Konsekuensinya dicatat jujur:
//
//   - penghitung hilang saat proses restart (jendela 15 menit kembali nol);
//   - bila kelak berjalan multi-instance, setiap instance punya penghitungnya
//     sendiri sehingga batas efektifnya N kali.
//
// Menyimpannya di database berarti menambah tabel/kolom — dan "Account Lockout
// auto-lock, unlocked by admin" sudah tercatat sebagai temuan terbuka **C-009**
// yang menunggu keputusan: tabel `users` hanya punya `is_active`, tidak ada
// `locked_until`/`failed_login_count`. Karena itu guard ini **hanya** menutup
// FR-AUTH-06 (rate limit per username); ia bukan auto-lock yang dapat dibuka
// admin, dan tidak boleh dipresentasikan sebagai itu.
type LoginGuard struct {
	maxAttempts int
	window      time.Duration
	now         func() time.Time

	mu       sync.Mutex
	attempts map[string][]time.Time
}

// NewLoginGuard membuat pembatas dengan ambang dari kebijakan login.
func NewLoginGuard(maxAttempts int, window time.Duration) *LoginGuard {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	if window <= 0 {
		window = 15 * time.Minute
	}
	return &LoginGuard{
		maxAttempts: maxAttempts,
		window:      window,
		now:         time.Now,
		attempts:    make(map[string][]time.Time),
	}
}

// Allowed menjawab apakah `key` (username) masih boleh mencoba login. Bila
// tidak, ia mengembalikan lama waktu tunggu yang tersisa.
func (g *LoginGuard) Allowed(key string) (bool, time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()

	recent := g.recentLocked(key)
	if len(recent) < g.maxAttempts {
		return true, 0
	}

	// Percobaan tertua menentukan kapan jendela bergeser.
	retryAfter := recent[0].Add(g.window).Sub(g.now())
	if retryAfter < 0 {
		retryAfter = 0
	}
	return false, retryAfter
}

// RecordFailure mencatat satu percobaan gagal.
func (g *LoginGuard) RecordFailure(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.attempts[key] = append(g.recentLocked(key), g.now())
}

// Reset menghapus riwayat percobaan gagal setelah login berhasil.
func (g *LoginGuard) Reset(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.attempts, key)
}

// recentLocked mengembalikan percobaan di dalam jendela waktu dan sekaligus
// membuang yang sudah kedaluwarsa. Pemanggil wajib memegang `g.mu`.
func (g *LoginGuard) recentLocked(key string) []time.Time {
	cutoff := g.now().Add(-g.window)
	kept := g.attempts[key][:0]

	for _, at := range g.attempts[key] {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}

	if len(kept) == 0 {
		delete(g.attempts, key)
		return nil
	}
	g.attempts[key] = kept
	return kept
}
