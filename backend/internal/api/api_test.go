package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lorigard/game-day-board/backend/internal/football"
	"github.com/lorigard/game-day-board/backend/internal/store"
)

var now = time.Date(2026, 11, 28, 18, 0, 0, 0, time.UTC)

func pts(n int) *int { return &n }

// seeded returns a server over an in-memory store holding a small week of data.
func seeded(t *testing.T) (*Server, *store.Memory) {
	t.Helper()
	st := store.NewMemory()
	ctx := context.Background()
	st.SaveTeams(ctx, []football.Team{
		{ID: 2305, School: "Kansas", Conference: "Big 12"},
		{ID: 61, School: "Georgia", Conference: "SEC"},
	})
	st.SaveGames(ctx, []football.Game{
		{ID: 2, Season: 2026, Week: 14, SeasonType: "regular", StartTime: now.Add(time.Hour),
			Home: football.Side{TeamID: 61, Name: "Georgia", Conference: "SEC"}, Away: football.Side{TeamID: 9, Name: "Georgia Tech", Conference: "ACC"}},
		{ID: 1, Season: 2026, Week: 14, SeasonType: "regular", StartTime: now.Add(-4 * time.Hour), Completed: true,
			Home: football.Side{TeamID: 2305, Name: "Kansas", Conference: "Big 12", Points: pts(28)},
			Away: football.Side{TeamID: 2306, Name: "Kansas State", Conference: "Big 12", Points: pts(21)}},
		{ID: 3, Season: 2026, Week: 13, SeasonType: "regular", StartTime: now.Add(-7 * 24 * time.Hour), Completed: true,
			Home: football.Side{TeamID: 12, Name: "Arizona", Conference: "Big 12", Points: pts(30)},
			Away: football.Side{TeamID: 2305, Name: "Kansas", Conference: "Big 12", Points: pts(10)}},
	})
	st.SaveCalendar(ctx, []football.Week{
		{Season: 2026, Week: 14, SeasonType: "regular", Start: now.Add(-48 * time.Hour), End: now.Add(5 * 24 * time.Hour)},
		{Season: 2026, Week: 13, SeasonType: "regular", Start: now.Add(-9 * 24 * time.Hour), End: now.Add(-48 * time.Hour)},
	})
	st.SaveIngestStatus(ctx, football.IngestStatus{LastSuccessAt: now.Add(-10 * time.Minute)})
	return &Server{Store: st, Now: func() time.Time { return now }}, st
}

// get calls the server and decodes the JSON body into out.
func get(t *testing.T, s *Server, path string, out any) int {
	t.Helper()
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if out != nil {
		if err := json.NewDecoder(rec.Body).Decode(out); err != nil {
			t.Fatalf("GET %s: body is not JSON: %v", path, err)
		}
	}
	return rec.Code
}

func TestGamesForWeekSortedWithStatusAndLastUpdated(t *testing.T) {
	s, _ := seeded(t)
	var body struct {
		Data        []GameView `json:"data"`
		LastUpdated time.Time  `json:"lastUpdated"`
	}
	if code := get(t, s, "/api/games?season=2026&week=14", &body); code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	if len(body.Data) != 2 || body.Data[0].ID != 1 || body.Data[1].ID != 2 {
		t.Fatalf("games = %+v, want IDs 1 then 2 (by kickoff)", body.Data)
	}
	if body.Data[0].Status != football.StatusFinal || body.Data[1].Status != football.StatusScheduled {
		t.Errorf("statuses = %q, %q; want final, scheduled", body.Data[0].Status, body.Data[1].Status)
	}
	if !body.LastUpdated.Equal(now.Add(-10 * time.Minute)) {
		t.Errorf("lastUpdated = %v, want %v", body.LastUpdated, now.Add(-10*time.Minute))
	}
}

func TestGamesFilteredByConference(t *testing.T) {
	s, _ := seeded(t)
	var body struct{ Data []GameView }
	get(t, s, "/api/games?season=2026&week=14&conference=sec", &body)
	if len(body.Data) != 1 || body.Data[0].ID != 2 {
		t.Errorf("games = %+v, want only game 2 (Georgia, SEC)", body.Data)
	}
}

func TestGamesRejectsBadQuery(t *testing.T) {
	s, _ := seeded(t)
	for _, path := range []string{"/api/games", "/api/games?season=x&week=1", "/api/games?season=2026&week=1&seasonType=spring"} {
		var body struct{ Error struct{ Code string } }
		if code := get(t, s, path, &body); code != http.StatusBadRequest || body.Error.Code != "bad_request" {
			t.Errorf("GET %s = %d %q, want 400 bad_request", path, code, body.Error.Code)
		}
	}
}

func TestCurrentWeek(t *testing.T) {
	s, _ := seeded(t)
	var body struct {
		Data struct {
			Current football.Week   `json:"current"`
			Weeks   []football.Week `json:"weeks"`
		}
	}
	get(t, s, "/api/weeks/current", &body)
	if body.Data.Current.Week != 14 {
		t.Errorf("current week = %d, want 14", body.Data.Current.Week)
	}
	if len(body.Data.Weeks) != 2 || body.Data.Weeks[0].Week != 13 {
		t.Errorf("weeks = %+v, want weeks 13 then 14", body.Data.Weeks)
	}
}

func TestCurrentWeekBeforeAnyIngest(t *testing.T) {
	s := &Server{Store: store.NewMemory(), Now: func() time.Time { return now }}
	if code := get(t, s, "/api/weeks/current", nil); code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503 before the calendar is loaded", code)
	}
}

func TestTeamScheduleWithRecord(t *testing.T) {
	s, _ := seeded(t)
	var body struct {
		Data struct {
			Season int             `json:"season"`
			Record football.Record `json:"record"`
			Games  []GameView      `json:"games"`
		}
	}
	if code := get(t, s, "/api/teams/2305/schedule", &body); code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	if body.Data.Season != 2026 || len(body.Data.Games) != 2 || body.Data.Games[0].ID != 3 {
		t.Errorf("schedule = %+v, want 2026 with games 3 then 1", body.Data)
	}
	if want := (football.Record{Wins: 1, Losses: 1}); body.Data.Record != want {
		t.Errorf("record = %+v, want %+v", body.Data.Record, want)
	}
}

func TestTeamNotFoundAndBadID(t *testing.T) {
	s, _ := seeded(t)
	if code := get(t, s, "/api/teams/424242", nil); code != http.StatusNotFound {
		t.Errorf("unknown team: status = %d, want 404", code)
	}
	if code := get(t, s, "/api/teams/abc", nil); code != http.StatusBadRequest {
		t.Errorf("non-numeric ID: status = %d, want 400", code)
	}
	if code := get(t, s, "/api/nope", nil); code != http.StatusNotFound {
		t.Errorf("unknown endpoint: status = %d, want 404", code)
	}
}

func TestTeamsSortedBySchool(t *testing.T) {
	s, _ := seeded(t)
	var body struct{ Data []football.Team }
	get(t, s, "/api/teams", &body)
	if len(body.Data) != 2 || body.Data[0].School != "Georgia" {
		t.Errorf("teams = %+v, want Georgia then Kansas", body.Data)
	}
}

func TestHealthAndReadiness(t *testing.T) {
	s, st := seeded(t)
	if code := get(t, s, "/api/health", nil); code != http.StatusOK {
		t.Errorf("/api/health = %d, want 200", code)
	}
	if code := get(t, s, "/api/ready", nil); code != http.StatusOK {
		t.Errorf("/api/ready = %d, want 200 while the database answers", code)
	}

	st.PingErr = errors.New("connection refused")
	if code := get(t, s, "/api/ready", nil); code != http.StatusServiceUnavailable {
		t.Errorf("/api/ready = %d, want 503 when the database is down", code)
	}
	if code := get(t, s, "/api/health", nil); code != http.StatusOK {
		t.Errorf("/api/health = %d, want 200: liveness must not depend on the database", code)
	}
}
