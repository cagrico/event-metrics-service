package usecase_test

import (
	"context"
	"errors"
	"testing"

	"event-metrics-service/internal/metrics/core/domain"
	"event-metrics-service/internal/metrics/core/ports"
	"event-metrics-service/internal/metrics/core/usecase"
)

// fakeMetricsReader, MetricsReaderPort'u test için fake'ler.
type fakeMetricsReader struct {
	QueryFn    func(ctx context.Context, f ports.MetricsFilter) (*domain.AggregatedMetrics, error)
	lastFilter ports.MetricsFilter
	called     bool
}

func (f *fakeMetricsReader) QueryMetrics(ctx context.Context, flt ports.MetricsFilter) (*domain.AggregatedMetrics, error) {
	f.called = true
	f.lastFilter = flt
	if f.QueryFn != nil {
		return f.QueryFn(ctx, flt)
	}
	return nil, nil
}

// ------------------------------------------------------------
// SUCCESS (no group_by)
// ------------------------------------------------------------

func TestGetMetrics_Success_NoGroupBy(t *testing.T) {
	reader := &fakeMetricsReader{
		QueryFn: func(ctx context.Context, flt ports.MetricsFilter) (*domain.AggregatedMetrics, error) {
			if flt.EventName != "product_view" {
				t.Fatalf("expected event_name=product_view, got %s", flt.EventName)
			}
			if flt.From != 100 || flt.To != 200 {
				t.Fatalf("expected from=100,to=200, got from=%d,to=%d", flt.From, flt.To)
			}
			if flt.GroupBy != "" {
				t.Fatalf("expected group_by empty, got %s", flt.GroupBy)
			}
			// Channel filter boş (nil) olmalı
			if flt.Channel != nil {
				t.Fatalf("expected channel=nil, got %v", *flt.Channel)
			}

			return &domain.AggregatedMetrics{
				EventName:   flt.EventName,
				From:        flt.From,
				To:          flt.To,
				TotalCount:  150,
				UniqueUsers: 40,
				GroupBy:     "",
				Groups:      nil,
			}, nil
		},
	}

	uc := usecase.NewGetMetricsUseCase(reader)

	in := usecase.GetMetricsInput{
		EventName: "product_view",
		From:      100,
		To:        200,
	}

	out, err := uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatalf("expected non-nil result")
	}
	if out.TotalCount != 150 || out.UniqueUsers != 40 {
		t.Fatalf("unexpected result: %+v", out)
	}
	if !reader.called {
		t.Fatalf("expected QueryMetrics to be called")
	}
}

// ------------------------------------------------------------
// SUCCESS (group_by=channel)
// ------------------------------------------------------------

func TestGetMetrics_Success_GroupByChannel(t *testing.T) {
	reader := &fakeMetricsReader{
		QueryFn: func(ctx context.Context, flt ports.MetricsFilter) (*domain.AggregatedMetrics, error) {
			if flt.GroupBy != "channel" {
				t.Fatalf("expected group_by=channel, got %s", flt.GroupBy)
			}
			return &domain.AggregatedMetrics{
				EventName:   flt.EventName,
				From:        flt.From,
				To:          flt.To,
				TotalCount:  200,
				UniqueUsers: 80,
				GroupBy:     "channel",
				Groups: []domain.MetricsGroup{
					{Key: "web", TotalCount: 120, UniqueUsers: 50},
					{Key: "mobile", TotalCount: 80, UniqueUsers: 30},
				},
			}, nil
		},
	}

	uc := usecase.NewGetMetricsUseCase(reader)

	in := usecase.GetMetricsInput{
		EventName: "product_view",
		From:      100,
		To:        200,
		GroupBy:   "channel",
	}

	out, err := uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.GroupBy != "channel" {
		t.Fatalf("expected group_by=channel, got %s", out.GroupBy)
	}
	if len(out.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(out.Groups))
	}
}

// ------------------------------------------------------------
// SUCCESS (group_by=time, interval=hour)
// ------------------------------------------------------------

func TestGetMetrics_Success_GroupByTime(t *testing.T) {
	reader := &fakeMetricsReader{
		QueryFn: func(ctx context.Context, flt ports.MetricsFilter) (*domain.AggregatedMetrics, error) {
			if flt.GroupBy != "time" {
				t.Fatalf("expected group_by=time, got %s", flt.GroupBy)
			}
			if flt.Interval != "hour" {
				t.Fatalf("expected interval=hour, got %s", flt.Interval)
			}
			return &domain.AggregatedMetrics{
				EventName:   flt.EventName,
				From:        flt.From,
				To:          flt.To,
				TotalCount:  300,
				UniqueUsers: 100,
				GroupBy:     "time",
				Groups: []domain.MetricsGroup{
					{Key: "2025-12-07T10:00:00Z", TotalCount: 100, UniqueUsers: 40},
					{Key: "2025-12-07T11:00:00Z", TotalCount: 200, UniqueUsers: 60},
				},
			}, nil
		},
	}

	uc := usecase.NewGetMetricsUseCase(reader)

	in := usecase.GetMetricsInput{
		EventName: "product_view",
		From:      100,
		To:        200,
		GroupBy:   "time",
		Interval:  "hour",
	}

	out, err := uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.GroupBy != "time" {
		t.Fatalf("expected group_by=time, got %s", out.GroupBy)
	}
	if len(out.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(out.Groups))
	}
}

// ------------------------------------------------------------
// VALIDATION: missing event_name
// ------------------------------------------------------------

func TestGetMetrics_InvalidEventName(t *testing.T) {
	reader := &fakeMetricsReader{}
	uc := usecase.NewGetMetricsUseCase(reader)

	in := usecase.GetMetricsInput{
		EventName: "",
		From:      100,
		To:        200,
	}

	out, err := uc.Execute(context.Background(), in)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, usecase.ErrInvalidMetricsQuery) {
		t.Fatalf("expected ErrInvalidMetricsQuery, got %v", err)
	}
	if out != nil {
		t.Fatalf("expected nil result on error")
	}
	if reader.called {
		t.Fatalf("repository should not be called on invalid input")
	}
}

// ------------------------------------------------------------
// VALIDATION: from > to
// ------------------------------------------------------------

func TestGetMetrics_InvalidTimeRange(t *testing.T) {
	reader := &fakeMetricsReader{}
	uc := usecase.NewGetMetricsUseCase(reader)

	in := usecase.GetMetricsInput{
		EventName: "product_view",
		From:      200,
		To:        100,
	}

	out, err := uc.Execute(context.Background(), in)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, usecase.ErrInvalidTimeRange) {
		t.Fatalf("expected ErrInvalidTimeRange, got %v", err)
	}
	if out != nil {
		t.Fatalf("expected nil result on error")
	}
	if reader.called {
		t.Fatalf("repository should not be called on invalid time range")
	}
}

// ------------------------------------------------------------
// VALIDATION: group_by=time ama interval boş/geçersiz
// ------------------------------------------------------------

func TestGetMetrics_InvalidIntervalForTimeGroup(t *testing.T) {
	reader := &fakeMetricsReader{}
	uc := usecase.NewGetMetricsUseCase(reader)

	// interval boş
	in := usecase.GetMetricsInput{
		EventName: "product_view",
		From:      100,
		To:        200,
		GroupBy:   "time",
		Interval:  "",
	}

	out, err := uc.Execute(context.Background(), in)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, usecase.ErrInvalidInterval) {
		t.Fatalf("expected ErrInvalidInterval, got %v", err)
	}
	if out != nil {
		t.Fatalf("expected nil result")
	}
	if reader.called {
		t.Fatalf("repository should not be called on invalid interval")
	}
}

// ------------------------------------------------------------
// VALIDATION: group_by bilinmeyen değer
// ------------------------------------------------------------

func TestGetMetrics_InvalidGroupBy(t *testing.T) {
	reader := &fakeMetricsReader{}
	uc := usecase.NewGetMetricsUseCase(reader)

	in := usecase.GetMetricsInput{
		EventName: "product_view",
		From:      100,
		To:        200,
		GroupBy:   "something_else",
	}

	out, err := uc.Execute(context.Background(), in)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, usecase.ErrInvalidGroupBy) {
		t.Fatalf("expected ErrInvalidGroupBy, got %v", err)
	}
	if out != nil {
		t.Fatalf("expected nil result")
	}
	if reader.called {
		t.Fatalf("repository should not be called on invalid group_by")
	}
}

// ------------------------------------------------------------
// REPOSITORY ERROR PROPAGATION
// ------------------------------------------------------------

func TestGetMetrics_RepositoryError(t *testing.T) {
	reader := &fakeMetricsReader{
		QueryFn: func(ctx context.Context, f ports.MetricsFilter) (*domain.AggregatedMetrics, error) {
			return nil, errors.New("db failure")
		},
	}

	uc := usecase.NewGetMetricsUseCase(reader)

	in := usecase.GetMetricsInput{
		EventName: "product_view",
		From:      100,
		To:        200,
	}

	out, err := uc.Execute(context.Background(), in)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "db failure" {
		t.Fatalf("expected db failure, got %v", err)
	}
	if out != nil {
		t.Fatalf("expected nil result on error")
	}
}
