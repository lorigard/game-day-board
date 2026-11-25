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

// main is where the program starts. It stays tiny on purpose: all the work is
// in run, which returns an error instead of exiting, so it is easy to read
// and every failure is logged the same way before the program stops.
func main() {
	// Cloud Run collects whatever a service prints to standard output and
	// sends it to Cloud Logging, so logging to os.Stdout is all we need.
	logger := httpx.NewLogger(os.Stdout)
	slog.SetDefault(logger)
	if err := run(logger); err != nil {
		logger.Error("ingest stopped with an error", "error", err.Error())
		os.Exit(1)
	}
}

// run wires the pieces together and serves requests until the service is
// told to stop.
func run(logger *slog.Logger) error {
	ctx := context.Background()

	// Fail fast: without a key every run would fail, so refuse to start and
	// make the problem obvious in the logs right away.
	apiKey := os.Getenv("CFBD_API_KEY")
	if apiKey == "" {
		return errors.New("CFBD_API_KEY is not set")
	}
	// The "demo-" default matches the Firestore emulator's test project, so
	// running locally can never touch the real database by accident.
	projectID := httpx.Getenv("GOOGLE_CLOUD_PROJECT", "demo-game-day-board")

	db, err := store.NewFirestore(ctx, projectID)
	if err != nil {
		return err
	}
	defer db.Close()

	runner := &ingest.Runner{
		Source:      cfbd.New(apiKey, cfbd.WithLogger(logger)),
		Store:       db,
		Conferences: splitList(httpx.Getenv("CONFERENCES", "SEC,B12")),
		Now:         time.Now,
		Logger:      logger,
	}

	// Go 1.22 and later let a route name its HTTP method, as in "POST /run".
	// A GET to /run gets an automatic 405 Method Not Allowed.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("POST /run", runHandler(runner))

	handler := httpx.Middleware(logger, projectID, mux)
	return httpx.Serve(ctx, httpx.NewServer(httpx.Getenv("PORT", "8080"), handler), logger, 8*time.Second)
}

// splitList turns "SEC, B12" into ["SEC", "B12"], ignoring spaces and empty
// entries, so a stray space in the setting cannot create a conference named " B12".
func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// runHandler performs one ingest run per request. The mutex makes overlapping
// calls (for example a scheduled run and a manual one) take turns instead of
// writing at the same time.
//
// Go note: the function returned below is a "closure": it can still use mu
// and runner after runHandler has returned. Every request shares that one mu,
// which is what makes them take turns.
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
			// Sending the error text back is safe here only because this
			// service is private: the scheduler is the only caller. A public
			// service should log the details and send a generic message.
			httpx.WriteError(w, http.StatusBadGateway, "ingest_failed", err.Error())
			return
		}
		httpx.Logger(ctx).InfoContext(ctx, "ingest run succeeded",
			"fullRefresh", res.FullRefresh, "gamesSaved", res.GamesSaved, "callsRemaining", res.CallsRemaining)
		httpx.WriteJSON(w, http.StatusOK, res)
	})
}
