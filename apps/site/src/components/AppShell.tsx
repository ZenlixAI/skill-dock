import { ArrowUpRight, Github, Menu, X } from 'lucide-react';
import { useEffect, useState } from 'react';
import type { PropsWithChildren } from 'react';
import { Link, NavLink, useLocation } from 'react-router-dom';
import { DOCS_URL, LICENSE_URL, REPO_URL } from '../lib/content';
import { useI18n } from '../lib/i18n';
import { getRouteMeta } from '../lib/routeMeta';
import clsx from '../vendor/clsx';
import { ThreadsBackground } from './ThreadsBackground';
import { DocumentMeta } from './DocumentMeta';
import { LanguageToggle } from './LanguageToggle';
import { ScrollProgress } from './ScrollProgress';
import { ThemeToggle } from './ThemeToggle';

function useHashScroll() {
  const location = useLocation();

  useEffect(() => {
    if (!location.hash) return;
    const id = decodeURIComponent(location.hash.slice(1));
    const element = document.getElementById(id);
    if (!element) return;
    requestAnimationFrame(() => {
      const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
      element.scrollIntoView({ behavior: reduce ? 'auto' : 'smooth', block: 'start' });
    });
  }, [location]);
}

export function AppShell({ children }: PropsWithChildren) {
  const location = useLocation();
  const routeMeta = getRouteMeta(location.pathname);
  const { content, path } = useI18n();
  const { shell } = content;
  const [menuOpen, setMenuOpen] = useState(false);
  useHashScroll();

  useEffect(() => {
    setMenuOpen(false);
  }, [location.pathname]);

  return (
    <div className="site-shell">
      <ThreadsBackground />
      <DocumentMeta description={routeMeta.description} title={routeMeta.title} />
      <ScrollProgress />

      <a className="skip-link" href="#main-content">
        {shell.skipToContent}
      </a>

      <header className="site-header">
        <div className="site-header__inner">
          <Link aria-label={shell.homeLabel} className="brand" to={path('/')}>
            <span className="brand__copy">
              <strong>Skill Dock</strong>
              <small>{shell.brandTagline}</small>
            </span>
          </Link>

          <nav aria-label={shell.primaryNavLabel} className="site-nav">
            {shell.nav.map((item) => (
              <Link className="site-nav__link" key={item.path} to={path(item.path)}>
                {item.label}
              </Link>
            ))}
          </nav>

          <div className="site-header__actions">
            <LanguageToggle />
            <ThemeToggle />
            <a
              aria-label={shell.githubLabel}
              className="icon-button"
              href={REPO_URL}
              rel="noreferrer"
              target="_blank"
            >
              <Github aria-hidden="true" size={17} strokeWidth={1.8} />
            </a>
            <Link className="button button--small button--primary" to={path('/#quick-start')}>
              {shell.quickStart}
            </Link>
            <button
              aria-controls="mobile-navigation"
              aria-expanded={menuOpen}
              aria-label={menuOpen ? shell.closeMenu : shell.openMenu}
              className="icon-button mobile-toggle"
              onClick={() => setMenuOpen((open) => !open)}
              type="button"
            >
              {menuOpen ? (
                <X aria-hidden="true" size={18} strokeWidth={1.8} />
              ) : (
                <Menu aria-hidden="true" size={18} strokeWidth={1.8} />
              )}
            </button>
          </div>
        </div>

        <div
          className={clsx('mobile-menu', menuOpen && 'mobile-menu--open')}
          id="mobile-navigation"
        >
          <nav aria-label={shell.mobileNavLabel} className="mobile-menu__nav">
            {shell.nav.map((item) => (
              <Link className="mobile-menu__link" key={item.path} to={path(item.path)}>
                {item.label}
              </Link>
            ))}
            <a className="mobile-menu__link" href={DOCS_URL} rel="noreferrer" target="_blank">
              {shell.docs}
              <ArrowUpRight aria-hidden="true" size={14} />
            </a>
            <a className="mobile-menu__link" href={REPO_URL} rel="noreferrer" target="_blank">
              {shell.github}
              <ArrowUpRight aria-hidden="true" size={14} />
            </a>
          </nav>
        </div>
      </header>

      <main id="main-content" tabIndex={-1}>
        {children}
      </main>

      <footer className="site-footer">
        <div className="site-footer__top">
          <div className="site-footer__brand">
            <div>
              <strong>Skill Dock</strong>
              <p>{shell.footer.tagline}</p>
            </div>
          </div>

          <div className="site-footer__links">
            <div>
              <h2>{shell.footer.productHeading}</h2>
              <ul>
                <li>
                  <Link to={path('/registry')}>{shell.footer.registry}</Link>
                </li>
                <li>
                  <Link to={path('/#how-it-works')}>{shell.footer.howItWorks}</Link>
                </li>
                <li>
                  <Link to={path('/#architecture')}>{shell.footer.architecture}</Link>
                </li>
                <li>
                  <Link to={path('/#quick-start')}>{shell.footer.quickStart}</Link>
                </li>
                <li>
                  <Link to={path('/#faq')}>{shell.footer.faq}</Link>
                </li>
              </ul>
            </div>
            <div>
              <h2>{shell.footer.projectHeading}</h2>
              <ul>
                <li>
                  <a href={REPO_URL} rel="noreferrer" target="_blank">
                    {shell.github}
                    <ArrowUpRight aria-hidden="true" size={12} />
                  </a>
                </li>
                <li>
                  <a href={DOCS_URL} rel="noreferrer" target="_blank">
                    {shell.footer.docs}
                    <ArrowUpRight aria-hidden="true" size={12} />
                  </a>
                </li>
                <li>
                  <a href={LICENSE_URL} rel="noreferrer" target="_blank">
                    {shell.footer.license}
                    <ArrowUpRight aria-hidden="true" size={12} />
                  </a>
                </li>
              </ul>
            </div>
            <div>
              <h2>{shell.footer.legalHeading}</h2>
              <ul>
                <li>
                  <NavLink to={path('/privacy')}>{shell.footer.privacy}</NavLink>
                </li>
                <li>
                  <NavLink to={path('/terms')}>{shell.footer.terms}</NavLink>
                </li>
              </ul>
            </div>
          </div>
        </div>

        <div className="site-footer__bottom">
          <span>{shell.footer.copyright}</span>
          <a className="site-footer__license" href={LICENSE_URL} rel="noreferrer" target="_blank">
            {shell.footer.openSource}
          </a>
        </div>
      </footer>
    </div>
  );
}
