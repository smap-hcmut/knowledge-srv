package usecase

import "testing"

func TestShouldUseAnalyticsFallback(t *testing.T) {
	tests := []struct {
		name   string
		intent QueryIntent
		query  string
		want   bool
	}{
		{
			name:   "allows aggregate comparison",
			intent: IntentStructured,
			query:  "So sánh tỷ lệ sentiment giữa các nền tảng",
			want:   true,
		},
		{
			name:   "rejects evidence query even when structured",
			intent: IntentStructured,
			query:  "Top bài viết tích cực nổi bật là gì?",
			want:   false,
		},
		{
			name:   "allows campaign summary query",
			intent: IntentNarrative,
			query:  "Tóm tắt insight chính của campaign",
			want:   true,
		},
		{
			name:   "allows sentiment driver query",
			intent: IntentNarrative,
			query:  "Đào sâu lý do sentiment thấp nhất theo nền tảng",
			want:   true,
		},
		{
			name:   "allows campaign contrast query",
			intent: IntentNarrative,
			query:  "Tại sao có sự trái chiều trong đánh giá về chiến dịch này?",
			want:   true,
		},
		{
			name:   "rejects qualitative evidence query",
			intent: IntentNarrative,
			query:  "Phân tích insight khách hàng nói gì",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldUseAnalyticsFallback(tt.intent, tt.query)
			if got != tt.want {
				t.Fatalf("ShouldUseAnalyticsFallback() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldUseAnalyticsFirst(t *testing.T) {
	tests := []struct {
		name   string
		intent QueryIntent
		query  string
		want   bool
	}{
		{
			name:   "routes platform comparison to analytics first",
			intent: IntentStructured,
			query:  "So sánh giữa các nền tảng?",
			want:   true,
		},
		{
			name:   "routes sentiment driver to analytics first",
			intent: IntentNarrative,
			query:  "Đào sâu lý do sentiment thấp nhất theo nền tảng",
			want:   true,
		},
		{
			name:   "routes contrast question to analytics first",
			intent: IntentNarrative,
			query:  "Tại sao có sự trái chiều trong đánh giá về chiến dịch này?",
			want:   true,
		},
		{
			name:   "keeps evidence post query on search path",
			intent: IntentStructured,
			query:  "Top bài viết tích cực nổi bật là gì?",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldUseAnalyticsFirst(tt.intent, tt.query)
			if got != tt.want {
				t.Fatalf("ShouldUseAnalyticsFirst() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsSmallTalkMessage(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{name: "simple alo", query: "Alo ALo", want: true},
		{name: "hello", query: "hello!", want: true},
		{name: "connectivity ping", query: "ping test", want: true},
		{name: "analysis request with greeting", query: "Alo so sánh nền tảng giúp tôi", want: false},
		{name: "real campaign question", query: "Tại sao sentiment Facebook thấp?", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSmallTalkMessage(tt.query)
			if got != tt.want {
				t.Fatalf("IsSmallTalkMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}
