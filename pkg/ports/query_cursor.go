package ports

import "fmt"

// QueryCursorError distinguishes a changed canonical generation from an invalid
// request. HTTP adapters can preserve this reason without inspecting DB errors.
type QueryCursorError struct {
	Stale  bool
	Reason string
}

func (e *QueryCursorError) Error() string { return fmt.Sprintf("query cursor: %s", e.Reason) }

func (e *QueryCursorError) Extensions() map[string]any {
	code := "INVALID_CURSOR"
	if e.Stale {
		code = "CURSOR_STALE"
	}
	return map[string]any{"code": code, "restartPagination": e.Stale}
}
