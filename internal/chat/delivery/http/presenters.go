package http

import (
	"time"

	"knowledge-srv/internal/chat"
)

type chatReq struct {
	CampaignID     string         `json:"campaign_id" binding:"required"`
	ConversationID string         `json:"conversation_id,omitempty"`
	Message        string         `json:"message" binding:"required,min=3,max=2000"`
	Filters        *chatFilterReq `json:"filters,omitempty"`
}

type chatFilterReq struct {
	Sentiments []string `json:"sentiments,omitempty"`
	Aspects    []string `json:"aspects,omitempty"`
	Platforms  []string `json:"platforms,omitempty"`
	DateFrom   *int64   `json:"date_from,omitempty"`
	DateTo     *int64   `json:"date_to,omitempty"`
	RiskLevels []string `json:"risk_levels,omitempty"`
}

func (r chatReq) toInput() chat.ChatInput {
	input := chat.ChatInput{
		CampaignID:     r.CampaignID,
		ConversationID: r.ConversationID,
		Message:        r.Message,
	}
	if r.Filters != nil {
		input.Filters = chat.ChatFilters{
			Sentiments: r.Filters.Sentiments,
			Aspects:    r.Filters.Aspects,
			Platforms:  r.Filters.Platforms,
			DateFrom:   r.Filters.DateFrom,
			DateTo:     r.Filters.DateTo,
			RiskLevels: r.Filters.RiskLevels,
		}
	}
	return input
}

type getConversationReq struct {
	ConversationID string
}

func (r getConversationReq) toInput() chat.GetConversationInput {
	return chat.GetConversationInput{
		ConversationID: r.ConversationID,
	}
}

type listConversationsReq struct {
	CampaignID string
	Limit      int
	Offset     int
}

func (r listConversationsReq) toInput() chat.ListConversationsInput {
	return chat.ListConversationsInput{
		CampaignID: r.CampaignID,
		Limit:      r.Limit,
		Offset:     r.Offset,
	}
}

type getSuggestionsReq struct {
	CampaignID string
}

func (r getSuggestionsReq) toInput() chat.GetSuggestionsInput {
	return chat.GetSuggestionsInput{
		CampaignID: r.CampaignID,
	}
}

type chatResp struct {
	ConversationID string         `json:"conversation_id"`
	Answer         string         `json:"answer"`
	Citations      []citationResp `json:"citations"`
	Suggestions    []string       `json:"suggestions"`
	SearchMetadata searchMetaResp `json:"search_metadata"`
}

type citationResp struct {
	ID             string  `json:"id"`
	Content        string  `json:"content"`
	RelevanceScore float64 `json:"relevance_score"`
	Platform       string  `json:"platform"`
	Sentiment      string  `json:"sentiment"`
}

type searchMetaResp struct {
	TotalDocsSearched int    `json:"total_docs_searched"`
	DocsUsed          int    `json:"docs_used"`
	ProcessingTimeMs  int64  `json:"processing_time_ms"`
	ModelUsed         string `json:"model_used"`
}

type conversationResp struct {
	ID            string        `json:"id"`
	CampaignID    string        `json:"campaign_id"`
	UserID        string        `json:"user_id"`
	Title         string        `json:"title"`
	Status        string        `json:"status"`
	MessageCount  int           `json:"message_count"`
	Messages      []messageResp `json:"messages,omitempty"`
	LastMessageAt *time.Time    `json:"last_message_at,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
}

type messageResp struct {
	ID             string          `json:"id"`
	Role           string          `json:"role"`
	Content        string          `json:"content"`
	Citations      []citationResp  `json:"citations,omitempty"`
	SearchMetadata *searchMetaResp `json:"search_metadata,omitempty"`
	Suggestions    []string        `json:"suggestions,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

type suggestionsResp struct {
	Suggestions []smartSuggestionResp `json:"suggestions"`
}

type smartSuggestionResp struct {
	Query       string `json:"query"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

func (h *handler) newChatResp(o chat.ChatOutput) chatResp {
	resp := chatResp{
		ConversationID: o.ConversationID,
		Answer:         o.Answer,
		Suggestions:    o.Suggestions,
		SearchMetadata: searchMetaResp{
			TotalDocsSearched: o.SearchMetadata.TotalDocsSearched,
			DocsUsed:          o.SearchMetadata.DocsUsed,
			ProcessingTimeMs:  o.SearchMetadata.ProcessingTimeMs,
			ModelUsed:         o.SearchMetadata.ModelUsed,
		},
	}
	resp.Citations = make([]citationResp, len(o.Citations))
	for i, c := range o.Citations {
		resp.Citations[i] = citationResp{
			ID:             c.ID,
			Content:        c.Content,
			RelevanceScore: c.RelevanceScore,
			Platform:       c.Platform,
			Sentiment:      c.Sentiment,
		}
	}
	return resp
}

func (h *handler) newConversationResp(o chat.ConversationOutput) conversationResp {
	resp := conversationResp{
		ID:            o.ID,
		CampaignID:    o.CampaignID,
		UserID:        o.UserID,
		Title:         o.Title,
		Status:        o.Status,
		MessageCount:  o.MessageCount,
		LastMessageAt: o.LastMessageAt,
		CreatedAt:     o.CreatedAt,
	}
	for _, m := range o.Messages {
		msgResp := messageResp{
			ID:          m.ID,
			Role:        m.Role,
			Content:     m.Content,
			Suggestions: m.Suggestions,
			CreatedAt:   m.CreatedAt,
		}
		for _, c := range m.Citations {
			msgResp.Citations = append(msgResp.Citations, citationResp{
				ID:             c.ID,
				Content:        c.Content,
				RelevanceScore: c.RelevanceScore,
				Platform:       c.Platform,
				Sentiment:      c.Sentiment,
			})
		}
		if m.SearchMetadata != nil {
			msgResp.SearchMetadata = &searchMetaResp{
				TotalDocsSearched: m.SearchMetadata.TotalDocsSearched,
				DocsUsed:          m.SearchMetadata.DocsUsed,
				ProcessingTimeMs:  m.SearchMetadata.ProcessingTimeMs,
				ModelUsed:         m.SearchMetadata.ModelUsed,
			}
		}
		resp.Messages = append(resp.Messages, msgResp)
	}
	return resp
}

func (h *handler) newListConversationsResp(convos []chat.ConversationOutput) []conversationResp {
	resp := make([]conversationResp, len(convos))
	for i, c := range convos {
		resp[i] = h.newConversationResp(c)
	}
	return resp
}

func (h *handler) newSuggestionsResp(o chat.SuggestionOutput) suggestionsResp {
	resp := suggestionsResp{}
	resp.Suggestions = make([]smartSuggestionResp, len(o.Suggestions))
	for i, s := range o.Suggestions {
		resp.Suggestions[i] = smartSuggestionResp{
			Query:       s.Query,
			Category:    s.Category,
			Description: s.Description,
		}
	}
	return resp
}
