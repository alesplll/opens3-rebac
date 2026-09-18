package httpapi

import (
	"io"
	"net/http"
	"strings"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/service"
)

type Handler struct{ objects service.ObjectService }

func NewHandler(objects service.ObjectService) http.Handler {
	return &Handler{objects: objects}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthz" {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "ok\n")
		return
	}
	bucket, key, ok := objectPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPut:
		h.putObject(w, r, bucket, key)
	case http.MethodGet:
		h.getObject(w, r, bucket, key)
	default:
		w.Header().Set("Allow", "GET, PUT")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func objectPath(path string) (bucket, key string, ok bool) {
	bucket, key, ok = strings.Cut(strings.TrimPrefix(path, "/"), "/")
	return bucket, key, ok && bucket != "" && key != ""
}
