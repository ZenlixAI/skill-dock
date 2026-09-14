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
    expect(screen.getByText('A simple home for the skills your agents rely on.')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Everything your team needs' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'A shared foundation for agent work' })).toBeInTheDocument();
    expect(screen.queryByText('Command model')).not.toBeInTheDocument();
    expect(screen.queryByText(/npx skills@latest add/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Stored at/)).not.toBeInTheDocument();
    expect(screen.queryByText('One product, three responsibilities')).not.toBeInTheDocument();
    expect(screen.queryByText('Explore catalog')).not.toBeInTheDocument();
  });
});
