import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { PrivacyPage } from './PrivacyPage';

afterEach(() => {
  cleanup();
});

describe('PrivacyPage', () => {
  it('renders a single-column legal layout without home navigation blocks', () => {
    render(<PrivacyPage />);

    expect(screen.getByRole('heading', { name: 'Privacy' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'What we collect' })).toBeInTheDocument();
    expect(screen.getByRole('article', { name: 'Privacy content' })).toBeInTheDocument();
    expect(screen.queryByText('Back to the site')).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Return home' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Browse catalog' })).not.toBeInTheDocument();
  });
});
