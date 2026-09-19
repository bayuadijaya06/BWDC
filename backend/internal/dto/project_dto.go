package dto

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
)

// DateFormat adalah satu-satunya bentuk tanggal yang diterima dan dikirim
// `projects.start_date` / `.target_end_date` (`42-API.md` §3, `50-FSD.md` §3.2).
const DateFormat = "2006-01-02"

// ProjectCodePattern adalah pola `projects.code` (ADR-0017): huruf besar/angka
// yang dipisah tanda hubung, mis. `WEB` atau `WEB-REDESIGN`.
//
// Diperiksa dengan regex eksplisit karena tag `alphanum` milik validator
// menolak tanda hubung (`42-API.md` §3).
var ProjectCodePattern = regexp.MustCompile(`^[A-Z0-9]+(-[A-Z0-9]+)*$`)

// NormalizeProjectCode menyeragamkan kode project: spasi dibuang dan seluruh
// huruf dijadikan besar, sehingga "web " dan "WEB" tidak menjadi dua project.
func NormalizeProjectCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// Date adalah tanggal kalender tanpa waktu. JSON `"2026-10-01"` bukan format
// RFC 3339, sehingga `time.Time` mentah tidak dapat membacanya.
type Date struct {
	time.Time
}

// MarshalJSON mengirim tanggal sebagai `YYYY-MM-DD`.
func (d Date) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + d.Format(DateFormat) + `"`), nil
}

// UnmarshalJSON membaca `YYYY-MM-DD`; nilai null menjadi zero time.
func (d *Date) UnmarshalJSON(data []byte) error {
	raw := strings.Trim(string(data), `"`)
	if raw == "" || raw == "null" {
		d.Time = time.Time{}
		return nil
	}

	parsed, err := time.Parse(DateFormat, raw)
	if err != nil {
		return fmt.Errorf("tanggal harus berformat %s: %w", DateFormat, err)
	}
	d.Time = parsed
	return nil
}

// CreateProjectRequest adalah body `POST /projects` (`42-API.md` §3).
type CreateProjectRequest struct { // Tanpa tag `binding`: pemeriksaan wajib-isi, pola `code`, dan urutan
	// tanggal dilakukan handler dengan pesan per field (`42-API.md` §12).
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	OwnerID       uuid.UUID `json:"owner_id"`
	StartDate     *Date     `json:"start_date"`
	TargetEndDate *Date     `json:"target_end_date"`
}

// UpdateProjectRequest adalah body `PATCH /projects/:id`. Field yang tidak
// dikirim bernilai nil dan tidak diubah.
//
// `Code` ada di sini **hanya** supaya mengirimnya dapat ditolak `409 CONFLICT`
// dengan pesan yang jelas; nilai kolomnya tidak pernah diubah (ADR-0017).
type UpdateProjectRequest struct {
	Code          *string    `json:"code"`
	Name          *string    `json:"name"`
	Description   *string    `json:"description"`
	OwnerID       *uuid.UUID `json:"owner_id"`
	StartDate     *Date      `json:"start_date"`
	TargetEndDate *Date      `json:"target_end_date"`
}

// AddProjectMemberRequest adalah body `POST /projects/:id/members`.
type AddProjectMemberRequest struct {
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
}

// ProjectResponse adalah bentuk project pada response `42-API.md` §3.
type ProjectResponse struct {
	ID            uuid.UUID `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	OwnerID       uuid.UUID `json:"owner_id"`
	OwnerUsername string    `json:"owner_username,omitempty"`
	Status        string    `json:"status"`
	StartDate     *string   `json:"start_date"`
	TargetEndDate *string   `json:"target_end_date"`
	MemberCount   int       `json:"member_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ProjectMemberResponse adalah satu anggota pada response
// `GET /projects/:id/members`.
type ProjectMemberResponse struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

// ProjectDetailResponse adalah response `GET /projects/:id`: detail + anggota.
type ProjectDetailResponse struct {
	Project ProjectResponse         `json:"project"`
	Members []ProjectMemberResponse `json:"members"`
}

// NewProjectResponse memetakan model project ke bentuk response.
func NewProjectResponse(project *model.Project) ProjectResponse {
	return ProjectResponse{
		ID:            project.ID,
		Code:          project.Code,
		Name:          project.Name,
		Description:   project.Description,
		OwnerID:       project.OwnerID,
		OwnerUsername: project.OwnerUsername,
		Status:        project.Status,
		StartDate:     formatDate(project.StartDate),
		TargetEndDate: formatDate(project.TargetEndDate),
		MemberCount:   project.MemberCount,
		CreatedAt:     project.CreatedAt,
		UpdatedAt:     project.UpdatedAt,
	}
}

// NewProjectListResponse memetakan daftar project; hasilnya selalu array
// (bukan null) supaya klien tidak perlu menangani dua bentuk.
func NewProjectListResponse(projects []model.Project) []ProjectResponse {
	out := make([]ProjectResponse, 0, len(projects))
	for i := range projects {
		out = append(out, NewProjectResponse(&projects[i]))
	}
	return out
}

// NewProjectMemberResponse memetakan satu baris `project_members` + user.
func NewProjectMemberResponse(member model.ProjectMember) ProjectMemberResponse {
	return ProjectMemberResponse{
		UserID:   member.UserID,
		Username: member.Username,
		Email:    member.Email,
		Role:     member.Role,
		JoinedAt: member.CreatedAt,
	}
}

// NewProjectMemberListResponse memetakan daftar anggota.
func NewProjectMemberListResponse(members []model.ProjectMember) []ProjectMemberResponse {
	out := make([]ProjectMemberResponse, 0, len(members))
	for _, member := range members {
		out = append(out, NewProjectMemberResponse(member))
	}
	return out
}

// formatDate mengubah tanggal menjadi `YYYY-MM-DD`; nil tetap null.
func formatDate(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(DateFormat)
	return &formatted
}
