package usecase

import "knowledge-srv/internal/report"

// getTemplates returns the section templates for a given report type.
func getTemplates(reportType string) []report.SectionTemplate {
	switch reportType {
	case report.ReportTypeSummary:
		return summaryTemplates
	case report.ReportTypeComparison:
		return comparisonTemplates
	case report.ReportTypeTrend:
		return trendTemplates
	case report.ReportTypeAspectDeep:
		return aspectDeepDiveTemplates
	default:
		return summaryTemplates
	}
}

// ----------- SUMMARY templates -----------
var summaryTemplates = []report.SectionTemplate{
	{
		Title: "Tổng Quan Chung",
		Prompt: `Dựa trên dữ liệu phân tích sau đây, hãy viết một phần TỔng QUAN CHUNG cho báo cáo.
Bao gồm:
- Số lượng phản hồi đã phân tích
- Phân bố cảm xúc tổng quát (tích cực, tiêu cực, trung tính)
- Các điểm nổi bật chính
- Đánh giá sức khỏe thương hiệu tổng thể

%s`,
	},
	{
		Title: "Phân Tích Cảm Xúc",
		Prompt: `Dựa trên dữ liệu sau, hãy viết phần PHÂN TÍCH CẢM XÚC chi tiết.
Bao gồm:
- Tỷ lệ cảm xúc tích cực/tiêu cực/trung tính
- Các chủ đề chính gây ra cảm xúc tiêu cực
- Các chủ đề nhận được đánh giá tích cực
- Xu hướng cảm xúc đáng chú ý

%s`,
	},
	{
		Title: "Các Khía Cạnh Nổi Bật",
		Prompt: `Dựa trên dữ liệu sau, hãy viết phần phân tích CÁC KHÍA CẠNH NỔI BẬT.
Bao gồm:
- Top khía cạnh được đề cập nhiều nhất
- Khía cạnh có điểm cảm xúc cao nhất và thấp nhất
- Mối tương quan giữa các khía cạnh
- Khuyến nghị cải thiện cho từng khía cạnh

%s`,
	},
	{
		Title: "Kết Luận và Khuyến Nghị",
		Prompt: `Dựa trên toàn bộ phân tích ở trên, hãy viết phần KẾT LUẬN VÀ KHUYẾN NGHỊ.
Bao gồm:
- Tóm tắt các phát hiện chính
- Ưu tiên hành động (cao/trung bình/thấp)
- Khuyến nghị cụ thể cho từng lĩnh vực cải thiện
- Các bước tiếp theo được đề xuất

%s`,
	},
}

// ----------- COMPARISON templates -----------
var comparisonTemplates = []report.SectionTemplate{
	{
		Title: "So Sánh Theo Nền Tảng",
		Prompt: `Dựa trên dữ liệu sau, hãy viết phần SO SÁNH THEO NỀN TẢNG.
So sánh phản hồi giữa các nền tảng (Facebook, Shopee, Tiki, ...).
Bao gồm:
- Phân bố phản hồi theo nền tảng
- Điểm cảm xúc trung bình của mỗi nền tảng
- Sự khác biệt chính giữa các nền tảng
- Nền tảng nào cần chú ý nhất

%s`,
	},
	{
		Title: "So Sánh Theo Khía Cạnh",
		Prompt: `Dựa trên dữ liệu sau, hãy viết phần SO SÁNH THEO KHÍA CẠNH.
So sánh hiệu suất giữa các khía cạnh khác nhau.
Bao gồm:
- Bảng so sánh điểm cảm xúc theo khía cạnh
- Khía cạnh mạnh nhất và yếu nhất
- Gap analysis giữa kỳ vọng và thực tế
- Khuyến nghị theo từng khía cạnh

%s`,
	},
	{
		Title: "Phân Tích Cross-Platform",
		Prompt: `Dựa trên dữ liệu sau, hãy viết phần PHÂN TÍCH CROSS-PLATFORM.
Tìm các pattern chung và khác biệt giữa nền tảng × khía cạnh.
Bao gồm:
- Ma trận nền tảng × khía cạnh
- Các insight cross-platform nổi bật
- Cơ hội cải thiện đa kênh

%s`,
	},
}

// ----------- TREND templates -----------
var trendTemplates = []report.SectionTemplate{
	{
		Title: "Xu Hướng Cảm Xúc Theo Thời Gian",
		Prompt: `Dựa trên dữ liệu sau, hãy viết phần XU HƯỚNG CẢM XÚC THEO THỜI GIAN.
Phân tích sự thay đổi cảm xúc qua các giai đoạn.
Bao gồm:
- Trend cảm xúc chung (tăng/giảm/ổn định)
- Các điểm đột biến (spike) và nguyên nhân
- Dự báo xu hướng ngắn hạn

%s`,
	},
	{
		Title: "Xu Hướng Theo Khía Cạnh",
		Prompt: `Dựa trên dữ liệu sau, hãy viết phần XU HƯỚNG THEO KHÍA CẠNH.
Phân tích sự thay đổi của từng khía cạnh theo thời gian.
Bao gồm:
- Khía cạnh đang cải thiện vs. xấu đi
- Tốc độ thay đổi của mỗi khía cạnh
- Dự đoán khía cạnh cần ưu tiên

%s`,
	},
	{
		Title: "Phân Tích Sự Kiện và Tác Động",
		Prompt: `Dựa trên dữ liệu sau, hãy viết phần PHÂN TÍCH SỰ KIỆN VÀ TÁC ĐỘNG.
Xác định các sự kiện quan trọng ảnh hưởng đến phản hồi.
Bao gồm:
- Các sự kiện gây ra thay đổi đáng kể
- Mức độ tác động của mỗi sự kiện
- Thời gian phục hồi sau sự kiện tiêu cực

%s`,
	},
}

// ----------- ASPECT_DEEP_DIVE templates -----------
var aspectDeepDiveTemplates = []report.SectionTemplate{
	{
		Title: "Phân Tích Chi Tiết Từng Khía Cạnh",
		Prompt: `Dựa trên dữ liệu sau, hãy viết phần PHÂN TÍCH CHI TIẾT TỪNG KHÍA CẠNH.
Cho mỗi khía cạnh, phân tích sâu:
- Số lượng và tỷ lệ phản hồi
- Phân bố cảm xúc chi tiết
- Từ khóa đặc trưng
- Ví dụ phản hồi tiêu biểu (trích dẫn)

%s`,
	},
	{
		Title: "Mối Quan Hệ Giữa Các Khía Cạnh",
		Prompt: `Dựa trên dữ liệu sau, hãy viết phần MỐI QUAN HỆ GIỮA CÁC KHÍA CẠNH.
Phân tích correlation giữa các khía cạnh.
Bao gồm:
- Khía cạnh nào thường xuất hiện cùng nhau
- Ảnh hưởng của khía cạnh này lên khía cạnh khác
- Cluster analysis nếu có thể

%s`,
	},
	{
		Title: "Gap Analysis và Khuyến Nghị Chi Tiết",
		Prompt: `Dựa trên phân tích chi tiết ở trên, hãy viết phần GAP ANALYSIS VÀ KHUYẾN NGHỊ CHI TIẾT.
Bao gồm:
- Gap giữa kỳ vọng khách hàng và thực tế cho mỗi khía cạnh
- Benchmark so với tiêu chuẩn ngành (nếu có thể suy ra)
- Roadmap cải thiện theo thứ tự ưu tiên
- KPIs đề xuất cho mỗi khía cạnh

%s`,
	},
}
