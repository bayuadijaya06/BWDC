package handler

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bwdcs/backend/internal/middleware"
	"bwdcs/backend/internal/pkg/response"
	"bwdcs/backend/internal/repository"
	"bwdcs/backend/internal/service"
)

// AuditHandler melayani `GET /audit` (`42-API.md` §9).
type AuditHandler struct {
	audits *service.AuditReadService
	logger *slog.Logger
}

func NewAuditHandler(audits *service.AuditReadService, logger *slog.Logger) *AuditHandler {
	return &AuditHandler{audits: audits, logger: logger}
}

// List melayani `GET /audit`.
func (h *AuditHandler) List(c *gin.Context) {
	filter, fields := parseAuditListQuery(c)
	if len(fields) > 0 {
		response.Validation(c, fields)
		return
	}
	logs, total, err := h.audits.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("baca audit gagal", "error", err.Error(), "correlation_id", middleware.CurrentCorrelationID(c))
		response.Internal(c)
		return
	}
	totalPage := 0
	if total > 0 {
		totalPage = (total + filter.Limit - 1) / filter.Limit
	}
	response.OKWithMeta(c, logs, response.Meta{Page: filter.Page, Limit: filter.Limit, Total: total, TotalPage: totalPage})
}

func parseAuditListQuery(c *gin.Context) (repository.AuditListFilter, []response.FieldError) {
	var fields []response.FieldError

	page, err := parsePositiveInt(c.Query("page"), 1)
	if err != nil {
		fields = append(fields, response.FieldError{Field: "page", Error: "harus angka >= 1"})
	}
	limit, err := parsePositiveInt(c.Query("limit"), 50)
	if err != nil || limit > 100 {
		fields = append(fields, response.FieldError{Field: "limit", Error: "harus angka 1 sampai 100"})
	}

	var actorID *uuid.UUID
	if raw := strings.TrimSpace(c.Query("actor_id")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			fields = append(fields, response.FieldError{Field: "actor_id", Error: "harus UUID yang sah"})
		} else {
			actorID = &parsed
		}
	}

	action := strings.TrimSpace(c.Query("action"))
	entity := strings.TrimSpace(c.Query("entity"))
	entityID := strings.TrimSpace(c.Query("entity_id"))

	var projectID *uuid.UUID
	if raw := strings.TrimSpace(c.Query("project_id")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			fields = append(fields, response.FieldError{Field: "project_id", Error: "harus UUID yang sah"})
		} else {
			projectID = &parsed
		}
	}

	var dateFrom, dateTo *time.Time
	if raw := strings.TrimSpace(c.Query("date_from")); raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			// Coba RFC3339 juga
			if p2, e2 := parseRFC3339Query(raw); e2 == nil {
				parsed = p2
			} else {
				fields = append(fields, response.FieldError{Field: "date_from", Error: "harus tanggal YYYY-MM-DD atau waktu RFC 3339 yang sah"})
			}
		}
		if len(fields) == 0 || fields[len(fields)-1].Field != "date_from" {
			dateFrom = &parsed
		}
	}
	if raw := strings.TrimSpace(c.Query("date_to")); raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			if p2, e2 := parseRFC3339Query(raw); e2 == nil {
				parsed = p2
			} else {
				fields = append(fields, response.FieldError{Field: "date_to", Error: "harus tanggal YYYY-MM-DD atau waktu RFC 3339 yang sah"})
			}
		}
		if len(fields) == 0 || fields[len(fields)-1].Field != "date_to" {
			// date_to inclusive: set ke akhir hari bila hanya tanggal
			if len(raw) == 10 {
				end := parsed.Add(24*time.Hour - time.Nanosecond)
				dateTo = &end
			} else {
				dateTo = &parsed
			}
		}
	}
	if dateFrom != nil && dateTo != nil && dateTo.Before(*dateFrom) {
		fields = append(fields, response.FieldError{Field: "date_to", Error: "harus lebih besar atau sama dengan date_from"})
	}

	if len(fields) > 0 {
		return repository.AuditListFilter{}, fields
	}
	return repository.AuditListFilter{
		ActorID:   actorID,
		Action:    action,
		Entity:    entity,
		EntityID:  entityID,
		DateFrom:  dateFrom,
		DateTo:    dateTo,
		ProjectID: projectID,
		Page:      page,
		Limit:     limit,
	}, nil
}
