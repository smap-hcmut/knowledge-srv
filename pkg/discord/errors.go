package discord

import "errors"

var (
	errWebhookRequired = errors.New("webhook ID and token are required")
)
