package chat

import "time"

const (
	MaxHistoryMessages = 20
	MaxSearchDocs      = 10
	MaxDocContentLen   = 500
	MinMessageLength   = 3
	MaxMessageLength   = 2000
	MaxTokenWindow     = 28000
	ModelUsed          = "gemini-2.0-flash"
)

type ChatInput struct {
	CampaignID     string
	ConversationID string // "" = new conversation, non-empty = multi-turn
	Message        string
	Filters        ChatFilters
}

// ChatFilters - Optional search filters cho chat
type ChatFilters struct {
	Sentiments []string
	Aspects    []string
	Platforms  []string
	DateFrom   *int64
	DateTo     *int64
	RiskLevels []string
}

// GetConversationInput - Input cho GetConversation
type GetConversationInput struct {
	ConversationID string
}

// ListConversationsInput - Input cho ListConversations
type ListConversationsInput struct {
	CampaignID string
	Limit      int
	Offset     int
}

// GetSuggestionsInput - Input cho GetSuggestions
type GetSuggestionsInput struct {
	CampaignID string
}

// =====================================================
// Output Types
// =====================================================

// ChatOutput - Output cho Chat method
type ChatOutput struct {
	ConversationID string
	Answer         string
	Citations      []Citation
	Suggestions    []string
	SearchMetadata SearchMeta
}

// Citation - Trích dẫn từ search results
type Citation struct {
	ID             string
	Content        string
	RelevanceScore float64
	Platform       string
	Sentiment      string
}

// SearchMeta - Metadata thống kê xử lý
type SearchMeta struct {
	TotalDocsSearched int
	DocsUsed          int
	ProcessingTimeMs  int64
	ModelUsed         string
}

// ConversationOutput - Output cho conversation detail/list
type ConversationOutput struct {
	ID            string
	CampaignID    string
	UserID        string
	Title         string
	Status        string
	MessageCount  int
	Messages      []MessageOutput
	LastMessageAt *time.Time
	CreatedAt     time.Time
}

// MessageOutput - Output cho single message
type MessageOutput struct {
	ID             string
	Role           string
	Content        string
	Citations      []Citation
	SearchMetadata *SearchMeta
	Suggestions    []string
	FiltersUsed    *ChatFilters
	CreatedAt      time.Time
}

// SuggestionOutput - Output cho GetSuggestions
type SuggestionOutput struct {
	Suggestions []SmartSuggestion
}

// SmartSuggestion - Gợi ý câu hỏi thông minh
type SmartSuggestion struct {
	Query       string
	Category    string // "trending_negative", "sentiment_shift", "comparison", "insight"
	Description string
}
