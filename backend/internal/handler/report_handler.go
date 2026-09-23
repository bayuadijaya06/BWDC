package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bwdcs/backend/internal/middleware"
	"bwdcs/backend/internal/pkg/response"
	"bwdcs/backend/internal/service"
)

// ReportHandler melayani GET /reports/export (42-API §10, 50-FSD §10.6, FR-REP-01).
type ReportHandler struct {
	reports *service.ReportService
	logger  *slog.Logger
}

func NewReportHandler(reports *service.ReportService, logger *slog.Logger) *ReportHandler {
	return &ReportHandler{reports: reports, logger: logger}
}

// Export melayani GET /reports/export?type=projects|documents|tasks&format=csv
//
// Izin report:export diperiksa middleware; cakupan barisnya diterapkan di
// kueri via service (44-SECURITY §3.1.3). MVP hanya CSV.
func (h *ReportHandler) Export(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	q, fields := parseReportExportQuery(c)
	if len(fields) > 0 {
		response.Validation(c, fields)
		return
	}
	result, err := h.reports.Export(c.Request.Context(), actor, q)
	if err != nil {
		h.logger.Error("export laporan gagal", "error", err.Error(), "correlation_id", middleware.CurrentCorrelationID(c))
		response.Internal(c)
		return
	}
	c.Header("Content-Disposition", "attachment; filename=\""+result.Filename+"\"")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", result.Content)
}

func parseReportExportQuery(c *gin.Context) (service.ExportQuery, []response.FieldError) {
	var fields []response.FieldError
	var q service.ExportQuery

	rawType := c.Query("type")
	if rawType != "projects" && rawType != "documents" && rawType != "tasks" {
		fields = append(fields, response.FieldError{Field: "type", Error: "harus salah satu dari projects, documents, tasks"})
	} else {
		q.Type = rawType
	}
	rawFormat := c.Query("format")
	if rawFormat != "csv" {
		fields = append(fields, response.FieldError{Field: "format", Error: "harus csv"})
	}
	if raw := c.Query("project_id"); raw != "" {
		if _, err := uuid.Parse(raw); err != nil {
			fields = append(fields, response.FieldError{Field: "project_id", Error: "harus UUID yang sah"})
		} else {
			parsed, _ := uuid.Parse(raw)
			q.ProjectID = &parsed
		}
	}
	if raw := c.Query("status"); raw != "" {
		q.Status = raw
	}
	if raw := c.Query("search"); raw != "" {
		q.Search = raw
	}
	return q, fields
}
