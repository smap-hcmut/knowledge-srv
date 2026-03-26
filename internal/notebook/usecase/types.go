package usecase

// Config holds the configuration for the notebook usecase.
type Config struct {
	NotebookEnabled    bool
	JobPollIntervalMs  int
	JobPollMaxAttempts int
	SyncMaxRetries     int
	WebhookCallbackURL string
	WebhookSecret      string
}
