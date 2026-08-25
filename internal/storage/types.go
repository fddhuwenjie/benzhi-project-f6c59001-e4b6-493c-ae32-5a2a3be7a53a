package storage

import (
	"encoding/json"
	"time"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"
)

type CaseFilter struct {
	Status conservation.Status
	Query  string
}

type IdempotencyRecord struct {
	RequestID   string          `json:"request_id"`
	Operation   string          `json:"operation"`
	CaseID      string          `json:"case_id"`
	Result      json.RawMessage `json:"result"`
	CompletedAt time.Time       `json:"completed_at"`
}

type Repository interface {
	Create(item *conservation.ConservationCase, event conservation.Event) error
	Save(item *conservation.ConservationCase, previousRevision int64, event conservation.Event) error
	Load(id string) (*conservation.ConservationCase, error)
	List(filter CaseFilter) ([]*conservation.ConservationCase, error)
	Events(id string) ([]conservation.Event, error)
	GetIdempotency(requestID string) (*IdempotencyRecord, error)
	PutIdempotency(record IdempotencyRecord) error
}
