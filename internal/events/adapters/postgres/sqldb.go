package postgres

import (
	"context"
	"database/sql"
)

type sqlDB struct {
	db *sql.DB
}

func NewSQLDB(db *sql.DB) DB {
	return &sqlDB{db: db}
}

func (s *sqlDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}
