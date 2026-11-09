import { cleanup, render, screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { AppShell } from './AppShell';

afterEach(() => {
  cleanup();
});

function renderAppShell(pathname: string) {
  return render(
    <MemoryRouter initialEntries={[pathname]}>
      <AppShell>
        <div>content</div>
      </AppShell>
    </MemoryRouter>
  );
}

describe('AppShell', () => {
  it('renders the primary navigation and skip link', () => {
    renderAppShell('/');

    const primary = screen.getByRole('navigation', { name: 'Primary' });

    expect(screen.getByRole('link', { name: 'Skip to main content' })).toHaveAttribute(
      'href',
      '#main-content'
    );
    expect(screen.getByRole('link', { name: 'Skill Dock home' })).toHaveAttribute('href', '/');
    expect(screen.getByText('agent skill registry')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Open source under MIT' })).toHaveAttribute(
      'href',
      'https://github.com/ZenlixAI/skill-dock/blob/master/LICENSE'
    );
    expect(within(primary).getByRole('link', { name: 'How it works' })).toHaveAttribute(
      'href',
      '/#how-it-works'
    );
    expect(within(primary).getByRole('link', { name: 'Quick start' })).toHaveAttribute(
      'href',
      '/#quick-start'
    );
    expect(screen.getByRole('link', { name: 'Skill Dock on GitHub' })).toHaveAttribute(
      'href',
      'https://github.com/ZenlixAI/skill-dock'
    );
  });

  it('renders legal pages inside the same shell', () => {
    renderAppShell('/privacy');

    expect(screen.getByRole('contentinfo')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Privacy' })).toHaveAttribute('href', '/privacy');
    expect(screen.getByRole('link', { name: 'Terms' })).toHaveAttribute('href', '/terms');
  });
});
