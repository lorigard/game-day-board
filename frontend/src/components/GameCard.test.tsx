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

  it('shows the final score, marks the winner and loser, and still shows the date', () => {
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
    expect(within(card).getByText('Kansas').closest('.team-line')).toHaveClass('won');
    expect(within(card).getByText('Kansas State').closest('.team-line')).toHaveClass('lost');
    // A finished game keeps its date and time, so you can tell when it was played.
    expect(card).toHaveTextContent('Sat, Dec 5, 5:00 PM UTC');
  });

  it('says who is home and where: "at Kansas · Memorial Stadium"', () => {
    expect(renderCard()).toHaveTextContent('at Kansas · Memorial Stadium');
  });

  it('says "Neutral site" instead of a home team at a neutral site', () => {
    const card = renderCard(makeGame({ id: 1, neutralSite: true, venue: 'AT&T Stadium' }));
    expect(card).toHaveTextContent('Neutral site · AT&T Stadium');
    expect(card).not.toHaveTextContent('at Kansas');
  });

  it('leaves the stadium off when the data has none', () => {
    const card = renderCard(makeGame({ id: 1, venue: '' }));
    expect(within(card).getByText('at Kansas')).toBeInTheDocument();
  });

  it('labels a game that has not started "Upcoming", with no scores', () => {
    const card = renderCard();
    expect(within(card).getByText('Upcoming')).toBeInTheDocument();
    expect(within(card).queryByText('–')).not.toBeInTheDocument();
  });

  it('calls a started game "Kicked off", never live, with a dash for each score', () => {
    // The free data has no live scores, so all we honestly know is that it started.
    const card = renderCard(makeGame({ id: 1, status: 'started' }));
    expect(within(card).getByText('Kicked off')).toBeInTheDocument();
    expect(within(card).getAllByText('–')).toHaveLength(2);
    // Screen readers hear words instead of "dash".
    expect(within(card).getAllByText('score not available yet')).toHaveLength(2);
    expect(card).toHaveTextContent('Sat, Dec 5, 5:00 PM UTC');
  });

  it('links only tracked teams to a team page', () => {
    const card = renderCard(
      makeGame({ id: 1, away: { teamId: 59, name: 'Georgia Tech', conference: 'ACC', points: null } }),
    );
    expect(within(card).getByRole('link', { name: 'Kansas' })).toHaveAttribute('href', '/teams/2305');
    expect(within(card).queryByRole('link', { name: 'Georgia Tech' })).not.toBeInTheDocument();
  });

  it('explains on hover why a team has no link', () => {
    const card = renderCard(
      makeGame({ id: 1, away: { teamId: 59, name: 'Georgia Tech', conference: 'ACC', points: null } }),
    );
    // styles.css shows the data-hint text as a tooltip on hover.
    expect(within(card).getByText('Georgia Tech')).toHaveAttribute(
      'data-hint',
      'No team page: this site covers Big 12 and SEC teams only.',
    );
  });
});
