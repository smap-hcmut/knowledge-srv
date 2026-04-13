package point

import "fmt"

const (
	CollectionAnalyticsLegacy = "smap_analytics"
	CollectionMacroInsights   = "macro_insights"
)

// CollectionForProject returns the Qdrant collection name for a given project.
// This matches the naming used by the indexing pipeline (index_batch.go).
func CollectionForProject(projectID string) string {
	return fmt.Sprintf("proj_%s", projectID)
}
