package chat

import "time"

const (
	// Trimmed 2026-06-09 to cut DeepSeek round-trip from ~30 s to
	// ~5-10 s. Each value tracks a different cost lever:
	//   MaxHistoryMessages  — more history = longer prompt = slower TTFT
	//   MaxSearchDocs       — bigger context window per turn
	//   MaxDocContentLen    — each citation excerpt size
	//   MaxTokenWindow      — hard ceiling before buildReducedPrompt kicks in
	// The reduced values still cover normal RAG turns; the previous
	// 28 000-token ceiling was sized for a multi-doc deep dive that the
	// chat box rarely actually issues.
	MaxHistoryMessages = 8
	MaxSearchDocs      = 6
	MaxDocContentLen   = 320
	MinMessageLength   = 3
	MaxMessageLength   = 2000
	MaxTokenWindow     = 8000
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
	Backend        string
	QueryIntent    string
}

type Citation struct {
	ID             string
	Content        string
	RelevanceScore float64
	Platform       string
	Sentiment      string
	URL            string
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
