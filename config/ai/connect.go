package ai

import (
	"sync"

	"knowledge-srv/config"
	"knowledge-srv/pkg/ai"
)

var (
	voyageInstance *ai.VoyageClient
	geminiInstance *ai.GeminiClient
	voyageOnce     sync.Once
	geminiOnce     sync.Once
	mu             sync.RWMutex
)

// ConnectVoyage initializes the Voyage AI client.
func ConnectVoyage(cfg config.AIConfig) *ai.VoyageClient {
	mu.Lock()
	defer mu.Unlock()

	if voyageInstance != nil {
		return voyageInstance
	}

	voyageOnce.Do(func() {
		voyageInstance = ai.NewVoyageClient(cfg.VoyageAPIKey)
	})

	return voyageInstance
}

// ConnectGemini initializes the Google Gemini client.
func ConnectGemini(cfg config.AIConfig) *ai.GeminiClient {
	mu.Lock()
	defer mu.Unlock()

	if geminiInstance != nil {
		return geminiInstance
	}

	geminiOnce.Do(func() {
		geminiInstance = ai.NewGeminiClient(cfg.GeminiAPIKey, cfg.GeminiModel)
	})

	return geminiInstance
}

// GetVoyageClient returns the singleton Voyage client.
func GetVoyageClient() *ai.VoyageClient {
	mu.RLock()
	defer mu.RUnlock()
	if voyageInstance == nil {
		panic("Voyage client not initialized. Call ConnectVoyage() first")
	}
	return voyageInstance
}

// GetGeminiClient returns the singleton Gemini client.
func GetGeminiClient() *ai.GeminiClient {
	mu.RLock()
	defer mu.RUnlock()
	if geminiInstance == nil {
		panic("Gemini client not initialized. Call ConnectGemini() first")
	}
	return geminiInstance
}
