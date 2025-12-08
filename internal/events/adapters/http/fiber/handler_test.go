package fiber

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"event-metrics-service/internal/events/core/usecase"

	"github.com/gofiber/fiber/v2"
)

type fakeStoreEventUseCase struct {
	ExecuteFunc         func(ctx context.Context, in usecase.StoreEventInput) (bool, error)
	BulkCreateFunc      func(ctx context.Context, in usecase.BulkCreateEventsInput) (usecase.BulkCreateEventsResult, error)
	LastExecuteInput    usecase.StoreEventInput
	LastBulkCreateInput usecase.BulkCreateEventsInput
}

func (f *fakeStoreEventUseCase) Execute(ctx context.Context, in usecase.StoreEventInput) (bool, error) {
	f.LastExecuteInput = in
	if f.ExecuteFunc != nil {
		return f.ExecuteFunc(ctx, in)
	}
	return false, nil
}

func (f *fakeStoreEventUseCase) BulkCreateEvents(ctx context.Context, in usecase.BulkCreateEventsInput) (usecase.BulkCreateEventsResult, error) {
	f.LastBulkCreateInput = in
	if f.BulkCreateFunc != nil {
		return f.BulkCreateFunc(ctx, in)
	}
	return usecase.BulkCreateEventsResult{}, nil
}

// helper: create fiber app and routes
func setupTestApp(uc StoreEventUseCase) *fiber.App {
	app := fiber.New()
	h := NewEventHandler(uc)

	app.Post("/events", h.CreateEvent)
	app.Post("/events/bulk", h.BulkCreateEvents)

	return app
}

// helper: send request
func doRequest(t *testing.T, app *fiber.App, method, path string, body any) (*http.Response, []byte) {
	t.Helper()

	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal body: %v", err)
		}
		buf = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, path, buf)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	_ = resp.Body.Close()

	return resp, respBody
}

func TestCreateEvent_Success_Created(t *testing.T) {
	now := time.Now().Add(-time.Minute).Unix()

	fakeUC := &fakeStoreEventUseCase{
		ExecuteFunc: func(ctx context.Context, in usecase.StoreEventInput) (bool, error) {
			// created = true
			return true, nil
		},
	}

	app := setupTestApp(fakeUC)

	reqBody := CreateEventRequest{
		EventName:  "product_view",
		Channel:    "web",
		CampaignID: "cmp_1",
		UserID:     "user_123",
		Timestamp:  now,
		Tags:       []string{"electronics"},
		Metadata:   map[string]any{"product_id": "p1"},
	}

	resp, body := doRequest(t, app, http.MethodPost, "/events", reqBody)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d (body: %s)", http.StatusCreated, resp.StatusCode, string(body))
	}

	var respJSON map[string]any
	if err := json.Unmarshal(body, &respJSON); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}

	if respJSON["status"] != "created" {
		t.Errorf("expected status=created, got %v", respJSON["status"])
	}
}

func TestCreateEvent_Success_Duplicate(t *testing.T) {
	now := time.Now().Add(-time.Minute).Unix()

	fakeUC := &fakeStoreEventUseCase{
		ExecuteFunc: func(ctx context.Context, in usecase.StoreEventInput) (bool, error) {
			// created = false → duplicate
			return false, nil
		},
	}

	app := setupTestApp(fakeUC)

	reqBody := CreateEventRequest{
		EventName: "product_view",
		Channel:   "web",
		UserID:    "user_123",
		Timestamp: now,
	}

	resp, body := doRequest(t, app, http.MethodPost, "/events", reqBody)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d (body: %s)", http.StatusOK, resp.StatusCode, string(body))
	}

	var respJSON map[string]any
	if err := json.Unmarshal(body, &respJSON); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}

	if respJSON["status"] != "duplicate" {
		t.Errorf("expected status=duplicate, got %v", respJSON["status"])
	}
}

func TestCreateEvent_InvalidJSON(t *testing.T) {
	fakeUC := &fakeStoreEventUseCase{}
	app := setupTestApp(fakeUC)

	// Undefined JSON
	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewBufferString(`{"event_name":`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("expected status %d, got %d (body: %s)", http.StatusBadRequest, resp.StatusCode, string(body))
	}
}

func TestCreateEvent_ValidationError(t *testing.T) {
	now := time.Now().Add(-time.Minute).Unix()

	fakeUC := &fakeStoreEventUseCase{
		ExecuteFunc: func(ctx context.Context, in usecase.StoreEventInput) (bool, error) {
			return false, usecase.ErrInvalidEvent
		},
	}

	app := setupTestApp(fakeUC)

	reqBody := CreateEventRequest{
		// Empty EventName or usecase return ErrInvalidEvent
		EventName: "",
		Channel:   "web",
		UserID:    "user_123",
		Timestamp: now,
	}

	resp, body := doRequest(t, app, http.MethodPost, "/events", reqBody)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d (body: %s)", http.StatusBadRequest, resp.StatusCode, string(body))
	}

	var respJSON map[string]any
	if err := json.Unmarshal(body, &respJSON); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}

	// Handler Error: "invalid_event", Message: err.Error()
	if respJSON["error"] != "invalid_event" {
		t.Errorf("expected error=%q, got %v", "invalid_event", respJSON["error"])
	}
}

func TestCreateEvent_FutureTimeError(t *testing.T) {
	fakeUC := &fakeStoreEventUseCase{
		ExecuteFunc: func(ctx context.Context, in usecase.StoreEventInput) (bool, error) {
			return false, usecase.ErrFutureTime
		},
	}

	app := setupTestApp(fakeUC)

	reqBody := CreateEventRequest{
		EventName: "product_view",
		Channel:   "web",
		UserID:    "user_123",
		Timestamp: time.Now().Add(time.Hour).Unix(),
	}

	resp, body := doRequest(t, app, http.MethodPost, "/events", reqBody)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d (body: %s)", http.StatusBadRequest, resp.StatusCode, string(body))
	}

	var respJSON map[string]any
	if err := json.Unmarshal(body, &respJSON); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}

	// Handler Error: "invalid_event"
	if respJSON["error"] != "invalid_event" {
		t.Errorf("expected error=%q, got %v", "invalid_event", respJSON["error"])
	}
}

func TestCreateEvent_InternalError(t *testing.T) {
	now := time.Now().Add(-time.Minute).Unix()

	fakeUC := &fakeStoreEventUseCase{
		ExecuteFunc: func(ctx context.Context, in usecase.StoreEventInput) (bool, error) {
			return false, errors.New("db error")
		},
	}

	app := setupTestApp(fakeUC)

	reqBody := CreateEventRequest{
		EventName: "product_view",
		Channel:   "web",
		UserID:    "user_123",
		Timestamp: now,
	}

	resp, body := doRequest(t, app, http.MethodPost, "/events", reqBody)

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d (body: %s)", http.StatusInternalServerError, resp.StatusCode, string(body))
	}

	var respJSON map[string]any
	if err := json.Unmarshal(body, &respJSON); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}

	if respJSON["error"] != "internal_server_error" {
		t.Errorf("expected error=internal_server_error, got %v", respJSON["error"])
	}
}

// ---- Bulk tests ----

func TestBulkCreateEvents_Success_AllCreated(t *testing.T) {
	now := time.Now().Add(-time.Minute).Unix()

	fakeUC := &fakeStoreEventUseCase{
		BulkCreateFunc: func(ctx context.Context, in usecase.BulkCreateEventsInput) (usecase.BulkCreateEventsResult, error) {
			return usecase.BulkCreateEventsResult{
				Created:    len(in.Events),
				Duplicates: 0,
			}, nil
		},
	}

	app := setupTestApp(fakeUC)

	reqBody := BulkCreateEventsRequest{
		Events: []bulkEventItem{
			{
				EventName:  "product_view",
				Channel:    "web",
				CampaignID: "cmp_1",
				UserID:     "u1",
				Timestamp:  now,
				Tags:       []string{"electronics"},
				Metadata:   map[string]any{"product_id": "p1"},
			},
			{
				EventName:  "add_to_cart",
				Channel:    "web",
				CampaignID: "cmp_2",
				UserID:     "u2",
				Timestamp:  now,
				Tags:       []string{"cart"},
				Metadata:   map[string]any{"product_id": "p2"},
			},
		},
	}

	resp, body := doRequest(t, app, http.MethodPost, "/events/bulk", reqBody)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d (body: %s)", http.StatusCreated, resp.StatusCode, string(body))
	}

	var respJSON map[string]any
	if err := json.Unmarshal(body, &respJSON); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}

	if int(respJSON["created"].(float64)) != 2 {
		t.Errorf("expected created=2, got %v", respJSON["created"])
	}
	if int(respJSON["duplicates"].(float64)) != 0 {
		t.Errorf("expected duplicates=0, got %v", respJSON["duplicates"])
	}
}

func TestBulkCreateEvents_MixedCreatedAndDuplicate(t *testing.T) {
	now := time.Now().Add(-time.Minute).Unix()

	fakeUC := &fakeStoreEventUseCase{
		BulkCreateFunc: func(ctx context.Context, in usecase.BulkCreateEventsInput) (usecase.BulkCreateEventsResult, error) {
			return usecase.BulkCreateEventsResult{
				Created:    1,
				Duplicates: 1,
			}, nil
		},
	}

	app := setupTestApp(fakeUC)

	reqBody := BulkCreateEventsRequest{
		Events: []bulkEventItem{
			{
				EventName: "product_view",
				Channel:   "web",
				UserID:    "u1",
				Timestamp: now,
			},
			{
				EventName: "product_view",
				Channel:   "web",
				UserID:    "u1",
				Timestamp: now,
			},
		},
	}

	resp, body := doRequest(t, app, http.MethodPost, "/events/bulk", reqBody)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d (body: %s)", http.StatusCreated, resp.StatusCode, string(body))
	}

	var respJSON map[string]any
	if err := json.Unmarshal(body, &respJSON); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}

	if int(respJSON["created"].(float64)) != 1 {
		t.Errorf("expected created=1, got %v", respJSON["created"])
	}
	if int(respJSON["duplicates"].(float64)) != 1 {
		t.Errorf("expected duplicates=1, got %v", respJSON["duplicates"])
	}
}

func TestBulkCreateEvents_InvalidJSON(t *testing.T) {
	fakeUC := &fakeStoreEventUseCase{}
	app := setupTestApp(fakeUC)

	req := httptest.NewRequest(http.MethodPost, "/events/bulk", bytes.NewBufferString(`{"events":[`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("expected status %d, got %d (body: %s)", http.StatusBadRequest, resp.StatusCode, string(body))
	}
}

func TestBulkCreateEvents_EmptyEvents(t *testing.T) {
	fakeUC := &fakeStoreEventUseCase{}
	app := setupTestApp(fakeUC)

	reqBody := BulkCreateEventsRequest{
		Events: []bulkEventItem{},
	}

	resp, body := doRequest(t, app, http.MethodPost, "/events/bulk", reqBody)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d (body: %s)", http.StatusBadRequest, resp.StatusCode, string(body))
	}

	var respJSON map[string]any
	if err := json.Unmarshal(body, &respJSON); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}

	if respJSON["error"] != "events_list_required" {
		t.Errorf("expected error=events_list_required, got %v", respJSON["error"])
	}
}

func TestBulkCreateEvents_ValidationError(t *testing.T) {
	fakeUC := &fakeStoreEventUseCase{
		BulkCreateFunc: func(ctx context.Context, in usecase.BulkCreateEventsInput) (usecase.BulkCreateEventsResult, error) {
			return usecase.BulkCreateEventsResult{}, usecase.ErrInvalidEvent
		},
	}

	app := setupTestApp(fakeUC)

	reqBody := BulkCreateEventsRequest{
		Events: []bulkEventItem{
			{
				EventName: "",
				Channel:   "web",
				UserID:    "u1",
				Timestamp: time.Now().Unix(),
			},
		},
	}

	resp, body := doRequest(t, app, http.MethodPost, "/events/bulk", reqBody)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d (body: %s)", http.StatusBadRequest, resp.StatusCode, string(body))
	}

	var respJSON map[string]any
	if err := json.Unmarshal(body, &respJSON); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}

	// Handler: Error: "invalid_event"
	if respJSON["error"] != "invalid_event" {
		t.Errorf("expected error=%q, got %v", "invalid_event", respJSON["error"])
	}
}

func TestBulkCreateEvents_InternalError(t *testing.T) {
	fakeUC := &fakeStoreEventUseCase{
		BulkCreateFunc: func(ctx context.Context, in usecase.BulkCreateEventsInput) (usecase.BulkCreateEventsResult, error) {
			return usecase.BulkCreateEventsResult{}, errors.New("db error")
		},
	}

	app := setupTestApp(fakeUC)

	now := time.Now().Add(-time.Minute).Unix()

	reqBody := BulkCreateEventsRequest{
		Events: []bulkEventItem{
			{
				EventName: "product_view",
				Channel:   "web",
				UserID:    "u1",
				Timestamp: now,
			},
		},
	}

	resp, body := doRequest(t, app, http.MethodPost, "/events/bulk", reqBody)

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d (body: %s)", http.StatusInternalServerError, resp.StatusCode, string(body))
	}

	var respJSON map[string]any
	if err := json.Unmarshal(body, &respJSON); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}

	if respJSON["error"] != "internal_server_error" {
		t.Errorf("expected error=internal_server_error, got %v", respJSON["error"])
	}
}
