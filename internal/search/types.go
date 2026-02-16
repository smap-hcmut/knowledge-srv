package search

// =====================================================
// Input Types
// =====================================================

// SearchInput - Input cho method Search
type SearchInput struct {
	CampaignID string        // Campaign scope (resolve → project_ids)
	Query      string        // Search query text
	Filters    SearchFilters // Optional filters
	Limit      int           // Max results (default 10, max 50)
	MinScore   float64       // Min relevance score (default 0.65)
}

// SearchFilters - Multi-criteria filters
type SearchFilters struct {
	Sentiments    []string // Filter by overall_sentiment: POSITIVE, NEGATIVE, NEUTRAL, MIXED
	Aspects       []string // Filter by aspect name: DESIGN, BATTERY, PRICE, ...
	Platforms     []string // Filter by platform: tiktok, youtube, facebook, ...
	DateFrom      *int64   // Unix timestamp — start of date range
	DateTo        *int64   // Unix timestamp — end of date range
	RiskLevels    []string // Filter by risk_level: LOW, MEDIUM, HIGH, CRITICAL
	MinEngagement *float64 // Min engagement_score
}

// =====================================================
// Output Types
// =====================================================

// SearchOutput - Kết quả search
type SearchOutput struct {
	Results           []SearchResult // Ranked results
	TotalFound        int            // Total matching documents
	Aggregations      Aggregations   // Aggregated stats
	NoRelevantContext bool           // True khi không đủ relevant docs → hallucination control
	CacheHit          bool           // True nếu kết quả từ cache
	ProcessingTimeMs  int64          // Processing time
}

// SearchResult - 1 document tìm được
type SearchResult struct {
	ID               string                 // analytics_id (= Qdrant point ID)
	Score            float64                // Relevance score (0.0 - 1.0)
	Content          string                 // Content text (truncated)
	ProjectID        string                 // project_id
	Platform         string                 // Platform name
	OverallSentiment string                 // POSITIVE | NEGATIVE | NEUTRAL | MIXED
	SentimentScore   float64                // -1.0 to 1.0
	Aspects          []AspectResult         // Aspects matched
	Keywords         []string               // Keywords
	RiskLevel        string                 // LOW | MEDIUM | HIGH | CRITICAL
	EngagementScore  float64                // 0.0 to 1.0
	ContentCreatedAt int64                  // Unix timestamp
	Metadata         map[string]interface{} // Author, engagement, etc.
}

// AspectResult - Aspect trong search result
type AspectResult struct {
	Aspect            string  // Aspect name
	AspectDisplayName string  // Display name
	Sentiment         string  // Sentiment of this aspect
	SentimentScore    float64 // Score
	Keywords          []string
}

// =====================================================
// Aggregation Types
// =====================================================

// Aggregations - Tổng hợp thống kê
type Aggregations struct {
	BySentiment []SentimentAgg // Phân bố sentiment
	ByAspect    []AspectAgg    // Phân bố aspects
	ByPlatform  []PlatformAgg  // Phân bố platforms
}

// SentimentAgg - Aggregation by sentiment
type SentimentAgg struct {
	Sentiment  string // POSITIVE, NEGATIVE, NEUTRAL, MIXED
	Count      int
	Percentage float64
}

// AspectAgg - Aggregation by aspect
type AspectAgg struct {
	Aspect            string
	AspectDisplayName string
	Count             int
	AvgSentimentScore float64
}

// PlatformAgg - Aggregation by platform
type PlatformAgg struct {
	Platform   string
	Count      int
	Percentage float64
}
