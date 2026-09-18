package httpapi

import (
	"io"
	"net/http"
	"strconv"

	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"go.uber.org/zap"
)

func (h *Handler) getObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	result, err := h.objects.Get(r.Context(), bucket, key)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if result.ContentType != "" {
		w.Header().Set("Content-Type", result.ContentType)
	}
	w.Header().Set("Content-Length", strconv.FormatInt(result.Size, 10))
	w.Header().Set("ETag", strconv.Quote(result.ETag))
	w.Header().Set("X-Object-Version-Id", result.VersionID)
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, result.Body); err != nil {
		logger.Error(r.Context(), "object response interrupted", zap.String("bucket", bucket), zap.String("key", key), zap.Error(err))
	}
}
