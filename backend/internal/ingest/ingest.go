// Package ingest copies football data from CollegeFootballData into our store.
//
// Each run does the cheapest thing that keeps the data fresh:
//
//   - A full refresh (calendar, teams, and the whole season's games) when the
//     store is empty or the last full refresh is more than a week old. That is
//     5 calls to the outside API.
//   - Otherwise, only the current week's games: 1 call.
//
// That keeps the project far under the free tier's 1,000 calls a month (NFR-7).
package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lorigard/game-day-board/backend/internal/cfbd"
	"github.com/lorigard/game-day-board/backend/internal/football"
	"github.com/lorigard/game-day-board/backend/internal/store"
)

// FullRefreshEvery is how old the last full refresh may get before the next run does another.
const FullRefreshEvery = 7 * 24 * time.Hour

// Source is the part of the CollegeFootballData client that ingest uses.
// Tests pass a fake that returns canned data.
//
// Defining the interface here, where it is used, rather than in the cfbd
// package is a Go habit: "accept interfaces, return concrete types". The
// ingest package says exactly what it needs and nothing more, and *cfbd.Client
// satisfies it without knowing this interface exists.
type Source interface {
	Games(ctx context.Context, q cfbd.GamesQuery) ([]cfbd.Game, error)
	Teams(ctx context.Context, conference string) ([]cfbd.Team, error)
	Calendar(ctx context.Context, year int) ([]cfbd.Week, error)
	CallsRemaining() int
}

// Runner performs ingest runs. Everything it depends on is a field, set by
// whoever creates it (cmd/ingest/main.go in production, the tests in tests).
// This is dependency injection done by hand, with no framework.
type Runner struct {
	Source      Source
	Store       store.Store
	Conferences []string // CollegeFootballData abbreviations, for example "SEC" and "B12"
	// Now returns the current time. Production passes time.Now; tests pass a
	// function that returns a fixed time, so "which week is it?" is predictable.
	Now    func() time.Time
	Logger *slog.Logger
}

// Result summarizes one run, for the ingest endpoint's response and the logs.
type Result struct {
	FullRefresh    bool `json:"fullRefresh"`
	GamesSaved     int  `json:"gamesSaved"`
	TeamsSaved     int  `json:"teamsSaved"`
	CallsRemaining int  `json:"callsRemaining"`
}

// Run performs one ingest run and records its outcome in the store's ingest
// status. On failure, the data already in the store is left as it was, so the
// site keeps serving the last good copy (NFR-4).
func (r *Runner) Run(ctx context.Context) (Result, error) {
	now := r.Now().UTC()
	status, err := r.Store.IngestStatus(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("reading ingest status: %w", err)
	}
	status.LastAttemptAt = now

	res, runErr := r.run(ctx, now, status)
	res.CallsRemaining = r.Source.CallsRemaining()
	status.CallsRemaining = res.CallsRemaining

	if runErr != nil {
		status.LastError = runErr.Error()
	} else {
		status.LastError = ""
		status.LastSuccessAt = now
		status.GamesSaved = res.GamesSaved
		if res.FullRefresh {
			status.LastFullRefreshAt = now
		}
	}
	// Record the outcome even when the run failed: the error message is what
	// the operator needs to see. Use a fresh context so a run that timed out can
	// still save its status.
	// context.WithoutCancel keeps the request's values (such as its trace ID
	// for the logs) but drops its deadline; WithTimeout then gives the save
	// its own 5 seconds.
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := r.Store.SaveIngestStatus(saveCtx, status); err != nil {
		r.Logger.ErrorContext(ctx, "saving ingest status failed", "error", err.Error())
	}
	return res, runErr
}

// run decides which kind of run this is. A zero time (IsZero) means a full
// refresh has never happened, so the store has no teams or calendar yet.
func (r *Runner) run(ctx context.Context, now time.Time, status football.IngestStatus) (Result, error) {
	if status.LastFullRefreshAt.IsZero() || now.Sub(status.LastFullRefreshAt) >= FullRefreshEvery {
		return r.fullRefresh(ctx, now)
	}
	return r.currentWeek(ctx, now)
}

// fullRefresh fetches the season calendar, the tracked conferences' teams,
// and every regular-season and postseason game, then saves them. Order
// matters: teams come before games, because the team list decides which
// games to keep. It stops at the first failure; whatever was saved before
// that point stays saved, and the next run tries again.
func (r *Runner) fullRefresh(ctx context.Context, now time.Time) (Result, error) {
	season := football.SeasonFor(now)
	res := Result{FullRefresh: true}

	weeks, err := r.Source.Calendar(ctx, season)
	if err != nil {
		return res, fmt.Errorf("fetching calendar: %w", err)
	}
	if err := r.Store.SaveCalendar(ctx, mapWeeks(weeks)); err != nil {
		return res, fmt.Errorf("saving calendar: %w", err)
	}

	var teams []football.Team
	for _, conf := range r.Conferences {
		ts, err := r.Source.Teams(ctx, conf)
		if err != nil {
			return res, fmt.Errorf("fetching teams for %s: %w", conf, err)
		}
		for _, t := range ts {
			teams = append(teams, mapTeam(t, now))
		}
	}
	if err := r.Store.SaveTeams(ctx, teams); err != nil {
		return res, fmt.Errorf("saving teams: %w", err)
	}
	res.TeamsSaved = len(teams)

	tracked := teamIDs(teams)
	for _, seasonType := range []string{football.SeasonRegular, football.SeasonPostseason} {
		games, err := r.Source.Games(ctx, cfbd.GamesQuery{Year: season, SeasonType: seasonType})
		if err != nil {
			return res, fmt.Errorf("fetching %s games: %w", seasonType, err)
		}
		n, err := r.saveTracked(ctx, games, tracked, now)
		res.GamesSaved += n
		if err != nil {
			return res, err
		}
	}
	r.Logger.InfoContext(ctx, "full refresh finished", "season", season, "teams", res.TeamsSaved, "games", res.GamesSaved)
	return res, nil
}

// currentWeek refreshes only this week's games: one outside call. It reads
// the calendar and team list from our own store, which costs no CFBD calls.
func (r *Runner) currentWeek(ctx context.Context, now time.Time) (Result, error) {
	weeks, err := r.Store.Calendar(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("reading calendar: %w", err)
	}
	week, ok := football.CurrentWeek(weeks, now)
	if !ok {
		// No calendar stored yet: a full refresh will fetch one.
		return r.fullRefresh(ctx, now)
	}
	teams, err := r.Store.Teams(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("reading teams: %w", err)
	}

	games, err := r.Source.Games(ctx, cfbd.GamesQuery{Year: week.Season, Week: week.Week, SeasonType: week.SeasonType})
	if err != nil {
		return Result{}, fmt.Errorf("fetching games for week %d: %w", week.Week, err)
	}
	n, err := r.saveTracked(ctx, games, teamIDs(teams), now)
	r.Logger.InfoContext(ctx, "current week refreshed", "season", week.Season, "week", week.Week,
		"seasonType", week.SeasonType, "games", n)
	return Result{GamesSaved: n}, err
}

// saveTracked keeps games that involve at least one tracked team, then saves them.
// It returns how many it saved.
func (r *Runner) saveTracked(ctx context.Context, games []cfbd.Game, tracked map[int]bool, now time.Time) (int, error) {
	var keep []football.Game
	for _, g := range games {
		// Looking up a missing key in a map returns the zero value, false
		// for a bool, so an untracked team simply reads as false here.
		if tracked[g.HomeID] || tracked[g.AwayID] {
			keep = append(keep, mapGame(g, now))
		}
	}
	if err := r.Store.SaveGames(ctx, keep); err != nil {
		return 0, fmt.Errorf("saving games: %w", err)
	}
	return len(keep), nil
}

// teamIDs turns a team list into a set of IDs. Go has no built-in set type;
// a map whose values are all true is the usual stand-in, and checking it is
// much faster than searching the list for every game.
func teamIDs(teams []football.Team) map[int]bool {
	ids := make(map[int]bool, len(teams))
	for _, t := range teams {
		ids[t.ID] = true
	}
	return ids
}

// The map functions translate CollegeFootballData's shape into ours. This is
// the only place that knows both, so an API change is fixed here and nowhere else.

// mapGame converts one CFBD game into ours. CFBD lists home and away fields
// side by side (HomeTeam, AwayTeam, ...); we group each team's fields into a
// Side. Times are stored in UTC; the browser converts them to local time.
func mapGame(g cfbd.Game, now time.Time) football.Game {
	return football.Game{
		ID:             g.ID,
		Season:         g.Season,
		Week:           g.Week,
		SeasonType:     g.SeasonType,
		StartTime:      g.StartDate.UTC(),
		StartTimeTBD:   g.StartTimeTBD,
		Completed:      g.Completed,
		NeutralSite:    g.NeutralSite,
		ConferenceGame: g.ConferenceGame,
		Venue:          g.Venue,
		Home: football.Side{
			TeamID:     g.HomeID,
			Name:       g.HomeTeam,
			Conference: g.HomeConference,
			Points:     g.HomePoints,
		},
		Away: football.Side{
			TeamID:     g.AwayID,
			Name:       g.AwayTeam,
			Conference: g.AwayConference,
			Points:     g.AwayPoints,
		},
		UpdatedAt: now,
	}.WithKeys()
}

// mapTeam converts one CFBD team into ours.
func mapTeam(t cfbd.Team, now time.Time) football.Team {
	return football.Team{
		ID:             t.ID,
		School:         t.School,
		Mascot:         t.Mascot,
		Abbreviation:   t.Abbreviation,
		Conference:     t.Conference,
		Color:          t.Color,
		AlternateColor: t.AlternateColor,
		UpdatedAt:      now,
	}
}

// mapWeeks converts the CFBD season calendar into ours.
func mapWeeks(ws []cfbd.Week) []football.Week {
	out := make([]football.Week, 0, len(ws))
	for _, w := range ws {
		out = append(out, football.Week{
			Season:     w.Season,
			Week:       w.Week,
			SeasonType: w.SeasonType,
			Start:      w.StartDate.UTC(),
			End:        w.EndDate.UTC(),
		})
	}
	return out
}
