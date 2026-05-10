package usecase

import (
	"fmt"
	"strings"

	"knowledge-srv/internal/report"
	"knowledge-srv/internal/search"
)

// promptData holds the context data injected into section prompts.
type promptData struct {
	CampaignID     string
	ReportType     string
	Samples        string
	TotalDocs      int
	Aggregation    string
	UserPrompt     string
	Sections       string
	CompetitorURLs string
}

// buildSectionPrompt injects context data into the section template prompt.
func buildSectionPrompt(tmpl report.SectionTemplate, data promptData) string {
	context := fmt.Sprintf(`**Campaign:** %s
**Loại báo cáo:** %s
**Tổng số documents:** %d
**Yêu cầu người dùng:** %s
**Sections mong muốn:** %s
**Nguồn/đối thủ tham chiếu:** %s

### Quy tắc bắt buộc:
- Chỉ dùng dữ liệu trong context này; không suy diễn sự kiện, đối thủ, chính sách hoặc số liệu ngoài dữ liệu.
- Nếu bằng chứng yếu hoặc thiếu, ghi rõ "chưa đủ bằng chứng" thay vì viết chắc chắn.
- Ưu tiên góc nhìn marketing: vấn đề khách hàng, insight thông điệp, kênh ưu tiên, hành động đề xuất.
- Khi kết luận, gắn với ví dụ trong phần mẫu phản hồi bằng platform/sentiment/nội dung.

### Dữ liệu tổng hợp:
%s

### Mẫu phản hồi tiêu biểu:
%s`,
		data.CampaignID,
		data.ReportType,
		data.TotalDocs,
		emptyAsDash(data.UserPrompt),
		emptyAsDash(data.Sections),
		emptyAsDash(data.CompetitorURLs),
		data.Aggregation,
		data.Samples,
	)

	return fmt.Sprintf(tmpl.Prompt, context)
}

func emptyAsDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

// formatSamples formats search results into a human-readable string for LLM prompts.
func formatSamples(samples []search.SearchResult) string {
	if len(samples) == 0 {
		return "(Không có dữ liệu mẫu)"
	}

	var sb strings.Builder
	for i, s := range samples {
		if i >= 20 { // Limit to 20 samples in prompts for token efficiency
			sb.WriteString(fmt.Sprintf("\n... và %d mẫu khác", len(samples)-20))
			break
		}

		sb.WriteString(fmt.Sprintf("\n**[%d]** Platform: %s | Sentiment: %s (%.2f)",
			i+1, s.Platform, s.OverallSentiment, s.SentimentScore))

		if len(s.Aspects) > 0 {
			aspects := make([]string, 0, len(s.Aspects))
			for _, a := range s.Aspects {
				aspects = append(aspects, fmt.Sprintf("%s(%s)", a.Aspect, a.Sentiment))
			}
			sb.WriteString(fmt.Sprintf(" | Aspects: %s", strings.Join(aspects, ", ")))
		}

		// Truncate content for prompt efficiency
		content := s.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		sb.WriteString(fmt.Sprintf("\nNội dung: %s\n", content))
	}

	return sb.String()
}

// formatAggregation formats search aggregations into a readable summary.
func formatAggregation(agg search.Aggregations) string {
	var sb strings.Builder

	// Sentiment distribution
	if len(agg.BySentiment) > 0 {
		sb.WriteString("**Phân bố cảm xúc:**\n")
		for _, s := range agg.BySentiment {
			sb.WriteString(fmt.Sprintf("- %s: %d (%.1f%%)\n", s.Sentiment, s.Count, s.Percentage))
		}
		sb.WriteString("\n")
	}

	// Aspect breakdown
	if len(agg.ByAspect) > 0 {
		sb.WriteString("**Phân bố theo khía cạnh:**\n")
		for _, a := range agg.ByAspect {
			sb.WriteString(fmt.Sprintf("- %s: %d mentions, avg sentiment %.2f\n",
				a.Aspect, a.Count, a.AvgSentimentScore))
		}
		sb.WriteString("\n")
	}

	// Platform breakdown
	if len(agg.ByPlatform) > 0 {
		sb.WriteString("**Phân bố theo nền tảng:**\n")
		for _, p := range agg.ByPlatform {
			sb.WriteString(fmt.Sprintf("- %s: %d (%.1f%%)\n", p.Platform, p.Count, p.Percentage))
		}
	}

	if sb.Len() == 0 {
		return "(Không có dữ liệu tổng hợp)"
	}

	return sb.String()
}
