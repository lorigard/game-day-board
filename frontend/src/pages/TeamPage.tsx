import { Link, useParams } from 'react-router';
import type { Game, Schedule, Team } from '../api/types';
import { ErrorState, LastUpdated, Loading } from '../components/StatusViews';
import { useApi } from '../hooks/useApi';
import { formatKickoff, formatRecord, weekLabel } from '../lib/format';

/** One team's page: its season schedule, results, and record (FR-8). */
export function TeamPage() {
  // useParams reads the parts of the address that App.tsx named in the route.
  // The route is "/teams/:id", so for /teams/2305 this gives id = "2305".
  // The back end checks that it is a number; a bad one comes back as a 400.
  const { id } = useParams();
  const team = useApi<Team>(`/api/teams/${id}`);
  const schedule = useApi<Schedule>(`/api/teams/${id}/schedule`);

  if (team.state.status === 'loading') return <Loading what="the team" />;
  if (team.state.status === 'error') {
    if (team.state.httpStatus === 404) {
      return (
        <p>
          This site only covers Big 12 and SEC teams, and there is no team with that ID. <Link to="/">Back to this week’s games</Link>
        </p>
      );
    }
    return <ErrorState message={team.state.message} onRetry={team.retry} />;
  }

  const t = team.state.data;
  return (
    // The team color is used only for a decorative stripe, never behind text,
    // so it cannot make text hard to read (NFR-15).
    <article className="team-page" style={{ borderTopColor: t.color || undefined }}>
      <h1>
        {t.school} {t.mascot}
      </h1>
      <p className="muted">{t.conference}</p>

      {schedule.state.status === 'loading' && <Loading what="the schedule" />}
      {schedule.state.status === 'error' && <ErrorState message={schedule.state.message} onRetry={schedule.retry} />}
      {schedule.state.status === 'success' && (
        <>
          <p className="record">
            {schedule.state.data.season} record: <strong>{formatRecord(schedule.state.data.record)}</strong>
          </p>
          <LastUpdated at={schedule.state.lastUpdated} />
          <ScheduleTable teamId={t.id} games={schedule.state.data.games} />
        </>
      )}
    </article>
  );
}

/**
 * The season schedule as a table. A real <table> (not a grid of divs) is the
 * right choice for rows and columns of data: screen readers can announce the
 * column heading for each cell, which scope="col" below tells them.
 */
function ScheduleTable({ teamId, games }: { teamId: number; games: Game[] }) {
  if (games.length === 0) return <p>No games found for this season yet.</p>;
  return (
    <table className="schedule">
      <caption className="visually-hidden">Season schedule and results</caption>
      <thead>
        <tr>
          <th scope="col">Week</th>
          <th scope="col">Opponent</th>
          <th scope="col">Result or kickoff</th>
        </tr>
      </thead>
      <tbody>
        {games.map((g) => {
          const isHome = g.home.teamId === teamId;
          const opponent = isHome ? g.away : g.home;
          return (
            <tr key={g.id}>
              <td>{weekLabel(g)}</td>
              {/* Schedule convention: "vs. X" for a home game (or a neutral
                  site), "at X" when this team travels to X. */}
              <td>
                {isHome || g.neutralSite ? 'vs.' : 'at'} {opponent.name}
              </td>
              <td>{result(g, isHome)}</td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

/** "W 28-21", "L 10-30", or the kickoff time, from this team's point of view. */
function result(g: Game, isHome: boolean): string {
  const us = isHome ? g.home.points : g.away.points;
  const them = isHome ? g.away.points : g.home.points;
  if (g.status === 'final' && us !== null && them !== null) {
    // Two "?:" in a row read as: if us > them then W, else if us < them then L, else T.
    const letter = us > them ? 'W' : us < them ? 'L' : 'T';
    return `${letter} ${us}-${them}`;
  }
  // Same honest wording as the game cards: we know it started, not the score.
  if (g.status === 'started') return 'Kicked off, awaiting final score';
  return formatKickoff(g.startTime, g.startTimeTbd);
}
