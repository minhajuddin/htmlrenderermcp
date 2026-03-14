package main

import (
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"regexp"
	"time"
)

var uuidRegexp = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func SetupHTTPServer(store Storage, addr string, baseURL string) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /render/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !uuidRegexp.MatchString(id) {
			http.Error(w, "invalid render id", http.StatusBadRequest)
			return
		}

		data, err := store.Fetch(r.Context(), id)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				http.Error(w, "render not found", http.StatusNotFound)
				return
			}
			log.Printf("fetch error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	mux.HandleFunc("GET /renders", func(w http.ResponseWriter, r *http.Request) {
		renders, err := store.List(r.Context())
		if err != nil {
			log.Printf("list error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
<html><head>
<meta charset="UTF-8">
<title>Renders</title>
<style>
body { font-family: system-ui, sans-serif; max-width: 800px; margin: 2rem auto; padding: 0 1rem; }
a { color: #0066cc; }
table { width: 100%; border-collapse: collapse; }
th, td { text-align: left; padding: 0.5rem; border-bottom: 1px solid #eee; }
th { font-weight: 600; }
</style>
</head><body>
<h1>Renders</h1>`)

		if len(renders) == 0 {
			fmt.Fprint(w, "<p>No renders yet.</p>")
		} else {
			fmt.Fprint(w, "<table><thead><tr><th>Title</th><th>Created</th><th>Link</th></tr></thead><tbody>")
			for _, r := range renders {
				title := r.Title
				if title == "" {
					title = r.ID
				}
				url := fmt.Sprintf("%s/render/%s", baseURL, r.ID)
				fmt.Fprintf(w, "<tr><td>%s</td><td>%s</td><td><a href=\"%s\">Open</a></td></tr>",
					html.EscapeString(title),
					r.CreatedAt.Format(time.RFC3339),
					html.EscapeString(url),
				)
			}
			fmt.Fprint(w, "</tbody></table>")
		}

		fmt.Fprint(w, "</body></html>")
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}
