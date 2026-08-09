package islands_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/SalvucciFacundo/templ-islands/islands"
)

// ExampleRegistry_Register shows the minimal flow: declare an optimistic
// mutation island and read the manifest that describes it to the client
// runtime.
func ExampleRegistry_Register() {
	reg := islands.New()
	reg.Register("like",
		[]islands.Field{
			{Name: "likes", Op: islands.OpInc, Selector: "[data-mutate=likes]", Delta: 1},
			{Name: "liked", Op: islands.OpToggleText, Selector: "[data-mutate=label]", TrueText: "Liked", FalseText: "Like"},
		},
		"/api/like/{post_id}", "POST")

	// RuntimeHandler serves the client JS and the JSON manifest.
	srv := httptest.NewServer(reg.RuntimeHandler())
	defer srv.Close()

	res, err := http.Get(srv.URL + "/manifest.json")
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	var manifest map[string]any
	if err := json.NewDecoder(res.Body).Decode(&manifest); err != nil {
		panic(err)
	}

	like := manifest["like"].(map[string]any)
	fmt.Println(like["method"], like["endpoint"])
	// Output:
	// POST /api/like/{post_id}
}
