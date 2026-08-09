# Capa SSE — islas en tiempo real (diseño)

Estado: **diseñada, pendiente de implementar**. La necesitan GAIA (chat en vivo,
agente trabajando, tareas en progreso) y MisCanarios (likes/comentarios de otros
en vivo). Es la pieza que cubre el problema 3 del doc de propuesta: consistencia
en tiempo real (que lo de OTROS aparezca solo).

## Por qué SSE y no WebSocket

| | SSE (Server-Sent Events) | WebSocket |
|---|---|---|
| Dirección | Unidireccional server → client | Bidireccional |
| Transporte | HTTP estándar (`text/event-stream`) | Upgrade de protocolo |
| Reconexión | **Automática nativa** (`EventSource`) | Manual |
| Complejidad | Baja | Alta (estado de conexión, heartbeats) |
| Para GAIA | Perfecto: el envío del usuario es POST normal; la recepción es server → client | Sobredimensionado |

En GAIA el usuario NO necesita enviar por el socket: el mensaje va por form submit
(POST, ya cubierto por la librería). Solo la recepción es push → **SSE es la
elección correcta**. WebSocket sería sobreingeniería.

## Contrato

### En el .templ (directivas)

```templ
// @island chat-stream endpoint=/events/chat stream=true render=/static/chat-renderer.js
templ ChatStream() {
	<div data-stream="chat-stream" data-target="#chat-messages"></div>
}
```

- La isla stream se declara como las demás (`@island`), pero con `stream=true`.
- El DOM activa el stream con un elemento `data-stream="<nombre>"` + `data-target`
  (igual que las otras islas declaran su target en el DOM). Sin elemento, el
  stream no se abre — es opt-in por página.

### En el manifiesto (generado)

```json
{
  "chat-stream": {
    "endpoint": "/events/chat",
    "method": "GET",
    "stream": true,
    "render": "/static/chat-renderer.js"
  }
}
```

### En el registro Go

```go
reg.RegisterStream("chat-stream", "/events/chat", "/static/chat-renderer.js")
```

## Flujo

```
Usuario escribe en el chat
   └─ form submit (LIBRERÍA, ya existe) ──► POST /api/chat ──► el agente procesa
                                                                    │
Agente genera mensajes / cambia de subagente / agrega tareas          │
   └─ el servidor emite eventos SSE a /events/chat ◄──────────────────┘
                                                                    │
Navegador: EventSource(/events/chat) (runtime)                       │
   └─ por cada evento: renderInto(target, renderer, data) ──► re-render del chat
```

## Runtime client (`islands/runtime.js`)

```js
// Al cargar el manifiesto, por cada [data-stream] en la pagina:
function startStreams() {
  document.querySelectorAll("[data-stream]").forEach(function (root) {
    var cfg = manifest[root.dataset.stream];
    if (!cfg || !cfg.stream) return;
    var es = new EventSource(cfg.endpoint);
    es.onmessage = function (e) {
      var data = JSON.parse(e.data);
      renderInto(root, cfg, data);        // reusa el renderer existente
      emit("islands:success", { island: root.dataset.stream, stream: true });
    };
    es.onerror = function () {
      emit("islands:error", { island: root.dataset.stream, stream: true, error: "stream closed" });
      // EventSource reconecta solo; el servidor controla el backoff con "retry:".
    };
  });
}
```

Reutiliza `renderInto` y los renderers existentes — **un renderer más en el
manifiesto, cero lógica nueva por stream**. La delegación de eventos ya hace que
las islas dentro del contenido re-renderizado sigan funcionando.

## Servidor Go (helpers en el paquete islands)

```go
// SSEHeaders prepara la respuesta para Server-Sent Events.
func SSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
}

// WriteSSE serializa data como un evento SSE: "data: {...}\n\n".
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
```

El hub de eventos (quiénes se suscriben a qué stream, fan-out de los eventos del
agente) es responsabilidad de la app — GAIA ya tiene el backend del agente; el
hub conecta el agente con los suscriptores SSE.

## Decisiones de diseño (v1)

1. **Estado completo, no delta.** El evento SSE trae el estado (ej: la lista de
   mensajes) y el renderer re-renderiza el target entero. Simple y consistente;
   para chats/tareas de decenas de items es trivial. **Append incremental queda
   en BACKLOG** (necesita la operación `append` del runtime).
2. **Opt-in por página.** El stream solo se abre si hay `[data-stream]` en el
   DOM — coherente con el resto de la librería.
3. **Reconexión nativa** de `EventSource`; el servidor controla el retry con el
   campo `retry:`. Para evitar el thundering herd (mil clientes caidos que
   reconectan al mismo segundo), el servidor emite `retry:` con jitter usando
   el helper:

   ```go
   islands.SSEHeaders(w)
   islands.WriteSSERetry(w, 3000, 2000) // 3000 + [0, 2000) ms aleatorios
   ```

   El runtime NO reimplementa backoff: el `EventSource` respeta el `retry:` del
   servidor, que es donde se controla el jitter.
4. **SSE unidireccional.** El envío del usuario sigue siendo form submit (POST).
5. El evento que llega puede traer **cualquier JSON** que el renderer sepa
   interpretar; el renderer es la fuente de la forma del DOM (mismo patrón que
   el re-render por click).

## Casos de uso

### GAIA (el que la destapó)

| Panel | Evento SSE | Renderer |
|---|---|---|
| Chat | `{messages: [...]}` → re-render del chat | chat-renderer.js |
| Agente trabajando | `{agent: "explorer"}` → re-render del indicador | agent-status.js |
| Tareas | `{tasks: [...]}` → re-render de la lista | tasks-renderer.js |

El envío del chat es form submit; los proyectos son re-render por click. GAIA
queda 100% cubierto: librería (interacción) + SSE (eventos del agente).

### MisCanarios

| Evento | Qué actualiza |
|---|---|
| `{likes: 9}` de un post | mutación atómica del contador (o re-render del post) |
| `{comments: [...]}` de un post | re-render de la lista de comentarios |

## Plan de implementación (cuando se ataque)

1. `islands/registry.go`: `RegisterStream` + campo `Stream` en `Island` + manifiesto.
2. `islands/runtime.js`: `startStreams()` + EventSource + integración con `renderInto`.
3. `islands/sse.go`: helpers `SSEHeaders` / `WriteSSE`.
4. CLI: parsear `stream=true` en `@island` → generar `RegisterStream`.
5. Ejemplo: un chat simulado con eventos (ej: un endpoint que emite un evento
   cada N segundos, o un form que dispara el evento).
6. Tests: unitario del helper SSE; el parity se reutiliza.
7. README: sección "Streams (SSE)".

## Riesgos honestos

- **Re-render completo por evento**: para listas largas hay que pasar a append
  (BACKLOG). El límite real es el renderer.
- **EventSource y proxies**: algunos proxies cortan conexiones largas; el
  `retry:` + heartbeats del server lo mitigan.
- **Los eventos SSE no son transaccionales**: si el browser pierde eventos
  durante la reconexión, el estado puede quedar desactualizado — el renderer
  debe poder pedir el estado completo al reconectar (o el server mandar el
  estado en el primer evento tras la reconexión).
