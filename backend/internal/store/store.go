// Package store saves and loads the app's own copy of the football data.
//
// Store is an interface so the API and ingest code never depend on Firestore
// directly (NFR-14). Production uses the Firestore implementation; unit tests
// use the in-memory one, which needs no network and runs in microseconds.
package store

import (
	"context"
	"errors"

	"github.com/lorigard/game-day-board/backend/internal/football"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("not found")

// Store is everything the services need from the database.
//
// Go note: an interface lists methods, and any type that has those methods
// satisfies it automatically. There is no "implements" keyword as in C#.
// Firestore (firestore.go) and Memory (memory.go) both satisfy Store, so code
// that takes a Store works with either one.
type Store interface {
	// SaveGames writes games keyed by their ID, so saving the same game twice
	// overwrites it rather than adding a copy (idempotent, FR-2).
	SaveGames(ctx context.Context, games []football.Game) error
	SaveTeams(ctx context.Context, teams []football.Team) error
	SaveCalendar(ctx context.Context, weeks []football.Week) error
	SaveIngestStatus(ctx context.Context, s football.IngestStatus) error

	GamesForWeek(ctx context.Context, season, week int, seasonType string) ([]football.Game, error)
	GamesForTeam(ctx context.Context, season, teamID int) ([]football.Game, error)
	Team(ctx context.Context, id int) (football.Team, error)
	Teams(ctx context.Context) ([]football.Team, error)
	Calendar(ctx context.Context) ([]football.Week, error)
	// IngestStatus returns an empty status, not an error, if ingest has never
	// run. Its CallsRemaining is -1, meaning "unknown", as everywhere else.
	IngestStatus(ctx context.Context) (football.IngestStatus, error)

	// Ping checks that the database can be reached; /api/ready uses it (NFR-1).
	Ping(ctx context.Context) error
}
