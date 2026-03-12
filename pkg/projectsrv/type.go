package projectsrv

import pkghttp "github.com/smap-hcmut/shared-libs/go/http"

// ProjectConfig holds configuration for the Project Service client.
type ProjectConfig struct {
	BaseURL    string
	HTTPClient pkghttp.Client
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
	httpClient pkghttp.Client
}
