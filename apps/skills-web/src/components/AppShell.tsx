import type { PropsWithChildren } from 'react';
import { NavLink } from 'react-router-dom';
import { getRegistryClientConfig } from '../api/client';

const navItems = [
  { to: '/', label: 'Overview' },
  { to: '/authoring', label: 'Authoring Flow' },
  { to: '/catalog', label: 'Catalog' },
  { to: '/import', label: 'Import & Package' },
  { to: '/governance', label: 'Governance' },
  { to: '/audit', label: 'Audit' },
];

export function AppShell({ children }: PropsWithChildren) {
  const config = getRegistryClientConfig();

  return (
    <div className="app-shell">
      <div className="app-shell__bg app-shell__bg--one" />
      <div className="app-shell__bg app-shell__bg--two" />
      <header className="topbar">
        <div className="brand">
          <div className="brand__mark">SR</div>
          <div>
            <h1>Skill Registry Web</h1>
            <p>Catalog, packaging, release review and governance in one registry-first console.</p>
          </div>
        </div>
        <nav className="topbar__nav" aria-label="Primary">
          {navItems.map((item) => (
            <NavLink
              className={({ isActive }) => (isActive ? 'nav-chip nav-chip--active' : 'nav-chip')}
              key={item.to}
              to={item.to}
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
      </header>
      <div className="env-banner">
        <Status mode={config.mode} />
        {config.baseUrl ? <span>{config.baseUrl}</span> : <span>using local mock client</span>}
        {config.role ? <span>role {config.role}</span> : null}
        {config.hasToken ? <span>bearer token</span> : null}
      </div>
      <main className="page-shell">{children}</main>
    </div>
  );
}

function Status({ mode }: { mode: 'mock' | 'real' }) {
  return <strong>{mode === 'real' ? 'Real API mode' : 'Mock mode'}</strong>;
}
