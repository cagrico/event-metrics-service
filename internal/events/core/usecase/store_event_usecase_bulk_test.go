package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"event-metrics-service/internal/events/core/domain"
)

// Fake repo
type fakeBulkRepo struct {
	InsertCalls []*domain.Event
	Results     []bool
	Err         error
}

func (f *fakeBulkRepo) InsertEvent(ctx context.Context, e *domain.Event) (bool, error) {
	if f.Err != nil {
		return false, f.Err
	}
	f.InsertCalls = append(f.InsertCalls, e)

	if len(f.Results) == 0 {
		// default: created
		return true, nil
	}

	res := f.Results[0]
	f.Results = f.Results[1:]
	return res, nil
}

func TestBulkCreateEvents_AllCreated(t *testing.T) {
	ctx := context.Background()

	repo := &fakeBulkRepo{
		Results: []bool{true, true, true},
	}

	uc := NewStoreEventUseCase(repo)

	now := time.Now().Add(-time.Minute).Unix()

	input := BulkCreateEventsInput{
		Events: []StoreEventInput{
			{
				EventName:  "product_view",
				Channel:    "web",
				CampaignID: "cmp_1",
				UserID:     "user_1",
				Timestamp:  now,
				Tags:       []string{"electronics"},
				Metadata:   map[string]any{"product_id": "p1"},
			},
			{
				EventName:  "product_view",
				Channel:    "mobile",
				CampaignID: "cmp_1",
				UserID:     "user_2",
				Timestamp:  now,
				Tags:       []string{"electronics"},
				Metadata:   map[string]any{"product_id": "p2"},
			},
			{
				EventName:  "add_to_cart",
				Channel:    "web",
				CampaignID: "cmp_2",
				UserID:     "user_3",
				Timestamp:  now,
				Tags:       []string{"cart"},
				Metadata:   map[string]any{"product_id": "p3"},
			},
		},
	}

	res, err := uc.BulkCreateEvents(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Created != 3 {
		t.Errorf("expected Created=3, got %d", res.Created)
	}
	if res.Duplicates != 0 {
		t.Errorf("expected Duplicates=0, got %d", res.Duplicates)
	}

	if len(repo.InsertCalls) != 3 {
		t.Errorf("expected 3 InsertEvent calls, got %d", len(repo.InsertCalls))
	}
}

func TestBulkCreateEvents_MixedCreatedAndDuplicate(t *testing.T) {
	ctx := context.Background()

	// created, duplicate, created
	repo := &fakeBulkRepo{
		Results: []bool{true, false, true},
	}

	uc := NewStoreEventUseCase(repo)

	now := time.Now().Add(-time.Minute).Unix()

	input := BulkCreateEventsInput{
		Events: []StoreEventInput{
			{
				EventName: "product_view",
				Channel:   "web",
				UserID:    "user_1",
				Timestamp: now,
			},
			{
				EventName: "product_view",
				Channel:   "web",
				UserID:    "user_1",
				Timestamp: now,
			},
			{
				EventName: "add_to_cart",
				Channel:   "web",
				UserID:    "user_2",
				Timestamp: now,
			},
		},
	}

	res, err := uc.BulkCreateEvents(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Created != 2 {
		t.Errorf("expected Created=2, got %d", res.Created)
	}
	if res.Duplicates != 1 {
		t.Errorf("expected Duplicates=1, got %d", res.Duplicates)
	}

	if len(repo.InsertCalls) != 3 {
		t.Errorf("expected 3 InsertEvent calls, got %d", len(repo.InsertCalls))
	}
}

func TestBulkCreateEvents_ValidationErrorInOneEvent(t *testing.T) {
	ctx := context.Background()

	repo := &fakeBulkRepo{}
	uc := NewStoreEventUseCase(repo)

	now := time.Now().Add(-time.Minute).Unix()

	input := BulkCreateEventsInput{
		Events: []StoreEventInput{
			{
				EventName: "product_view",
				Channel:   "web",
				UserID:    "user_1",
				Timestamp: now,
			},
			{
				// Error : empty EventName
				EventName: "",
				Channel:   "web",
				UserID:    "user_2",
				Timestamp: now,
			},
			{
				EventName: "add_to_cart",
				Channel:   "web",
				UserID:    "user_3",
				Timestamp: now,
			},
		},
	}

	_, err := uc.BulkCreateEvents(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrInvalidEvent) {
		t.Errorf("expected ErrInvalidEvent, got %v", err)
	}

	if len(repo.InsertCalls) != 0 {
		t.Errorf("expected 0 InsertEvent calls, got %d", len(repo.InsertCalls))
	}
}
