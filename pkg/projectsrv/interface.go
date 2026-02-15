package projectsrv

import "context"

// IProject defines the interface for Project Service API client.
// Implementations are safe for concurrent use.
type IProject interface {
	GetCampaign(ctx context.Context, campaignID string) (*Campaign, error)
	ValidateProjectAccess(ctx context.Context, userID, projectID string) (bool, error)
}

// New creates a new Project Service client. Returns the interface.
func New(cfg ProjectConfig) IProject {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = defaultHTTPClient()
	}
	return &projectImpl{
		baseURL:    cfg.BaseURL,
		httpClient: cfg.HTTPClient,
	}
}
