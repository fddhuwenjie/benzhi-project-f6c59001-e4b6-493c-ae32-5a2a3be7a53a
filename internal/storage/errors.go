package storage

import (
	"errors"
	"fmt"
)

var ErrIdempotencyConflict = errors.New("request_id 已用于不同操作")

type CorruptDataError struct {
	Path string
	Err  error
}

func (e *CorruptDataError) Error() string {
	return fmt.Sprintf("持久化数据损坏 %s: %v", e.Path, e.Err)
}
func (e *CorruptDataError) Unwrap() error { return e.Err }
