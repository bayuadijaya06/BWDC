package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"bwdcs/backend/internal/repository"
	"bwdcs/backend/internal/service"
)

// TestRBACPermissions adalah test `70-TESTING.md` §4.1. Izin dibaca dari
// tabel `role_permissions` hasil seed `008` — bukan dari cabang kode — karena
// FR-ROLE-03 mewajibkan matriks ADR-0014 yang berlaku, dan bypass di kode
// dilarang dipakai sebagai bukti.
func TestRBACPermissions(t *testing.T) {
	pool := requirePool(t)
	checker := service.NewPermissionChecker(repository.NewUserRepository(pool))
	ctx := context.Background()

	actors := map[string]uuid.UUID{
		"administrator": createActor(t, "administrator").ID,
		"manager":       createActor(t, "manager").ID,
		"contributor":   createActor(t, "contributor").ID,
		"viewer":        createActor(t, "viewer").ID,
	}

	cases := []struct {
		name     string
		role     string
		resource string
		action   string
		expected bool
	}{
		{"administrator dapat menghapus dokumen", "administrator", "document", "delete", true},
		{"viewer tidak dapat membuat project", "viewer", "project", "create", false},
		{"contributor dapat mengunggah versi", "contributor", "document_version", "upload", true},
		{"manager dapat approve", "manager", "workflow_instance", "approve", true},
		{"viewer tidak dapat approve", "viewer", "workflow_instance", "approve", false},
		{"manager tidak boleh membaca audit log", "manager", "audit", "read", false},
		{"viewer boleh membaca task (dipersempit scoping, bukan ditolak)", "viewer", "task", "read", true},
		{"administrator dapat mengelola role user", "administrator", "user_role", "manage", true},
		{"contributor tidak dapat mengelola role user", "contributor", "user_role", "manage", false},
		{"viewer tidak dapat mengubah pengaturan", "viewer", "setting", "manage", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			userID, ok := actors[tc.role]
			if !ok {
				t.Fatalf("role %q tidak punya aktor uji", tc.role)
			}

			got, err := checker.HasPermission(ctx, userID, tc.resource, tc.action)
			if err != nil {
				t.Fatalf("periksa izin: %v", err)
			}
			if got != tc.expected {
				t.Errorf("izin %s %s:%s = %v, diharapkan %v", tc.role, tc.resource, tc.action, got, tc.expected)
			}
		})
	}
}

// TestPermissionCheckerWithoutRole menegaskan tidak ada jalan pintas: user
// tanpa role tidak memiliki izin apa pun, termasuk izin yang terlihat "umum".
func TestPermissionCheckerWithoutRole(t *testing.T) {
	pool := requirePool(t)
	checker := service.NewPermissionChecker(repository.NewUserRepository(pool))
	actor := createActor(t)

	for _, pair := range [][2]string{{"project", "read"}, {"audit", "read"}, {"document", "create"}} {
		got, err := checker.HasPermission(context.Background(), actor.ID, pair[0], pair[1])
		if err != nil {
			t.Fatalf("periksa izin: %v", err)
		}
		if got {
			t.Errorf("user tanpa role tidak boleh punya %s:%s", pair[0], pair[1])
		}
	}
}

// TestPermissionCheckerUnknownPair memastikan pasangan di luar matriks selalu
// ditolak, bukan diam-diam lolos. Kosakata tertutupnya ada di
// `44-SECURITY.md` §3.1.1.
func TestPermissionCheckerUnknownPair(t *testing.T) {
	pool := requirePool(t)
	checker := service.NewPermissionChecker(repository.NewUserRepository(pool))
	admin := createActor(t, "administrator")

	got, err := checker.HasPermission(context.Background(), admin.ID, "document", "teleport")
	if err != nil {
		t.Fatalf("periksa izin: %v", err)
	}
	if got {
		t.Error("action di luar kosakata tidak boleh diizinkan")
	}
}

// TestPermissionCheckerCounts mengunci hitungan izin per role (44/30/18/12)
// lewat jalur service, bukan hanya lewat SQL test migrasi — supaya pemetaan
// role → izin terbukti utuh sampai ke pemakainya.
func TestPermissionCheckerCounts(t *testing.T) {
	pool := requirePool(t)
	checker := service.NewPermissionChecker(repository.NewUserRepository(pool))

	want := map[string]int{
		"administrator": 44,
		"manager":       30,
		"contributor":   18,
		"viewer":        12,
	}

	for role, expected := range want {
		actor := createActor(t, role)

		perms, err := checker.Permissions(context.Background(), actor.ID)
		if err != nil {
			t.Fatalf("baca izin %s: %v", role, err)
		}
		if len(perms) != expected {
			t.Errorf("%s punya %d izin, diharapkan %d (44-SECURITY.md §3.1.2)", role, len(perms), expected)
		}
	}
}
