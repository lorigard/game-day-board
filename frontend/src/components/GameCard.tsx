import { Link } from 'react-router';
import type { Game, Side } from '../api/types';
import { formatKickoff } from '../lib/format';

interface GameCardProps {
  game: Game;
  /** IDs of tracked teams. Only those have a team page, so only those are links. */
  trackedIds: Set<number>;
}

/** One game: both teams, the score once there is one, and the kickoff or status. */
export function GameCard({ game, trackedIds }: GameCardProps) {
  const winner = winningTeamId(game);
  return (
    <article className="game-card" aria-label={`${game.away.name} at ${game.home.name}`}>
      <TeamLine side={game.away} isTracked={trackedIds.has(game.away.teamId)} isWinner={winner === game.away.teamId} />
      <TeamLine side={game.home} isTracked={trackedIds.has(game.home.teamId)} isWinner={winner === game.home.teamId} />
      <p className="game-meta">
        <StatusText game={game} />
        {game.venue && <span className="muted"> · {game.neutralSite ? `${game.venue} (neutral site)` : game.venue}</span>}
      </p>
    </article>
  );
}

function TeamLine({ side, isTracked, isWinner }: { side: Side; isTracked: boolean; isWinner: boolean }) {
  return (
    <div className={isWinner ? 'team-line winner' : 'team-line'}>
      <span className="team-name">
        {isTracked ? <Link to={`/teams/${side.teamId}`}>{side.name}</Link> : side.name}
      </span>
      <span className="team-score">{side.points ?? ''}</span>
    </div>
  );
}

/**
 * The free data source has no live scoreboard, so a game that has kicked off
 * but is not marked complete is described honestly, not called "live".
 */
function StatusText({ game }: { game: Game }) {
  switch (game.status) {
    case 'final':
      return <strong>Final</strong>;
    case 'started':
      return <span>In progress or awaiting final score</span>;
    default:
      return <span>{formatKickoff(game.startTime, game.startTimeTbd)}</span>;
  }
}

/** The team ID of a completed game's winner, or null for a tie or an unfinished game. */
function winningTeamId(game: Game): number | null {
  const { home, away } = game;
  if (!game.completed || home.points === null || away.points === null || home.points === away.points) return null;
  return home.points > away.points ? home.teamId : away.teamId;
}
