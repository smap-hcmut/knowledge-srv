package contentquality

import (
	"regexp"
	"strings"
	"unicode"
)

var phoneNumberPattern = regexp.MustCompile(`(^|[^0-9])0[0-9]{8,10}([^0-9]|$)`)

// IsLowValueMarketingContent filters evidence that is technically present in the
// index but poor for marketer-facing reports: corporate/internal posts,
// hashtag-only scraps, and obvious off-topic content.
func IsLowValueMarketingContent(content string) bool {
	normalized := strings.ToLower(strings.Join(strings.Fields(content), " "))
	if normalized == "" {
		return true
	}
	if looksHashtagOnly(normalized) {
		return true
	}

	lowValuePatterns := []string{
		"ahamovecareers",
		"aha connect",
		"ahaconnect",
		"yes a.i do",
		"yes ai do",
		"a.i driven",
		"ai driven",
		"powered logistics",
		"workshop nội bộ",
		"workshop noi bo",
		"tuyển dụng",
		"tuyen dung",
		"careers",
		"văn phòng ahamove",
		"van phong ahamove",
		"minigame",
		"rinh quà",
		"rinh qua",
		"chỉ vàng",
		"chi vang",
		"e-voucher",
		"got it trị giá",
		"got it tri gia",
		"giao hàng đồng giá",
		"giao hang dong gia",
		"10namdongdieu",
		"tạo dáng cực ngầu",
		"tao dang cuc ngau",
		"mạng lưới tài xế hùng hậu",
		"mang luoi tai xe hung hau",
		"dịch vụ chính của ahamove",
		"dich vu chinh cua ahamove",
		"collshp.com",
		"share_channel_code",
		"ủng hộ mua hàng",
		"ung ho mua hang",
		"shopee qua kênh",
		"shoppe qua kênh",
		"shopee qua kenh",
		"shoppe qua kenh",
		"cod mobile",
		"codm",
		"cod has",
		"cod is",
		"cod are",
		"cod officially",
		"cod officaly",
		"offically lost it",
		"officially lost it",
		"officaly lost it",
		"call of duty",
		"cod fandom",
		"apex movement",
		"youtube.com/shopcollection",
		"bộ đồ nghề",
		"bo do nghe",
		"menu trái cây",
		"menu trai cay",
		"bánh cuốn",
		"banh cuon",
		"serum",
		"cọ gấu",
		"co gau",
		"hàng có sẵn",
		"hang co san",
		"hàng_có_sẵn",
		"định danh chủ",
		"định danh chu",
		"cà mau aa",
		"ca mau aa",
		"ship toàn quốc",
		"ship toan quoc",
		"ship từ",
		"ship tu",
		"phí ship",
		"phi ship",
	}
	for _, pattern := range lowValuePatterns {
		if strings.Contains(normalized, pattern) {
			return true
		}
	}
	if phoneNumberPattern.MatchString(normalized) && containsAny(normalized, []string{
		"zalo",
		"alo",
		"liên hệ",
		"lien he",
		"ib",
		"inbox",
		"tư vấn",
		"tu van",
		"chuyên bán",
		"chuyen ban",
	}) {
		return true
	}

	return false
}

func containsAny(content string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}
	return false
}

func looksHashtagOnly(content string) bool {
	words := strings.Fields(content)
	if len(words) == 0 {
		return true
	}

	hashtags := 0
	nonHashtagLetters := 0
	for _, word := range words {
		if strings.HasPrefix(word, "#") {
			hashtags++
			continue
		}
		for _, r := range word {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				nonHashtagLetters++
			}
		}
	}

	return hashtags >= 3 && float64(hashtags)/float64(len(words)) >= 0.7 && nonHashtagLetters < 24
}
