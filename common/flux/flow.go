package flux

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Flow...
type Flow struct {
	r      *http.Request
	w      *statusWriter
	s      *Server
	ctx    context.Context
	logger *slog.Logger
	id     string
	ip     string
	start  time.Time
	sess   *Session
}

// ID...
func (f *Flow) ID() string {
	return f.id
}

// IP...
func (f *Flow) IP() string {
	return f.ip
}

// Context...
func (f *Flow) Context() context.Context {
	return f.ctx
}

// Logger...
func (f *Flow) Logger() *slog.Logger {
	return f.logger
}

// Start...
func (f *Flow) Start() time.Time {
	return f.start
}

// Session...
func (f *Flow) Session() *Session {
	return f.sess
}

// init...
func (f *Flow) init(w http.ResponseWriter, r *http.Request) {
	if f.w == nil {
		f.w = &statusWriter{ResponseWriter: w}
	} else {
		f.w.reset(w)
	}
	ctx := r.Context()
	start := time.Now().UTC()
	id := uuid.New().String()
	ip := f.s.ipExtractor(r)

	logger := f.s.logger.With(
		slog.String("request_path", r.URL.Path),
		slog.String("request_method", r.Method),
		slog.String("request_id", id),
		slog.String("request_ip", ip),
	)

	f.r = r
	f.ctx = ctx
	f.logger = logger
	f.id = id
	f.ip = ip
	f.start = start
}

// bind...
func (f *Flow) bind(v any) error {
	dec := json.NewDecoder(f.r.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(&v)
	if ute, ok := err.(*json.UnmarshalTypeError); ok {
		return InvalidError("Unmarshal type error: expected=%v, got=%v, field=%v, offset=%v", ute.Type, ute.Value, ute.Field, ute.Offset).SetInternal(err)
	} else if se, ok := err.(*json.SyntaxError); ok {
		return InvalidError("Syntax error: offset=%v, error=%v", se.Offset, se.Error()).SetInternal(err)
	} else if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return InvalidError("Invalid input.").SetInternal(err)
	}
	return err
}

// respond...
func (f *Flow) respond(status int, v any) error {
	if status == 0 {
		status = http.StatusOK
	}
	if v == nil {
		f.w.WriteHeader(status)
		return nil
	}

	f.w.Header().Add(HeaderContentType, ContentTypeApplicationJSON)
	f.w.WriteHeader(status)
	return json.NewEncoder(f.w).Encode(v)
}

// statusWriter...
type statusWriter struct {
	http.ResponseWriter
	Status int
}

// WriteHeader...
func (w *statusWriter) WriteHeader(status int) {
	w.Status = status
	w.ResponseWriter.WriteHeader(status)
}

// reset...
func (sw *statusWriter) reset(w http.ResponseWriter) {
	sw.ResponseWriter = w
	sw.Status = 0
}
