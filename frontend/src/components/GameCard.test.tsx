import { render, screen, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { describe, expect, it } from 'vitest';
import { makeGame } from '../test/server';
import { GameCard } from './GameCard';

const tracked = new Set([2305, 2306]);

function renderCard(game = makeGame({ id: 1 })) {
  render(
    <MemoryRouter>
      <GameCard game={game} trackedIds={tracked} />
    </MemoryRouter>,
  );
  return screen.getByRole('article');
}

describe('GameCard', () => {
  it('shows the kickoff time for a scheduled game, in the visitor’s time zone', () => {
    // Tests run in UTC (see vite.config.ts), so 17:00 UTC prints as 5:00 PM UTC.
    const card = renderCard();
    expect(within(card).getByText('Sat, Dec 5, 5:00 PM UTC')).toBeInTheDocument();
  });

  it('shows the final score and marks the winner', () => {
    const card = renderCard(
      makeGame({
        id: 1,
        completed: true,
        status: 'final',
        home: { teamId: 2305, name: 'Kansas', conference: 'Big 12', points: 28 },
        away: { teamId: 2306, name: 'Kansas State', conference: 'Big 12', points: 21 },
      }),
    );
    expect(within(card).getByText('Final')).toBeInTheDocument();
    expect(within(card).getByText('28')).toBeInTheDocument();
    expect(within(card).getByText('Kansas').closest('.team-line')).toHaveClass('winner');
    expect(within(card).getByText('Kansas State').closest('.team-line')).not.toHaveClass('winner');
  });

  it('does not call a started game live, because the free data has no live scores', () => {
    const card = renderCard(makeGame({ id: 1, status: 'started' }));
    expect(within(card).getByText('In progress or awaiting final score')).toBeInTheDocument();
  });

  it('links only tracked teams to a team page', () => {
    const card = renderCard(
      makeGame({ id: 1, away: { teamId: 59, name: 'Georgia Tech', conference: 'ACC', points: null } }),
    );
    expect(within(card).getByRole('link', { name: 'Kansas' })).toHaveAttribute('href', '/teams/2305');
    expect(within(card).queryByRole('link', { name: 'Georgia Tech' })).not.toBeInTheDocument();
  });
});
