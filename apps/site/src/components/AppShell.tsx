import type { PropsWithChildren } from 'react';
import { Link, NavLink, useLocation } from 'react-router-dom';
import { portalNavItems } from '../lib/navigation';
import { getRouteMeta } from '../lib/routeMeta';
import { getRouteTheme } from '../lib/routeTheme';
import clsx from '../vendor/clsx';
import { DocumentMeta } from './DocumentMeta';

export function AppShell({ children }: PropsWithChildren) {
  const location = useLocation();
  const routeTheme = getRouteTheme(location.pathname);
  const routeMeta = getRouteMeta(location.pathname);

  return (
    <div className={clsx('app-shell', `app-shell--${routeTheme.tone}`, `app-shell--${routeTheme.accent}`)}>
      <DocumentMeta description={routeMeta.description} title={routeMeta.title} />

      <div className="shell-backdrop" aria-hidden="true">
        <div className="shell-backdrop__grid" />
        <div className="glow-orb-primary" style={{ top: '-220px', right: '-140px' }} />
        <div className="glow-orb-accent" style={{ top: '220px', left: '-180px' }} />
      </div>

      <a className="skip-link" href="#main-content">
        Skip to main content
      </a>

      <header className="shell-header">
        <div className="shell-header__inner">
          <Link to="/" className="brand-lockup" aria-label="Skill Registry home">
            <div className="brand-lockup__mark">SR</div>
            <div className="brand-lockup__copy">
              <span className="brand-lockup__name" role="heading" aria-level={2}>Skill Registry</span>
            </div>
          </Link>

          <nav aria-label="Primary" className="shell-header__nav" id="primary-navigation">
            {portalNavItems.map((item) => (
              <NavLink
                className={({ isActive }) => clsx('nav-chip', isActive && 'nav-chip--active')}
                key={item.to}
                to={item.to}
              >
                <span>{item.label}</span>
              </NavLink>
            ))}
            <a className="nav-chip nav-chip--external" href="https://github.com/ZenlixAI/skill-dock#docker-compose" rel="noreferrer" target="_blank">
              <span>Installation</span>
            </a>
          </nav>
        </div>
      </header>

      <main className="page-shell page-shell--enter" id="main-content" key={location.pathname} tabIndex={-1}>
        {children}
      </main>

      <footer className="shell-footer">
        <div className="shell-footer__inner">
          <div className="shell-footer__brand">
            <div className="brand-lockup__mark brand-lockup__mark--footer">SR</div>
            <span className="shell-footer__name">Skill Registry</span>
          </div>

          <div className="shell-footer__groups">
            <div>
              <h2>Browse</h2>
              <ul>
                <li><Link to="/">Home</Link></li>
                <li><Link to="/features">Features</Link></li>
                <li><a href="https://github.com/ZenlixAI/skill-dock#docker-compose" rel="noreferrer" target="_blank">Installation</a></li>
              </ul>
            </div>
            <div>
              <h2>Legal</h2>
              <ul>
                <li><Link to="/privacy">Privacy</Link></li>
                <li><Link to="/terms">Terms</Link></li>
              </ul>
            </div>
          </div>
        </div>

        <div className="shell-footer__bottom">
          <span>© 2026 Skill Registry · MIT License</span>
          <span>Install from the terminal. Govern releases from the catalog.</span>
        </div>
      </footer>
    </div>
  );
}
