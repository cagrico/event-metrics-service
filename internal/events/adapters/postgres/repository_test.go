package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"event-metrics-service/internal/events/core/domain"
)

// fakeResult implements sql.Result for tests.
type fakeResult struct {
	rowsAffected int64
}

func (f *fakeResult) LastInsertId() (int64, error) {
	return 0, errors.New("not implemented")
}

func (f *fakeResult) RowsAffected() (int64, error) {
	return f.rowsAffected, nil
}

// fakeDB implements DB interface for tests.
type fakeDB struct {
	ExecFn     func(ctx context.Context, query string, args ...any) (sql.Result, error)
	lastQuery  string
	lastArgs   []any
	execCalled bool
}

func (f *fakeDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	f.execCalled = true
	f.lastQuery = query
	f.lastArgs = args
	if f.ExecFn != nil {
		return f.ExecFn(ctx, query, args...)
	}
	return &fakeResult{rowsAffected: 1}, nil
}

// ------------------------------------------------------------
// SUCCESS (created)
// ------------------------------------------------------------

func TestEventRepository_InsertEvent_Created(t *testing.T) {
	db := &fakeDB{
		ExecFn: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			// Basit bir sanity check: doğru tabloya mı insert ediyoruz?
			if !strings.Contains(query, "INSERT INTO events") {
				t.Fatalf("unexpected query: %s", query)
			}
			return &fakeResult{rowsAffected: 1}, nil
		},
	}

	repo := NewEventRepository(db)

	e := &domain.Event{
		EventName:  "product_view",
		Channel:    "web",
		CampaignID: "cmp_1",
		UserID:     "user_1",
		EventTime:  time.Now().UTC(),
		Tags:       []string{"a", "b"},
		Metadata:   map[string]any{"k": "v"},
		DedupeKey:  "dk",
	}

	created, err := repo.InsertEvent(context.Background(), e)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Fatalf("expected created=true, got false")
	}
	if !db.execCalled {
		t.Fatalf("expected ExecContext to be called")
	}
	if len(db.lastArgs) != 8 {
		t.Fatalf("expected 8 args, got %d", len(db.lastArgs))
	}
}

// ------------------------------------------------------------
// DUPLICATE (rowsAffected=0)
// ------------------------------------------------------------

func TestEventRepository_InsertEvent_Duplicate(t *testing.T) {
	db := &fakeDB{
		ExecFn: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			return &fakeResult{rowsAffected: 0}, nil
		},
	}

	repo := NewEventRepository(db)

	e := &domain.Event{
		EventName: "product_view",
		Channel:   "web",
		UserID:    "user_1",
		EventTime: time.Now().UTC(),
		Tags:      []string{},
		Metadata:  map[string]any{},
		DedupeKey: "dk",
	}

	created, err := repo.InsertEvent(context.Background(), e)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Fatalf("expected created=false for duplicate")
	}
}

// ------------------------------------------------------------
// DB ERROR
// ------------------------------------------------------------

func TestEventRepository_InsertEvent_Error(t *testing.T) {
	db := &fakeDB{
		ExecFn: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			return nil, errors.New("db error")
		},
	}

	repo := NewEventRepository(db)

	e := &domain.Event{
		EventName: "product_view",
		Channel:   "web",
		UserID:    "user_1",
		EventTime: time.Now().UTC(),
		Tags:      []string{},
		Metadata:  map[string]any{},
		DedupeKey: "dk",
	}

	created, err := repo.InsertEvent(context.Background(), e)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if created {
		t.Fatalf("expected created=false on error")
	}
}
