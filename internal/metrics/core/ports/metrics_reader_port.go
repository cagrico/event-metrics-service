package ports

import (
	"context"

	"event-metrics-service/internal/metrics/core/domain"
)

type MetricsFilter struct {
	EventName string
	From      int64
	To        int64
	Channel   *string // optional
	GroupBy   string  // "", "channel", "time"
	Interval  string  // "hour" / "day" (GroupBy = "time" required)
}

type MetricsReaderPort interface {
	QueryMetrics(ctx context.Context, f MetricsFilter) (*domain.AggregatedMetrics, error)
}
