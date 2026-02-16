package chat

import (
	"context"
	"knowledge-srv/internal/model"
)

//go:generate mockery --name UseCase
type UseCase interface {
	Chat(ctx context.Context, sc model.Scope, input ChatInput) (ChatOutput, error)
	GetConversation(ctx context.Context, sc model.Scope, input GetConversationInput) (ConversationOutput, error)
	ListConversations(ctx context.Context, sc model.Scope, input ListConversationsInput) ([]ConversationOutput, error)
	GetSuggestions(ctx context.Context, sc model.Scope, input GetSuggestionsInput) (SuggestionOutput, error)
}
