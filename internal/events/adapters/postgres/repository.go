package postgres

import (
	"context"
	"encoding/json"
	"event-metrics-service/internal/events/core/domain"
	"event-metrics-service/internal/events/core/ports"

	"github.com/lib/pq"
)

type EventRepository struct {
	db DB
}

func NewEventRepository(db DB) *EventRepository {
	return &EventRepository{db: db}
}

var _ ports.EventRepositoryPort = (*EventRepository)(nil)

// SQL template
const insertEventSQL = `
INSERT INTO events (
    event_name,
    channel,
    campaign_id,
    user_id,
    event_time,
    tags,
    metadata,
    dedupe_key
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8
)
ON CONFLICT (dedupe_key) DO NOTHING;
`

func (r *EventRepository) InsertEvent(ctx context.Context, e *domain.Event) (bool, error) {

	var campaignID any
	if e.CampaignID == "" {
		campaignID = nil
	} else {
		campaignID = e.CampaignID
	}

	metadataJSON, err := json.Marshal(e.Metadata)
	if err != nil {
		return false, err
	}

	res, err := r.db.ExecContext(ctx, insertEventSQL,
		e.EventName,
		e.Channel,
		campaignID,
		e.UserID,
		e.EventTime,
		pqStringArray(e.Tags),
		metadataJSON,
		e.DedupeKey,
	)
	if err != nil {
		return false, err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	// rows == 1  -> new record
	// rows == 0  -> duplicate (ON CONFLICT DO NOTHING)
	return rows > 0, nil
}

func pqStringArray(tags []string) any {
	return pq.Array(tags)
}
