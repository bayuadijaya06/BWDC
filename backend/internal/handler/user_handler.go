package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

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

// ListUsers melayani GET /admin/users?search=&page=&limit= (42-API §11, user:read)
func (h *UserHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")
	if page < 1 || limit < 1 || limit > 100 {
		var fields []response.FieldError
		if page < 1 {
			fields = append(fields, response.FieldError{Field: "page", Error: "harus >= 1"})
		}
		if limit < 1 || limit > 100 {
			fields = append(fields, response.FieldError{Field: "limit", Error: "harus 1-100"})
		}
		if len(fields) > 0 {
			response.Validation(c, fields)
			return
		}
	}
	users, total, err := h.users.ListUsers(c.Request.Context(), service.UserListFilter{Search: search, Page: page, Limit: limit})
	if err != nil {
		h.logger.Error("daftar user gagal", "error", err.Error())
		response.Internal(c)
		return
	}
	totalPage := (total + limit - 1) / limit
	if total == 0 {
		totalPage = 0
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    users,
		"meta": gin.H{"page": page, "limit": limit, "total": total, "total_page": totalPage},
	})
}

// CreateUser melayani POST /admin/users (42-API §11, user:create)
func (h *UserHandler) CreateUser(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}
	var req struct {
		Username string   `json:"username"`
		Email    string   `json:"email"`
		Password string   `json:"password"`
		RoleIDs  []string `json:"role_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, []response.FieldError{{Field: "body", Error: "harus JSON objek yang sah"}})
		return
	}
	var fields []response.FieldError
	if req.Username == "" {
		fields = append(fields, response.FieldError{Field: "username", Error: "wajib diisi"})
	}
	if req.Email == "" {
		fields = append(fields, response.FieldError{Field: "email", Error: "wajib diisi"})
	}
	if req.Password == "" {
		fields = append(fields, response.FieldError{Field: "password", Error: "wajib diisi"})
	} else if len(req.Password) < 8 {
		fields = append(fields, response.FieldError{Field: "password", Error: "minimal 8 karakter"})
	}
	if len(req.RoleIDs) == 0 {
		fields = append(fields, response.FieldError{Field: "role_ids", Error: "wajib diisi"})
	}
	var roleIDs []uuid.UUID
	for i, s := range req.RoleIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			fields = append(fields, response.FieldError{Field: "role_ids[" + strconv.Itoa(i) + "]", Error: "harus UUID yang sah"})
		} else {
			roleIDs = append(roleIDs, id)
		}
	}
	if len(fields) > 0 {
		response.Validation(c, fields)
		return
	}
	user, err := h.users.CreateUser(c.Request.Context(), actor.ID, service.CreateUserInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		RoleIDs:  roleIDs,
	})
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			response.Fail(c, http.StatusConflict, response.CodeConflict, "username atau email sudah dipakai")
			return
		}
		if errors.Is(err, service.ErrRoleNotFound) {
			response.Validation(c, []response.FieldError{{Field: "role_ids", Error: "role tidak ditemukan"}})
			return
		}
		h.logger.Error("buat user gagal", "error", err.Error())
		response.Internal(c)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": user})
}

// ListRoles melayani GET /admin/roles (42-API §11, role:read)
func (h *UserHandler) ListRoles(c *gin.Context) {
	roles, err := h.users.ListRoles(c.Request.Context())
	if err != nil {
		h.logger.Error("daftar role gagal", "error", err.Error())
		response.Internal(c)
		return
	}
	// map ke response shape
	out := make([]gin.H, 0, len(roles))
	for _, r := range roles {
		out = append(out, gin.H{"id": r.ID.String(), "name": r.Name})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}

// ListOrganizations melayani GET /admin/organizations (42-API §11, organization:read)
func (h *UserHandler) ListOrganizations(c *gin.Context) {
	orgs, err := h.users.ListOrganizations(c.Request.Context())
	if err != nil {
		h.logger.Error("daftar organisasi gagal", "error", err.Error())
		response.Internal(c)
		return
	}
	out := make([]gin.H, 0, len(orgs))
	for _, o := range orgs {
		out = append(out, gin.H{"id": o.ID.String(), "name": o.Name, "code": o.Code})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
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
