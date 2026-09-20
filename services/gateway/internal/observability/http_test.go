package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPHandlerPreservesResponse(t *testing.T) {
	handler, err := NewHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	}))
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/bucket/key", nil))
	if response.Code != http.StatusCreated || response.Body.String() != "created" {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
}

func TestRouteNameDoesNotExposeObjectKey(t *testing.T) {
	if got := routeName("/private/customer-42/report.pdf"); got != "/{bucket}/{key}" {
		t.Fatalf("routeName() = %q", got)
	}
}
