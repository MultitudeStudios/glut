package flux

import (
	"fmt"
	"log/slog"
	"net/http"
)

// Error represents an HTTP error that occurred while handling a request.
type Error struct {
	Code     string
	Status   int
	Message  string
	Errors   any
	Internal error
}

// Errors
var (
	InternalError     = NewError("internal", http.StatusInternalServerError, "Something went wrong.")
	UnauthorizedError = NewError("unauthorized", http.StatusUnauthorized, "Unauthorized")
	InvalidError      = func(format string, args ...any) *Error {
		return &Error{
			Code:    "invalid",
			Status:  http.StatusBadRequest,
			Message: fmt.Sprintf(format, args...),
		}
	}
	ValidationError = func(errs any) *Error {
		return &Error{
			Code:    "validation",
			Status:  http.StatusBadRequest,
			Message: "A validation error occurred.",
			Errors:  errs,
		}
	}
)

// NewError constructs a new Error with the given code, status, and message.
func NewError(code string, status int, message string) *Error {
	return &Error{
		Code:    code,
		Status:  status,
		Message: message,
	}
}

// Error satisfies the error interface.
func (e *Error) Error() string {
	if e.Internal == nil {
		return fmt.Sprintf("code=%s, status=%d, message=%s", e.Code, e.Status, e.Message)
	}
	return fmt.Sprintf("code=%s, status=%d, message=%s, internal=%v", e.Code, e.Status, e.Message, e.Internal)
}

// SetInternal sets the internal error of Error.
func (e *Error) SetInternal(err error) *Error {
	e.Internal = err
	return e
}

// handleError handles errors that occur during an HTTP request.
func (s *Server) handleError(f *Flow, err error) {
	e, ok := err.(*Error)
	if !ok {
		e = InternalError
		f.Logger().Error("An unexpected error occurred.", slog.String("error", err.Error()))
	}

	res := map[string]any{
		"code":    e.Code,
		"status":  e.Status,
		"message": e.Message,
	}
	if e.Errors != nil {
		res["errors"] = e.Errors
	}

	if s.debug {
		if e.Internal != nil {
			res["internal"] = e.Internal.Error()
		} else if !ok {
			res["internal"] = err.Error()
		}
	}
	if err := f.respond(e.Status, res); err != nil {
		f.Logger().Error("Error writing response.", slog.String("error", err.Error()))
	}
}
