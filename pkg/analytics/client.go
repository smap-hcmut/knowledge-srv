package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Config struct {
	BaseURL    string
	Timeout    time.Duration
	CacheTTL   time.Duration
	HTTPClient *http.Client
}

type Client interface {
	Snapshot(ctx context.Context, campaignID string) (Snapshot, error)
}

type implClient struct {
	baseURL  string
	client   *http.Client
	cacheTTL time.Duration
	mu       sync.RWMutex
	cache    map[string]cachedSnapshot
}

type cachedSnapshot struct {
	value     Snapshot
	expiresAt time.Time
}

func New(cfg Config) Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 12 * time.Second
	}
	cacheTTL := cfg.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = 45 * time.Second
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	return &implClient{
		baseURL:  strings.TrimRight(cfg.BaseURL, "/"),
		client:   client,
		cacheTTL: cacheTTL,
		cache:    make(map[string]cachedSnapshot),
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

func (s Snapshot) HasCoreAnalytics() bool {
	return len(s.KPIs.Metrics) > 0 && len(s.Platforms.Stats) > 0 && s.Sentiment.Total > 0
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
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return snapshot, fmt.Errorf("campaign id is empty")
	}

	if cached, ok := c.getCachedSnapshot(campaignID); ok {
		return cached, nil
	}

	var mu sync.Mutex
	requests := []struct {
		name string
		run  func(context.Context) error
	}{
		{name: "kpis", run: func(ctx context.Context) error {
			var out KPIsResponse
			if err := c.getJSON(ctx, "/api/v1/analytics/kpis", map[string]string{"campaignId": campaignID}, &out); err != nil {
				return err
			}
			mu.Lock()
			snapshot.KPIs = out
			mu.Unlock()
			return nil
		}},
		{name: "platforms", run: func(ctx context.Context) error {
			var out PlatformsResponse
			if err := c.getJSON(ctx, "/api/v1/analytics/platforms", map[string]string{"campaignId": campaignID}, &out); err != nil {
				return err
			}
			mu.Lock()
			snapshot.Platforms = out
			mu.Unlock()
			return nil
		}},
		{name: "sentiment", run: func(ctx context.Context) error {
			var out SentimentResponse
			if err := c.getJSON(ctx, "/api/v1/analytics/sentiment", map[string]string{"campaignId": campaignID}, &out); err != nil {
				return err
			}
			mu.Lock()
			snapshot.Sentiment = out
			mu.Unlock()
			return nil
		}},
		{name: "keywords", run: func(ctx context.Context) error {
			var out KeywordsResponse
			if err := c.getJSON(ctx, "/api/v1/analytics/keywords", map[string]string{"campaignId": campaignID, "limit": "12"}, &out); err != nil {
				return err
			}
			mu.Lock()
			snapshot.Keywords = out
			mu.Unlock()
			return nil
		}},
		{name: "posts", run: func(ctx context.Context) error {
			var out PostsResponse
			if err := c.getJSON(ctx, "/api/v1/analytics/posts", map[string]string{"campaignId": campaignID, "sort": "engagement", "limit": "12", "offset": "0"}, &out); err != nil {
				return err
			}
			mu.Lock()
			snapshot.Posts = out
			mu.Unlock()
			return nil
		}},
	}

	var wg sync.WaitGroup
	for _, req := range requests {
		req := req
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := req.run(ctx); err != nil {
				mu.Lock()
				snapshot.Errors = append(snapshot.Errors, fmt.Sprintf("%s: %v", req.name, err))
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if !snapshot.HasData() && len(snapshot.Errors) > 0 {
		return snapshot, fmt.Errorf("analytics snapshot unavailable: %s", strings.Join(snapshot.Errors, "; "))
	}
	if snapshot.HasData() {
		ttl := c.cacheTTL
		if len(snapshot.Errors) > 0 || !snapshot.HasCoreAnalytics() {
			ttl = 5 * time.Second
		}
		c.saveCachedSnapshot(campaignID, snapshot, ttl)
	}
	return snapshot, nil
}

func (c *implClient) getCachedSnapshot(campaignID string) (Snapshot, bool) {
	c.mu.RLock()
	item, ok := c.cache[campaignID]
	c.mu.RUnlock()
	if !ok || time.Now().After(item.expiresAt) {
		if ok {
			c.mu.Lock()
			delete(c.cache, campaignID)
			c.mu.Unlock()
		}
		return Snapshot{}, false
	}
	return item.value, true
}

func (c *implClient) saveCachedSnapshot(campaignID string, snapshot Snapshot, ttl time.Duration) {
	if ttl <= 0 {
		ttl = c.cacheTTL
	}
	c.mu.Lock()
	c.cache[campaignID] = cachedSnapshot{
		value:     snapshot,
		expiresAt: time.Now().Add(ttl),
	}
	c.mu.Unlock()
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
