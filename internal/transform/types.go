package transform

// TransformInput selects one analytics run and project scope for Qdrant-backed markdown export.
type TransformInput struct {
	ProjectID   string
	CampaignID  string
	RunID       string
	WindowStart string // RFC3339 — used for ISO week label
}
