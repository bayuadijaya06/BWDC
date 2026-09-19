package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/repository"
)

// MaxDocumentFileSize adalah batas ukuran satu berkas dokumen
// (`50-FSD.md` §4.2: 100 MB).
const MaxDocumentFileSize int64 = 100 << 20

// allowedDocumentExtensions dan allowedDocumentMIMETypes adalah daftar tertutup
// unggahan (`50-FSD.md` §4.2 + `44-SECURITY.md` §4.2). Ekstensi diperiksa
// sebagai penjaga kedua: MIME hasil deteksi isi berkas tidak selalu sama dengan
// label yang diharapkan pengguna (mis. `.csv` yang isinya teks biasa).
var (
	allowedDocumentExtensions = map[string]bool{
		".pdf": true, ".txt": true, ".csv": true,
		".xls": true, ".xlsx": true, ".jpg": true, ".jpeg": true, ".png": true,
	}

	allowedDocumentMIMETypes = map[string]bool{
		"application/pdf":          true,
		"text/plain":               true,
		"text/csv":                 true,
		"application/vnd.ms-excel": true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
		"image/jpeg": true,
		"image/png":  true,
	}
)

// AllowedDocumentExtensions mengembalikan ekstensi yang diterima, dipakai pesan
// validasi `422` supaya klien tahu batasnya tanpa membaca dokumen desain.
func AllowedDocumentExtensions() []string {
	return []string{".pdf", ".txt", ".csv", ".xls", ".xlsx", ".jpg", ".jpeg", ".png"}
}

// UploadVersionInput adalah satu unggahan versi dokumen.
//
// `MimeType` adalah hasil deteksi isi berkas (magic bytes) di handler, bukan
// `Content-Type` yang dikirim klien: `44-SECURITY.md` §4.2 memvalidasi tipe
// berdasarkan isinya. `Size` adalah ukuran yang **diklaim** header multipart;
// ukuran sebenarnya dihitung ulang saat berkas mengalir.
type UploadVersionInput struct {
	OriginalName string
	MimeType     string
	Size         int64
	RevisionNote string
	Content      io.Reader
}

// UploadVersion menambahkan satu versi berkas ke dokumen (FR-VER-01/FR-VER-02).
//
// Urutan di dalam satu transaksi: tentukan nomor versi dari versi terakhir +
// status dokumen, tulis berkas ke storage sambil menghitung SHA-256, simpan
// baris versi, selaraskan `documents.current_version`, lalu tulis audit. Bila
// langkah setelah penulisan berkas gagal, berkas itu dihapus kembali sehingga
// tidak ada versi "hantu" di storage tanpa baris database.
func (s *DocumentService) UploadVersion(ctx context.Context, actor Actor, documentID uuid.UUID, input UploadVersionInput) (*model.DocumentVersion, error) {
	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, err
	}

	document, err := s.documents.FindByID(ctx, scope, documentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDocumentNotFound
		}
		return nil, err
	}

	if err := validateUpload(input); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi unggah versi: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	documents := s.documents.WithTx(tx)

	// Nomor versi berikutnya ditentukan server (FR-VER-02). Unggahan setelah
	// revisi diminta menjadi versi major (ADR-0016: pelanjut review wajib
	// membawa versi baru), unggahan biasa menjadi minor.
	latest, err := documents.LatestVersion(ctx, documentID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	latestVersion := ""
	if err == nil && latest != nil {
		latestVersion = latest.Version
	}

	majorBump := document.Status == model.DocumentStatusRevisionRequired
	version, err := model.NextVersion(latestVersion, majorBump)
	if err != nil {
		return nil, fmt.Errorf("tentukan versi berikutnya dokumen %s: %w", document.DocumentNumber, err)
	}

	hasher := sha256.New()
	limited := &limitedReader{reader: input.Content, remaining: MaxDocumentFileSize}

	key, err := s.storage.Save(
		actor.OrganizationID.String(), document.ProjectID.String(), documentID.String(),
		version, input.OriginalName, io.TeeReader(limited, hasher),
	)
	if err != nil {
		if errors.Is(err, ErrDocumentFileTooLarge) {
			return nil, ErrDocumentFileTooLarge
		}
		return nil, fmt.Errorf("simpan berkas versi %s: %w", version, err)
	}

	// Setelah titik ini berkas sudah ada di storage: setiap kegagalan harus
	// membersihkannya agar storage dan database tidak berbeda diam-diam.
	cleanup := func() { _ = s.storage.Delete(key) }

	record := &model.DocumentVersion{
		DocumentID:   documentID,
		Version:      version,
		FileKey:      key,
		OriginalName: input.OriginalName,
		MimeType:     input.MimeType,
		Size:         limited.Size(),
		Checksum:     hex.EncodeToString(hasher.Sum(nil)),
		RevisionNote: input.RevisionNote,
		UploadedByID: actor.ID,
	}
	if err := documents.AddVersion(ctx, record); err != nil {
		cleanup()
		return nil, err
	}

	if err := documents.SetCurrentVersion(ctx, documentID, document.CurrentVersion+1); err != nil {
		cleanup()
		return nil, err
	}

	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionDocumentVersionCreated, EntityDocument, document.DocumentNumber,
		"Versi "+version+" dokumen "+document.DocumentNumber+" diunggah", map[string]any{
			"document_id":     documentID.String(),
			"document_number": document.DocumentNumber,
			"project_id":      document.ProjectID.String(),
			"version_id":      record.ID.String(),
			"version":         version,
			"file_size":       record.Size,
		}); err != nil {
		cleanup()
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		cleanup()
		return nil, fmt.Errorf("commit unggah versi: %w", err)
	}

	s.logger.Info("versi dokumen diunggah",
		"document_id", documentID.String(),
		"document_number", document.DocumentNumber,
		"version", version,
		"size", record.Size,
		"actor_id", actor.ID.String(),
	)

	// Dibaca ulang supaya bentuk yang dikembalikan sama dengan endpoint daftar
	// versi (termasuk `uploaded_by_username` dari JOIN).
	stored, err := s.documents.FindVersion(ctx, documentID, record.ID)
	if err != nil {
		return nil, err
	}
	return stored, nil
}

// Versions mengembalikan riwayat versi satu dokumen, terbaru lebih dulu
// (FR-VER-04/FR-VER-05). Dokumen di luar cakupan dibalas 404, bukan daftar kosong.
func (s *DocumentService) Versions(ctx context.Context, actor Actor, documentID uuid.UUID) ([]model.DocumentVersion, error) {
	if _, err := s.Get(ctx, actor, documentID); err != nil {
		return nil, err
	}
	return s.documents.Versions(ctx, documentID)
}

// Download membuka satu versi dokumen untuk diunduh (`42-API.md` §4).
//
// Audit unduhan ditulis di transaksi singkat tersendiri **sebelum** berkas
// dibuka: `FR-AUDIT-01` mewajibkan "download doc" tercatat, dan ADR-0011 butir 4
// menetapkan aksi read-only yang wajib diaudit memakai transaksinya sendiri.
// Bila penulisan audit gagal, unduhan tidak dilayani — bukan dilayani tanpa jejak.
func (s *DocumentService) Download(ctx context.Context, actor Actor, documentID, versionID uuid.UUID) (*DocumentDownload, error) {
	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, err
	}

	document, err := s.documents.FindByID(ctx, scope, documentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDocumentNotFound
		}
		return nil, err
	}

	version, err := s.documents.FindVersion(ctx, documentID, versionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDocumentVersionNotFound
		}
		return nil, err
	}

	if err := s.auditDownload(ctx, actor, document, version); err != nil {
		return nil, err
	}

	content, size, err := s.storage.Download(version.FileKey)
	if err != nil {
		s.logger.Error("berkas versi tidak dapat dibuka",
			"document_id", documentID.String(),
			"version_id", versionID.String(),
			"file_key", version.FileKey,
			"error", err.Error(),
		)
		return nil, ErrDocumentFileMissing
	}

	return &DocumentDownload{Version: version, Content: content, Size: size}, nil
}

// auditDownload menulis entri audit unduhan di transaksi singkat tersendiri
// (ADR-0011 butir 4).
func (s *DocumentService) auditDownload(ctx context.Context, actor Actor, document *model.Document, version *model.DocumentVersion) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mulai transaksi audit unduhan: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionDocumentDownloaded, EntityDocument, document.DocumentNumber,
		"Versi "+version.Version+" dokumen "+document.DocumentNumber+" diunduh", map[string]any{
			"document_id":     document.ID.String(),
			"document_number": document.DocumentNumber,
			"project_id":      document.ProjectID.String(),
			"version_id":      version.ID.String(),
			"version":         version.Version,
		}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit audit unduhan: %w", err)
	}
	return nil
}

// validateUpload memeriksa ekstensi, MIME, dan ukuran yang diklaim header.
//
// Ukuran sebenarnya diperiksa ulang saat berkas mengalir (lihat limitedReader):
// header multipart berasal dari klien dan tidak boleh menjadi satu-satunya penjaga.
func validateUpload(input UploadVersionInput) error {
	name := strings.TrimSpace(input.OriginalName)
	if name == "" {
		return ErrDocumentFileType
	}

	extension := strings.ToLower(filepath.Ext(name))
	if !allowedDocumentExtensions[extension] {
		return ErrDocumentFileType
	}

	if !allowedDocumentMIMETypes[strings.ToLower(strings.TrimSpace(input.MimeType))] {
		return ErrDocumentFileType
	}

	if input.Size > MaxDocumentFileSize {
		return ErrDocumentFileTooLarge
	}

	return nil
}

// limitedReader mengalirkan isi berkas sambil menegakkan batas ukuran dan
// menghitung byte yang benar-benar dibaca.
//
// Dipakai, bukan hanya `io.LimitReader`, karena `LimitReader` memotong berkas
// besar diam-diam sehingga checksum dan ukuran yang tersimpan akan menipu.
// Pembaca ini mengembalikan ErrDocumentFileTooLarge saat isinya melampaui batas.
type limitedReader struct {
	reader    io.Reader
	remaining int64
	read      int64
}

// Read meneruskan bacaan selama kuota tersisa dan menolak byte kelebihannya.
func (l *limitedReader) Read(p []byte) (int, error) {
	if l.remaining <= 0 {
		var probe [1]byte
		n, err := l.reader.Read(probe[:])
		if n > 0 {
			return 0, ErrDocumentFileTooLarge
		}
		return 0, err
	}

	if int64(len(p)) > l.remaining {
		p = p[:l.remaining]
	}

	n, err := l.reader.Read(p)
	l.read += int64(n)
	l.remaining -= int64(n)
	return n, err
}

// Size mengembalikan jumlah byte yang sudah dibaca.
func (l *limitedReader) Size() int64 { return l.read }
