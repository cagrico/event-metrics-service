package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"event-metrics-service/internal/events/core/domain"
	"event-metrics-service/internal/events/core/usecase"
)

// Fake repository implementing EventRepositoryPort
type fakeEventRepo struct {
	InsertFn func(ctx context.Context, e *domain.Event) (bool, error)
}

func (f *fakeEventRepo) InsertEvent(ctx context.Context, e *domain.Event) (bool, error) {
	return f.InsertFn(ctx, e)
}

// ------------------------------------------------------------
// SUCCESS TEST
// ------------------------------------------------------------
func TestStoreEvent_Success(t *testing.T) {
	called := false

	repo := &fakeEventRepo{
		InsertFn: func(ctx context.Context, e *domain.Event) (bool, error) {
			called = true

			if e.EventName != "product_view" {
				t.Fatalf("expected event_name 'product_view', got %s", e.EventName)
			}
			if e.Channel != "web" {
				t.Fatalf("expected channel 'web', got %s", e.Channel)
			}
			if e.UserID != "user_123" {
				t.Fatalf("expected user 'user_123', got %s", e.UserID)
			}
			if e.DedupeKey == "" {
				t.Fatalf("expected dedupe key, got empty")
			}

			return true, nil
		},
	}

	uc := usecase.NewStoreEventUseCase(repo)

	input := usecase.StoreEventInput{
		EventName: "product_view",
		Channel:   "web",
		UserID:    "user_123",
		Timestamp: time.Now().Unix(),
	}

	created, err := uc.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Fatalf("expected created=true, got false")
	}
	if !called {
		t.Fatalf("repository InsertEvent was not called")
	}
}

// ------------------------------------------------------------
// INVALID EVENT NAME
// ------------------------------------------------------------
func TestStoreEvent_InvalidEventName(t *testing.T) {
	repo := &fakeEventRepo{}

	uc := usecase.NewStoreEventUseCase(repo)

	input := usecase.StoreEventInput{
		EventName: "",
		Channel:   "web",
		UserID:    "user_123",
		Timestamp: time.Now().Unix(),
	}

	created, err := uc.Execute(context.Background(), input)

	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if created {
		t.Fatalf("expected created=false for invalid event")
	}
	if !errors.Is(err, usecase.ErrInvalidEvent) {
		t.Fatalf("expected ErrInvalidEvent, got %v", err)
	}
}

// ------------------------------------------------------------
// INVALID USER OR CHANNEL
// ------------------------------------------------------------
func TestStoreEvent_InvalidUserOrChannel(t *testing.T) {
	repo := &fakeEventRepo{}
	uc := usecase.NewStoreEventUseCase(repo)

	tests := []usecase.StoreEventInput{
		{EventName: "product_view", Channel: "web", UserID: "", Timestamp: time.Now().Unix()},
		{EventName: "product_view", Channel: "", UserID: "user_123", Timestamp: time.Now().Unix()},
	}

	for _, in := range tests {
		created, err := uc.Execute(context.Background(), in)

		if err == nil {
			t.Fatalf("expected error for invalid input, got nil")
		}
		if created {
			t.Fatalf("expected created=false")
		}
		if !errors.Is(err, usecase.ErrInvalidEvent) {
			t.Fatalf("expected ErrInvalidEvent, got %v", err)
		}
	}
}

// ------------------------------------------------------------
// FUTURE TIMESTAMP
// ------------------------------------------------------------
func TestStoreEvent_FutureTimestamp(t *testing.T) {
	repo := &fakeEventRepo{}
	uc := usecase.NewStoreEventUseCase(repo)

	input := usecase.StoreEventInput{
		EventName: "product_view",
		Channel:   "web",
		UserID:    "user_123",
		Timestamp: time.Now().Add(5 * time.Minute).Unix(), // future
	}

	created, err := uc.Execute(context.Background(), input)

	if err == nil {
		t.Fatalf("expected error for future timestamp, got nil")
	}
	if created {
		t.Fatalf("expected created=false")
	}
	if !errors.Is(err, usecase.ErrFutureTime) {
		t.Fatalf("expected ErrFutureTime, got %v", err)
	}
}

// ------------------------------------------------------------
// DUPLICATE
// ------------------------------------------------------------
func TestStoreEvent_Duplicate(t *testing.T) {
	repo := &fakeEventRepo{
		InsertFn: func(ctx context.Context, e *domain.Event) (bool, error) {
			return false, nil // duplicate
		},
	}

	uc := usecase.NewStoreEventUseCase(repo)

	input := usecase.StoreEventInput{
		EventName: "product_view",
		Channel:   "web",
		UserID:    "user_123",
		Timestamp: time.Now().Unix(),
	}

	created, err := uc.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Fatalf("expected created=false for duplicate")
	}
}

// ------------------------------------------------------------
// REPOSITORY ERROR
// ------------------------------------------------------------
func TestStoreEvent_RepositoryError(t *testing.T) {
	repo := &fakeEventRepo{
		InsertFn: func(ctx context.Context, e *domain.Event) (bool, error) {
			return false, errors.New("db failure")
		},
	}

	uc := usecase.NewStoreEventUseCase(repo)

	input := usecase.StoreEventInput{
		EventName: "product_view",
		Channel:   "web",
		UserID:    "user_123",
		Timestamp: time.Now().Unix(),
	}

	created, err := uc.Execute(context.Background(), input)

	if err == nil {
		t.Fatalf("expected db error, got nil")
	}
	if created {
		t.Fatalf("expected created=false")
	}
	if err.Error() != "db failure" {
		t.Fatalf("expected 'db failure', got %v", err)
	}
}
