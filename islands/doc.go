// Package islands is the Go core of templ-islands: interactive components
// for templ applications driven by a generic client runtime.
//
// Declare each interactive component once in the Go registry and the client
// runtime (served by RuntimeHandler) handles optimistic UI, server sync and
// rollback:
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
// The page markup carries data-island and data-* attributes that the runtime
// interprets. The manifest is served at /islands/manifest.json and describes
// the islands to the client (endpoints, fields, renderers, streams).
//
// Client runtime capabilities:
//
//   - Optimistic mutation (inc, toggle-text, class-toggle) with rollback
//   - JSON re-render (input, change, click, submit) with JS renderers
//   - File upload: automatic multipart + progress (islands:progress)
//   - Optimistic media previews (data-preview, createObjectURL)
//   - Real-time streams (SSE) with Last-Event-ID reconnect
//   - Multi-tab sync (BroadcastChannel)
//
// Full documentation: docs/usage.md, docs/SSE.md and docs/multitab.md in
// the project repository.
package islands
