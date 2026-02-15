package discord

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"identity-srv/pkg/log"
)

// DiscordWebhook contains webhook information for Discord API.
type DiscordWebhook struct {
	ID    string
	Token string
}

// NewDiscordWebhook creates a new Discord webhook instance.
func NewDiscordWebhook(id, token string) (*DiscordWebhook, error) {
	if id == "" || token == "" {
		return nil, errors.New("id and token are required")
	}

	return &DiscordWebhook{
		ID:    id,
		Token: token,
	}, nil
}

// Discord is the Discord service implementation for sending webhook messages.
type Discord struct {
	l       log.Logger
	webhook *DiscordWebhook
	config  Config
	client  *http.Client
}

// New creates a new Discord service instance with the provided logger and webhook.
// Logger can be nil, but logging will be skipped if not provided.
func New(l log.Logger, webhook *DiscordWebhook) (*Discord, error) {
	if webhook == nil {
		return nil, errors.New("webhook is required")
	}

	// Validate webhook ID and token
	if webhook.ID == "" || webhook.Token == "" {
		return nil, errors.New("webhook ID and token are required")
	}

	// Get default config for timeout and retry settings
	config := DefaultConfig()

	// Create HTTP client with timeout from config
	client := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	return &Discord{
		l:       l,
		webhook: webhook,
		config:  config,
		client:  client,
	}, nil
}

// GetWebhookURL returns the Discord webhook URL.
func (d *Discord) GetWebhookURL() string {
	return fmt.Sprintf(webhookURL, d.webhook.ID, d.webhook.Token)
}

// Close closes idle connections in the HTTP client.
func (d *Discord) Close() error {
	d.client.CloseIdleConnections()
	return nil
}
