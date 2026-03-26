package usecase

// Config holds the configuration for the chat usecase.
type Config struct {
	NotebookEnabled          bool
	NotebookFallbackEnabled  bool
	ChatTimeoutSec           int
}
