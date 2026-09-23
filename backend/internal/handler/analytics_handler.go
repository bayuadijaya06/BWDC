package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bwdcs/backend/internal/dto"
	"bwdcs/backend/internal/middleware"
	"bwdcs/backend/internal/pkg/response"
	"bwdcs/backend/internal/service"
)

// AnalyticsHandler melayani `GET /analytics/dashboard` (`42-API.md` §13, `52-*`).
type AnalyticsHandler struct {
	analytics *service.AnalyticsService
	logger    *slog.Logger
}

func NewAnalyticsHandler(analytics *service.AnalyticsService, logger *slog.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{analytics: analytics, logger: logger}
}

// Dashboard melayani `GET /analytics/dashboard`.
//
// Izin `report:read` diperiksa middleware; cakupan barisnya diterapkan di
// kueri (`44-SECURITY.md` §3.1.3) via service.
func (h *AnalyticsHandler) Dashboard(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	q, fields := parseAnalyticsQuery(c)
	if len(fields) > 0 {
		response.Validation(c, fields)
		return
	}
	data, err := h.analytics.Dashboard(c.Request.Context(), actor, q)
	if err != nil {
		h.logger.Error("dashboard analytics gagal", "error", err.Error(), "correlation_id", middleware.CurrentCorrelationID(c))
		response.Internal(c)
		return
	}
	response.OK(c, data)
}

func parseAnalyticsQuery(c *gin.Context) (dto.AnalyticsQuery, []response.FieldError) {
	var fields []response.FieldError
	var q dto.AnalyticsQuery

	if raw := c.Query("from"); raw != "" {
		parsed, err := parseRFC3339Query(raw)
		if err != nil {
			fields = append(fields, response.FieldError{Field: "from", Error: "harus waktu RFC 3339 yang sah"})
		} else {
			q.From = &parsed
		}
	}
	if raw := c.Query("to"); raw != "" {
		parsed, err := parseRFC3339Query(raw)
		if err != nil {
			fields = append(fields, response.FieldError{Field: "to", Error: "harus waktu RFC 3339 yang sah"})
		} else {
			q.To = &parsed
		}
	}
	if q.From != nil && q.To != nil && q.To.Before(*q.From) {
		fields = append(fields, response.FieldError{Field: "to", Error: "harus lebih besar atau sama dengan from (kedua batas inklusif)"})
	}
	if raw := c.Query("project_id"); raw != "" {
		if _, err := uuid.Parse(raw); err != nil {
			fields = append(fields, response.FieldError{Field: "project_id", Error: "harus UUID yang sah"})
		} else {
			q.ProjectID = &raw
		}
	}
	if raw := c.Query("status"); raw != "" {
		// status diteruskan apa adanya untuk funnel filter, validasi lanjutan di repository (abaikan bila tidak dikenal)
		q.Status = raw
	}
	return q, fields
}
