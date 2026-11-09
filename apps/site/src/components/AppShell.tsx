import { ArrowUpRight, Github, Menu, X } from 'lucide-react';
import { useEffect, useState } from 'react';
import type { PropsWithChildren } from 'react';
import { Link, NavLink, useLocation } from 'react-router-dom';
import { DOCS_URL, LICENSE_URL, NAV_ITEMS, REPO_URL } from '../lib/content';
import { getRouteMeta } from '../lib/routeMeta';
import clsx from '../vendor/clsx';
import { BrandMark } from './BrandMark';
import { DocumentMeta } from './DocumentMeta';
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
  const [menuOpen, setMenuOpen] = useState(false);
  useHashScroll();

  useEffect(() => {
    setMenuOpen(false);
  }, [location.pathname]);

  return (
    <div className="site-shell">
      <DocumentMeta description={routeMeta.description} title={routeMeta.title} />
      <ScrollProgress />

      <a className="skip-link" href="#main-content">
        Skip to main content
      </a>

      <header className="site-header">
        <div className="site-header__inner">
          <Link aria-label="Skill Dock home" className="brand" to="/">
            <span className="brand__mark">
              <BrandMark />
            </span>
            <span className="brand__copy">
              <strong>Skill Dock</strong>
              <small>agent skill registry</small>
            </span>
          </Link>

          <nav aria-label="Primary" className="site-nav">
            {NAV_ITEMS.map((item) => (
              <Link className="site-nav__link" key={item.href} to={item.href}>
                {item.label}
              </Link>
            ))}
          </nav>

          <div className="site-header__actions">
            <ThemeToggle />
            <a
              aria-label="Skill Dock on GitHub"
              className="icon-button"
              href={REPO_URL}
              rel="noreferrer"
              target="_blank"
            >
              <Github aria-hidden="true" size={17} strokeWidth={1.8} />
            </a>
            <Link className="button button--small button--primary" to="/#quick-start">
              Quick start
            </Link>
            <button
              aria-controls="mobile-navigation"
              aria-expanded={menuOpen}
              aria-label={menuOpen ? 'Close menu' : 'Open menu'}
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
          <nav aria-label="Mobile" className="mobile-menu__nav">
            {NAV_ITEMS.map((item) => (
              <Link className="mobile-menu__link" key={item.href} to={item.href}>
                {item.label}
              </Link>
            ))}
            <a className="mobile-menu__link" href={DOCS_URL} rel="noreferrer" target="_blank">
              Docs
              <ArrowUpRight aria-hidden="true" size={14} />
            </a>
            <a className="mobile-menu__link" href={REPO_URL} rel="noreferrer" target="_blank">
              GitHub
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
            <span className="brand__mark brand__mark--footer">
              <BrandMark />
            </span>
            <div>
              <strong>Skill Dock</strong>
              <p>A self-hosted registry for agent skills.</p>
            </div>
          </div>

          <div className="site-footer__links">
            <div>
              <h2>Product</h2>
              <ul>
                <li>
                  <Link to="/#how-it-works">How it works</Link>
                </li>
                <li>
                  <Link to="/#architecture">Architecture</Link>
                </li>
                <li>
                  <Link to="/#quick-start">Quick start</Link>
                </li>
                <li>
                  <Link to="/#faq">FAQ</Link>
                </li>
              </ul>
            </div>
            <div>
              <h2>Project</h2>
              <ul>
                <li>
                  <a href={REPO_URL} rel="noreferrer" target="_blank">
                    GitHub
                    <ArrowUpRight aria-hidden="true" size={12} />
                  </a>
                </li>
                <li>
                  <a href={DOCS_URL} rel="noreferrer" target="_blank">
                    Docs
                    <ArrowUpRight aria-hidden="true" size={12} />
                  </a>
                </li>
                <li>
                  <a href={LICENSE_URL} rel="noreferrer" target="_blank">
                    License
                    <ArrowUpRight aria-hidden="true" size={12} />
                  </a>
                </li>
              </ul>
            </div>
            <div>
              <h2>Legal</h2>
              <ul>
                <li>
                  <NavLink to="/privacy">Privacy</NavLink>
                </li>
                <li>
                  <NavLink to="/terms">Terms</NavLink>
                </li>
              </ul>
            </div>
          </div>
        </div>

        <div className="site-footer__bottom">
          <span>© 2026 Skill Dock</span>
          <a className="site-footer__license" href={LICENSE_URL} rel="noreferrer" target="_blank">
            Open source under MIT
          </a>
        </div>
      </footer>
    </div>
  );
}
