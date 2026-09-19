// Package config memuat konfigurasi aplikasi dari environment.
//
// Sumber tunggal daftar environment variable: docs/design/60-DEPLOYMENT.md §2.1.
// Kontrak struct: docs/design/40-TSD.md §2.1.
//
// Aturan yang mengikat (12-DEVELOPMENT-WORKFLOW.md §7): aplikasi WAJIB gagal start
// dengan pesan jelas bila env wajib tidak ada. Tidak ada default rahasia yang
// membuat aplikasi terasa jalan padahal tidak terkonfigurasi.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config adalah akar konfigurasi runtime.
type Config struct {
	Server    ServerConfig
	Database  DBConfig
	JWT       JWTConfig
	Storage   StorageConfig
	Redis     RedisConfig
	Bootstrap BootstrapConfig
}

// ServerConfig berisi pengaturan HTTP server.
type ServerConfig struct {
	Env      string // development | production
	Port     int    // APP_PORT
	LogLevel string // debug | info | warn | error
}

// DBConfig berisi kredensial PostgreSQL.
type DBConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
}

// JWTConfig berisi kunci tanda tangan dan masa berlaku token.
type JWTConfig struct {
	Secret string
	Expiry time.Duration
}

// StorageConfig menentukan backend penyimpanan berkas (ADR-0005).
type StorageConfig struct {
	Type string // local (MVP)
	Path string
}

// RedisConfig opsional. URL kosong berarti nonaktif: invalidasi token memakai
// tabel token_revocations (ADR-0009), bukan Redis.
type RedisConfig struct {
	URL string
}

// BootstrapConfig memuat kredensial admin pertama (ADR-0010). Nilai-nilai ini
// hanya dipakai bila tabel users masih kosong; validasinya milik
// internal/bootstrap, bukan package ini.
type BootstrapConfig struct {
	OrgName  string
	OrgCode  string
	Username string
	Password string
	Email    string
}

// minJWTSecretLen mengikuti 60-DEPLOYMENT.md §2.1.
const minJWTSecretLen = 32

// Load membaca environment (dan berkas .env bila ada) lalu memvalidasinya.
func Load() (*Config, error) {
	v := viper.New()
	if err := loadEnvFile(v); err != nil {
		return nil, err
	}
	applyDefaults(v)
	v.AutomaticEnv()

	expiry, err := time.ParseDuration(v.GetString("JWT_EXPIRY"))
	if err != nil {
		return nil, fmt.Errorf("JWT_EXPIRY %q bukan durasi yang sah (contoh: 24h): %w", v.GetString("JWT_EXPIRY"), err)
	}

	cfg := &Config{
		Server: ServerConfig{
			Env:      v.GetString("APP_ENV"),
			Port:     v.GetInt("APP_PORT"),
			LogLevel: strings.ToLower(v.GetString("LOG_LEVEL")),
		},
		Database: DBConfig{
			Host:     v.GetString("DB_HOST"),
			Port:     v.GetInt("DB_PORT"),
			Name:     v.GetString("DB_NAME"),
			User:     v.GetString("DB_USER"),
			Password: v.GetString("DB_PASSWORD"),
		},
		JWT: JWTConfig{
			Secret: v.GetString("JWT_SECRET"),
			Expiry: expiry,
		},
		Storage: StorageConfig{
			Type: strings.ToLower(v.GetString("STORAGE_TYPE")),
			Path: v.GetString("STORAGE_PATH"),
		},
		Redis: RedisConfig{
			URL: strings.TrimSpace(v.GetString("REDIS_URL")),
		},
		Bootstrap: BootstrapConfig{
			OrgName:  v.GetString("ADMIN_ORG_NAME"),
			OrgCode:  v.GetString("ADMIN_ORG_CODE"),
			Username: v.GetString("ADMIN_USERNAME"),
			Password: v.GetString("ADMIN_PASSWORD"),
			Email:    v.GetString("ADMIN_EMAIL"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// DSN menyusun connection string lib/pq-style untuk pgx dan goose.
func (d DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		d.Host, d.Port, d.User, d.Password, d.Name,
	)
}

// IsProduction menandai lingkungan produksi (dipakai untuk memilih perilaku
// logging dan pengamanan cookie nanti).
func (s ServerConfig) IsProduction() bool { return s.Env == "production" }

// validate mengumpulkan SEMUA masalah sekaligus supaya operator tidak memperbaiki
// konfigurasi satu variabel per percobaan.
func (c *Config) validate() error {
	var problems []string

	if c.Database.Password == "" {
		problems = append(problems, "DB_PASSWORD wajib diisi")
	}
	if c.JWT.Secret == "" {
		problems = append(problems, "JWT_SECRET wajib diisi (buat dengan: openssl rand -base64 48)")
	} else if len(c.JWT.Secret) < minJWTSecretLen {
		problems = append(problems, fmt.Sprintf("JWT_SECRET minimal %d karakter, saat ini %d", minJWTSecretLen, len(c.JWT.Secret)))
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		problems = append(problems, fmt.Sprintf("APP_PORT %d di luar rentang 1-65535", c.Server.Port))
	}
	if c.Database.Port < 1 || c.Database.Port > 65535 {
		problems = append(problems, fmt.Sprintf("DB_PORT %d di luar rentang 1-65535", c.Database.Port))
	}
	if c.Database.Host == "" || c.Database.Name == "" || c.Database.User == "" {
		problems = append(problems, "DB_HOST, DB_NAME, dan DB_USER tidak boleh kosong")
	}
	if c.Storage.Type != "local" {
		problems = append(problems, fmt.Sprintf("STORAGE_TYPE %q belum didukung; MVP hanya \"local\" (ADR-0005)", c.Storage.Type))
	}
	if strings.TrimSpace(c.Storage.Path) == "" {
		problems = append(problems, "STORAGE_PATH wajib diisi")
	}
	if c.JWT.Expiry <= 0 {
		problems = append(problems, "JWT_EXPIRY harus durasi positif")
	}

	if len(problems) > 0 {
		return fmt.Errorf(
			"konfigurasi tidak lengkap (%d masalah):\n  - %s\nLihat daftar environment variable di 60-DEPLOYMENT.md §2.1 dan salin .env.example menjadi .env",
			len(problems), strings.Join(problems, "\n  - "),
		)
	}
	return nil
}

// applyDefaults hanya memuat nilai NON-rahasia. Rahasia (DB_PASSWORD, JWT_SECRET,
// ADMIN_PASSWORD) tidak punya default dengan sengaja.
func applyDefaults(v *viper.Viper) {
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("APP_PORT", 8080)
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", 5432)
	v.SetDefault("DB_NAME", "bwdcs")
	v.SetDefault("DB_USER", "bwdcs")
	v.SetDefault("JWT_EXPIRY", "24h")
	v.SetDefault("STORAGE_TYPE", "local")
	v.SetDefault("STORAGE_PATH", "./storage")
	v.SetDefault("REDIS_URL", "")
}

// loadEnvFile membaca .env bila ada. Berkas ini opsional: di produksi
// konfigurasi datang dari environment container. Urutan pencarian mengikuti
// cara dev menjalankan server (`go run ./cmd/server` dari dalam backend/).
func loadEnvFile(v *viper.Viper) error {
	candidates := []string{".env", filepath.Join("..", ".env")}

	for _, path := range candidates {
		info, err := os.Stat(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("periksa berkas %s: %w", path, err)
		}
		if info.IsDir() {
			continue
		}

		v.SetConfigFile(path)
		v.SetConfigType("dotenv")
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("baca berkas %s: %w", path, err)
		}
		return nil
	}
	return nil
}

// Port sebagai string untuk pesan log/URL tanpa konversi di pemanggil.
func (s ServerConfig) PortString() string { return strconv.Itoa(s.Port) }
