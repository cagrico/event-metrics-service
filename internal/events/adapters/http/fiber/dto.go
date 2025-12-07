package fiber

// CreateEventRequest represents event creation payload
// @Description Event creation DTO
type CreateEventRequest struct {
	EventName  string         `json:"event_name"`
	Channel    string         `json:"channel"`
	CampaignID string         `json:"campaign_id"`
	UserID     string         `json:"user_id"`
	Timestamp  int64          `json:"timestamp"`
	Tags       []string       `json:"tags"`
	Metadata   map[string]any `json:"metadata"`
}

type CreateEventResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type BulkCreateEventsRequest struct {
	Events []bulkEventItem `json:"events"`
}

type bulkEventItem struct {
	EventName  string         `json:"event_name"`
	Channel    string         `json:"channel"`
	CampaignID string         `json:"campaign_id"`
	UserID     string         `json:"user_id"`
	Timestamp  int64          `json:"timestamp"`
	Tags       []string       `json:"tags"`
	Metadata   map[string]any `json:"metadata"`
}

type BulkCreateEventsResponse struct {
	Created    int `json:"created"`
	Duplicates int `json:"duplicates"`
}

type ErrorResponse struct {
	Error   string `json:"error" example:"invalid_event"`
	Message string `json:"message" example:"Event payload is invalid"`
}
