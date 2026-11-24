package cfbd

import (
	"context"
	"net/url"
	"strconv"
	"time"
)

// Game is one game as CollegeFootballData returns it. Only the fields this
// project uses are listed; encoding/json ignores the rest.
type Game struct {
	ID             int       `json:"id"`
	Season         int       `json:"season"`
	Week           int       `json:"week"`
	SeasonType     string    `json:"seasonType"`
	StartDate      time.Time `json:"startDate"`
	StartTimeTBD   bool      `json:"startTimeTBD"`
	Completed      bool      `json:"completed"`
	NeutralSite    bool      `json:"neutralSite"`
	ConferenceGame bool      `json:"conferenceGame"`
	Venue          string    `json:"venue"`
	HomeID         int       `json:"homeId"`
	HomeTeam       string    `json:"homeTeam"`
	HomeConference string    `json:"homeConference"`
	HomePoints     *int      `json:"homePoints"`
	AwayID         int       `json:"awayId"`
	AwayTeam       string    `json:"awayTeam"`
	AwayConference string    `json:"awayConference"`
	AwayPoints     *int      `json:"awayPoints"`
}

// Team is one team as CollegeFootballData returns it.
type Team struct {
	ID             int    `json:"id"`
	School         string `json:"school"`
	Mascot         string `json:"mascot"`
	Abbreviation   string `json:"abbreviation"`
	Conference     string `json:"conference"`
	Color          string `json:"color"`
	AlternateColor string `json:"alternateColor"`
}

// Week is one entry of the season calendar.
type Week struct {
	Season     int       `json:"season"`
	Week       int       `json:"week"`
	SeasonType string    `json:"seasonType"`
	StartDate  time.Time `json:"startDate"`
	EndDate    time.Time `json:"endDate"`
}

// GamesQuery selects games. Week 0 means "every week of the season type".
type GamesQuery struct {
	Year       int
	Week       int
	SeasonType string
}

// Games returns Football Bowl Subdivision (FBS) games for the query. One call
// covers every conference; the ingest service keeps only the teams it tracks,
// which costs fewer calls than asking once per conference.
func (c *Client) Games(ctx context.Context, q GamesQuery) ([]Game, error) {
	v := url.Values{}
	v.Set("year", strconv.Itoa(q.Year))
	v.Set("classification", "fbs")
	if q.Week > 0 {
		v.Set("week", strconv.Itoa(q.Week))
	}
	if q.SeasonType != "" {
		v.Set("seasonType", q.SeasonType)
	}
	var games []Game
	err := c.get(ctx, "/games", v, &games)
	return games, err
}

// Teams returns the teams in one conference, by its abbreviation (for example "SEC" or "B12").
func (c *Client) Teams(ctx context.Context, conference string) ([]Team, error) {
	v := url.Values{}
	v.Set("conference", conference)
	var teams []Team
	err := c.get(ctx, "/teams", v, &teams)
	return teams, err
}

// Calendar returns every week of a season, regular season and postseason.
func (c *Client) Calendar(ctx context.Context, year int) ([]Week, error) {
	v := url.Values{}
	v.Set("year", strconv.Itoa(year))
	var weeks []Week
	err := c.get(ctx, "/calendar", v, &weeks)
	return weeks, err
}
