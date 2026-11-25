// Shapes of the data the back end returns. They mirror the Go types in
// backend/internal/football and backend/internal/api: the JSON names here
// match the `json:"..."` tags there. If one side changes, change the other.
//
// TypeScript note: types exist only while compiling; none of this reaches the
// browser. A type like 'regular' | 'postseason' allows exactly those two
// strings, so a typo such as 'regualr' is caught before the code ever runs.

export type SeasonType = 'regular' | 'postseason';
export type GameStatus = 'scheduled' | 'started' | 'final';

export interface Side {
  teamId: number;
  name: string;
  conference: string;
  points: number | null; // null means no score yet, which is different from 0
}

export interface Game {
  id: number;
  season: number;
  week: number;
  seasonType: SeasonType;
  startTime: string; // ISO 8601, in UTC
  startTimeTbd: boolean;
  completed: boolean;
  neutralSite: boolean;
  conferenceGame: boolean;
  venue: string;
  home: Side;
  away: Side;
  status: GameStatus;
}

export interface Team {
  id: number;
  school: string;
  mascot: string;
  abbreviation: string;
  conference: string;
  color: string;
  alternateColor: string;
}

export interface Week {
  season: number;
  week: number;
  seasonType: SeasonType;
  start: string;
  end: string;
}

export interface TeamRecord {
  wins: number;
  losses: number;
  ties: number;
}

export interface Schedule {
  season: number;
  record: TeamRecord;
  games: Game[];
}

export interface CurrentWeek {
  current: Week;
  weeks: Week[];
}

/** Every successful response wraps its data with the time it was last refreshed. */
export interface Envelope<T> {
  data: T;
  lastUpdated: string | null;
}
