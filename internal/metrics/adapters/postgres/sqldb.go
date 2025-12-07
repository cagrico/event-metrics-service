package postgres

import (
	"context"
	"database/sql"
)

type sqlRows struct {
	rows *sql.Rows
}

func (r *sqlRows) Next() bool {
	return r.rows.Next()
}

func (r *sqlRows) Scan(dest ...any) error {
	return r.rows.Scan(dest...)
}

func (r *sqlRows) Err() error {
	return r.rows.Err()
}

func (r *sqlRows) Close() error {
	return r.rows.Close()
}

type sqlDB struct {
	db *sql.DB
}

func NewSQLDB(db *sql.DB) DB {
	return &sqlDB{db: db}
}

func (s *sqlDB) QueryContext(ctx context.Context, query string, args ...any) (RowScanner, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRows{rows: rows}, nil
}
