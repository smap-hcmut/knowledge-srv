package usecase

import (
	"strings"
	"testing"

	analyticspkg "knowledge-srv/pkg/analytics"
)

func TestBuildAnalyticsAnswerContrastQuestion(t *testing.T) {
	answer, citations, suggestions, docsUsed := buildAnalyticsAnswer(
		"Tại sao có sự trái chiều trong đánh giá về chiến dịch này?",
		testAnalyticsSnapshot(),
	)

	if !strings.Contains(answer, "Có sự trái chiều") {
		t.Fatalf("answer missing contrast explanation: %s", answer)
	}
	if !strings.Contains(answer, "Mẫu phản hồi tiêu cực") {
		t.Fatalf("answer missing negative evidence: %s", answer)
	}
	if !strings.Contains(answer, "Mẫu phản hồi tích cực") {
		t.Fatalf("answer missing positive evidence: %s", answer)
	}
	if docsUsed == 0 || len(citations) == 0 {
		t.Fatalf("expected citations, docsUsed=%d citations=%d", docsUsed, len(citations))
	}
	if len(suggestions) == 0 {
		t.Fatal("expected follow-up suggestions")
	}
}

func TestBuildAnalyticsAnswerSentimentDriverQuestion(t *testing.T) {
	answer, citations, _, docsUsed := buildAnalyticsAnswer(
		"Đào sâu lý do sentiment thấp nhất theo nền tảng",
		testAnalyticsSnapshot(),
	)

	if !strings.Contains(answer, "Nền tảng kéo sentiment thấp nhất hiện là Facebook") {
		t.Fatalf("answer missing weakest platform: %s", answer)
	}
	if !strings.Contains(answer, "Các cụm nguyên nhân trong mẫu negative") {
		t.Fatalf("answer missing negative drivers: %s", answer)
	}
	if strings.Contains(answer, "Mình chưa tìm thấy đủ dữ liệu") {
		t.Fatalf("answer should not use generic fallback: %s", answer)
	}
	if docsUsed == 0 || len(citations) == 0 {
		t.Fatalf("expected citations, docsUsed=%d citations=%d", docsUsed, len(citations))
	}
}

func testAnalyticsSnapshot() analyticspkg.Snapshot {
	return analyticspkg.Snapshot{
		KPIs: analyticspkg.KPIsResponse{
			Metrics: []analyticspkg.KPIMetric{
				{Label: "Total Mentions", Value: 100, Formatted: "100"},
				{Label: "Engagement", Value: 9500, Formatted: "9.5K"},
				{Label: "Sentiment Score", Value: -0.7, Formatted: "-0.7%"},
			},
		},
		Platforms: analyticspkg.PlatformsResponse{Stats: []analyticspkg.PlatformStat{
			{Platform: "facebook", Name: "Facebook", Mentions: 40, EngagementRaw: 576, Sentiment: -6},
			{Platform: "youtube", Name: "YouTube", Mentions: 35, EngagementRaw: 5900, Sentiment: 1},
			{Platform: "tiktok", Name: "TikTok", Mentions: 25, EngagementRaw: 3100, Sentiment: 3},
		}},
		Sentiment: analyticspkg.SentimentResponse{
			Donut: []analyticspkg.SentimentItem{
				{Label: "negative", Value: 33},
				{Label: "neutral", Value: 35},
				{Label: "positive", Value: 32},
			},
			Total: 100,
		},
		Keywords: analyticspkg.KeywordsResponse{Keywords: []analyticspkg.KeywordItem{
			{Text: "app", Volume: 20},
			{Text: "phí", Volume: 18},
			{Text: "shipper", Volume: 15},
		}},
		Posts: analyticspkg.PostsResponse{
			Total: 5,
			Posts: []analyticspkg.PostItem{
				{
					ID:             "neg-facebook",
					Platform:       "facebook",
					AuthorUsername: "user1",
					Content:        "App lỗi, phí giao hàng cao và tài xế hủy đơn làm khách rất bực.",
					Sentiment:      "negative",
					Engagement:     120,
					Keywords:       []string{"app", "phí", "tài xế"},
				},
				{
					ID:             "neg-youtube",
					Platform:       "youtube",
					AuthorUsername: "user2",
					Content:        "Shipper bị khách sai vặt và không được hỗ trợ khi đơn hàng nặng.",
					Sentiment:      "negative",
					Engagement:     90,
					Keywords:       []string{"shipper", "hỗ trợ", "đơn hàng"},
				},
				{
					ID:             "pos-tiktok",
					Platform:       "tiktok",
					AuthorUsername: "fan1",
					Content:        "Giao nhanh, tài xế vui vẻ, trải nghiệm Ahamove lần này rất ổn.",
					Sentiment:      "positive",
					Engagement:     210,
					Keywords:       []string{"giao nhanh", "tài xế"},
				},
				{
					ID:             "pos-facebook",
					Platform:       "facebook",
					AuthorUsername: "fan2",
					Content:        "Mình dùng Ahamove giao nội thành thấy tiện và hỗ trợ tốt.",
					Sentiment:      "positive",
					Engagement:     80,
					Keywords:       []string{"giao nội thành", "hỗ trợ"},
				},
			},
		},
	}
}
