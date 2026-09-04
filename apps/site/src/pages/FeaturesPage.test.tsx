import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { FeaturesPage } from './FeaturesPage';

afterEach(() => {
  cleanup();
});

describe('FeaturesPage', () => {
  it('reads like a product introduction with clear boundaries', () => {
    render(
      <MemoryRouter>
        <FeaturesPage />
      </MemoryRouter>
    );

    expect(screen.getByRole('heading', { name: 'Features' })).toBeInTheDocument();
    expect(screen.getByText('A self-hosted control plane for reusable skills.')).toBeInTheDocument();
    expect(screen.getByText('Self-hosted')).toBeInTheDocument();
    expect(screen.getByText('Server authoritative')).toBeInTheDocument();
    expect(screen.getByText('Private npm-compatible storage')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Designed around clear boundaries' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'One product, three responsibilities' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Control' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Storage' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'CLI' })).toBeInTheDocument();
    expect(screen.getByText(/Built to stay self-hosted, predictable, and easy to explain\./i)).toBeInTheDocument();
    expect(screen.queryByText('Deployment is explicit')).not.toBeInTheDocument();
    expect(screen.queryByText('Explore catalog')).not.toBeInTheDocument();
  });
});
