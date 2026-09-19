package dto

import (
	"time"

	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
)

// CreateDocumentRequest adalah body `POST /documents` (`42-API.md` §4).
//
// `DocumentNumber` ada di sini **hanya** supaya kiriman klien dapat ditolak
// dengan pesan yang jelas (`422 VALIDATION_ERROR`): nomor dibangkitkan server
// `{PROJECT_CODE}-{NNN}` dan tidak pernah dibaca sebagai input (ADR-0017).
type CreateDocumentRequest struct {
	ProjectID      uuid.UUID  `json:"project_id"`
	Title          string     `json:"title"`
	CategoryID     *uuid.UUID `json:"category_id"`
	Description    string     `json:"description"`
	DocumentNumber *string    `json:"document_number"`
}

// DocumentResponse adalah bentuk dokumen pada response `42-API.md` §4.
//
// `CurrentVersion` adalah jumlah versi yang tersimpan; label versinya dikirim
// terpisah sebagai `latest_version` (`major.minor`, FR-VER-02).
type DocumentResponse struct {
	ID                 uuid.UUID  `json:"id"`
	ProjectID          uuid.UUID  `json:"project_id"`
	ProjectCode        string     `json:"project_code,omitempty"`
	ProjectName        string     `json:"project_name,omitempty"`
	ProjectArchived    bool       `json:"project_archived"`
	DocumentNumber     string     `json:"document_number"`
	Title              string     `json:"title"`
	CategoryID         *uuid.UUID `json:"category_id,omitempty"`
	CategoryName       string     `json:"category_name,omitempty"`
	Description        string     `json:"description"`
	OwnerID            uuid.UUID  `json:"owner_id"`
	OwnerUsername      string     `json:"owner_username,omitempty"`
	Status             string     `json:"status"`
	CurrentVersion     int        `json:"current_version"`
	LatestVersion      string     `json:"latest_version,omitempty"`
	WorkflowInstanceID *uuid.UUID `json:"workflow_instance_id,omitempty"`
	WorkflowStatus     string     `json:"workflow_instance_status,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// DocumentVersionResponse adalah bentuk satu versi dokumen
// (`42-API.md` §4 POST /documents/:id/upload, GET /documents/:id/versions).
//
// `Checksum` adalah digest SHA-256 dalam heksadesimal tanpa prefiks algoritma:
// kolomnya `VARCHAR(64)` (`41-DATABASE.md` §2.3), sehingga label `sha256:` tidak
// muat di sana dan tidak ditambahkan di response — bentuk yang sama dengan yang
// tersimpan adalah bentuk yang dapat diverifikasi ulang penerima.
type DocumentVersionResponse struct {
	ID                 uuid.UUID `json:"id"`
	DocumentID         uuid.UUID `json:"document_id"`
	Version            string    `json:"version"`
	FileKey            string    `json:"file_key"`
	OriginalName       string    `json:"original_name"`
	MimeType           string    `json:"mime_type"`
	Size               int64     `json:"size"`
	Checksum           string    `json:"checksum"`
	RevisionNote       string    `json:"revision_note,omitempty"`
	UploadedByID       uuid.UUID `json:"uploaded_by_id"`
	UploadedByUsername string    `json:"uploaded_by_username,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

// DocumentDetailResponse adalah response `GET /documents/:id`: detail dokumen +
// informasi versi berjalan. Dokumen yang belum punya unggahan mengirim
// `current_version: null`.
type DocumentDetailResponse struct {
	Document       DocumentResponse         `json:"document"`
	CurrentVersion *DocumentVersionResponse `json:"current_version"`
}

// NewDocumentResponse memetakan model dokumen ke bentuk response.
func NewDocumentResponse(document *model.Document) DocumentResponse {
	return DocumentResponse{
		ID:                 document.ID,
		ProjectID:          document.ProjectID,
		ProjectCode:        document.ProjectCode,
		ProjectName:        document.ProjectName,
		ProjectArchived:    document.ProjectArchived,
		DocumentNumber:     document.DocumentNumber,
		Title:              document.Title,
		CategoryID:         document.CategoryID,
		CategoryName:       document.CategoryName,
		Description:        document.Description,
		OwnerID:            document.OwnerID,
		OwnerUsername:      document.OwnerUsername,
		Status:             document.Status,
		CurrentVersion:     document.CurrentVersion,
		LatestVersion:      document.LatestVersion,
		WorkflowInstanceID: document.WorkflowInstanceID,
		WorkflowStatus:     document.WorkflowInstance,
		CreatedAt:          document.CreatedAt,
		UpdatedAt:          document.UpdatedAt,
	}
}

// NewDocumentListResponse memetakan daftar dokumen; hasilnya selalu array
// (bukan null) supaya klien tidak perlu menangani dua bentuk.
func NewDocumentListResponse(documents []model.Document) []DocumentResponse {
	out := make([]DocumentResponse, 0, len(documents))
	for i := range documents {
		out = append(out, NewDocumentResponse(&documents[i]))
	}
	return out
}

// NewDocumentVersionResponse memetakan satu baris `document_versions`.
func NewDocumentVersionResponse(version *model.DocumentVersion) DocumentVersionResponse {
	return DocumentVersionResponse{
		ID:                 version.ID,
		DocumentID:         version.DocumentID,
		Version:            version.Version,
		FileKey:            version.FileKey,
		OriginalName:       version.OriginalName,
		MimeType:           version.MimeType,
		Size:               version.Size,
		Checksum:           version.Checksum,
		RevisionNote:       version.RevisionNote,
		UploadedByID:       version.UploadedByID,
		UploadedByUsername: version.UploadedByUsername,
		CreatedAt:          version.CreatedAt,
	}
}

// NewDocumentVersionListResponse memetakan daftar versi; hasilnya selalu array.
func NewDocumentVersionListResponse(versions []model.DocumentVersion) []DocumentVersionResponse {
	out := make([]DocumentVersionResponse, 0, len(versions))
	for i := range versions {
		out = append(out, NewDocumentVersionResponse(&versions[i]))
	}
	return out
}
