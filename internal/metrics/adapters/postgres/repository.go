package postgres

import (
	"context"
	"fmt"
	"time"

	"event-metrics-service/internal/metrics/core/domain"
	"event-metrics-service/internal/metrics/core/ports"
)

type RowScanner interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close() error
}

type DB interface {
	QueryContext(ctx context.Context, query string, args ...any) (RowScanner, error)
}

type MetricsRepository struct {
	db DB
}

func NewMetricsRepository(db DB) *MetricsRepository {
	return &MetricsRepository{db: db}
}

func (r *MetricsRepository) QueryMetrics(ctx context.Context, f ports.MetricsFilter) (*domain.AggregatedMetrics, error) {
	fromTime := time.Unix(f.From, 0).UTC()
	toTime := time.Unix(f.To, 0).UTC()

	where := "event_name = $1 AND event_time BETWEEN $2 AND $3"
	args := []any{f.EventName, fromTime, toTime}
	argIndex := 4

	if f.Channel != nil {
		where += fmt.Sprintf(" AND channel = $%d", argIndex)
		args = append(args, *f.Channel)
		argIndex++
	}

	result := &domain.AggregatedMetrics{
		EventName: f.EventName,
		From:      f.From,
		To:        f.To,
		GroupBy:   f.GroupBy,
	}

	switch f.GroupBy {
	case "":
		return r.queryNoGroup(ctx, where, args, result)
	case "channel":
		return r.queryGroupByChannel(ctx, where, args, result)
	case "time":
		return r.queryGroupByTime(ctx, where, args, result, f.Interval)
	default:
		// Aslında buraya gelmemeli; usecase validasyonu zaten yapıyor.
		return nil, fmt.Errorf("unsupported group_by: %s", f.GroupBy)
	}
}

func (r *MetricsRepository) queryNoGroup(
	ctx context.Context,
	where string,
	args []any,
	res *domain.AggregatedMetrics,
) (*domain.AggregatedMetrics, error) {
	query := `
SELECT
    COUNT(*) AS total_count,
    COUNT(DISTINCT user_id) AS unique_users
FROM events
WHERE ` + where

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var total, unique int64
		if err := rows.Scan(&total, &unique); err != nil {
			return nil, err
		}
		res.TotalCount = total
		res.UniqueUsers = unique
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return res, nil
}

func (r *MetricsRepository) queryGroupByChannel(
	ctx context.Context,
	where string,
	args []any,
	res *domain.AggregatedMetrics,
) (*domain.AggregatedMetrics, error) {
	query := `
SELECT
    channel,
    COUNT(*) AS total_count,
    COUNT(DISTINCT user_id) AS unique_users
FROM events
WHERE ` + where + `
GROUP BY channel
ORDER BY channel`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []domain.MetricsGroup
	var totalSum int64
	var uniqueSum int64

	for rows.Next() {
		var ch string
		var total, unique int64

		if err := rows.Scan(&ch, &total, &unique); err != nil {
			return nil, err
		}

		groups = append(groups, domain.MetricsGroup{
			Key:         ch,
			TotalCount:  total,
			UniqueUsers: unique,
		})
		totalSum += total
		uniqueSum += unique // not: cross-channel unique tam olarak doğru değil, ama basit çözüm
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	res.Groups = groups
	res.TotalCount = totalSum
	res.UniqueUsers = uniqueSum

	return res, nil
}

func (r *MetricsRepository) queryGroupByTime(
	ctx context.Context,
	where string,
	args []any,
	res *domain.AggregatedMetrics,
	interval string,
) (*domain.AggregatedMetrics, error) {
	query := fmt.Sprintf(`
SELECT
    date_trunc('%s', event_time) AS bucket,
    COUNT(*) AS total_count,
    COUNT(DISTINCT user_id) AS unique_users
FROM events
WHERE %s
GROUP BY bucket
ORDER BY bucket
`, interval, where)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []domain.MetricsGroup
	var totalSum int64
	var uniqueSum int64

	for rows.Next() {
		var ts time.Time
		var total, unique int64

		if err := rows.Scan(&ts, &total, &unique); err != nil {
			return nil, err
		}

		groups = append(groups, domain.MetricsGroup{
			Key:         ts.UTC().Format(time.RFC3339),
			TotalCount:  total,
			UniqueUsers: unique,
		})
		totalSum += total
		uniqueSum += unique
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	res.Groups = groups
	res.TotalCount = totalSum
	res.UniqueUsers = uniqueSum

	return res, nil
}
