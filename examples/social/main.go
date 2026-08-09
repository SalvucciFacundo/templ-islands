// Command social is the migrated demo: the same feed as demo-social, but the
// island is declared ONCE in the Go registry and the generic client runtime
// (served by the library) handles optimistic UI, sync and rollback.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
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

	// GET / — feed SSR con la primera pagina + el sentinel de infinite scroll.
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		views.Layout("Demo Social", views.PostList(store.SearchPaged("", 1, 3))).Render(r.Context(), w)
	})

	// GET /api/posts?q=...&page=N&per=M — datos JSON. Con page pagina el
	// resultado (infinite scroll); sin page devuelve la lista completa
	// filtrada (busqueda, re-render por input).
	mux.HandleFunc("GET /api/posts", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		per, _ := strconv.Atoi(r.URL.Query().Get("per"))
		if per == 0 {
			per = 5
		}
		posts := store.SearchPaged(q, page, per)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(posts)
	})

	// POST /api/posts — form submit (isla new-post): crea el post y devuelve
	// la lista completa para que el renderer re-renderice el feed. Los errores
	// de validacion vuelven como field_errors para el binding automatico.
	// El form puede venir urlencoded (sin archivos) o multipart (con imagen).
	mux.HandleFunc("POST /api/posts", func(w http.ResponseWriter, r *http.Request) {
		r.ParseMultipartForm(10 << 20) // 10MB en memoria; el resto a disco temp
		text := strings.TrimSpace(r.FormValue("text"))
		if text == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"field_errors": map[string]string{"text": "El texto no puede estar vacio"},
			})
			return
		}
		image := ""
		if file, _, err := r.FormFile("image"); err == nil {
			defer file.Close()
			if image, err = saveUpload(file); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]any{
					"field_errors": map[string]string{"image": "Solo imagenes (png/jpg/gif/webp) de hasta 5MB"},
				})
				return
			}
		}
		store.Create(text, image)
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

	// GET /events/chat — stream SSE de la isla chat-stream. Reanuda por
	// Last-Event-ID: si el navegador reconecta, se reenvia el historial
	// posterior al ultimo evento recibido (resiliencia sin perder mensajes).
	mux.HandleFunc("GET /events/chat", func(w http.ResponseWriter, r *http.Request) {
		islands.SSEHeaders(w)
		// Reconexion con jitter: mil clientes caidos no vuelven al mismo segundo.
		islands.WriteSSERetry(w, 3000, 2000)

		after := 0
		if h := r.Header.Get("Last-Event-ID"); h != "" {
			if n, err := strconv.Atoi(h); err == nil {
				after = n
			}
		}

		if after > 0 {
			for _, ev := range broker.History(after) {
				islands.WriteSSEID(w, ev.ID, ev.Data)
			}
		} else {
			islands.WriteSSEID(w, 0, map[string]any{"messages": store.ChatMessages()})
		}

		ch := broker.Subscribe()
		defer broker.Unsubscribe(ch)

		// Heartbeat: mantiene el stream vivo a traves de proxies que cortan
		// conexiones largas (comentario SSE que el navegador ignora).
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				islands.WriteSSEPing(w)
			case ev := <-ch:
				if err := islands.WriteSSEID(w, ev.ID, ev.Data); err != nil {
					return
				}
			}
		}
	})

	log.Println("templ-islands example en http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", mux))
}

// maxUploadBytes limita el tamano de las imagenes subidas (5MB).
const maxUploadBytes = 5 << 20

// saveUpload guarda una imagen subida en static/uploads/ y devuelve su URL
// publica. Valida el tipo REAL por contenido (http.DetectContentType, no el
// header del cliente, que es facil de falsear) y el tamano maximo.
func saveUpload(f multipart.File) (string, error) {
	head := make([]byte, 512)
	n, err := f.Read(head)
	if err != nil && n == 0 {
		return "", errors.New("archivo vacio")
	}

	var ext string
	switch http.DetectContentType(head[:n]) {
	case "image/png":
		ext = ".png"
	case "image/jpeg":
		ext = ".jpg"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	default:
		return "", errors.New("no es una imagen")
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	if err := os.MkdirAll("static/uploads", 0o755); err != nil {
		return "", err
	}

	name := fmt.Sprintf("uploads/%d%s", time.Now().UnixNano(), ext)
	dst, err := os.Create(filepath.Join("static", name))
	if err != nil {
		return "", err
	}
	defer dst.Close()

	written, err := io.Copy(dst, io.LimitReader(f, maxUploadBytes+1))
	if err != nil {
		os.Remove(dst.Name())
		return "", err
	}
	if written > maxUploadBytes {
		os.Remove(dst.Name())
		return "", errors.New("imagen muy grande")
	}
	return "/static/" + name, nil
}
