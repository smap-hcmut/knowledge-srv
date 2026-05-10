package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/chat/repository"
	"knowledge-srv/internal/contentquality"
	"knowledge-srv/internal/model"
	analyticspkg "knowledge-srv/pkg/analytics"
)

func (uc *implUseCase) tryAnalyticsFallback(
	ctx context.Context,
	conversation model.Conversation,
	input chat.ChatInput,
	startTime time.Time,
	intent QueryIntent,
	snapshot *analyticspkg.Snapshot,
) (chat.ChatOutput, bool) {
	if snapshot == nil || !snapshot.HasData() {
		return chat.ChatOutput{}, false
	}

	answer, citations, suggestions, docsUsed := buildAnalyticsAnswer(input.Message, *snapshot)
	searchMeta := chat.SearchMeta{
		TotalDocsSearched: int(totalDocsFromSnapshot(*snapshot)),
		DocsUsed:          docsUsed,
		ProcessingTimeMs:  time.Since(startTime).Milliseconds(),
		ModelUsed:         "analysis-api",
	}

	uc.persistChatExchange(ctx, conversation, input, answer, citations, suggestions, searchMeta)

	return chat.ChatOutput{
		ConversationID: conversation.ID,
		Answer:         answer,
		Citations:      citations,
		Suggestions:    suggestions,
		SearchMetadata: searchMeta,
		QueryIntent:    string(intent),
		Backend:        "AnalysisAPI",
	}, true
}

func (uc *implUseCase) loadAnalyticsSnapshot(ctx context.Context, campaignID string, timeout time.Duration) (*analyticspkg.Snapshot, bool) {
	if uc.analytics == nil {
		return nil, false
	}
	if timeout <= 0 {
		timeout = 4 * time.Second
	}
	snapshotCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	snapshot, err := uc.analytics.Snapshot(snapshotCtx, campaignID)
	if err != nil {
		uc.l.Warnf(ctx, "chat.usecase.loadAnalyticsSnapshot: analytics snapshot failed: %v", err)
		return nil, false
	}
	if !snapshot.HasData() {
		return nil, false
	}
	return &snapshot, true
}

func (uc *implUseCase) persistChatExchange(
	ctx context.Context,
	conversation model.Conversation,
	input chat.ChatInput,
	answer string,
	citations []chat.Citation,
	suggestions []string,
	searchMeta chat.SearchMeta,
) {
	filtersJSON, _ := json.Marshal(input.Filters)
	_, _ = uc.repo.CreateMessage(ctx, repository.CreateMessageOptions{
		ConversationID: conversation.ID,
		Role:           "user",
		Content:        input.Message,
		FiltersUsed:    filtersJSON,
	})

	citationsJSON, _ := json.Marshal(citations)
	suggestionsJSON, _ := json.Marshal(suggestions)
	searchMetaJSON, _ := json.Marshal(searchMeta)
	_, _ = uc.repo.CreateMessage(ctx, repository.CreateMessageOptions{
		ConversationID: conversation.ID,
		Role:           "assistant",
		Content:        answer,
		Citations:      citationsJSON,
		SearchMetadata: searchMetaJSON,
		Suggestions:    suggestionsJSON,
	})

	_ = uc.repo.UpdateConversationLastMessage(ctx, repository.UpdateLastMessageOptions{
		ConversationID: conversation.ID,
		MessageCount:   conversation.MessageCount + 2,
	})
}

func buildAnalyticsAnswer(question string, snapshot analyticspkg.Snapshot) (string, []chat.Citation, []string, int) {
	platforms := sortedPlatformStats(snapshot.Platforms.Stats)
	totalMentions := metricFormatted(snapshot.KPIs.Metrics, "Total Mentions", totalDocsFromSnapshot(snapshot))
	engagementFallback := totalEngagement(platforms)
	if engagementFallback == 0 {
		engagementFallback = totalPostEngagement(snapshot.Posts.Posts)
	}
	engagement := metricFormatted(snapshot.KPIs.Metrics, "Engagement", engagementFallback)
	negativeShare := sentimentShare(snapshot.Sentiment.Donut, "negative")
	positiveShare := sentimentShare(snapshot.Sentiment.Donut, "positive")

	var lines []string
	intro := fmt.Sprintf("Mình đang dựa trên analytics live của campaign: %s mentions, engagement %s.", totalMentions, engagement)
	if sentimentScore, ok := metricFormattedIfPresent(snapshot.KPIs.Metrics, "Sentiment Score"); ok {
		intro = fmt.Sprintf("Mình đang dựa trên analytics live của campaign: %s mentions, sentiment trung bình %s, engagement %s.", totalMentions, sentimentScore, engagement)
	}
	lines = append(lines, intro)
	if len(snapshot.Errors) > 0 || !snapshot.HasCoreAnalytics() {
		lines = append(lines, "Một phần endpoint analytics chưa đủ dữ liệu trong snapshot này, nên các kết luận định lượng bên dưới chỉ dùng những phần đã lấy được.")
	}
	if snapshot.Sentiment.Total > 0 {
		lines = append(lines, fmt.Sprintf("Cơ cấu cảm xúc hiện tại: negative %.1f%%, neutral %.1f%%, positive %.1f%%.", negativeShare, sentimentShare(snapshot.Sentiment.Donut, "neutral"), positiveShare))
	}

	if len(platforms) > 0 {
		lines = append(lines, "")
		lines = append(lines, "So sánh theo nền tảng:")
		for _, p := range platforms {
			lines = append(lines, fmt.Sprintf("- %s: %s mentions, sentiment %.1f%%, engagement %s.", platformLabel(p), formatInt(p.Mentions), p.Sentiment, formatInt(p.EngagementRaw)))
		}

		if mostNegative, ok := mostNegativePlatform(platforms); ok {
			lines = append(lines, "")
			lines = append(lines, fmt.Sprintf("Nền tảng cần ưu tiên xử lý nhất là %s vì sentiment đang thấp nhất trong các kênh có dữ liệu (%.1f%%).", platformLabel(mostNegative), mostNegative.Sentiment))
		}
	}

	drivers := topKeywords(snapshot.Keywords.Keywords, 5)
	if len(drivers) > 0 {
		lines = append(lines, fmt.Sprintf("Các chủ đề đang kéo câu chuyện nhiều nhất: %s.", strings.Join(drivers, ", ")))
	}

	negativePosts := samplePosts(snapshot.Posts.Posts, "negative", 3)
	if len(negativePosts) > 0 {
		lines = append(lines, "")
		lines = append(lines, "Một vài tín hiệu tiêu cực nổi bật:")
		for _, post := range negativePosts {
			lines = append(lines, fmt.Sprintf("- %s · %s: %s", platformLabelName(post.Platform), authorLabel(post), trimRunes(post.Content, 140)))
		}
	}

	lines = append(lines, "")
	lines = append(lines, "Gợi ý marketing: tách thông điệp phản hồi theo từng nền tảng, ưu tiên xử lý kênh sentiment thấp nhất trước, rồi biến các keyword đang lên thành checklist nội dung/FAQ để giảm lặp lại phản hồi tiêu cực.")

	citations := citationsFromPosts(snapshot.Posts.Posts, 5)
	suggestions := []string{
		"Đào sâu lý do sentiment thấp nhất theo nền tảng",
		"Tạo report về các chủ đề tiêu cực nổi bật",
		"So sánh engagement và sentiment trong 7 ngày gần nhất",
	}

	_ = question // reserved for future intent-specific phrasing.
	return strings.Join(lines, "\n"), citations, suggestions, len(citations)
}

func sortedPlatformStats(stats []analyticspkg.PlatformStat) []analyticspkg.PlatformStat {
	out := make([]analyticspkg.PlatformStat, 0, len(stats))
	for _, stat := range stats {
		if stat.Mentions <= 0 {
			continue
		}
		out = append(out, stat)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Mentions > out[j].Mentions
	})
	return out
}

func mostNegativePlatform(stats []analyticspkg.PlatformStat) (analyticspkg.PlatformStat, bool) {
	var selected analyticspkg.PlatformStat
	ok := false
	for _, stat := range stats {
		if stat.Mentions <= 0 {
			continue
		}
		if !ok || stat.Sentiment < selected.Sentiment {
			selected = stat
			ok = true
		}
	}
	return selected, ok
}

func totalDocsFromSnapshot(snapshot analyticspkg.Snapshot) int64 {
	if snapshot.Sentiment.Total > 0 {
		return snapshot.Sentiment.Total
	}
	if snapshot.Posts.Total > 0 {
		return snapshot.Posts.Total
	}
	var total int64
	for _, stat := range snapshot.Platforms.Stats {
		total += stat.Mentions
	}
	return total
}

func totalEngagement(platforms []analyticspkg.PlatformStat) int64 {
	var total int64
	for _, platform := range platforms {
		total += platform.EngagementRaw
	}
	return total
}

func totalPostEngagement(posts []analyticspkg.PostItem) int64 {
	var total int64
	for _, post := range posts {
		total += post.Engagement
	}
	return total
}

func metricFormatted(metrics []analyticspkg.KPIMetric, label string, fallback any) string {
	for _, metric := range metrics {
		if strings.EqualFold(metric.Label, label) {
			if strings.TrimSpace(metric.Formatted) != "" {
				return metric.Formatted
			}
			return formatFloat(metric.Value)
		}
	}
	switch v := fallback.(type) {
	case int64:
		return formatInt(v)
	case float64:
		return formatFloat(v)
	default:
		return fmt.Sprint(v)
	}
}

func metricFormattedIfPresent(metrics []analyticspkg.KPIMetric, label string) (string, bool) {
	for _, metric := range metrics {
		if strings.EqualFold(metric.Label, label) {
			if strings.TrimSpace(metric.Formatted) != "" {
				return metric.Formatted, true
			}
			return formatFloat(metric.Value), true
		}
	}
	return "", false
}

func sentimentShare(items []analyticspkg.SentimentItem, label string) float64 {
	var total int64
	var matched int64
	for _, item := range items {
		total += item.Value
		if strings.EqualFold(item.Label, label) {
			matched += item.Value
		}
	}
	if total == 0 {
		return 0
	}
	return float64(matched) / float64(total) * 100
}

func topKeywords(keywords []analyticspkg.KeywordItem, limit int) []string {
	out := make([]string, 0, limit)
	for _, keyword := range keywords {
		text := strings.TrimSpace(keyword.Text)
		if text == "" {
			continue
		}
		out = append(out, fmt.Sprintf("%s (%s)", text, formatInt(keyword.Volume)))
		if len(out) >= limit {
			break
		}
	}
	return out
}

func samplePosts(posts []analyticspkg.PostItem, sentiment string, limit int) []analyticspkg.PostItem {
	out := make([]analyticspkg.PostItem, 0, limit)
	for _, post := range posts {
		if sentiment != "" && !strings.EqualFold(post.Sentiment, sentiment) {
			continue
		}
		if strings.TrimSpace(post.Content) == "" {
			continue
		}
		if contentquality.IsLowValueMarketingContent(post.Content) {
			continue
		}
		out = append(out, post)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func citationsFromPosts(posts []analyticspkg.PostItem, limit int) []chat.Citation {
	citations := make([]chat.Citation, 0, limit)
	for _, post := range posts {
		content := strings.TrimSpace(post.Content)
		if content == "" {
			continue
		}
		if contentquality.IsLowValueMarketingContent(content) {
			continue
		}
		citations = append(citations, chat.Citation{
			ID:             post.ID,
			Content:        trimRunes(content, 200),
			RelevanceScore: float64(post.Engagement),
			Platform:       post.Platform,
			Sentiment:      post.Sentiment,
			URL:            post.URL,
		})
		if len(citations) >= limit {
			break
		}
	}
	return citations
}

func authorLabel(post analyticspkg.PostItem) string {
	if strings.TrimSpace(post.AuthorUsername) != "" {
		return "@" + strings.TrimPrefix(post.AuthorUsername, "@")
	}
	if strings.TrimSpace(post.Author) != "" {
		return post.Author
	}
	return "unknown author"
}

func platformLabel(stat analyticspkg.PlatformStat) string {
	if strings.TrimSpace(stat.Name) != "" {
		return stat.Name
	}
	return platformLabelName(stat.Platform)
}

func platformLabelName(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "tiktok":
		return "TikTok"
	case "facebook":
		return "Facebook"
	case "youtube":
		return "YouTube"
	default:
		if strings.TrimSpace(platform) == "" {
			return "Unknown"
		}
		return strings.Title(strings.ToLower(platform))
	}
}

func trimRunes(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "..."
}

func formatInt(value int64) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	switch {
	case value >= 1_000_000:
		return fmt.Sprintf("%s%.1fM", sign, float64(value)/1_000_000)
	case value >= 1_000:
		return fmt.Sprintf("%s%.1fK", sign, float64(value)/1_000)
	default:
		return fmt.Sprintf("%s%d", sign, value)
	}
}

func formatFloat(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", value), "0"), ".")
}
