package handler_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"bwdcs/backend/internal/handler"
	"bwdcs/backend/internal/middleware"
	"bwdcs/backend/internal/migration"
	"bwdcs/backend/internal/pkg/filestorage"
	bwjwt "bwdcs/backend/internal/pkg/jwt"
	"bwdcs/backend/internal/repository"
	"bwdcs/backend/internal/service"
)

var testPool *pgxpool.Pool

const (
	testPassword  = "kata-sandi-uji-handler-2026"
	testJWTSecret = "rahasia-uji-integrasi-handler-lebih-dari-32-karakter"
)

func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "TEST_DATABASE_URL tidak diset: test integrasi handler dilewati")
		os.Exit(m.Run())
	}

	logger := discardLogger()
	if err := migration.Up(context.Background(), dsn, logger); err != nil {
		fmt.Fprintf(os.Stderr, "migrasi database test gagal: %v\n", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "buka pool test: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	gin.SetMode(gin.TestMode)
	testPool = pool
	os.Exit(m.Run())
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// stubStorage memenuhi `filestorage.Prober` supaya health handler dapat dirakit
// tanpa menyentuh disk; test ini tidak memanggil `GET /health`.
type stubStorage struct{ err error }

func (s stubStorage) Ping() error { return s.err }

// engineParts adalah komponen yang dirakit `newEngineParts`. Test modul tertentu
// membutuhkannya untuk menyiapkan data — modul dokumen, mis. memerlukan storage
// nyata supaya unggahan dan unduhan benar-benar menyentuh berkas.
type engineParts struct {
	engine    *gin.Engine
	storage   *filestorage.LocalStorage
	projects  *service.ProjectService
	documents *service.DocumentService
	tasks     *service.TaskService
}

// newEngine merakit engine lengkap seperti `cmd/server/main.go` merakitnya.
func newEngine(t *testing.T, maxLoginAttempts int) *gin.Engine {
	t.Helper()
	return newEngineParts(t, maxLoginAttempts).engine
}

// newEngineParts merakit engine beserta service yang dipasang di dalamnya.
func newEngineParts(t *testing.T, maxLoginAttempts int) engineParts {
	t.Helper()
	requirePool(t)

	tokens, err := bwjwt.New(bwjwt.Config{Secret: testJWTSecret, Expiry: time.Hour})
	if err != nil {
		t.Fatalf("buat service token: %v", err)
	}

	store, err := filestorage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("siapkan storage uji: %v", err)
	}

	users := repository.NewUserRepository(testPool)
	revocations := repository.NewRevocationRepository(testPool, 0)
	authService := service.NewAuthService(testPool, users, revocations, tokens,
		service.NewLoginGuard(maxLoginAttempts, 0), discardLogger())
	projectService := service.NewProjectService(testPool, repository.NewProjectRepository(testPool), users, discardLogger())
	documentService := service.NewDocumentService(testPool,
		repository.NewDocumentRepository(testPool),
		repository.NewProjectRepository(testPool),
		users, store, discardLogger())
	taskService := service.NewTaskService(testPool,
		repository.NewTaskRepository(testPool),
		repository.NewProjectRepository(testPool),
		users, discardLogger())

	permissionChecker := service.NewPermissionChecker(users)

	engine := gin.New()
	handler.Setup(engine, handler.RouterDeps{
		Logger:       discardLogger(),
		IsProduction: false,
		Health:       handler.NewHealthHandler(testPool, stubStorage{}),
		Auth:         handler.NewAuthHandler(authService, discardLogger()),
		Project:      handler.NewProjectHandler(projectService, discardLogger()),
		Document:     handler.NewDocumentHandler(documentService, discardLogger()),
		Task:         handler.NewTaskHandler(taskService, permissionChecker, discardLogger()),
		AuthMiddleware: middleware.AuthMiddleware(middleware.AuthConfig{
			Validator:   tokens,
			Revocations: revocations,
		}),
		Permission: permissionChecker,
	})

	return engineParts{
		engine:    engine,
		storage:   store,
		projects:  projectService,
		documents: documentService,
		tasks:     taskService,
	}
}

// testActor adalah user yang benar-benar ter-commit: service auth membuka
// transaksinya sendiri, sehingga user di dalam transaksi test tidak terlihat.
type testActor struct {
	ID       uuid.UUID
	OrgID    uuid.UUID
	Username string
	Password string
}

// requirePool melewati test bila TEST_DATABASE_URL tidak diset, sehingga
// `go test ./...` tetap hijau di mesin tanpa database.
func requirePool(t *testing.T) {
	t.Helper()
	if testPool == nil {
		t.Skip("TEST_DATABASE_URL tidak diset; test integrasi butuh PostgreSQL")
	}
}

func createActor(t *testing.T, roles ...string) testActor {
	t.Helper()
	requirePool(t)

	ctx := context.Background()
	suffix := uuid.NewString()[:8]

	actor := testActor{Username: "uji-http-" + suffix, Password: testPassword}

	if err := testPool.QueryRow(ctx,
		`INSERT INTO organizations (name, code) VALUES ($1, $2) RETURNING id`,
		"Organisasi Uji Handler", "UJI-HTTP-"+suffix,
	).Scan(&actor.OrgID); err != nil {
		t.Fatalf("buat organisasi uji: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password uji: %v", err)
	}

	if err := testPool.QueryRow(ctx,
		`INSERT INTO users (organization_id, username, email, password_hash)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		actor.OrgID, actor.Username, actor.Username+"@example.invalid", string(hash),
	).Scan(&actor.ID); err != nil {
		t.Fatalf("buat user uji: %v", err)
	}

	for _, role := range roles {
		if _, err := testPool.Exec(ctx,
			`INSERT INTO user_roles (user_id, role_id) SELECT $1, id FROM roles WHERE name = $2`,
			actor.ID, role,
		); err != nil {
			t.Fatalf("tetapkan role %s: %v", role, err)
		}
	}

	t.Cleanup(func() { cleanupActor(t, actor.ID, actor.OrgID) })
	return actor
}

// cleanupActor memakai jalur pemeliharaan `bwdcs.audit_maintenance` karena
// `audit_logs` append-only (FR-AUDIT-03; `70-TESTING.md` §8).
func cleanupActor(t *testing.T, userID, orgID uuid.UUID) {
	t.Helper()
	if testPool == nil {
		return
	}

	ctx := context.Background()
	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Errorf("mulai transaksi pembersihan: %v", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	steps := []struct {
		sql  string
		args []any
	}{
		{`SET LOCAL bwdcs.audit_maintenance = 'on'`, nil},
		{`DELETE FROM audit_logs WHERE actor_id = $1`, []any{userID}},
		{`DELETE FROM user_roles WHERE user_id = $1`, []any{userID}},
		{`DELETE FROM users WHERE id = $1`, []any{userID}},
		{`DELETE FROM organizations WHERE id = $1`, []any{orgID}},
	}
	for _, step := range steps {
		if _, err := tx.Exec(ctx, step.sql, step.args...); err != nil {
			t.Errorf("pembersihan gagal pada %q: %v", step.sql, err)
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		t.Errorf("commit pembersihan: %v", err)
	}
}

// apiResponse adalah amplop response yang dipakai untuk membaca hasil test.
type apiResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
		User      struct {
			ID       string   `json:"id"`
			Username string   `json:"username"`
			Roles    []string `json:"roles"`
		} `json:"user"`
		ID          string   `json:"id"`
		Username    string   `json:"username"`
		Roles       []string `json:"roles"`
		Permissions []string `json:"permissions"`
	} `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
