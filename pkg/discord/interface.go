package discord

import (
	"context"

	"knowledge-srv/pkg/log"
)

// IDiscord defines the interface for Discord webhook service.
// Implementations are safe for concurrent use.
type IDiscord interface {
	SendMessage(ctx context.Context, content string) error
	SendEmbed(ctx context.Context, options MessageOptions) error
	SendError(ctx context.Context, title, description string, err error) error
	SendSuccess(ctx context.Context, title, description string) error
	SendWarning(ctx context.Context, title, description string) error
	SendInfo(ctx context.Context, title, description string) error
	ReportBug(ctx context.Context, message string) error
	SendNotification(ctx context.Context, title, description string, fields map[string]string) error
	SendActivityLog(ctx context.Context, action, user, details string) error
	GetWebhookURL() string
	Close() error
}

// DiscordWebhook contains webhook information for Discord API.
type DiscordWebhook struct {
	ID    string
	Token string
}

// NewDiscordWebhook creates a new Discord webhook instance.
func NewDiscordWebhook(id, token string) (*DiscordWebhook, error) {
	if id == "" || token == "" {
		return nil, errWebhookRequired
	}
	return &DiscordWebhook{ID: id, Token: token}, nil
}

// New creates a new Discord service. Returns the interface.
func New(l log.Logger, webhook *DiscordWebhook) (IDiscord, error) {
	if webhook == nil || webhook.ID == "" || webhook.Token == "" {
		return nil, errWebhookRequired
	}
	cfg := DefaultConfig()
	client := newHTTPClient(cfg.Timeout)
	return &discordImpl{
		l:       l,
		webhook: webhook,
		config:  cfg,
		client:  client,
	}, nil
}
