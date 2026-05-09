package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	BaseURL    string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type Client interface {
	Snapshot(ctx context.Context, campaignID string) (Snapshot, error)
}

type implClient struct {
	baseURL string
	client  *http.Client
}

func New(cfg Config) Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 12 * time.Second
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	return &implClient{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		client:  client,
	}
}

type Snapshot struct {
	KPIs      KPIsResponse
	Platforms PlatformsResponse
	Sentiment SentimentResponse
	Keywords  KeywordsResponse
	Posts     PostsResponse
	Errors    []string
}

func (s Snapshot) HasData() bool {
	if s.Sentiment.Total > 0 || s.Posts.Total > 0 || len(s.Posts.Posts) > 0 {
		return true
	}
	for _, stat := range s.Platforms.Stats {
		if stat.Mentions > 0 {
			return true
		}
	}
	for _, metric := range s.KPIs.Metrics {
		if strings.EqualFold(metric.Label, "Total Mentions") && metric.Value > 0 {
			return true
		}
	}
	return false
}

type KPIsResponse struct {
	Metrics    []KPIMetric `json:"metrics"`
	Engagement Engagement  `json:"engagement"`
}

type KPIMetric struct {
	Label     string  `json:"label"`
	Value     float64 `json:"value"`
	Formatted string  `json:"formatted"`
	Change    float64 `json:"change"`
}

type Engagement struct {
	Views    int64 `json:"views"`
	Likes    int64 `json:"likes"`
	Comments int64 `json:"comments"`
	Shares   int64 `json:"shares"`
}

type PlatformsResponse struct {
	Stats []PlatformStat `json:"stats"`
}

type PlatformStat struct {
	Platform      string  `json:"platform"`
	Name          string  `json:"name"`
	Mentions      int64   `json:"mentions"`
	EngagementRaw int64   `json:"engagementRaw"`
	Sentiment     float64 `json:"sentiment"`
	Reach         int64   `json:"reach"`
}

type SentimentResponse struct {
	Donut []SentimentItem `json:"donut"`
	Pulse float64         `json:"pulse"`
	Total int64           `json:"total"`
}

type SentimentItem struct {
	Label string `json:"label"`
	Value int64  `json:"value"`
}

type KeywordsResponse struct {
	Keywords []KeywordItem `json:"keywords"`
}

type KeywordItem struct {
	Text      string  `json:"text"`
	Volume    int64   `json:"volume"`
	Sentiment float64 `json:"sentiment"`
	Change    float64 `json:"change"`
}

type PostsResponse struct {
	Posts []PostItem `json:"posts"`
	Total int64      `json:"total"`
}

type PostItem struct {
	ID             string   `json:"id"`
	Platform       string   `json:"platform"`
	Author         string   `json:"author"`
	AuthorUsername string   `json:"authorUsername"`
	Content        string   `json:"content"`
	URL            string   `json:"url"`
	Sentiment      string   `json:"sentiment"`
	SentimentScore float64  `json:"sentimentScore"`
	Engagement     int64    `json:"engagement"`
	Likes          int64    `json:"likes"`
	Comments       int64    `json:"comments"`
	Shares         int64    `json:"shares"`
	Keywords       []string `json:"keywords"`
	RiskLevel      string   `json:"riskLevel"`
}

func (c *implClient) Snapshot(ctx context.Context, campaignID string) (Snapshot, error) {
	var snapshot Snapshot
	if strings.TrimSpace(c.baseURL) == "" {
		return snapshot, fmt.Errorf("analytics base URL is empty")
	}

	requests := []struct {
		name   string
		path   string
		params map[string]string
		target any
	}{
		{name: "kpis", path: "/api/v1/analytics/kpis", params: map[string]string{"campaignId": campaignID}, target: &snapshot.KPIs},
		{name: "platforms", path: "/api/v1/analytics/platforms", params: map[string]string{"campaignId": campaignID}, target: &snapshot.Platforms},
		{name: "sentiment", path: "/api/v1/analytics/sentiment", params: map[string]string{"campaignId": campaignID}, target: &snapshot.Sentiment},
		{name: "keywords", path: "/api/v1/analytics/keywords", params: map[string]string{"campaignId": campaignID, "limit": "8"}, target: &snapshot.Keywords},
		{name: "posts", path: "/api/v1/analytics/posts", params: map[string]string{"campaignId": campaignID, "sort": "engagement", "limit": "8", "offset": "0"}, target: &snapshot.Posts},
	}

	for _, req := range requests {
		if err := c.getJSON(ctx, req.path, req.params, req.target); err != nil {
			snapshot.Errors = append(snapshot.Errors, fmt.Sprintf("%s: %v", req.name, err))
		}
	}

	if !snapshot.HasData() && len(snapshot.Errors) > 0 {
		return snapshot, fmt.Errorf("analytics snapshot unavailable: %s", strings.Join(snapshot.Errors, "; "))
	}
	return snapshot, nil
}

func (c *implClient) getJSON(ctx context.Context, path string, params map[string]string, target any) error {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return err
	}
	q := u.Query()
	for key, value := range params {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errorBody struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errorBody)
		if strings.TrimSpace(errorBody.Error) != "" {
			return fmt.Errorf("status %d: %s", resp.StatusCode, errorBody.Error)
		}
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return err
	}
	if len(raw) == 0 {
		return nil
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && len(envelope.Data) > 0 {
		raw = envelope.Data
	}
	return json.Unmarshal(raw, target)
}
