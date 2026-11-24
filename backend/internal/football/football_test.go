package football

import (
	"testing"
	"time"
)

func intPtr(n int) *int { return &n }

func TestGameStatus(t *testing.T) {
	kickoff := time.Date(2026, 11, 28, 17, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		game Game
		now  time.Time
		want string
	}{
		{"before kickoff", Game{StartTime: kickoff}, kickoff.Add(-time.Minute), StatusScheduled},
		{"at kickoff", Game{StartTime: kickoff}, kickoff, StatusStarted},
		{"after kickoff, not completed", Game{StartTime: kickoff}, kickoff.Add(2 * time.Hour), StatusStarted},
		{"completed", Game{StartTime: kickoff, Completed: true}, kickoff.Add(4 * time.Hour), StatusFinal},
		// A TBD start time is a placeholder date, so passing it proves nothing.
		{"time to be decided", Game{StartTime: kickoff, StartTimeTBD: true}, kickoff.Add(time.Hour), StatusScheduled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.game.Status(tt.now); got != tt.want {
				t.Errorf("Status() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRecordFor(t *testing.T) {
	const us = 1
	games := []Game{
		{Completed: true, Home: Side{TeamID: us, Points: intPtr(30)}, Away: Side{TeamID: 2, Points: intPtr(10)}}, // home win
		{Completed: true, Home: Side{TeamID: 3, Points: intPtr(21)}, Away: Side{TeamID: us, Points: intPtr(24)}}, // away win
		{Completed: true, Home: Side{TeamID: 4, Points: intPtr(14)}, Away: Side{TeamID: us, Points: intPtr(7)}},  // away loss
		{Completed: false, Home: Side{TeamID: us}, Away: Side{TeamID: 5}},                                        // not played
		{Completed: true, Home: Side{TeamID: us}, Away: Side{TeamID: 6}},                                         // no score: skipped
		{Completed: true, Home: Side{TeamID: 7, Points: intPtr(3)}, Away: Side{TeamID: 8, Points: intPtr(0)}},    // other teams
	}
	want := Record{Wins: 2, Losses: 1}
	if got := RecordFor(us, games); got != want {
		t.Errorf("RecordFor() = %+v, want %+v", got, want)
	}
}

func TestCurrentWeek(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 11, d, 12, 0, 0, 0, time.UTC) }
	weeks := []Week{
		{Week: 13, Start: day(16), End: day(23)},
		{Week: 14, Start: day(23), End: day(30)},
		{Week: 15, Start: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 12, 8, 0, 0, 0, 0, time.UTC)},
	}
	tests := []struct {
		name string
		now  time.Time
		want int
	}{
		{"inside a week", day(18), 13},
		{"on the boundary belongs to the later week", day(23), 14},
		{"in a gap between weeks picks the next week", day(30).Add(time.Hour), 15},
		{"before the season picks the first week", day(1), 13},
		{"after the season picks the last week", time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC), 15},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := CurrentWeek(weeks, tt.now)
			if !ok || got.Week != tt.want {
				t.Errorf("CurrentWeek() = week %d (ok=%v), want week %d", got.Week, ok, tt.want)
			}
		})
	}
	if _, ok := CurrentWeek(nil, day(1)); ok {
		t.Error("CurrentWeek(nil) ok = true, want false")
	}
}

func TestSeasonFor(t *testing.T) {
	tests := []struct {
		date time.Time
		want int
	}{
		{time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC), 2026},
		{time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), 2026},
		{time.Date(2027, 1, 12, 0, 0, 0, 0, time.UTC), 2026}, // bowl season belongs to the fall season
		{time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC), 2027},
	}
	for _, tt := range tests {
		if got := SeasonFor(tt.date); got != tt.want {
			t.Errorf("SeasonFor(%s) = %d, want %d", tt.date.Format(time.DateOnly), got, tt.want)
		}
	}
}

func TestWithKeys(t *testing.T) {
	g := Game{Season: 2026, Home: Side{TeamID: 145}, Away: Side{TeamID: 344}}.WithKeys()
	want := []string{"2026-145", "2026-344"}
	if len(g.TeamSeasonKeys) != 2 || g.TeamSeasonKeys[0] != want[0] || g.TeamSeasonKeys[1] != want[1] {
		t.Errorf("TeamSeasonKeys = %v, want %v", g.TeamSeasonKeys, want)
	}
}
