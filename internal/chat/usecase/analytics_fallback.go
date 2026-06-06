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
	if isContrastAnalyticsQuestion(question) {
		return buildContrastAnalyticsAnswer(question, snapshot)
	}
	if isSentimentDriverAnalyticsQuestion(question) {
		return buildSentimentDriverAnalyticsAnswer(question, snapshot)
	}

	platforms := sortedPlatformStats(snapshot.Platforms.Stats)
	totalMentions := metricFormatted(snapshot.KPIs.Metrics, "Total Mentions", totalDocsFromSnapshot(snapshot))
	engagementFallback := totalEngagement(platforms)
	if engagementFallback == 0 {
		engagementFallback = totalPostEngagement(snapshot.Posts.Posts)
	}
	engagement := metricFormatted(snapshot.KPIs.Metrics, "Engagement", engagementFallback)
	negativeShare := sentimentShare(snapshot.Sentiment.Donut, "negative")
	positiveShare := sentimentShare(snapshot.Sentiment.Donut, "positive")
	summaryMode := isCampaignSummaryQuestion(question)

	var lines []string
	if summaryMode {
		lines = append(lines, "Insight chính của campaign:")
	}
	intro := fmt.Sprintf("Mình đang dựa trên analytics live của campaign: %s mentions, engagement %s.", totalMentions, engagement)
	if sentimentScore, ok := metricFormattedIfPresent(snapshot.KPIs.Metrics, "Sentiment Score"); ok {
		intro = fmt.Sprintf("Mình đang dựa trên analytics live của campaign: %s mentions, sentiment trung bình %s, engagement %s.", totalMentions, sentimentScore, engagement)
	}
	lines = append(lines, intro)
	if len(snapshot.Errors) > 0 && !snapshot.HasCoreAnalytics() {
		lines = append(lines, "Một phần endpoint analytics chưa đủ dữ liệu trong snapshot này, nên các kết luận định lượng bên dưới chỉ dùng những phần đã lấy được.")
	}
	if snapshot.Sentiment.Total > 0 {
		lines = append(lines, fmt.Sprintf("Cơ cấu cảm xúc hiện tại: negative %.1f%%, neutral %.1f%%, positive %.1f%%.", negativeShare, sentimentShare(snapshot.Sentiment.Donut, "neutral"), positiveShare))
	}

	if len(platforms) > 0 {
		lines = append(lines, "")
		if summaryMode {
			lines = append(lines, "Các điểm cần chú ý:")
		} else {
			lines = append(lines, "So sánh theo nền tảng:")
		}
		for _, p := range platforms {
			lines = append(lines, fmt.Sprintf("- %s: %s mentions, sentiment %.1f%%, engagement %s.", platformLabel(p), formatInt(p.Mentions), p.Sentiment, formatInt(p.EngagementRaw)))
		}

		if mostNegative, ok := mostNegativePlatform(platforms); ok {
			lines = append(lines, "")
			if summaryMode {
				lines = append(lines, fmt.Sprintf("Ưu tiên marketing: xử lý %s trước vì đây là kênh có sentiment thấp nhất trong các kênh có dữ liệu (%.1f%%).", platformLabel(mostNegative), mostNegative.Sentiment))
			} else {
				lines = append(lines, fmt.Sprintf("Nền tảng cần ưu tiên xử lý nhất là %s vì sentiment đang thấp nhất trong các kênh có dữ liệu (%.1f%%).", platformLabel(mostNegative), mostNegative.Sentiment))
			}
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
	if summaryMode {
		lines = append(lines, "Hành động đề xuất: ưu tiên phản hồi nhóm chủ đề tiêu cực có tần suất cao, viết lại FAQ theo các keyword đang tăng, và theo dõi riêng kênh có sentiment thấp để giảm lan truyền rủi ro thương hiệu.")
	} else {
		lines = append(lines, "Gợi ý marketing: tách thông điệp phản hồi theo từng nền tảng, ưu tiên xử lý kênh sentiment thấp nhất trước, rồi biến các keyword đang lên thành checklist nội dung/FAQ để giảm lặp lại phản hồi tiêu cực.")
	}

	citations := citationsFromPosts(snapshot.Posts.Posts, 5)
	suggestions := []string{
		"Đào sâu lý do sentiment thấp nhất theo nền tảng",
		"Tạo report về các chủ đề tiêu cực nổi bật",
		"So sánh engagement và sentiment trong 7 ngày gần nhất",
	}

	return strings.Join(lines, "\n"), citations, suggestions, len(citations)
}

// buildContrastAnalyticsAnswer explains why a campaign can look split by
// combining sentiment distribution, platform deltas, and representative posts.
func buildContrastAnalyticsAnswer(question string, snapshot analyticspkg.Snapshot) (string, []chat.Citation, []string, int) {
	platforms := sortedPlatformStats(snapshot.Platforms.Stats)
	totalMentions := metricFormatted(snapshot.KPIs.Metrics, "Total Mentions", totalDocsFromSnapshot(snapshot))
	engagement := metricFormatted(snapshot.KPIs.Metrics, "Engagement", totalEngagement(platforms))
	positiveShare := sentimentShare(snapshot.Sentiment.Donut, "positive")
	neutralShare := sentimentShare(snapshot.Sentiment.Donut, "neutral")
	negativeShare := sentimentShare(snapshot.Sentiment.Donut, "negative")
	positivePosts := samplePosts(snapshot.Posts.Posts, "positive", 3)
	negativePosts := samplePosts(snapshot.Posts.Posts, "negative", 3)
	selectedPosts := append([]analyticspkg.PostItem{}, negativePosts...)
	selectedPosts = append(selectedPosts, positivePosts...)

	lines := []string{
		fmt.Sprintf("Có sự trái chiều vì campaign đang có đủ ba nhóm phản hồi cùng tồn tại, không phải một chiều: %s mentions, engagement %s.", totalMentions, engagement),
	}
	if snapshot.Sentiment.Total > 0 {
		lines = append(lines, fmt.Sprintf("Cơ cấu hiện tại: negative %.1f%%, neutral %.1f%%, positive %.1f%%. Khoảng cách positive-negative là %.1f điểm phần trăm, nên chỉ cần vài cụm nội dung lớn là cảm nhận campaign sẽ đổi hướng.", negativeShare, neutralShare, positiveShare, positiveShare-negativeShare))
	}
	if len(platforms) > 0 {
		lines = append(lines, "")
		lines = append(lines, "Khác biệt theo nền tảng:")
		for _, p := range platforms {
			lines = append(lines, fmt.Sprintf("- %s: %s mentions, sentiment %.1f%%, engagement %s.", platformLabel(p), formatInt(p.Mentions), p.Sentiment, formatInt(p.EngagementRaw)))
		}
	}

	if drivers := topKeywordsFromPosts(negativePosts, 5); len(drivers) > 0 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Cụm kéo tiêu cực trong mẫu đang nằm ở: %s.", strings.Join(drivers, ", ")))
	} else if drivers := topKeywords(snapshot.Keywords.Keywords, 5); len(drivers) > 0 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Các chủ đề lớn đang chi phối cuộc thảo luận: %s.", strings.Join(drivers, ", ")))
	}

	appendPostEvidence(&lines, "Mẫu phản hồi tiêu cực:", negativePosts, 2)
	appendPostEvidence(&lines, "Mẫu phản hồi tích cực:", positivePosts, 2)

	lines = append(lines, "")
	lines = append(lines, "Kết luận demo: campaign không bị một chiều hoàn toàn. Phần tích cực thường đến từ nội dung viral/brand/ủng hộ thương hiệu; phần tiêu cực tập trung vào trải nghiệm vận hành như tài xế, shipper, phí, app, đơn hàng hoặc kỳ vọng dịch vụ. Vì vậy khi nhìn tổng sentiment gần cân bằng, nên tách câu chuyện theo nền tảng và chủ đề thay vì đọc một chỉ số trung bình.")

	citations := citationsFromSelectedPosts(selectedPosts, 5)
	suggestions := []string{
		"Đào sâu lý do sentiment thấp nhất theo nền tảng",
		"So sánh các chủ đề tích cực và tiêu cực",
		"Tạo checklist phản hồi cho nhóm chủ đề tiêu cực",
	}
	return strings.Join(lines, "\n"), citations, suggestions, len(citations)
}

// buildSentimentDriverAnalyticsAnswer explains low sentiment using the weakest
// platform and negative high-engagement examples from the analytics snapshot.
func buildSentimentDriverAnalyticsAnswer(question string, snapshot analyticspkg.Snapshot) (string, []chat.Citation, []string, int) {
	platforms := sortedPlatformStats(snapshot.Platforms.Stats)
	totalMentions := metricFormatted(snapshot.KPIs.Metrics, "Total Mentions", totalDocsFromSnapshot(snapshot))
	negativePosts := samplePosts(snapshot.Posts.Posts, "negative", 5)
	selectedPosts := append([]analyticspkg.PostItem{}, negativePosts...)

	lines := []string{
		fmt.Sprintf("Mình đào sâu bằng analytics live của campaign: %s mentions, tập trung vào nền tảng có sentiment thấp và các mẫu negative engagement cao.", totalMentions),
	}
	if snapshot.Sentiment.Total > 0 {
		lines = append(lines, fmt.Sprintf("Cơ cấu cảm xúc: negative %.1f%%, neutral %.1f%%, positive %.1f%%.", sentimentShare(snapshot.Sentiment.Donut, "negative"), sentimentShare(snapshot.Sentiment.Donut, "neutral"), sentimentShare(snapshot.Sentiment.Donut, "positive")))
	}

	if mostNegative, ok := mostNegativePlatform(platforms); ok {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Nền tảng kéo sentiment thấp nhất hiện là %s: sentiment %.1f%% trên %s mentions.", platformLabel(mostNegative), mostNegative.Sentiment, formatInt(mostNegative.Mentions)))
	}

	if len(platforms) > 0 {
		lines = append(lines, "")
		lines = append(lines, "Theo từng nền tảng:")
		for _, p := range platforms {
			platformPosts := samplePostsForPlatform(snapshot.Posts.Posts, "negative", p.Platform, 1)
			if len(platformPosts) == 0 {
				lines = append(lines, fmt.Sprintf("- %s: sentiment %.1f%%, %s mentions, engagement %s. Snapshot top posts chưa có mẫu negative đủ sạch cho kênh này.", platformLabel(p), p.Sentiment, formatInt(p.Mentions), formatInt(p.EngagementRaw)))
				continue
			}
			selectedPosts = append(selectedPosts, platformPosts[0])
			lines = append(lines, fmt.Sprintf("- %s: sentiment %.1f%%, %s mentions, engagement %s. Mẫu kéo âm: %s", platformLabel(p), p.Sentiment, formatInt(p.Mentions), formatInt(p.EngagementRaw), trimRunes(platformPosts[0].Content, 120)))
		}
	}

	if drivers := topKeywordsFromPosts(negativePosts, 6); len(drivers) > 0 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Các cụm nguyên nhân trong mẫu negative: %s.", strings.Join(drivers, ", ")))
	} else if drivers := topKeywords(snapshot.Keywords.Keywords, 6); len(drivers) > 0 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Các keyword lớn cần soi tiếp: %s.", strings.Join(drivers, ", ")))
	}

	appendPostEvidence(&lines, "Bằng chứng negative nổi bật:", negativePosts, 3)

	lines = append(lines, "")
	lines = append(lines, "Khuyến nghị xử lý: ưu tiên kênh có sentiment thấp nhất, gom các phản hồi theo nhóm vận hành như shipper/tài xế/phí/app/đơn hàng, rồi phản hồi bằng FAQ hoặc nội dung giải thích theo từng nền tảng. Không nên chỉ nhìn tổng sentiment vì neutral và positive đang còn khá lớn.")

	citations := citationsFromSelectedPosts(selectedPosts, 5)
	suggestions := []string{
		"So sánh phản hồi tiêu cực theo keyword",
		"Lọc riêng các phản hồi về tài xế và shipper",
		"Tạo gợi ý phản hồi marketing cho Facebook",
	}
	return strings.Join(lines, "\n"), citations, suggestions, len(citations)
}

// isContrastAnalyticsQuestion detects "why are evaluations mixed" style
// questions that should use aggregate sentiment plus examples.
func isContrastAnalyticsQuestion(question string) bool {
	q := strings.ToLower(strings.TrimSpace(question))
	signals := []string{"trái chiều", "trai chieu", "mâu thuẫn", "mau thuan", "đối lập", "doi lap"}
	for _, signal := range signals {
		if strings.Contains(q, signal) {
			return true
		}
	}
	return (strings.Contains(q, "tại sao") || strings.Contains(q, "vì sao")) &&
		(strings.Contains(q, "đánh giá") || strings.Contains(q, "campaign") || strings.Contains(q, "chiến dịch"))
}

// isSentimentDriverAnalyticsQuestion detects sentiment root-cause questions
// that should not fall through to the generic no-context answer.
func isSentimentDriverAnalyticsQuestion(question string) bool {
	q := strings.ToLower(strings.TrimSpace(question))
	signals := []string{
		"sentiment thấp", "sentiment thap", "cảm xúc thấp", "cam xuc thap",
		"đào sâu lý do", "dao sau ly do", "lý do sentiment", "ly do sentiment",
		"nguyên nhân sentiment", "nguyen nhan sentiment",
	}
	for _, signal := range signals {
		if strings.Contains(q, signal) {
			return true
		}
	}
	return (strings.Contains(q, "tại sao") || strings.Contains(q, "vì sao")) && strings.Contains(q, "sentiment")
}

func isCampaignSummaryQuestion(question string) bool {
	q := strings.ToLower(strings.TrimSpace(question))
	if q == "" {
		return false
	}
	signals := []string{
		"tóm tắt", "tom tat", "tổng quan", "tong quan", "insight chính", "insight chinh",
		"điểm chính", "diem chinh", "kết luận chính", "ket luan chinh", "campaign summary",
		"main insight", "summary", "overview",
	}
	for _, signal := range signals {
		if strings.Contains(q, signal) {
			return true
		}
	}
	return false
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

func samplePostsForPlatform(posts []analyticspkg.PostItem, sentiment string, platform string, limit int) []analyticspkg.PostItem {
	platform = strings.ToLower(strings.TrimSpace(platform))
	out := make([]analyticspkg.PostItem, 0, limit)
	for _, post := range posts {
		if platform != "" && strings.ToLower(strings.TrimSpace(post.Platform)) != platform {
			continue
		}
		if sentiment != "" && !strings.EqualFold(post.Sentiment, sentiment) {
			continue
		}
		content := strings.TrimSpace(post.Content)
		if content == "" || contentquality.IsLowValueMarketingContent(content) {
			continue
		}
		out = append(out, post)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func topKeywordsFromPosts(posts []analyticspkg.PostItem, limit int) []string {
	counts := make(map[string]int)
	for _, post := range posts {
		for _, raw := range post.Keywords {
			keyword := strings.ToLower(strings.TrimSpace(raw))
			if keyword == "" {
				continue
			}
			counts[keyword]++
		}
	}
	type keywordCount struct {
		text  string
		count int
	}
	ranked := make([]keywordCount, 0, len(counts))
	for text, count := range counts {
		ranked = append(ranked, keywordCount{text: text, count: count})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].count == ranked[j].count {
			return ranked[i].text < ranked[j].text
		}
		return ranked[i].count > ranked[j].count
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	out := make([]string, 0, len(ranked))
	for _, item := range ranked {
		out = append(out, fmt.Sprintf("%s (%d)", item.text, item.count))
	}
	return out
}

func appendPostEvidence(lines *[]string, title string, posts []analyticspkg.PostItem, limit int) {
	if len(posts) == 0 {
		return
	}
	*lines = append(*lines, "")
	*lines = append(*lines, title)
	if len(posts) > limit {
		posts = posts[:limit]
	}
	for _, post := range posts {
		*lines = append(*lines, fmt.Sprintf("- %s · %s: %s", platformLabelName(post.Platform), authorLabel(post), trimRunes(post.Content, 150)))
	}
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

func citationsFromSelectedPosts(posts []analyticspkg.PostItem, limit int) []chat.Citation {
	citations := make([]chat.Citation, 0, limit)
	seen := make(map[string]struct{}, len(posts))
	for _, post := range posts {
		id := strings.TrimSpace(post.ID)
		if id == "" {
			id = strings.TrimSpace(post.Platform) + ":" + trimRunes(post.Content, 80)
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		content := strings.TrimSpace(post.Content)
		if content == "" || contentquality.IsLowValueMarketingContent(content) {
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
		normalized := strings.ToLower(platform)
		return strings.ToUpper(normalized[:1]) + normalized[1:]
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
