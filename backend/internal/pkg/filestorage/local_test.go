package filestorage

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestStorage(t *testing.T) *LocalStorage {
	t.Helper()

	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}
	return store
}

func TestNewLocalMembuatDirektori(t *testing.T) {
	root := filepath.Join(t.TempDir(), "belum", "ada")

	store, err := NewLocal(root)
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	info, err := os.Stat(root)
	if err != nil {
		t.Fatalf("direktori storage tidak dibuat: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("STORAGE_PATH bukan direktori: %s", root)
	}
	if store.Root() != root {
		t.Fatalf("Root() = %q, ingin %q", store.Root(), root)
	}
}

func TestNewLocalMenolakPathKosong(t *testing.T) {
	if _, err := NewLocal("   "); err == nil {
		t.Fatal("NewLocal(\"   \") seharusnya gagal untuk STORAGE_PATH kosong")
	}
}

func TestSaveMengembalikanKeySesuaiSkemaADR0005(t *testing.T) {
	store := newTestStorage(t)

	key, err := store.Save("org-1", "proj-1", "doc-1", "v1", "Laporan Akhir.pdf", strings.NewReader("isi berkas"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	want := "orgs/org-1/projects/proj-1/docs/doc-1/v1/Laporan Akhir.pdf"
	if key != want {
		t.Fatalf("key = %q, ingin %q", key, want)
	}
	if !store.Exists(key) {
		t.Fatalf("berkas hasil Save tidak ditemukan: %s", key)
	}

	reader, size, err := store.Download(key)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("baca isi berkas: %v", err)
	}
	if string(content) != "isi berkas" {
		t.Fatalf("isi berkas = %q, ingin %q", content, "isi berkas")
	}
	if size != int64(len("isi berkas")) {
		t.Fatalf("ukuran = %d, ingin %d", size, len("isi berkas"))
	}
}

func TestSaveTidakDapatMenimpaVersi(t *testing.T) {
	store := newTestStorage(t)

	key, err := store.Save("org-1", "proj-1", "doc-1", "v1", "kontrak.pdf", strings.NewReader("versi pertama"))
	if err != nil {
		t.Fatalf("Save pertama: %v", err)
	}

	// Versi dokumen imutabel (ADR-0005, FR-VER-03).
	if _, err := store.Save("org-1", "proj-1", "doc-1", "v1", "kontrak.pdf", strings.NewReader("versi penimpa")); err == nil {
		t.Fatal("Save kedua dengan key sama seharusnya ditolak")
	}

	reader, _, err := store.Download(key)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	defer reader.Close()

	content, _ := io.ReadAll(reader)
	if string(content) != "versi pertama" {
		t.Fatalf("isi berkas berubah menjadi %q; penimpaan seharusnya dilarang", content)
	}
}

func TestSaveMensanitasiNamaBerkasKlien(t *testing.T) {
	store := newTestStorage(t)

	key, err := store.Save("org-1", "proj-1", "doc-1", "v1", "../../../etc/pas swd;rm -rf", strings.NewReader("x"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if strings.Contains(key, "..") {
		t.Fatalf("key masih memuat \"..\": %q", key)
	}
	abs := filepath.Join(store.Root(), filepath.FromSlash(key))
	if !strings.HasPrefix(abs, store.Root()+string(filepath.Separator)) {
		t.Fatalf("berkas ditulis di luar direktori storage: %s", abs)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("berkas tidak ada di lokasi yang dikembalikan: %v", err)
	}
}

func TestSaveMenolakSegmenKeyYangTidakValid(t *testing.T) {
	store := newTestStorage(t)
	cases := map[string]struct{ orgID, projectID, docID, version string }{
		"org kosong":         {orgID: "", projectID: "proj-1", docID: "doc-1", version: "v1"},
		"version titik dua":  {orgID: "org-1", projectID: "proj-1", docID: "doc-1", version: ".."},
		"docID garis miring": {orgID: "org-1", projectID: "proj-1", docID: "doc/1", version: "v1"},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := store.Save(c.orgID, c.projectID, c.docID, c.version, "berkas.pdf", strings.NewReader("x")); err == nil {
				t.Fatalf("Save dengan %s seharusnya ditolak", name)
			}
		})
	}
}

func TestSaveMenolakNamaBerkasKosong(t *testing.T) {
	store := newTestStorage(t)

	if _, err := store.Save("org-1", "proj-1", "doc-1", "v1", "   ", strings.NewReader("x")); err == nil {
		t.Fatal("Save dengan nama berkas kosong seharusnya ditolak")
	}
}

func TestDownloadDanExistsMenolakKeyDiLuarStorage(t *testing.T) {
	store := newTestStorage(t)

	for _, key := range []string{"../rahasia.txt", "orgs/../../rahasia.txt", "/etc/passwd", ""} {
		if _, _, err := store.Download(key); err == nil {
			t.Fatalf("Download(%q) seharusnya ditolak", key)
		}
		if store.Exists(key) {
			t.Fatalf("Exists(%q) seharusnya false", key)
		}
	}
}

func TestDeleteIdempotent(t *testing.T) {
	store := newTestStorage(t)

	if err := store.Delete("orgs/tidak-ada/docs/x"); err != nil {
		t.Fatalf("Delete key yang tidak ada seharusnya berhasil: %v", err)
	}

	key, err := store.Save("org-1", "proj-1", "doc-1", "v1", "hapus.pdf", strings.NewReader("x"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.Delete(key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if store.Exists(key) {
		t.Fatalf("berkas %q masih ada setelah Delete", key)
	}
}

func TestPingMenandaiStorageYangTidakDapatDipakai(t *testing.T) {
	store := newTestStorage(t)

	if err := store.Ping(); err != nil {
		t.Fatalf("Ping pada storage sehat: %v", err)
	}

	if err := os.RemoveAll(store.Root()); err != nil {
		t.Fatalf("hapus direktori storage: %v", err)
	}
	if err := store.Ping(); err == nil {
		t.Fatal("Ping seharusnya gagal setelah direktori storage dihapus")
	}
}
