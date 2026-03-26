package usecase

// Config holds the configuration for the notebook usecase.
type Config struct {
	NotebookEnabled    bool
	JobPollIntervalMs  int
	JobPollMaxAttempts int
	WebhookCallbackURL string
	WebhookSecret      string
}
