package repository

import (
	"context"
	"knowledge-srv/internal/model"
)

//go:generate mockery --name PostgresRepository
type PostgresRepository interface {
	ConversationRepository
	MessageRepository
}

// ConversationRepository - Interface cho conversation CRUD
type ConversationRepository interface {
	CreateConversation(ctx context.Context, opt CreateConversationOptions) (model.Conversation, error)
	GetConversationByID(ctx context.Context, id string) (model.Conversation, error)
	ListConversations(ctx context.Context, opt ListConversationsOptions) ([]model.Conversation, error)
	UpdateConversationLastMessage(ctx context.Context, opt UpdateLastMessageOptions) error
	ArchiveConversation(ctx context.Context, id string) error
}

// MessageRepository - Interface cho message CRUD
type MessageRepository interface {
	CreateMessage(ctx context.Context, opt CreateMessageOptions) (model.Message, error)
	ListMessages(ctx context.Context, opt ListMessagesOptions) ([]model.Message, error)
}
