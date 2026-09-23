package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bwdcs/backend/internal/pkg/response"
	"bwdcs/backend/internal/service"
)

// UserHandler melayani bagian Administration > Users `42-API.md` §11.
//
// Handler hanya: parse input, memanggil service, memetakan error domain ke
// status HTTP. Penulisan audit dan transaksi adalah milik service (ADR-0011).
type UserHandler struct {
	users  *service.UserService
	logger *slog.Logger
}

// NewUserHandler merakit handler administrasi user.
func NewUserHandler(users *service.UserService, logger *slog.Logger) *UserHandler {
	return &UserHandler{users: users, logger: logger}
}

// Unlock melayani `POST /admin/users/:id/unlock` (ADR-0022 butir 5).
//
// Pemetaan error (`42-API.md` §11/§12):
//
//	422 VALIDATION_ERROR — `:id` bukan UUID
//	404 NOT_FOUND        — user tidak ada
//
// Izin `user:update` dipasang di route (`44-SECURITY.md` §3.1.2: hanya
// Administrator), bukan diperiksa ulang di sini.
func (h *UserHandler) Unlock(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	userID, ok := userIDParam(c)
	if !ok {
		return
	}

	if err := h.users.Unlock(c.Request.Context(), actor.ID, userID); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.Fail(c, http.StatusNotFound, response.CodeNotFound, "user not found")
			return
		}
		h.logger.Error("unlock akun gagal", "user_id", userID.String(), "error", err.Error())
		response.Internal(c)
		return
	}

	response.OK(c, nil)
}

// userIDParam membaca `:id` dan memetakan UUID tidak sah ke
// `422 VALIDATION_ERROR` dengan nama field-nya (`42-API.md` §12, temuan C-045).
func userIDParam(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Validation(c, []response.FieldError{{Field: "id", Error: "harus UUID yang sah"}})
		return uuid.Nil, false
	}
	return id, true
}
