# Capa multi-pestaña — sincronización por BroadcastChannel (diseño)

Estado: **diseñada (fase v1), pendiente de implementar**. La fase v2 (stream SSE
compartido con liderazgo) está documentada al final.

## Problema

Dos pestañas del mismo origin muestran la misma isla (feed con likes, posts).
Una mutación en la pestaña A (like, follow, publicar) no se refleja en la
pestaña B hasta que B recarga. Cada pestaña vive aislada aunque comparten el
mismo servidor como fuente de verdad.

## Objetivo

- Las mutaciones exitosas en una pestaña se reflejan al instante en las otras
  que tengan la misma isla visible.
- Sin cambiar el modelo actual: el **server sigue siendo la source of truth**;
  el canal solo transporta la respuesta que el server ya dio.
- Degradación a cero: si `BroadcastChannel` no existe o algo falla, cada
  pestaña funciona como hoy.

## Modelo

```
pestaña A                     pestaña B
  like(3) → POST /api/like/3     |
      |                          |
  server responde {likes:8}      |
      |                          |
  A aplica local (applyServer)   |
  A emite al canal ─────────────► B recibe {island,key,data}
      |                          B busca [data-island][data-key]
      |                          B aplica con applyServer (silencioso)
```

El canal es un bus *same-origin*. El dato viaja del server → A → canal → B,
nunca se inventa en el cliente.

## Protocolo (fase v1)

Mensajes JSON sobre `new BroadcastChannel("templ-islands")`.

| type | Campos | Emisor | Receptor |
|---|---|---|---|
| `mutated` | `island`, `key`, `data`, `at` | A aplica una mutación con `data-key` y el server respondió | B busca sus instancias `island+key` y aplica `data` con `applyServer` |
| `refresh` | `island`, `target`, `at` | A completó un form submit exitoso (nuevo post, cambio de estado global) | B re-fetchea el endpoint de la isla y re-renderiza su `target` |

Reglas del protocolo:

- **`mutated` solo con `data-key`.** Sin key no hay dominio compartido que
  sincronizar (cada instancia es privada de su pestaña).
- **`data` es la respuesta completa del server** (el mismo objeto que A aplicó).
  Nada se recalcula en B.
- **Filtrado por página:** si B no tiene ninguna instancia `island+key` en su
  DOM, ignora el mensaje. A en `/` y B en `/chat` no se pisan.
- **El peer aplica en silencio:** no re-emite `islands:success` (evita toasts
  duplicados en B). El que mutó es el único que ve feedback.
- **`refresh` para listas:** para feeds/colecciones mandar la lista entera por
  el canal sería pesado; B re-fetchea su endpoint (un GET más, igual al que
  haría un re-render manual).

## Cuándo emite el runtime

| Evento local | Mensaje | Condición |
|---|---|---|
| Mutación exitosa (click con `data-key`) | `mutated` | `cfg.fields` aplicado, server respondió 2xx |
| Form submit exitoso | `refresh` | `renderInto` del submit terminó OK |

El chat SSE **no se toca en v1**: cada pestaña mantiene su `EventSource` (el
server ya difunde por publicación). El SSE compartido es la fase v2.

## Ciclo de vida

- El canal se crea una vez, al arrancar el runtime (junto al manifest).
- `BroadcastChannel` se cierra solo cuando la pestaña cierra; no requiere
  limpieza manual.
- Feature-detect: `if (window.BroadcastChannel)` — sin soporte, el runtime
  queda idéntico al actual.
- `at` (timestamp del emisor) permite descartar mensajes viejos si un peer
  estuvo dormido (threshold opcional, p. ej. > 60s se ignora).

## Edge cases y límites

| Caso | Comportamiento |
|---|---|
| B no tiene la isla | Ignora el mensaje (búsqueda de instancias vacía) |
| B tiene el post pero stale | `applyServer` escribe los valores del server; los elementos inexistentes se saltan (`elementFor` → null) |
| Race A↔B mutan casi a la vez | Último mensaje del server gana; mismo modelo que las instancias de una página hoy |
| `BroadcastChannel` ausente | Cero sincronización, cero errores |
| El canal se rompe (bug de runtime) | Cada pestaña sigue su camino; la mutación local ya fue persistida por A |

## Fase v2 (documentada, no implementada): stream SSE compartido

Compartir un solo `EventSource` entre pestañas del mismo chat para no abrir N
conexiones por origin.

- **Liderazgo:** una pestaña abre el `EventSource` y re-emite cada evento por el
  canal (`{type:"stream-event", island, data}`); las demás no abren conexión y
  re-renderizan desde el canal.
- **Elección:** claim perezoso con id de pestaña; el primer claim (o el menor
  id) es líder; los peers confirman.
- **Heartbeat + failover:** el líder emite `ping` periódico; si el canal queda
  silencio por N segundos, otra pestaña toma el rol y abre el stream (con
  `Last-Event-ID` el server reenvía lo perdido — ya soportado).
- **Costo real:** elección ambigua con pestañas que nacen/mueren a la vez,
  doble emisión transitoria (dos líderes brevemente), complejidad de debug.
  Por eso queda fuera de v1: el beneficio (1 conexión vs N) no justifica el
  riesgo hasta que una app real lo pida.

## Plan de implementación (v1)

1. `islands/runtime.js`:
   - Abrir el canal tras `loadManifest` (con feature-detect).
   - Emitir `mutated` al final del `.then` de una mutación con `data-key`.
   - Emitir `refresh` al final del `renderInto` exitoso de un form submit.
   - Listener del canal: `mutated` → buscar instancias + `applyServer`;
     `refresh` → `reRender` de la isla con su `target` local.
2. `tests/social.spec.js` — E2E multi-pestaña (Playwright, dos `page` del mismo
   `context` comparten el canal):
   - `like en A → contador y clase liked en B sin recargar`.
   - `post en A → feed de B se actualiza (refresh)`.
3. Docs: esta página + sección corta en `usage.md`.

## Decisiones descartadas (con criterio)

- **Mandar el evento SSE por el canal en v1:** es la fase v2 (liderazgo).
- **Aplicar la mutación en B con el payload local de A (sin esperar server):**
  rompe "el server gana"; el mensaje lleva la respuesta del server, no el
  optimismo.
- **Un solo mensaje universal con la lista entera:** costoso para feeds; el
  tipo `refresh` (re-fetch) cubre los casos de colección.
- **WebSocket para todo:** sobreingeniería; SSE + BroadcastChannel cubren el
  espectro push/mutación con menos estado.
