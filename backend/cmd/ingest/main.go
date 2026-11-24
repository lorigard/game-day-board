// Command ingest copies football data from CollegeFootballData into Firestore.
//
// It runs as a private Cloud Run service. Cloud Scheduler calls POST /run on a
// timer, signing each call with its own identity; Cloud Run itself refuses any
// call without that signature, so this code does not check it (NFR-9).
//
// Settings come from environment variables:
//
//	PORT                  port to listen on (Cloud Run sets it; default 8080)
//	GOOGLE_CLOUD_PROJECT  Google Cloud project that holds the Firestore database
//	CFBD_API_KEY          CollegeFootballData key (from Secret Manager in the cloud, .env locally)
//	CONFERENCES           comma-separated conference abbreviations (default "SEC,B12")
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/lorigard/game-day-board/backend/internal/cfbd"
	"github.com/lorigard/game-day-board/backend/internal/httpx"
	"github.com/lorigard/game-day-board/backend/internal/ingest"
	"github.com/lorigard/game-day-board/backend/internal/store"
)

// runTimeout bounds one ingest run, so a stuck run cannot pile up behind the next one.
const runTimeout = 90 * time.Second

func main() {
	logger := httpx.NewLogger(os.Stdout)
	slog.SetDefault(logger)
	if err := run(logger); err != nil {
		logger.Error("ingest stopped with an error", "error", err.Error())
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx := context.Background()
	apiKey := os.Getenv("CFBD_API_KEY")
	if apiKey == "" {
		return errors.New("CFBD_API_KEY is not set")
	}
	projectID := httpx.Getenv("GOOGLE_CLOUD_PROJECT", "demo-game-day-board")

	db, err := store.NewFirestore(ctx, projectID)
	if err != nil {
		return err
	}
	defer db.Close()

	runner := &ingest.Runner{
		Source:      cfbd.New(apiKey, cfbd.WithLogger(logger)),
		Store:       db,
		Conferences: strings.Split(httpx.Getenv("CONFERENCES", "SEC,B12"), ","),
		Now:         time.Now,
		Logger:      logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("POST /run", runHandler(runner))

	handler := httpx.Middleware(logger, projectID, mux)
	return httpx.Serve(ctx, httpx.NewServer(httpx.Getenv("PORT", "8080"), handler), logger, 8*time.Second)
}

// runHandler performs one ingest run per request. The mutex makes overlapping
// calls (for example a scheduled run and a manual one) take turns instead of
// writing at the same time.
func runHandler(runner *ingest.Runner) http.Handler {
	var mu sync.Mutex
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		ctx, cancel := context.WithTimeout(r.Context(), runTimeout)
		defer cancel()
		res, err := runner.Run(ctx)
		if err != nil {
			httpx.Logger(ctx).ErrorContext(ctx, "ingest run failed", "error", err.Error(), "callsRemaining", res.CallsRemaining)
			// 502: we are fine, but the data source we depend on failed.
			// Cloud Scheduler sees a failure and records it in the job's history.
			httpx.WriteError(w, http.StatusBadGateway, "ingest_failed", err.Error())
			return
		}
		httpx.Logger(ctx).InfoContext(ctx, "ingest run succeeded",
			"fullRefresh", res.FullRefresh, "gamesSaved", res.GamesSaved, "callsRemaining", res.CallsRemaining)
		httpx.WriteJSON(w, http.StatusOK, res)
	})
}
