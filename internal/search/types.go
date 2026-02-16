package search

const (
	MinScore       = 0.65
	MaxResults     = 10
	MinQueryLength = 3
	MaxQueryLength = 1000
)

type SearchInput struct {
	CampaignID string
	Query      string
	Filters    SearchFilters
	Limit      int
	MinScore   float64
}

type SearchFilters struct {
	Sentiments    []string
	Aspects       []string
	Platforms     []string
	DateFrom      *int64
	DateTo        *int64
	RiskLevels    []string
	MinEngagement *float64
}

type SearchOutput struct {
	Results           []SearchResult
	TotalFound        int
	Aggregations      Aggregations
	NoRelevantContext bool
	CacheHit          bool
	ProcessingTimeMs  int64
}

type SearchResult struct {
	ID               string
	Score            float64
	Content          string
	ProjectID        string
	Platform         string
	OverallSentiment string
	SentimentScore   float64
	Aspects          []AspectResult
	Keywords         []string
	RiskLevel        string
	EngagementScore  float64
	ContentCreatedAt int64
	Metadata         map[string]interface{}
}

type AspectResult struct {
	Aspect            string
	AspectDisplayName string
	Sentiment         string
	SentimentScore    float64
	Keywords          []string
}

type Aggregations struct {
	BySentiment []SentimentAgg
	ByAspect    []AspectAgg
	ByPlatform  []PlatformAgg
}

type SentimentAgg struct {
	Sentiment  string
	Count      int
	Percentage float64
}

type AspectAgg struct {
	Aspect            string
	AspectDisplayName string
	Count             int
	AvgSentimentScore float64
}

type PlatformAgg struct {
	Platform   string
	Count      int
	Percentage float64
}

type AggregateInput struct {
	CampaignID string
}

type AggregateOutput struct {
	TotalDocs          uint64
	SentimentBreakdown map[string]uint64
	PlatformBreakdown  map[string]uint64
	TopNegativeAspects []AspectCount
}

type AspectCount struct {
	Aspect string
	Count  uint64
}
