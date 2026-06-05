import type { PropsWithChildren } from 'react';
import { motion } from 'framer-motion';
import { Activity, AppWindow, BookOpenText, Boxes, ScanSearch, ShieldAlert } from 'lucide-react';
import { NavLink, useLocation, useNavigate } from 'react-router-dom';
import { getRegistryClientConfig, logoutSession } from '../api/client';
import { hasAction } from '../api/session';
import { getRouteTheme } from '../lib/routeTheme';
import { useRegistrySession } from '../session/SessionProvider';
import clsx from '../vendor/clsx';
import { StatusPill } from './StatusPill';

const navItems = [
  { to: '/', label: '首页', icon: AppWindow },
  { to: '/catalog', label: '目录', icon: ScanSearch },
  { to: '/workbench', label: '工作台', icon: AppWindow },
  { to: '/workbench/authoring', label: '作者台', icon: BookOpenText, requiredAction: 'packaging.prepare' },
  { to: '/workbench/import', label: '导入', icon: Boxes, requiredAction: 'packaging.prepare' },
  { to: '/workbench/governance', label: '治理', icon: ShieldAlert, requiredAction: 'catalog.governance.skill.apply' },
  { to: '/workbench/audit', label: '审计', icon: Activity },
];

export function AppShell({ children }: PropsWithChildren) {
  const config = getRegistryClientConfig();
  const location = useLocation();
  const navigate = useNavigate();
  const routeTheme = getRouteTheme(location.pathname);
  const { session, refreshSession } = useRegistrySession();
  const runtimeUsesLogin = config.mode === 'real' && !config.hasToken && !config.role;
  const visibleNavItems = navItems.filter((item) =>
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
      <div className="app-shell__halo app-shell__halo--ember" />
      <div className="app-shell__halo app-shell__halo--moss" />
      <div className="app-shell__grain" />

      <header className="shell-header">
        <div className="shell-header__inner">
          <div className="brand-lockup">
            <div className="brand-lockup__mark">SR</div>
            <div className="brand-lockup__copy">
              <div className="brand-lockup__eyebrow">Skill Registry</div>
              <h1>Skill Registry Web</h1>
              <p>发现、审阅和发布技能。</p>
            </div>
          </div>

          <div className="shell-header__meta">
            {runtimeUsesLogin && session.data ? <StatusPill tone="accent">当前用户：{session.data.actor}</StatusPill> : null}
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
          <nav className="shell-nav__rail" aria-label="Primary">
            {visibleNavItems.map((item) => {
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
