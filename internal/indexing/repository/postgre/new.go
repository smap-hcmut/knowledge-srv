package postgre

import (
	"database/sql"
	repo "knowledge-srv/internal/indexing/repository"
)

// implRepository implements repository.Repository interface
type implRepository struct {
	db *sql.DB
}

// New creates a new PostgreSQL repository for indexing domain
func New(db *sql.DB) repo.Repository {
	return &implRepository{
		db: db,
	}
}
