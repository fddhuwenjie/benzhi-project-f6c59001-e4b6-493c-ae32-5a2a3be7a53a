package storage

import (
	"encoding/json"
	"time"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"
)

type AuditPackage struct {
	SchemaVersion string                         `json:"schema_version"`
	ExportedAt    time.Time                      `json:"exported_at"`
	Case          *conservation.ConservationCase `json:"case"`
	Events        []conservation.Event           `json:"events"`
	Digest        string                         `json:"digest"`
}

func (r *FileRepository) AuditData(id string) (*conservation.ConservationCase, []conservation.Event, error) {
	item, err := r.Load(id)
	if err != nil {
		return nil, nil, err
	}
	events, err := r.Events(id)
	if err != nil {
		return nil, nil, err
	}
	return item, events, nil
}

func canonicalJSON(value any) ([]byte, error) { return json.Marshal(value) }
