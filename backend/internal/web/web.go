// Package web serves the built React app (the files Vite puts in dist/)
// alongside the API, so one Cloud Run service hosts the whole site.
//
// It is a "single-page app" server: the browser loads index.html once, and
// React Router then changes pages without asking the server. So a request for
// a page such as /teams/2305 has no matching file, and the server answers with
// index.html; React reads the address and shows the right page.
//
// Design note: serving the pages and the API from one service is a deliberate
// choice for a small site. The common alternative is to put the pages on a CDN
// (such as Firebase Hosting) and keep only the API here. The tradeoff, and when
// we would switch, is in docs/ARCHITECTURE.md under "Decision: one service for
// the pages and the API".
package web

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// Handler routes /api/ requests to api, and everything else to the app's files in dir.
func Handler(api http.Handler, files fs.FS) http.Handler {
	fileServer := http.FileServerFS(files)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			api.ServeHTTP(w, r)
			return
		}

		// path.Clean removes tricks like "/../" so a request can never reach
		// outside the site's folder. (fs.FS refuses such names too; this is a
		// second layer of the same protection.)
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "" || !exists(files, name) {
			serveIndex(w, r, files)
			return
		}

		// Vite puts a hash of each file's contents in its name under assets/,
		// so a changed file always gets a new name. That makes it safe for
		// browsers to keep these files for a year without asking again.
		if strings.HasPrefix(name, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		fileServer.ServeHTTP(w, r)
	})
}

// serveIndex sends index.html, telling browsers to check for a new version on
// every visit, so a deploy shows up straight away.
func serveIndex(w http.ResponseWriter, r *http.Request, files fs.FS) {
	data, err := fs.ReadFile(files, "index.html")
	if err != nil {
		http.Error(w, "The site files are missing.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	// A HEAD request asks for the headers only (uptime checkers use it), so
	// the body is skipped. A write error means the visitor already went away,
	// and there is no one left to tell, so it is ignored on purpose ("_, _ =").
	if r.Method != http.MethodHead {
		_, _ = w.Write(data)
	}
}

// exists reports whether name is a regular file (not a directory) in files.
func exists(files fs.FS, name string) bool {
	info, err := fs.Stat(files, name)
	return err == nil && !info.IsDir()
}
