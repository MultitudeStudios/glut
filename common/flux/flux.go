package flux

import (
	"log/slog"
	"net/http"
	"time"
)

// Flux...
type Flux struct {
	server  *Server
	options *Options
	handler HandlerFunc
}

// Options represent optional parameters of a Flux used to configure its behavior.
type Options struct {
	// Require authentication for this handler. If this option is set and
	// authentication fails due to a missing or invalid auth token, the
	// server will respond with a 401 Unauthorized error. If this option
	// is set, then a value must be provided for ServerOptions.Authenticator.
	RequireAuth bool
	// Set the required permissions for this handler. This option requires
	// the Authenticate option to also be set. If this option is set and
	// the authenticated user does not have all required permissions, then
	// the server will respond with a 403 Forbidden error.
	Permissions []string
	// Set a maximum request size for this handler. This option overrides
	// any value set for ServerOptions.MaxRequestSize.
	MaxRequestSize int64
	// SuccessStatus is the HTTP status code returned when execution is successful.
	// If not set, status 200 OK is returned by default.
	SuccessStatus int
}

// HandlerFunc...
type HandlerFunc func(*Flow) error

// Empty...
type Empty *struct{}

// ServeHTTP satisfies the http.Handler interface.
func (f *Flux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flow := f.server.pool.Get().(*Flow)
	defer f.server.pool.Put(flow)
	flow.init(w, r)

	if f.server.debug {
		defer func() {
			flow.Logger.With(
				slog.Int64("elapsed_ms", time.Since(flow.Time).Milliseconds()),
				slog.Int("http_status", flow.w.Status),
			).Debug("Handled HTTP request.")
		}()
	}

	// Set common headers.
	w.Header().Set(HeaderXRequestID, flow.ID)

	// Set security headers.
	if f.server.tls {
		w.Header().Add(HeaderStrictTransportSecurity, "max-age=63072000; includeSubDomains")
	}
	w.Header().Add(HeaderContentSecurityPolicy, "default-src 'none'; frame-ancestors 'none'")
	w.Header().Add(HeaderXContentTypeOptions, "nosniff")
	w.Header().Add(HeaderReferrerPolicy, "same-origin")
	w.Header().Add(HeaderXFrameOptions, "DENY")

	// TODO: rate limit requests using request IP

	// Limit the size of incoming request bodies.
	if f.options.MaxRequestSize != 0 {
		r.Body = http.MaxBytesReader(w, r.Body, f.options.MaxRequestSize)
	} else if f.server.maxRequestSize != 0 {
		r.Body = http.MaxBytesReader(w, r.Body, f.server.maxRequestSize)
	}

	// Attempt to authenticate request.
	token := f.server.authTokenExtractor(r)
	if token != "" {
		session, err := f.server.authenticator(flow, token)
		if err != nil {
			f.server.handleError(flow, err)
			return
		}
		// TODO: rate limit requests using user ID
		flow.Session = session
	}

	if f.options.RequireAuth && flow.Session == nil {
		f.server.handleError(flow, UnauthorizedError)
		return
	}

	if err := f.handler(flow); err != nil {
		f.server.handleError(flow, err)
	}
}
