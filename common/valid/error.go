package valid

import (
	"fmt"
	"strings"
)

// Errors represents a collection of validation errors.
type Errors []Error

// Error represents a validation error.
type Error struct {
	Field string `json:"field,omitempty"`
	Error string `json:"error"`
}

// Error satisfies the error interface.
func (e Errors) Error() string {
	errs := make([]string, len(e))
	for i, err := range e {
		if err.Field != "" {
			errs[i] = fmt.Sprintf("%s=%s", err.Field, err.Error)
		} else {
			errs[i] = err.Error
		}
	}
	return strings.Join(errs, ",")
}
