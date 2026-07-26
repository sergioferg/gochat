package routing

import (
	"github.com/google/uuid"
)

// ChatEvent represents a real-time event broadcasted over RabbitMQ
type ChatEvent struct {
	Type          string      `json:"type"`
	ChatID        uuid.UUID   `json:"chat_id"`
	SenderID      uuid.UUID   `json:"sender_id"`
	MessageID     uuid.UUID   `json:"message_id"`
	Content       string      `json:"content"`
	TargetUserIDs []uuid.UUID `json:"target_user_ids"`
}
