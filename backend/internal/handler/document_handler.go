package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bwdcs/backend/internal/dto"
	"bwdcs/backend/internal/middleware"
	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/pkg/response"
	"bwdcs/backend/internal/service"
)

// Batas panjang input dokumen (`41-DATABASE.md` §2.3, `44-SECURITY.md` §4.1).
const (
	maxDocumentTitleLength       = 255
	maxDocumentDescriptionLength = 5000
	maxDocumentSearchLength      = 255
	maxDocumentRevisionNote      = 2000
)

// magicByteWindow adalah jumlah byte yang dibaca untuk mendeteksi tipe berkas
// berdasarkan isinya, bukan berdasarkan header dari klien (`44-SECURITY.md` §4.2).
const magicByteWindow = 512

// DocumentHandler melayani bab Documents `42-API.md` §4.
//
// Handler hanya: parse & validasi input, memanggil service, lalu memetakan
// error domain ke status HTTP. Handler tidak menulis audit log dan tidak
// menyentuh transaksi — keduanya milik service (ADR-0011).
type DocumentHandler struct {
	documents *service.DocumentService
	logger    *slog.Logger
}

// NewDocumentHandler merakit handler dokumen.
func NewDocumentHandler(documents *service.DocumentService, logger *slog.Logger) *DocumentHandler {
	return &DocumentHandler{documents: documents, logger: logger}
}

// List melayani `GET /documents`.
//
// Cakupan data diterapkan di kueri (`44-SECURITY.md` §3.1.3): user non-admin
// hanya menerima dokumen pada project tempat ia menjadi anggota.
func (h *DocumentHandler) List(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	filter, fields := parseDocumentListQuery(c)
	if len(fields) > 0 {
		response.Validation(c, fields)
		return
	}

	documents, total, err := h.documents.List(c.Request.Context(), actor, filter)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	totalPage := 0
	if total > 0 {
		totalPage = (total + filter.Limit - 1) / filter.Limit
	}

	response.OKWithMeta(c, dto.NewDocumentListResponse(documents), response.Meta{
		Page:      filter.Page,
		Limit:     filter.Limit,
		Total:     total,
		TotalPage: totalPage,
	})
}

// Get melayani `GET /documents/:id` (dokumen + versi berjalan).
func (h *DocumentHandler) Get(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	documentID, ok := documentIDParam(c)
	if !ok {
		return
	}

	detail, err := h.documents.Get(c.Request.Context(), actor, documentID)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	h.writeDetail(c, detail, http.StatusOK)
}

// Create melayani `POST /documents` (201).
//
// Izin `document:create` diperiksa middleware (matriks `44-SECURITY.md` §3.1.2:
// Administrator, Manager, Contributor). Nomor dokumen **tidak** dibaca dari
// body: kiriman yang memuatnya ditolak di sini sebagai `422 VALIDATION_ERROR`
// (ADR-0017 butir 4).
func (h *DocumentHandler) Create(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	var req dto.CreateDocumentRequest
	if !bindJSON(c, &req) {
		return
	}

	fields := validateCreateDocument(req)
	if len(fields) > 0 {
		response.Validation(c, fields)
		return
	}

	detail, err := h.documents.Create(c.Request.Context(), actor, service.CreateDocumentInput{
		ProjectID:   req.ProjectID,
		Title:       strings.TrimSpace(req.Title),
		CategoryID:  req.CategoryID,
		Description: strings.TrimSpace(req.Description),
	})
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	h.writeDetail(c, detail, http.StatusCreated)
}

// Upload melayani `POST /documents/:id/upload` (201, multipart/form-data).
//
// Tipe berkas ditentukan dari **isi** berkas (magic bytes), bukan dari
// `Content-Type` kiriman klien; ekstensi nama berkas diperiksa sebagai penjaga
// kedua (`44-SECURITY.md` §4.2). Ukuran sebenarnya ditegakkan service saat
// berkas mengalir, sehingga header multipart yang berbohong tidak lolos.
func (h *DocumentHandler) Upload(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	documentID, ok := documentIDParam(c)
	if !ok {
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Validation(c, []response.FieldError{{Field: "file", Error: "wajib diisi (multipart/form-data)"}})
		return
	}

	revisionNote := strings.TrimSpace(c.PostForm("revision_note"))
	if utf8.RuneCountInString(revisionNote) > maxDocumentRevisionNote {
		response.Validation(c, []response.FieldError{{Field: "revision_note", Error: "maksimal 2000 karakter"}})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		h.logger.Error("buka berkas unggahan gagal", "error", err.Error(),
			"correlation_id", middleware.CurrentCorrelationID(c))
		response.Internal(c)
		return
	}
	defer func() { _ = file.Close() }()

	detected, err := detectUploadedMimeType(file)
	if err != nil {
		h.logger.Error("baca awal berkas unggahan gagal", "error", err.Error(),
			"correlation_id", middleware.CurrentCorrelationID(c))
		response.Internal(c)
		return
	}

	version, err := h.documents.UploadVersion(c.Request.Context(), actor, documentID, service.UploadVersionInput{
		OriginalName: fileHeader.Filename,
		MimeType:     detected,
		Size:         fileHeader.Size,
		RevisionNote: revisionNote,
		Content:      file,
	})
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	response.Created(c, dto.NewDocumentVersionResponse(version))
}

// Versions melayani `GET /documents/:id/versions` (terbaru lebih dulu).
func (h *DocumentHandler) Versions(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	documentID, ok := documentIDParam(c)
	if !ok {
		return
	}

	versions, err := h.documents.Versions(c.Request.Context(), actor, documentID)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	response.OK(c, map[string]any{"versions": dto.NewDocumentVersionListResponse(versions)})
}

// Download melayani `GET /documents/:id/download/:versionId`: aliran berkas
// dengan `Content-Disposition` (`42-API.md` §4).
func (h *DocumentHandler) Download(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	documentID, ok := documentIDParam(c)
	if !ok {
		return
	}

	versionID, err := uuid.Parse(c.Param("versionId"))
	if err != nil {
		response.Validation(c, []response.FieldError{{Field: "versionId", Error: "harus UUID yang sah"}})
		return
	}

	download, err := h.documents.Download(c.Request.Context(), actor, documentID, versionID)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	defer func() { _ = download.Content.Close() }()

	c.DataFromReader(http.StatusOK, download.Size, download.Version.MimeType, download.Content,
		map[string]string{"Content-Disposition": contentDisposition(download.Version.OriginalName)})
}

// Archive melayani `POST /documents/:id/archive` (200).
//
// Izin `document:update` (Administrator, Manager, Contributor — matriks
// `44-SECURITY.md` §3.1.2), **bukan** `document:delete`: arsip adalah perubahan
// keadaan, dan baris `document:delete` disediakan untuk penghapusan permanen
// yang belum ada di MVP (ADR-0019 butir 4).
//
// Response memuat dokumen yang sudah terarsip, sehingga klien tidak perlu
// memanggil `GET` lagi untuk melihat `status` dan `archived_at`-nya.
func (h *DocumentHandler) Archive(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	documentID, ok := documentIDParam(c)
	if !ok {
		return
	}

	detail, err := h.documents.Archive(c.Request.Context(), actor, documentID)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	h.writeDetail(c, detail, http.StatusOK)
}

// writeDetail mengirim bentuk `GET /documents/:id` dengan status yang diminta.
func (h *DocumentHandler) writeDetail(c *gin.Context, detail *service.DocumentDetail, status int) {
	out := dto.DocumentDetailResponse{Document: dto.NewDocumentResponse(detail.Document)}
	if detail.CurrentVersion != nil {
		version := dto.NewDocumentVersionResponse(detail.CurrentVersion)
		out.CurrentVersion = &version
	}

	if status == http.StatusCreated {
		response.Created(c, out)
		return
	}
	response.OK(c, out)
}

// writeServiceError memetakan error domain modul dokumen ke response
// (`42-API.md` §4/§12).
func (h *DocumentHandler) writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrDocumentNotFound):
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "document not found")

	case errors.Is(err, service.ErrProjectNotFound):
		// Dokumen hanya boleh lahir di project yang ada **dan** di dalam cakupan
		// aktor; keduanya dibalas sama supaya keberadaan project tidak bocor.
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "project not found")

	case errors.Is(err, service.ErrDocumentVersionNotFound):
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "document version not found")

	case errors.Is(err, service.ErrDocumentWorkflowRunning):
		response.Fail(c, http.StatusConflict, response.CodeConflict,
			"dokumen masih memiliki workflow yang berjalan")

	case errors.Is(err, service.ErrDocumentAlreadyArchived):
		// Sengaja bukan `200` idempoten: `archived_at` mencatat **kapan** arsip
		// terjadi, dan permintaan kedua tidak boleh menggesernya (ADR-0019).
		response.Fail(c, http.StatusConflict, response.CodeConflict,
			"dokumen sudah diarsipkan")

	case errors.Is(err, service.ErrDocumentArchived):
		response.Fail(c, http.StatusConflict, response.CodeConflict,
			"dokumen terarsip tidak dapat menerima versi baru")

	case errors.Is(err, service.ErrDocumentCategoryInvalid):
		response.Validation(c, []response.FieldError{{
			Field: "category_id",
			Error: "kategori tidak ditemukan di organisasi ini",
		}})

	case errors.Is(err, service.ErrDocumentFileType):
		response.Validation(c, []response.FieldError{{
			Field: "file",
			Error: "tipe berkas tidak didukung; diterima: " + strings.Join(service.AllowedDocumentExtensions(), ", "),
		}})

	case errors.Is(err, service.ErrDocumentFileTooLarge):
		response.Validation(c, []response.FieldError{{
			Field: "file",
			Error: "ukuran berkas maksimal 100 MB",
		}})

	case errors.Is(err, service.ErrDocumentFileMissing):
		h.logger.Error("berkas dokumen hilang dari penyimpanan",
			"error", err.Error(),
			"path", c.FullPath(),
			"correlation_id", middleware.CurrentCorrelationID(c),
		)
		response.Internal(c)

	default:
		h.logger.Error("permintaan dokumen gagal",
			"error", err.Error(),
			"path", c.FullPath(),
			"correlation_id", middleware.CurrentCorrelationID(c),
		)
		response.Internal(c)
	}
}

// documentIDParam membaca `:id` dan memetakan UUID tidak sah ke
// `422 VALIDATION_ERROR`.
func documentIDParam(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Validation(c, []response.FieldError{{Field: "id", Error: "harus UUID yang sah"}})
		return uuid.Nil, false
	}
	return id, true
}

// parseDocumentListQuery membaca dan memvalidasi query daftar dokumen
// (`42-API.md` §4:
// `?project_id=&status=&search=&updated_from=&updated_to=&page=&limit=`).
func parseDocumentListQuery(c *gin.Context) (service.DocumentListFilter, []response.FieldError) {
	var fields []response.FieldError

	page, err := parsePositiveInt(c.Query("page"), 1)
	if err != nil {
		fields = append(fields, response.FieldError{Field: "page", Error: "harus angka >= 1"})
	}

	limit, err := parsePositiveInt(c.Query("limit"), defaultPageLimit)
	if err != nil || limit > maxPageLimit {
		fields = append(fields, response.FieldError{Field: "limit", Error: "harus angka 1 sampai 100"})
	}

	var projectID *uuid.UUID
	if raw := strings.TrimSpace(c.Query("project_id")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			fields = append(fields, response.FieldError{Field: "project_id", Error: "harus UUID yang sah"})
		} else {
			projectID = &parsed
		}
	}

	status := strings.TrimSpace(c.Query("status"))
	if status != "" && !model.IsDocumentStatus(status) {
		fields = append(fields, response.FieldError{
			Field: "status",
			Error: "harus salah satu dari " + strings.Join(model.DocumentStatuses(), ", "),
		})
	}

	search := strings.TrimSpace(c.Query("search"))
	if utf8.RuneCountInString(search) > maxDocumentSearchLength {
		fields = append(fields, response.FieldError{Field: "search", Error: "maksimal 255 karakter"})
	}

	// Rentang `updated_at` memakai interval **tertutup** `[updated_from,
	// updated_to]`: kedua batas inklusif, sama semantiknya dengan
	// `due_from`/`due_to` pada task (`42-API.md` §6, keputusan user P-028).
	// Batasnya waktu RFC 3339 dengan offset eksplisit — tidak ada tanggal
	// tanpa zona waktu yang harus ditebak server. Rentang terbalik ditolak
	// `422` di field `updated_to`, bukan dikembalikan kosong diam-diam;
	// `updated_to == updated_from` sah dan berarti satu instan.
	var updatedFrom, updatedTo *time.Time
	if raw := strings.TrimSpace(c.Query("updated_from")); raw != "" {
		parsed, err := parseRFC3339Query(raw)
		if err != nil {
			fields = append(fields, response.FieldError{Field: "updated_from", Error: "harus waktu RFC 3339 yang sah"})
		} else {
			updatedFrom = &parsed
		}
	}
	if raw := strings.TrimSpace(c.Query("updated_to")); raw != "" {
		parsed, err := parseRFC3339Query(raw)
		if err != nil {
			fields = append(fields, response.FieldError{Field: "updated_to", Error: "harus waktu RFC 3339 yang sah"})
		} else {
			updatedTo = &parsed
		}
	}
	if updatedFrom != nil && updatedTo != nil && updatedTo.Before(*updatedFrom) {
		fields = append(fields, response.FieldError{
			Field: "updated_to",
			Error: "harus lebih besar atau sama dengan updated_from (kedua batas inklusif)",
		})
	}

	var categoryID *uuid.UUID
	if raw := strings.TrimSpace(c.Query("category_id")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			fields = append(fields, response.FieldError{Field: "category_id", Error: "harus UUID yang sah"})
		} else {
			categoryID = &parsed
		}
	}

	if len(fields) > 0 {
		return service.DocumentListFilter{}, fields
	}

	return service.DocumentListFilter{
		ProjectID:   projectID,
		Status:      status,
		Search:      search,
		UpdatedFrom: updatedFrom,
		UpdatedTo:   updatedTo,
		CategoryID:  categoryID,
		Page:        page,
		Limit:       limit,
	}, nil
}

// validateCreateDocument memeriksa body `POST /documents` (`44-SECURITY.md` §4.1).
func validateCreateDocument(req dto.CreateDocumentRequest) []response.FieldError {
	var fields []response.FieldError

	if req.ProjectID == uuid.Nil {
		fields = append(fields, response.FieldError{Field: "project_id", Error: "wajib diisi"})
	}

	title := strings.TrimSpace(req.Title)
	switch {
	case title == "":
		fields = append(fields, response.FieldError{Field: "title", Error: "wajib diisi"})
	case utf8.RuneCountInString(title) > maxDocumentTitleLength:
		fields = append(fields, response.FieldError{Field: "title", Error: "maksimal 255 karakter"})
	}

	if utf8.RuneCountInString(req.Description) > maxDocumentDescriptionLength {
		fields = append(fields, response.FieldError{Field: "description", Error: "maksimal 5000 karakter"})
	}

	if req.CategoryID != nil && *req.CategoryID == uuid.Nil {
		fields = append(fields, response.FieldError{Field: "category_id", Error: "tidak boleh UUID kosong"})
	}

	// ADR-0017 butir 4: nomor dokumen tidak diterima dari klien. Ditolak sebagai
	// validasi, bukan diabaikan diam-diam — klien yang mengandalkan nomornya
	// harus tahu bahwa nomor yang dipakai server adalah nomor lain.
	if req.DocumentNumber != nil {
		fields = append(fields, response.FieldError{
			Field: "document_number",
			Error: "dibangkitkan server (format {PROJECT_CODE}-{NNN}) dan tidak boleh dikirim",
		})
	}

	return fields
}

// detectUploadedMimeType membaca awal berkas untuk mengenali tipenya dari isi,
// lalu mengembalikan pembaca ke posisi semula supaya isi berkas utuh.
func detectUploadedMimeType(file multipartSeeker) (string, error) {
	header := make([]byte, magicByteWindow)
	read, err := file.Read(header)
	if err != nil && read == 0 {
		return "", err
	}

	detected := http.DetectContentType(header[:read])
	if _, err := file.Seek(0, 0); err != nil {
		return "", err
	}
	return detected, nil
}

// multipartSeeker adalah bagian berkas multipart yang dibutuhkan pemeriksaan
// magic bytes: baca lalu kembalikan posisinya.
type multipartSeeker interface {
	Read(p []byte) (int, error)
	Seek(offset int64, whence int) (int64, error)
}

// contentDisposition menyusun header unduhan.
//
// Nama berkas asli dapat memuat karakter non-ASCII, sehingga bentuk RFC 5987
// (`filename*=UTF-8”…`) dikirim bersama cadangan ASCII: klien lama tetap
// mendapat nama, klien baru mendapat nama yang benar.
func contentDisposition(originalName string) string {
	name := filepath.Base(strings.TrimSpace(originalName))
	if name == "" || name == "." {
		name = "document"
	}

	ascii := strings.Map(func(r rune) rune {
		if r < 32 || r > 126 || r == '"' || r == '\\' {
			return '_'
		}
		return r
	}, name)

	return `attachment; filename="` + ascii + `"; filename*=UTF-8''` + url.PathEscape(name)
}

// ListCategories melayani `GET /documents/categories`.
//
// Mengembalikan seluruh kategori dokumen pada organisasi aktor, diurutkan
// berdasarkan nama. Dipakai frontend untuk mengisi dropdown penyaring kategori.
// Tidak ada pagination: jumlah kategori per organisasi kecil (< 50).
func (h *DocumentHandler) ListCategories(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	categories, err := h.documents.ListCategories(c.Request.Context(), actor)
	if err != nil {
		h.logger.Error("daftar kategori gagal", "error", err.Error())
		response.Internal(c)
		return
	}

	type categoryItem struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Code string `json:"code"`
	}
	items := make([]categoryItem, len(categories))
	for i, cat := range categories {
		items[i] = categoryItem{
			ID:   cat.ID.String(),
			Name: cat.Name,
			Code: cat.Code,
		}
	}
	response.OK(c, items)
}
