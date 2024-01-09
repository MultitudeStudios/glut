package flux

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

// Server...
type Server struct {
	router             *router
	server             *http.Server
	pool               sync.Pool
	port               int
	debug              bool
	TLS                bool
	logger             *slog.Logger
	ipExtractor        IPExtractor
	authenticator      Authenticator
	authTokenExtractor AuthTokenExtractor
	shutdownTimeout    time.Duration
	maxRequestSize     int64
}

// ServerOptions...
type ServerOptions struct {
	Debug              bool
	Port               int
	TLS                bool
	Logger             *slog.Logger
	Authenticator      Authenticator
	AuthTokenExtractor AuthTokenExtractor
	IPExtractor        IPExtractor

	// MaxRequestSize is the maximum accepted request size in bytes.
	// This is used to prevent a denial of service attack where no Content-Length
	// is provided and the server is fed data until it exhausts memory.
	// Setting this option will enable a default maximum request size for all handlers.
	// This can be overridden in an individual handler by setting Options.MaxRequestSize.
	MaxRequestSize    int64
	MaxHeaderBytes    int
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

// ConfigureServer...
func (s *Server) ConfigureServer(options *ServerOptions) {
	// Set server defaults.
	s.port = 8000
	s.logger = s.DefaultLogger()
	s.authTokenExtractor = DefaultAuthTokenExtractor()
	s.ipExtractor = ExtractIPDirect()
	s.maxRequestSize = 1024 * 1024 // 1MB
	s.shutdownTimeout = 5 * time.Second

	// Configure flow pool.
	s.pool.New = func() interface{} {
		return &Flow{
			s: s,
		}
	}

	// Configure router.
	s.router = &router{
		server: s,
		table:  make(map[string]http.Handler),
	}

	// Configure HTTP server.
	if options.Port == 0 {
		options.Port = 8000
	}
	if options.MaxHeaderBytes == 0 {
		options.MaxHeaderBytes = 1024 * 1024 // 1MB
	}
	if options.ReadTimeout == 0 {
		options.ReadTimeout = 5 * time.Second
	}
	if options.ReadHeaderTimeout == 0 {
		options.ReadHeaderTimeout = 3 * time.Second
	}
	if options.WriteTimeout == 0 {
		options.WriteTimeout = 10 * time.Second
	}
	if options.IdleTimeout == 0 {
		options.IdleTimeout = 120 * time.Second
	}
	s.server = &http.Server{
		Handler:           s.router,
		ReadTimeout:       options.ReadTimeout,
		ReadHeaderTimeout: options.ReadHeaderTimeout,
		WriteTimeout:      options.WriteTimeout,
		IdleTimeout:       options.IdleTimeout,
		MaxHeaderBytes:    options.MaxHeaderBytes,
	}

	if options == nil {
		return
	}
	// Configure server with provided options.
	if options.Debug {
		s.debug = true
	}
	if options.Port != 0 {
		s.port = options.Port
	}
	if options.TLS {
		s.TLS = true
	}
	if options.Logger != nil {
		s.logger = options.Logger
	}
	if options.AuthTokenExtractor != nil {
		s.authTokenExtractor = options.AuthTokenExtractor
	}
	if options.Authenticator != nil {
		s.authenticator = options.Authenticator
	}
	if options.IPExtractor != nil {
		s.ipExtractor = options.IPExtractor
	}
	if options.MaxRequestSize != 0 {
		s.maxRequestSize = options.MaxRequestSize
	}
	if options.ShutdownTimeout != 0 {
		s.shutdownTimeout = options.ShutdownTimeout
	}
}

// Headers
const (
	HeaderAuthorization        = "Authorization"
	HeaderContentEncoding      = "Content-Encoding"
	HeaderContentLength        = "Content-Length"
	HeaderContentType          = "Content-Type"
	ContentTypeApplicationJSON = "application/json; charset=UTF-8"
	HeaderXForwardedFor        = "X-Forwarded-For"
	HeaderXRealIP              = "X-Real-Ip"
	HeaderXRequestID           = "X-Request-Id"

	// Access control
	HeaderAccessControlRequestMethod    = "Access-Control-Request-Method"
	HeaderAccessControlRequestHeaders   = "Access-Control-Request-Headers"
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	HeaderAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"

	// Security
	HeaderStrictTransportSecurity = "Strict-Transport-Security"
	HeaderXContentTypeOptions     = "X-Content-Type-Options"
	HeaderXFrameOptions           = "X-Frame-Options"
	HeaderContentSecurityPolicy   = "Content-Security-Policy"
	HeaderReferrerPolicy          = "Referrer-Policy"
)

// NewServer creates a new Server.
func NewServer(options *ServerOptions) *Server {
	s := &Server{}
	s.ConfigureServer(options)
	return s
}

// Start...
func (s *Server) Start(ctx context.Context) error {
	ln, err := s.newListener(":" + strconv.Itoa(s.port))
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	s.logger.Debug(fmt.Sprintf("Server listening at %s.", ln.Addr().String()))

	select {
	case <-ctx.Done():
	case err := <-errCh:
		return err
	}
	return nil
}

// Stop shuts down the server gracefully.
func (s *Server) Stop() {
	if s.server == nil {
		return
	}

	s.logger.Debug("Starting server shutdown.")

	shutdownCtx, done := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer done()

	s.server.SetKeepAlivesEnabled(false)
	if err := s.server.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("Failed to gracefully shutdown server; force closing.", slog.String("error", err.Error()))
		s.server.Close()
	}

	s.logger.Debug("Server shutdown complete.")
}

// newListener...
func (s *Server) newListener(addr string) (net.Listener, error) {
	if s.TLS {
		autoTLSManager := autocert.Manager{
			Prompt: autocert.AcceptTOS,
			// Cache certificates to avoid issues with rate limits (https://letsencrypt.org/docs/rate-limits)
			Cache: autocert.DirCache("/var/www/.cache"),
			// HostPolicy: autocert.HostWhitelist("<DOMAIN>"),
		}
		tlsConfig := &tls.Config{
			GetCertificate: autoTLSManager.GetCertificate,
			NextProtos:     []string{acme.ALPNProto},
		}
		ln, err := tls.Listen("tcp", addr, tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create listener on %s: %w", addr, err)
		}
		return ln, nil
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener on %s: %w", addr, err)
	}
	return ln, nil
}
