package usecase

import (
	"context"
	"fmt"
	"time"

	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/search"
)

// GetSuggestions - Generate smart suggestions for a campaign
func (uc *implUseCase) GetSuggestions(ctx context.Context, sc model.Scope, input chat.GetSuggestionsInput) (chat.SuggestionOutput, error) {
	cacheKey := suggestionCacheKey(sc.UserID, input.CampaignID)
	if cached, ok := uc.lookupSuggestionCache(cacheKey); ok {
		return cached, nil
	}

	// Call Search Domain to get aggregation data
	aggInput := search.AggregateInput{
		CampaignID: input.CampaignID,
	}
	aggOutput, err := uc.searchUC.Aggregate(ctx, sc, aggInput)
	if err != nil {
		uc.l.Warnf(ctx, "chat.usecase.GetSuggestions: Aggregate failed: %v", err)
		// Fallback to generic suggestions
		return chat.SuggestionOutput{
			Suggestions: getFallbackSuggestions(),
		}, nil
	}

	var suggestions []chat.SmartSuggestion

	// Rule 1: Trending Negative Aspects
	for _, aspect := range aggOutput.TopNegativeAspects {
		suggestions = append(suggestions, chat.SmartSuggestion{
			Query:       fmt.Sprintf("Tại sao vấn đề %s lại bị đánh giá tiêu cực?", aspect.Aspect),
			Category:    "trending_negative",
			Description: fmt.Sprintf("Phát hiện %d phản hồi tiêu cực về %s", aspect.Count, aspect.Aspect),
		})
		if len(suggestions) >= 2 { // Limit max 2 aspect suggestions
			break
		}
	}

	// Rule 2: Platform Comparison (if multiple platforms have significant data)
	if len(aggOutput.PlatformBreakdown) >= 2 {
		suggestions = append(suggestions, chat.SmartSuggestion{
			Query:       "So sánh phản hồi giữa các nền tảng?",
			Category:    "comparison",
			Description: "So sánh sentiment giữa các nguồn dữ liệu",
		})
	}

	// Rule 3: Polarized Sentiment (High Positive AND High Negative)
	posCount := aggOutput.SentimentBreakdown["POSITIVE"]
	negCount := aggOutput.SentimentBreakdown["NEGATIVE"]
	total := aggOutput.TotalDocs
	if total > 0 {
		posRatio := float64(posCount) / float64(total)
		negRatio := float64(negCount) / float64(total)
		if posRatio > 0.3 && negRatio > 0.3 {
			suggestions = append(suggestions, chat.SmartSuggestion{
				Query:       "Tại sao có sự trái chiều trong đánh giá về chiến dịch này?",
				Category:    "insight",
				Description: "Phát hiện luồng ý kiến trái chiều (Positive & Negative đều cao)",
			})
		}
	}

	// Rule 4: General Insight (Always add if list is short)
	if len(suggestions) < 4 {
		suggestions = append(suggestions, chat.SmartSuggestion{
			Query:       "Khách hàng đang phản hồi gì nổi bật về chiến dịch này?",
			Category:    "insight",
			Description: "Tóm tắt các phản hồi nổi bật gần đây",
		})
	}

	// Rule 5: Trend (Always add if list is still short)
	if len(suggestions) < 4 {
		suggestions = append(suggestions, chat.SmartSuggestion{
			Query:       "Các phản hồi tích cực nổi bật là gì?",
			Category:    "sentiment_shift",
			Description: "Xem các điểm tích cực nổi bật trong dữ liệu hiện có",
		})
	}

	// Limit total suggestions
	if len(suggestions) > 4 {
		suggestions = suggestions[:4]
	}

	out := chat.SuggestionOutput{
		Suggestions: suggestions,
	}
	uc.storeSuggestionCache(cacheKey, out)
	return out, nil
}

func suggestionCacheKey(userID, campaignID string) string {
	return userID + "|" + campaignID
}

func (uc *implUseCase) lookupSuggestionCache(key string) (chat.SuggestionOutput, bool) {
	uc.suggestionCacheMu.RLock()
	defer uc.suggestionCacheMu.RUnlock()
	entry, ok := uc.suggestionCache[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return chat.SuggestionOutput{}, false
	}
	return entry.response, true
}

func (uc *implUseCase) storeSuggestionCache(key string, out chat.SuggestionOutput) {
	uc.suggestionCacheMu.Lock()
	defer uc.suggestionCacheMu.Unlock()
	// Cheap GC: if the map grows past a few hundred campaigns, drop expired
	// entries inline so it can't grow unbounded with no eviction loop.
	if len(uc.suggestionCache) > 256 {
		now := time.Now()
		for k, v := range uc.suggestionCache {
			if now.After(v.expiresAt) {
				delete(uc.suggestionCache, k)
			}
		}
	}
	uc.suggestionCache[key] = suggestionCacheEntry{
		response:  out,
		expiresAt: time.Now().Add(suggestionCacheTTL),
	}
}

func getFallbackSuggestions() []chat.SmartSuggestion {
	return []chat.SmartSuggestion{
		{
			Query:       "Khách hàng đang phản hồi gì nổi bật về chiến dịch này?",
			Category:    "insight",
			Description: "Tóm tắt các phản hồi nổi bật gần đây",
		},
		{
			Query:       "Vấn đề nào được nhắc đến nhiều nhất gần đây?",
			Category:    "trending_negative",
			Description: "Phát hiện các vấn đề phổ biến từ phản hồi khách hàng",
		},
		{
			Query:       "Các phản hồi tích cực nổi bật là gì?",
			Category:    "insight",
			Description: "Tóm tắt các ý kiến tích cực đáng chú ý",
		},
	}
}
