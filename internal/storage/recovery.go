package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"
)

type eventDetails struct {
	Snapshot *conservation.ConservationCase `json:"snapshot"`
}

func (r *FileRepository) recoverSnapshots() error {
	entries, err := os.ReadDir(r.eventsDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		last, err := readLastEvent(filepath.Join(r.eventsDir, entry.Name()))
		if err != nil {
			return err
		}
		var details eventDetails
		if err := json.Unmarshal(last.Details, &details); err != nil || details.Snapshot == nil {
			return &CorruptDataError{Path: entry.Name(), Err: errors.New("事件缺少可恢复快照")}
		}
		current, loadErr := r.loadUnlocked(details.Snapshot.ID)
		if loadErr == nil && current.Revision >= details.Snapshot.Revision {
			continue
		}
		if loadErr != nil && !errors.Is(loadErr, conservation.ErrNotFound) {
			return loadErr
		}
		data, err := json.MarshalIndent(details.Snapshot, "", "  ")
		if err != nil {
			return err
		}
		if err := atomicWrite(r.casePath(details.Snapshot.ID), append(data, 10), 0o640); err != nil {
			return err
		}
	}
	return nil
}

func readLastEvent(path string) (conservation.Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return conservation.Event{}, err
	}
	defer f.Close()
	decoder := json.NewDecoder(bufio.NewReader(f))
	var last conservation.Event
	count := 0
	for {
		var event conservation.Event
		err := decoder.Decode(&event)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return conservation.Event{}, &CorruptDataError{Path: path, Err: err}
		}
		last = event
		count++
	}
	if count == 0 {
		return conservation.Event{}, &CorruptDataError{Path: path, Err: errors.New("事件日志为空")}
	}
	return last, nil
}
