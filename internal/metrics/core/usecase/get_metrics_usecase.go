package usecase

import (
	"context"
	"errors"

	"event-metrics-service/internal/metrics/core/domain"
	"event-metrics-service/internal/metrics/core/ports"
)

var (
	ErrInvalidMetricsQuery = errors.New("invalid metrics query")
	ErrInvalidTimeRange    = errors.New("invalid time range")
	ErrInvalidGroupBy      = errors.New("invalid group_by value")
	ErrInvalidInterval     = errors.New("invalid interval for time grouping")
)

type GetMetricsInput struct {
	EventName string
	From      int64
	To        int64

	Channel  *string
	GroupBy  string // "", "channel", "time"
	Interval string // "hour" / "day" (group_by=time ise zorunlu)
}

type GetMetricsUseCase struct {
	reader ports.MetricsReaderPort
}

func NewGetMetricsUseCase(reader ports.MetricsReaderPort) *GetMetricsUseCase {
	return &GetMetricsUseCase{reader: reader}
}

// Execute, input'u doğrular, filter'a çevirir ve MetricsReaderPort'u çağırır.
func (uc *GetMetricsUseCase) Execute(ctx context.Context, in GetMetricsInput) (*domain.AggregatedMetrics, error) {

	if in.EventName == "" {
		return nil, ErrInvalidMetricsQuery
	}

	if in.From <= 0 || in.To <= 0 || in.From > in.To {
		return nil, ErrInvalidTimeRange
	}

	switch in.GroupBy {
	case "":
		// no group
	case "channel":
		// valid
	case "time":
		// interval required and only "hour" / "day"
		if in.Interval != "hour" && in.Interval != "day" {
			return nil, ErrInvalidInterval
		}
	default:
		return nil, ErrInvalidGroupBy
	}

	filter := ports.MetricsFilter{
		EventName: in.EventName,
		From:      in.From,
		To:        in.To,
		Channel:   in.Channel,
		GroupBy:   in.GroupBy,
		Interval:  in.Interval,
	}

	result, err := uc.reader.QueryMetrics(ctx, filter)
	if err != nil {
		return nil, err
	}

	return result, nil
}
