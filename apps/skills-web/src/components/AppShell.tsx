import type { PropsWithChildren } from 'react';
import { motion } from 'framer-motion';
import { Activity, AppWindow, BookOpenText, Boxes, ScanSearch, ShieldAlert } from 'lucide-react';
import { NavLink, useLocation } from 'react-router-dom';
import { getRegistryClientConfig } from '../api/client';
import { getRouteTheme } from '../lib/routeTheme';
import clsx from '../vendor/clsx';
import { StatusPill } from './StatusPill';

const navItems = [
  { to: '/', label: 'Overview', icon: AppWindow },
  { to: '/catalog', label: 'Catalog', icon: ScanSearch },
  { to: '/authoring', label: 'Authoring', icon: BookOpenText },
  { to: '/import', label: 'Import', icon: Boxes },
  { to: '/governance', label: 'Governance', icon: ShieldAlert },
  { to: '/audit', label: 'Audit', icon: Activity },
];

export function AppShell({ children }: PropsWithChildren) {
  const config = getRegistryClientConfig();
  const location = useLocation();
  const routeTheme = getRouteTheme(location.pathname);

  return (
    <div className={clsx('app-shell', `app-shell--${routeTheme.tone}`, `app-shell--${routeTheme.accent}`)}>
      <div className="app-shell__halo app-shell__halo--ember" />
      <div className="app-shell__halo app-shell__halo--moss" />
      <div className="app-shell__grain" />

      <header className="shell-header">
        <div className="shell-header__inner">
          <div className="brand-lockup">
            <div className="brand-lockup__mark">SR</div>
            <div className="brand-lockup__copy">
              <div className="brand-lockup__eyebrow">Registry Core MVP</div>
              <h1>Skill Registry Web</h1>
              <p>Catalog, packaging, release review, and governance in one registry-first operating surface.</p>
            </div>
          </div>

          <div className="shell-header__meta">
            <StatusPill tone={routeTheme.tone === 'frontstage' ? 'accent' : 'default'}>
              {routeTheme.tone === 'frontstage' ? 'Frontstage' : 'Backstage'}
            </StatusPill>
            <Status mode={config.mode} />
          </div>
        </div>

        <div className="shell-nav">
          <nav className="shell-nav__rail" aria-label="Primary">
            {navItems.map((item) => {
              const Icon = item.icon;

              return (
                <NavLink
                  className={({ isActive }) => clsx('nav-chip', isActive && 'nav-chip--active')}
                  key={item.to}
                  to={item.to}
                >
                  <Icon aria-hidden="true" size={16} strokeWidth={2} />
                  <span>{item.label}</span>
                </NavLink>
              );
            })}
          </nav>

          <div className="env-banner">
            {config.baseUrl ? <span>{config.baseUrl}</span> : <span>using local mock client</span>}
            {config.role ? <span>role {config.role}</span> : null}
            {config.actor ? <span>actor {config.actor}</span> : null}
            {config.hasToken ? <span>bearer token</span> : null}
          </div>
        </div>
      </header>

      <motion.main
        animate={{ opacity: 1, y: 0 }}
        className="page-shell"
        initial={{ opacity: 0, y: 12 }}
        key={location.pathname}
        transition={{ duration: 0.28, ease: 'easeOut' }}
      >
        {children}
      </motion.main>
    </div>
  );
}

function Status({ mode }: { mode: 'mock' | 'real' }) {
  return <StatusPill tone={mode === 'real' ? 'good' : 'warn'}>{mode === 'real' ? 'Real API mode' : 'Mock mode'}</StatusPill>;
}
