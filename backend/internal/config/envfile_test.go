package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// chdirPindah memindahkan CWD selama satu test, karena loadEnvFile mencari
// berkas `.env` relatif terhadap direktori kerja.
func chdirPindah(t *testing.T, dir string) {
	t.Helper()

	lama, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%s): %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(lama) })
}

func TestLoadMembacaBerkasEnvDanKutipan(t *testing.T) {
	clearEnv(t)

	dir := t.TempDir()
	isi := strings.Join([]string{
		"DB_PASSWORD=dari-berkas",
		"JWT_SECRET=" + strings.Repeat("b", 40),
		`ADMIN_ORG_NAME="Organisasi Contoh"`,
		"DB_NAME=dari_berkas",
		"", // baris kosong di akhir berkas tidak boleh mengganggu
	}, "\n")
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(isi), 0o600); err != nil {
		t.Fatalf("tulis .env: %v", err)
	}
	chdirPindah(t, dir)

	// Environment yang sudah diset harus menang atas isi berkas.
	t.Setenv("DB_NAME", "dari_environment")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Database.Password != "dari-berkas" {
		t.Fatalf("DB_PASSWORD = %q, ingin %q", cfg.Database.Password, "dari-berkas")
	}
	if cfg.Database.Name != "dari_environment" {
		t.Fatalf("DB_NAME = %q, ingin nilai dari environment", cfg.Database.Name)
	}
	// Nilai berkutip harus utuh; inilah bug yang muncul saat .env di-source shell.
	if cfg.Bootstrap.OrgName != "Organisasi Contoh" {
		t.Fatalf("ADMIN_ORG_NAME = %q, ingin %q", cfg.Bootstrap.OrgName, "Organisasi Contoh")
	}
}

func TestLoadTetapJalanTanpaBerkasEnv(t *testing.T) {
	clearEnv(t)
	chdirPindah(t, t.TempDir())

	t.Setenv("DB_PASSWORD", "rahasia-test")
	t.Setenv("JWT_SECRET", strings.Repeat("k", minJWTSecretLen))

	if _, err := Load(); err != nil {
		t.Fatalf("Load tanpa .env seharusnya berhasil: %v", err)
	}
}
