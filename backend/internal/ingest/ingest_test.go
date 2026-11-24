package ingest

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/lorigard/game-day-board/backend/internal/cfbd"
	"github.com/lorigard/game-day-board/backend/internal/football"
	"github.com/lorigard/game-day-board/backend/internal/store"
)

// fakeSource returns canned data and counts calls, standing in for CollegeFootballData.
type fakeSource struct {
	calls    int
	gamesErr error
	lastQ    cfbd.GamesQuery
}

const (
	kansas      = 2305 // Big 12
	kansasState = 2306 // Big 12
	georgia     = 61   // SEC
	untracked   = 999  // a team outside both conferences
)

func (f *fakeSource) Calendar(context.Context, int) ([]cfbd.Week, error) {
	f.calls++
	return []cfbd.Week{{Season: 2026, Week: 14, SeasonType: "regular",
		StartDate: time.Date(2026, 11, 30, 8, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 12, 7, 8, 0, 0, 0, time.UTC)}}, nil
}

func (f *fakeSource) Teams(_ context.Context, conf string) ([]cfbd.Team, error) {
	f.calls++
	if conf == "B12" {
		return []cfbd.Team{{ID: kansas, School: "Kansas", Conference: "Big 12"}, {ID: kansasState, School: "Kansas State", Conference: "Big 12"}}, nil
	}
	return []cfbd.Team{{ID: georgia, School: "Georgia", Conference: "SEC"}}, nil
}

func (f *fakeSource) Games(_ context.Context, q cfbd.GamesQuery) ([]cfbd.Game, error) {
	f.calls++
	f.lastQ = q
	if f.gamesErr != nil {
		return nil, f.gamesErr
	}
	if q.SeasonType == "postseason" {
		return nil, nil
	}
	return []cfbd.Game{
		{ID: 1, Season: 2026, Week: 14, SeasonType: "regular", HomeID: kansas, AwayID: kansasState},
		{ID: 2, Season: 2026, Week: 14, SeasonType: "regular", HomeID: untracked, AwayID: georgia},
		{ID: 3, Season: 2026, Week: 14, SeasonType: "regular", HomeID: untracked, AwayID: untracked + 1}, // dropped
	}, nil
}

func (f *fakeSource) CallsRemaining() int { return 950 }

func newRunner(src *fakeSource, st store.Store, now time.Time) *Runner {
	return &Runner{
		Source: src, Store: st, Conferences: []string{"SEC", "B12"},
		Now:    func() time.Time { return now },
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

var monday = time.Date(2026, 12, 1, 15, 0, 0, 0, time.UTC)

func TestFirstRunDoesFullRefreshAndKeepsTrackedGames(t *testing.T) {
	src, st := &fakeSource{}, store.NewMemory()
	res, err := newRunner(src, st, monday).Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !res.FullRefresh {
		t.Error("FullRefresh = false, want true on the first run")
	}
	if src.calls != 5 {
		t.Errorf("outside API calls = %d, want 5 (calendar, 2 conferences, regular and postseason games)", src.calls)
	}
	if res.GamesSaved != 2 || st.GameCount() != 2 {
		t.Errorf("games saved = %d (store has %d), want 2: the game between two untracked teams is dropped", res.GamesSaved, st.GameCount())
	}
	status, _ := st.IngestStatus(context.Background())
	if !status.LastSuccessAt.Equal(monday) || !status.LastFullRefreshAt.Equal(monday) || status.CallsRemaining != 950 {
		t.Errorf("status = %+v, want success and full refresh at %v, 950 calls left", status, monday)
	}
}

func TestLaterRunFetchesOnlyTheCurrentWeek(t *testing.T) {
	st := store.NewMemory()
	if _, err := newRunner(&fakeSource{}, st, monday).Run(context.Background()); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}

	src := &fakeSource{}
	res, err := newRunner(src, st, monday.Add(6*time.Hour)).Run(context.Background())
	if err != nil {
		t.Fatalf("second Run() error = %v", err)
	}
	if res.FullRefresh || src.calls != 1 {
		t.Errorf("FullRefresh = %v with %d calls, want a 1-call current-week run", res.FullRefresh, src.calls)
	}
	if want := (cfbd.GamesQuery{Year: 2026, Week: 14, SeasonType: "regular"}); src.lastQ != want {
		t.Errorf("games query = %+v, want %+v", src.lastQ, want)
	}
	// Idempotency (FR-2): the same games again must not add copies.
	if st.GameCount() != 2 {
		t.Errorf("store has %d games after a repeat run, want 2", st.GameCount())
	}
}

func TestFullRefreshAgainAfterAWeek(t *testing.T) {
	st := store.NewMemory()
	newRunner(&fakeSource{}, st, monday).Run(context.Background())
	src := &fakeSource{}
	res, _ := newRunner(src, st, monday.Add(FullRefreshEvery)).Run(context.Background())
	if !res.FullRefresh {
		t.Error("FullRefresh = false, want true once the last full refresh is a week old")
	}
}

func TestFailureKeepsOldDataAndRecordsTheError(t *testing.T) {
	st := store.NewMemory()
	newRunner(&fakeSource{}, st, monday).Run(context.Background())

	later := monday.Add(6 * time.Hour)
	_, err := newRunner(&fakeSource{gamesErr: errors.New("upstream down")}, st, later).Run(context.Background())
	if err == nil {
		t.Fatal("Run() error = nil, want the upstream failure")
	}
	if st.GameCount() != 2 {
		t.Errorf("store has %d games, want the 2 from the last good run (NFR-4)", st.GameCount())
	}
	status, _ := st.IngestStatus(context.Background())
	if status.LastError == "" || !status.LastAttemptAt.Equal(later) || !status.LastSuccessAt.Equal(monday) {
		t.Errorf("status = %+v, want the error recorded, the attempt at %v, and the last success still %v", status, later, monday)
	}
}

func TestMapGameKeepsMissingScoresMissing(t *testing.T) {
	seven := 7
	g := mapGame(cfbd.Game{ID: 5, Season: 2026, HomeID: kansas, AwayID: georgia, AwayPoints: &seven}, monday)
	if g.Home.Points != nil || g.Away.Points == nil || *g.Away.Points != 7 {
		t.Errorf("points = %v / %v, want nil / 7", g.Home.Points, g.Away.Points)
	}
	if len(g.TeamSeasonKeys) != 2 || g.TeamSeasonKeys[0] != football.TeamSeasonKey(2026, kansas) {
		t.Errorf("TeamSeasonKeys = %v, want keys for both teams", g.TeamSeasonKeys)
	}
}
