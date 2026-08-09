package islands

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
)

// SSEHeaders prepara una respuesta para Server-Sent Events. Usala al abrir
// un endpoint /events/... en tu app:
//
//	islands.SSEHeaders(w)
//	islands.WriteSSE(w, map[string]any{"messages": msgs})
func SSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
}

// WriteSSE serializa data como un evento SSE y hace flush para que llegue de
// inmediato: "data: {...}\n\n". El runtime client lo parsea como JSON y lo
// pasa al renderer de la isla stream.
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

// WriteSSERetry emite el campo retry: con jitter para que una reconexion
// masiva (thundering herd) no golpee al servidor en el mismo segundo.
// El valor es base + [0, jitter) milisegundos. Llamalo una vez al abrir el
// stream, antes del primer evento:
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
