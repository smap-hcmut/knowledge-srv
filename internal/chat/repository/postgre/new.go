package postgre

import (
	"database/sql"

	"knowledge-srv/internal/chat/repository"
	"knowledge-srv/pkg/log"
)

type implRepository struct {
	db *sql.DB
	l  log.Logger
}

// New - Factory function
func New(db *sql.DB, l log.Logger) repository.PostgresRepository {
	return &implRepository{
		db: db,
		l:  l,
	}
}
