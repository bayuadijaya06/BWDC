package model

import (
	"time"

	"github.com/google/uuid"
)

// Status project kanonik (`projects.status`, `41-DATABASE.md` §2.2).
//
// Hanya dua nilai ini yang pernah tersimpan: `active` dan `archived`
// (FR-PROJ-03/FR-PROJ-07). Tidak ada nilai turunan seperti "overdue" atau
// "selesai" di kolom ini — ADR-0012 melarangnya; keadaan turunan dihitung saat
// dibaca, bukan disimpan.
const (
	ProjectStatusActive   = "active"
	ProjectStatusArchived = "archived"
)

// Role anggota project (`project_members.role`, FR-PROJ-05).
//
// Urutannya adalah hierarki **cakupan project** (owner > manager > contributor >
// viewer, `44-SECURITY.md` §3.3) dan **berbeda** dari role sistem yang dipakai
// matriks izin §3.1.2. Keduanya tidak pernah digabung menjadi satu rantai —
// penggabungan masih temuan terbuka **C-007**.
const (
	ProjectRoleOwner       = "owner"
	ProjectRoleManager     = "manager"
	ProjectRoleContributor = "contributor"
	ProjectRoleViewer      = "viewer"
)

// Project merepresentasikan tabel `projects` (`41-DATABASE.md` §2.2).
//
// `Code` tidak dapat diubah setelah dibuat karena menjadi prefiks nomor
// dokumen `{PROJECT_CODE}-{NNN}` (ADR-0017): mengubahnya akan membuat nomor
// dokumen lama menunjuk project yang salah.
type Project struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	Code           string     `json:"code"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	OwnerID        uuid.UUID  `json:"owner_id"`
	Status         string     `json:"status"`
	StartDate      *time.Time `json:"start_date"`
	TargetEndDate  *time.Time `json:"target_end_date"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// OwnerUsername dan MemberCount adalah kolom turunan untuk daftar project
	// (kolom "Owner" dan "Members count" di `50-FSD.md` §3.1). Keduanya dibaca
	// lewat JOIN/subquery, tidak disimpan di tabel.
	OwnerUsername string `json:"owner_username,omitempty"`
	MemberCount   int    `json:"member_count"`
}

// IsArchived adalah pembanding nilai kanonik status, dipakai supaya pemanggil
// tidak menuliskan string `archived` sendiri (ADR-0012).
//
// Catatan kebijakan: **tidak ada** dokumen desain yang melarang dokumen baru di
// project arsip. Kebijakan itu karena itu tidak ditegakkan di endpoint mana pun,
// dan pertanyaannya dicatat sebagai Q-016 — bukan disimpulkan dari komentar.
func (p *Project) IsArchived() bool { return p.Status == ProjectStatusArchived }

// ProjectMember merepresentasikan tabel `project_members` (`41-DATABASE.md` §2.2)
// beserta identitas user untuk daftar anggota (`GET /projects/:id/members`).
type ProjectMember struct {
	ProjectID uuid.UUID `json:"project_id"`
	UserID    uuid.UUID `json:"user_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`

	// Username dan Email adalah kolom turunan hasil JOIN ke `users`.
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
}

// ProjectRoles mengembalikan daftar role project yang sah, dipakai validasi
// input `POST /projects/:id/members` dan penjaga migrasi.
func ProjectRoles() []string {
	return []string{ProjectRoleOwner, ProjectRoleManager, ProjectRoleContributor, ProjectRoleViewer}
}

// IsProjectRole menjawab apakah `role` termasuk daftar tertutup FR-PROJ-05.
func IsProjectRole(role string) bool {
	for _, valid := range ProjectRoles() {
		if role == valid {
			return true
		}
	}
	return false
}
