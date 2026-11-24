import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import type { CurrentWeek, Game, Schedule, Team } from '../api/types';

// A small, fixed set of data that the default handlers return.
export const LAST_UPDATED = '2026-11-23T11:48:00Z';

export const teams: Team[] = [
  { id: 2305, school: 'Kansas', mascot: 'Jayhawks', abbreviation: 'KU', conference: 'Big 12', color: '#0051ba', alternateColor: '#e8000d' },
  { id: 61, school: 'Georgia', mascot: 'Bulldogs', abbreviation: 'UGA', conference: 'SEC', color: '#ba0c2f', alternateColor: '#000000' },
];

export const calendar: CurrentWeek = {
  current: { season: 2026, week: 14, seasonType: 'regular', start: '2026-11-30T08:00:00Z', end: '2026-12-07T08:00:00Z' },
  weeks: [
    { season: 2026, week: 13, seasonType: 'regular', start: '2026-11-23T08:00:00Z', end: '2026-11-30T08:00:00Z' },
    { season: 2026, week: 14, seasonType: 'regular', start: '2026-11-30T08:00:00Z', end: '2026-12-07T08:00:00Z' },
  ],
};

export function makeGame(overrides: Partial<Game> & Pick<Game, 'id'>): Game {
  return {
    season: 2026,
    week: 14,
    seasonType: 'regular',
    startTime: '2026-12-05T17:00:00Z',
    startTimeTbd: false,
    completed: false,
    neutralSite: false,
    conferenceGame: true,
    venue: 'Memorial Stadium',
    home: { teamId: 2305, name: 'Kansas', conference: 'Big 12', points: null },
    away: { teamId: 2306, name: 'Kansas State', conference: 'Big 12', points: null },
    status: 'scheduled',
    ...overrides,
  };
}

export const week14Games: Game[] = [
  makeGame({ id: 1 }),
  makeGame({
    id: 2,
    venue: 'Sanford Stadium',
    home: { teamId: 61, name: 'Georgia', conference: 'SEC', points: 31 },
    away: { teamId: 59, name: 'Georgia Tech', conference: 'ACC', points: 17 },
    completed: true,
    status: 'final',
  }),
];

export const week13Games: Game[] = [
  makeGame({ id: 3, week: 13, home: { teamId: 2305, name: 'Kansas', conference: 'Big 12', points: null }, away: { teamId: 12, name: 'Arizona', conference: 'Big 12', points: null } }),
];

export const schedule: Schedule = {
  season: 2026,
  record: { wins: 8, losses: 3, ties: 0 },
  games: [
    makeGame({
      id: 3,
      week: 13,
      completed: true,
      status: 'final',
      home: { teamId: 12, name: 'Arizona', conference: 'Big 12', points: 10 },
      away: { teamId: 2305, name: 'Kansas', conference: 'Big 12', points: 30 },
    }),
    makeGame({ id: 1 }),
  ],
};

const ok = <T,>(data: T) => HttpResponse.json({ data, lastUpdated: LAST_UPDATED });

export const handlers = [
  http.get('*/api/weeks/current', () => ok(calendar)),
  http.get('*/api/teams', () => ok(teams)),
  http.get('*/api/games', ({ request }) => {
    const week = new URL(request.url).searchParams.get('week');
    return ok(week === '13' ? week13Games : week14Games);
  }),
  http.get('*/api/teams/:id', ({ params }) => {
    const team = teams.find((t) => String(t.id) === params.id);
    return team ? ok(team) : HttpResponse.json({ error: { code: 'not_found', message: 'No such record.' } }, { status: 404 });
  }),
  http.get('*/api/teams/:id/schedule', () => ok(schedule)),
];

export const server = setupServer(...handlers);
