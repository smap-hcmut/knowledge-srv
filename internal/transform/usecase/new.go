package usecase

import (
	"knowledge-srv/internal/transform"

	"github.com/smap-hcmut/shared-libs/go/log"
)

type implUseCase struct {
	maxPostsPerPart int
	l               log.Logger
}

// New creates a new transform use case.
func New(maxPostsPerPart int, l log.Logger) transform.UseCase {
	if maxPostsPerPart <= 0 {
		maxPostsPerPart = 50
	}
	return &implUseCase{
		maxPostsPerPart: maxPostsPerPart,
		l:               l,
	}
}
