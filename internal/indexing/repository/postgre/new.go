package postgre

import (
	"database/sql"
	repo "knowledge-srv/internal/indexing/repository"

	"github.com/smap-hcmut/shared-libs/go/log"
)

type implPostgresRepository struct {
	db *sql.DB
	l  log.Logger
}

func New(db *sql.DB, l log.Logger) repo.PostgresRepository {
	return &implPostgresRepository{
		db: db,
		l:  l,
	}
}
