import { useMemo, useState } from 'react';
import type { CurrentWeek, Game, Team } from '../api/types';
import { GameCard } from '../components/GameCard';
import { ErrorState, LastUpdated, Loading } from '../components/StatusViews';
import { useApi } from '../hooks/useApi';
import { groupByConference, weekKey, weekLabel } from '../lib/format';

/**
 * The home page: one week's games, grouped by conference (FR-6), with
 * previous and next week buttons (FR-7) and a conference filter (FR-13).
 */
export function HomePage() {
  const calendar = useApi<CurrentWeek>('/api/weeks/current');
  const teams = useApi<Team[]>('/api/teams');

  if (calendar.state.status === 'loading') return <Loading what="the season calendar" />;
  if (calendar.state.status === 'error') return <ErrorState message={calendar.state.message} onRetry={calendar.retry} />;
  if (teams.state.status === 'loading') return <Loading what="teams" />;
  if (teams.state.status === 'error') return <ErrorState message={teams.state.message} onRetry={teams.retry} />;

  return <WeekView calendar={calendar.state.data} teams={teams.state.data} />;
}

function WeekView({ calendar, teams }: { calendar: CurrentWeek; teams: Team[] }) {
  const { weeks } = calendar;
  const [index, setIndex] = useState(() =>
    Math.max(0, weeks.findIndex((w) => weekKey(w) === weekKey(calendar.current))),
  );
  const [conference, setConference] = useState('');
  const week = weeks[index] ?? calendar.current;

  const path = `/api/games?season=${week.season}&week=${week.week}&seasonType=${week.seasonType}`;
  const games = useApi<Game[]>(path);

  const conferences = useMemo(() => [...new Set(teams.map((t) => t.conference))].sort(), [teams]);
  const trackedIds = useMemo(() => new Set(teams.map((t) => t.id)), [teams]);

  return (
    <section aria-labelledby="week-heading">
      <div className="toolbar">
        <button type="button" onClick={() => setIndex(index - 1)} disabled={index <= 0}>
          ← Previous week
        </button>
        <h1 id="week-heading">
          {weekLabel(week)} <span className="muted">· {week.season} season</span>
        </h1>
        <button type="button" onClick={() => setIndex(index + 1)} disabled={index >= weeks.length - 1}>
          Next week →
        </button>
      </div>

      <label className="filter">
        Conference{' '}
        <select value={conference} onChange={(e) => setConference(e.target.value)}>
          <option value="">All</option>
          {conferences.map((c) => (
            <option key={c} value={c}>
              {c}
            </option>
          ))}
        </select>
      </label>

      {games.state.status === 'loading' && <Loading what="games" />}
      {games.state.status === 'error' && <ErrorState message={games.state.message} onRetry={games.retry} />}
      {games.state.status === 'success' && (
        <>
          <LastUpdated at={games.state.lastUpdated} />
          <GameGroups games={games.state.data} teams={teams} trackedIds={trackedIds} conference={conference} />
        </>
      )}
    </section>
  );
}

function GameGroups(props: { games: Game[]; teams: Team[]; trackedIds: Set<number>; conference: string }) {
  const groups = groupByConference(props.games, props.teams).filter(
    (g) => props.conference === '' || g.conference === props.conference,
  );
  if (groups.length === 0) return <p>No games for these teams this week.</p>;
  return groups.map((group) => (
    <section key={group.conference} aria-label={group.conference} className="conference-group">
      <h2>{group.conference}</h2>
      <div className="game-grid">
        {group.games.map((game) => (
          <GameCard key={game.id} game={game} trackedIds={props.trackedIds} />
        ))}
      </div>
    </section>
  ));
}
