package models

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID          uuid.UUID
	ChatID      uuid.UUID
	SenderID    *uuid.UUID
	Body        string
	ClientMsgID *string
	CreatedAt   time.Time
}
