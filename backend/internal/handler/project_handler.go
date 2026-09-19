package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
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

// Batas panjang input project (`41-DATABASE.md` §2.2, `50-FSD.md` §3.2).
const (
	maxProjectCodeLength        = 50
	maxProjectNameLength        = 255
	maxProjectDescriptionLength = 2000
	maxProjectSearchLength      = 255
)

// Pagination daftar project (`42-API.md` §1/§3).
const (
	defaultPageLimit = 20
	maxPageLimit     = 100
)

// ProjectHandler melayani bab Projects `42-API.md` §3.
//
// Handler hanya: parse & validasi input, memanggil service, lalu memetakan
// error domain ke status HTTP. Handler tidak menulis audit log dan tidak
// menyentuh transaksi — keduanya milik service (ADR-0011).
type ProjectHandler struct {
	projects *service.ProjectService
	logger   *slog.Logger
}

// NewProjectHandler merakit handler project.
func NewProjectHandler(projects *service.ProjectService, logger *slog.Logger) *ProjectHandler {
	return &ProjectHandler{projects: projects, logger: logger}
}

// List melayani `GET /projects`.
//
// Cakupan data diterapkan di kueri (`44-SECURITY.md` §3.1.3): user non-admin
// hanya menerima project tempat ia menjadi anggota.
func (h *ProjectHandler) List(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	filter, fields := parseProjectListQuery(c)
	if len(fields) > 0 {
		response.Validation(c, fields)
		return
	}

	projects, total, err := h.projects.List(c.Request.Context(), actor, filter)
	if err != nil {
		h.writeServiceError(c, err, "")
		return
	}

	totalPage := 0
	if total > 0 {
		totalPage = (total + filter.Limit - 1) / filter.Limit
	}

	response.OKWithMeta(c, dto.NewProjectListResponse(projects), response.Meta{
		Page:      filter.Page,
		Limit:     filter.Limit,
		Total:     total,
		TotalPage: totalPage,
	})
}

// Get melayani `GET /projects/:id` (detail + anggota).
func (h *ProjectHandler) Get(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	projectID, ok := projectIDParam(c)
	if !ok {
		return
	}

	detail, err := h.projects.Get(c.Request.Context(), actor, projectID)
	if err != nil {
		h.writeServiceError(c, err, "")
		return
	}

	response.OK(c, dto.ProjectDetailResponse{
		Project: dto.NewProjectResponse(detail.Project),
		Members: dto.NewProjectMemberListResponse(detail.Members),
	})
}

// Create melayani `POST /projects` (201).
//
// Izin `project:create` diperiksa middleware (matriks `44-SECURITY.md` §3.1:
// Administrator dan Manager). Aturan `code` diperiksa di sini, bukan lewat tag
// validator: polanya perlu regex eksplisit (`42-API.md` §3, ADR-0017).
func (h *ProjectHandler) Create(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	var req dto.CreateProjectRequest
	if !bindJSON(c, &req) {
		return
	}

	fields := validateCreateProject(req)
	if len(fields) > 0 {
		response.Validation(c, fields)
		return
	}

	detail, err := h.projects.Create(c.Request.Context(), actor, service.CreateProjectInput{
		Code:          dto.NormalizeProjectCode(req.Code),
		Name:          strings.TrimSpace(req.Name),
		Description:   strings.TrimSpace(req.Description),
		OwnerID:       req.OwnerID,
		StartDate:     dateOrNil(req.StartDate),
		TargetEndDate: dateOrNil(req.TargetEndDate),
	})
	if err != nil {
		h.writeServiceError(c, err, "owner_id")
		return
	}

	response.Created(c, dto.ProjectDetailResponse{
		Project: dto.NewProjectResponse(detail.Project),
		Members: dto.NewProjectMemberListResponse(detail.Members),
	})
}

// Update melayani `PATCH /projects/:id`.
//
// `code` tidak dapat diubah (ADR-0017): mengirimkannya ditolak `409 CONFLICT`
// oleh service sebelum satu pun kolom tersentuh.
func (h *ProjectHandler) Update(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	projectID, ok := projectIDParam(c)
	if !ok {
		return
	}

	var req dto.UpdateProjectRequest
	if !bindJSON(c, &req) {
		return
	}

	fields := validateUpdateProject(req)
	if len(fields) > 0 {
		response.Validation(c, fields)
		return
	}

	detail, err := h.projects.Update(c.Request.Context(), actor, projectID, service.UpdateProjectInput{
		Code:          req.Code,
		Name:          trimmedOrNil(req.Name),
		Description:   trimmedOrNil(req.Description),
		OwnerID:       req.OwnerID,
		StartDate:     dateOrNil(req.StartDate),
		TargetEndDate: dateOrNil(req.TargetEndDate),
	})
	if err != nil {
		h.writeServiceError(c, err, "owner_id")
		return
	}

	response.OK(c, dto.ProjectDetailResponse{
		Project: dto.NewProjectResponse(detail.Project),
		Members: dto.NewProjectMemberListResponse(detail.Members),
	})
}

// Archive melayani `POST /projects/:id/archive` (FR-PROJ-07: diarsipkan, bukan
// dihapus). Idempotent.
func (h *ProjectHandler) Archive(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	projectID, ok := projectIDParam(c)
	if !ok {
		return
	}

	project, err := h.projects.Archive(c.Request.Context(), actor, projectID)
	if err != nil {
		h.writeServiceError(c, err, "")
		return
	}

	response.OK(c, dto.NewProjectResponse(project))
}

// Members melayani `GET /projects/:id/members`.
func (h *ProjectHandler) Members(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	projectID, ok := projectIDParam(c)
	if !ok {
		return
	}

	members, err := h.projects.Members(c.Request.Context(), actor, projectID)
	if err != nil {
		h.writeServiceError(c, err, "")
		return
	}

	response.OK(c, map[string]any{"members": dto.NewProjectMemberListResponse(members)})
}

// AddMember melayani `POST /projects/:id/members` (201).
func (h *ProjectHandler) AddMember(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	projectID, ok := projectIDParam(c)
	if !ok {
		return
	}

	var req dto.AddProjectMemberRequest
	if !bindJSON(c, &req) {
		return
	}

	fields := validateAddMember(req)
	if len(fields) > 0 {
		response.Validation(c, fields)
		return
	}

	member, err := h.projects.AddMember(c.Request.Context(), actor, projectID, req.UserID, req.Role)
	if err != nil {
		h.writeServiceError(c, err, "user_id")
		return
	}

	response.Created(c, dto.NewProjectMemberResponse(*member))
}

// RemoveMember melayani `DELETE /projects/:id/members/:userId`.
//
// Owner tidak dapat dihapus (`409 CONFLICT`): kehilangan keanggotaan berarti
// kehilangan akses ke project-nya sendiri (cakupan §3.1.3 membaca
// `project_members`).
func (h *ProjectHandler) RemoveMember(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	projectID, ok := projectIDParam(c)
	if !ok {
		return
	}

	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.Validation(c, []response.FieldError{{Field: "userId", Error: "harus UUID yang sah"}})
		return
	}

	if err := h.projects.RemoveMember(c.Request.Context(), actor, projectID, userID); err != nil {
		h.writeServiceError(c, err, "")
		return
	}

	response.OK(c, nil)
}

// writeServiceError memetakan error domain ke response (`42-API.md` §3/§12).
//
// `field` adalah nama field input yang dipakai saat error-nya bersifat
// validasi terhadap input (mis. `owner_id` vs `user_id`); pemanggil yang tahu
// field mana yang relevan, karena service sengaja tidak sadar HTTP.
func (h *ProjectHandler) writeServiceError(c *gin.Context, err error, field string) {
	switch {
	case errors.Is(err, service.ErrProjectNotFound):
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "project not found")

	case errors.Is(err, service.ErrProjectMemberNotFound):
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "user bukan anggota project")

	case errors.Is(err, service.ErrProjectCodeTaken):
		response.Fail(c, http.StatusConflict, response.CodeConflict, "project code already exists in this organization")

	case errors.Is(err, service.ErrProjectCodeImmutable):
		response.Fail(c, http.StatusConflict, response.CodeConflict,
			"project code tidak dapat diubah setelah dibuat (menjadi prefiks nomor dokumen)")

	case errors.Is(err, service.ErrProjectOwnerRemoval):
		response.Fail(c, http.StatusConflict, response.CodeConflict,
			"owner project tidak dapat dihapus dari daftar anggota")

	case errors.Is(err, service.ErrProjectMemberExists):
		response.Fail(c, http.StatusConflict, response.CodeConflict, "user sudah menjadi anggota project")

	case errors.Is(err, service.ErrUserNotInOrganization):
		if field == "" {
			field = "user_id"
		}
		response.Validation(c, []response.FieldError{{Field: field, Error: "user tidak ditemukan di organisasi ini"}})

	case errors.Is(err, service.ErrProjectRoleInvalid):
		response.Validation(c, []response.FieldError{{
			Field: "role",
			Error: "harus salah satu dari " + strings.Join(model.ProjectRoles(), ", "),
		}})

	case errors.Is(err, service.ErrProjectDateRange):
		response.Validation(c, []response.FieldError{{
			Field: "target_end_date",
			Error: "tidak boleh lebih awal dari start_date",
		}})

	case errors.Is(err, service.ErrProjectNoUpdateFields):
		response.Validation(c, []response.FieldError{{
			Field: "body",
			Error: "tidak ada field yang dapat diperbarui",
		}})

	default:
		h.logger.Error("permintaan project gagal",
			"error", err.Error(),
			"path", c.FullPath(),
			"correlation_id", middleware.CurrentCorrelationID(c),
		)
		response.Internal(c)
	}
}

// actorFrom mengambil aktor terautentikasi. Route project selalu dipasang di
// group ber-AuthMiddleware, jadi ketiadaan aktor berarti salah pemasangan route
// — dibalas 401, bukan 500, karena penyebabnya tetap "tidak terautentikasi".
func actorFrom(c *gin.Context) (service.Actor, bool) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "autentikasi diperlukan")
		return service.Actor{}, false
	}
	return service.Actor{ID: user.ID, OrganizationID: user.OrganizationID}, true
}

// projectIDParam membaca `:id` dan memetakan UUID tidak sah ke
// `422 VALIDATION_ERROR` (`42-API.md` §12, contoh `details`).
func projectIDParam(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Validation(c, []response.FieldError{{Field: "id", Error: "harus UUID yang sah"}})
		return uuid.Nil, false
	}
	return id, true
}

// bindJSON membaca body dan memetakan kegagalan decode ke `422` beserta nama
// field yang benar-benar bermasalah (`42-API.md` §12 memakai daftar
// `{field, error}` justru untuk itu).
//
// Dipakai semua modul (project, document, task), jadi namanya tidak lagi
// menyebut satu modul: yang berbeda antar modul hanyalah validasi isinya.
//
// Tiga sebab yang **berbeda** dan tidak boleh dijawab dengan pesan yang sama
// (temuan C-045 — dulu semuanya jatuh ke "harus JSON objek yang sah" sehingga
// klien diarahkan memperbaiki hal yang tidak salah):
//
//  1. tipe data tidak sesuai → `json.UnmarshalTypeError` sudah membawa nama field;
//  2. JSON-nya sendiri tidak sah (`json.SyntaxError`) → `field = body`;
//  3. JSON sah tetapi sebuah **nilai** tidak dapat diurai — bentuk yang paling
//     sering di sini adalah UUID/tanggal, karena `json.Unmarshaler` kustom
//     (`uuid.UUID`, `time.Time`) mengembalikan error yang **tidak** membawa nama
//     field (diperiksa langsung: `uuid.invalidLengthError`, bukan
//     `UnmarshalTypeError`) → field dicari lewat `undecodableField`.
func bindJSON(c *gin.Context, target any) bool {
	// Body dibaca sekali di sini, bukan lewat `c.ShouldBindJSON`, supaya byte
	// mentahnya masih tersedia untuk membedakan sebab (1)-(3).
	raw, readErr := io.ReadAll(c.Request.Body)
	if readErr != nil {
		response.Validation(c, []response.FieldError{{Field: "body", Error: "body tidak dapat dibaca"}})
		return false
	}

	if err := json.Unmarshal(raw, target); err != nil {
		var typeErr *json.UnmarshalTypeError
		var syntaxErr *json.SyntaxError

		switch {
		case errors.As(err, &typeErr):
			field := typeErr.Field
			if field == "" {
				field = "body"
			}
			response.Validation(c, []response.FieldError{{Field: field, Error: "tipe data tidak sesuai"}})
		case errors.As(err, &syntaxErr) || !json.Valid(raw):
			response.Validation(c, []response.FieldError{{Field: "body", Error: "harus JSON objek yang sah"}})
		default:
			field, message, ok := undecodableField(raw, target)
			if !ok {
				field, message = "body", "nilai field tidak dapat diurai"
			}
			response.Validation(c, []response.FieldError{{Field: field, Error: message}})
		}
		return false
	}
	return true
}

// undecodableField menamai field yang nilainya tidak dapat diurai dari body yang
// JSON-nya sudah terbukti sah, beserta pesan yang menyebut bentuk yang benar.
//
// Caranya: setiap field target yang ada di body dicoba diurai ulang satu per
// satu; field pertama yang gagal itulah jawabannya. Pemetaan tipe → pesan
// disengaja memakai **himpunan tertutup** (`uuid.UUID`, `time.Time`) supaya
// pesannya sejalan dengan validasi parameter kueri yang sudah ada ("harus UUID
// yang sah"), bukan menyalin pesan internal `encoding/json`.
func undecodableField(raw []byte, target any) (field, message string, ok bool) {
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		return "", "", false
	}

	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return "", "", false
	}
	value = value.Elem()
	if value.Kind() != reflect.Struct {
		return "", "", false
	}

	uuidType := reflect.TypeOf(uuid.UUID{})
	timeType := reflect.TypeOf(time.Time{})

	for i := 0; i < value.NumField(); i++ {
		structField := value.Type().Field(i)
		name := strings.Split(structField.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			continue
		}
		rawValue, present := body[name]
		if !present {
			continue
		}

		holder := reflect.New(structField.Type)
		if err := json.Unmarshal(rawValue, holder.Interface()); err != nil {
			return name, messageForUndecodable(structField.Type, uuidType, timeType), true
		}
	}

	return "", "", false
}

// messageForUndecodable memilih pesan untuk field yang nilainya tidak dapat
// diurai, dari bentuk tipenya (pointer dilepas lebih dulu).
func messageForUndecodable(fieldType, uuidType, timeType reflect.Type) string {
	elem := fieldType
	if elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}

	switch {
	case elem == uuidType:
		return "harus UUID yang sah"
	case elem == timeType:
		return "harus waktu RFC 3339 yang sah"
	case elem.Kind() == reflect.Struct:
		// Tipe yang membungkus `time.Time` (mis. `dto.Date` untuk `YYYY-MM-DD`)
		// tetap disebut sebagai tanggal, bukan "nilai tidak dapat diurai".
		if embedded, ok := elem.FieldByName("Time"); ok && embedded.Type == timeType {
			return "harus tanggal yang sah"
		}
	}
	return "nilai tidak dapat diurai"
}

// parseProjectListQuery membaca dan memvalidasi query daftar project.
func parseProjectListQuery(c *gin.Context) (service.ProjectListFilter, []response.FieldError) {
	var fields []response.FieldError

	page, err := parsePositiveInt(c.Query("page"), 1)
	if err != nil {
		fields = append(fields, response.FieldError{Field: "page", Error: "harus angka >= 1"})
	}

	limit, err := parsePositiveInt(c.Query("limit"), defaultPageLimit)
	if err != nil || limit > maxPageLimit {
		fields = append(fields, response.FieldError{Field: "limit", Error: "harus angka 1 sampai 100"})
	}

	status := strings.TrimSpace(c.Query("status"))
	if status != "" && status != modelStatusActive && status != modelStatusArchived {
		fields = append(fields, response.FieldError{
			Field: "status",
			Error: "harus salah satu dari " + modelStatusActive + ", " + modelStatusArchived,
		})
	}

	search := strings.TrimSpace(c.Query("search"))
	if utf8.RuneCountInString(search) > maxProjectSearchLength {
		fields = append(fields, response.FieldError{Field: "search", Error: "maksimal 255 karakter"})
	}

	if len(fields) > 0 {
		return service.ProjectListFilter{}, fields
	}

	return service.ProjectListFilter{Status: status, Search: search, Page: page, Limit: limit}, nil
}

// validateCreateProject memeriksa body `POST /projects` (`50-FSD.md` §3.2).
func validateCreateProject(req dto.CreateProjectRequest) []response.FieldError {
	var fields []response.FieldError

	code := dto.NormalizeProjectCode(req.Code)
	switch {
	case code == "":
		fields = append(fields, response.FieldError{Field: "code", Error: "wajib diisi"})
	case utf8.RuneCountInString(code) > maxProjectCodeLength:
		fields = append(fields, response.FieldError{Field: "code", Error: "maksimal 50 karakter"})
	case !dto.ProjectCodePattern.MatchString(code):
		fields = append(fields, response.FieldError{
			Field: "code",
			Error: "hanya huruf besar, angka, dan tanda hubung (pola ^[A-Z0-9]+(-[A-Z0-9]+)*$)",
		})
	}

	name := strings.TrimSpace(req.Name)
	switch {
	case name == "":
		fields = append(fields, response.FieldError{Field: "name", Error: "wajib diisi"})
	case utf8.RuneCountInString(name) > maxProjectNameLength:
		fields = append(fields, response.FieldError{Field: "name", Error: "maksimal 255 karakter"})
	}

	if utf8.RuneCountInString(req.Description) > maxProjectDescriptionLength {
		fields = append(fields, response.FieldError{Field: "description", Error: "maksimal 2000 karakter"})
	}

	if req.OwnerID == uuid.Nil {
		fields = append(fields, response.FieldError{Field: "owner_id", Error: "wajib diisi"})
	}

	fields = append(fields, validateDateRange(req.StartDate, req.TargetEndDate)...)
	return fields
}

// validateUpdateProject memeriksa body `PATCH /projects/:id`.
//
// `code` sengaja tidak divalidasi polanya: kehadirannya sudah cukup untuk
// ditolak `409 CONFLICT` oleh service (ADR-0017).
func validateUpdateProject(req dto.UpdateProjectRequest) []response.FieldError {
	var fields []response.FieldError

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		switch {
		case name == "":
			fields = append(fields, response.FieldError{Field: "name", Error: "tidak boleh kosong"})
		case utf8.RuneCountInString(name) > maxProjectNameLength:
			fields = append(fields, response.FieldError{Field: "name", Error: "maksimal 255 karakter"})
		}
	}

	if req.Description != nil && utf8.RuneCountInString(*req.Description) > maxProjectDescriptionLength {
		fields = append(fields, response.FieldError{Field: "description", Error: "maksimal 2000 karakter"})
	}

	if req.OwnerID != nil && *req.OwnerID == uuid.Nil {
		fields = append(fields, response.FieldError{Field: "owner_id", Error: "tidak boleh UUID kosong"})
	}

	fields = append(fields, validateDateRange(req.StartDate, req.TargetEndDate)...)
	return fields
}

// validateAddMember memeriksa body `POST /projects/:id/members`.
func validateAddMember(req dto.AddProjectMemberRequest) []response.FieldError {
	var fields []response.FieldError

	if req.UserID == uuid.Nil {
		fields = append(fields, response.FieldError{Field: "user_id", Error: "wajib diisi"})
	}

	role := strings.TrimSpace(req.Role)
	if !model.IsProjectRole(role) {
		fields = append(fields, response.FieldError{
			Field: "role",
			Error: "harus salah satu dari " + strings.Join(model.ProjectRoles(), ", "),
		})
	}

	return fields
}

// validateDateRange menegakkan `50-FSD.md` §3.2: target akhir >= tanggal mulai.
func validateDateRange(start, target *dto.Date) []response.FieldError {
	if start == nil || target == nil {
		return nil
	}
	if target.Before(start.Time) {
		return []response.FieldError{{Field: "target_end_date", Error: "tidak boleh lebih awal dari start_date"}}
	}
	return nil
}

// dateOrNil mengubah DTO tanggal menjadi `*time.Time`; tanggal yang tidak
// dikirim tetap nil, sehingga service tidak mengubah kolomnya.
func dateOrNil(value *dto.Date) *time.Time {
	if value == nil {
		return nil
	}
	converted := value.Time
	return &converted
}

// trimmedOrNil memangkas spasi nilai opsional tanpa mengubah kebolehannya
// bernilai string kosong (yang berarti mengosongkan isi kolom).
func trimmedOrNil(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

// parsePositiveInt membaca angka positif dengan nilai bawaan.
func parsePositiveInt(raw string, fallback int) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || parsed < 1 {
		return fallback, errors.New("harus angka positif")
	}
	return parsed, nil
}

// modelStatusActive/Archived adalah nilai kanonik `projects.status`
// (`model.ProjectStatusActive`/`Archived`) yang dipakai pesan validasi filter
// (ADR-0012).
const (
	modelStatusActive   = model.ProjectStatusActive
	modelStatusArchived = model.ProjectStatusArchived
)
