package flux

import (
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// Flux...
type Flux[I, O any] struct {
	server  *Server
	options *Options
	handler HandlerFunc[I, O]
}

// Options represent optional parameters of a Flux used to configure its behavior.
type Options struct {
	// Require authentication for this handler. If this option is set and
	// authentication fails due to a missing or invalid auth token, the
	// server will respond with a 401 Unauthorized error. If this option
	// is set, then a value must be provided for ServerOptions.Authenticator.
	Authenticate bool
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
type HandlerFunc[I, O any] func(*Flow, I) (O, error)

// Empty...
type Empty *struct{}

// New...
func New[I, O any](s *Server, name string, handler HandlerFunc[I, O], options *Options) {
	s.router.table[name] = &Flux[I, O]{
		server:  s,
		options: options,
		handler: handler,
	}
}

// ServeHTTP satisfies the http.Handler interface.
func (f *Flux[I, O]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flow := f.server.pool.Get().(*Flow)
	defer f.server.pool.Put(flow)
	flow.init(w, r)

	if f.server.debug {
		defer func() {
			flow.Logger().With(
				slog.Int64("elapsed_ms", time.Since(flow.Start()).Milliseconds()),
				slog.Int("http_status", flow.w.Status),
			).Debug("Handled HTTP request.")
		}()
	}

	// Set common headers.
	w.Header().Set(HeaderXRequestID, flow.ID())

	// Set security headers.
	if f.server.TLS {
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
		sess, err := f.server.authenticator(flow, token)
		if err != nil {
			switch {
			case errors.Is(err, ErrInvalidAuthToken):
				f.server.handleError(flow, UnauthorizedError)
			default:
				f.server.handleError(flow, err)
			}
			return
		}
		// TODO: rate limit requests using user ID
		flow.sess = sess
	}

	if f.options.Authenticate && flow.Session() == nil {
		f.server.handleError(flow, UnauthorizedError)
		return
	}

	var in I
	if _, ok := any(in).(Empty); !ok {
		// Input type is non-empty; bind the request to the input type.
		if err := flow.bind(&in); err != nil {
			f.server.handleError(flow, err)
			return
		}
	}

	out, err := f.handler(flow, in)
	if err != nil {
		f.server.handleError(flow, err)
		return
	}

	// If output is empty, return empty response.
	if _, ok := any(out).(Empty); ok {
		err = flow.respond(f.options.SuccessStatus, nil)
	} else {
		err = flow.respond(f.options.SuccessStatus, out)
	}
	if err != nil {
		flow.Logger().Error("Error writing HTTP response", slog.String("error", err.Error()))
	}
}
