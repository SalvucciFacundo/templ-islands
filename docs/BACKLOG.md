# Backlog templ-islands

Pendientes cosméticos y de largo plazo. Lo esencial está implementado (mira el
README para el estado completo). Esto es lo que quedó afuera deliberadamente,
con prioridades.

## Prioridad media

- [ ] **Adapter Datastar**: el fallback server-driven está hardcodeado a htmx
      (`hx-post`). El core es agnóstico; falta un helper de templ que emita los
      atributos `data-*` de Datastar según el adapter elegido.
- [ ] **Fallback sin JS (nivel 2)**: envolver los botones en `<form method="POST">`
      clásico para que funcionen con JavaScript 100% apagado (nicho <1%).
- [ ] **CLI con subcomando `generate` explícito** (`templ-islands generate` en vez
      de flags solos) + `--watch` para regenerar al editar templates.
- [ ] **Tabs/paneles**: cambiar de pestaña sin recargar (re-render de un contenedor
      según la tab activa).
- [ ] **Errores por campo en renderers**: patrón documentado para renderers que
      generan los errores inline dentro del HTML re-renderizado (el binding
      `[data-error-for]` ya existe para el form).

## Prioridad baja / infraestructura

- [ ] Dockerfile + ejemplo de deploy en una instancia propia.
- [ ] GoDoc completo para el paquete `islands`.
- [ ] Bench del modo client vs server-driven dentro del repo (el de `demo-social/`
      quedó fuera del repo).
- [ ] Package docs / doc.go para pkg.go.dev.

## Hecho recientemente (para referencia)

- ✅ Heartbeats SSE (`WriteSSEPing`) en el ejemplo del chat.
- ✅ Tests del runtime JS (11 tests de `runtime-core.js` con `node --test`).
- ✅ Cache headers en el runtime handler (JS inmutable con cache largo, manifiesto sin cache).
- ✅ Renderer compartido explícito (`renderer=` en `@island` → `WithRenderer`).
- ✅ Release automático en CI al pushear tags `v*`.
- ✅ Batching de mutaciones: **descartado con criterio** (sobreingeniería para web HTTP).

## Notas del análisis de otros ejemplos

Patrones derivados de los proyectos del autor (tecno-shop, sector-remeras,
appointments-app, dolar-hoy-arg, novel-editor, GAIA) y de apps CRUD en general.

**Cubiertos hoy:** toggle/contador (mutación con optimistic + data-key +
data-confirm), búsqueda (re-render input con data-debounce), filtros por select
(re-render change), click para expandir (re-render click), formularios (form
submit + field_errors), feedback (eventos), infinite scroll (intersect), chat
en vivo (stream SSE con Last-Event-ID), CSRF, anti-race (AbortController).
