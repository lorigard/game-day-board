import { describe, expect, it } from 'vitest';
import type { Game, Team } from '../api/types';
import { formatKickoff, formatRecord, groupByConference, timeAgo, weekLabel } from './format';

describe('formatKickoff', () => {
  it('shows the day and time in the given time zone', () => {
    // 17:00 UTC on Nov 27, 2026 is 10:00 AM in Denver (Mountain Standard Time).
    expect(formatKickoff('2026-11-27T17:00:00Z', false, 'America/Denver')).toBe('Fri, Nov 27, 10:00 AM MST');
  });

  it('shows only the day when the time is not decided', () => {
    expect(formatKickoff('2026-11-28T05:00:00Z', true, 'America/Chicago')).toBe('Fri, Nov 27 · Time TBA');
  });
});

describe('timeAgo', () => {
  const now = new Date('2026-11-23T12:00:00Z');
  it.each([
    ['2026-11-23T11:59:30Z', 'just now'],
    ['2026-11-23T11:48:00Z', '12 minutes ago'],
    ['2026-11-23T09:00:00Z', '3 hours ago'],
    ['2026-11-21T12:00:00Z', '2 days ago'],
  ])('describes %s as "%s"', (iso, want) => {
    expect(timeAgo(iso, now)).toBe(want);
  });
});

describe('formatRecord', () => {
  it('leaves ties out when there are none', () => {
    expect(formatRecord({ wins: 8, losses: 3, ties: 0 })).toBe('8-3');
    expect(formatRecord({ wins: 8, losses: 3, ties: 1 })).toBe('8-3-1');
  });
});

describe('weekLabel', () => {
  it('names regular season weeks by number and the postseason by name', () => {
    expect(weekLabel({ week: 13, seasonType: 'regular' })).toBe('Week 13');
    expect(weekLabel({ week: 1, seasonType: 'postseason' })).toBe('Bowls and playoff');
  });
});

describe('groupByConference', () => {
  const teams = [{ id: 1, conference: 'SEC' }, { id: 2, conference: 'Big 12' }, { id: 3, conference: 'Big 12' }] as Team[];
  const game = (id: number, homeId: number, homeConf: string, awayId: number, awayConf: string) =>
    ({ id, home: { teamId: homeId, conference: homeConf }, away: { teamId: awayId, conference: awayConf } }) as Game;

  it('puts each game under its tracked team’s conference, once', () => {
    const groups = groupByConference(
      [
        game(10, 1, 'SEC', 99, 'ACC'), // SEC home team
        game(11, 98, 'ACC', 2, 'Big 12'), // only the away team is tracked
        game(12, 2, 'Big 12', 3, 'Big 12'), // both tracked: listed once
      ],
      teams,
    );
    expect(groups.map((g) => [g.conference, g.games.map((x) => x.id)])).toEqual([
      ['Big 12', [11, 12]],
      ['SEC', [10]],
    ]);
  });
});
