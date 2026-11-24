package store

import (
	"context"
	"slices"
	"sync"

	"github.com/lorigard/game-day-board/backend/internal/football"
)

// Memory is an in-memory Store for tests. It is safe for concurrent use.
// PingErr, when set, makes Ping fail, to test how callers react to an outage.
type Memory struct {
	mu       sync.Mutex
	games    map[int]football.Game
	teams    map[int]football.Team
	calendar []football.Week
	status   football.IngestStatus
	PingErr  error
}

// NewMemory returns an empty in-memory store.
func NewMemory() *Memory {
	return &Memory{games: map[int]football.Game{}, teams: map[int]football.Team{}}
}

var _ Store = (*Memory)(nil) // compile-time check that Memory implements Store

func (m *Memory) SaveGames(_ context.Context, games []football.Game) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, g := range games {
		m.games[g.ID] = g.WithKeys()
	}
	return nil
}

func (m *Memory) SaveTeams(_ context.Context, teams []football.Team) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range teams {
		m.teams[t.ID] = t
	}
	return nil
}

func (m *Memory) SaveCalendar(_ context.Context, weeks []football.Week) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calendar = slices.Clone(weeks)
	return nil
}

func (m *Memory) SaveIngestStatus(_ context.Context, s football.IngestStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = s
	return nil
}

func (m *Memory) GamesForWeek(_ context.Context, season, week int, seasonType string) ([]football.Game, error) {
	return m.filterGames(func(g football.Game) bool {
		return g.Season == season && g.Week == week && g.SeasonType == seasonType
	}), nil
}

func (m *Memory) GamesForTeam(_ context.Context, season, teamID int) ([]football.Game, error) {
	key := football.TeamSeasonKey(season, teamID)
	return m.filterGames(func(g football.Game) bool { return slices.Contains(g.TeamSeasonKeys, key) }), nil
}

func (m *Memory) filterGames(keep func(football.Game) bool) []football.Game {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []football.Game
	for _, g := range m.games {
		if keep(g) {
			out = append(out, g)
		}
	}
	return out
}

func (m *Memory) Team(_ context.Context, id int) (football.Team, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.teams[id]
	if !ok {
		return football.Team{}, ErrNotFound
	}
	return t, nil
}

func (m *Memory) Teams(_ context.Context) ([]football.Team, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]football.Team, 0, len(m.teams))
	for _, t := range m.teams {
		out = append(out, t)
	}
	return out, nil
}

func (m *Memory) Calendar(_ context.Context) ([]football.Week, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.calendar), nil
}

func (m *Memory) IngestStatus(_ context.Context) (football.IngestStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status, nil
}

func (m *Memory) Ping(context.Context) error { return m.PingErr }

// GameCount returns how many games are stored. Tests use it to check idempotency.
func (m *Memory) GameCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.games)
}
