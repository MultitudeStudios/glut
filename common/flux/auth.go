package flux

import (
	"errors"
	"net/http"
	"strings"
)

// ErrInvalidAuthToken...
var ErrInvalidAuthToken = errors.New("invalid auth token")

// Session...
type Session struct {
	ID          string
	User        string
	Permissions []string
}

// Authenticator...
type Authenticator func(flow *Flow, token string) (*Session, error)

// AuthTokenExtractor...
type AuthTokenExtractor func(*http.Request) string

// DefaultAuthTokenExtractor...
func DefaultAuthTokenExtractor() AuthTokenExtractor {
	return func(r *http.Request) string {
		parts := strings.SplitN(r.Header.Get(HeaderAuthorization), " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return ""
		}
		return parts[1]
	}
}
