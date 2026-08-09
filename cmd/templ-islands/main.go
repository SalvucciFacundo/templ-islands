// Command templ-islands is the CLI companion to the islands library.
//
// Usage:
//
//	templ-islands -dir . -out islands_gen.go -package main
//
// It scans .templ files for // @island directives and generates the Go
// registry call for each island:
//
//	// @island like endpoint=/api/like/{post_id} method=POST
//	// @field likes selector=[data-mutate=likes] op=inc delta=1
//	// @field liked selector=[data-mutate=label] op=toggle-text true=Liked false=Like
//	templ LikeButton(post Post) { ... }
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var (
		dir = flag.String("dir", ".", "directorio donde escanear los .templ")
		out = flag.String("out", "islands_gen.go", "archivo Go de salida")
		pkg = flag.String("package", "main", "package del archivo generado")
	)
	flag.Parse()

	isles, err := scanDir(*dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "templ-islands:", err)
		os.Exit(1)
	}

	if len(isles) == 0 {
		fmt.Println("no se encontraron islas (busca // @island en los .templ)")
		return
	}

	if err := generate(*out, *pkg, isles); err != nil {
		fmt.Fprintln(os.Stderr, "templ-islands:", err)
		os.Exit(1)
	}

	total := 0
	for _, i := range isles {
		total += len(i.Fields)
	}
	fmt.Printf("generadas %d islas (%d campos) -> %s\n", len(isles), total, *out)
}
