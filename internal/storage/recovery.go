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
		path := filepath.Join(r.eventsDir, entry.Name())
		last, err := readLastEvent(path)
		if err != nil {
			return err
		}
		if last == nil {
			// 日志仅含未完成记录或为空，没有可恢复的快照。
			continue
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

// readLastEvent 返回事件日志中最后一条完整记录。
//
// 当日志末尾存在未完成的 JSON 残片（例如底层文件系统短写导致的
// io.ErrUnexpectedEOF）时，已成功落盘的完整记录仍然可用：此时返回
// 最后一条完整事件并将末尾残片从文件中截除，以便后续追加写入保持
// 日志结构一致。
//
// 日志中间位置的非法 JSON 仍然会被视为损坏并报错，因为这种情况
// 通常意味着真正的事务/写入乱序，无法安全地挑选“最后一条”记录。
// 仅当确无任何完整记录（首条即未完成，或日志为空）时返回 nil。
func readLastEvent(path string) (*conservation.Event, error) {
	events, truncatedAt, err := scanEvents(path)
	if err != nil {
		return nil, err
	}
	if truncatedAt > 0 {
		if err := truncateTo(path, truncatedAt); err != nil {
			return nil, err
		}
	}
	if len(events) == 0 {
		return nil, nil
	}
	last := events[len(events)-1]
	return &last, nil
}

// scanEvents 顺序解析事件日志，返回所有完整记录以及末尾未完成残片
// 的起始字节偏移（若存在）。仅末尾的 io.ErrUnexpectedEOF 被视为可容忍
// 的短写残片；其他解析错误（如 *json.SyntaxError，对应日志中间位置的
// 非法 JSON）一律按损坏处理。
func scanEvents(path string) (events []conservation.Event, truncatedAt int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	dec := json.NewDecoder(bufio.NewReader(f))
	for {
		var event conservation.Event
		decodeErr := dec.Decode(&event)
		if errors.Is(decodeErr, io.EOF) {
			return events, 0, nil
		}
		if decodeErr != nil {
			// io.ErrUnexpectedEOF 表示在解析一条记录途中读到了文件末尾，
			// 即末尾存在未完成的 JSON 残片。这种情形可以安全忽略：
			// 此时 decoder.InputOffset() 指向上一条已成功解码记录
			// （含其分隔换行符）之后的字节位置，也就是应保留的日志长度。
			if errors.Is(decodeErr, io.ErrUnexpectedEOF) {
				return events, dec.InputOffset(), nil
			}
			return nil, 0, &CorruptDataError{Path: path, Err: decodeErr}
		}
		events = append(events, event)
	}
}

// truncateTo 将文件截断至 offset 字节处，用于移除末尾未完成的 JSON 残片。
func truncateTo(path string, offset int64) error {
	if offset < 0 {
		return errors.New("无效的截断偏移")
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Truncate(offset); err != nil {
		return err
	}
	return f.Sync()
}
