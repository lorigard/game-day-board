package cfbd

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lorigard/game-day-board/backend/internal/retry"
)

// newTestClient returns a client pointed at srv that never really sleeps.
// It records every wait it was asked to do in *waits.
func newTestClient(srv *httptest.Server, waits *[]time.Duration, opts ...Option) *Client {
	base := []Option{
		WithBaseURL(srv.URL),
		WithRetryPolicy(retry.Policy{MaxRetries: 3, BaseDelay: time.Second, MaxDelay: 10 * time.Second, Rand: func() float64 { return 0.5 }}),
		WithSleep(func(_ context.Context, d time.Duration) error { *waits = append(*waits, d); return nil }),
		WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
	}
	return New("test-key", append(base, opts...)...)
}

// sequenceServer answers each request with the next status code in codes.
// A 200 answer carries body. It counts requests in *calls.
func sequenceServer(t *testing.T, calls *atomic.Int32, body string, codes ...int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(calls.Add(1)) - 1
		code := codes[min(n, len(codes)-1)]
		w.Header().Set("X-CallLimit-Remaining", "990")
		w.WriteHeader(code)
		if code == http.StatusOK {
			io.WriteString(w, body)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestTeamsSendsKeyAndDecodes(t *testing.T) {
	var gotAuth, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotQuery = r.URL.RawQuery
		w.Header().Set("X-CallLimit-Remaining", "997")
		io.WriteString(w, `[{"id":12,"school":"Arizona","conference":"Big 12","color":"#cc0033","logos":["ignored"]}]`)
	}))
	defer srv.Close()

	var waits []time.Duration
	c := newTestClient(srv, &waits)
	teams, err := c.Teams(context.Background(), "B12")
	if err != nil {
		t.Fatalf("Teams() error = %v", err)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer test-key")
	}
	if gotQuery != "conference=B12" {
		t.Errorf("query = %q, want %q", gotQuery, "conference=B12")
	}
	if len(teams) != 1 || teams[0].School != "Arizona" || teams[0].Conference != "Big 12" {
		t.Errorf("Teams() = %+v, want one team, Arizona of the Big 12", teams)
	}
	if got := c.CallsRemaining(); got != 997 {
		t.Errorf("CallsRemaining() = %d, want 997", got)
	}
}

func TestGamesBuildsQuery(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		io.WriteString(w, `[{"id":1,"homePoints":null,"awayPoints":7,"startDate":"2026-11-27T17:00:00.000Z"}]`)
	}))
	defer srv.Close()

	var waits []time.Duration
	games, err := newTestClient(srv, &waits).Games(context.Background(), GamesQuery{Year: 2026, Week: 13, SeasonType: "regular"})
	if err != nil {
		t.Fatalf("Games() error = %v", err)
	}
	if want := "classification=fbs&seasonType=regular&week=13&year=2026"; gotQuery != want {
		t.Errorf("query = %q, want %q", gotQuery, want)
	}
	if games[0].HomePoints != nil || games[0].AwayPoints == nil || *games[0].AwayPoints != 7 {
		t.Errorf("points = %v / %v, want nil / 7", games[0].HomePoints, games[0].AwayPoints)
	}
}

func TestRetriesServerErrorsThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	srv := sequenceServer(t, &calls, `[]`, 500, 503, 200)
	var waits []time.Duration
	if _, err := newTestClient(srv, &waits).Calendar(context.Background(), 2026); err != nil {
		t.Fatalf("Calendar() error = %v, want success on the third attempt", err)
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3", calls.Load())
	}
	// With Rand fixed at 0.5: half of 1s, then half of 2s.
	want := []time.Duration{500 * time.Millisecond, time.Second}
	if len(waits) != 2 || waits[0] != want[0] || waits[1] != want[1] {
		t.Errorf("waits = %v, want %v", waits, want)
	}
}

func TestDoesNotRetryClientErrors(t *testing.T) {
	var calls atomic.Int32
	srv := sequenceServer(t, &calls, ``, 404)
	var waits []time.Duration
	_, err := newTestClient(srv, &waits).Teams(context.Background(), "NOPE")

	var se *StatusError
	if !errors.As(err, &se) || se.StatusCode != 404 {
		t.Fatalf("Teams() error = %v, want a StatusError with code 404", err)
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1: a 404 must not be retried", calls.Load())
	}
}

func TestGivesUpAfterMaxRetries(t *testing.T) {
	var calls atomic.Int32
	srv := sequenceServer(t, &calls, ``, 502)
	var waits []time.Duration
	if _, err := newTestClient(srv, &waits).Calendar(context.Background(), 2026); err == nil {
		t.Fatal("Calendar() error = nil, want an error after every attempt failed")
	}
	if calls.Load() != 4 {
		t.Errorf("calls = %d, want 4 (1 attempt + 3 retries)", calls.Load())
	}
}

func TestRespectsRetryAfterOnTooManyRequests(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "7")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		io.WriteString(w, `[]`)
	}))
	defer srv.Close()

	var waits []time.Duration
	if _, err := newTestClient(srv, &waits).Calendar(context.Background(), 2026); err != nil {
		t.Fatalf("Calendar() error = %v", err)
	}
	if len(waits) != 1 || waits[0] != 7*time.Second {
		t.Errorf("waits = %v, want [7s] from the Retry-After header", waits)
	}
}

func TestSlowAttemptTimesOutAndIsRetried(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			// Hang until the client gives up on this attempt.
			<-r.Context().Done()
			return
		}
		io.WriteString(w, `[]`)
	}))
	defer srv.Close()

	var waits []time.Duration
	c := newTestClient(srv, &waits, WithAttemptTimeout(50*time.Millisecond))
	if _, err := c.Calendar(context.Background(), 2026); err != nil {
		t.Fatalf("Calendar() error = %v, want success after the slow attempt timed out", err)
	}
	if calls.Load() != 2 {
		t.Errorf("calls = %d, want 2", calls.Load())
	}
}

func TestStopsWhenContextIsCancelled(t *testing.T) {
	var calls atomic.Int32
	srv := sequenceServer(t, &calls, ``, 500)
	ctx, cancel := context.WithCancel(context.Background())
	c := New("k", WithBaseURL(srv.URL), WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		WithSleep(func(context.Context, time.Duration) error { cancel(); return context.Canceled }))

	if _, err := c.Calendar(ctx, 2026); !errors.Is(err, context.Canceled) {
		t.Fatalf("Calendar() error = %v, want context.Canceled", err)
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1: no retry after cancellation", calls.Load())
	}
}
