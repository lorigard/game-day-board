import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router';
import { describe, expect, it } from 'vitest';
import { TeamPage } from './TeamPage';

function renderTeam(id: string) {
  render(
    <MemoryRouter initialEntries={[`/teams/${id}`]}>
      <Routes>
        <Route path="/teams/:id" element={<TeamPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('TeamPage', () => {
  it('shows the team, its record, and its results from its own point of view', async () => {
    renderTeam('2305');
    expect(await screen.findByRole('heading', { name: 'Kansas Jayhawks' })).toBeInTheDocument();
    expect(await screen.findByText('8-3')).toBeInTheDocument();
    // Kansas won 30-10 as the away team at Arizona.
    expect(screen.getByRole('row', { name: /at Arizona W 30-10/ })).toBeInTheDocument();
  });

  it('explains when a team is not covered', async () => {
    renderTeam('99999');
    expect(await screen.findByText(/only covers Big 12 and SEC teams/)).toBeInTheDocument();
  });
});
