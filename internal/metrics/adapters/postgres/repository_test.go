package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"event-metrics-service/internal/metrics/core/ports"
)

// fakeRowScanner implements RowScanner for tests.
type fakeRowScanner struct {
	rows []fakeRow
	i    int
	err  error
}

type fakeRow struct {
	values []any
}

func (f *fakeRowScanner) Next() bool {
	return f.i < len(f.rows)
}

func (f *fakeRowScanner) Scan(dest ...any) error {
	if f.i >= len(f.rows) {
		return errors.New("no more rows")
	}
	row := f.rows[f.i]
	if len(dest) != len(row.values) {
		return errors.New("dest length mismatch")
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *int64:
			v, ok := row.values[i].(int64)
			if !ok {
				return errors.New("type assertion to int64 failed")
			}
			*d = v
		case *string:
			v, ok := row.values[i].(string)
			if !ok {
				return errors.New("type assertion to string failed")
			}
			*d = v
		case *time.Time:
			v, ok := row.values[i].(time.Time)
			if !ok {
				return errors.New("type assertion to time.Time failed")
			}
			*d = v
		default:
			return errors.New("unsupported dest type")
		}
	}
	f.i++
	return nil
}

func (f *fakeRowScanner) Err() error {
	return f.err
}

func (f *fakeRowScanner) Close() error {
	return nil
}

// fakeDB implements DB interface.
type fakeDB struct {
	QueryFn   func(ctx context.Context, query string, args ...any) (RowScanner, error)
	lastQuery string
	lastArgs  []any
	called    bool
}

func (f *fakeDB) QueryContext(ctx context.Context, query string, args ...any) (RowScanner, error) {
	f.called = true
	f.lastQuery = query
	f.lastArgs = args
	if f.QueryFn != nil {
		return f.QueryFn(ctx, query, args...)
	}
	return nil, nil
}

// ------------------------------------------------------------
// NO GROUP BY
// ------------------------------------------------------------

func TestMetricsRepository_NoGroupBy(t *testing.T) {
	db := &fakeDB{
		QueryFn: func(ctx context.Context, query string, args ...any) (RowScanner, error) {
			if !strings.Contains(query, "FROM events") {
				t.Fatalf("unexpected query: %s", query)
			}
			// Tek satır: total=150, unique=40
			return &fakeRowScanner{
				rows: []fakeRow{
					{values: []any{int64(150), int64(40)}},
				},
			}, nil
		},
	}

	repo := NewMetricsRepository(db)

	filter := ports.MetricsFilter{
		EventName: "product_view",
		From:      100,
		To:        200,
	}

	res, err := repo.QueryMetrics(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !db.called {
		t.Fatalf("expected QueryContext to be called")
	}
	if res.TotalCount != 150 || res.UniqueUsers != 40 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if res.GroupBy != "" {
		t.Fatalf("expected empty group_by, got %s", res.GroupBy)
	}
}

// ------------------------------------------------------------
// GROUP BY CHANNEL
// ------------------------------------------------------------

func TestMetricsRepository_GroupByChannel(t *testing.T) {
	db := &fakeDB{
		QueryFn: func(ctx context.Context, query string, args ...any) (RowScanner, error) {
			if !strings.Contains(query, "GROUP BY channel") {
				t.Fatalf("expected GROUP BY channel in query, got: %s", query)
			}
			return &fakeRowScanner{
				rows: []fakeRow{
					{values: []any{"mobile", int64(80), int64(30)}},
					{values: []any{"web", int64(120), int64(50)}},
				},
			}, nil
		},
	}

	repo := NewMetricsRepository(db)

	filter := ports.MetricsFilter{
		EventName: "product_view",
		From:      100,
		To:        200,
		GroupBy:   "channel",
	}

	res, err := repo.QueryMetrics(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.GroupBy != "channel" {
		t.Fatalf("expected group_by=channel, got %s", res.GroupBy)
	}
	if len(res.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(res.Groups))
	}

	// Toplamları, satırlardan hesaplanan toplamla karşılaştıralım
	if res.TotalCount != 200 {
		t.Fatalf("expected total_count=200, got %d", res.TotalCount)
	}
	if res.UniqueUsers != 80 {
		t.Fatalf("expected unique_users=80, got %d", res.UniqueUsers)
	}
}

// ------------------------------------------------------------
// GROUP BY TIME (hour)
// ------------------------------------------------------------

func TestMetricsRepository_GroupByTime(t *testing.T) {
	db := &fakeDB{
		QueryFn: func(ctx context.Context, query string, args ...any) (RowScanner, error) {
			if !strings.Contains(query, "date_trunc('hour'") {
				t.Fatalf("expected date_trunc('hour', ...) in query, got: %s", query)
			}

			t1 := time.Date(2025, 12, 7, 10, 0, 0, 0, time.UTC)
			t2 := time.Date(2025, 12, 7, 11, 0, 0, 0, time.UTC)

			return &fakeRowScanner{
				rows: []fakeRow{
					{values: []any{t1, int64(100), int64(40)}},
					{values: []any{t2, int64(200), int64(60)}},
				},
			}, nil
		},
	}

	repo := NewMetricsRepository(db)

	filter := ports.MetricsFilter{
		EventName: "product_view",
		From:      100,
		To:        200,
		GroupBy:   "time",
		Interval:  "hour",
	}

	res, err := repo.QueryMetrics(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.GroupBy != "time" {
		t.Fatalf("expected group_by=time, got %s", res.GroupBy)
	}
	if len(res.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(res.Groups))
	}
	if res.TotalCount != 300 {
		t.Fatalf("expected total_count=300, got %d", res.TotalCount)
	}
	if res.UniqueUsers != 100 {
		t.Fatalf("expected unique_users=100, got %d", res.UniqueUsers)
	}

	// key format RFC3339 mi?
	for _, g := range res.Groups {
		if _, err := time.Parse(time.RFC3339, g.Key); err != nil {
			t.Fatalf("expected RFC3339 key, got %s (%v)", g.Key, err)
		}
	}
}

// ------------------------------------------------------------
// DB ERROR
// ------------------------------------------------------------

func TestMetricsRepository_DBError(t *testing.T) {
	db := &fakeDB{
		QueryFn: func(ctx context.Context, query string, args ...any) (RowScanner, error) {
			return nil, errors.New("db failure")
		},
	}

	repo := NewMetricsRepository(db)

	filter := ports.MetricsFilter{
		EventName: "product_view",
		From:      100,
		To:        200,
	}

	res, err := repo.QueryMetrics(context.Background(), filter)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "db failure" {
		t.Fatalf("expected db failure, got %v", err)
	}
	if res != nil {
		t.Fatalf("expected nil result on error")
	}
}
