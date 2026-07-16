package models

import (
	"time"

	"github.com/google/uuid"
)

type ChatType string

const (
	ChatTypeDirect ChatType = "direct"
	ChatTypeGroup  ChatType = "group"
)

type ChatRole string

const (
	ChatRoleMember ChatRole = "member"
	ChatRoleAdmin  ChatRole = "admin"
)

type Chat struct {
	ID            uuid.UUID
	Type          ChatType
	Title         *string
	CreatedBy     *uuid.UUID
	DirectKey     *string
	LastMessageAt *time.Time
	CreatedAt     time.Time
}

type ChatMember struct {
	ChatID   uuid.UUID
	UserID   uuid.UUID
	Role     ChatRole
	JoinedAt time.Time
}

type ChatMemberInfo struct {
	ChatID   uuid.UUID
	UserID   uuid.UUID
	Login    string
	Role     ChatRole
	JoinedAt time.Time
}
