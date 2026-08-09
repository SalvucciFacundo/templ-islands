// Command templ-islands is the CLI companion to the islands library.
//
// Usage:
//
//	templ-islands generate [--dir .] [--out islands_gen.go] [--package main] [--watch]
//
// It scans .templ files for // @island directives and generates the Go
// registry call for each island:
//
//	// @island like endpoint=/api/like/{post_id} method=POST
//	// @field likes selector=[data-mutate=likes] op=inc delta=1
//	// @field liked selector=[data-mutate=label] op=toggle-text true=Liked false=Like
//	templ LikeButton(post Post) { ... }
//
// With --watch it stays running and regenerates whenever a .templ file under
// --dir changes. The watcher polls mod times with a debounce: it works on any
// filesystem (no inotify dependency) and never regenerates mid-save.
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "generate" {
		usage()
		os.Exit(2)
	}

	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	dir := fs.String("dir", ".", "directorio donde escanear los .templ")
	out := fs.String("out", "islands_gen.go", "archivo Go de salida")
	pkg := fs.String("package", "main", "package del archivo generado")
	watch := fs.Bool("watch", false, "regenerar al cambiar los .templ")
	fs.Parse(os.Args[2:])

	if err := run(*dir, *out, *pkg); err != nil {
		fmt.Fprintln(os.Stderr, "templ-islands:", err)
		os.Exit(1)
	}
	if *watch {
		watchLoop(*dir, *out, *pkg)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `templ-islands — genera el registro de islas desde los .templ

Uso:
  templ-islands generate [--dir .] [--out islands_gen.go] [--package main] [--watch]

  --dir       directorio donde escanear los .templ (default ".")
  --out       archivo Go de salida (default "islands_gen.go")
  --package   package del archivo generado (default "main")
  --watch     queda corriendo y regenera cuando un .templ cambia
`)
}

// run escanea y regenera una vez, imprimiendo el resultado.
func run(dir, out, pkg string) error {
	isles, err := scanDir(dir)
	if err != nil {
		return err
	}

	if len(isles) == 0 {
		fmt.Println("no se encontraron islas (busca // @island en los .templ)")
		return nil
	}

	if err := generate(out, pkg, isles); err != nil {
		return err
	}

	total := 0
	for _, i := range isles {
		total += len(i.Fields)
	}
	fmt.Printf("generadas %d islas (%d campos) -> %s\n", len(isles), total, out)
	return nil
}

// templFile es la firma que el watcher compara para detectar cambios.
type templFile struct {
	mtime time.Time
	size  int64
}

// templSnapshot captura (mtime, size) de cada .templ bajo dir.
func templSnapshot(dir string) map[string]templFile {
	out := make(map[string]templFile)
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".templ" {
			return nil
		}
		if info, err := d.Info(); err == nil {
			out[path] = templFile{mtime: info.ModTime(), size: info.Size()}
		}
		return nil
	})
	return out
}

// snapshotsDiffer compara dos snapshots (tambien detecta archivos nuevos y
// borrados: los mapas cambian de longitud).
func snapshotsDiffer(a, b map[string]templFile) bool {
	if len(a) != len(b) {
		return true
	}
	for path, fa := range a {
		fb, ok := b[path]
		if !ok || !fa.mtime.Equal(fb.mtime) || fa.size != fb.size {
			return true
		}
	}
	return false
}

// watchLoop regenera cada vez que cambia un .templ. Si la regeneracion falla
// (p. ej. el editor guardo a medias), mantiene el snapshot viejo y reintenta
// en el proximo tick en vez de matar el proceso.
func watchLoop(dir, out, pkg string) {
	prev := templSnapshot(dir)
	fmt.Println("observando cambios en " + dir + " (Ctrl+C para salir)")
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		cur := templSnapshot(dir)
		if !snapshotsDiffer(prev, cur) {
			continue
		}
		// debounce: esperar a que el editor termine de escribir
		time.Sleep(300 * time.Millisecond)
		if err := run(dir, out, pkg); err != nil {
			fmt.Fprintln(os.Stderr, "templ-islands:", err, "(reintentando...)")
			continue
		}
		prev = cur
	}
}
