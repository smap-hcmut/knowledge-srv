package transform

import "time"

// TransformInput contains the batch of posts to transform into markdown.
type TransformInput struct {
	CampaignID   string
	CampaignName string
	Posts        []AnalyticsPostLite
}

// AnalyticsPostLite is a lightweight subset of indexing.AnalyticsPost
// containing only the fields needed for markdown generation.
type AnalyticsPostLite struct {
	ID               string
	Content          string
	Platform         string
	Author           string
	ContentCreatedAt time.Time
	Sentiment        string
	SentimentScore   float64
	Aspects          []AspectLite
	Keywords         []string
	RiskLevel        string
	RiskScore        float64
	EngagementScore  float64
	Likes            int
	Comments         int
	Shares           int
	Views            int
}

// AspectLite is a lightweight subset of indexing.Aspect.
type AspectLite struct {
	Name           string
	DisplayName    string
	Sentiment      string
	SentimentScore float64
	Keywords       []string
}

// MarkdownPart represents a single markdown document to be uploaded as a NotebookLM source.
type MarkdownPart struct {
	Title       string // e.g. "SMAP | Campaign X | 2026-W12 | Part 1"
	Content     string // Full markdown text
	WeekLabel   string // e.g. "2026-W12"
	PartNum     int
	PostCount   int
	ContentHash string // SHA256 for dedup
}
