package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bwdcs/backend/internal/dto"
	"bwdcs/backend/internal/middleware"
	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/pkg/response"
	"bwdcs/backend/internal/service"
)

// CommentHandler melayani bab Comments `42-API.md` §7.
//
// Handler hanya: parse & validasi input, memanggil service, lalu memetakan error
// domain ke status HTTP. Cakupan baris dan kepemilikan komentar ditegakkan di
// kueri (`44-SECURITY.md` §3.1.3) — bukan di sini.
//
// Izin yang dijaga middleware adalah `comment:read` dan `comment:create`, satu-
// satunya pasangan yang ada di matriks ADR-0014. **Tidak ada `comment:update`
// atau `comment:delete` di matriks**, dan itu bukan kelalaian: §3.1.3 menetapkan
// edit/hapus komentar sebagai soal **kepemilikan**, bukan izin role. Karena itu
// `PATCH` dan `DELETE` dijaga `comment:read` — pemisahan "boleh mengubah"
// datang dari `created_by_id = actor` di dalam `WHERE`, sehingga Contributor dan
// Viewer pun dapat menyunting komentarnya sendiri (`50-FSD.md` §7: "Edit/delete
// (own comments only)"). Menambah pasangan izin baru di sini akan mengubah
// matriks tanpa ADR.
type CommentHandler struct {
	comments *service.CommentService
	logger   *slog.Logger
}

// NewCommentHandler merakit handler komentar.
func NewCommentHandler(comments *service.CommentService, logger *slog.Logger) *CommentHandler {
	return &CommentHandler{comments: comments, logger: logger}
}

// List melayani `GET /comments?entity_type=&entity_id=&page=&limit=`.
//
// Izin `comment:read` diperiksa middleware; cakupan barisnya diterapkan di kueri
// dengan project yang diturunkan dari entitas komentar. Entitas di luar cakupan
// menghasilkan daftar kosong, bukan `403` — sama seperti daftar project dan
// dokumen, dan sama seperti aturan §3.1.3 ("pelanggaran cakupan dibalas 404",
// yang di sini berarti "tidak ada baris yang boleh dikirim").
func (h *CommentHandler) List(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	entityType, entityID, page, limit, fields := parseCommentListQuery(c)
	if len(fields) > 0 {
		response.Validation(c, fields)
		return
	}

	comments, total, err := h.comments.List(c.Request.Context(), actor, entityType, entityID, page, limit)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	totalPage := 0
	if total > 0 {
		totalPage = (total + limit - 1) / limit
	}

	response.OKWithMeta(c, dto.NewCommentListResponse(comments), response.Meta{
		Page:      page,
		Limit:     limit,
		Total:     total,
		TotalPage: totalPage,
	})
}

// Get melayani `GET /comments/:id` (izin `comment:read`).
func (h *CommentHandler) Get(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	commentID, ok := commentIDParam(c)
	if !ok {
		return
	}

	comment, err := h.comments.Get(c.Request.Context(), actor, commentID)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	response.OK(c, dto.NewCommentResponse(comment))
}

// Create melayani `POST /comments` (201, izin `comment:create`, FR-CMT-01).
func (h *CommentHandler) Create(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	var req dto.CreateCommentRequest
	if !bindJSON(c, &req) {
		return
	}

	fields := validateCreateComment(req)
	if len(fields) > 0 {
		response.Validation(c, fields)
		return
	}

	comment, err := h.comments.Create(c.Request.Context(), actor, service.CreateCommentInput{
		EntityType: req.EntityType,
		EntityID:   req.EntityID,
		Content:    req.Content,
	})
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	response.Created(c, dto.NewCommentResponse(comment))
}

// Update melayani `PATCH /comments/:id`.
//
// Yang menjaganya `comment:read`, bukan izin `comment:update` yang tidak ada di
// matriks; sebelum mengubah, kueri memastikan komentarnya milik aktor dan
// sebaliknya dijawab `404` — "bukan milik Anda" tidak dibedakan dari "tidak ada"
// (`44-SECURITY.md` §3.1.3).
func (h *CommentHandler) Update(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	commentID, ok := commentIDParam(c)
	if !ok {
		return
	}

	var req dto.UpdateCommentRequest
	if !bindJSON(c, &req) {
		return
	}

	comment, err := h.comments.Update(c.Request.Context(), actor, commentID, service.UpdateCommentInput{
		Content: req.Content,
	})
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	response.OK(c, dto.NewCommentResponse(comment))
}

// Delete melayani `DELETE /comments/:id`.
//
// Barisnya benar-benar dihapus (komentar bukan entitas terarsip, berbeda dari
// dokumen ADR-0019), dan responsnya `200` dengan `data: null` seperti
// `DELETE /documents/:id` — bukan `204`, supaya bentuk amplop response tetap
// sama di seluruh API (`42-API.md` §2.4).
func (h *CommentHandler) Delete(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	commentID, ok := commentIDParam(c)
	if !ok {
		return
	}

	if err := h.comments.Delete(c.Request.Context(), actor, commentID); err != nil {
		h.writeServiceError(c, err)
		return
	}

	response.OK(c, nil)
}

// writeServiceError memetakan error domain modul komentar ke response
// (`42-API.md` §7/§12).
func (h *CommentHandler) writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCommentNotFound):
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "comment not found")

	case errors.Is(err, service.ErrCommentEntityNotFound):
		// Entitas yang dikomentari tidak ada **atau** di luar cakupan project
		// aktor; keduanya dibalas sama supaya keberadaannya tidak bocor.
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "entity not found")

	case errors.Is(err, service.ErrCommentEntityTypeInvalid):
		response.Validation(c, []response.FieldError{{
			Field: "entity_type",
			Error: "harus salah satu dari " + strings.Join(model.CommentEntityTypes(), ", "),
		}})

	case errors.Is(err, service.ErrCommentContentRequired):
		response.Validation(c, []response.FieldError{{
			Field: "content",
			Error: "wajib diisi",
		}})

	case errors.Is(err, service.ErrCommentContentTooLong):
		response.Validation(c, []response.FieldError{{
			Field: "content",
			Error: "maksimal 2000 karakter",
		}})

	case errors.Is(err, service.ErrCommentNoUpdateFields):
		response.Validation(c, []response.FieldError{{
			Field: "body",
			Error: "tidak ada field yang dapat diperbarui",
		}})

	default:
		h.logger.Error("permintaan komentar gagal",
			"error", err.Error(),
			"path", c.FullPath(),
			"correlation_id", middleware.CurrentCorrelationID(c),
		)
		response.Internal(c)
	}
}

// commentIDParam membaca `:id` dan memetakan UUID tidak sah ke
// `422 VALIDATION_ERROR` (`42-API.md` §12), sama seperti modul lain.
func commentIDParam(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Validation(c, []response.FieldError{{Field: "id", Error: "harus UUID yang sah"}})
		return uuid.Nil, false
	}
	return id, true
}

// parseCommentListQuery membaca dan memvalidasi kueri daftar komentar
// (`42-API.md` §7: `?entity_type=&entity_id=&page=&limit=`).
//
// `entity_type` dan `entity_id` **wajib**: daftar komentar selalu daftar
// komentar satu entitas. Membiarkannya kosong berarti "seluruh komentar dalam
// cakupan" — kueri lintas project yang tidak punya layar di `51-UX.md` dan tidak
// perlu ada, sementara membiarkannya terbuka berarti satu endpoint lagi yang
// harus diaudit cakupannya.
func parseCommentListQuery(c *gin.Context) (string, uuid.UUID, int, int, []response.FieldError) {
	var fields []response.FieldError

	entityType := model.NormalizeCommentEntityType(c.Query("entity_type"))
	switch {
	case entityType == "":
		fields = append(fields, response.FieldError{Field: "entity_type", Error: "wajib diisi"})
	case !model.IsCommentEntityType(entityType):
		fields = append(fields, response.FieldError{
			Field: "entity_type",
			Error: "harus salah satu dari " + strings.Join(model.CommentEntityTypes(), ", "),
		})
	}

	var entityID uuid.UUID
	switch raw := strings.TrimSpace(c.Query("entity_id")); {
	case raw == "":
		fields = append(fields, response.FieldError{Field: "entity_id", Error: "wajib diisi"})
	default:
		parsed, err := uuid.Parse(raw)
		if err != nil {
			fields = append(fields, response.FieldError{Field: "entity_id", Error: "harus UUID yang sah"})
		} else {
			entityID = parsed
		}
	}

	page, err := parsePositiveInt(c.Query("page"), 1)
	if err != nil {
		fields = append(fields, response.FieldError{Field: "page", Error: "harus angka >= 1"})
	}

	limit, err := parsePositiveInt(c.Query("limit"), defaultPageLimit)
	if err != nil || limit > maxPageLimit {
		fields = append(fields, response.FieldError{Field: "limit", Error: "harus angka 1 sampai 100"})
	}

	if len(fields) > 0 {
		return "", uuid.Nil, 0, 0, fields
	}

	return entityType, entityID, page, limit, nil
}

// validateCreateComment memeriksa **kehadiran** entitas yang dikomentari.
//
// Aturan isi komentar (wajib, maksimal 2000 karakter) sengaja **tidak** diperiksa
// di sini melainkan di service (`validateCommentContent`): aturan yang sama
// berlaku untuk `POST` dan `PATCH`, dan satu-satunya cara keduanya tidak dapat
// berbeda diam-diam adalah memilikinya satu tempat. Handler tetap memetakan
// hasilnya ke `422` dengan field `content`.
func validateCreateComment(req dto.CreateCommentRequest) []response.FieldError {
	var fields []response.FieldError

	if model.NormalizeCommentEntityType(req.EntityType) == "" {
		fields = append(fields, response.FieldError{Field: "entity_type", Error: "wajib diisi"})
	}

	if req.EntityID == uuid.Nil {
		fields = append(fields, response.FieldError{Field: "entity_id", Error: "wajib diisi"})
	}

	return fields
}
