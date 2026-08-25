package workflow

import (
	"fmt"
	"strings"
)

type Role string

const (
	RoleConservator Role = "conservator"
	RoleReviewer    Role = "reviewer"
	RoleManager     Role = "manager"
)

type Actor struct {
	Name string `json:"name"`
	Role Role   `json:"role"`
}

func (a Actor) Validate(allowed ...Role) error {
	if strings.TrimSpace(a.Name) == "" {
		return &AuthorizationError{Message: "操作者姓名不能为空"}
	}
	for _, role := range allowed {
		if a.Role == role {
			return nil
		}
	}
	return &AuthorizationError{Message: fmt.Sprintf("角色 %q 无权执行此操作", a.Role)}
}

type AuthorizationError struct {
	Message string `json:"message"`
}

func (e *AuthorizationError) Error() string { return e.Message }
