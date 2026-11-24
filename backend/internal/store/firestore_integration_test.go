//go:build integration

// Integration tests run against the Firestore emulator, a local copy of
// Firestore. They are skipped by a plain "go test ./..." and run in CI with:
//
//	npx firebase-tools emulators:exec --only firestore --project demo-game-day-board \
//	  "go test -tags=integration ./..."
//
// The "demo-" project prefix tells the emulator this is not a real project.
package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/lorigard/game-day-board/backend/internal/football"
)

func newEmulatorStore(t *testing.T) *Firestore {
	t.Helper()
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not set; start the Firestore emulator to run this test")
	}
	ctx := context.Background()
	f, err := NewFirestore(ctx, "demo-game-day-board")
	if err != nil {
		t.Fatalf("NewFirestore() error = %v", err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func pts(n int) *int { return &n }

func TestFirestoreGamesRoundTripAndIdempotency(t *testing.T) {
	f := newEmulatorStore(t)
	ctx := context.Background()
	kickoff := time.Date(2026, 11, 28, 17, 0, 0, 0, time.UTC)
	game := football.Game{
		ID: 9001, Season: 2026, Week: 14, SeasonType: football.SeasonRegular, StartTime: kickoff,
		Home: football.Side{TeamID: 2305, Name: "Kansas", Conference: "Big 12"},
		Away: football.Side{TeamID: 2306, Name: "Kansas State", Conference: "Big 12"},
	}

	// Saving the same game twice (FR-2) must leave one document, with the newer data.
	if err := f.SaveGames(ctx, []football.Game{game}); err != nil {
		t.Fatalf("first SaveGames() error = %v", err)
	}
	game.Completed, game.Home.Points, game.Away.Points = true, pts(27), pts(24)
	if err := f.SaveGames(ctx, []football.Game{game}); err != nil {
		t.Fatalf("second SaveGames() error = %v", err)
	}

	week, err := f.GamesForWeek(ctx, 2026, 14, football.SeasonRegular)
	if err != nil {
		t.Fatalf("GamesForWeek() error = %v", err)
	}
	var found []football.Game
	for _, g := range week {
		if g.ID == 9001 {
			found = append(found, g)
		}
	}
	if len(found) != 1 {
		t.Fatalf("found %d copies of game 9001, want exactly 1", len(found))
	}
	if !found[0].Completed || found[0].Home.Points == nil || *found[0].Home.Points != 27 {
		t.Errorf("stored game = %+v, want the second, completed version", found[0])
	}
	if !found[0].StartTime.Equal(kickoff) {
		t.Errorf("StartTime = %v, want %v", found[0].StartTime, kickoff)
	}

	byTeam, err := f.GamesForTeam(ctx, 2026, 2306)
	if err != nil || len(byTeam) != 1 {
		t.Errorf("GamesForTeam() = %d games, %v; want 1 game", len(byTeam), err)
	}
}

func TestFirestoreTeamNotFound(t *testing.T) {
	f := newEmulatorStore(t)
	if _, err := f.Team(context.Background(), 999999); err != ErrNotFound {
		t.Errorf("Team() error = %v, want ErrNotFound", err)
	}
}

func TestFirestoreCalendarAndStatus(t *testing.T) {
	f := newEmulatorStore(t)
	ctx := context.Background()
	weeks := []football.Week{{Season: 2026, Week: 14, SeasonType: football.SeasonRegular,
		Start: time.Date(2026, 11, 30, 8, 0, 0, 0, time.UTC), End: time.Date(2026, 12, 7, 8, 0, 0, 0, time.UTC)}}
	if err := f.SaveCalendar(ctx, weeks); err != nil {
		t.Fatalf("SaveCalendar() error = %v", err)
	}
	got, err := f.Calendar(ctx)
	if err != nil || len(got) != 1 || got[0].Week != 14 {
		t.Errorf("Calendar() = %+v, %v; want week 14", got, err)
	}

	s := football.IngestStatus{LastSuccessAt: time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC), GamesSaved: 31, CallsRemaining: 990}
	if err := f.SaveIngestStatus(ctx, s); err != nil {
		t.Fatalf("SaveIngestStatus() error = %v", err)
	}
	gotStatus, err := f.IngestStatus(ctx)
	if err != nil || gotStatus.GamesSaved != 31 || !gotStatus.LastSuccessAt.Equal(s.LastSuccessAt) {
		t.Errorf("IngestStatus() = %+v, %v; want %+v", gotStatus, err, s)
	}
	if err := f.Ping(ctx); err != nil {
		t.Errorf("Ping() error = %v, want nil", err)
	}
}
