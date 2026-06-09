import type { PropsWithChildren } from 'react';
import { motion } from 'framer-motion';
import { NavLink, Link, useLocation, useNavigate } from 'react-router-dom';
import { getRegistryClientConfig, logoutSession } from '../api/client';
import { hasAction } from '../api/session';
import { consoleNavItems, portalNavItems } from '../lib/navigation';
import { getRouteMeta } from '../lib/routeMeta';
import { getRouteTheme } from '../lib/routeTheme';
import { useRegistrySession } from '../session/SessionProvider';
import clsx from '../vendor/clsx';
import { DocumentMeta } from './DocumentMeta';
import { StatusPill } from './StatusPill';

export function AppShell({ children }: PropsWithChildren) {
  const config = getRegistryClientConfig();
  const location = useLocation();
  const navigate = useNavigate();
  const routeTheme = getRouteTheme(location.pathname);
  const routeMeta = getRouteMeta(location.pathname);
  const { session, refreshSession } = useRegistrySession();
  const runtimeUsesLogin = config.mode === 'real' && !config.hasToken && !config.role;
  const visibleConsoleNavItems = consoleNavItems.filter((item) =>
    item.requiredAction ? hasAction(session.data, item.requiredAction) : true
  );

  async function handleLogout() {
    await logoutSession();
    refreshSession();
    navigate('/login', { replace: true });
  }

  async function handleSwitchAccount() {
    const next = `${location.pathname}${location.search}`;
    await logoutSession();
    refreshSession();
    navigate(`/login?next=${encodeURIComponent(next)}`, { replace: true });
  }

  return (
    <div className={clsx('app-shell', `app-shell--${routeTheme.tone}`, `app-shell--${routeTheme.accent}`)}>
      <DocumentMeta description={routeMeta.description} title={routeMeta.title} />
      <a className="skip-link" href="#main-content">
        Skip to content
      </a>
      <div className="app-shell__halo app-shell__halo--ember" />
      <div className="app-shell__halo app-shell__halo--moss" />
      <div className="app-shell__grain" />

      <header className="shell-header">
        <div className="shell-header__inner">
          <div className="brand-lockup">
            <div className="brand-lockup__mark">SR</div>
            <div className="brand-lockup__copy">
              <div className="brand-lockup__eyebrow">Skill Registry</div>
              <h1>Portal and Console</h1>
              <p>Portal handles discovery and installation. Console handles import, authoring, governance, and audit workflows.</p>
            </div>
          </div>

          <div className="shell-header__meta">
            <StatusPill tone="accent">{routeMeta.section === 'console' ? 'Console' : 'Portal'}</StatusPill>
            {runtimeUsesLogin && session.data ? <StatusPill tone="accent">Signed in as {session.data.actor}</StatusPill> : null}
            {runtimeUsesLogin && session.data ? (
              <>
                <button className="button button--secondary" onClick={handleSwitchAccount} type="button">
                  Switch account
                </button>
                <button className="button button--secondary" onClick={handleLogout} type="button">
                  Sign out
                </button>
              </>
            ) : null}
          </div>
        </div>

        <div className="shell-nav">
          <div className="shell-nav__group">
            <span className="shell-nav__label">Portal</span>
            <nav aria-label="Portal" className="shell-nav__rail">
              {portalNavItems.map((item) => {
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
          </div>

          <div className="shell-nav__divider" aria-hidden="true" />

          <div className="shell-nav__group">
            <span className="shell-nav__label">Console</span>
            <nav aria-label="Console" className="shell-nav__rail">
              {visibleConsoleNavItems.map((item) => {
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
          </div>
        </div>
      </header>

      <motion.main
        animate={{ opacity: 1, y: 0 }}
        className="page-shell"
        id="main-content"
        initial={{ opacity: 0, y: 12 }}
        key={location.pathname}
        tabIndex={-1}
        transition={{ duration: 0.28, ease: 'easeOut' }}
      >
        {children}
      </motion.main>

      <footer className="shell-footer">
        <div className="shell-footer__inner">
          <p className="shell-footer__copy">Internal registry surface for package discovery, install flows, and operational governance.</p>
          <div className="shell-footer__links">
            <Link to="/privacy">Privacy</Link>
            <Link to="/terms">Terms</Link>
          </div>
        </div>
      </footer>
    </div>
  );
}
