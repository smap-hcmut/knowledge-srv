package postgre

import (
	"database/sql"
	"knowledge-srv/internal/chat/repository"

	"github.com/smap-hcmut/shared-libs/go/log"
)

type implRepository struct {
	db *sql.DB
	l  log.Logger
}

func New(db *sql.DB, l log.Logger) repository.PostgresRepository {
	return &implRepository{
		db: db,
		l:  l,
	}
}
