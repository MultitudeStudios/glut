package flux

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// router...
type router struct {
	server *Server
	table  map[string]http.Handler
}

// ServeHTTP...
func (rt *router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		notFound(w, r)
		return
	}
	f, ok := rt.table[r.URL.Path[1:]]
	if !ok {
		notFound(w, r)
		return
	}

	defer func() {
		if err := recover(); err != nil {
			rt.handlePanic(w, r, err)
		}
	}()
	f.ServeHTTP(w, r)
}

var (
	panicResponse    = []byte(fmt.Sprintf(`{"code": "%s", "status": "%d", "message": "%s"}`, "internal", http.StatusInternalServerError, "Something went wrong."))
	notFoundResponse = []byte(fmt.Sprintf(`{"code": "%s", "status": "%d", "message": "%s"}`, "not_found", http.StatusNotFound, "Not found."))
)

// handlePanic...
func (rt *router) handlePanic(w http.ResponseWriter, r *http.Request, v interface{}) {
	rt.server.logger.Error("A panic occurred.", slog.Any("error", v), slog.String("trace", string(debug.Stack())))
	w.Header().Set(HeaderContentType, ContentTypeApplicationJSON)
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(panicResponse)
}

// notFound...
func notFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(HeaderContentType, ContentTypeApplicationJSON)
	w.WriteHeader(http.StatusNotFound)
	w.Write(notFoundResponse)
}
