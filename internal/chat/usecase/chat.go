package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/chat/repository"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/notebook"
	nbrepo "knowledge-srv/internal/notebook/repository"
	"knowledge-srv/internal/search"
)

// Chat is the main RAG / NotebookLM routing entrypoint.
func (uc *implUseCase) Chat(ctx context.Context, sc model.Scope, input chat.ChatInput) (chat.ChatOutput, error) {
	startTime := time.Now()

	intent := ClassifyIntent(input.Message)

	if err := uc.validateChatInput(input); err != nil {
		uc.l.Errorf(ctx, "chat.usecase.Chat: validateChatInput failed: %v", err)
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
			uc.l.Errorf(ctx, "chat.usecase.Chat: conversation is archived")
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

	notebookEnabled := uc.cfg.NotebookEnabled && uc.notebookUC != nil
	notebookAvailable := false
	if notebookEnabled {
		ok, err := uc.notebookUC.HasSyncedForCampaign(ctx, input.CampaignID)
		if err != nil {
			uc.l.Warnf(ctx, "chat.usecase.Chat: HasSyncedForCampaign: %v", err)
		} else {
			notebookAvailable = ok
		}
	}

	route := RouteQuery(input.Message, notebookEnabled, notebookAvailable)

	if route.UseNotebook {
		filtersJSON, _ := json.Marshal(input.Filters)
		_, err := uc.repo.CreateMessage(ctx, repository.CreateMessageOptions{
			ConversationID: conversation.ID,
			Role:           "user",
			Content:        input.Message,
			FiltersUsed:    filtersJSON,
		})
		if err != nil {
			uc.l.Warnf(ctx, "chat.usecase.Chat: CreateMessage (user, async) failed: %v", err)
		}

		jobID, err := uc.notebookUC.SubmitChatJob(ctx, sc, conversation.ID, input.CampaignID, input.Message)
		if err != nil {
			uc.l.Errorf(ctx, "chat.usecase.Chat: SubmitChatJob failed: %v", err)
			return chat.ChatOutput{}, fmt.Errorf("notebook chat submission failed: %w", err)
		}

		_ = uc.repo.UpdateConversationLastMessage(ctx, repository.UpdateLastMessageOptions{
			ConversationID: conversation.ID,
			MessageCount:   conversation.MessageCount + 1,
		})

		return chat.ChatOutput{
			ConversationID: conversation.ID,
			ChatJobID:      jobID,
			IsAsync:        true,
			QueryIntent:    string(intent),
			Backend:        "NotebookLM",
		}, nil
	}

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

	prompt := uc.buildPrompt(input.Message, searchOutput.Results, history)

	answer, err := uc.llm.Generate(ctx, prompt)
	if err != nil {
		uc.l.Errorf(ctx, "chat.usecase.Chat: LLM failed: %v", err)
		return chat.ChatOutput{}, fmt.Errorf("%w: %v", chat.ErrLLMFailed, err)
	}

	citations := uc.extractCitations(searchOutput.Results)
	suggestions := uc.generateSuggestions(input.Message, searchOutput)

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
		IsAsync:        false,
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

func (uc *implUseCase) GetChatJobStatus(ctx context.Context, sc model.Scope, jobID string) (chat.JobStatusOutput, error) {
	if uc.notebookUC == nil {
		return chat.JobStatusOutput{}, fmt.Errorf("notebook integration is disabled")
	}

	job, err := uc.notebookUC.GetChatJobStatus(ctx, sc, jobID)
	if err != nil {
		if errors.Is(err, nbrepo.ErrChatJobNotFound) {
			return chat.JobStatusOutput{}, chat.ErrChatJobNotFound
		}
		return chat.JobStatusOutput{}, fmt.Errorf("failed to retrieve chat job status: %w", err)
	}

	timeout := uc.cfg.ChatTimeoutSec
	if timeout <= 0 {
		timeout = 45
	}

	if uc.cfg.NotebookFallbackEnabled && strings.EqualFold(job.Status, "PROCESSING") {
		if created, perr := time.Parse(time.RFC3339, job.CreatedAt); perr == nil {
			if time.Since(created) > time.Duration(timeout)*time.Second {
				answer, ferr := uc.runQdrantFallbackAnswer(ctx, sc, job.CampaignID, job.UserMessage)
				if ferr == nil && answer != "" {
					_ = uc.notebookUC.ApplyChatFallback(ctx, jobID, answer)
					job, _ = uc.notebookUC.GetChatJobStatus(ctx, sc, jobID)
				}
			}
		}
	}

	return mapNotebookJobToStatus(job), nil
}

func (uc *implUseCase) runQdrantFallbackAnswer(ctx context.Context, sc model.Scope, campaignID, message string) (string, error) {
	searchInput := search.SearchInput{
		CampaignID: campaignID,
		Query:      message,
		Limit:      chat.MaxSearchDocs,
		MinScore:   0.65,
	}
	searchOutput, err := uc.searchUC.Search(ctx, sc, searchInput)
	if err != nil {
		return "", err
	}
	prompt := uc.buildPrompt(message, searchOutput.Results, nil)
	return uc.llm.Generate(ctx, prompt)
}

func mapNotebookJobToStatus(job notebook.ChatJob) chat.JobStatusOutput {
	backend := "NotebookLM"
	if job.FallbackUsed {
		backend = "Qdrant"
	}

	st := strings.ToUpper(strings.TrimSpace(job.Status))
	switch st {
	case "COMPLETED":
		return chat.JobStatusOutput{State: chat.JobCompleted, Answer: job.NotebookAnswer, Backend: backend}
	case "FAILED":
		return chat.JobStatusOutput{State: chat.JobFailed, Backend: backend}
	case "EXPIRED":
		return chat.JobStatusOutput{State: chat.JobExpired, Backend: backend}
	case "PENDING":
		return chat.JobStatusOutput{State: chat.JobPending, Backend: backend}
	case "PROCESSING":
		return chat.JobStatusOutput{State: chat.JobProcessing, Backend: backend}
	default:
		return chat.JobStatusOutput{State: chat.JobProcessing, Backend: backend}
	}
}
