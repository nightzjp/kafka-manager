package webassets

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestHandlerServesSPAAndKeepsAPIErrors(t *testing.T) {
	assets := fstest.MapFS{"index.html": {Data: []byte("app")}, "assets/app.js": {Data: []byte("js")}}
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "api missing", 404) })
	handler := NewHandler(api, assets)
	for _, path := range []string{"/", "/topics"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		if response.Code != 200 || response.Body.String() != "app" {
			t.Fatalf("%s => %d %q", path, response.Code, response.Body.String())
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/missing", nil))
	if response.Code != 404 || response.Body.String() != "api missing\n" {
		t.Fatalf("api => %d %q", response.Code, response.Body.String())
	}
	var _ fs.FS = assets
}
