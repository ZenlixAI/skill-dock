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
        跳到主要内容
      </a>
      <div className="app-shell__halo app-shell__halo--ember" />
      <div className="app-shell__halo app-shell__halo--moss" />
      <div className="app-shell__grain" />

      <header className="shell-header">
        <div className="shell-header__inner">
          <div className="brand-lockup">
            <div className="brand-lockup__mark">SR</div>
            <div className="brand-lockup__copy">
              <div className="brand-lockup__eyebrow">技能注册表</div>
              <h1>门户与控制台</h1>
              <p>门户用于发现和安装。控制台用于导入、编写、治理和审计流程。</p>
            </div>
          </div>

          <div className="shell-header__meta">
            <StatusPill tone="accent">{routeMeta.section === 'console' ? '控制台' : '门户'}</StatusPill>
            {runtimeUsesLogin && session.data ? <StatusPill tone="accent">已登录：{session.data.actor}</StatusPill> : null}
            {runtimeUsesLogin && session.data ? (
              <>
                <button className="button button--secondary" onClick={handleSwitchAccount} type="button">
                  切换账号
                </button>
                <button className="button button--secondary" onClick={handleLogout} type="button">
                  退出登录
                </button>
              </>
            ) : null}
          </div>
        </div>

        <div className="shell-nav">
          <div className="shell-nav__group">
            <span className="shell-nav__label">门户</span>
            <nav aria-label="门户" className="shell-nav__rail">
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
            <span className="shell-nav__label">控制台</span>
            <nav aria-label="控制台" className="shell-nav__rail">
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
          <p className="shell-footer__copy">内部注册表界面，用于包发现、安装流程和运营治理。</p>
          <div className="shell-footer__links">
            <Link to="/privacy">隐私</Link>
            <Link to="/terms">条款</Link>
          </div>
        </div>
      </footer>
    </div>
  );
}
