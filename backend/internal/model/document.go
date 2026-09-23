package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Status dokumen kanonik (`documents.status`, `41-DATABASE.md` §2.3, FR-DOC-03,
// ADR-0012).
//
// Hanya enam nilai ini yang pernah tersimpan. Label tampilan ("Draft", "In
// Review", ...) dan seluruh keadaan turunan dihitung saat dibaca, tidak pernah
// disimpan (ADR-0012, `50-FSD.md` §11).
//
// `archived` bukan status alur review, melainkan hasil `POST /documents/:id/archive`
// (ADR-0019): dokumen terarsip keluar dari daftar default, tetap dapat dibaca
// dan diunduh, tetapi menolak unggahan versi baru maupun submit ke workflow.
// Kosakata di sini **wajib** sama dengan `CHECK` di migrasi `010`; test
// `TestDocumentStatusVocabularyIncludesArchived` (migrasi) membandingkan keduanya
// supaya keduanya tidak dapat berbeda diam-diam.
const (
	DocumentStatusDraft            = "draft"
	DocumentStatusInReview         = "in_review"
	DocumentStatusRevisionRequired = "revision_required"
	DocumentStatusApproved         = "approved"
	DocumentStatusRejected         = "rejected"
	DocumentStatusArchived         = "archived"
)

// DocumentStatuses mengembalikan daftar status dokumen yang sah, dipakai
// validasi filter daftar dan pesan error yang menyebut nilai yang diterima.
//
// Urutannya mengikuti `50-FSD.md` §11 (status alur lebih dulu, `archived`
// terakhir) dan dipakai apa adanya pada pesan `422`.
func DocumentStatuses() []string {
	return []string{
		DocumentStatusDraft,
		DocumentStatusInReview,
		DocumentStatusRevisionRequired,
		DocumentStatusApproved,
		DocumentStatusRejected,
		DocumentStatusArchived,
	}
}

// IsDocumentArchived menjawab apakah dokumen sudah diarsipkan (ADR-0019).
//
// Aturannya bergantung pada kolom kanonik `status`, bukan pada `archived_at`:
// satu-satunya penulis keduanya adalah operasi arsip itu sendiri, dan `status`
// adalah nilai yang juga dilihat penyaring daftar serta klien.
func (d *Document) IsDocumentArchived() bool {
	return d.Status == DocumentStatusArchived
}

// IsDocumentStatus menjawab apakah `status` termasuk himpunan tertutup FR-DOC-03.
func IsDocumentStatus(status string) bool {
	for _, valid := range DocumentStatuses() {
		if status == valid {
			return true
		}
	}
	return false
}

// Document merepresentasikan tabel `documents` (`41-DATABASE.md` §2.3).
//
// `DocumentNumber` dibangkitkan server `{PROJECT_CODE}-{NNN}` dan **immutable**
// (ADR-0017): tidak ada endpoint yang mengubahnya, dan kolomnya tidak pernah
// muncul sebagai input klien.
type Document struct {
	ID                 uuid.UUID  `json:"id"`
	ProjectID          uuid.UUID  `json:"project_id"`
	DocumentNumber     string     `json:"document_number"`
	Title              string     `json:"title"`
	CategoryID         *uuid.UUID `json:"category_id,omitempty"`
	Description        string     `json:"description"`
	OwnerID            uuid.UUID  `json:"owner_id"`
	Status             string     `json:"status"`
	ArchivedAt         *time.Time `json:"archived_at,omitempty"`
	CurrentVersion     int        `json:"current_version"`
	WorkflowInstanceID *uuid.UUID `json:"workflow_instance_id,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`

	// Kolom turunan untuk daftar/detail (`50-FSD.md` §4.1 kolom tabel dan §4.3
	// metadata). Dibaca lewat JOIN/subquery, tidak disimpan di tabel.
	//
	// `ProjectArchived` sengaja tetap bernama begitu (bukan `Archived`): yang
	// diarsipkan adalah **project**-nya, dan modul dokumen tetap menerima dokumen
	// baru di project arsip sampai Q-016 butir 6 diputuskan user.
	ProjectCode      string `json:"project_code,omitempty"`
	ProjectName      string `json:"project_name,omitempty"`
	CategoryName     string `json:"category_name,omitempty"`
	OwnerUsername    string `json:"owner_username,omitempty"`
	LatestVersion    string `json:"latest_version,omitempty"`
	ProjectArchived  bool   `json:"project_archived"`
	WorkflowInstance string `json:"workflow_instance_status,omitempty"`
}

// DocumentVersion merepresentasikan tabel `document_versions`
// (`41-DATABASE.md` §2.3).
//
// Baris ini **immutable** (FR-VER-03): tidak ada endpoint yang mengubah atau
// menghapus satu versi. Imutabilitas ditegakkan **tiga lapis**: service (tidak
// ada jalur ubah/hapus), storage (`Save` menolak menimpa berkas), dan database —
// sejak migrasi `010` tabelnya append-only dengan trigger yang sama seperti
// `audit_logs` (`44-SECURITY.md` §6.1, ADR-0019 butir 2).
type DocumentVersion struct {
	ID           uuid.UUID `json:"id"`
	DocumentID   uuid.UUID `json:"document_id"`
	Version      string    `json:"version"`
	FileKey      string    `json:"file_key"`
	OriginalName string    `json:"original_name"`
	MimeType     string    `json:"mime_type"`
	Size         int64     `json:"size"`
	Checksum     string    `json:"checksum"`
	RevisionNote string    `json:"revision_note"`
	UploadedByID uuid.UUID `json:"uploaded_by_id"`
	CreatedAt    time.Time `json:"created_at"`

	// UploadedByUsername adalah kolom turunan hasil JOIN ke `users`
	// (`FR-VER-04`: riwayat versi menampilkan penulisnya).
	UploadedByUsername string `json:"uploaded_by_username,omitempty"`
}

// DocumentCategory merepresentasikan tabel `document_categories`
// (`41-DATABASE.md` §2.3). Belum ada endpoint kategori di `42-API.md` §4 —
// kategori hanya dapat dirujuk dan dibaca namanya lewat dokumen.
type DocumentCategory struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	Code           string    `json:"code"`
	CreatedAt      time.Time `json:"created_at"`
}

// VersionFormat adalah bentuk `document_versions.version` (FR-VER-02):
// `major.minor`, mis. `1.0`, `1.1`, `2.0`.
const VersionFormat = "%d.%d"

// FormatVersion menyusun string versi dari pasangan major/minor.
func FormatVersion(major, minor int) string {
	return fmt.Sprintf(VersionFormat, major, minor)
}

// ParseVersion membalik FormatVersion. Nilai yang tidak berbentuk major.minor
// dikembalikan sebagai error: string versi selalu dibuat server, jadi bentuk
// yang tidak dikenal berarti data di luar dugaan, bukan input klien.
func ParseVersion(version string) (major, minor int, err error) {
	parts := strings.Split(strings.TrimSpace(version), ".")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("versi %q bukan bentuk major.minor", version)
	}

	major, err = strconv.Atoi(parts[0])
	if err != nil || major < 1 {
		return 0, 0, fmt.Errorf("versi %q tidak memuat nomor major yang sah", version)
	}

	minor, err = strconv.Atoi(parts[1])
	if err != nil || minor < 0 {
		return 0, 0, fmt.Errorf("versi %q tidak memuat nomor minor yang sah", version)
	}

	return major, minor, nil
}

// NextVersion menghitung versi berikutnya dari versi terakhir (FR-VER-02).
//
// Aturannya deterministik dan tidak bergantung pada teks catatan revisi:
//
//   - unggahan pertama (`latest` kosong) → `1.0`;
//   - unggahan biasa → minor naik (`1.0` → `1.1`, `1.1` → `1.2`);
//   - unggahan setelah revisi diminta (`majorBump`, yaitu `documents.status =
//     revision_required`) → major naik dan minor kembali nol (`1.1` → `2.0`).
//
// Butir ketiga itulah yang mewujudkan "subsequent uploads → next major based on
// revision" pada `50-FSD.md` §4.2 tanpa menuntut klien mengirim jenis versi:
// status dokumen sudah menyatakan bahwa unggahan ini adalah jawaban atas
// permintaan revisi (ADR-0016).
func NextVersion(latest string, majorBump bool) (string, error) {
	if strings.TrimSpace(latest) == "" {
		return FormatVersion(1, 0), nil
	}

	major, minor, err := ParseVersion(latest)
	if err != nil {
		return "", err
	}

	if majorBump {
		return FormatVersion(major+1, 0), nil
	}
	return FormatVersion(major, minor+1), nil
}
