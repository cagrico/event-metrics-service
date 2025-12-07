package fiber_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	httpadapter "event-metrics-service/internal/metrics/adapters/http/fiber"
	"event-metrics-service/internal/metrics/core/domain"
	"event-metrics-service/internal/metrics/core/usecase"

	"github.com/gofiber/fiber/v2"
)

// Fake usecase implementing the interface that handler depends on.
type fakeGetMetricsUseCase struct {
	ExecuteFn func(ctx context.Context, in usecase.GetMetricsInput) (*domain.AggregatedMetrics, error)
	lastInput usecase.GetMetricsInput
	called    bool
}

func (f *fakeGetMetricsUseCase) Execute(ctx context.Context, in usecase.GetMetricsInput) (*domain.AggregatedMetrics, error) {
	f.called = true
	f.lastInput = in
	if f.ExecuteFn != nil {
		return f.ExecuteFn(ctx, in)
	}
	return nil, nil
}

func setupApp(t *testing.T, uc httpadapter.GetMetricsUseCase) *fiber.App {
	t.Helper()
	app := fiber.New()
	h := httpadapter.NewMetricsHandler(uc)
	app.Get("/metrics", h.GetMetrics)
	return app
}

// ------------------------------------------------------------
// SUCCESS: no group_by
// ------------------------------------------------------------

func TestGetMetrics_Success_NoGroupBy(t *testing.T) {
	uc := &fakeGetMetricsUseCase{
		ExecuteFn: func(ctx context.Context, in usecase.GetMetricsInput) (*domain.AggregatedMetrics, error) {
			if in.EventName != "product_view" {
				t.Fatalf("expected event_name=product_view, got %s", in.EventName)
			}
			if in.From != 100 || in.To != 200 {
				t.Fatalf("expected from=100,to=200 got from=%d,to=%d", in.From, in.To)
			}
			return &domain.AggregatedMetrics{
				EventName:   in.EventName,
				From:        in.From,
				To:          in.To,
				TotalCount:  150,
				UniqueUsers: 40,
				GroupBy:     "",
				Groups:      nil,
			}, nil
		},
	}

	app := setupApp(t, uc)

	params := url.Values{}
	params.Set("event_name", "product_view")
	params.Set("from", "100")
	params.Set("to", "200")

	req := httptest.NewRequest(http.MethodGet, "/metrics?"+params.Encode(), nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	ucCalled := uc.called
	if !ucCalled {
		t.Fatalf("expected usecase to be called")
	}
}

// ------------------------------------------------------------
// SUCCESS: group_by=channel
// ------------------------------------------------------------

func TestGetMetrics_Success_GroupByChannel(t *testing.T) {
	uc := &fakeGetMetricsUseCase{
		ExecuteFn: func(ctx context.Context, in usecase.GetMetricsInput) (*domain.AggregatedMetrics, error) {
			if in.GroupBy != "channel" {
				t.Fatalf("expected group_by=channel, got %s", in.GroupBy)
			}
			return &domain.AggregatedMetrics{
				EventName:   in.EventName,
				From:        in.From,
				To:          in.To,
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

	app := setupApp(t, uc)

	params := url.Values{}
	params.Set("event_name", "product_view")
	params.Set("from", "100")
	params.Set("to", "200")
	params.Set("group_by", "channel")

	req := httptest.NewRequest(http.MethodGet, "/metrics?"+params.Encode(), nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

// ------------------------------------------------------------
// SUCCESS: group_by=time&interval=hour
// ------------------------------------------------------------

func TestGetMetrics_Success_GroupByTime(t *testing.T) {
	uc := &fakeGetMetricsUseCase{
		ExecuteFn: func(ctx context.Context, in usecase.GetMetricsInput) (*domain.AggregatedMetrics, error) {
			if in.GroupBy != "time" || in.Interval != "hour" {
				t.Fatalf("expected group_by=time, interval=hour got group_by=%s, interval=%s", in.GroupBy, in.Interval)
			}
			return &domain.AggregatedMetrics{
				EventName:   in.EventName,
				From:        in.From,
				To:          in.To,
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

	app := setupApp(t, uc)

	params := url.Values{}
	params.Set("event_name", "product_view")
	params.Set("from", "100")
	params.Set("to", "200")
	params.Set("group_by", "time")
	params.Set("interval", "hour")

	req := httptest.NewRequest(http.MethodGet, "/metrics?"+params.Encode(), nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

// ------------------------------------------------------------
// INVALID QUERY PARAM (bad int)
// ------------------------------------------------------------

func TestGetMetrics_InvalidQueryParam(t *testing.T) {
	uc := &fakeGetMetricsUseCase{
		ExecuteFn: func(ctx context.Context, in usecase.GetMetricsInput) (*domain.AggregatedMetrics, error) {
			t.Fatalf("usecase should not be called on invalid query params")
			return nil, nil
		},
	}

	app := setupApp(t, uc)

	params := url.Values{}
	params.Set("event_name", "product_view")
	params.Set("from", "abc") // invalid
	params.Set("to", "200")

	req := httptest.NewRequest(http.MethodGet, "/metrics?"+params.Encode(), nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
}

// ------------------------------------------------------------
// USECASE-LEVEL VALIDATION ERRORS -> 400
// ------------------------------------------------------------

func TestGetMetrics_UsecaseValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		ucError error
	}{
		{"invalid_query", usecase.ErrInvalidMetricsQuery},
		{"invalid_time_range", usecase.ErrInvalidTimeRange},
		{"invalid_group_by", usecase.ErrInvalidGroupBy},
		{"invalid_interval", usecase.ErrInvalidInterval},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &fakeGetMetricsUseCase{
				ExecuteFn: func(ctx context.Context, in usecase.GetMetricsInput) (*domain.AggregatedMetrics, error) {
					return nil, tt.ucError
				},
			}

			app := setupApp(t, uc)

			params := url.Values{}
			params.Set("event_name", "product_view")
			params.Set("from", "100")
			params.Set("to", "200")

			req := httptest.NewRequest(http.MethodGet, "/metrics?"+params.Encode(), nil)

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test error: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", resp.StatusCode)
			}
		})
	}
}

// ------------------------------------------------------------
// USECASE OTHER ERROR -> 500
// ------------------------------------------------------------

func TestGetMetrics_InternalError(t *testing.T) {
	uc := &fakeGetMetricsUseCase{
		ExecuteFn: func(ctx context.Context, in usecase.GetMetricsInput) (*domain.AggregatedMetrics, error) {
			return nil, context.DeadlineExceeded // herhangi bir 5xx kabul edilebilir hata
		},
	}

	app := setupApp(t, uc)

	params := url.Values{}
	params.Set("event_name", "product_view")
	params.Set("from", "100")
	params.Set("to", "200")

	req := httptest.NewRequest(http.MethodGet, "/metrics?"+params.Encode(), nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", resp.StatusCode)
	}
}
