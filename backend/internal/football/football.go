// Package football holds the domain types that the rest of the back end shares:
// games, teams, weeks of the season, and the status of the last ingest run.
//
// These types are our own shape for the data, not CollegeFootballData's. Keeping
// them separate means a change in the outside API only touches the cfbd package
// and the mapping in ingest, never the API handlers or the database code.
package football

import (
	"fmt"
	"time"
)

// Season types used by CollegeFootballData. Bowl games and the playoff are
// "postseason" and restart their week numbers at 1.
const (
	SeasonRegular    = "regular"
	SeasonPostseason = "postseason"
)

// Game status values shown to visitors.
//
// The free CollegeFootballData tier has no live scoreboard, so the app cannot
// truly know that a game is being played right now. "started" honestly means
// "kickoff time has passed and the final result has not arrived yet".
const (
	StatusScheduled = "scheduled"
	StatusStarted   = "started"
	StatusFinal     = "final"
)

// Side is one team's half of a game: who they are and how many points they scored.
type Side struct {
	TeamID     int    `firestore:"teamId" json:"teamId"`
	Name       string `firestore:"name" json:"name"`
	Conference string `firestore:"conference" json:"conference"`
	// Points is a pointer because "no score yet" (nil) is different from a score of 0.
	Points *int `firestore:"points" json:"points"`
}

// Game is one game, stored under its CollegeFootballData game ID.
type Game struct {
	ID             int       `firestore:"id" json:"id"`
	Season         int       `firestore:"season" json:"season"`
	Week           int       `firestore:"week" json:"week"`
	SeasonType     string    `firestore:"seasonType" json:"seasonType"`
	StartTime      time.Time `firestore:"startTime" json:"startTime"`
	StartTimeTBD   bool      `firestore:"startTimeTbd" json:"startTimeTbd"`
	Completed      bool      `firestore:"completed" json:"completed"`
	NeutralSite    bool      `firestore:"neutralSite" json:"neutralSite"`
	ConferenceGame bool      `firestore:"conferenceGame" json:"conferenceGame"`
	Venue          string    `firestore:"venue" json:"venue"`
	Home           Side      `firestore:"home" json:"home"`
	Away           Side      `firestore:"away" json:"away"`

	// TeamSeasonKeys lets the database find "every game for team X in season Y"
	// with a single array-contains query, which needs no custom index.
	TeamSeasonKeys []string  `firestore:"teamSeasonKeys" json:"-"`
	UpdatedAt      time.Time `firestore:"updatedAt" json:"-"`
}

// TeamSeasonKey builds the lookup key stored in Game.TeamSeasonKeys.
func TeamSeasonKey(season, teamID int) string {
	return fmt.Sprintf("%d-%d", season, teamID)
}

// WithKeys returns a copy of the game with TeamSeasonKeys filled in for both teams.
func (g Game) WithKeys() Game {
	g.TeamSeasonKeys = []string{
		TeamSeasonKey(g.Season, g.Home.TeamID),
		TeamSeasonKey(g.Season, g.Away.TeamID),
	}
	return g
}

// Status reports whether the game is scheduled, started, or final at the given time.
// The time is passed in rather than read from the clock so tests are exact.
func (g Game) Status(now time.Time) string {
	switch {
	case g.Completed:
		return StatusFinal
	case !g.StartTimeTBD && !now.Before(g.StartTime):
		return StatusStarted
	default:
		return StatusScheduled
	}
}

// Team is one school's football team.
type Team struct {
	ID             int       `firestore:"id" json:"id"`
	School         string    `firestore:"school" json:"school"`
	Mascot         string    `firestore:"mascot" json:"mascot"`
	Abbreviation   string    `firestore:"abbreviation" json:"abbreviation"`
	Conference     string    `firestore:"conference" json:"conference"`
	Color          string    `firestore:"color" json:"color"`
	AlternateColor string    `firestore:"alternateColor" json:"alternateColor"`
	UpdatedAt      time.Time `firestore:"updatedAt" json:"-"`
}

// Week is one week of the season calendar.
type Week struct {
	Season     int       `firestore:"season" json:"season"`
	Week       int       `firestore:"week" json:"week"`
	SeasonType string    `firestore:"seasonType" json:"seasonType"`
	Start      time.Time `firestore:"start" json:"start"`
	End        time.Time `firestore:"end" json:"end"`
}

// Contains reports whether t falls inside the week (start inclusive, end exclusive).
func (w Week) Contains(t time.Time) bool {
	return !t.Before(w.Start) && t.Before(w.End)
}

// CurrentWeek picks the week a visitor most likely wants to see at time now:
// the week that contains now; otherwise the next week to start (between
// weeks, or before the season); otherwise the last week (after the season).
// It returns false only when the calendar is empty.
func CurrentWeek(weeks []Week, now time.Time) (Week, bool) {
	if len(weeks) == 0 {
		return Week{}, false
	}
	var next *Week
	last := weeks[0]
	for i := range weeks {
		w := weeks[i]
		if w.Contains(now) {
			return w, true
		}
		if w.Start.After(now) && (next == nil || w.Start.Before(next.Start)) {
			next = &weeks[i]
		}
		if w.End.After(last.End) {
			last = w
		}
	}
	if next != nil {
		return *next, true
	}
	return last, true
}

// Record is a team's win-loss record from completed games.
type Record struct {
	Wins   int `json:"wins"`
	Losses int `json:"losses"`
	Ties   int `json:"ties"`
}

// RecordFor counts wins, losses, and ties for one team. Games that are not
// completed, or that have no score, are skipped rather than guessed at.
func RecordFor(teamID int, games []Game) Record {
	var r Record
	for _, g := range games {
		if !g.Completed || g.Home.Points == nil || g.Away.Points == nil {
			continue
		}
		us, them := *g.Home.Points, *g.Away.Points
		switch teamID {
		case g.Home.TeamID:
		case g.Away.TeamID:
			us, them = them, us
		default:
			continue
		}
		switch {
		case us > them:
			r.Wins++
		case us < them:
			r.Losses++
		default:
			r.Ties++
		}
	}
	return r
}

// SeasonFor returns the season a date belongs to. Bowl games in January
// belong to the season that started the previous August.
func SeasonFor(t time.Time) int {
	if t.Month() < time.July {
		return t.Year() - 1
	}
	return t.Year()
}

// IngestStatus records how the last ingest run went. It is what lets the site
// say how old its data is (FR-9), and what an alert can check for stale data.
type IngestStatus struct {
	LastAttemptAt     time.Time `firestore:"lastAttemptAt" json:"lastAttemptAt"`
	LastSuccessAt     time.Time `firestore:"lastSuccessAt" json:"lastSuccessAt"`
	LastFullRefreshAt time.Time `firestore:"lastFullRefreshAt" json:"lastFullRefreshAt"`
	LastError         string    `firestore:"lastError" json:"lastError"`
	GamesSaved        int       `firestore:"gamesSaved" json:"gamesSaved"`
	// CallsRemaining is CollegeFootballData's own count of calls left this
	// month, read from its X-CallLimit-Remaining response header. -1 means unknown.
	CallsRemaining int `firestore:"callsRemaining" json:"callsRemaining"`
}
