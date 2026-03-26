package usecase

import (
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/transform"

	"github.com/smap-hcmut/shared-libs/go/log"
)

type implUseCase struct {
	pointUC         point.UseCase
	maxPostsPerPart int
	l               log.Logger
}

// New creates a transform use case that scrolls Qdrant via pointUC.
func New(pointUC point.UseCase, maxPostsPerPart int, l log.Logger) transform.UseCase {
	if maxPostsPerPart <= 0 {
		maxPostsPerPart = 50
	}
	return &implUseCase{
		pointUC:         pointUC,
		maxPostsPerPart: maxPostsPerPart,
		l:               l,
	}
}
