package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// IslandSpec is one island declared with // @island in a .templ file.
type IslandSpec struct {
	Name     string
	Endpoint string
	Method   string
	Render   string
	Trigger  string
	Fields   []FieldSpec
}

// FieldSpec is one field declared with // @field.
type FieldSpec struct {
	Name     string
	Selector string
	Op       string
	Delta    int
	TrueText string
	FalseText string
	Class    string
}

// scanDir walks dir recursively and returns every island declared in .templ files.
func scanDir(dir string) ([]IslandSpec, error) {
	var isles []IslandSpec
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".templ" {
			return nil
		}
		found, err := scanFile(path)
		if err != nil {
			return err
		}
		isles = append(isles, found...)
		return nil
	})
	return isles, err
}

// scanFile reads one .templ and extracts islands from comment directives:
//
//	// @island like endpoint=/api/like/{post_id} method=POST
//	// @field likes selector=[data-mutate=likes] op=inc delta=1
//	// @field liked selector=[data-mutate=label] op=toggle-text true=Liked false=Like
//	templ LikeButton(post Post) { ... }
func scanFile(path string) ([]IslandSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")

	var isles []IslandSpec
	var cur *IslandSpec
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "// @island"):
			spec, err := parseIsland(trimmed)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			isles = append(isles, spec)
			cur = &isles[len(isles)-1]
		case strings.HasPrefix(trimmed, "// @field") && cur != nil:
			f, err := parseField(trimmed)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			cur.Fields = append(cur.Fields, f)
		case strings.HasPrefix(trimmed, "templ "):
			// Un nuevo componente empieza: las directivas siguientes
			// pertenecen a otra isla.
			cur = nil
		}
	}

	// Validacion de selectores: cada data-mutate declarado debe existir
	// en el template. Selector roto = error de generacion, no bug en runtime.
	// OJO: el cuerpo se construye SIN las lineas de directivas, porque la
	// propia directiva @field contiene la cadena data-mutate=<name>.
	var body []string
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "// @") {
			continue
		}
		body = append(body, line)
	}
	text := strings.Join(body, "\n")
	for _, isle := range isles {
		if err := validateIsland(path, text, isle); err != nil {
			return nil, err
		}
	}
	return isles, nil
}

// validateIsland checks that every field selector referencing a data-mutate
// exists in the template text. Empty selectors (root mutations like
// class-toggle) are not validated.
func validateIsland(path, text string, isle IslandSpec) error {
	for _, f := range isle.Fields {
		if f.Selector == "" {
			continue
		}
		mut, ok := dataMutateFromSelector(f.Selector)
		if !ok {
			// Selector custom (no data-mutate): no lo validamos por ahora.
			continue
		}
		if !strings.Contains(text, `data-mutate="`+mut+`"`) && !strings.Contains(text, `data-mutate=`+mut) {
			return fmt.Errorf("%s: isla %q: el selector %q no existe en el template (falta data-mutate=%q)", path, isle.Name, f.Selector, mut)
		}
	}
	return nil
}

// dataMutateFromSelector extracts the name from "[data-mutate=<name>]".
func dataMutateFromSelector(selector string) (string, bool) {
	trimmed := strings.TrimSpace(selector)
	if !strings.HasPrefix(trimmed, "[data-mutate=") || !strings.HasSuffix(trimmed, "]") {
		return "", false
	}
	return strings.TrimSuffix(strings.TrimPrefix(trimmed, "[data-mutate="), "]"), true
}

// parseIsland parses: // @island <name> key=value...
// Keys: endpoint, method (mutacion) | render, trigger (re-render).
func parseIsland(line string) (IslandSpec, error) {
	tokens := strings.Fields(line)
	if len(tokens) < 3 {
		return IslandSpec{}, fmt.Errorf("directiva @island invalida: %q (uso: // @island <name> endpoint=... method=...)", line)
	}
	spec := IslandSpec{Name: tokens[2], Method: "POST"}
	for _, tok := range tokens[3:] {
		k, v, ok := strings.Cut(tok, "=")
		if !ok {
			return IslandSpec{}, fmt.Errorf("directiva @island: par invalido %q", tok)
		}
		switch k {
		case "endpoint":
			spec.Endpoint = v
		case "method":
			spec.Method = v
		case "render":
			spec.Render = v
		case "trigger":
			spec.Trigger = v
		}
	}
	if spec.Endpoint == "" {
		return IslandSpec{}, fmt.Errorf("isla %q: falta endpoint=", spec.Name)
	}
	if spec.Render != "" && spec.Trigger == "" {
		return IslandSpec{}, fmt.Errorf("isla %q: render= sin trigger=", spec.Name)
	}
	return spec, nil
}

// parseField parses: // @field <name> key=value...
func parseField(line string) (FieldSpec, error) {
	tokens := strings.Fields(line)
	if len(tokens) < 3 {
		return FieldSpec{}, fmt.Errorf("directiva @field invalida: %q (uso: // @field <name> op=...)", line)
	}
	f := FieldSpec{Name: tokens[2]}
	for _, tok := range tokens[3:] {
		k, v, ok := strings.Cut(tok, "=")
		if !ok {
			return FieldSpec{}, fmt.Errorf("directiva @field: par invalido %q", tok)
		}
		switch k {
		case "selector":
			f.Selector = v
		case "op":
			f.Op = v
		case "delta":
			d, err := strconv.Atoi(v)
			if err != nil {
				return FieldSpec{}, fmt.Errorf("field %q: delta invalido %q", f.Name, v)
			}
			f.Delta = d
		case "true":
			f.TrueText = v
		case "false":
			f.FalseText = v
		case "class":
			f.Class = v
		}
	}
	if f.Op == "" {
		return FieldSpec{}, fmt.Errorf("field %q: falta op=", f.Name)
	}
	return f, nil
}
