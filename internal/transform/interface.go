package transform

//go:generate mockery --name UseCase
type UseCase interface {
	// BuildParts transforms a batch of analytics posts into markdown parts
	// grouped by ISO week and split by MaxPostsPerPart.
	BuildParts(input TransformInput) ([]MarkdownPart, error)
}
