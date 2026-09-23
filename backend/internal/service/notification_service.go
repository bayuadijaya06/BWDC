package service

import (
	"context"

	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/repository"
)

// NotificationService melayani `42-API.md` §8.
type NotificationService struct {
	notifications *repository.NotificationRepository
}

func NewNotificationService(notifications *repository.NotificationRepository) *NotificationService {
	return &NotificationService{notifications: notifications}
}

// List mengembalikan notifikasi milik aktor.
//
// isReadStr: "" = tanpa filter, "true"/"false" sudah divalidasi handler.
func (s *NotificationService) List(ctx context.Context, actor Actor, isRead *bool, page, limit int) ([]model.Notification, int, error) {
	return s.notifications.List(ctx, actor.ID, isRead, page, limit)
}

// MarkRead menandai satu notifikasi milik aktor. ErrNotFound bila bukan milik aktor.
func (s *NotificationService) MarkRead(ctx context.Context, actor Actor, notificationID uuid.UUID) error {
	affected, err := s.notifications.MarkRead(ctx, actor.ID, notificationID)
	if err != nil {
		return err
	}
	if affected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// MarkAllRead menandai seluruh notifikasi milik aktor.
func (s *NotificationService) MarkAllRead(ctx context.Context, actor Actor) (int64, error) {
	return s.notifications.MarkAllRead(ctx, actor.ID)
}
