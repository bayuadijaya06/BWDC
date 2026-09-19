package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

// envKeys adalah seluruh variabel yang dibaca `Load`.
var envKeys = []string{
	"APP_ENV", "APP_PORT", "LOG_LEVEL",
	"DB_HOST", "DB_PORT", "DB_NAME", "DB_USER", "DB_PASSWORD",
	"JWT_SECRET", "JWT_EXPIRY",
	"STORAGE_TYPE", "STORAGE_PATH", "REDIS_URL",
	"ADMIN_ORG_NAME", "ADMIN_ORG_CODE", "ADMIN_USERNAME", "ADMIN_PASSWORD", "ADMIN_EMAIL",
}

// clearEnv menghapus seluruh variabel yang dibaca `Load` sebelum test berjalan
// (dan mengembalikannya setelah test selesai).
//
// Alasannya: `Load` memanggil `viper.AutomaticEnv()`, sehingga environment yang
// sudah ter-export SELALU menang atas berkas `.env`. Pengembang yang menjalankan
// `set -a; . .env; set +a` lalu `go test ./...` akan mendapat kegagalan palsu —
// itu utang `T-033`, dan helper ini yang menutupnya: test menjadi hermetis
// terhadap environment pemanggil.
func clearEnv(t *testing.T) {
	t.Helper()

	for _, key := range envKeys {
		key := key
		previous, existed := os.LookupEnv(key)
		if existed {
			t.Cleanup(func() { _ = os.Setenv(key, previous) })
		} else {
			t.Cleanup(func() { _ = os.Unsetenv(key) })
		}
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("hapus variabel %s: %v", key, err)
		}
	}
}

// setEnvBaseline mengisi environment minimal yang sah supaya tiap test hanya
// perlu mengubah variabel yang sedang diuji.
func setEnvBaseline(t *testing.T, jwtSecret string) {
	t.Helper()

	clearEnv(t)

	t.Setenv("DB_PASSWORD", "rahasia-test")
	t.Setenv("JWT_SECRET", jwtSecret)
	t.Setenv("JWT_EXPIRY", "24h")
	t.Setenv("APP_PORT", "8081")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_NAME", "bwdcs")
	t.Setenv("DB_USER", "bwdcs")
	t.Setenv("STORAGE_TYPE", "local")
	t.Setenv("STORAGE_PATH", "./storage")
}

func TestLoadMembacaEnvironment(t *testing.T) {
	setEnvBaseline(t, strings.Repeat("k", minJWTSecretLen))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Server.Port != 8081 {
		t.Fatalf("APP_PORT = %d, ingin 8081", cfg.Server.Port)
	}
	if cfg.Database.Name != "bwdcs" || cfg.Database.Password != "rahasia-test" {
		t.Fatalf("konfigurasi database tidak terbaca: %+v", cfg.Database)
	}
	if cfg.JWT.Expiry != 24*time.Hour {
		t.Fatalf("JWT_EXPIRY = %s, ingin 24h", cfg.JWT.Expiry)
	}
	if !strings.Contains(cfg.Database.DSN(), "dbname=bwdcs") {
		t.Fatalf("DSN tidak memuat dbname: %s", cfg.Database.DSN())
	}
}

func TestLoadGagalBilaEnvWajibKosong(t *testing.T) {
	setEnvBaseline(t, strings.Repeat("k", minJWTSecretLen))
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("JWT_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load seharusnya gagal bila DB_PASSWORD dan JWT_SECRET kosong")
	}

	// Pesan harus menyebut keduanya sekaligus, bukan satu per percobaan.
	for _, want := range []string{"DB_PASSWORD", "JWT_SECRET"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("pesan error tidak menyebut %s: %v", want, err)
		}
	}
}

func TestLoadMenolakJWTSecretTerlaluPendek(t *testing.T) {
	setEnvBaseline(t, "pendek")

	_, err := Load()
	if err == nil {
		t.Fatal("Load seharusnya gagal untuk JWT_SECRET kurang dari 32 karakter")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("pesan error tidak menyebut JWT_SECRET: %v", err)
	}
}

func TestLoadMenolakDurasiDanStorageTypeTidakValid(t *testing.T) {
	t.Run("JWT_EXPIRY bukan durasi", func(t *testing.T) {
		setEnvBaseline(t, strings.Repeat("k", minJWTSecretLen))
		t.Setenv("JWT_EXPIRY", "sehari")

		if _, err := Load(); err == nil {
			t.Fatal("Load seharusnya gagal untuk JWT_EXPIRY yang bukan durasi")
		}
	})

	t.Run("STORAGE_TYPE di luar MVP", func(t *testing.T) {
		setEnvBaseline(t, strings.Repeat("k", minJWTSecretLen))
		t.Setenv("STORAGE_TYPE", "s3")

		_, err := Load()
		if err == nil {
			t.Fatal("Load seharusnya gagal untuk STORAGE_TYPE yang belum didukung")
		}
		if !strings.Contains(err.Error(), "ADR-0005") {
			t.Fatalf("pesan error tidak menunjuk ADR-0005: %v", err)
		}
	})
}

func TestLoadMemakaiDefaultNonRahasia(t *testing.T) {
	clearEnv(t)

	// Hanya rahasia yang diisi; sisanya harus jatuh ke default yang tidak sensitif.
	t.Setenv("DB_PASSWORD", "rahasia-test")
	t.Setenv("JWT_SECRET", strings.Repeat("k", minJWTSecretLen))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Server.Env != "development" || cfg.Server.Port != 8080 {
		t.Fatalf("default server tidak dipakai: %+v", cfg.Server)
	}
	if cfg.Storage.Type != "local" || cfg.Redis.URL != "" {
		t.Fatalf("default storage/redis tidak dipakai: %+v %+v", cfg.Storage, cfg.Redis)
	}
	if cfg.Server.IsProduction() {
		t.Fatal("APP_ENV default seharusnya bukan production")
	}
}
