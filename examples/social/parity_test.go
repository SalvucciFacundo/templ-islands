package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/SalvucciFacundo/templ-islands/examples/social/views"
)

var wsRegexp = regexp.MustCompile(`>\s+<`)

// normalize quita el whitespace entre tags para comparar estructura, no
// formato. Ambos renderers pasan por la misma normalizacion.
func normalize(s string) string {
	return strings.TrimSpace(wsRegexp.ReplaceAllString(s, "><"))
}

func renderTemplPostCards(posts []views.Post) string {
	var buf bytes.Buffer
	for _, p := range posts {
		views.PostCard(p).Render(context.Background(), &buf)
	}
	return buf.String()
}

// TestPostListParity es el golden test del doble renderer: el renderer JS
// (static/post-list.js) debe producir exactamente el mismo HTML que templ
// (views.PostCard) para los mismos datos. Si divergen, falla.
func TestPostListParity(t *testing.T) {
	posts := []views.Post{
		{ID: 1, Text: "Hola mundo", Likes: 7, Liked: false, AuthorID: 1, Following: false},
		{ID: 2, Text: "Segundo post con <em>HTML</em> & ampersand", Likes: 3, Liked: true, AuthorID: 2, Following: true},
	}

	templHTML := renderTemplPostCards(posts)

	fixtures, err := json.Marshal(posts)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("node", "parity_runner.js")
	cmd.Stdin = strings.NewReader(string(fixtures))
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("node parity runner: %v", err)
	}

	if got, want := normalize(string(out)), normalize(templHTML); got != want {
		t.Fatalf("paridad rota:\n--- JS renderer ---\n%s\n--- templ ---\n%s", got, want)
	}
}
