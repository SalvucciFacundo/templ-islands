# Backlog templ-islands

Pendientes cosméticos y de largo plazo. Lo esencial está implementado (mira el
README para el estado completo). Esto es lo que quedó afuera deliberadamente,
con prioridades.

## Prioridad alta (antes de producción real)

- [ ] **Errores por campo en renderers**: el binding automático de `field_errors`
      ya existe para `[data-error-for]`, pero falta un patrón documentado para
      renderers que generan los errores inline dentro del HTML re-renderizado.
- [ ] **Heartbeats en streams SSE**: para proxies que cortan conexiones largas,
      el servidor debería emitir un comentario (`: ping\n\n`) periódicamente
      para mantener el stream vivo.
- [ ] **Tests del runtime JS**: extraer funciones puras (fillPlaceholders,
      applyFieldErrors, ops) y testearlas con Node sin DOM.

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
- [ ] **Renderer compartido explícito**: hoy un renderer registra su función bajo
      varios nombres de isla (ej: `comments` y `delete-comment`); falta un campo
      `renderer` en el manifiesto para reutilizar sin duplicar el registro.

## Prioridad baja / infraestructura

- [ ] Dockerfile + ejemplo de deploy en una instancia propia.
- [ ] Cache headers en el runtime handler (runtime.js/manifest.json).
- [ ] GoDoc completo para el paquete `islands`.
- [ ] Bench del modo client vs server-driven dentro del repo (el de `demo-social/`
      quedó fuera del repo).
- [ ] GitHub Actions para el tag/release automático al pushear tags `v*`.

## Notas del análisis de otros ejemplos

Patrones derivados de los proyectos del autor (tecno-shop, sector-remeras,
appointments-app, dolar-hoy-arg, novel-editor, GAIA) y de apps CRUD en general.

**Cubiertos hoy:** toggle/contador (mutación con optimistic + data-key +
data-confirm), búsqueda (re-render input con data-debounce), filtros por select
(re-render change), click para expandir (re-render click), formularios (form
submit + field_errors), feedback (eventos), infinite scroll (intersect), chat
en vivo (stream SSE con Last-Event-ID), CSRF, anti-race (AbortController).

**Descartados con criterio:** batching de mutaciones (sobreingeniería para web
HTTP; los likes a recursos distintos son endpoints independientes y el disable
cubre los rápidos al mismo recurso). Ver discusión en el historial de diseño.
