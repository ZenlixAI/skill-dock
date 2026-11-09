import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { PrivacyPage } from './PrivacyPage';

afterEach(() => {
  cleanup();
});

describe('PrivacyPage', () => {
  it('renders the privacy content and a way back', () => {
    render(
      <MemoryRouter>
        <PrivacyPage />
      </MemoryRouter>
    );

    expect(screen.getByRole('heading', { name: 'Privacy' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'What the site collects' })).toBeInTheDocument();
    expect(screen.getByRole('article', { name: 'Privacy content' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Back to the product' })).toHaveAttribute('href', '/');
  });
});
