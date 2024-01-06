package flux

import (
	"log/slog"
	"os"
)

// DefaultLogger creates the default logger used by the server.
func (s *Server) DefaultLogger() *slog.Logger {
	logLevel := &slog.LevelVar{}
	if s.debug {
		logLevel.Set(slog.LevelDebug)
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
}
