import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { TermsPage } from './TermsPage';

afterEach(() => {
  cleanup();
});

describe('TermsPage', () => {
  it('renders the terms content and a way back', () => {
    render(
      <MemoryRouter>
        <TermsPage />
      </MemoryRouter>
    );

    expect(screen.getByRole('heading', { name: 'Terms' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Software license' })).toBeInTheDocument();
    expect(screen.getByRole('article', { name: 'Terms content' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Back to the product' })).toHaveAttribute('href', '/');
  });
});
