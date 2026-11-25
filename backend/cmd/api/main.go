// Command api serves the Game Day Board site: the read-only API under /api/,
// and, when STATIC_DIR is set, the built React app for every other path.
//
// Settings come from environment variables, never from files in the
// repository:
//
//	PORT                  port to listen on (Cloud Run sets it; default 8080)
//	GOOGLE_CLOUD_PROJECT  Google Cloud project that holds the Firestore database
//	STATIC_DIR            folder holding the built React app (the container sets /static)
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/lorigard/game-day-board/backend/internal/api"
	"github.com/lorigard/game-day-board/backend/internal/httpx"
	"github.com/lorigard/game-day-board/backend/internal/store"
	"github.com/lorigard/game-day-board/backend/internal/web"
)

func main() {
	logger := httpx.NewLogger(os.Stdout)
	slog.SetDefault(logger)
	if err := run(logger); err != nil {
		logger.Error("api stopped with an error", "error", err.Error())
		os.Exit(1)
	}
}

// run holds the real work, so that deferred cleanup (closing the database
// client) runs before main calls os.Exit, which would skip it.
func run(logger *slog.Logger) error {
	ctx := context.Background()
	projectID := httpx.Getenv("GOOGLE_CLOUD_PROJECT", "demo-game-day-board")

	db, err := store.NewFirestore(ctx, projectID)
	if err != nil {
		return err
	}
	defer db.Close()

	// The production pieces are chosen here and nowhere else: the real
	// database and the real clock. Tests build the same api.Server with an
	// in-memory store and a fixed clock instead.
	srv := &api.Server{Store: db, Now: time.Now}
	var handler http.Handler = srv.Routes()
	// With STATIC_DIR set (as it is in the container), this same service also
	// serves the React pages. Without it (for example when running the API alone
	// on your machine while Vite serves the pages), only /api/ answers. Why one
	// service instead of a CDN: docs/ARCHITECTURE.md, "Decision: one service for
	// the pages and the API".
	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		handler = web.Handler(handler, os.DirFS(dir))
	}
	// Handlers are wrapped like layers of an onion. A request passes through
	// the outermost layer first: Middleware (logging, trace ID, panic
	// recovery), then web.Handler (pages or API), then the API routes. That
	// way every request is logged, whether it is for a page or for data.
	handler = httpx.Middleware(logger, projectID, handler)
	// 8 seconds to finish requests in progress after SIGTERM: under the
	// 10 seconds Cloud Run allows before it stops the instance for good.
	return httpx.Serve(ctx, httpx.NewServer(httpx.Getenv("PORT", "8080"), handler), logger, 8*time.Second)
}
