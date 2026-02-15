package locale

import (
	"context"
	"errors"
	"strings"
)

// ErrLocaleNotFound is returned when a requested locale is not supported.
var ErrLocaleNotFound = errors.New("locale not found")

// Supported languages
const (
	EN = "en" // English
	VI = "vi" // Vietnamese
	JA = "ja" // Japanese
)

// LangList contains all supported language codes.
var LangList = []string{EN, VI, JA}

// DefaultLang is the default language used when no valid locale is provided.
var DefaultLang = EN

// Locale is a context key type for storing locale information.
type Locale struct{}

// ParseLang parses and validates a language code.
// It returns the default language if the provided code is not supported.
// The ip is case-insensitive and trimmed of whitespace.
func ParseLang(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))

	switch lang {
	case EN, "english":
		return EN
	case VI, "vietnamese", "việt nam":
		return VI
	case JA, "japanese":
		return JA
	default:
		return DefaultLang
	}
}

// IsValidLang checks if a language code is supported.
func IsValidLang(lang string) bool {
	lang = strings.TrimSpace(strings.ToLower(lang))
	for _, supported := range LangList {
		if lang == supported {
			return true
		}
	}
	return false
}

// GetLang retrieves the locale from context, returning the default if not found.
func GetLang(ctx context.Context) string {
	lang, ok := GetLocaleFromContext(ctx)
	if !ok {
		return DefaultLang
	}
	return lang
}

// SetLocaleToContext sets the locale in the context for use in handlers.
func SetLocaleToContext(ctx context.Context, lang string) context.Context {
	// Validate before setting
	if !IsValidLang(lang) {
		lang = DefaultLang
	}
	return context.WithValue(ctx, Locale{}, lang)
}

// GetLocaleFromContext retrieves the locale from context.
// Returns the locale and true if found, empty string and false otherwise.
func GetLocaleFromContext(ctx context.Context) (string, bool) {
	lang, ok := ctx.Value(Locale{}).(string)
	if !ok || lang == "" {
		return "", false
	}
	return lang, true
}
