# templ-islands

![templ-islands banner](docs/assets/banner.jpg)

**Same templ component, rendered server-side or hydrated client-side under Go.**
Mark components as *islands*, and a generic embedded runtime handles optimistic
UI, local re-render, forms, infinite scroll and real-time streams — without
touching JavaScript per island.

## What it solves

Server-driven apps (templ + htmx/Datastar) re-render a fragment **on the server
for every interaction**. Under heavy interactive traffic that CPU cost adds up,
and every click pays a round-trip.

templ-islands flips the model: the page renders server-side with templ, but each
**island** runs its interactions in the browser with optimistic UI. The server
only persists and answers JSON.

The validation demo measured the difference with real traffic:

| Mode | Renders server-side | Cost per like |
|---|---|---|
| Server-driven (htmx) | 300 | 0.061 ms render + round-trip |
| Island (client) | **0** | 0.010 ms persist only |

The client mode eliminated **100% of the server-side render**, and rendering one
tiny button cost ~6x more than the entire JSON operation.

## Quick start

```bash
go get github.com/SalvucciFacundo/templ-islands@latest
go install github.com/SalvucciFacundo/templ-islands/cmd/templ-islands@latest
```

1. **Declare** an island next to your component with `// @island` + `// @field`.
2. **Generate** the registry: `templ-islands generate -dir . -out islands_gen.go -package main`.
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
| [**SSE design**](docs/SSE.md) | Real-time layer: retry, jitter, Last-Event-ID, heartbeats |
| [**Proposal**](docs/propuesta-v2.md) | Original idea and tradeoff analysis (Spanish) |
| [**Backlog**](docs/BACKLOG.md) | Deferred ideas and priorities |

## Tests

```bash
go test ./...            # unit + golden parity test (needs Node)
node --test islands/     # runtime core JS tests
```

## License

[MIT](LICENSE)
