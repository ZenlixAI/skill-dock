import { cleanup, render, screen } from '@testing-library/react';
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
  it('renders the dark shell with landing navigation and skip link', async () => {
    const { container } = renderAppShell('/docs');

    expect(await screen.findByRole('link', { name: 'Skip to main content' })).toHaveAttribute('href', '#main-content');
    expect(container.firstElementChild).toHaveClass('app-shell', 'app-shell--frontstage', 'app-shell--detail');
    expect(screen.getByRole('heading', { name: 'Skill Registry' })).toBeInTheDocument();
    expect(screen.getAllByRole('link', { name: 'Home' })).toHaveLength(2);
    expect(screen.getAllByRole('link', { name: 'Features' })).toHaveLength(2);
    expect(screen.getAllByRole('link', { name: 'Docs' })).toHaveLength(2);
    expect(screen.getByRole('link', { name: 'GitHub' })).toHaveAttribute('href', 'https://github.com/ZenlixAI/skill-dock');
    expect(screen.queryByRole('link', { name: 'Explore' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'registry' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'governance' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'audit' })).not.toBeInTheDocument();
  });

  it('keeps the shell background and landing accent classes on the home page', async () => {
    const { container } = renderAppShell('/');

    await screen.findByRole('link', { name: 'Skip to main content' });
    expect(container.firstElementChild).toHaveClass('app-shell', 'app-shell--frontstage', 'app-shell--landing');
  });
});
