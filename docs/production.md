# Production guide

Things that break templ-islands in real deployments — and how to fix them.
Written for "other platforms": reverse proxies, CDNs, auth, browsers, scaling.

## 1. Reverse proxies buffer the SSE stream

**Symptom:** real-time updates arrive in bursts (or never) behind nginx or a
CDN. The proxy is buffering the response instead of streaming it.

**Fixes:**
- The library already sends `X-Accel-Buffering: no` (nginx and Cloudflare
  respect it on origin responses).
- nginx: also set `proxy_buffering off;` for the `/events/` location and a long
  `proxy_read_timeout` (default 60s kills SSE).
- **Do not gzip `text/event-stream`.** Compressing a stream forces the proxy to
  buffer it. Keep `gzip off` (or exclude the type) for `/events/`.

## 2. Authentication in SSE: EventSource cannot send custom headers

**Symptom:** the stream endpoint returns 401 while your JSON endpoints work.

`EventSource` can only send **cookies**, never `Authorization` headers. If your
app authenticates with a JWT in a header, the SSE endpoint can't carry it.

**Options:**
- Use an auth cookie (mind `SameSite`) for the stream endpoint.
- Issue a short-lived stream token: the app validates the real auth, redirects
  to `/events/chat?token=...`, and the stream endpoint validates the token
  (never log query strings; expire fast).
- Keep the SSE endpoint public but only stream non-sensitive data (not for
  agent chats).

## 3. CSP (Content Security Policy)

If your app sends CSP headers, allow:

```
default-src 'self';
script-src 'self';          # the runtime and renderers are external files
connect-src 'self';         # fetch() + EventSource to your endpoints
```

The runtime does **not** use `eval`, so you do NOT need `unsafe-eval`. Inline
`onclick` attributes are not used either — no `unsafe-inline` required.

## 4. Version skew between the cached runtime and the manifest

The runtime JS is served with a long cache (`max-age=86400`) and the manifest
with `no-cache`. After a deploy, a browser can hold an **old runtime** and
fetch a **new manifest**.

**Impact:** harmless for additive changes (the old runtime ignores unknown
manifest fields) but can break if a manifest field the old runtime needs is
removed or renamed.

**Mitigation:** version the runtime URL per release, e.g. serve
`/islands/runtime.js` behind `/islands/runtime.js?v=<release>`, or bump the
cache-busting query when the runtime changes. Simple and effective.

## 5. Horizontal scaling: the event hub is the bottleneck

The example broker is in-memory: with two instances behind a load balancer, a
client connected to instance A never sees events published on instance B.

**This is an example limitation, not a library one.** The library only writes
SSE; who emits events is the app. For multi-instance production, back the hub
with Redis pub/sub, NATS or your message broker (GAIA already plans NATS).

## 6. Browsers

Supported: modern evergreen browsers (Chrome/Edge/Firefox/Safari 2019+).
Everything used — `EventSource`, `IntersectionObserver`, `AbortController`,
`fetch`, `CustomEvent`, `insertAdjacentHTML` — is available in all of them.

Notes:
- Load the runtime as a classic script (`<script src="..." defer>`), not
  `type="module"`: `document.currentScript` (used to derive the manifest URL)
  is null in modules. The fallback URL is documented in the code.
- iOS Safari kills SSE tabs in the background; `Last-Event-ID` resume covers it
  on return.

## 7. Multiple tabs

Two tabs with the same stream open two SSE connections and render twice. It
works, but it duplicates. A future improvement is sharing one connection per
domain key via `BroadcastChannel` (see the backlog).

## 8. Deploy checklist

- [ ] Reverse proxy: `X-Accel-Buffering`, `proxy_buffering off`, long timeout, no gzip on `/events/`
- [ ] Auth strategy for the SSE endpoint (cookie or short-lived token)
- [ ] CSP allows `connect-src 'self'`
- [ ] Runtime URL versioned per release (or accept additive-only changes)
- [ ] Broker backed by pub/sub if more than one instance
- [ ] `WriteSSERetry` (jitter) and heartbeats (`WriteSSEPing`) on every stream
