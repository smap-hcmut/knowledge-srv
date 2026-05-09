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

// Chat is the main synchronous RAG entrypoint.
func (uc *implUseCase) Chat(ctx context.Context, sc model.Scope, input chat.ChatInput) (chat.ChatOutput, error) {
	startTime := time.Now()

	intent := ClassifyIntent(input.Message)

	if err := uc.validateChatInput(input); err != nil {
		uc.l.Warnf(ctx, "chat.usecase.Chat: validateChatInput failed: %v", err)
		return chat.ChatOutput{}, err
	}

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
			uc.l.Errorf(ctx, "chat.usecase.Chat: GetConversationByID failed: %v", err)
			return chat.ChatOutput{}, chat.ErrConversationNotFound
		}
		if conv.Status == "ARCHIVED" {
			uc.l.Warnf(ctx, "chat.usecase.Chat: conversation is archived")
			return chat.ChatOutput{}, chat.ErrConversationArchived
		}
		conversation = conv

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

	// Build search input — tune params based on query intent
	var searchLimit int
	var searchMinScore float64
	switch intent {
	case IntentNarrative:
		searchLimit = 15
		searchMinScore = 0.58
	default: // IntentStructured
		searchLimit = chat.MaxSearchDocs
		searchMinScore = 0.52
	}

	searchInput := search.SearchInput{
		CampaignID: input.CampaignID,
		Query:      input.Message,
		Limit:      searchLimit,
		MinScore:   searchMinScore,
	}
	searchFilters := search.SearchFilters{
		Sentiments: input.Filters.Sentiments,
		Aspects:    input.Filters.Aspects,
		Platforms:  input.Filters.Platforms,
		DateFrom:   input.Filters.DateFrom,
		DateTo:     input.Filters.DateTo,
		RiskLevels: input.Filters.RiskLevels,
	}
	if len(searchFilters.Platforms) == 0 {
		searchFilters.Platforms = InferPlatforms(input.Message)
	}
	if len(searchFilters.Sentiments) > 0 || len(searchFilters.Aspects) > 0 ||
		len(searchFilters.Platforms) > 0 || len(searchFilters.RiskLevels) > 0 ||
		searchFilters.DateFrom != nil || searchFilters.DateTo != nil {
		searchInput.Filters = searchFilters
	}

	searchOutput, err := uc.searchUC.Search(ctx, sc, searchInput)
	if err != nil {
		uc.l.Errorf(ctx, "chat.usecase.Chat: Search failed: %v", err)
		return chat.ChatOutput{}, fmt.Errorf("%w: %v", chat.ErrSearchFailed, err)
	}
	if searchOutput.NoRelevantContext || len(searchOutput.Results) == 0 {
		if output, ok := uc.tryAnalyticsFallback(ctx, conversation, input, startTime, intent); ok {
			return output, nil
		}

		answer := "Mình chưa tìm thấy đủ dữ liệu liên quan trong campaign này để trả lời chắc chắn. Bạn có thể hỏi hẹp hơn theo nền tảng, khoảng thời gian, hoặc chủ đề cụ thể như phí giao hàng, tài xế, hủy đơn, hỗ trợ."
		suggestions := uc.generateSuggestions(input.Message, searchOutput)
		searchMeta := chat.SearchMeta{
			TotalDocsSearched: searchOutput.TotalFound,
			DocsUsed:          0,
			ProcessingTimeMs:  time.Since(startTime).Milliseconds(),
			ModelUsed:         uc.llm.Name(),
		}
		uc.persistChatExchange(ctx, conversation, input, answer, nil, suggestions, searchMeta)
		return chat.ChatOutput{
			ConversationID: conversation.ID,
			Answer:         answer,
			Citations:      nil,
			Suggestions:    suggestions,
			SearchMetadata: searchMeta,
			QueryIntent:    string(intent),
			Backend:        "Qdrant",
		}, nil
	}

	prompt := uc.buildPrompt(input.Message, searchOutput.Results, history)

	llmCtx, llmCancel := context.WithTimeout(ctx, 60*time.Second)
	defer llmCancel()
	answer, err := uc.llm.Generate(llmCtx, prompt)
	if err != nil {
		uc.l.Errorf(ctx, "chat.usecase.Chat: LLM failed: %v", err)
		return chat.ChatOutput{}, fmt.Errorf("%w: %v", chat.ErrLLMFailed, err)
	}

	citations := uc.extractCitations(searchOutput.Results)
	suggestions := uc.generateSuggestions(input.Message, searchOutput)

	// Persist user message
	filtersJSON, _ := json.Marshal(input.Filters)
	_, err = uc.repo.CreateMessage(ctx, repository.CreateMessageOptions{
		ConversationID: conversation.ID,
		Role:           "user",
		Content:        input.Message,
		FiltersUsed:    filtersJSON,
	})
	if err != nil {
		uc.l.Warnf(ctx, "chat.usecase.Chat: CreateMessage failed: %v", err)
	}

	// Persist assistant message
	searchMeta := chat.SearchMeta{
		TotalDocsSearched: searchOutput.TotalFound,
		DocsUsed:          len(citations),
		ProcessingTimeMs:  time.Since(startTime).Milliseconds(),
		ModelUsed:         uc.llm.Name(),
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
		uc.l.Warnf(ctx, "chat.usecase.Chat: CreateMessage failed: %v", err)
	}

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
		QueryIntent:    string(intent),
		Backend:        "Qdrant",
	}, nil
}

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
