package main

import (
	"strings"
	"testing"
)

func TestParseIsland(t *testing.T) {
	s, err := parseIsland("// @island like endpoint=/api/like/{post_id} method=POST")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "like" || s.Endpoint != "/api/like/{post_id}" || s.Method != "POST" {
		t.Fatalf("got %+v", s)
	}
}

func TestParseIslandDefaultsMethod(t *testing.T) {
	s, err := parseIsland("// @island like endpoint=/api/like/1")
	if err != nil {
		t.Fatal(err)
	}
	if s.Method != "POST" {
		t.Fatalf("default method = %q, want POST", s.Method)
	}
}

func TestParseIslandMissingEndpoint(t *testing.T) {
	if _, err := parseIsland("// @island like"); err == nil {
		t.Fatal("want error for missing endpoint")
	}
}

func TestParseFieldInc(t *testing.T) {
	f, err := parseField("// @field likes selector=[data-mutate=likes] op=inc delta=1")
	if err != nil {
		t.Fatal(err)
	}
	if f.Name != "likes" || f.Selector != "[data-mutate=likes]" || f.Op != "inc" || f.Delta != 1 {
		t.Fatalf("got %+v", f)
	}
}

func TestParseFieldToggleText(t *testing.T) {
	f, err := parseField("// @field liked selector=[data-mutate=label] op=toggle-text true=Liked false=Like")
	if err != nil {
		t.Fatal(err)
	}
	if f.Op != "toggle-text" || f.TrueText != "Liked" || f.FalseText != "Like" {
		t.Fatalf("got %+v", f)
	}
}

func TestParseIslandRender(t *testing.T) {
	s, err := parseIsland("// @island post-list endpoint=/api/posts method=GET render=/static/post-list.js trigger=input")
	if err != nil {
		t.Fatal(err)
	}
	if s.Render != "/static/post-list.js" || s.Trigger != "input" || s.Method != "GET" {
		t.Fatalf("got %+v", s)
	}
}

func TestParseIslandRenderWithoutTrigger(t *testing.T) {
	if _, err := parseIsland("// @island x endpoint=/api/x render=/static/x.js"); err == nil {
		t.Fatal("want error for render without trigger")
	}
}

func TestOpGoName(t *testing.T) {
	cases := map[string]string{
		"inc":         "OpInc",
		"toggle-text": "OpToggleText",
		"class-toggle": "OpClassToggle",
		"desconocido": "OpNone",
	}
	for in, want := range cases {
		if got := opGoName(in); got != want {
			t.Errorf("opGoName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateIslandSelectorExists(t *testing.T) {
	text := `templ LikeButton(post Post) {
		<span data-mutate="likes">7</span>
	}`
	isle := IslandSpec{Name: "like", Fields: []FieldSpec{
		{Name: "likes", Selector: "[data-mutate=likes]", Op: "inc", Delta: 1},
	}}
	if err := validateIsland("test.templ", text, isle); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateIslandSelectorMissing(t *testing.T) {
	text := `templ LikeButton(post Post) {
		<span data-mutate="likes">7</span>
	}`
	isle := IslandSpec{Name: "like", Fields: []FieldSpec{
		{Name: "likes", Selector: "[data-mutate=likes]", Op: "inc", Delta: 1},
		{Name: "bad", Selector: "[data-mutate=noexiste]", Op: "inc", Delta: 1},
	}}
	err := validateIsland("test.templ", text, isle)
	if err == nil {
		t.Fatal("want error for missing selector")
	}
	if !strings.Contains(err.Error(), "noexiste") {
		t.Fatalf("error does not name the missing mutate: %v", err)
	}
}

func TestValidateIslandSkipsRootSelectors(t *testing.T) {
	text := `templ FollowButton(userID int, following bool) {
		<button>Follow</button>
	}`
	isle := IslandSpec{Name: "follow", Fields: []FieldSpec{
		{Name: "following", Op: "class-toggle", Class: "following"},
	}}
	if err := validateIsland("test.templ", text, isle); err != nil {
		t.Fatalf("root selector should not be validated: %v", err)
	}
}
