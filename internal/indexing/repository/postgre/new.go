package postgre

import (
	"database/sql"
	"knowledge-srv/internal/indexing"
)

// implRepository implements indexing.Repository interface
type implRepository struct {
	db *sql.DB
}

// New creates a new PostgreSQL repository for indexing domain
func New(db *sql.DB) indexing.Repository {
	return &implRepository{
		db: db,
	}
}
