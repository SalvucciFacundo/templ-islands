package main

// Bench del modo client (runtime) vs server-driven (htmx).
//
// Mide el costo de UNA accion representativa en cada modo, sobre los mismos
// datos, para documentar el tradeoff de la libreria:
//
//   - Modo server-driven (htmx): el server renderiza HTML y htmx lo swappea.
//     El server hace mas trabajo (render templ) y transfiere mas bytes.
//   - Modo client (runtime): el server responde JSON y el runtime re-renderiza
//     con JS (renderer). Menos bytes y menos trabajo de server por accion;
//     el costo se corre al cliente (el renderer duplica el markup, controlado
//     por el parity test).
//
// Correr con: go test ./examples/social -bench=. -benchmem

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/SalvucciFacundo/templ-islands/examples/social/views"
)

// benchPosts es un feed representativo (los 12 posts del store de ejemplo).
func benchPosts() []views.Post {
	posts := make([]views.Post, 0, 12)
	for i := 1; i <= 12; i++ {
		posts = append(posts, views.Post{
			ID:       i,
			Text:     "Post de ejemplo #" + string(rune('0'+i/10)) + string(rune('0'+i%10)) + " — renderizado en el servidor, isla en el cliente.",
			Likes:    i * 7,
			AuthorID: i,
		})
	}
	return posts
}

// renderFeedHTML renderiza el feed completo como lo haria el modo htmx.
func renderFeedHTML(posts []views.Post) []byte {
	var buf bytes.Buffer
	for _, p := range posts {
		views.PostCard(p).Render(context.Background(), &buf)
	}
	return buf.Bytes()
}

// BenchmarkFeedServerHTML: el server renderiza el feed HTML (modo htmx).
func BenchmarkFeedServerHTML(b *testing.B) {
	posts := benchPosts()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = renderFeedHTML(posts)
	}
}

// BenchmarkFeedClientJSON: el server responde el mismo feed como JSON (modo
// client); el render del HTML corre en el browser.
func BenchmarkFeedClientJSON(b *testing.B) {
	posts := benchPosts()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := json.Marshal(posts); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLikeServerHTML: la accion like, modo htmx (el server renderiza el
// boton entero y htmx lo reemplaza).
func BenchmarkLikeServerHTML(b *testing.B) {
	post := views.Post{ID: 1, Text: "post", Likes: 7, AuthorID: 1}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		views.LikeButton(post).Render(context.Background(), &buf)
	}
}

// BenchmarkLikeClientJSON: la accion like, modo client (el server responde
// solo los campos que el runtime aplica con applyServer).
func BenchmarkLikeClientJSON(b *testing.B) {
	post := views.Post{ID: 1, Text: "post", Likes: 7, Liked: true, AuthorID: 1}
	payload := map[string]any{"likes": post.Likes, "liked": post.Liked}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := json.Marshal(payload); err != nil {
			b.Fatal(err)
		}
	}
}

// TestPayloadSizes documenta la diferencia de bytes por accion en cada modo.
func TestPayloadSizes(t *testing.T) {
	posts := benchPosts()
	html := renderFeedHTML(posts)
	jsonFeed, err := json.Marshal(posts)
	if err != nil {
		t.Fatal(err)
	}

	var likeBuf bytes.Buffer
	views.LikeButton(posts[0]).Render(context.Background(), &likeBuf)
	likeJSON, _ := json.Marshal(map[string]any{"likes": posts[0].Likes, "liked": posts[0].Liked})

	t.Logf("feed:    server-driven HTML %d B | client JSON %d B (%0.1f%% del HTML)",
		len(html), len(jsonFeed), float64(len(jsonFeed))*100/float64(len(html)))
	t.Logf("like:    server-driven HTML %d B | client JSON %d B (%0.1f%% del HTML)",
		len(likeBuf.Bytes()), len(likeJSON), float64(len(likeJSON))*100/float64(len(likeBuf.Bytes())))
}
