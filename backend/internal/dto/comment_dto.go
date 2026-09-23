package dto

import (
	"time"

	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
)

// CreateCommentRequest adalah badan `POST /comments` (`42-API.md` §7).
//
// Entitas yang dikomentari disebut di badan permintaan, bukan di path —
// bentuknya dipertahankan dari kontrak §7 dan sama dengan yang dikirim klien
// halaman dokumen/task (`50-FSD.md` §7).
//
// `Content` sengaja `string`, bukan `*string`, dan validatornya hidup di service
// (`validateCommentContent`): aturan isi komentar hanya boleh ada di satu tempat
// supaya `POST` dan `PATCH` tidak berbeda diam-diam.
type CreateCommentRequest struct {
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	Content    string    `json:"content"`
}

// UpdateCommentRequest adalah badan `PATCH /comments/:id`. `Content` adalah
// pointer supaya "tidak dikirim" dapat dibedakan dari string kosong.
type UpdateCommentRequest struct {
	Content *string `json:"content"`
}

// CommentResponse adalah representasi komentar pada API.
//
// Nama kolomnya mengikuti `42-API.md` §2.3 (snake_case) dan memuat
// `created_by_username` karena tampilan komentar selalu menampilkan penulis
// (`50-FSD.md` §7) — tanpa kolom itu klien harus memanggil endpoint user terpisah
// untuk setiap baris.
//
// `updated_at` tidak ada, dan itu bukan kelalaian: kolomnya tidak ada di tabel
// `comments` (`41-DATABASE.md` §2.5). Yang tersedia adalah jejak di `audit_logs`.
type CommentResponse struct {
	ID                uuid.UUID `json:"id"`
	EntityID          uuid.UUID `json:"entity_id"`
	EntityType        string    `json:"entity_type"`
	Content           string    `json:"content"`
	CreatedByID       uuid.UUID `json:"created_by_id"`
	CreatedByUsername string    `json:"created_by_username,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// NewCommentResponse memetakan model komentar ke bentuk response.
func NewCommentResponse(comment *model.Comment) CommentResponse {
	if comment == nil {
		return CommentResponse{}
	}
	return CommentResponse{
		ID:                comment.ID,
		EntityID:          comment.EntityID,
		EntityType:        comment.EntityType,
		Content:           comment.Content,
		CreatedByID:       comment.CreatedByID,
		CreatedByUsername: comment.CreatedByUsername,
		CreatedAt:         comment.CreatedAt,
	}
}

// NewCommentListResponse memetakan daftar komentar ke bentuk response (selalu
// array, bukan null, supaya klien tidak perlu menangani dua bentuk kosong).
//
// Metadata paginasi tidak di sini: ia dikirim pada `meta` amplop response
// (`42-API.md` §2.4), sama seperti modul project, document, dan task.
func NewCommentListResponse(comments []model.Comment) []CommentResponse {
	out := make([]CommentResponse, 0, len(comments))
	for i := range comments {
		out = append(out, NewCommentResponse(&comments[i]))
	}
	return out
}
