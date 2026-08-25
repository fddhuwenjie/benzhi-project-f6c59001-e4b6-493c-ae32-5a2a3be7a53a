package storage

import (
	"encoding/json"
	"errors"
	"os"
)

func (r *FileRepository) GetIdempotency(requestID string) (*IdempotencyRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	data, err := os.ReadFile(r.requestPath(requestID))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var record IdempotencyRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, &CorruptDataError{Path: r.requestPath(requestID), Err: err}
	}
	if record.RequestID != requestID {
		return nil, &CorruptDataError{Path: r.requestPath(requestID), Err: errors.New("幂等记录标识不匹配")}
	}
	return &record, nil
}

func (r *FileRepository) PutIdempotency(record IdempotencyRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	path := r.requestPath(record.RequestID)
	if data, err := os.ReadFile(path); err == nil {
		var existing IdempotencyRecord
		if err := json.Unmarshal(data, &existing); err != nil {
			return err
		}
		if existing.Operation != record.Operation || existing.CaseID != record.CaseID {
			return ErrIdempotencyConflict
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(path, append(data, '\n'), 0o640)
}
