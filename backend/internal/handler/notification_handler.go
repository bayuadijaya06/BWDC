package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bwdcs/backend/internal/middleware"
	"bwdcs/backend/internal/pkg/response"
	"bwdcs/backend/internal/repository"
	"bwdcs/backend/internal/service"
)

// NotificationHandler melayani `42-API.md` §8.
type NotificationHandler struct {
	notifications *service.NotificationService
	logger        *slog.Logger
}

func NewNotificationHandler(notifications *service.NotificationService, logger *slog.Logger) *NotificationHandler {
	return &NotificationHandler{notifications: notifications, logger: logger}
}

// List melayani `GET /notifications?is_read=&page=&limit=`.
func (h *NotificationHandler) List(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	var isRead *bool
	if raw := strings.TrimSpace(c.Query("is_read")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			response.Validation(c, []response.FieldError{{Field: "is_read", Error: "harus true atau false"}})
			return
		}
		isRead = &parsed
	}
	page, err := parsePositiveInt(c.Query("page"), 1)
	if err != nil {
		response.Validation(c, []response.FieldError{{Field: "page", Error: "harus angka >= 1"}})
		return
	}
	limit, err := parsePositiveInt(c.Query("limit"), 20)
	if err != nil || limit > 100 {
		response.Validation(c, []response.FieldError{{Field: "limit", Error: "harus angka 1 sampai 100"}})
		return
	}
	notifications, total, err := h.notifications.List(c.Request.Context(), actor, isRead, page, limit)
	if err != nil {
		h.logger.Error("baca notifikasi gagal", "error", err.Error(), "correlation_id", middleware.CurrentCorrelationID(c))
		response.Internal(c)
		return
	}
	totalPage := 0
	if total > 0 {
		totalPage = (total + limit - 1) / limit
	}
	response.OKWithMeta(c, notifications, response.Meta{Page: page, Limit: limit, Total: total, TotalPage: totalPage})
}

// MarkRead melayani `PATCH /notifications/:id/read`.
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Validation(c, []response.FieldError{{Field: "id", Error: "harus UUID yang sah"}})
		return
	}
	if err := h.notifications.MarkRead(c.Request.Context(), actor, id); err != nil {
		if err == repository.ErrNotFound {
			response.Fail(c, http.StatusNotFound, response.CodeNotFound, "notification not found")
			return
		}
		h.logger.Error("tandai notifikasi gagal", "error", err.Error(), "correlation_id", middleware.CurrentCorrelationID(c))
		response.Internal(c)
		return
	}
	response.OK(c, map[string]any{"id": id.String(), "is_read": true})
}

// MarkAllRead melayani `POST /notifications/read-all`.
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	count, err := h.notifications.MarkAllRead(c.Request.Context(), actor)
	if err != nil {
		h.logger.Error("tandai semua notifikasi gagal", "error", err.Error(), "correlation_id", middleware.CurrentCorrelationID(c))
		response.Internal(c)
		return
	}
	response.OK(c, map[string]any{"updated": count})
}
