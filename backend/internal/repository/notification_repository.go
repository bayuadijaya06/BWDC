package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
)

// NotificationRepository membaca dan menandai notifikasi milik user (`42-API.md` §8).
type NotificationRepository struct {
	db DBTX
}

func NewNotificationRepository(db DBTX) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// List mengembalikan notifikasi milik user, terbaru dulu, dengan filter is_read opsional.
//
// isRead nil = tanpa filter; true/false = hanya baris dengan is_read tersebut.
func (r *NotificationRepository) List(ctx context.Context, userID uuid.UUID, isRead *bool, page, limit int) ([]model.Notification, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	query := `
		SELECT id, user_id, type, title, message, entity_id, entity_type, is_read, created_at,
			COUNT(*) OVER() AS total
		FROM notifications
		WHERE user_id = $1 AND ($2::boolean IS NULL OR is_read = $2)
		ORDER BY created_at DESC, id DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.Query(ctx, query, userID, isRead, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("baca daftar notifikasi: %w", err)
	}
	defer rows.Close()

	notifications := make([]model.Notification, 0, limit)
	total := 0
	for rows.Next() {
		var n model.Notification
		var totalCount int
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message, &n.EntityID, &n.EntityType, &n.IsRead, &n.CreatedAt, &totalCount); err != nil {
			return nil, 0, fmt.Errorf("scan notifikasi: %w", err)
		}
		total = totalCount
		notifications = append(notifications, n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterasi notifikasi: %w", err)
	}
	if len(notifications) == 0 && offset > 0 {
		counted, err := r.count(ctx, userID, isRead)
		if err != nil {
			return nil, 0, err
		}
		total = counted
	}
	return notifications, total, nil
}

func (r *NotificationRepository) count(ctx context.Context, userID uuid.UUID, isRead *bool) (int, error) {
	query := `SELECT count(*) FROM notifications WHERE user_id = $1 AND ($2::boolean IS NULL OR is_read = $2)`
	var total int
	if err := r.db.QueryRow(ctx, query, userID, isRead).Scan(&total); err != nil {
		return 0, fmt.Errorf("hitung notifikasi: %w", err)
	}
	return total, nil
}

// MarkRead menandai satu notifikasi milik user sebagai dibaca.
func (r *NotificationRepository) MarkRead(ctx context.Context, userID, notificationID uuid.UUID) (int64, error) {
	tag, err := r.db.Exec(ctx, `UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2 AND is_read = false`, notificationID, userID)
	if err != nil {
		return 0, fmt.Errorf("tandai notifikasi dibaca: %w", err)
	}
	return tag.RowsAffected(), nil
}

// MarkAllRead menandai seluruh notifikasi milik user yang belum dibaca.
func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	tag, err := r.db.Exec(ctx, `UPDATE notifications SET is_read = true WHERE user_id = $1 AND is_read = false`, userID)
	if err != nil {
		return 0, fmt.Errorf("tandai semua notifikasi dibaca: %w", err)
	}
	return tag.RowsAffected(), nil
}
