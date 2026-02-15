package projectsrv

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	pkghttp "knowledge-srv/pkg/http"
)

func defaultHTTPClient() pkghttp.IClient {
	return pkghttp.NewClient(pkghttp.ClientConfig{
		Timeout:   DefaultTimeout,
		Retries:   DefaultRetries,
		RetryWait: DefaultRetryWait,
	})
}

// GetCampaign retrieves campaign details by ID.
func (c *projectImpl) GetCampaign(ctx context.Context, campaignID string) (*Campaign, error) {
	url := fmt.Sprintf("%s%s/%s", c.baseURL, PathCampaigns, campaignID)

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

// ValidateProjectAccess checks if a user has access to a project.
func (c *projectImpl) ValidateProjectAccess(ctx context.Context, userID, projectID string) (bool, error) {
	url := fmt.Sprintf("%s%s/%s/access?user_id=%s", c.baseURL, PathProjects, projectID, userID)

	_, statusCode, err := c.httpClient.Get(ctx, url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to validate project access: %w", err)
	}

	if statusCode == http.StatusOK {
		return true, nil
	}
	if statusCode == http.StatusForbidden || statusCode == http.StatusUnauthorized {
		return false, nil
	}

	return false, fmt.Errorf("unexpected status code: %d", statusCode)
}
