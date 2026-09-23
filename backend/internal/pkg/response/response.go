// Package response membungkus format response API BWDCS.
//
// Kontrak struct: docs/design/40-TSD.md §3. Daftar kode error: docs/design/42-API.md §12.
//
// Aturan: setiap handler membalas lewat package ini agar bentuk JSON, nama kode,
// dan status HTTP tidak ditentukan ulang per handler. Handler tidak pernah
// menulis `gin.H{"error": ...}` sendiri.
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Kode error kanonik. Sumber: `42-API.md` §12. Menambah kode baru wajib
// menambahkannya di §12 pada perubahan yang sama.
const (
	CodeValidation   = "VALIDATION_ERROR"
	CodeUnauthorized = "UNAUTHORIZED"
	CodeTokenRevoked = "TOKEN_REVOKED"
	CodeForbidden    = "FORBIDDEN"
	CodeNotFound     = "NOT_FOUND"
	CodeConflict     = "CONFLICT"
	// CodeWorkflowConflict hanya dipakai transisi workflow instance
	// (`42-API.md` §5/§12): permintaan mungkin sah, tetapi dibuat untuk keadaan
	// instance yang sudah berubah — klien harus memuat ulang, bukan memperbaiki
	// input. Karena itu `details`-nya objek keadaan terkini, bukan daftar field.
	CodeWorkflowConflict       = "WORKFLOW_CONFLICT"
	CodeInvalidCredentials     = "INVALID_CREDENTIALS"
	CodeAccountInactive        = "ACCOUNT_INACTIVE"
	CodeTooManyRequests        = "TOO_MANY_REQUESTS"
	CodeLocked                 = "LOCKED"
	CodeInvalidCurrentPassword = "INVALID_CURRENT_PASSWORD"
	CodeInternal               = "INTERNAL_ERROR"
)

// APIResponse adalah amplop response (`40-TSD.md` §3).
type APIResponse struct {
	Success bool      `json:"success"`
	Data    any       `json:"data,omitempty"`
	Error   *APIError `json:"error,omitempty"`
	Meta    *Meta     `json:"meta,omitempty"`
}

// APIError adalah isi `error` pada response gagal.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Meta membawa informasi pagination.
type Meta struct {
	Page      int `json:"page"`
	Limit     int `json:"limit"`
	Total     int `json:"total"`
	TotalPage int `json:"total_page"`
}

// OK membalas 200 dengan data. `data` nil menghasilkan `{"success":true}`
// (dipakai endpoint yang tidak mengembalikan isi, mis. logout).
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: data})
}

// OKWithMeta membalas 200 dengan data berhalaman dan blok `meta`
// (`42-API.md` §1/§3). Endpoint daftar memakai ini supaya bentuk pagination
// tidak ditentukan ulang per handler.
func OKWithMeta(c *gin.Context, data any, meta Meta) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: data, Meta: &meta})
}

// Created membalas 201 dengan data.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, APIResponse{Success: true, Data: data})
}

// Fail membalas response gagal dengan kode dan pesan.
func Fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
	})
}

// FailWithDetails membalas response gagal dengan daftar detail terstruktur
// (dipakai 422: `details` berisi daftar {field, error}).
func FailWithDetails(c *gin.Context, status int, code, message string, details any) {
	c.JSON(status, APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Message: message, Details: details},
	})
}

// FieldError adalah satu baris `details` pada 422 (`42-API.md` §12).
type FieldError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

// Validation membalas 422 dengan daftar field bermasalah.
func Validation(c *gin.Context, fields []FieldError) {
	FailWithDetails(c, http.StatusUnprocessableEntity, CodeValidation, "validation failed", fields)
}

// Internal membalas 500 untuk kegagalan tak terduga. Pesan internal TIDAK
// diteruskan ke klien (informasi bocor); detailnya masuk log middleware.
func Internal(c *gin.Context) {
	Fail(c, http.StatusInternalServerError, CodeInternal, "terjadi kesalahan di server")
}
