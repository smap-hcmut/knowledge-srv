package usecase

import (
	"context"

	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/model"
)

// GetSuggestions - Generate smart suggestions for a campaign
func (uc *implUseCase) GetSuggestions(ctx context.Context, sc model.Scope, input chat.GetSuggestionsInput) (chat.SuggestionOutput, error) {
	suggestions := []chat.SmartSuggestion{
		{
			Query:       "Tổng quan sentiment về sản phẩm/dịch vụ?",
			Category:    "insight",
			Description: "Xem tổng quan cảm xúc của khách hàng",
		},
		{
			Query:       "Vấn đề nào được nhắc đến nhiều nhất gần đây?",
			Category:    "trending_negative",
			Description: "Phát hiện các vấn đề phổ biến từ phản hồi khách hàng",
		},
		{
			Query:       "So sánh phản hồi giữa các nền tảng?",
			Category:    "comparison",
			Description: "So sánh sentiment giữa Facebook, TikTok, Shopee...",
		},
		{
			Query:       "Xu hướng sentiment thay đổi thế nào trong tuần qua?",
			Category:    "sentiment_shift",
			Description: "Theo dõi biến động sentiment theo thời gian",
		},
	}

	return chat.SuggestionOutput{
		Suggestions: suggestions,
	}, nil
}
