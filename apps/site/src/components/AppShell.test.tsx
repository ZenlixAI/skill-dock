import { cleanup, render, screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { LocaleProvider } from '../lib/i18n';
import { AppShell } from './AppShell';

afterEach(() => {
  cleanup();
});

function renderAppShell(pathname: string) {
  return render(
    <MemoryRouter initialEntries={[pathname]}>
      <LocaleProvider>
        <AppShell>
          <div>content</div>
        </AppShell>
      </LocaleProvider>
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

  it('renders the Chinese shell and localized links for /zh routes', () => {
    renderAppShell('/zh/privacy');

    expect(screen.getByRole('link', { name: '跳到主要内容' })).toHaveAttribute(
      'href',
      '#main-content'
    );
    expect(screen.getByRole('link', { name: 'Skill Dock 首页' })).toHaveAttribute('href', '/zh');
    expect(screen.getByText('Agent 技能 registry')).toBeInTheDocument();

    const primary = screen.getByRole('navigation', { name: '主导航' });
    expect(within(primary).getByRole('link', { name: '技能目录' })).toHaveAttribute(
      'href',
      '/zh/registry'
    );
    expect(within(primary).getByRole('link', { name: '工作方式' })).toHaveAttribute(
      'href',
      '/zh#how-it-works'
    );
    expect(within(primary).getByRole('link', { name: '常见问题' })).toHaveAttribute(
      'href',
      '/zh#faq'
    );

    expect(screen.getByRole('button', { name: '切换语言，当前：简体中文' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '隐私' })).toHaveAttribute('href', '/zh/privacy');
    expect(screen.getByRole('link', { name: '条款' })).toHaveAttribute('href', '/zh/terms');
    expect(screen.getByRole('link', { name: 'MIT 开源协议' })).toHaveAttribute(
      'href',
      'https://github.com/ZenlixAI/skill-dock/blob/master/LICENSE'
    );
  });
});
