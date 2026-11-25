import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { MemoryRouter } from 'react-router';
import { describe, expect, it } from 'vitest';
import { server } from '../test/server';
import { HomePage } from './HomePage';

function renderHome() {
  render(
    <MemoryRouter>
      <HomePage />
    </MemoryRouter>,
  );
}

describe('HomePage', () => {
  it('shows the current week’s games grouped by conference, with the data’s age', async () => {
    renderHome();
    expect(screen.getByRole('status')).toHaveTextContent('Loading');

    expect(await screen.findByRole('heading', { level: 1, name: /Week 14/ })).toBeInTheDocument();
    const big12 = await screen.findByRole('region', { name: 'Big 12' });
    expect(within(big12).getByRole('link', { name: 'Kansas' })).toBeInTheDocument();
    const sec = screen.getByRole('region', { name: 'SEC' });
    expect(within(sec).getByText('Final')).toBeInTheDocument();
    expect(screen.getByText(/Data updated/)).toBeInTheDocument();
  });

  it('moves to the previous week', async () => {
    renderHome();
    const user = userEvent.setup();
    await user.click(await screen.findByRole('button', { name: /Previous week/ }));

    expect(await screen.findByRole('heading', { level: 1, name: /Week 13/ })).toBeInTheDocument();
    expect(await screen.findByText('Arizona')).toBeInTheDocument();
    // Arizona is not in the tracked teams of this test data, so it is plain text, not a link.
    expect(screen.queryByRole('link', { name: 'Arizona' })).not.toBeInTheDocument();
    // Week 13 is the first week in the calendar, so there is no earlier one.
    expect(screen.getByRole('button', { name: /Previous week/ })).toBeDisabled();
  });

  it('filters to one conference', async () => {
    renderHome();
    const user = userEvent.setup();
    await screen.findByRole('region', { name: 'SEC' });

    await user.selectOptions(screen.getByRole('combobox', { name: /Conference/ }), 'SEC');
    expect(screen.getByRole('region', { name: 'SEC' })).toBeInTheDocument();
    expect(screen.queryByRole('region', { name: 'Big 12' })).not.toBeInTheDocument();
  });

  it('shows the server’s message and recovers when Try again works', async () => {
    let calls = 0;
    server.use(
      http.get('*/api/games', () => {
        calls += 1;
        if (calls === 1) {
          return HttpResponse.json(
            { error: { code: 'unavailable', message: 'The data is temporarily unavailable. Please try again.' } },
            { status: 503 },
          );
        }
        return HttpResponse.json({ data: [], lastUpdated: null });
      }),
    );
    renderHome();
    const user = userEvent.setup();

    expect(await screen.findByRole('alert')).toHaveTextContent('The data is temporarily unavailable');
    await user.click(screen.getByRole('button', { name: 'Try again' }));
    expect(await screen.findByText('No Big 12 or SEC games in this week.')).toBeInTheDocument();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('explains a network failure instead of showing a blank page', async () => {
    server.use(http.get('*/api/weeks/current', () => HttpResponse.error()));
    renderHome();
    expect(await screen.findByRole('alert')).toHaveTextContent('Could not reach the server');
  });
});
