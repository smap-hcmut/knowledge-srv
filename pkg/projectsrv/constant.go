package projectsrv

import "time"

const (
	// DefaultTimeout is the default HTTP client timeout for Project Service.
	DefaultTimeout = 10 * time.Second
	// DefaultRetries is the default number of retries.
	DefaultRetries = 3
	// DefaultRetryWait is the default wait between retries.
	DefaultRetryWait = 1 * time.Second
)

// API path segments (for reference; full URLs built in projectsrv.go).
const (
	PathCampaigns         = "/api/v1/internal/campaigns"
	PathProjects          = "/api/v1/projects"
	InternalKeyHeaderName = "X-Internal-Key"
)
