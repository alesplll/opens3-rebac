package httpapi_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/handler/httpapi"
	"github.com/alesplll/opens3-rebac/services/gateway/internal/service"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
)

// The transport test supplies only Get; other operations must not be called.
type getService struct {
	service.ObjectService
	result service.GetResult
}

func (s getService) Get(context.Context, string, string) (service.GetResult, error) {
	return s.result, nil
}

type failedReader struct{}

func (failedReader) Read([]byte) (int, error) { return 0, errors.New("storage stream failed") }

func TestGetAbortsIncompleteResponse(t *testing.T) {
	for _, tc := range []struct {
		name string
		body io.Reader
		size int64
	}{
		{"short clean EOF", strings.NewReader("abc"), 6},
		{"failure after declared bytes", io.MultiReader(strings.NewReader("abc"), failedReader{}), 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logger.SetNopLogger()
			handler := httpapi.NewHandler(getService{result: service.GetResult{Body: tc.body, Size: tc.size}})
			defer func() {
				if got := recover(); got != http.ErrAbortHandler {
					t.Fatalf("panic = %v, want http.ErrAbortHandler", got)
				}
			}()
			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/bucket/key", nil))
		})
	}
}
