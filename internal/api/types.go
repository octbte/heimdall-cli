package api

import "time"

type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

type OverviewSummary struct {
	EventsCount         int     `json:"events_count"`
	ErrorsCount         int     `json:"errors_count"`
	PerformanceCount    int     `json:"performance_count"`
	AvgDurationMs       float64 `json:"avg_duration_ms"`
	CriticalErrorsCount int     `json:"critical_errors_count"`
}

type TopFailingEvent struct {
	EventName string `json:"event_name"`
	Count     int    `json:"count"`
}

type OverviewResponse struct {
	Summary          OverviewSummary   `json:"summary"`
	TopFailingEvents []TopFailingEvent `json:"top_failing_events"`
	LatestErrors     []ErrorItem       `json:"latest_errors"`
}

type EventItem struct {
	ID            string         `json:"id"`
	EventName     string         `json:"event_name"`
	EventCategory string         `json:"event_category"`
	Status        string         `json:"status"`
	Level         string         `json:"level"`
	DurationMs    *int           `json:"duration_ms"`
	OccurredAt    time.Time      `json:"occurred_at"`
	ReceivedAt    *time.Time     `json:"received_at"`
	ProjectID     string         `json:"project_id"`
	EnvironmentID *string        `json:"environment_id"`
	CorrelationID *string        `json:"correlation_id"`
	RequestID     *string        `json:"request_id"`
	TraceID       *string        `json:"trace_id"`
	SpanID        *string        `json:"span_id"`
	ParentSpanID  *string        `json:"parent_span_id"`
	Metadata      map[string]any `json:"metadata"`
	Tags          map[string]any `json:"tags"`
}

type EventsResponse struct {
	Items      []EventItem `json:"items"`
	Pagination Pagination  `json:"pagination"`
}

type ErrorItem struct {
	ID            string         `json:"id"`
	ErrorType     string         `json:"error_type"`
	Message       string         `json:"message"`
	Stacktrace    *string        `json:"stacktrace"`
	Fingerprint   *string        `json:"fingerprint"`
	Level         string         `json:"level"`
	Status        string         `json:"status"`
	OccurredAt    time.Time      `json:"occurred_at"`
	Endpoint      *string        `json:"endpoint"`
	JobName       *string        `json:"job_name"`
	ProjectID     string         `json:"project_id"`
	EnvironmentID *string        `json:"environment_id"`
	CorrelationID *string        `json:"correlation_id"`
	TraceID       *string        `json:"trace_id"`
	SpanID        *string        `json:"span_id"`
	ParentSpanID  *string        `json:"parent_span_id"`
	Metadata      map[string]any `json:"metadata"`
	Tags          map[string]any `json:"tags"`
}

type ErrorsResponse struct {
	Items      []ErrorItem `json:"items"`
	Pagination Pagination  `json:"pagination"`
}

type PerfItem struct {
	ID            string         `json:"id"`
	MetricName    string         `json:"metric_name"`
	TargetType    string         `json:"target_type"`
	TargetName    string         `json:"target_name"`
	DurationMs    int            `json:"duration_ms"`
	Level         string         `json:"level"`
	Status        string         `json:"status"`
	OccurredAt    time.Time      `json:"occurred_at"`
	ProjectID     string         `json:"project_id"`
	EnvironmentID *string        `json:"environment_id"`
	CorrelationID *string        `json:"correlation_id"`
	TraceID       *string        `json:"trace_id"`
	SpanID        *string        `json:"span_id"`
	ParentSpanID  *string        `json:"parent_span_id"`
	Metadata      map[string]any `json:"metadata"`
	Tags          map[string]any `json:"tags"`
}

type PerfResponse struct {
	Items      []PerfItem `json:"items"`
	Pagination Pagination `json:"pagination"`
}

type ListEventsParams struct {
	Level       string
	Status      string
	EventName   string
	Environment string
	From        string
	To          string
	Limit       int
	Offset      int
}

type ListErrorsParams struct {
	Level       string
	Fingerprint string
	Endpoint    string
	JobName     string
	Environment string
	From        string
	To          string
	Limit       int
	Offset      int
}

type ListPerfParams struct {
	MetricName  string
	TargetType  string
	TargetName  string
	Environment string
	From        string
	To          string
	Limit       int
	Offset      int
}

type TraceRecord struct {
	ID           string     `json:"id"`
	Type         string     `json:"type"` // "event" | "error" | "performance"
	SpanID       *string    `json:"span_id"`
	ParentSpanID *string    `json:"parent_span_id"`
	Name         string     `json:"name"`
	OccurredAt   time.Time  `json:"occurred_at"`
	DurationMs   *int       `json:"duration_ms"`
	Status       *string    `json:"status"`
	Level        *string    `json:"level"`
}

type TraceResponse struct {
	TraceID string        `json:"trace_id"`
	Records []TraceRecord `json:"records"`
}
