package service

import (
	"testing"
	"time"
)

func TestLoginGuardAllowsUntilLimit(t *testing.T) {
	guard := NewLoginGuard(3, 15*time.Minute)
	now := time.Now()
	guard.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if allowed, _ := guard.Allowed("admin"); !allowed {
			t.Fatalf("percobaan ke-%d seharusnya masih diizinkan", i+1)
		}
		guard.RecordFailure("admin")
	}

	allowed, retryAfter := guard.Allowed("admin")
	if allowed {
		t.Fatal("percobaan ke-4 seharusnya ditolak (FR-AUTH-06: 5 gagal / 15 menit)")
	}
	if retryAfter <= 0 {
		t.Fatalf("lama tunggu harus positif, dapat %s", retryAfter)
	}
}

func TestLoginGuardResetsAfterSuccess(t *testing.T) {
	guard := NewLoginGuard(2, 15*time.Minute)
	now := time.Now()
	guard.now = func() time.Time { return now }

	guard.RecordFailure("admin")
	guard.RecordFailure("admin")
	if allowed, _ := guard.Allowed("admin"); allowed {
		t.Fatal("batas tercapai, seharusnya ditolak")
	}

	guard.Reset("admin")
	if allowed, _ := guard.Allowed("admin"); !allowed {
		t.Fatal("setelah login berhasil, penghitung seharusnya nol kembali")
	}
}

func TestLoginGuardWindowExpires(t *testing.T) {
	guard := NewLoginGuard(1, 15*time.Minute)
	now := time.Now()
	guard.now = func() time.Time { return now }

	guard.RecordFailure("admin")
	if allowed, _ := guard.Allowed("admin"); allowed {
		t.Fatal("seharusnya ditolak di dalam jendela")
	}

	now = now.Add(16 * time.Minute)
	if allowed, _ := guard.Allowed("admin"); !allowed {
		t.Fatal("setelah jendela lewat, percobaan seharusnya diizinkan kembali")
	}
}

// TestLoginGuardIsPerUsername memastikan percobaan pada satu akun tidak
// memblokir akun lain.
func TestLoginGuardIsPerUsername(t *testing.T) {
	guard := NewLoginGuard(1, 15*time.Minute)

	guard.RecordFailure("admin")
	if allowed, _ := guard.Allowed("admin"); allowed {
		t.Fatal("admin seharusnya ditolak")
	}
	if allowed, _ := guard.Allowed("manager"); !allowed {
		t.Fatal("akun lain tidak boleh ikut terblokir")
	}
}

func TestLoginGuardUsesConfiguredThreshold(t *testing.T) {
	guard := NewLoginGuard(5, 15*time.Minute)
	for i := 0; i < 4; i++ {
		guard.RecordFailure("viewer")
	}
	if allowed, _ := guard.Allowed("viewer"); !allowed {
		t.Fatal("4 percobaan masih di bawah ambang 5")
	}
}
