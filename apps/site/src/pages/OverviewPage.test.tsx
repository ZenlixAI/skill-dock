import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { OverviewPage } from './OverviewPage';

afterEach(() => {
  cleanup();
});

describe('OverviewPage', () => {
  it('renders a launch-style home page without catalog-oriented sections', () => {
    render(
      <MemoryRouter>
        <OverviewPage />
      </MemoryRouter>
    );

    expect(screen.getByRole('heading', { name: 'Ship the skills your agents rely on.' })).toBeInTheDocument();
    expect(screen.getAllByText('skills add')).toHaveLength(1);
    expect(screen.getByText(/Successful source installs sync through the existing backend import flow by default/i)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'See how it works' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Copy command/i })).toBeInTheDocument();
    expect(screen.getAllByText('npx skills find')).toHaveLength(2);
    expect(screen.getAllByText(/self-hosted npm-compatible private artifact repository/i)).toHaveLength(1);
    expect(screen.getByRole('region', { name: 'Product highlights' })).toBeInTheDocument();
    expect(screen.getByText('Find')).toBeInTheDocument();
    expect(screen.getByText('Install')).toBeInTheDocument();
    expect(screen.getByText('Control')).toBeInTheDocument();
    expect(screen.getByText(/Successful source installs sync through the existing backend import flow by default/i)).toBeInTheDocument();
    expect(screen.getByText(/Apply lifecycle and access rules without turning the homepage into a console/i)).toBeInTheDocument();

    expect(screen.queryByRole('button', { name: 'Explore catalog' })).not.toBeInTheDocument();
    expect(screen.queryByText('Release')).not.toBeInTheDocument();
    expect(screen.queryByText('Discover')).not.toBeInTheDocument();
    expect(screen.queryByText('Publish')).not.toBeInTheDocument();
    expect(screen.queryByText('Featured skills')).not.toBeInTheDocument();
    expect(screen.queryByText('Latest additions')).not.toBeInTheDocument();
    expect(screen.queryByText('View all')).not.toBeInTheDocument();
    expect(screen.queryByText('View catalog')).not.toBeInTheDocument();
  });
});
