package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestBuildLLMProvidersUsesConfiguredOrderAsWhitelist(t *testing.T) {
	v := viper.New()
	v.Set("llm.provider_order", "deepseek")
	v.Set("llm.disabled_providers", "gemini,openai,qwen")
	v.Set("llm.deepseek_api_key", "deepseek-key")
	v.Set("llm.deepseek_model", "deepseek-v4-pro")
	v.Set("llm.openai_api_key", "openai-key")
	v.Set("llm.qwen_api_key", "qwen-key")

	providers := buildLLMProviders(GeminiConfig{APIKey: "gemini-key", Model: "gemini-2.0-flash"}, v)
	if len(providers) != 1 {
		t.Fatalf("provider count = %d, want 1", len(providers))
	}
	if providers[0].Name != "deepseek" {
		t.Fatalf("provider name = %q, want deepseek", providers[0].Name)
	}
	if providers[0].Model != "deepseek-v4-pro" {
		t.Fatalf("provider model = %q, want deepseek-v4-pro", providers[0].Model)
	}
}

func TestBuildLLMProvidersDefaultsToDeepSeekFirst(t *testing.T) {
	v := viper.New()
	v.Set("llm.deepseek_api_key", "deepseek-key")
	v.Set("llm.openai_api_key", "openai-key")

	providers := buildLLMProviders(GeminiConfig{APIKey: "gemini-key", Model: "gemini-2.0-flash"}, v)
	if len(providers) < 2 {
		t.Fatalf("provider count = %d, want at least 2", len(providers))
	}
	if providers[0].Name != "deepseek" {
		t.Fatalf("first provider = %q, want deepseek", providers[0].Name)
	}
}
