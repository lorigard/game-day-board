// Package api serves the site's own back end: the read-only endpoints the
// React front end calls. It reads only from our store, never from
// CollegeFootballData, so visitors are never slowed down or affected by the
// outside API (design decision D2).
//
// It is the back end of this one site, not a public data service:
// CollegeFootballData's terms forbid running a mirror or substitute API of
// its data, so it sends no CORS headers and is not documented for others.
package api

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/lorigard/game-day-board/backend/internal/football"
	"github.com/lorigard/game-day-board/backend/internal/httpx"
	"github.com/lorigard/game-day-board/backend/internal/store"
)

// storeTimeout bounds every database call made for one request (NFR-2).
const storeTimeout = 5 * time.Second

// Server holds what the handlers need. Now is a field so tests control the clock.
type Server struct {
	Store store.Store
	Now   func() time.Time
}

// Envelope wraps every successful response with the time the data was last
// refreshed, so the page can always say how old it is (FR-9).
type Envelope struct {
	Data        any        `json:"data"`
	LastUpdated *time.Time `json:"lastUpdated"`
}

// GameView is a game as the front end sees it, with its status worked out.
//
// Go note: football.Game is "embedded" (listed with no field name). Its
// fields are promoted, so a GameView has .ID, .Home, and so on directly, and
// the JSON comes out flat: {"id": ..., "home": ..., "status": "final"}.
// It is Go's way to build on a type without inheritance.
type GameView struct {
	football.Game
	Status string `json:"status"`
}

// Routes returns the router. Health checks live under /api/ rather than at
// the common /healthz, because Cloud Run reserves some paths that end in "z"
// and may answer them itself before they reach the app.
//
// Go 1.22 and later match the method and path
// parameters, such as {id}, without a third-party router (design decision D8).
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/ready", s.ready)
	mux.HandleFunc("GET /api/weeks/current", s.currentWeek)
	mux.HandleFunc("GET /api/games", s.games)
	mux.HandleFunc("GET /api/teams", s.teams)
	mux.HandleFunc("GET /api/teams/{id}", s.team)
	mux.HandleFunc("GET /api/teams/{id}/schedule", s.schedule)
	// A pattern ending in "/" matches everything under it. The router always
	// prefers the most specific pattern, so this catch-all only answers
	// addresses none of the routes above matched: a clear JSON 404.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "No such endpoint.")
	})
	return mux
}

// health answers "is the process alive?" (liveness). It checks nothing else on purpose:
// if it depended on the database, a database blip would make the platform
// restart healthy instances, which cannot help.
func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ready answers "can this instance do useful work?", meaning: can it reach the database.
func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), storeTimeout)
	defer cancel()
	if err := s.Store.Ping(ctx); err != nil {
		httpx.Logger(ctx).WarnContext(ctx, "readiness check failed", "error", err.Error())
		httpx.WriteError(w, http.StatusServiceUnavailable, "not_ready", "The database cannot be reached.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// currentWeek returns the week to show first, plus the whole calendar for the
// previous and next week buttons (FR-7).
//
// Every handler has the same shape: func(w, r). r is the incoming request; w
// is where the response is written. Each one starts by giving the database
// call a deadline (storeTimeout), so a slow database answers with a clear 503
// instead of leaving the visitor's page loading forever.
func (s *Server) currentWeek(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), storeTimeout)
	defer cancel()
	weeks, err := s.Store.Calendar(ctx)
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	current, ok := football.CurrentWeek(weeks, s.Now())
	if !ok {
		httpx.WriteError(w, http.StatusServiceUnavailable, "no_data", "The season calendar has not been loaded yet.")
		return
	}
	// SortFunc takes a compare function that returns a negative number,
	// zero, or a positive number, like C#'s IComparer. Earliest week first.
	slices.SortFunc(weeks, func(a, b football.Week) int { return a.Start.Compare(b.Start) })
	s.respond(w, r, map[string]any{"current": current, "weeks": weeks})
}

// games returns one week's games, optionally for one conference (FR-6, FR-13).
func (s *Server) games(w http.ResponseWriter, r *http.Request) {
	// Validate everything from the address before using it. Anything a
	// visitor can type is untrusted input.
	q := r.URL.Query()
	season, err1 := strconv.Atoi(q.Get("season"))
	week, err2 := strconv.Atoi(q.Get("week"))
	seasonType := q.Get("seasonType")
	if seasonType == "" {
		seasonType = football.SeasonRegular
	}
	if err1 != nil || err2 != nil || (seasonType != football.SeasonRegular && seasonType != football.SeasonPostseason) {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request",
			"Use ?season=2026&week=13, and optionally seasonType=regular or postseason.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), storeTimeout)
	defer cancel()
	games, err := s.Store.GamesForWeek(ctx, season, week, seasonType)
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	// Filtering by conference happens here rather than in the database query,
	// because a game matches if either team is in that conference, and a
	// week has few enough games that this costs nothing. DeleteFunc removes
	// every game for which the function returns true.
	if conf := q.Get("conference"); conf != "" {
		games = slices.DeleteFunc(games, func(g football.Game) bool {
			return !strings.EqualFold(g.Home.Conference, conf) && !strings.EqualFold(g.Away.Conference, conf)
		})
	}
	s.respond(w, r, s.views(games))
}

// teams returns every tracked team, sorted by school name.
func (s *Server) teams(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), storeTimeout)
	defer cancel()
	teams, err := s.Store.Teams(ctx)
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	slices.SortFunc(teams, func(a, b football.Team) int { return strings.Compare(a.School, b.School) })
	s.respond(w, r, teams)
}

// team returns one team (FR-8).
func (s *Server) team(w http.ResponseWriter, r *http.Request) {
	id, ok := teamID(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), storeTimeout)
	defer cancel()
	t, err := s.Store.Team(ctx, id)
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	s.respond(w, r, t)
}

// schedule returns one team's games for a season, oldest first, and its record (FR-5).
func (s *Server) schedule(w http.ResponseWriter, r *http.Request) {
	id, ok := teamID(w, r)
	if !ok {
		return
	}
	season := football.SeasonFor(s.Now())
	if v := r.URL.Query().Get("season"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "bad_request", "season must be a year, such as 2026.")
			return
		}
		season = n
	}

	ctx, cancel := context.WithTimeout(r.Context(), storeTimeout)
	defer cancel()
	games, err := s.Store.GamesForTeam(ctx, season, id)
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	s.respond(w, r, map[string]any{
		"season": season,
		"record": football.RecordFor(id, games),
		"games":  s.views(games),
	})
}

// views sorts games by kickoff and adds each one's status.
func (s *Server) views(games []football.Game) []GameView {
	slices.SortFunc(games, func(a, b football.Game) int {
		if c := a.StartTime.Compare(b.StartTime); c != 0 {
			return c
		}
		return a.ID - b.ID // same kickoff: keep a stable order
	})
	now := s.Now()
	out := make([]GameView, 0, len(games))
	for _, g := range games {
		out = append(out, GameView{Game: g, Status: g.Status(now)})
	}
	return out
}

// respond writes data inside the envelope, adding when the data was last refreshed.
func (s *Server) respond(w http.ResponseWriter, r *http.Request, data any) {
	ctx, cancel := context.WithTimeout(r.Context(), storeTimeout)
	defer cancel()
	env := Envelope{Data: data}
	if st, err := s.Store.IngestStatus(ctx); err == nil && !st.LastSuccessAt.IsZero() {
		env.LastUpdated = &st.LastSuccessAt
	}
	// Browsers may reuse a response for a minute instead of asking again.
	// The data changes at most every 30 minutes, so this costs nothing in
	// freshness and saves database reads. ("public" also allows a shared
	// cache, such as a CDN, to keep a copy if one is ever added.)
	w.Header().Set("Cache-Control", "public, max-age=60")
	httpx.WriteJSON(w, http.StatusOK, env)
}

// storeError turns a database error into the right response: 404 for a
// missing record, 503 for anything else, with the details only in the log.
func (s *Server) storeError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "No such record.")
		return
	}
	httpx.Logger(r.Context()).ErrorContext(r.Context(), "database call failed", "path", r.URL.Path, "error", err.Error())
	httpx.WriteError(w, http.StatusServiceUnavailable, "unavailable", "The data is temporarily unavailable. Please try again.")
}

// teamID reads the {id} path parameter, answering 400 if it is not a number.
func teamID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", "Team ID must be a number.")
		return 0, false
	}
	return id, true
}
