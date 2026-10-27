package webui

import (
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

const fallbackHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>v2home · frontend not built</title>
<style>
  :root { color-scheme: dark; }
  body { margin: 0; min-height: 100vh; display: grid; place-items: center; background: #020617; color: #e2e8f0; font-family: ui-sans-serif, system-ui, sans-serif; }
  main { max-width: 34rem; padding: 2rem; }
  h1 { font-size: 1.25rem; margin: 0 0 .5rem; }
  p { color: #94a3b8; line-height: 1.6; }
  code { background: #0f172a; border: 1px solid #1e293b; border-radius: .375rem; padding: .15rem .4rem; font-size: .85em; }
  a { color: #6ee7b7; }
</style>
</head>
<body>
<main>
  <h1>v2home backend is running</h1>
  <p>The Vue bundle is not embedded in this binary yet. Build it with <code>make frontend-build</code> (or <code>make build</code> for the full image), then restart the server.</p>
  <p>The API is already available: <a href="/api/status">/api/status</a> · <a href="/api/healthz">/api/healthz</a></p>
</main>
</body>
</html>
`

// Handler returns an http.Handler that serves the embedded SPA with a
// history-API fallback to index.html.
func Handler() http.Handler {
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeFallback(w)
		})
	}

	files := http.FileServer(http.FS(sub))
	hasIndex := fileExists(sub, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		clean := path.Clean("/" + r.URL.Path)
		name := strings.TrimPrefix(clean, "/")
		if name == "" {
			name = "index.html"
		}

		if fileExists(sub, name) {
			setCacheHeaders(w, name)
			files.ServeHTTP(w, r)
			return
		}

		if !hasIndex {
			writeFallback(w)
			return
		}

		w.Header().Set("Cache-Control", "no-cache")
		r.URL.Path = "/"
		files.ServeHTTP(w, r)
	})
}

func fileExists(fsys fs.FS, name string) bool {
	info, err := fs.Stat(fsys, name)
	return err == nil && !info.IsDir()
}

func setCacheHeaders(w http.ResponseWriter, name string) {
	if strings.HasPrefix(name, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
}

func writeFallback(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, fallbackHTML)
}
