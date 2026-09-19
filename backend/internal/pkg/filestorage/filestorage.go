// Package filestorage menyediakan abstraksi penyimpanan berkas dokumen.
//
// ADR-0005: implementasi MVP adalah filesystem lokal; implementasi S3-compatible
// menyusul tanpa mengubah service pemanggil. ADR-0013: package ini hanya
// infrastruktur tanpa aturan domain.
package filestorage

import "io"

// FileStorage adalah kontrak penyimpanan berkas (`40-TSD.md` §2.4).
//
// Key yang dikembalikan Save bersifat OPAQUE bagi pemanggil: service menyimpan
// key apa adanya di kolom `document_versions.file_key` dan tidak pernah menyusun
// atau menafsirkan key sendiri. Metadata resmi berkas (nama asli, MIME, ukuran,
// checksum) tetap disimpan di PostgreSQL (ADR-0005).
type FileStorage interface {
	// Save menulis satu versi berkas dan mengembalikan key-nya.
	//
	// originalName hanya dipakai untuk penamaan berkas di penyimpanan dan wajib
	// di-sanitasi implementasi: nama dari klien tidak boleh menentukan path.
	Save(orgID, projectID, docID, version, originalName string, data io.Reader) (string, error)

	// Download membuka berkas beserta ukurannya dalam byte.
	Download(key string) (io.ReadCloser, int64, error)

	// Delete menghapus key. Implementasi wajib idempotent (key yang tidak ada
	// bukan error), karena pemanggilnya bisa berada di jalur pembersihan.
	Delete(key string) error

	// Exists melaporkan apakah key ada.
	Exists(key string) bool
}

// Prober adalah pemeriksaan kesehatan penyimpanan untuk `GET /health`
// (`60-DEPLOYMENT.md` §5). Dipisahkan dari FileStorage supaya double uji tidak
// wajib mengimplementasikan seluruh operasi berkas.
type Prober interface {
	Ping() error
}
