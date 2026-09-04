import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { TermsPage } from './TermsPage';

afterEach(() => {
  cleanup();
});

describe('TermsPage', () => {
  it('renders a single-column legal layout without home navigation blocks', () => {
    render(<TermsPage />);

    expect(screen.getByRole('heading', { name: 'Terms' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Usage requirements' })).toBeInTheDocument();
    expect(screen.getByRole('article', { name: 'Terms content' })).toBeInTheDocument();
    expect(screen.queryByText('Back to the site')).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Return home' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Browse catalog' })).not.toBeInTheDocument();
  });
});
