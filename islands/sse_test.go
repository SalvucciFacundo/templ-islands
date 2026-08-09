package islands

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteSSEFormat(t *testing.T) {
	rec := httptest.NewRecorder()
	err := WriteSSE(rec, map[string]any{"messages": []string{"hola"}})
	if err != nil {
		t.Fatal(err)
	}
	got := rec.Body.String()
	if !strings.HasPrefix(got, "data: ") || !strings.HasSuffix(got, "\n\n") {
		t.Fatalf("formato SSE invalido: %q", got)
	}
	if !strings.Contains(got, `"messages"`) {
		t.Fatalf("el evento no lleva el payload: %q", got)
	}
}

func TestSSEHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	SSEHeaders(rec)
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", cc)
	}
}

func TestWriteSSEOnResponseWriter(t *testing.T) {
	// http.ResponseWriter real (no solo el recorder) soporta Flush; el
	// recorder tambien, y WriteSSE no debe fallar.
	rec := httptest.NewRecorder()
	if err := WriteSSE(rec, map[string]any{"ok": true}); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}
