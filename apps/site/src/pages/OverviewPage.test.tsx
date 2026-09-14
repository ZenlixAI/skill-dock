import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { OverviewPage } from './OverviewPage';

afterEach(() => {
  cleanup();
});

describe('OverviewPage', () => {
  it('introduces the product through the terminal workflow', () => {
    render(
      <MemoryRouter>
        <OverviewPage />
      </MemoryRouter>
    );

    expect(screen.getByRole('heading', { name: 'Private skills for every agent.' })).toBeInTheDocument();
    expect(screen.getByText('npx skills@latest find <query>')).toBeInTheDocument();
    expect(screen.getByText(/Search, install, and publish from one CLI/i)).toBeInTheDocument();
    expect(screen.getByText(/Optional source sync/i)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Install the registry/i })).toHaveAttribute(
      'href',
      'https://github.com/ZenlixAI/skill-dock#docker-compose'
    );
    expect(screen.getByRole('button', { name: /Copy command/i })).toBeInTheDocument();
    expect(screen.getByRole('region', { name: 'Product highlights' })).toBeInTheDocument();
    expect(screen.getByText('Find')).toBeInTheDocument();
    expect(screen.getByText('Install')).toBeInTheDocument();
    expect(screen.getByText('Control')).toBeInTheDocument();
    expect(screen.queryByText('Featured skills')).not.toBeInTheDocument();
    expect(screen.queryByText('View catalog')).not.toBeInTheDocument();
  });
});
