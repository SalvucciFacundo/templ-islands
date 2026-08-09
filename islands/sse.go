package islands

import (
	"encoding/json"
	"fmt"
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
