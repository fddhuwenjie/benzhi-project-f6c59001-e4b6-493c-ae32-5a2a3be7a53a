package conservation

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound = errors.New("修复事项不存在")
	ErrConflict = errors.New("修订版本冲突")
)

type FieldIssue struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationError struct {
	Issues []FieldIssue `json:"issues"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("输入校验失败，共 %d 项", len(e.Issues))
}

func (e *ValidationError) Add(field, message string) {
	e.Issues = append(e.Issues, FieldIssue{Field: field, Message: message})
}

func (e *ValidationError) OrNil() error {
	if len(e.Issues) == 0 {
		return nil
	}
	return e
}

type TransitionError struct {
	From    Status `json:"from"`
	Message string `json:"message"`
}

func (e *TransitionError) Error() string { return e.Message }

func CheckRevision(actual, expected int64) error {
	if expected <= 0 || actual != expected {
		return fmt.Errorf("%w：当前为 %d，请求基于 %d", ErrConflict, actual, expected)
	}
	return nil
}
