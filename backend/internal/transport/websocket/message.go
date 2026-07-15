package websocket

import (
	"time"

	"github.com/google/uuid"
)

type InboundMessage struct {
	Body        string  `json:"body"`
	ClientMsgID *string `json:"client_msg_id,omitempty"`
}

type OutboundType string

const (
	OutboundTypeMessage OutboundType = "message"
	OutboundTypeAck     OutboundType = "ack"
	OutboundTypeError   OutboundType = "error"
)

type OutboundMessage struct {
	Type    OutboundType    `json:"type"`
	Message *MessagePayload `json:"message,omitempty"`
	Error   string          `json:"error,omitempty"`
}

type MessagePayload struct {
	ID          uuid.UUID  `json:"id"`
	ChatID      uuid.UUID  `json:"chat_id"`
	SenderID    *uuid.UUID `json:"sender_id,omitempty"`
	Body        string     `json:"body"`
	ClientMsgID *string    `json:"client_msg_id,omitempty"`
	Seq         int64      `json:"seq"`
	CreatedAt   time.Time  `json:"created_at"`
}
