# Backlog templ-islands

Pendientes cosméticos y de largo plazo. Lo esencial está implementado (mira el
README para el estado completo). Esto es lo que quedó afuera deliberadamente,
con prioridades.

## Prioridad alta (antes de producción real)

- [ ] **Stream SSE compartido entre pestañas (v2 de multitab)**: una sola
      conexión `EventSource` por origin con elección de líder, heartbeat y
      failover; el líder re-emite por `BroadcastChannel` y las demás pestañas
      re-renderizan desde el canal. Diseñado en [`multitab.md`](multitab.md),
      fuera de v1 por el costo de los edge cases.

## Prioridad media

- [ ] **Adapter Datastar**: el fallback server-driven está hardcodeado a htmx
      (`hx-post`). El core es agnóstico; falta un helper de templ que emita los
      atributos `data-*` de Datastar según el adapter elegido.
- [ ] **Fallback sin JS (nivel 2)**: envolver los botones en `<form method="POST">`
      clásico para que funcionen con JavaScript 100% apagado (nicho <1%).
- [ ] **Tabs/paneles**: cambiar de pestaña sin recargar (re-render de un contenedor
      según la tab activa).
- [ ] **Errores por campo en renderers**: patrón documentado para renderers que
      generan los errores inline dentro del HTML re-renderizado (el binding
      `[data-error-for]` ya existe para el form).

## Prioridad baja / infraestructura

- [ ] Dockerfile + ejemplo de deploy en una instancia propia.
- [ ] Bench del modo client vs server-driven dentro del repo (el de `demo-social/`
      quedó fuera del repo).

## Hecho recientemente (para referencia)

- ✅ GoDoc: `doc.go` con el package comment completo + ejemplo ejecutable
      (`ExampleRegistry_Register`) para pkg.go.dev; doc pública del paquete en
      inglés (pkg.go.dev es una superficie global).
- ✅ CLI `templ-islands generate` con subcomando + `--watch` (regenera al guardar
      un .templ; polling con debounce, sin dependencias).
- ✅ Multi-tab v1 (BroadcastChannel): mutaciones con `data-key` y form submits
      se sincronizan entre pestañas del mismo origin; el server gana, los peers
      aplican en silencio. E2E multi-pestaña real (like + refresh).
- ✅ E2E Playwright completo (9 tests: like, search, comments, chat SSE, field
      errors, upload con imagen, infinite scroll) — destapó y confirmó bugs que
      ningún test Go/JS veía.
- ✅ Chat SSE arreglado: los roots de stream usan `data-stream` (no `data-island`)
      y `renderInto` ahora recibe el nombre de la isla explícito (ba77ab8).
- ✅ Subida de archivos: multipart automático + `islands:progress` (XHR solo con
      archivos, fetch no mide progreso) + previews optimistas con
      `data-preview` (createObjectURL con revoke).
- ✅ Renderer compartido explícito (`renderer=` en `@island` → `WithRenderer`):
      `new-post` ahora lo declara (el happy path del form submit no se había
      probado en browser y fallaba con "renderer not found").
- ✅ Parity runner simula `escapeHtml` (el golden test estaba roto desde el
      commit inicial sin que nadie lo notara).
- ✅ Heartbeats SSE (`WriteSSEPing`) en el ejemplo del chat.
- ✅ Tests del runtime JS (14 tests de `runtime-core.js` con `node --test`).
- ✅ Cache headers en el runtime handler (JS inmutable con cache largo, manifiesto sin cache).
- ✅ Release automático en CI al pushear tags `v*`.
- ✅ Batching de mutaciones: **descartado con criterio** (sobreingeniería para web HTTP).

## Notas del análisis de otros ejemplos

Patrones derivados de los proyectos del autor (tecno-shop, sector-remeras,
appointments-app, dolar-hoy-arg, novel-editor, GAIA) y de apps CRUD en general.

**Cubiertos hoy:** toggle/contador (mutación con optimistic + data-key +
data-confirm), búsqueda (re-render input con data-debounce), filtros por select
(re-render change), click para expandir (re-render click), formularios (form
submit + field_errors), feedback (eventos), infinite scroll (intersect), chat
en vivo (stream SSE con Last-Event-ID), CSRF, anti-race (AbortController),
upload de archivos (multipart + islands:progress + previews optimistas),
multi-tab (BroadcastChannel: mutaciones y refrescos entre pestañas).
