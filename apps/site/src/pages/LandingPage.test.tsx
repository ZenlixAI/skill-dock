import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { LandingPage } from './LandingPage';

afterEach(() => {
  cleanup();
});

describe('LandingPage', () => {
  it('introduces the catalog, the release path, and the quick start', async () => {
    const { container } = render(
      <MemoryRouter>
        <LandingPage />
      </MemoryRouter>
    );

    expect(
      screen.getByRole('heading', { name: 'Agent skills, with a release process.' })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: 'A git ref is not a release.' })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: 'Publish, resolve, govern.' })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: 'Three pieces, one release path.' })
    ).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Run it locally.' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Questions that come up.' })).toBeInTheDocument();

    expect(screen.getAllByText('docker compose up --build -d').length).toBeGreaterThan(0);
    expect(screen.getAllByText('skills publish ./my-skill').length).toBeGreaterThan(0);
    expect(screen.queryByRole('tabpanel', { name: 'Install' })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('tab', { name: 'Install' }));
    expect(await screen.findByRole('tabpanel', { name: 'Install' })).toBeInTheDocument();

    expect(container.querySelectorAll('.faq-item')).toHaveLength(5);
    expect(screen.queryByText('MIT · Go · TypeScript')).not.toBeInTheDocument();
  });

  it('keeps the public copy focused on the product path', () => {
    render(
      <MemoryRouter>
        <LandingPage />
      </MemoryRouter>
    );

    expect(screen.queryByText(/one-for-one/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/skills\.sh/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/marketplace/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/coming soon/i)).not.toBeInTheDocument();
  });
});
