package filestorage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalStorage menyimpan berkas di filesystem lokal dengan skema path
// (ADR-0005, contoh key di `42-API.md` §4):
//
//	{root}/orgs/{orgID}/projects/{projectID}/docs/{docID}/{version}/{nama-asli}
//
// Versi bersifat imutabel: Save menolak menimpa berkas yang sudah ada, sehingga
// riwayat approval tidak dapat diubah diam-diam (FR-VER-03).
type LocalStorage struct {
	root string
}

var _ FileStorage = (*LocalStorage)(nil)
var _ Prober = (*LocalStorage)(nil)

// NewLocal menyiapkan direktori penyimpanan. Direktori dibuat bila belum ada,
// dan aplikasi gagal start (bukan diam-diam) bila path tidak dapat dipakai.
func NewLocal(root string) (*LocalStorage, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("STORAGE_PATH kosong")
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve STORAGE_PATH %q: %w", root, err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("buat direktori storage %q: %w", abs, err)
	}

	return &LocalStorage{root: abs}, nil
}

// Root mengembalikan path absolut direktori penyimpanan.
func (s *LocalStorage) Root() string { return s.root }

// Ping membuktikan direktori storage ada dan dapat ditulisi. Berkas probe
// dibuat lalu dihapus, sehingga kesehatan yang dilaporkan bukan sekadar "path
// terlihat ada".
func (s *LocalStorage) Ping() error {
	f, err := os.CreateTemp(s.root, ".healthcheck-*")
	if err != nil {
		return fmt.Errorf("storage %q tidak dapat ditulisi: %w", s.root, err)
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		return fmt.Errorf("tutup berkas probe %q: %w", name, err)
	}
	return os.Remove(name)
}

// Save menulis isi reader sebagai versi baru dan mengembalikan key relatif.
func (s *LocalStorage) Save(orgID, projectID, docID, version, originalName string, data io.Reader) (string, error) {
	name, err := sanitizeFilename(originalName)
	if err != nil {
		return "", err
	}

	segments := []string{"orgs", orgID, "projects", projectID, "docs", docID, version}
	for _, segment := range segments {
		if err := validateSegment(segment); err != nil {
			return "", err
		}
	}

	key := filepath.Join(append(segments, name)...)
	abs := filepath.Join(s.root, key)

	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		return "", fmt.Errorf("buat direktori %q: %w", filepath.Dir(key), err)
	}

	file, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("berkas %q sudah ada: versi dokumen tidak boleh ditimpa (ADR-0005)", key)
		}
		return "", fmt.Errorf("buat berkas %q: %w", key, err)
	}

	if _, err := io.Copy(file, data); err != nil {
		_ = file.Close()
		_ = os.Remove(abs) // jangan tinggalkan berkas separuh jadi
		return "", fmt.Errorf("tulis berkas %q: %w", key, err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(abs)
		return "", fmt.Errorf("tutup berkas %q: %w", key, err)
	}

	return filepath.ToSlash(key), nil
}

// Download membuka berkas dan mengembalikan ukurannya.
func (s *LocalStorage) Download(key string) (io.ReadCloser, int64, error) {
	abs, err := s.resolve(key)
	if err != nil {
		return nil, 0, err
	}

	file, err := os.Open(abs)
	if err != nil {
		return nil, 0, fmt.Errorf("buka berkas %q: %w", key, err)
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, 0, fmt.Errorf("stat berkas %q: %w", key, err)
	}
	if info.IsDir() {
		_ = file.Close()
		return nil, 0, fmt.Errorf("key %q adalah direktori, bukan berkas", key)
	}

	return file, info.Size(), nil
}

// Delete menghapus berkas. Idempotent: key yang tidak ada dianggap berhasil.
func (s *LocalStorage) Delete(key string) error {
	abs, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(abs); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("hapus berkas %q: %w", key, err)
	}
	return nil
}

// Exists melaporkan keberadaan berkas.
func (s *LocalStorage) Exists(key string) bool {
	abs, err := s.resolve(key)
	if err != nil {
		return false
	}
	info, err := os.Stat(abs)
	return err == nil && !info.IsDir()
}

// resolve mengubah key relatif menjadi path absolut sambil memastikan hasilnya
// tidak keluar dari direktori penyimpanan (path traversal).
func (s *LocalStorage) resolve(key string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(key))
	if clean == "." || clean == ".." || filepath.IsAbs(clean) ||
		strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("key tidak valid: %q", key)
	}

	abs := filepath.Join(s.root, clean)
	if !strings.HasPrefix(abs, s.root+string(filepath.Separator)) {
		return "", fmt.Errorf("key keluar dari direktori penyimpanan: %q", key)
	}
	return abs, nil
}

// validateSegment menolak segmen key yang dapat mengubah struktur path.
func validateSegment(segment string) error {
	if segment == "" || segment == "." || segment == ".." ||
		strings.ContainsAny(segment, `/\`) {
		return fmt.Errorf("segmen key tidak valid: %q", segment)
	}
	return nil
}

// sanitizeFilename menyisakan nama berkas yang aman dipakai di filesystem.
// Nama dari klien hanya menentukan label berkas, bukan lokasinya.
func sanitizeFilename(originalName string) (string, error) {
	base := filepath.Base(filepath.FromSlash(strings.TrimSpace(originalName)))
	if base == "" || base == "." || base == ".." || base == string(filepath.Separator) {
		return "", fmt.Errorf("nama berkas tidak valid: %q", originalName)
	}

	var safe strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			safe.WriteRune(r)
		case r == '.', r == '-', r == '_', r == ' ':
			safe.WriteRune(r)
		default:
			safe.WriteRune('_')
		}
	}

	name := strings.Trim(safe.String(), " .")
	if name == "" {
		return "", fmt.Errorf("nama berkas tidak valid: %q", originalName)
	}
	if len(name) > 255 {
		return "", fmt.Errorf("nama berkas melebihi 255 karakter (%d): dikirim %q", len(name), originalName)
	}
	return name, nil
}
