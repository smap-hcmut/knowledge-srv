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
	ConversationID string
	Message        string
	Filters        ChatFilters
}

type ChatFilters struct {
	Sentiments []string
	Aspects    []string
	Platforms  []string
	DateFrom   *int64
	DateTo     *int64
	RiskLevels []string
}

type GetConversationInput struct {
	ConversationID string
}

type ListConversationsInput struct {
	CampaignID string
	Limit      int
	Offset     int
}

type GetSuggestionsInput struct {
	CampaignID string
}

type ChatOutput struct {
	ConversationID string
	Answer         string
	Citations      []Citation
	Suggestions    []string
	SearchMetadata SearchMeta
}

type Citation struct {
	ID             string
	Content        string
	RelevanceScore float64
	Platform       string
	Sentiment      string
}

type SearchMeta struct {
	TotalDocsSearched int
	DocsUsed          int
	ProcessingTimeMs  int64
	ModelUsed         string
}

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

type SuggestionOutput struct {
	Suggestions []SmartSuggestion
}

type SmartSuggestion struct {
	Query       string
	Category    string
	Description string
}
