package httpapi

import (
	"io"
	"net/http"
	"strconv"

	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"go.uber.org/zap"
)

// GetObject godoc
// @Summary Read the current object version
// @Description Development-only API. Streams object bytes; range and version selection are not supported.
// @Tags objects
// @Produce octet-stream
// @Param bucket path string true "Bucket name"
// @Param key path string true "Object key (may contain slashes)"
// @Success 200 {file} binary
// @Failure 404 {string} string
// @Failure 502 {string} string
// @Failure 503 {string} string
// @Failure 504 {string} string
// @Router /{bucket}/{key} [get]
func (h *Handler) GetObject(w http.ResponseWriter, r *http.Request) {
	bucket, key, ok := objectPath(r.URL.Path)
	if !ok {
		http.Error(w, "bucket and key are required", http.StatusBadRequest)
		return
	}
	h.getObject(w, r, bucket, key)
}

func (h *Handler) getObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	result, err := h.objects.Get(r.Context(), bucket, key)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if closer, ok := result.Body.(io.Closer); ok {
		defer closer.Close()
	}
	if result.ContentType != "" {
		w.Header().Set("Content-Type", result.ContentType)
	}
	w.Header().Set("Content-Length", strconv.FormatInt(result.Size, 10))
	w.Header().Set("ETag", strconv.Quote(result.ETag))
	w.Header().Set("X-Object-Version-Id", result.VersionID)
	w.WriteHeader(http.StatusOK)
	n, err := io.Copy(w, result.Body)
	if err == nil && n != result.Size {
		err = io.ErrUnexpectedEOF
	}
	if err != nil {
		logger.Error(r.Context(), "object response interrupted", zap.String("bucket", bucket), zap.String("key", key), zap.Error(err))
		// Headers may already be on the wire; abort instead of completing a successful response.
		panic(http.ErrAbortHandler)
	}
}
