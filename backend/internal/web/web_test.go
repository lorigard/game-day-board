package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// files stands in for the built React app.
var files = fstest.MapFS{
	"index.html":           {Data: []byte("<html>app</html>")},
	"favicon.svg":          {Data: []byte("<svg/>")},
	"assets/index-ab12.js": {Data: []byte("console.log(1)")},
}

var api = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "api:"+r.URL.Path)
})

func serve(path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	Handler(api, files).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestAPIRequestsGoToTheAPI(t *testing.T) {
	if got := serve("/api/games").Body.String(); got != "api:/api/games" {
		t.Errorf("body = %q, want the API handler's answer", got)
	}
}

func TestPagesWithoutAFileGetIndexHTML(t *testing.T) {
	for _, p := range []string{"/", "/teams/2305", "/about"} {
		rec := serve(p)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "app") {
			t.Errorf("GET %s = %d %q, want index.html", p, rec.Code, rec.Body.String())
		}
		if rec.Header().Get("Cache-Control") != "no-cache" {
			t.Errorf("GET %s Cache-Control = %q, want no-cache", p, rec.Header().Get("Cache-Control"))
		}
	}
}

func TestHashedAssetsAreCachedForAYear(t *testing.T) {
	rec := serve("/assets/index-ab12.js")
	if rec.Code != http.StatusOK || rec.Body.String() != "console.log(1)" {
		t.Fatalf("GET asset = %d %q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
		t.Errorf("Cache-Control = %q, want long-lived immutable caching", got)
	}
}

func TestOtherFilesAreServed(t *testing.T) {
	rec := serve("/favicon.svg")
	if rec.Body.String() != "<svg/>" || strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("GET favicon = %q with Cache-Control %q", rec.Body.String(), rec.Header().Get("Cache-Control"))
	}
}

func TestPathsCannotEscapeTheFolder(t *testing.T) {
	if strings.Contains(serve("/../../etc/passwd").Body.String(), "root:") {
		t.Error("a path with .. reached outside the app's files")
	}
}
