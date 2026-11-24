// Package cfbd is a small client for the CollegeFootballData.com API.
//
// Every call to the outside API goes through this one client, so timeouts,
// retries, logging, and the monthly quota count live in one place (SRS 7.2).
package cfbd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/lorigard/game-day-board/backend/internal/retry"
)

// DefaultBaseURL is the production address of the API.
const DefaultBaseURL = "https://api.collegefootballdata.com"

// Client calls the CollegeFootballData API. Create one with New.
type Client struct {
	baseURL        string
	apiKey         string
	httpClient     *http.Client
	attemptTimeout time.Duration
	policy         retry.Policy
	sleep          func(context.Context, time.Duration) error
	logger         *slog.Logger

	// callsRemaining is the last X-CallLimit-Remaining value seen; -1 if none yet.
	// It is atomic because one client may serve several goroutines.
	callsRemaining atomic.Int64
}

// Option changes one setting of a Client. Options keep New's signature short
// while letting tests swap in a fake server, a fake clock, or a quieter logger.
type Option func(*Client)

// WithBaseURL points the client at a different server, such as a test server.
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = u } }

// WithRetryPolicy replaces the default retry policy.
func WithRetryPolicy(p retry.Policy) Option { return func(c *Client) { c.policy = p } }

// WithSleep replaces the function used to wait between retries, so tests run instantly.
func WithSleep(f func(context.Context, time.Duration) error) Option {
	return func(c *Client) { c.sleep = f }
}

// WithAttemptTimeout sets how long a single attempt may take (default 10 seconds, NFR-2).
func WithAttemptTimeout(d time.Duration) Option { return func(c *Client) { c.attemptTimeout = d } }

// WithLogger sets the logger used for retry warnings.
func WithLogger(l *slog.Logger) Option { return func(c *Client) { c.logger = l } }

// New returns a client that authenticates with apiKey.
func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		baseURL:        DefaultBaseURL,
		apiKey:         apiKey,
		httpClient:     &http.Client{},
		attemptTimeout: 10 * time.Second,
		policy:         retry.Default(),
		sleep:          retry.Sleep,
		logger:         slog.Default(),
	}
	c.callsRemaining.Store(-1)
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// CallsRemaining returns the monthly calls left, as last reported by the API, or -1 if unknown.
func (c *Client) CallsRemaining() int { return int(c.callsRemaining.Load()) }

// StatusError is returned when the API answers with an HTTP error status.
type StatusError struct {
	StatusCode int
	Body       string
	retryAfter time.Duration
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("cfbd: HTTP %d: %s", e.StatusCode, e.Body)
}

// retryable reports whether an error is worth another attempt (NFR-3): timeouts
// and network failures, 429 (too many requests), and 5xx (server errors).
// Any other 4xx means our request was wrong, and repeating it cannot help.
func retryable(err error) bool {
	var se *StatusError
	if errors.As(err, &se) {
		return se.StatusCode == http.StatusTooManyRequests || se.StatusCode >= 500
	}
	return true // network error or per-attempt timeout
}

// get calls path with query parameters and decodes the JSON response into out,
// retrying temporary failures according to the client's policy.
func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var lastErr error
	for attempt := 0; attempt <= c.policy.MaxRetries; attempt++ {
		if attempt > 0 {
			wait := c.policy.Delay(attempt)
			var se *StatusError
			if errors.As(lastErr, &se) && se.retryAfter > wait {
				wait = se.retryAfter // the server asked us to wait longer; respect that
			}
			c.logger.WarnContext(ctx, "retrying CollegeFootballData call",
				"path", path, "attempt", attempt+1, "wait", wait.String(), "error", lastErr.Error())
			if err := c.sleep(ctx, wait); err != nil {
				return fmt.Errorf("cfbd: %s: gave up waiting to retry: %w (last error: %v)", path, err, lastErr)
			}
		}

		lastErr = c.attempt(ctx, u, out)
		if lastErr == nil {
			return nil
		}
		if ctx.Err() != nil || !retryable(lastErr) {
			break
		}
	}
	return fmt.Errorf("cfbd: %s: %w", path, lastErr)
}

// attempt makes exactly one request, bounded by the per-attempt timeout.
func (c *Client) attempt(ctx context.Context, u string, out any) error {
	ctx, cancel := context.WithTimeout(ctx, c.attemptTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if v, err := strconv.Atoi(resp.Header.Get("X-CallLimit-Remaining")); err == nil {
		c.callsRemaining.Store(int64(v))
	}

	if resp.StatusCode != http.StatusOK {
		// Keep only the start of an error body: enough to explain it in a log line.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		se := &StatusError{StatusCode: resp.StatusCode, Body: string(body)}
		if secs, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil {
			se.retryAfter = time.Duration(secs) * time.Second
		}
		return se
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}
