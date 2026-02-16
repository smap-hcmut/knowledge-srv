package usecase

import (
	"context"
	"encoding/json"

	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/chat/repository"
	"knowledge-srv/internal/model"
)

// GetConversation - Lấy conversation + messages
func (uc *implUseCase) GetConversation(ctx context.Context, sc model.Scope, input chat.GetConversationInput) (chat.ConversationOutput, error) {
	conv, err := uc.repo.GetConversationByID(ctx, input.ConversationID)
	if err != nil {
		return chat.ConversationOutput{}, chat.ErrConversationNotFound
	}

	msgs, err := uc.repo.ListMessages(ctx, repository.ListMessagesOptions{
		ConversationID: conv.ID,
		OrderASC:       true,
	})
	if err != nil {
		uc.l.Warnf(ctx, "chat.usecase.GetConversation: ListMessages failed: %v", err)
	}

	return uc.toConversationOutput(conv, msgs), nil
}

// ListConversations - Liệt kê conversations theo campaign + user
func (uc *implUseCase) ListConversations(ctx context.Context, sc model.Scope, input chat.ListConversationsInput) ([]chat.ConversationOutput, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}

	convos, err := uc.repo.ListConversations(ctx, repository.ListConversationsOptions{
		CampaignID: input.CampaignID,
		UserID:     sc.UserID,
		Limit:      limit,
		Offset:     input.Offset,
	})
	if err != nil {
		return nil, err
	}

	results := make([]chat.ConversationOutput, len(convos))
	for i, c := range convos {
		results[i] = uc.toConversationOutput(c, nil) // no messages for list view
	}
	return results, nil
}

// toConversationOutput - Convert model.Conversation + messages to chat.ConversationOutput
func (uc *implUseCase) toConversationOutput(conv model.Conversation, msgs []model.Message) chat.ConversationOutput {
	output := chat.ConversationOutput{
		ID:            conv.ID,
		CampaignID:    conv.CampaignID,
		UserID:        conv.UserID,
		Title:         conv.Title,
		Status:        conv.Status,
		MessageCount:  conv.MessageCount,
		LastMessageAt: conv.LastMessageAt,
		CreatedAt:     conv.CreatedAt,
	}
	for _, m := range msgs {
		output.Messages = append(output.Messages, uc.toMessageOutput(m))
	}
	return output
}

// toMessageOutput - Convert model.Message to chat.MessageOutput
func (uc *implUseCase) toMessageOutput(m model.Message) chat.MessageOutput {
	output := chat.MessageOutput{
		ID:        m.ID,
		Role:      m.Role,
		Content:   m.Content,
		CreatedAt: m.CreatedAt,
	}

	// Parse JSONB fields
	if len(m.Citations) > 0 && string(m.Citations) != "null" {
		var citations []chat.Citation
		if err := json.Unmarshal(m.Citations, &citations); err == nil {
			output.Citations = citations
		}
	}
	if len(m.SearchMetadata) > 0 && string(m.SearchMetadata) != "null" {
		var meta chat.SearchMeta
		if err := json.Unmarshal(m.SearchMetadata, &meta); err == nil {
			output.SearchMetadata = &meta
		}
	}
	if len(m.Suggestions) > 0 && string(m.Suggestions) != "null" {
		var suggestions []string
		if err := json.Unmarshal(m.Suggestions, &suggestions); err == nil {
			output.Suggestions = suggestions
		}
	}
	if len(m.FiltersUsed) > 0 && string(m.FiltersUsed) != "null" {
		var filters chat.ChatFilters
		if err := json.Unmarshal(m.FiltersUsed, &filters); err == nil {
			output.FiltersUsed = &filters
		}
	}

	return output
}
