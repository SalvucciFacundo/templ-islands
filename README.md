# templ-islands

![templ-islands banner](docs/assets/banner.jpg)

**Same templ component, rendered server-side or hydrated client-side under Go.**
Mark components as *islands*, and a generic embedded runtime handles optimistic
UI, local re-render, forms, infinite scroll, real-time streams, file uploads
with progress, optimistic media previews, tabs and multi-tab sync — without
touching JavaScript per island.

## What it solves

Server-driven apps (templ + htmx) re-render a fragment **on the server
for every interaction**. Under heavy interactive traffic that CPU cost adds up,
and every click pays a round-trip.

templ-islands flips the model: the page renders server-side with templ, but each
**island** runs its interactions in the browser with optimistic UI. The server
only persists and answers JSON.

The in-repo benchmark (`go test ./examples/social -bench=. -benchmem`) measures
the same representative feed in both modes:

| Mode | Feed payload | Cost to produce |
|---|---|---|
| Server-driven (htmx) | 10.9 KB HTML | 44.7 µs, 452 allocs |
| Island (client) | **1.9 KB JSON** | **4.2 µs, 2 allocs** |

The client mode transfers ~**6x less** per feed render and costs ~**10x less**
CPU and memory on the server. The render cost moves to the browser (the JS
renderer, kept in sync by the parity test) and the user gains optimistic UI.

## Quick start

```bash
go get github.com/SalvucciFacundo/templ-islands@latest
go install github.com/SalvucciFacundo/templ-islands/cmd/templ-islands@latest
```

1. **Declare** an island next to your component with `// @island` + `// @field`.
2. **Generate** the registry: `templ-islands generate --dir . --out islands_gen.go --package main`.
3. **Serve** the runtime: `mux.Handle("GET /islands/", http.StripPrefix("/islands/", reg.RuntimeHandler()))`.
4. Run the example with every capability: `cd examples/social && go run .` → http://localhost:8081.

A minimal island:

```templ
// @island like endpoint=/api/like/{post_id} method=POST
// @field likes selector=[data-mutate=likes] op=inc delta=1
templ LikeButton(post Post) {
	<button data-island="like" data-post-id={ strconv.Itoa(post.ID) } hx-post={ fmt.Sprintf("/like/%d", post.ID) } hx-swap="outerHTML">
		<span class="count" data-mutate="likes">{ post.Likes }</span>
	</button>
}
```

## Documentation

Pick a page — the README stays short on purpose:

| Page | What you'll find |
|---|---|
| [**Usage guide**](docs/usage.md) | The full contract, every capability with examples, runtime behaviors, server API, CLI and directives reference |
| [**Architecture**](docs/architecture.md) | Diagram, layout, design decisions, the three problems it solves |
| [**Production guide**](docs/production.md) | What breaks behind proxies/CDNs, auth in SSE, CSP, version skew, scaling |
| [**SSE design**](docs/SSE.md) | Real-time layer: retry, jitter, Last-Event-ID, heartbeats |
| [**Multi-tab design**](docs/multitab.md) | BroadcastChannel sync: protocol, phases, SSE sharing design |
| [**Proposal**](docs/propuesta-v2.md) | Original idea and tradeoff analysis (Spanish) |
| [**Backlog**](docs/BACKLOG.md) | Deferred ideas and priorities |

## Tests

```bash
go test ./...                     # unit + golden parity test (needs Node)
node --test islands/runtime-core.test.js   # runtime core JS tests
npx playwright test               # E2E browser suite (11 tests)
```

## License

[MIT](LICENSE)
