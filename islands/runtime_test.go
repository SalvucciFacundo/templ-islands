package islands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testRegistry() *Registry {
	reg := New()
	reg.Register("like",
		[]Field{
			{Name: "likes", Selector: "[data-mutate=likes]", Op: OpInc, Delta: 1},
			{Name: "liked", Selector: "", Op: OpClassToggle, Class: "liked"},
		},
		"/api/like/{post_id}", "POST")
	return reg
}

func TestRuntimeHandlerServesManifestDirect(t *testing.T) {
	reg := testRegistry()
	req := httptest.NewRequest(http.MethodGet, "/manifest.json", nil)
	rec := httptest.NewRecorder()
	reg.RuntimeHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%q", rec.Code, rec.Body.String())
	}
	var m map[string]manifestIsland
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("manifest is not valid JSON: %v", err)
	}
	like, ok := m["like"]
	if !ok {
		t.Fatalf("manifest missing island 'like': %v", m)
	}
	if like.Endpoint != "/api/like/{post_id}" {
		t.Fatalf("endpoint = %q", like.Endpoint)
	}
	if len(like.Fields) != 2 {
		t.Fatalf("fields = %d, want 2", len(like.Fields))
	}
}

func TestRuntimeHandlerServesJSDirect(t *testing.T) {
	reg := testRegistry()
	req := httptest.NewRequest(http.MethodGet, "/runtime.js", nil)
	rec := httptest.NewRecorder()
	reg.RuntimeHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if len(rec.Body.Bytes()) == 0 {
		t.Fatal("runtime.js body is empty")
	}
}

// TestMount simulates the example's mounting:
// mux.Handle("GET /islands/", http.StripPrefix("/islands/", reg.RuntimeHandler()))
func TestMountWithStripPrefix(t *testing.T) {
	reg := testRegistry()
	mux := http.NewServeMux()
	mux.Handle("GET /islands/", http.StripPrefix("/islands/", reg.RuntimeHandler()))

	for _, path := range []string{"/islands/manifest.json", "/islands/runtime.js"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200; body=%q", path, rec.Code, rec.Body.String())
		}
	}
}
