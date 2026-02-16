package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/chat/repository"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/search"
)

// Chat - Main RAG pipeline
// Flow: validate → resolve/create conversation → load history → search → build prompt → LLM → save → return
func (uc *implUseCase) Chat(ctx context.Context, sc model.Scope, input chat.ChatInput) (chat.ChatOutput, error) {
	startTime := time.Now()

	// Step 0: Validate input
	if err := uc.validateChatInput(input); err != nil {
		return chat.ChatOutput{}, err
	}

	// Step 1: Resolve or create conversation
	var conversation model.Conversation
	var history []model.Message
	isNewConversation := input.ConversationID == ""

	if isNewConversation {
		title := uc.generateTitle(input.Message)
		conv, err := uc.repo.CreateConversation(ctx, repository.CreateConversationOptions{
			CampaignID: input.CampaignID,
			UserID:     sc.UserID,
			Title:      title,
		})
		if err != nil {
			uc.l.Errorf(ctx, "chat.usecase.Chat: CreateConversation failed: %v", err)
			return chat.ChatOutput{}, fmt.Errorf("create conversation: %w", err)
		}
		conversation = conv
	} else {
		conv, err := uc.repo.GetConversationByID(ctx, input.ConversationID)
		if err != nil {
			return chat.ChatOutput{}, chat.ErrConversationNotFound
		}
		if conv.Status == "ARCHIVED" {
			return chat.ChatOutput{}, chat.ErrConversationArchived
		}
		conversation = conv

		// Load history (max N recent messages)
		msgs, err := uc.repo.ListMessages(ctx, repository.ListMessagesOptions{
			ConversationID: conversation.ID,
			Limit:          chat.MaxHistoryMessages,
			OrderASC:       true,
		})
		if err != nil {
			uc.l.Warnf(ctx, "chat.usecase.Chat: ListMessages failed: %v", err)
		}
		history = msgs
	}

	// Step 2: Search relevant documents (via search.UseCase)
	searchInput := search.SearchInput{
		CampaignID: input.CampaignID,
		Query:      input.Message,
		Limit:      chat.MaxSearchDocs,
		MinScore:   0.65,
	}
	if len(input.Filters.Sentiments) > 0 || len(input.Filters.Aspects) > 0 ||
		len(input.Filters.Platforms) > 0 || len(input.Filters.RiskLevels) > 0 ||
		input.Filters.DateFrom != nil || input.Filters.DateTo != nil {
		searchInput.Filters = search.SearchFilters{
			Sentiments: input.Filters.Sentiments,
			Aspects:    input.Filters.Aspects,
			Platforms:  input.Filters.Platforms,
			DateFrom:   input.Filters.DateFrom,
			DateTo:     input.Filters.DateTo,
			RiskLevels: input.Filters.RiskLevels,
		}
	}

	searchOutput, err := uc.searchUC.Search(ctx, sc, searchInput)
	if err != nil {
		uc.l.Errorf(ctx, "chat.usecase.Chat: Search failed: %v", err)
		return chat.ChatOutput{}, fmt.Errorf("%w: %v", chat.ErrSearchFailed, err)
	}

	// Step 3: Build LLM prompt (with token window management)
	prompt := uc.buildPrompt(input.Message, searchOutput.Results, history)

	// Step 4: Call LLM
	answer, err := uc.gemini.Generate(ctx, prompt)
	if err != nil {
		uc.l.Errorf(ctx, "chat.usecase.Chat: LLM failed: %v", err)
		return chat.ChatOutput{}, fmt.Errorf("%w: %v", chat.ErrLLMFailed, err)
	}

	// Step 5: Extract citations from search results
	citations := uc.extractCitations(searchOutput.Results)

	// Step 6: Generate follow-up suggestions
	suggestions := uc.generateSuggestions(input.Message, searchOutput)

	// Step 7: Save user message
	filtersJSON, _ := json.Marshal(input.Filters)
	_, err = uc.repo.CreateMessage(ctx, repository.CreateMessageOptions{
		ConversationID: conversation.ID,
		Role:           "user",
		Content:        input.Message,
		FiltersUsed:    filtersJSON,
	})
	if err != nil {
		uc.l.Warnf(ctx, "chat.usecase.Chat: save user message failed: %v", err)
	}

	// Step 8: Save assistant message
	searchMeta := chat.SearchMeta{
		TotalDocsSearched: searchOutput.TotalFound,
		DocsUsed:          len(citations),
		ProcessingTimeMs:  time.Since(startTime).Milliseconds(),
		ModelUsed:         chat.ModelUsed,
	}
	citationsJSON, _ := json.Marshal(citations)
	suggestionsJSON, _ := json.Marshal(suggestions)
	searchMetaJSON, _ := json.Marshal(searchMeta)

	_, err = uc.repo.CreateMessage(ctx, repository.CreateMessageOptions{
		ConversationID: conversation.ID,
		Role:           "assistant",
		Content:        answer,
		Citations:      citationsJSON,
		SearchMetadata: searchMetaJSON,
		Suggestions:    suggestionsJSON,
	})
	if err != nil {
		uc.l.Warnf(ctx, "chat.usecase.Chat: save assistant message failed: %v", err)
	}

	// Step 9: Update conversation metadata
	newCount := conversation.MessageCount + 2
	_ = uc.repo.UpdateConversationLastMessage(ctx, repository.UpdateLastMessageOptions{
		ConversationID: conversation.ID,
		MessageCount:   newCount,
	})

	return chat.ChatOutput{
		ConversationID: conversation.ID,
		Answer:         answer,
		Citations:      citations,
		Suggestions:    suggestions,
		SearchMetadata: searchMeta,
	}, nil
}

// validateChatInput - Validate chat input
func (uc *implUseCase) validateChatInput(input chat.ChatInput) error {
	if input.CampaignID == "" {
		return chat.ErrCampaignRequired
	}
	if len(input.Message) < chat.MinMessageLength {
		return chat.ErrMessageTooShort
	}
	if len(input.Message) > chat.MaxMessageLength {
		return chat.ErrMessageTooLong
	}
	return nil
}
