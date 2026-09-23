package model

import (
	"time"

	"github.com/google/uuid"
)

// Notification adalah baris `notifications` (`41-DATABASE.md` §2.5).
type Notification struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	Type       string     `json:"type"`
	Title      string     `json:"title"`
	Message    string     `json:"message"`
	EntityID   *uuid.UUID `json:"entity_id"`
	EntityType *string    `json:"entity_type"`
	IsRead     bool       `json:"is_read"`
	CreatedAt  time.Time  `json:"created_at"`
}
