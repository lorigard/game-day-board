import type { Game, Team, TeamRecord, Week } from '../api/types';

/**
 * Formats a kickoff time in the visitor's own time zone (FR-6), for example
 * "Sat, Nov 28, 11:00 AM MST". When the start time is not yet decided, the
 * date is only a placeholder, so only the day is shown.
 */
export function formatKickoff(iso: string, timeTbd: boolean, timeZone?: string): string {
  const date = new Date(iso);
  const day = new Intl.DateTimeFormat('en-US', { weekday: 'short', month: 'short', day: 'numeric', timeZone }).format(date);
  if (timeTbd) return `${day}, time to be announced`;
  const time = new Intl.DateTimeFormat('en-US', { hour: 'numeric', minute: '2-digit', timeZoneName: 'short', timeZone }).format(date);
  // Newer browsers put a narrow no-break space before AM and PM. Replacing
  // every kind of space with a plain one makes the text the same everywhere.
  return `${day}, ${time.replace(/\s/gu, ' ')}`;
}

/** Says how long ago a time was, in words, for example "12 minutes ago". */
export function timeAgo(iso: string, now: Date = new Date()): string {
  const seconds = Math.round((new Date(iso).getTime() - now.getTime()) / 1000);
  const rtf = new Intl.RelativeTimeFormat('en-US', { numeric: 'auto' });
  const units: [Intl.RelativeTimeFormatUnit, number][] = [
    ['day', 86400],
    ['hour', 3600],
    ['minute', 60],
  ];
  for (const [unit, size] of units) {
    if (Math.abs(seconds) >= size) return rtf.format(Math.round(seconds / size), unit);
  }
  return 'just now';
}

/** "8-3", or "8-3-1" when there are ties. */
export function formatRecord(r: TeamRecord): string {
  return r.ties > 0 ? `${r.wins}-${r.losses}-${r.ties}` : `${r.wins}-${r.losses}`;
}

/** "Week 13" for the regular season; "Bowls and playoff" for the postseason. */
export function weekLabel(w: Pick<Week, 'week' | 'seasonType'>): string {
  return w.seasonType === 'postseason' ? 'Bowls and playoff' : `Week ${w.week}`;
}

/** A stable key for a week, since week numbers repeat between regular season and postseason. */
export function weekKey(w: Pick<Week, 'season' | 'week' | 'seasonType'>): string {
  return `${w.season}-${w.seasonType}-${w.week}`;
}

/**
 * Groups games under the conference of the tracked team in each game, so a
 * game appears once even when both teams are tracked. Groups are sorted by
 * conference name, and games keep the order they came in (by kickoff).
 */
export function groupByConference(games: Game[], teams: Team[]): { conference: string; games: Game[] }[] {
  const trackedIds = new Set(teams.map((t) => t.id));
  const groups = new Map<string, Game[]>();
  for (const game of games) {
    const side = trackedIds.has(game.home.teamId) ? game.home : game.away;
    const list = groups.get(side.conference) ?? [];
    list.push(game);
    groups.set(side.conference, list);
  }
  return [...groups.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([conference, list]) => ({ conference, games: list }));
}
