package project

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	pkghttp "knowledge-srv/pkg/http"
)

// Campaign represents a campaign in the Project Service
type Campaign struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	ProjectIDs []string `json:"project_ids"`
}

// Client interacts with the Project Service
type Client struct {
	baseURL    string
	httpClient *pkghttp.Client
}

// NewClient creates a new Project Service client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: pkghttp.NewClient(pkghttp.ClientConfig{
			Timeout:   10 * time.Second,
			Retries:   3,
			RetryWait: 1 * time.Second,
		}),
	}
}

// GetCampaign retrieves campaign details by ID
func (c *Client) GetCampaign(ctx context.Context, campaignID string) (*Campaign, error) {
	url := fmt.Sprintf("%s/api/v1/campaigns/%s", c.baseURL, campaignID)

	body, statusCode, err := c.httpClient.Get(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get campaign: %w", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", statusCode)
	}

	var campaign Campaign
	if err := json.Unmarshal(body, &campaign); err != nil {
		return nil, fmt.Errorf("failed to unmarshal campaign: %w", err)
	}

	return &campaign, nil
}

// ValidateProjectAccess checks if a user has access to a project
func (c *Client) ValidateProjectAccess(ctx context.Context, userID, projectID string) (bool, error) {
	// Implementation depends on the actual API of Project Service
	// For now, assuming an endpoint exists or we might just use GetCampaign and check locally?
	// The spec says: "Call Project Service API"
	// Let's assume an endpoint like /api/v1/projects/{project_id}/access?user_id={user_id}

	url := fmt.Sprintf("%s/api/v1/projects/%s/access?user_id=%s", c.baseURL, projectID, userID)

	_, statusCode, err := c.httpClient.Get(ctx, url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to validate project access: %w", err)
	}

	if statusCode == http.StatusOK {
		return true, nil
	} else if statusCode == http.StatusForbidden || statusCode == http.StatusUnauthorized {
		return false, nil
	}

	return false, fmt.Errorf("unexpected status code: %d", statusCode)
}
