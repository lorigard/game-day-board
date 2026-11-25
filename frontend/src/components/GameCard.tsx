import { Link } from 'react-router';
import type { Game, Side } from '../api/types';
import { formatKickoff } from '../lib/format';

/** Shown on hover over a team that has no page on this site. */
const NO_PAGE_HINT = 'No team page: this site covers Big 12 and SEC teams only.';

interface GameCardProps {
  game: Game;
  /** IDs of tracked teams. Only those have a team page, so only those are links. */
  trackedIds: Set<number>;
}

/**
 * One game as a card. Layout, top to bottom:
 *
 *   ┌──────────────────────────────────────────┐
 *   │ FINAL             Sat, Nov 21, 10:00 AM  │  <- navy strip: status and date
 *   ├──────────────────────────────────────────┤
 *   │ BYU                                  31  │  <- away team first, the usual order
 *   │ Kansas                               29  │  <- home team second
 *   │ at Kansas · Memorial Stadium             │  <- who is home, and where
 *   └──────────────────────────────────────────┘
 *
 * The styles for every class used here are in styles.css, under "Game cards".
 */
export function GameCard({ game, trackedIds }: GameCardProps) {
  const winner = winningTeamId(game);

  return (
    // aria-label gives screen readers a short summary of the whole card,
    // such as "BYU at Kansas", before they read the details inside it.
    <article className="game-card" aria-label={`${game.away.name} ${game.neutralSite ? 'vs.' : 'at'} ${game.home.name}`}>
      <div className="game-strip">
        <span className="game-status">{statusLabel(game)}</span>
        {/* The date and time always show, so a finished game still says when it was played. */}
        <span className="game-when">{formatKickoff(game.startTime, game.startTimeTbd)}</span>
      </div>

      <div className="game-body">
        <TeamLine
          side={game.away}
          isTracked={trackedIds.has(game.away.teamId)}
          result={resultFor(game.away.teamId, winner)}
          started={game.status === 'started'}
        />
        <TeamLine
          side={game.home}
          isTracked={trackedIds.has(game.home.teamId)}
          result={resultFor(game.home.teamId, winner)}
          started={game.status === 'started'}
        />
        <p className="game-where">{whereText(game)}</p>
      </div>
    </article>
  );
}

/**
 * How a team's row should look. "won" and "lost" apply only to finished games;
 * "none" covers everything else, including ties, where neither row stands out.
 */
type Result = 'won' | 'lost' | 'none';

interface TeamLineProps {
  side: Side;
  isTracked: boolean;
  result: Result;
  /** True when the game has kicked off but has no final score yet. */
  started: boolean;
}

/** One team's row: its name (a link when we have a page for it) and its score. */
function TeamLine({ side, isTracked, result, started }: TeamLineProps) {
  return (
    // The result becomes a CSS class: "won" makes the row bold with a larger
    // score, "lost" softens its score, "none" leaves it plain.
    <div className={`team-line ${result}`}>
      <span className="team-name">
        {isTracked ? (
          <Link to={`/teams/${side.teamId}`}>{side.name}</Link>
        ) : (
          // Opponents from other conferences have no team page here. Without
          // an explanation, a name that is not a link looks like a bug, so
          // hovering shows why, straight away. styles.css draws the message
          // from data-hint (the browser's own "title" tooltip waits about a
          // second, which people missed). Hover does not exist on phones,
          // which is why the home page also says which names are links (see
          // HomePage.tsx); that note covers screen readers too.
          <span className="team-no-page" data-hint={NO_PAGE_HINT}>
            {side.name}
          </span>
        )}
      </span>
      <span className="team-score">
        <Score points={side.points} started={started} />
      </span>
    </div>
  );
}

/**
 * What goes in the score spot. There are three cases:
 *
 * - There is a score: show it.
 * - The game has kicked off but has no score yet: show a gray dash. The free
 *   data has no live scores, so a dash honestly says "not available yet"
 *   instead of leaving a gap that looks like something failed to load.
 * - The game has not started: show nothing, because there is no score to wait for.
 */
function Score({ points, started }: { points: number | null; started: boolean }) {
  if (points !== null) return <>{points}</>;
  if (!started) return null;
  return (
    <>
      {/* aria-hidden stops screen readers from reading the dash as "dash";
          the hidden text below tells them what it means instead. */}
      <span className="score-pending" aria-hidden="true">
        –
      </span>
      <span className="visually-hidden">score not available yet</span>
    </>
  );
}

/**
 * The word in the strip. The free data source has no live scoreboard, so a
 * game that has started but is not marked complete is "Kicked off", never
 * "Live": that is all we actually know about it.
 */
function statusLabel(game: Game): string {
  switch (game.status) {
    case 'final':
      return 'Final';
    case 'started':
      return 'Kicked off';
    default:
      return 'Upcoming';
  }
}

/**
 * The small line under the teams, for example "at Kansas · Memorial Stadium".
 * At a neutral site neither team is at home, so it says "Neutral site"
 * instead. If the data has no stadium name, that part is simply left off.
 */
function whereText(game: Game): string {
  const host = game.neutralSite ? 'Neutral site' : `at ${game.home.name}`;
  return game.venue ? `${host} · ${game.venue}` : host;
}

/** The team ID of a completed game's winner, or null for a tie or an unfinished game. */
function winningTeamId(game: Game): number | null {
  const { home, away } = game;
  if (!game.completed || home.points === null || away.points === null || home.points === away.points) return null;
  return home.points > away.points ? home.teamId : away.teamId;
}

/** Turns the winner into a per-row result: the winner "won", the other team "lost". */
function resultFor(teamId: number, winner: number | null): Result {
  if (winner === null) return 'none';
  return teamId === winner ? 'won' : 'lost';
}
