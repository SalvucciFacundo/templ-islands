// Package islands es el core Go de templ-islands: islas interactivas para
// aplicaciones templ con un runtime client genérico.
//
// Declarás cada componente interactivo una vez en el registro Go y el runtime
// client (servido por RuntimeHandler) se encarga de la UI optimista, la
// sincronización con el server y el rollback:
//
//	reg := islands.New()
//	reg.Register("like",
//		[]islands.Field{
//			{Name: "likes", Op: islands.OpInc, Selector: "[data-mutate=likes]", Delta: 1},
//			{Name: "liked", Op: islands.OpToggleText, Selector: "[data-mutate=label]", TrueText: "Liked", FalseText: "Like"},
//		},
//		"/api/like/{post_id}", "POST")
//	mux.Handle("GET /islands/", http.StripPrefix("/islands/", reg.RuntimeHandler()))
//
// El markup de la página lleva data-island y data-* que el runtime
// interpreta. El manifest se sirve en /islands/manifest.json y describe las
// islas al cliente (endpoints, campos, renderers, streams).
//
// Capacidades del runtime client:
//
//   - Mutación optimista (inc, toggle-text, class-toggle) con rollback
//   - Re-render desde JSON (input, change, click, submit) con renderers JS
//   - Subida de archivos: multipart automático + progreso (islands:progress)
//   - Vistas previas optimistas de media (data-preview, createObjectURL)
//   - Streams en tiempo real (SSE) con reconexión por Last-Event-ID
//   - Sincronización multi-pestaña (BroadcastChannel)
//
// Documentación completa: docs/usage.md, docs/SSE.md y docs/multitab.md en
// el repositorio del proyecto.
package islands
