package domain

import "time"

type Event struct {
	EventName  string
	Channel    string
	CampaignID string
	UserID     string
	EventTime  time.Time
	Tags       []string
	Metadata   map[string]any
	DedupeKey  string
}
