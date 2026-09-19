package service

import (
	"context"

	"github.com/google/uuid"

	"bwdcs/backend/internal/repository"
)

// Role sistem yang disebut namanya oleh aturan cakupan `44-SECURITY.md` §3.1.3.
const (
	systemRoleAdministrator = "administrator"
	systemRoleManager       = "manager"
	systemRoleContributor   = "contributor"
)

// Actor adalah pemanggil request: identitas minimum yang dibutuhkan service
// untuk menerapkan cakupan data.
//
// Tipe ini sengaja bukan `middleware.AuthUser` supaya lapisan service tidak
// bergantung pada lapisan HTTP (`40-TSD.md` §2.4). Semua modul yang bercakupan
// project (project, document, dan kelak task/comment/workflow) memakai tipe
// yang sama.
type Actor struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
}

// systemRoleSet membaca himpunan role **sistem** aktor sekali, dipakai semua
// penyusun cakupan supaya aturan role tidak ditulis ulang per modul.
func systemRoleSet(ctx context.Context, users *repository.UserRepository, actor Actor) (map[string]bool, error) {
	roles, err := users.Roles(ctx, actor.ID)
	if err != nil {
		return nil, err
	}

	set := make(map[string]bool, len(roles))
	for _, role := range roles {
		set[role] = true
	}
	return set, nil
}

// systemScope menyusun cakupan data aktor dari role **sistem**-nya.
//
// Satu fungsi, bukan salinan per modul: `44-SECURITY.md` §3.1.3 menetapkan satu
// aturan cakupan untuk `project`, `document`, `document_version`, dan kelak
// `task`/`comment` — dan aturan yang disalin per modul adalah aturan yang
// berbeda-beda diam-diam.
//
// `AllInOrganization` diisi hanya bila user berrole `administrator`; itu bukan
// bypass izin: matriks §3.1.2 tetap menentukan boleh-tidaknya, dan pemeriksa
// izin tidak punya cabang khusus Administrator (temuan C-037).
func systemScope(ctx context.Context, users *repository.UserRepository, actor Actor) (repository.ProjectScope, error) {
	roles, err := systemRoleSet(ctx, users, actor)
	if err != nil {
		return repository.ProjectScope{}, err
	}

	return repository.ProjectScope{
		OrganizationID:    actor.OrganizationID,
		UserID:            actor.ID,
		AllInOrganization: roles[systemRoleAdministrator],
	}, nil
}

// taskScope menyusun cakupan data task dari role sistem aktor.
//
// Modul task punya **cakupan baca yang lebih luas untuk Manager** daripada
// project: `44-SECURITY.md` §3.1.3 menyebut "`task:read` (Contributor, Viewer)
// — hanya task miliknya dan task pada project yang diikutinya. Manager/
// Administrator: seluruh organisasi", sedangkan dua baris lain mengetatkan
// **tulis** untuk Contributor (hanya task yang ditugaskan kepadanya atau
// dibuatnya). Karena itu scope ini tidak dapat memakai `systemScope` apa adanya:
// asimetri itu memang yang ditetapkan dokumen.
//
// Bila aktor punya beberapa role, yang dipakai adalah yang **terluas**
// (administrator > manager > contributor) — cerminan cara matriks §3.1.2
// bekerja, yaitu gabungan izin seluruh role.
func taskScope(ctx context.Context, users *repository.UserRepository, actor Actor) (repository.TaskScope, error) {
	roles, err := systemRoleSet(ctx, users, actor)
	if err != nil {
		return repository.TaskScope{}, err
	}

	isAdmin := roles[systemRoleAdministrator]
	isManager := roles[systemRoleManager]

	scope := repository.TaskScope{
		OrganizationID: actor.OrganizationID,
		UserID:         actor.ID,

		// Baca: administrator dan manager melihat seluruh organisasi.
		AllInOrganization: isAdmin || isManager,

		// Tulis: administrator tanpa batas project; manager hanya pada project
		// yang ia ikuti (baris dasar §3.1.3 untuk `task`); contributor hanya
		// task miliknya/ditugaskan kepadanya.
		WriteAllInOrganization: isAdmin,
		WriteMemberProjects:    isManager,
		WriteOwnTasksOnly:      !isAdmin && !isManager && roles[systemRoleContributor],
	}

	return scope, nil
}
