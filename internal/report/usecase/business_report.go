package usecase

import (
	"fmt"
	"knowledge-srv/internal/report"
	"knowledge-srv/internal/search"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	defaultBusinessEvidenceLimit = 24
	maxEvidenceContentRunes      = 420
)

type businessPromptData struct {
	TotalDocs      int
	Aggregation    string
	Evidence       string
	Sections       string
	CompetitorURLs string
}

type businessEvidence struct {
	ID          string
	SearchID    string
	Platform    string
	Sentiment   string
	Author      string
	URL         string
	PostedAt    string
	Content     string
	Engagement  report.ReportPostEngagement
	Keywords    []string
	Aspects     []string
	RankScore   float64
	SearchScore float64
}

func buildBusinessEvidencePack(results []search.SearchResult, limit int) []businessEvidence {
	if limit <= 0 {
		limit = defaultBusinessEvidenceLimit
	}
	if limit > defaultBusinessEvidenceLimit {
		limit = defaultBusinessEvidenceLimit
	}

	items := make([]businessEvidence, 0, len(results))
	seen := make(map[string]struct{}, len(results))
	for _, result := range results {
		content := strings.TrimSpace(result.Content)
		if content == "" {
			continue
		}
		key := compactEvidenceFingerprint(content)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		ev := businessEvidenceFromSearchResult(result)
		ev.RankScore = businessEvidenceRankScore(result, ev)
		items = append(items, ev)
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].RankScore == items[j].RankScore {
			return items[i].SearchScore > items[j].SearchScore
		}
		return items[i].RankScore > items[j].RankScore
	})

	if len(items) > limit {
		items = items[:limit]
	}
	for i := range items {
		items[i].ID = fmt.Sprintf("E%02d", i+1)
	}
	return items
}

func businessEvidenceFromSearchResult(result search.SearchResult) businessEvidence {
	metadata := result.Metadata
	engagement := report.ReportPostEngagement{
		Likes:    intFromPayload(metadata, "likes"),
		Comments: intFromPayload(metadata, "comments"),
		Shares:   intFromPayload(metadata, "shares"),
		Views:    intFromPayload(metadata, "views"),
	}
	if engagement.Likes == 0 && engagement.Comments == 0 && engagement.Shares == 0 && engagement.Views == 0 {
		engagement = report.ReportPostEngagement{
			Likes:    intFromNestedPayload(metadata, "metadata", "engagement", "likes"),
			Comments: intFromNestedPayload(metadata, "metadata", "engagement", "comments"),
			Shares:   intFromNestedPayload(metadata, "metadata", "engagement", "shares"),
			Views:    intFromNestedPayload(metadata, "metadata", "engagement", "views"),
		}
	}

	postedAt := ""
	if result.ContentCreatedAt > 0 {
		postedAt = time.Unix(result.ContentCreatedAt, 0).Format("02/01/2006 15:04")
	} else if raw := stringFromPayload(metadata, "published_at"); raw != "" {
		postedAt = raw
	}

	return businessEvidence{
		SearchID:    result.ID,
		Platform:    strings.ToLower(result.Platform),
		Sentiment:   normalizeSentiment(result.OverallSentiment),
		Author:      authorFromMetadata(metadata),
		URL:         sourceURLFromMetadata(metadata),
		PostedAt:    postedAt,
		Content:     strings.TrimSpace(result.Content),
		Engagement:  engagement,
		Keywords:    result.Keywords,
		Aspects:     evidenceAspects(result.Aspects),
		SearchScore: result.Score,
	}
}

func businessEvidenceRankScore(result search.SearchResult, ev businessEvidence) float64 {
	engagement := float64(ev.Engagement.Likes) +
		float64(ev.Engagement.Comments*2) +
		float64(ev.Engagement.Shares*3) +
		float64(ev.Engagement.Views)*0.02
	score := result.Score*4 +
		math.Log10(engagement+1)*0.9 +
		result.EngagementScore*1.4 +
		floatFromPayload(result.Metadata, "business_relevance_score")*1.6 +
		floatFromNestedPayload(result.Metadata, "metadata", "business_relevance_score")*1.6 +
		minFloat(float64(len([]rune(ev.Content)))/240, 1)*0.35

	switch ev.Sentiment {
	case "negative":
		score += 0.35
	case "positive":
		score += 0.18
	}
	if ev.URL != "" {
		score += 0.12
	}
	if len(ev.Aspects) > 0 {
		score += 0.25
	}
	return score
}

func buildBusinessReportPrompt(input report.GenerateInput, data businessPromptData) string {
	return fmt.Sprintf(`Bạn là senior marketing intelligence analyst của SMAP, viết báo cáo cho người dùng marketing/business.

Nhiệm vụ: biến dữ liệu social listening thành một brief ra quyết định, không phải danh sách post. Báo cáo phải sâu, rõ trade-off, có hành động marketing/ops cụ thể và chỉ dùng bằng chứng được cung cấp.

Yêu cầu bắt buộc:
- Viết bằng tiếng Việt tự nhiên, sắc gọn, phục vụ business stakeholder.
- Không thêm H1/title đầu báo cáo; hệ thống sẽ tự thêm.
- Mỗi nhận định quan trọng, rủi ro, hoặc khuyến nghị phải cite evidence ID dạng [E01], [E02].
- Không bịa số liệu, không suy diễn sự kiện ngoài evidence. Khi chưa đủ chắc, ghi rõ "chưa đủ bằng chứng".
- Không copy nguyên văn hàng loạt post. Chỉ trích ý chính và dùng evidence ID.
- Ưu tiên insight thực dụng: tác động đến acquisition, conversion, retention, trust, brand safety, cost-to-serve.
- Nếu người dùng yêu cầu section riêng, lồng ý đó vào cấu trúc bên dưới thay vì phá cấu trúc.

Cấu trúc markdown bắt buộc:
## Executive Verdict
- 3-5 bullet có luận điểm chính, mức độ chắc chắn, và vì sao quan trọng với marketing.

## Business Impact
- Chuyển tín hiệu thành ảnh hưởng đến funnel/thương hiệu/vận hành.
- Nêu nhóm rủi ro lớn nhất và nhóm cơ hội nếu có.

## Customer Pain Drivers
- Gom thành 3-6 driver, mỗi driver có bằng chứng và hành động xử lý.

## Platform Strategy
- Nói rõ nền tảng nào đang tạo tín hiệu gì, nên ưu tiên kênh nào để listen/respond/content.

## Recommended Actions
- Bảng 5-8 hành động gồm: priority, owner gợi ý, action, evidence, KPI cần theo dõi.

## Watchouts & Confidence
- Nêu giới hạn dữ liệu, bias nguồn, missing source links nếu có, và các giả thuyết cần kiểm chứng.

Campaign ID: %s
Loại báo cáo: %s
Yêu cầu người dùng: %s
Sections người dùng chọn: %s
Nguồn/đối thủ tham chiếu: %s
Tổng số evidence có thể truy xuất: %d

### Dữ liệu tổng hợp
%s

### Evidence pack
%s

Chỉ trả về markdown body theo cấu trúc bắt buộc.`, input.CampaignID, input.ReportType, emptyAsDash(input.Filters.Prompt), emptyAsDash(data.Sections), emptyAsDash(data.CompetitorURLs), data.TotalDocs, data.Aggregation, data.Evidence)
}

func formatBusinessEvidenceForPrompt(evidence []businessEvidence) string {
	if len(evidence) == 0 {
		return "(Không có evidence đủ chất lượng)"
	}

	var sb strings.Builder
	for _, ev := range evidence {
		source := "missing"
		if ev.URL != "" {
			source = ev.URL
		}
		sb.WriteString(fmt.Sprintf("- [%s] platform=%s sentiment=%s author=%s time=%s engagement=%d source=%s\n",
			ev.ID, emptyAsDash(ev.Platform), emptyAsDash(ev.Sentiment), emptyAsDash(ev.Author), emptyAsDash(ev.PostedAt), engagementTotal(ev.Engagement), source))
		if len(ev.Aspects) > 0 {
			sb.WriteString(fmt.Sprintf("  aspects=%s\n", strings.Join(ev.Aspects, ", ")))
		}
		if len(ev.Keywords) > 0 {
			sb.WriteString(fmt.Sprintf("  keywords=%s\n", strings.Join(limitStrings(ev.Keywords, 8), ", ")))
		}
		sb.WriteString(fmt.Sprintf("  content=%q\n", truncateRunes(ev.Content, maxEvidenceContentRunes)))
	}
	return sb.String()
}

func compileBusinessMarkdown(input report.GenerateInput, body string, evidence []businessEvidence, totalDocs int) string {
	var sb strings.Builder
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = fmt.Sprintf("Campaign intelligence report · %s", time.Now().Format("2/1/2006"))
	}

	sb.WriteString(fmt.Sprintf("# %s\n\n", title))
	sb.WriteString(fmt.Sprintf("**Campaign ID:** %s\n\n", input.CampaignID))
	sb.WriteString(fmt.Sprintf("**Loại báo cáo:** %s\n\n", input.ReportType))
	if len(input.Filters.Sections) > 0 {
		sb.WriteString(fmt.Sprintf("**Sections yêu cầu:** %s\n\n", strings.Join(input.Filters.Sections, ", ")))
	}
	if strings.TrimSpace(input.Filters.Prompt) != "" {
		sb.WriteString(fmt.Sprintf("**Yêu cầu:** %s\n\n", input.Filters.Prompt))
	}
	sb.WriteString(fmt.Sprintf("**Tổng số documents phân tích:** %d\n\n", totalDocs))
	sb.WriteString(fmt.Sprintf("**Evidence nổi bật:** %d\n\n", len(evidence)))
	sb.WriteString(fmt.Sprintf("**Thời gian tạo:** %s\n\n", time.Now().Format("02/01/2006 15:04")))
	sb.WriteString("---\n\n")
	sb.WriteString(strings.TrimSpace(body))
	sb.WriteString("\n\n---\n\n")
	sb.WriteString(formatEvidenceAppendix(evidence))
	sb.WriteString("\n\n*Báo cáo được tạo tự động bởi SMAP Knowledge Service từ dữ liệu đã index trong campaign.*\n")
	return sb.String()
}

func formatEvidenceAppendix(evidence []businessEvidence) string {
	if len(evidence) == 0 {
		return "## Evidence Appendix\n\nChưa có evidence đủ chất lượng để đính kèm.\n"
	}

	var sb strings.Builder
	sb.WriteString("## Evidence Appendix\n\n")
	sb.WriteString("Các nguồn dưới đây là evidence đã được dùng để tạo báo cáo. Source chỉ mở được khi crawler/index nhận được permalink gốc từ nền tảng.\n\n")
	for _, ev := range evidence {
		source := "source unavailable"
		if ev.URL != "" {
			source = fmt.Sprintf("[source](%s)", ev.URL)
		}
		sb.WriteString(fmt.Sprintf("- [%s] %s · %s · %s · engagement %d · %s\n",
			ev.ID, displayPlatform(ev.Platform), ev.Sentiment, emptyAsDash(ev.Author), engagementTotal(ev.Engagement), source))
		sb.WriteString(fmt.Sprintf("  - %s\n", truncateRunes(ev.Content, 260)))
	}
	return sb.String()
}

func normalizeBusinessReportMarkdown(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```markdown")
	content = strings.TrimPrefix(content, "```md")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	lines := strings.Split(content, "\n")
	for len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "# ") {
		lines = lines[1:]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func countBusinessSections(content string) int {
	count := 0
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "## ") {
			count++
		}
	}
	if count == 0 {
		return 1
	}
	return count
}

func authorFromMetadata(metadata map[string]interface{}) string {
	return firstNonEmpty(
		stringFromNestedPayload(metadata, "metadata", "author_display_name"),
		stringFromNestedPayload(metadata, "metadata", "author_username"),
		stringFromNestedPayload(metadata, "metadata", "author"),
		stringFromPayload(metadata, "author_display_name"),
		stringFromPayload(metadata, "author_username"),
		stringFromPayload(metadata, "author"),
		"Unknown",
	)
}

func displayPlatform(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "youtube":
		return "YouTube"
	case "tiktok":
		return "TikTok"
	case "facebook":
		return "Facebook"
	default:
		return emptyAsDash(platform)
	}
}

func sourceURLFromMetadata(metadata map[string]interface{}) string {
	return firstURL(
		stringFromNestedPayload(metadata, "metadata", "comment_url"),
		stringFromNestedPayload(metadata, "metadata", "original_url"),
		stringFromNestedPayload(metadata, "metadata", "post_url"),
		stringFromNestedPayload(metadata, "metadata", "permalink"),
		stringFromNestedPayload(metadata, "metadata", "url"),
		stringFromNestedPayload(metadata, "metadata", "source_url"),
		stringFromNestedPayload(metadata, "metadata", "web_url"),
		stringFromNestedPayload(metadata, "metadata", "parent_post_url"),
		stringFromNestedPayload(metadata, "metadata", "video_url"),
		stringFromPayloadPath(metadata, "metadata", "platform_meta", "youtube", "parent_url"),
		stringFromPayloadPath(metadata, "metadata", "platform_meta", "youtube", "video_url"),
		stringFromPayload(metadata, "comment_url"),
		stringFromPayload(metadata, "original_url"),
		stringFromPayload(metadata, "post_url"),
		stringFromPayload(metadata, "permalink"),
		stringFromPayload(metadata, "url"),
		stringFromPayload(metadata, "source_url"),
		stringFromPayload(metadata, "web_url"),
		stringFromPayload(metadata, "parent_post_url"),
		stringFromPayload(metadata, "video_url"),
		stringFromPayloadPath(metadata, "platform_meta", "youtube", "parent_url"),
		stringFromPayloadPath(metadata, "platform_meta", "youtube", "video_url"),
		derivedSourceURLFromIDs(metadata),
	)
}

func derivedSourceURLFromIDs(metadata map[string]interface{}) string {
	id := firstNonEmpty(
		stringFromPayload(metadata, "uap_id"),
		stringFromPayload(metadata, "source_id"),
		stringFromNestedPayload(metadata, "metadata", "root_id"),
		stringFromNestedPayload(metadata, "metadata", "parent_id"),
		stringFromPayload(metadata, "root_id"),
		stringFromPayload(metadata, "parent_id"),
	)
	id = strings.TrimSpace(id)
	switch {
	case strings.HasPrefix(id, "yt_p_"):
		return "https://www.youtube.com/watch?v=" + strings.TrimPrefix(id, "yt_p_")
	case strings.HasPrefix(id, "yt_v_"):
		return "https://www.youtube.com/watch?v=" + strings.TrimPrefix(id, "yt_v_")
	default:
		return ""
	}
}

func evidenceAspects(aspects []search.AspectResult) []string {
	out := make([]string, 0, len(aspects))
	for _, aspect := range aspects {
		name := firstNonEmpty(aspect.AspectDisplayName, aspect.Aspect)
		if name == "" {
			continue
		}
		if aspect.Sentiment != "" {
			out = append(out, fmt.Sprintf("%s(%s)", name, normalizeSentiment(aspect.Sentiment)))
		} else {
			out = append(out, name)
		}
	}
	return out
}

func engagementTotal(engagement report.ReportPostEngagement) int {
	return engagement.Likes + engagement.Comments + engagement.Shares + engagement.Views
}

func compactEvidenceFingerprint(content string) string {
	content = strings.ToLower(strings.TrimSpace(content))
	content = strings.Join(strings.Fields(content), " ")
	return truncateRunes(content, 160)
}

func truncateRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max]) + "..."
}

func limitStrings(values []string, max int) []string {
	if max <= 0 || len(values) <= max {
		return values
	}
	return values[:max]
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func floatFromPayload(payload map[string]interface{}, key string) float64 {
	if payload == nil {
		return 0
	}
	switch v := payload[key].(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case jsonNumber:
		f, _ := v.Float64()
		return f
	default:
		return 0
	}
}

func floatFromNestedPayload(payload map[string]interface{}, keys ...string) float64 {
	var current interface{} = payload
	for _, key := range keys {
		obj, ok := current.(map[string]interface{})
		if !ok {
			return 0
		}
		current = obj[key]
	}
	switch v := current.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case jsonNumber:
		f, _ := v.Float64()
		return f
	default:
		return 0
	}
}

func stringFromPayloadPath(payload map[string]interface{}, keys ...string) string {
	var current interface{} = payload
	for _, key := range keys {
		obj, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current = obj[key]
	}
	if s, ok := current.(string); ok {
		return s
	}
	return ""
}

type jsonNumber interface {
	Float64() (float64, error)
}
