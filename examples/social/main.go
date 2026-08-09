// Command social is the migrated demo: the same feed as demo-social, but the
// island is declared ONCE in the Go registry and the generic client runtime
// (served by the library) handles optimistic UI, sync and rollback.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/SalvucciFacundo/templ-islands/examples/social/views"
	"github.com/SalvucciFacundo/templ-islands/islands"
)

func main() {
	store := NewStore()
	broker := NewBroker()

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

	// POST /api/posts — form submit (isla new-post): crea el post y devuelve
	// la lista completa para que el renderer re-renderice el feed.
	mux.HandleFunc("POST /api/posts", func(w http.ResponseWriter, r *http.Request) {
		text := strings.TrimSpace(r.FormValue("text"))
		if text == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"error": "texto vacio"})
			return
		}
		store.Create(text)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(store.Posts())
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

	// GET /api/comments/{post_id} — datos para la isla comments (click -> re-render).
	mux.HandleFunc("GET /api/comments/{post_id}", func(w http.ResponseWriter, r *http.Request) {
		postID, err := strconv.Atoi(r.PathValue("post_id"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(store.Comments(postID))
	})

	// POST /api/delete_comment/{id} — elimina (data-confirm) y devuelve la
	// lista actualizada del post para re-renderizar.
	mux.HandleFunc("POST /api/delete_comment/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		comments, ok := store.DeleteComment(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(comments)
	})

	// GET /chat — el chat con el agente (form submit + stream SSE).
	mux.HandleFunc("GET /chat", func(w http.ResponseWriter, r *http.Request) {
		views.Layout("Demo Social — Chat", views.ChatPage()).Render(r.Context(), w)
	})

	// POST /api/chat — form submit de la isla chat-form: agrega el mensaje y
	// difunde; el agente "responde" solo (simulado) y eso llega por SSE.
	mux.HandleFunc("POST /api/chat", func(w http.ResponseWriter, r *http.Request) {
		text := strings.TrimSpace(r.FormValue("message"))
		if text == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"error": "mensaje vacio"})
			return
		}
		store.AddChatMessage("user", text)
		broker.Publish(map[string]any{"messages": store.ChatMessages()})

		// El agente responde SOLO (simulado): llega por SSE sin que el
		// usuario haga nada — el problema 3 del diseño.
		go func(replyTo string) {
			time.Sleep(600 * time.Millisecond)
			store.AddChatMessage("agent", "Procesado: "+replyTo)
			broker.Publish(map[string]any{"messages": store.ChatMessages()})
		}(text)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"messages": store.ChatMessages()})
	})

	// GET /events/chat — stream SSE de la isla chat-stream: envia el estado
	// al conectar y despues cada evento que difunde el broker.
	mux.HandleFunc("GET /events/chat", func(w http.ResponseWriter, r *http.Request) {
		islands.SSEHeaders(w)
		ch := broker.Subscribe()
		defer broker.Unsubscribe(ch)

		islands.WriteSSE(w, map[string]any{"messages": store.ChatMessages()})
		for {
			select {
			case <-r.Context().Done():
				return
			case msg := <-ch:
				if err := islands.WriteSSE(w, msg); err != nil {
					return
				}
			}
		}
	})

	log.Println("templ-islands example en http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", mux))
}
