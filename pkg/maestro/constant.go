package maestro

import "time"

const (
	DefaultTimeout   = 30 * time.Second
	DefaultRetries   = 2
	DefaultRetryWait = 1 * time.Second
)

// API path segments (relative to BaseURL).
const (
	PathSessions  = "/notebooklm/sessions"
	PathNotebooks = "/notebooklm/notebooks"
	PathJobs      = "/notebooklm/jobs"
	PathPipelines = "/notebooklm/pipelines"
)

// Session statuses returned by Maestro API.
const (
	SessionReady       = "ready"
	SessionBusy        = "busy"
	SessionTearingDown = "tearingDown"
)

// Job statuses returned by Maestro API.
const (
	JobQueued     = "queued"
	JobProcessing = "processing"
	JobCompleted  = "completed"
	JobFailed     = "failed"
)

// Headers used by Maestro API.
const (
	HeaderAPIKey    = "X-Api-Key"
	HeaderSessionID = "X-Session-Id"
	HeaderSignature = "X-Maestro-Signature"
)
