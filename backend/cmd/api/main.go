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

	srv := &api.Server{Store: db, Now: time.Now}
	var handler http.Handler = srv.Routes()
	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		handler = web.Handler(handler, os.DirFS(dir))
	}
	handler = httpx.Middleware(logger, projectID, handler)
	return httpx.Serve(ctx, httpx.NewServer(httpx.Getenv("PORT", "8080"), handler), logger, 8*time.Second)
}
