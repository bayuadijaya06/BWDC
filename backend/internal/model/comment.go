package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Jenis entitas yang dapat dikomentari (`50-FSD.md` §7) — nilainya adalah
// vocabular `comments.entity_type` apa adanya, termasuk `CHECK`-nya
// (`41-DATABASE.md` §2.5, migrasi `007`). Sumbernya satu: kolom database,
// bukan daftar karangan di Go.
//
// `workflow` berarti **workflow instance**, dan `entity_id`-nya adalah
// `workflow_instances.id` — bukan `workflow_definitions.id`. Di API, dokumen
// menyebutnya "workflow instance" (`50-FSD.md` §7); nilainya sengaja tetap
// mengikuti kolom supaya tidak lahir vocabular kedua yang harus dipetakan.
const (
	CommentEntityProject  = "project"
	CommentEntityDocument = "document"
	CommentEntityTask     = "task"
	CommentEntityWorkflow = "workflow"
)

// CommentContentMaxLength adalah batas panjang isi komentar (`50-FSD.md` §7:
// "Content (required, max 2000 chars)").
//
// Batas ini ditegakkan di handler (422), bukan di kolom `TEXT`: `41-DATABASE.md`
// §2.5 memakai `TEXT` dengan alasan isi komentar tidak dibatasi database, dan
// batas produk boleh berubah tanpa migrasi.
const CommentContentMaxLength = 2000

// CommentEntityTypes mengembalikan himpunan tertutup jenis entitas komentar,
// dipakai pesan validasi supaya klien tahu nilai mana yang sah.
func CommentEntityTypes() []string {
	return []string{CommentEntityProject, CommentEntityDocument, CommentEntityTask, CommentEntityWorkflow}
}

// NormalizeCommentEntityType menormalkan `entity_type` sebelum diperiksa:
// huruf kecil tanpa spasi tepi, sehingga `"Document"` dan `" document "` sah
// dan bernilai sama dengan `document`.
//
// Hanya **satu** hal yang dinormalkan — besar-kecil huruf. Tidak ada alias:
// `workflow_instance` bukan nilai yang diterima, karena menerima dua nama untuk
// satu kolom berarti setiap pemakaian berikutnya harus menebak mana yang
// kanonik.
func NormalizeCommentEntityType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// IsCommentEntityType melaporkan apakah nilai sudah berupa salah satu jenis
// entitas yang sah (pemanggil menormalkan lebih dulu).
func IsCommentEntityType(value string) bool {
	switch value {
	case CommentEntityProject, CommentEntityDocument, CommentEntityTask, CommentEntityWorkflow:
		return true
	default:
		return false
	}
}

// Comment adalah komentar pada sebuah entitas.
//
// Kolom `comments` (`41-DATABASE.md` §2.5) tidak punya `updated_at`, dan itu
// memang disengaja: tabel ini diciptakan sebagai **jejak diskusi**, dan isi
// komentar yang disunting tidak menyimpan riwayat versinya. Perubahan isi tetap
// tercatat di `audit_logs` (aksi `COMMENT_UPDATED`) — jejaknya ada, tetapi tidak
// di tabel ini.
type Comment struct {
	ID          uuid.UUID
	EntityID    uuid.UUID
	EntityType  string
	Content     string
	CreatedByID uuid.UUID
	CreatedAt   time.Time

	// CreatedByUsername adalah kolom turunan untuk tampilan "Author name"
	// (`50-FSD.md` §7); bukan kolom tabel.
	CreatedByUsername string
}
