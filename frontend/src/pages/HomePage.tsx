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
  // Both requests start at once; neither waits for the other.
  const calendar = useApi<CurrentWeek>('/api/weeks/current');
  const teams = useApi<Team[]>('/api/teams');

  // React note: a component is a function that returns what to draw. It can
  // return early, like any function, so each "if" below shows a loading or
  // error view and stops there. Only when both requests have succeeded does
  // the page go on to draw the week.
  if (calendar.state.status === 'loading') return <Loading what="the season calendar" />;
  if (calendar.state.status === 'error') return <ErrorState message={calendar.state.message} onRetry={calendar.retry} />;
  if (teams.state.status === 'loading') return <Loading what="teams" />;
  if (teams.state.status === 'error') return <ErrorState message={teams.state.message} onRetry={teams.retry} />;

  return <WeekView calendar={calendar.state.data} teams={teams.state.data} />;
}

/**
 * The week on screen, with its buttons, filter, and games.
 *
 * It is a separate component from HomePage because hooks must run in the same
 * order on every render. HomePage returns early while loading, so it could not
 * safely call useState for the week index after those returns. Here, the data
 * is guaranteed to be loaded, and every hook runs every time.
 *
 * React note: { calendar, teams } are its "props", the inputs a parent passes
 * in, like <WeekView calendar={...} teams={...} />.
 */
function WeekView({ calendar, teams }: { calendar: CurrentWeek; teams: Team[] }) {
  const { weeks } = calendar;
  // Which week is showing, as a position in the calendar list. Passing a
  // function to useState means it runs only once, for the first render,
  // to start on the current week. findIndex returns -1 if not found, and
  // Math.max turns that into 0 (the first week) instead.
  const [index, setIndex] = useState(() =>
    Math.max(0, weeks.findIndex((w) => weekKey(w) === weekKey(calendar.current))),
  );
  // The chosen conference filter; '' means "All".
  const [conference, setConference] = useState('');
  const week = weeks[index] ?? calendar.current;

  // Changing week changes the path, and useApi fetches again on its own.
  const path = `/api/games?season=${week.season}&week=${week.week}&seasonType=${week.seasonType}`;
  const games = useApi<Game[]>(path);

  // useMemo remembers a computed value and only recomputes it when the
  // listed inputs ([teams]) change, not on every click.
  // new Set(...) removes duplicates; [...set] turns it back into an array.
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

      {/* "condition && <Thing />" draws Thing only when the condition is true.
          Exactly one of these three lines draws something at a time. */}
      {games.state.status === 'loading' && <Loading what="games" />}
      {games.state.status === 'error' && <ErrorState message={games.state.message} onRetry={games.retry} />}
      {games.state.status === 'success' && (
        <>
          <LastUpdated at={games.state.lastUpdated} />
          {/* Says up front why some names are links and others are not.
              The cards also explain on hover, but phones cannot hover. */}
          <p className="muted small">Big 12 and SEC team names link to team pages.</p>
          <GameGroups games={games.state.data} teams={teams} trackedIds={trackedIds} conference={conference} />
        </>
      )}
    </section>
  );
}

/**
 * The games, one section per conference, after the conference filter.
 *
 * React note: when drawing a list, each item needs a "key" that is unique
 * among its siblings and stays the same between renders (here the conference
 * name, and each game's ID). React uses keys to tell which items changed,
 * moved, or disappeared, instead of redrawing the whole list.
 */
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
