package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditLog adalah baris `audit_logs` (`41-DATABASE.md` §2.5, `44-SECURITY.md` §6).
type AuditLog struct {
	ID          uuid.UUID       `json:"id"`
	ActorID     uuid.UUID       `json:"actor_id"`
	ActorName   string          `json:"actor_name"`
	Action      string          `json:"action"`
	Entity      string          `json:"entity"`
	EntityID    string          `json:"entity_id"`
	Description string          `json:"description"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedAt   time.Time       `json:"created_at"`
}
