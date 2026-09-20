package httpapi

import (
	"errors"
	"net/http"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrBucketNotFound):
		http.Error(w, "bucket not found", http.StatusNotFound)
		return
	case errors.Is(err, service.ErrBodyRead):
		http.Error(w, "could not read request body", http.StatusBadRequest)
		return
	case errors.Is(err, service.ErrEmptyStorageStream):
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	code := http.StatusBadGateway
	switch status.Code(err) {
	case codes.InvalidArgument:
		code = http.StatusBadRequest
	case codes.NotFound:
		code = http.StatusNotFound
	case codes.ResourceExhausted:
		code = http.StatusInsufficientStorage
	case codes.Unavailable:
		code = http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		code = http.StatusGatewayTimeout
	}
	http.Error(w, http.StatusText(code), code)
}
