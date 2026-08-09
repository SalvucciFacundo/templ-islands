// registry.go: the Go-declared island registry — the source of truth for the
// manifest consumed by the client runtime.
package islands

import (
	"sort"
	"sync"
)

// OptimisticOp describes how a field is mutated locally before the server
// answers, so the user sees the change instantly (optimistic UI).
type OptimisticOp string

const (
	// OpNone means the field waits for the server response; no local change.
	OpNone OptimisticOp = ""
	// OpInc increments (or decrements) a numeric element by Field.Delta.
	OpInc OptimisticOp = "inc"
	// OpToggleText alternates an element between TrueText and FalseText.
	OpToggleText OptimisticOp = "toggle-text"
	// OpClassToggle toggles Field.Class on the island root from a boolean response field.
	OpClassToggle OptimisticOp = "class-toggle"
)

// Field binds one JSON response field to a DOM element and an optimistic op.
type Field struct {
	// Name is the field name in the JSON response (e.g. "likes", "liked").
	Name string
	// Selector targets the element inside the island root. Empty means the root itself.
	Selector string
	// Op is the optimistic mutation applied before the server answers.
	Op OptimisticOp
	// Delta is the local change for OpInc (e.g. +1 or -1).
	Delta int
	// TrueText and FalseText are the two texts for OpToggleText.
	TrueText  string
	FalseText string
	// Class is the class toggled for OpClassToggle.
	Class string
}

// Island is one registered island.
type Island struct {
	// Name matches data-island="..." in the markup.
	Name string
	// Endpoint is the JSON endpoint the runtime calls. It may contain
	// {placeholder} tokens filled from data-* attributes (e.g. /api/like/{post_id}).
	Endpoint string
	// Method is the HTTP method for the endpoint.
	Method string
	// Fields binds response fields to DOM elements.
	Fields []Field
	// Render is the URL of the JS renderer for re-render islands.
	// When set, the runtime re-renders the target from JSON data instead
	// of applying atomic mutations.
	Render string
	// Trigger is the DOM event that fires the re-render (e.g. "input").
	Trigger string
	// Stream marks a real-time island: the runtime opens an EventSource to
	// Endpoint and re-renders the target whenever the server emits an event.
	Stream bool
	// Renderer is the name of another island whose JS renderer this island
	// reuses (e.g. a "post-more" island rendering with the "post-list"
	// renderer). Empty means the renderer is registered under this island's
	// own name.
	Renderer string
}

// RenderOption tweaks a re-render or stream island at registration time.
type RenderOption func(*Island)

// WithRenderer makes the island use the JS renderer registered under
// another island's name. Use it when two islands share one renderer:
//
//	reg.RegisterRender("post-more", "/api/posts", "GET", "/static/post-list.js", "intersect",
//		islands.WithRenderer("post-list"))
func WithRenderer(islandName string) RenderOption {
	return func(i *Island) { i.Renderer = islandName }
}

// Registry holds all registered islands and is the single source of truth.
type Registry struct {
	mu    sync.RWMutex
	isles map[string]Island
}

// New creates an empty registry.
func New() *Registry {
	return &Registry{isles: make(map[string]Island)}
}

// RegisterStream adds a real-time island: the runtime subscribes to Endpoint
// via SSE and re-renders the target with the JS renderer at render on every
// event the server emits. See docs/SSE.md for the full design.
func (r *Registry) RegisterStream(name, endpoint, render string, opts ...RenderOption) {
	r.mu.Lock()
	defer r.mu.Unlock()
	i := Island{
		Name:     name,
		Endpoint: endpoint,
		Method:   "GET",
		Render:   render,
		Stream:   true,
	}
	for _, o := range opts {
		o(&i)
	}
	r.isles[name] = i
}

// Register adds an island under name.
func (r *Registry) Register(name string, fields []Field, endpoint, method string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.isles[name] = Island{
		Name:     name,
		Endpoint: endpoint,
		Method:   method,
		Fields:   fields,
	}
}

// RegisterRender adds a re-render island: the runtime fetches JSON from
// Endpoint on Trigger and re-renders the target using the JS renderer at
// Render. Unlike Register, this island has no atomic fields — the whole
// target is re-rendered from data. Method is the HTTP method used
// (GET for filters/search, POST for form submits).
func (r *Registry) RegisterRender(name, endpoint, method, render, trigger string, opts ...RenderOption) {
	r.mu.Lock()
	defer r.mu.Unlock()
	i := Island{
		Name:     name,
		Endpoint: endpoint,
		Method:   method,
		Render:   render,
		Trigger:  trigger,
	}
	for _, o := range opts {
		o(&i)
	}
	r.isles[name] = i
}

// Island returns a registered island by name.
func (r *Registry) Island(name string) (Island, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	i, ok := r.isles[name]
	return i, ok
}

// Names returns the registered island names, sorted for determinism.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.isles))
	for n := range r.isles {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// manifestField is the JSON shape of a field, consumed by the client runtime.
type manifestField struct {
	Name     string `json:"name"`
	Selector string `json:"selector,omitempty"`
	Op       string `json:"op,omitempty"`
	Delta    int    `json:"delta,omitempty"`
	True     string `json:"true,omitempty"`
	False    string `json:"false,omitempty"`
	Class    string `json:"class,omitempty"`
}

type manifestIsland struct {
	Endpoint string          `json:"endpoint"`
	Method   string          `json:"method"`
	Fields   []manifestField `json:"fields,omitempty"`
	Render   string          `json:"render,omitempty"`
	Trigger  string          `json:"trigger,omitempty"`
	Stream   bool            `json:"stream,omitempty"`
	Renderer string          `json:"renderer,omitempty"`
}

// Manifest returns the JSON-serializable view of the registry.
func (r *Registry) Manifest() map[string]manifestIsland {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]manifestIsland, len(r.isles))
	for _, i := range r.isles {
		fields := make([]manifestField, 0, len(i.Fields))
		for _, f := range i.Fields {
			fields = append(fields, manifestField{
				Name:     f.Name,
				Selector: f.Selector,
				Op:       string(f.Op),
				Delta:    f.Delta,
				True:     f.TrueText,
				False:    f.FalseText,
				Class:    f.Class,
			})
		}
		out[i.Name] = manifestIsland{
			Endpoint: i.Endpoint,
			Method:   i.Method,
			Fields:   fields,
			Render:   i.Render,
			Trigger:  i.Trigger,
			Stream:   i.Stream,
			Renderer: i.Renderer,
		}
	}
	return out
}
