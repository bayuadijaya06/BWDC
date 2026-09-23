package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"bwdcs/backend/internal/repository"
)

// ActionReportExported adalah aksi audit untuk export laporan (42-API §10).
const ActionReportExported = "REPORT_EXPORTED"

const EntityReport = "report"

// ReportService melayani GET /reports/export (42-API §10, 50-FSD §10.6, FR-REP-01).
//
// MVP hanya CSV. Tiga tipe didukung: projects, documents, tasks.
// Isi mengikuti cakupan user 44-SECURITY §3.1.3: Manager hanya data project
// yang diikutinya, Administrator seluruh organisasi.
type ReportService struct {
	pool      *pgxpool.Pool
	projects  *repository.ProjectRepository
	documents *repository.DocumentRepository
	tasks     *repository.TaskRepository
	users     *repository.UserRepository
}

func NewReportService(
	pool *pgxpool.Pool,
	projects *repository.ProjectRepository,
	documents *repository.DocumentRepository,
	tasks *repository.TaskRepository,
	users *repository.UserRepository,
) *ReportService {
	return &ReportService{
		pool:      pool,
		projects:  projects,
		documents: documents,
		tasks:     tasks,
		users:     users,
	}
}

// ExportQuery adalah filter export yang diteruskan dari handler.
type ExportQuery struct {
	Type      string
	ProjectID *uuid.UUID
	Status    string
	Search    string
}

// ExportResult adalah hasil CSV untuk dikirim sebagai attachment.
type ExportResult struct {
	Filename string
	Content  []byte // CSV bytes
	RowCount int
}

// Export menghasilkan CSV sesuai type dan cakupan aktor.
// Audit REPORT_EXPORTED ditulis dalam satu transaksi (ADR-0011).
func (s *ReportService) Export(ctx context.Context, actor Actor, q ExportQuery) (*ExportResult, error) {
	scope, err := systemScope(ctx, s.users, actor)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	var rowCount int
	var filename string

	switch q.Type {
	case "projects":
		filename = fmt.Sprintf("bwdcs-projects-%s.csv", time.Now().Format("20060102"))
		// Header = kolom tabel 50-FSD §3.1
		if err := w.Write([]string{"code", "name", "status", "owner", "start_date", "target_end_date", "member_count", "created_at"}); err != nil {
			return nil, err
		}
		filter := repository.ProjectListFilter{
			Status: q.Status,
			Search: q.Search,
			Page:   1,
			Limit:  10000,
		}
		projects, _, err := s.projects.List(ctx, scope, filter)
		if err != nil {
			return nil, err
		}
		for _, p := range projects {
			if err := w.Write([]string{
				p.Code,
				p.Name,
				p.Status,
				p.OwnerUsername,
				formatTimePtr(p.StartDate),
				formatTimePtr(p.TargetEndDate),
				fmt.Sprintf("%d", p.MemberCount),
				p.CreatedAt.Format(time.RFC3339),
			}); err != nil {
				return nil, err
			}
		}
		rowCount = len(projects)

	case "documents":
		filename = fmt.Sprintf("bwdcs-documents-%s.csv", time.Now().Format("20060102"))
		if err := w.Write([]string{"document_number", "title", "category", "status", "version", "owner", "updated_at"}); err != nil {
			return nil, err
		}
		filter := repository.DocumentListFilter{
			ProjectID: q.ProjectID,
			Status:    q.Status,
			Search:    q.Search,
			Page:      1,
			Limit:     10000,
		}
		docs, _, err := s.documents.List(ctx, scope, filter)
		if err != nil {
			return nil, err
		}
		for _, d := range docs {
			if err := w.Write([]string{
				d.DocumentNumber,
				d.Title,
				d.CategoryName,
				d.Status,
				d.LatestVersion,
				d.OwnerUsername,
				d.UpdatedAt.Format(time.RFC3339),
			}); err != nil {
				return nil, err
			}
		}
		rowCount = len(docs)

	case "tasks":
		filename = fmt.Sprintf("bwdcs-tasks-%s.csv", time.Now().Format("20060102"))
		if err := w.Write([]string{"title", "status", "priority", "due_date", "assignee", "project"}); err != nil {
			return nil, err
		}
		// tasks memakai taskScope yang membedakan read
		taskScope, err := taskScope(ctx, s.users, actor)
		if err != nil {
			return nil, err
		}
		filter := repository.TaskListFilter{
			ProjectID: q.ProjectID,
			Status:    q.Status,
			Page:      1,
			Limit:     10000,
		}
		tasks, _, err := s.tasks.List(ctx, taskScope, filter)
		if err != nil {
			return nil, err
		}
		for _, t := range tasks {
			if err := w.Write([]string{
				t.Title,
				t.Status,
				t.Priority,
				formatTimePtr(t.DueDate),
				t.AssigneeUsername,
				t.ProjectCode,
			}); err != nil {
				return nil, err
			}
		}
		rowCount = len(tasks)

	default:
		return nil, fmt.Errorf("type tidak dikenal: %s", q.Type)
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}

	// Audit dalam transaksi (ADR-0011) — row read-only tetap dapat diaudit
	// sebagai jejak export (FR-AUDIT-01 tambahan REPORT_EXPORTED).
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		// fallback: tanpa transaksi, tetap kirim CSV
		return &ExportResult{Filename: filename, Content: buf.Bytes(), RowCount: rowCount}, nil
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionReportExported, EntityReport, q.Type,
		fmt.Sprintf("export %s %d baris", q.Type, rowCount),
		map[string]any{"type": q.Type, "row_count": rowCount}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &ExportResult{Filename: filename, Content: buf.Bytes(), RowCount: rowCount}, nil
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
