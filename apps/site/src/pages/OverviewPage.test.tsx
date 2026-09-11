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

    expect(screen.getByRole('heading', { name: 'Ship the skills your agents rely on.' })).toBeInTheDocument();
    expect(screen.getByText('npx skills@latest find <query>')).toBeInTheDocument();
    expect(screen.getByText(/Find, install, and publish from one CLI/i)).toBeInTheDocument();
    expect(screen.getByText(/Successful source installs are published through Catalog/i)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Installation guide/i })).toHaveAttribute(
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
