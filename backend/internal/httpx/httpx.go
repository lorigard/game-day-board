// Package httpx holds the HTTP plumbing both services share: structured
// logging, request logging and panic recovery, JSON responses, and a server
// that shuts down gracefully.
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// NewLogger returns a JSON logger whose field names match what Google Cloud
// Logging expects ("severity" and "message"), so each log line shows with the
// right level and text in the console, and can be searched by field (NFR-12).
func NewLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) > 0 {
				return a
			}
			switch a.Key {
			case slog.LevelKey:
				a.Key = "severity"
			case slog.MessageKey:
				a.Key = "message"
			}
			return a
		},
	}))
}

// statusRecorder remembers the status code a handler wrote, for the request log.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// traceKey is the field Cloud Logging uses to group every log line of one request.
const traceKey = "logging.googleapis.com/trace"

type loggerKey struct{}

// Logger returns the request-scoped logger stored by Middleware, or the default logger.
func Logger(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// Middleware wraps a handler with three things every request needs:
//
//   - a logger carrying the request's trace ID, so all lines for one request
//     can be pulled up together in Cloud Logging;
//   - one log line per request with method, path, status, and duration;
//   - recovery from a panic, answering 500 instead of crashing the server.
func Middleware(logger *slog.Logger, projectID string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqLogger := logger
		// Cloud Run adds this header to every request: "TRACE_ID/SPAN_ID;o=1".
		if h := r.Header.Get("X-Cloud-Trace-Context"); h != "" && projectID != "" {
			traceID, _, _ := strings.Cut(h, "/")
			reqLogger = logger.With(traceKey, fmt.Sprintf("projects/%s/traces/%s", projectID, traceID))
		}
		r = r.WithContext(context.WithValue(r.Context(), loggerKey{}, reqLogger))
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		defer func() {
			if p := recover(); p != nil {
				reqLogger.ErrorContext(r.Context(), "panic while handling request", "panic", fmt.Sprint(p), "path", r.URL.Path)
				WriteError(rec, http.StatusInternalServerError, "internal", "Something went wrong on our side.")
			}
			level := slog.LevelInfo
			if rec.status >= 500 {
				level = slog.LevelError
			}
			reqLogger.Log(r.Context(), level, "request",
				"method", r.Method, "path", r.URL.Path, "status", rec.status,
				"durationMs", time.Since(start).Milliseconds())
		}()
		next.ServeHTTP(rec, r)
	})
}

// WriteJSON writes v as JSON with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Default().Error("writing JSON response failed", "error", err.Error())
	}
}

// ErrorBody is the one error shape every endpoint uses (SRS 7.3), so the front
// end needs only one piece of error-handling code.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail is a machine-readable code plus a message a person can read.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteError writes an error in the shared shape.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorBody{Error: ErrorDetail{Code: code, Message: message}})
}

// Serve runs srv until the process receives SIGINT or SIGTERM, then shuts it
// down gracefully (NFR-5): it stops accepting new connections and lets
// requests already in progress finish, waiting at most drainTimeout. Cloud Run
// sends SIGTERM before stopping an instance, and gives it 10 seconds.
func Serve(ctx context.Context, srv *http.Server, logger *slog.Logger, drainTimeout time.Duration) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	return serveUntil(ctx, srv, logger, drainTimeout)
}

// serveUntil is Serve without the signal handling, so a test can trigger the
// shutdown by cancelling ctx.
func serveUntil(ctx context.Context, srv *http.Server, logger *slog.Logger, drainTimeout time.Duration) error {
	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err // the server failed to start, for example because the port is in use
	case <-ctx.Done():
	}

	logger.Info("shutting down, finishing requests in progress")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), drainTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	logger.Info("shutdown complete")
	return nil
}

// NewServer returns an http.Server with timeouts set, so a slow or stuck
// client cannot hold a connection open forever.
func NewServer(port string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + port,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      2 * time.Minute, // long enough for an ingest run
		IdleTimeout:       60 * time.Second,
	}
}

// Getenv returns the environment variable key, or fallback when it is unset or empty.
func Getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
