import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { FeaturesPage } from './FeaturesPage';

afterEach(() => {
  cleanup();
});

describe('FeaturesPage', () => {
  it('explains the product through the user workflow', () => {
    render(
      <MemoryRouter>
        <FeaturesPage />
      </MemoryRouter>
    );

    expect(screen.getByRole('heading', { name: 'How it works' })).toBeInTheDocument();
    expect(screen.getByText(/A one-for-one replacement for the skills\.sh CLI/i)).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'A one-for-one CLI replacement' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Familiar commands, private control' })).toBeInTheDocument();
    expect(screen.getByText('npx skills add <owner/repo>')).toBeInTheDocument();
    expect(screen.getByText('npx skills update')).toBeInTheDocument();
    expect(screen.queryByText('Everything your team needs')).not.toBeInTheDocument();
    expect(screen.queryByText('A shared foundation for agent work')).not.toBeInTheDocument();
    expect(screen.queryByText(/Stored at/)).not.toBeInTheDocument();
  });
});
