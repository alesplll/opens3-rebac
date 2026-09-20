package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/handler/httpapi"
)

func TestSwaggerEndpoints(t *testing.T) {
	handler := httpapi.NewHandler(nil)

	document := httptest.NewRecorder()
	handler.ServeHTTP(document, httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil))
	if document.Code != http.StatusOK || !strings.Contains(document.Body.String(), `"/{bucket}/{key}"`) {
		t.Fatalf("Swagger document: status=%d body=%q", document.Code, document.Body.String())
	}

	ui := httptest.NewRecorder()
	handler.ServeHTTP(ui, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	if ui.Code != http.StatusOK {
		t.Fatalf("Swagger UI status = %d", ui.Code)
	}
}
