package projectsrv

import pkghttp "knowledge-srv/pkg/http"

// ProjectConfig holds configuration for the Project Service client.
type ProjectConfig struct {
	BaseURL    string
	HTTPClient pkghttp.IClient
}

// Campaign represents a campaign in the Project Service.
type Campaign struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	ProjectIDs []string `json:"project_ids"`
}

// projectImpl implements IProject.
type projectImpl struct {
	baseURL    string
	httpClient pkghttp.IClient
}
