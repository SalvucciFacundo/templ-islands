package islands

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
)

// SSEHeaders prepares a response for Server-Sent Events. Call it when
// opening an /events/... endpoint in your app:
//
//	islands.SSEHeaders(w)
//	islands.WriteSSE(w, map[string]any{"messages": msgs})
//
// It also sets X-Accel-Buffering: no, so nginx (and proxies that honor it)
// do not buffer the stream and every event arrives live.
func SSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
}

// WriteSSE serializes data as an SSE event and flushes it so it arrives
// immediately: "data: {...}\n\n". The client runtime parses it as JSON and
// passes it to the stream island's renderer.
func WriteSSE(w http.ResponseWriter, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
		return err
	}
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	return nil
}

// WriteSSEID serializes an SSE event with an id: "id: N\ndata: {...}\n\n".
// The id enables Last-Event-ID resilience: when the connection drops, the
// browser sends the Last-Event-ID header on reconnect and the server can
// resend the events after it without losing messages.
func WriteSSEID(w http.ResponseWriter, id int, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "id: %d\ndata: %s\n\n", id, b); err != nil {
		return err
	}
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	return nil
}

// WriteSSERetry emits the retry: field with jitter so a mass reconnect
// (thundering herd) does not hit the server in the same second. The value is
// base + [0, jitter) milliseconds. Call it once when opening the stream,
// before the first event:
//
//	islands.SSEHeaders(w)
//	islands.WriteSSERetry(w, 3000, 2000)
func WriteSSERetry(w http.ResponseWriter, base, jitter int) {
	ms := base
	if jitter > 0 {
		ms += rand.IntN(jitter)
	}
	fmt.Fprintf(w, "retry: %d\n\n", ms)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// WriteSSEPing emits an SSE comment (": ping") that the browser ignores. It
// works as a heartbeat to keep the connection alive through proxies that cut
// long-lived connections. Call it periodically (e.g. every 15s) while there
// are no real events:
//
//	ticker := time.NewTicker(15 * time.Second)
//	...
//	case <-ticker.C:
//		islands.WriteSSEPing(w)
func WriteSSEPing(w http.ResponseWriter) {
	fmt.Fprint(w, ": ping\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
