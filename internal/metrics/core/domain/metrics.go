package domain

type AggregatedMetrics struct {
	EventName   string
	From        int64 // unix second
	To          int64 // unix second
	TotalCount  int64
	UniqueUsers int64

	GroupBy string         // "", "channel", "time"
	Groups  []MetricsGroup // grup bazlı breakdown
}

type MetricsGroup struct {
	Key         string // örn: "web" veya "2025-12-07T10:00:00Z"
	TotalCount  int64
	UniqueUsers int64
}
