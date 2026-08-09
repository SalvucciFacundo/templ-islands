// Command social is the migrated demo: the same feed as demo-social, but the
// island is declared ONCE in the Go registry and the generic client runtime
// (served by the library) handles optimistic UI, sync and rollback.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/SalvucciFacundo/templ-islands/examples/social/views"
	"github.com/SalvucciFacundo/templ-islands/islands"
)

func main() {
	store := NewStore()

	// El contrato vive en el .templ (// @island + // @field).
	// templ-islands generate escanea las directivas y genera RegisterIslands.
	reg := islands.New()
	RegisterIslands(reg)

	mux := http.NewServeMux()

	// El runtime de la libreria: JS generico + manifiesto, embebidos.
	// Se monta en /islands/ (fuera de /static/) para no competir con el FileServer.
	mux.Handle("GET /islands/", http.StripPrefix("/islands/", reg.RuntimeHandler()))
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// GET / — feed SSR completo con la isla de busqueda (re-render).
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		views.Layout("Demo Social", views.PostList(store.Posts())).Render(r.Context(), w)
	})

	// GET /api/posts?q=... — datos JSON para la isla post-list (re-render).
	mux.HandleFunc("GET /api/posts", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		posts := store.Search(q)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(posts)
	})

	// POST /like/{id} — fallback server-driven (modo A). El boton conserva
	// hx-post; si el runtime client no carga, htmx hace esto.
	mux.HandleFunc("POST /like/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if _, ok := store.Like(id); !ok {
			http.NotFound(w, r)
			return
		}
		post, _ := store.Get(id)
		views.LikeButton(post).Render(r.Context(), w)
	})

	// POST /api/like/{id} — modo client: persiste y devuelve JSON, cero render.
	mux.HandleFunc("POST /api/like/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		post, ok := store.Like(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"likes": post.Likes, "liked": post.Liked})
	})

	// POST /follow/{id} — fallback server-driven para la isla follow.
	mux.HandleFunc("POST /follow/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if _, ok := store.Follow(id); !ok {
			http.NotFound(w, r)
			return
		}
		post, _ := store.Get(id)
		views.FollowButton(post.AuthorID, post.Following).Render(r.Context(), w)
	})

	// POST /api/follow/{id} — modo client para la isla follow.
	mux.HandleFunc("POST /api/follow/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		post, ok := store.Follow(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"following": post.Following})
	})

	log.Println("templ-islands example en http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", mux))
}
