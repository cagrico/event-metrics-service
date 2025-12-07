package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"event-metrics-service/internal/events/core/domain"
	"event-metrics-service/internal/events/core/ports"
)

var (
	ErrInvalidEvent = errors.New("invalid event")
	ErrFutureTime   = errors.New("timestamp cannot be in the future")
)

type StoreEventUseCase struct {
	repo ports.EventRepositoryPort
}

func NewStoreEventUseCase(repo ports.EventRepositoryPort) *StoreEventUseCase {
	return &StoreEventUseCase{repo: repo}
}

type StoreEventInput struct {
	EventName  string
	Channel    string
	CampaignID string
	UserID     string
	Timestamp  int64
	Tags       []string
	Metadata   map[string]any
}

func (uc *StoreEventUseCase) Execute(ctx context.Context, in StoreEventInput) (bool, error) {

	if err := uc.validateInput(in); err != nil {
		return false, err
	}

	eventTime := time.Unix(in.Timestamp, 0).UTC()

	if in.Tags == nil {
		in.Tags = []string{}
	}
	if in.Metadata == nil {
		in.Metadata = map[string]any{}
	}

	dedupeKey := buildDedupeKey(in, eventTime)

	e := &domain.Event{
		EventName:  in.EventName,
		Channel:    in.Channel,
		CampaignID: in.CampaignID,
		UserID:     in.UserID,
		EventTime:  eventTime,
		Tags:       in.Tags,
		Metadata:   in.Metadata,
		DedupeKey:  dedupeKey,
	}

	created, err := uc.repo.InsertEvent(ctx, e)
	if err != nil {
		return false, err
	}

	return created, nil
}

func buildDedupeKey(in StoreEventInput, t time.Time) string {
	// event_name + user_id + channel + campaign_id + unix_timestamp
	return fmt.Sprintf("%s|%s|%s|%s|%d",
		in.EventName,
		in.UserID,
		in.Channel,
		in.CampaignID,
		t.Unix(),
	)
}

type BulkCreateEventsInput struct {
	Events []StoreEventInput
}

type BulkCreateEventsResult struct {
	Created    int
	Duplicates int
}

func (uc *StoreEventUseCase) BulkCreateEvents(ctx context.Context, in BulkCreateEventsInput) (BulkCreateEventsResult, error) {
	var res BulkCreateEventsResult

	for _, ev := range in.Events {
		if err := uc.validateInput(ev); err != nil {
			return res, err
		}
	}

	for _, ev := range in.Events {
		ok, err := uc.Execute(ctx, ev)
		if err != nil {
			return res, err
		}

		if ok {
			res.Created++
		} else {
			res.Duplicates++
		}
	}

	return res, nil
}

func (uc *StoreEventUseCase) validateInput(in StoreEventInput) error {

	if in.EventName == "" || in.Channel == "" || in.UserID == "" {
		return ErrInvalidEvent
	}

	now := time.Now().Unix()
	if in.Timestamp > now {
		return ErrFutureTime
	}

	return nil
}
