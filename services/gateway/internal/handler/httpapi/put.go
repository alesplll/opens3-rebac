package httpapi

import (
	"net/http"
	"strconv"
)

// PutObject godoc
// @Summary Store an object in an existing bucket
// @Description Development-only API. No authentication, authorization or quota enforcement. A failed response after Storage may have an unknown Metadata outcome; do not retry blindly.
// @Tags objects
// @Accept octet-stream
// @Param bucket path string true "Bucket name"
// @Param key path string true "Object key (may contain slashes)"
// @Param body body string true "Raw object bytes"
// @Param Content-Type header string false "MIME type"
// @Success 200
// @Failure 400 {string} string
// @Failure 404 {string} string
// @Failure 503 {string} string
// @Failure 507 {string} string
// @Router /{bucket}/{key} [put]
func (h *Handler) PutObject(w http.ResponseWriter, r *http.Request) {
	bucket, key, ok := objectPath(r.URL.Path)
	if !ok {
		http.Error(w, "bucket and key are required", http.StatusBadRequest)
		return
	}
	h.putObject(w, r, bucket, key)
}

func (h *Handler) putObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	var size *int64
	if r.ContentLength >= 0 {
		length := r.ContentLength
		size = &length
	}
	result, err := h.objects.Put(r.Context(), bucket, key, r.Body, size, contentType)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	w.Header().Set("ETag", strconv.Quote(result.ETag))
	w.Header().Set("X-Object-Version-Id", result.VersionID)
	w.WriteHeader(http.StatusOK)
}
