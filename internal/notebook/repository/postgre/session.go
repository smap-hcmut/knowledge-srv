package postgre

import (
	"knowledge-srv/internal/notebook/repository"
)

// noopSessionRepo satisfies repository.SessionRepo (no persistence yet).
type noopSessionRepo struct{}

// NewSessionRepo returns a no-op session repository.
func NewSessionRepo() repository.SessionRepo {
	return &noopSessionRepo{}
}
