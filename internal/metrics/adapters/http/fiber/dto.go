package fiber

type MetricsGroupResponse struct {
	Key         string `json:"key"`
	TotalCount  int64  `json:"total_count"`
	UniqueUsers int64  `json:"unique_users"`
}

type MetricsResponse struct {
	EventName   string                 `json:"event_name"`
	From        int64                  `json:"from"`
	To          int64                  `json:"to"`
	TotalCount  int64                  `json:"total_count"`
	UniqueUsers int64                  `json:"unique_users"`
	GroupBy     string                 `json:"group_by,omitempty"`
	Groups      []MetricsGroupResponse `json:"groups,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error" example:"invalid_event"`
	Message string `json:"message" example:"Event payload is invalid"`
}
