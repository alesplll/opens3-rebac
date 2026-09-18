package httpapi

import (
	"net/http"
	"strconv"
)

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
