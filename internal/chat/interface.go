package chat

import (
	"context"
	"knowledge-srv/internal/model"
)

//go:generate mockery --name UseCase
type UseCase interface {
	// Chat Logic
	Chat(ctx context.Context, sc model.Scope, input ChatInput) (ChatOutput, error)

	// Conversation Management
	GetConversation(ctx context.Context, sc model.Scope, input GetConversationInput) (ConversationOutput, error)
	ListConversations(ctx context.Context, sc model.Scope, input ListConversationsInput) ([]ConversationOutput, error)

	// Smart Suggestions
	GetSuggestions(ctx context.Context, sc model.Scope, input GetSuggestionsInput) (SuggestionOutput, error)
}
