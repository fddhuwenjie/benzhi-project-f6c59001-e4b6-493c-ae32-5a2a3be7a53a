package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a/internal/conservation"
)

type FileRepository struct {
	root        string
	casesDir    string
	eventsDir   string
	requestsDir string
	mu          sync.RWMutex
}

func NewFileRepository(root string) (*FileRepository, error) {
	if root == "" {
		return nil, errors.New("数据目录不能为空")
	}
	r := &FileRepository{
		root:        root,
		casesDir:    filepath.Join(root, "cases"),
		eventsDir:   filepath.Join(root, "events"),
		requestsDir: filepath.Join(root, "requests"),
	}
	for _, dir := range []string{r.root, r.casesDir, r.eventsDir, r.requestsDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("创建数据目录: %w", err)
		}
	}
	if err := r.recoverSnapshots(); err != nil {
		return nil, fmt.Errorf("恢复事项快照: %w", err)
	}
	return r, nil
}

func (r *FileRepository) Create(item *conservation.ConservationCase, event conservation.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	path := r.casePath(item.ID)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%w：事项 %s 已存在", conservation.ErrConflict, item.ID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return r.commit(item, event)
}

func (r *FileRepository) Save(item *conservation.ConservationCase, previousRevision int64, event conservation.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, err := r.loadUnlocked(item.ID)
	if err != nil {
		return err
	}
	if current.Revision != previousRevision {
		return fmt.Errorf("%w：持久化版本为 %d，请求基于 %d", conservation.ErrConflict, current.Revision, previousRevision)
	}
	if item.Revision <= current.Revision {
		return fmt.Errorf("%w：新版本必须大于当前版本", conservation.ErrConflict)
	}
	return r.commit(item, event)
}

func (r *FileRepository) commit(item *conservation.ConservationCase, event conservation.Event) error {
	data, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return err
	}
	if err := r.appendEvent(event); err != nil {
		return err
	}
	if err := atomicWrite(r.casePath(item.ID), append(data, '\n'), 0o640); err != nil {
		return fmt.Errorf("写入事项快照: %w", err)
	}
	return nil
}

func (r *FileRepository) appendEvent(event conservation.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(r.eventPath(event.CaseID), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

func (r *FileRepository) Load(id string) (*conservation.ConservationCase, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.loadUnlocked(id)
}

func (r *FileRepository) loadUnlocked(id string) (*conservation.ConservationCase, error) {
	data, err := os.ReadFile(r.casePath(id))
	if errors.Is(err, os.ErrNotExist) {
		return nil, conservation.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var item conservation.ConservationCase
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, &CorruptDataError{Path: r.casePath(id), Err: err}
	}
	if item.ID != id || !item.Status.Valid() || item.Revision < 1 {
		return nil, &CorruptDataError{Path: r.casePath(id), Err: errors.New("事项标识、状态或版本无效")}
	}
	if err := item.ValidateAggregate(); err != nil {
		return nil, &CorruptDataError{Path: r.casePath(id), Err: err}
	}
	return &item, nil
}

func (r *FileRepository) List(filter CaseFilter) ([]*conservation.ConservationCase, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entries, err := os.ReadDir(r.casesDir)
	if err != nil {
		return nil, err
	}
	items := make([]*conservation.ConservationCase, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(r.casesDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var item conservation.ConservationCase
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, &CorruptDataError{Path: entry.Name(), Err: err}
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if filter.Query != "" && !containsFold(item.Title+" "+item.ShelfMark+" "+item.ResponsibleConservator, filter.Query) {
			continue
		}
		copy := item
		items = append(items, &copy)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return items, nil
}

func (r *FileRepository) Events(id string) ([]conservation.Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, err := os.Open(r.eventPath(id))
	if errors.Is(err, os.ErrNotExist) {
		return nil, conservation.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	result := make([]conservation.Event, 0)
	decoder := json.NewDecoder(bufio.NewReader(f))
	for {
		var event conservation.Event
		if err := decoder.Decode(&event); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, &CorruptDataError{Path: r.eventPath(id), Err: err}
		}
		result = append(result, event)
	}
	return result, nil
}

func (r *FileRepository) casePath(id string) string {
	return filepath.Join(r.casesDir, safeName(id)+".json")
}
func (r *FileRepository) eventPath(id string) string {
	return filepath.Join(r.eventsDir, safeName(id)+".jsonl")
}
func (r *FileRepository) requestPath(id string) string {
	return filepath.Join(r.requestsDir, safeName(id)+".json")
}
