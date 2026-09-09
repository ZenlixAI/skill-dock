import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { DocsPage } from './DocsPage';

afterEach(() => {
  cleanup();
});

describe('DocsPage', () => {
  it('explains how to deploy and configure the backend with Docker', () => {
    render(
      <MemoryRouter>
        <DocsPage />
      </MemoryRouter>
    );

    expect(screen.getByRole('heading', { name: 'Get started' })).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: 'On this page' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Deploy the backend in three steps' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Configure the server' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Required configuration' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Repository and auth overrides' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Sync source installs automatically' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'skills.config.json' })).toBeInTheDocument();
    expect(screen.getByText(/Source installs sync through the backend import flow by default/i)).toBeInTheDocument();
    expect(screen.getByText(/set it to/i)).toBeInTheDocument();
    expect(screen.getByText(/POSTGRES_DB=skill_registry/)).toBeInTheDocument();
    expect(screen.getByText(/replace every placeholder with a unique secret/i)).toBeInTheDocument();
    expect(screen.getByText(/SKILLS_REGISTRY_URL=http:\/\/127\.0\.0\.1:8080/)).toBeInTheDocument();
    expect(screen.getByText(/Set these values in a `.env` file or export them before you run `docker compose`\./i)).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Explore catalog' })).not.toBeInTheDocument();
  });
});
