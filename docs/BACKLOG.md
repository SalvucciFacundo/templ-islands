# Backlog templ-islands

Pendientes cosméticos y de largo plazo. Los esenciales (trigger change, store por
clave, error visible, CI, tag v0.1.0, form submit) están implementados; esto es
lo que quedó afuera deliberadamente.

## Diseñado, pendiente de implementar

- [ ] **Capa SSE (streams en tiempo real)** — ver [docs/SSE.md](SSE.md). La
      necesitan GAIA (chat/agente/tareas en vivo) y MisCanarios (lo de otros en
      tiempo real). Diseño completo: RegisterStream, startStreams(), helpers
      SSEHeaders/WriteSSE, CLI stream=true.

## Prioridad alta (antes de producción real)

- [ ] **Errores por campo en forms**: hoy el form submit emite `islands:error` con
      el body del server. El renderer JS puede mostrar errores inline, pero falta
      un patrón documentado y un helper de render de errores.
- [ ] **Confirmación antes de mutaciones destructivas**: delete con confirm dialog
      (el click handler del runtime no pregunta antes de disparar).
- [ ] **Append / "load more"**: el re-render reemplaza el target completo; falta
      un modo `append` para paginación e infinite scroll.

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
- [ ] **Tests del runtime JS**: extraer funciones puras (fillPlaceholders, ops) y
      testearlas con Node sin DOM.

## Prioridad baja / infraestructura

- [ ] Dockerfile + ejemplo de deploy en una instancia propia.
- [ ] Cache headers en el runtime handler (runtime.js/manifest.json).
- [ ] Documentar CSRF en endpoints JSON (responsabilidad de la app).
- [ ] GoDoc completo para el paquete `islands`.
- [ ] Bench del modo client vs server-driven dentro del repo (el de `demo-social/`
      quedó fuera del repo).
- [ ] GitHub Actions para el tag/release automático al pushear tags `v*`.

## Notas del análisis de otros ejemplos

Patrones derivados de los proyectos del autor (tecno-shop, sector-remeras,
appointments-app, dolar-hoy-arg, novel-editor) y de apps CRUD en general.
Los cubiertos hoy: toggle/contador (mutación), búsqueda (re-render input),
filtros por select (re-render change), formularios (form submit), feedback
(eventos). Los no cubiertos están arriba.
