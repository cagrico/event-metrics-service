package ports

import (
	"context"
	"event-metrics-service/internal/events/core/domain"
)

type EventRepositoryPort interface {
	// InsertEvent:
	//   created = true,  err = nil  -> new record
	//   created = false, err = nil  -> duplicate (idempotent)
	//   created = false, err != nil -> DB error
	InsertEvent(ctx context.Context, e *domain.Event) (created bool, err error)
}
