package conservation

import (
	"encoding/json"
	"time"
)

type Event struct {
	ID             string          `json:"id"`
	CaseID         string          `json:"case_id"`
	RequestID      string          `json:"request_id"`
	Type           string          `json:"type"`
	Actor          string          `json:"actor"`
	Role           string          `json:"role"`
	FromStatus     Status          `json:"from_status,omitempty"`
	ToStatus       Status          `json:"to_status"`
	BeforeRevision int64           `json:"before_revision"`
	AfterRevision  int64           `json:"after_revision"`
	OccurredAt     time.Time       `json:"occurred_at"`
	Details        json.RawMessage `json:"details,omitempty"`
}

type TimelineEntry struct {
	Type       string    `json:"type"`
	Label      string    `json:"label"`
	Actor      string    `json:"actor"`
	Role       string    `json:"role"`
	FromStatus Status    `json:"from_status,omitempty"`
	ToStatus   Status    `json:"to_status"`
	OccurredAt time.Time `json:"occurred_at"`
}
