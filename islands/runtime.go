package islands

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"path"
)

//go:embed runtime.js
var runtimeJS string

// RuntimeHandler serves the client runtime (runtime.js) and the generated
// manifest (manifest.json). Mount it under any prefix:
//
//	mux.Handle("GET /islands/", http.StripPrefix("/islands/", reg.RuntimeHandler()))
//
// The handler matches by basename, so it works with or without StripPrefix
// and regardless of the mount point.
func (r *Registry) RuntimeHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch path.Base(req.URL.Path) {
		case "runtime.js":
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			w.Write([]byte(runtimeJS))
		case "manifest.json":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(r.Manifest())
		default:
			http.NotFound(w, req)
		}
	})
}
