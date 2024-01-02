package log

import (
	"log/slog"
	"os"
)

// logLevel represents the logging level. The level is info by default.
// The log level can be changed to debug by calling the SetDebug() function.
var logLevel *slog.LevelVar = &slog.LevelVar{}

// New creates a new logger.
func New() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
}

// SetDebug sets the global log level to debug.
func SetDebug() {
	logLevel.Set(slog.LevelDebug)
}
