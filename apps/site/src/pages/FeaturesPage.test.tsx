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
    expect(screen.getByText('A private package registry for reusable agent skills.')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'What developers can do' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Automatic source publication' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Command model' })).toBeInTheDocument();
    expect(screen.getByText(/npx skills@latest add/)).toBeInTheDocument();
    expect(screen.getByText(/Stored at/)).toBeInTheDocument();
    expect(screen.queryByText('One product, three responsibilities')).not.toBeInTheDocument();
    expect(screen.queryByText('Explore catalog')).not.toBeInTheDocument();
  });
});
