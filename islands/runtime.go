package islands

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
)

//go:embed runtime-core.js
var runtimeCoreJS string

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
			// El core (funciones puras, testeado) se sirve ANTES del runtime
			// que lo consume. El JS embebido es inmutable entre builds: cache largo.
			w.Header().Set("Cache-Control", "public, max-age=86400")
			fmt.Fprintf(w, "%s\n%s", runtimeCoreJS, runtimeJS)
		case "manifest.json":
			w.Header().Set("Content-Type", "application/json")
			// El manifiesto cambia con el registro: nada de cache.
			w.Header().Set("Cache-Control", "no-cache")
			json.NewEncoder(w).Encode(r.Manifest())
		default:
			http.NotFound(w, req)
		}
	})
}
