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
	r       *http.Request
	w       *statusWriter
	s       *Server
	Ctx     context.Context
	Logger  *slog.Logger
	ID      string
	IP      string
	Time    time.Time
	Session *Session
}

// Bind...
func (f *Flow) Bind(v any) error {
	dec := json.NewDecoder(f.r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		if ute, ok := err.(*json.UnmarshalTypeError); ok {
			return InvalidError("Unmarshal type error: expected=%v, got=%v, field=%v, offset=%v", ute.Type, ute.Value, ute.Field, ute.Offset).SetInternal(err)
		} else if se, ok := err.(*json.SyntaxError); ok {
			return InvalidError("Syntax error: offset=%v, error=%v", se.Offset, se.Error()).SetInternal(err)
		} else if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return InvalidError("Invalid input.").SetInternal(err)
		}
		return err
	}
	return nil
}

// Respond...
func (f *Flow) Respond(status int, v any) error {
	if v == nil {
		f.w.WriteHeader(status)
		return nil
	}

	f.w.Header().Add(HeaderContentType, ContentTypeApplicationJSON)
	f.w.WriteHeader(status)
	return json.NewEncoder(f.w).Encode(v)
}

// init...
func (f *Flow) init(w http.ResponseWriter, r *http.Request) {
	if f.w == nil {
		f.w = &statusWriter{ResponseWriter: w}
	} else {
		f.w.reset(w)
	}
	ctx := r.Context()
	id := uuid.New().String()
	ip := f.s.ipExtractor(r)
	now := time.Now().UTC()

	logger := f.s.logger.With(
		slog.String("request_path", r.URL.Path),
		slog.String("request_method", r.Method),
		slog.String("request_id", id),
		slog.String("request_ip", ip),
	)

	f.r = r
	f.Ctx = ctx
	f.Logger = logger
	f.ID = id
	f.IP = ip
	f.Time = now
	f.Session = nil
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
