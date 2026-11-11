import { ArrowUpRight, Check, Copy, RefreshCw, X } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import type { ExternalSkill, ExternalSkillDetail } from '../lib/skills-directory-api';
import { fetchSkillsShDetail, skillsShSkillURL, skillsShSourceURL } from '../lib/skills-directory-api';

interface ExternalSkillDrawerProps {
  open: boolean;
  source: string;
  skillId: string;
  onClose: () => void;
  onSelectRelated: (skill: ExternalSkill) => void;
}

export function ExternalSkillDrawer({
  open,
  source,
  skillId,
  onClose,
  onSelectRelated,
}: ExternalSkillDrawerProps) {
  const [detail, setDetail] = useState<ExternalSkillDetail | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [reloadToken, setReloadToken] = useState(0);
  const closeRef = useRef<HTMLButtonElement>(null);
  const bodyRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open || !source || !skillId) return;
    const controller = new AbortController();
    setLoading(true);
    setError(null);
    setDetail(null);

    fetchSkillsShDetail(source, skillId, controller.signal)
      .then((next) => setDetail(next))
      .catch((cause: unknown) => {
        if (cause instanceof DOMException && cause.name === 'AbortError') return;
        setError(cause instanceof Error ? cause.message : 'Could not load this skill.');
      })
      .finally(() => setLoading(false));

    return () => controller.abort();
  }, [open, source, skillId, reloadToken]);

  useEffect(() => {
    if (bodyRef.current) bodyRef.current.scrollTop = 0;
  }, [source, skillId]);

  useEffect(() => {
    if (!open) return;
    const previous = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const originalOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    requestAnimationFrame(() => closeRef.current?.focus());

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', onKeyDown);

    return () => {
      document.removeEventListener('keydown', onKeyDown);
      document.body.style.overflow = originalOverflow;
      previous?.focus();
    };
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      className="registry-drawer"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
      role="presentation"
    >
      <div
        aria-label={detail ? undefined : 'External skill details'}
        aria-labelledby={detail ? 'external-skill-title' : undefined}
        aria-modal="true"
        className="registry-drawer__panel"
        onKeyDown={(event) => {
          if (event.key !== 'Tab') return;
          const focusable = event.currentTarget.querySelectorAll<HTMLElement>(
            'a[href], button:not([disabled]), [tabindex]:not([tabindex="-1"])'
          );
          if (focusable.length === 0) return;
          const first = focusable[0];
          const last = focusable[focusable.length - 1];
          if (event.shiftKey && document.activeElement === first) {
            event.preventDefault();
            last.focus();
          } else if (!event.shiftKey && document.activeElement === last) {
            event.preventDefault();
            first.focus();
          }
        }}
        role="dialog"
      >
        <header className="registry-drawer__header">
          <span className="registry-source registry-source--external">
            skills.sh
            <ArrowUpRight aria-hidden="true" size={12} />
          </span>
          <button
            aria-label="Close skill details"
            className="icon-button registry-drawer__close"
            onClick={onClose}
            ref={closeRef}
            type="button"
          >
            <X aria-hidden="true" size={18} strokeWidth={1.8} />
          </button>
        </header>

        <div aria-busy={loading} className="registry-drawer__body" ref={bodyRef}>
          {loading ? <DrawerSkeleton /> : null}
          {error ? (
            <div className="registry-state registry-state--error">
              <p>{error}</p>
              <button
                className="button button--secondary button--small"
                onClick={() => setReloadToken((value) => value + 1)}
                type="button"
              >
                <RefreshCw aria-hidden="true" size={14} />
                Retry
              </button>
            </div>
          ) : null}
          {detail ? (
            <ExternalDetail
              detail={detail}
              onSelectRelated={onSelectRelated}
            />
          ) : null}
        </div>
      </div>
    </div>
  );
}

function ExternalDetail({
  detail,
  onSelectRelated,
}: {
  detail: ExternalSkillDetail;
  onSelectRelated: (skill: ExternalSkill) => void;
}) {
  const displayName = detail.displayName && detail.displayName !== detail.name ? detail.displayName : null;
  const command = detail.installCommand || '';
  const skillURL = detail.url || skillsShSkillURL(detail.source, detail.skillId);

  return (
    <div className="registry-drawer__content">
      <div>
        <a
          className="registry-drawer__source-line"
          href={skillsShSourceURL(detail.source)}
          rel="noreferrer"
          target="_blank"
        >
          {detail.source}
          <span className="sr-only"> (opens in a new tab)</span>
        </a>
        <h2 className="registry-drawer__title" id="external-skill-title">
          {detail.name}
        </h2>
        {displayName ? <p className="registry-drawer__display-name">{displayName}</p> : null}
        {detail.isOfficial ? <span className="registry-official">Official</span> : null}
        {detail.topics && detail.topics.length > 0 ? (
          <div className="registry-topics">
            {detail.topics.map((topic) => (
              <span className="registry-topic" key={topic}>
                {topic}
              </span>
            ))}
          </div>
        ) : null}
        {detail.description ? <p className="registry-drawer__description">{detail.description}</p> : null}
      </div>

      <section className="registry-drawer__section">
        <h3>Installation</h3>
        {command ? (
          <CopyCommand command={command} />
        ) : (
          <p className="registry-drawer__note">Install command unavailable.</p>
        )}
        <p className="registry-drawer__note">
          Fetches from the source repository. Refresh with <code>skills sync</code>.
        </p>
      </section>

      <section className="registry-drawer__stats">
        <div>
          <span className="registry-drawer__stat-label">Installs</span>
          <strong>{formatInstalls(detail.installs)}</strong>
        </div>
        {detail.weeklyInstalls && detail.weeklyInstalls.length > 1 ? (
          <WeeklySparkline values={detail.weeklyInstalls} />
        ) : null}
      </section>

      {detail.summaryHtml ? (
        <section className="registry-drawer__section">
          <h3>Summary</h3>
          <div className="registry-prose" dangerouslySetInnerHTML={{ __html: detail.summaryHtml }} />
        </section>
      ) : null}

      {detail.contentHtml ? (
        <section className="registry-drawer__section">
          <h3>SKILL.md</h3>
          <div className="registry-prose" dangerouslySetInnerHTML={{ __html: detail.contentHtml }} />
          <p className="registry-drawer__note">Rendered from the skill&apos;s SKILL.md.</p>
        </section>
      ) : null}

      {detail.relatedSkills && detail.relatedSkills.length > 0 ? (
        <section className="registry-drawer__section">
          <h3>Related skills</h3>
          <ul className="registry-related">
            {detail.relatedSkills.map((skill) => (
              <li key={skill.id}>
                <button onClick={() => onSelectRelated(skill)} type="button">
                  <span>
                    <strong>{skill.name}</strong>
                    <small>{skill.source}</small>
                  </span>
                  {skill.installs > 0 ? (
                    <span className="registry-related__meta">{formatInstalls(skill.installs)} installs</span>
                  ) : null}
                </button>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      <a className="button button--secondary registry-drawer__external" href={skillURL} rel="noreferrer" target="_blank">
        View on skills.sh
        <span className="sr-only"> (opens in a new tab)</span>
        <ArrowUpRight aria-hidden="true" size={15} />
      </a>
    </div>
  );
}

function CopyCommand({ command }: { command: string }) {
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(command);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      setCopied(false);
    }
  };

  return (
    <button className="registry-command" onClick={copy} title="Copy command to clipboard" type="button">
      <code>
        <span aria-hidden="true">$</span> {command}
      </code>
      {copied ? (
        <span className="registry-command__status">
          <Check aria-hidden="true" size={14} /> Copied
        </span>
      ) : (
        <Copy aria-hidden="true" size={14} />
      )}
    </button>
  );
}

function WeeklySparkline({ values }: { values: number[] }) {
  const width = 160;
  const height = 48;
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;
  const points = values
    .map((value, index) => {
      const x = values.length === 1 ? 0 : (index / (values.length - 1)) * width;
      const y = height - ((value - min) / range) * (height - 6) - 3;
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    })
    .join(' ');

  return (
    <div className="registry-sparkline">
      <svg aria-label={`Weekly installs: ${values.join(', ')}`} role="img" viewBox={`0 0 ${width} ${height}`}>
        <polyline fill="none" points={points} stroke="currentColor" strokeWidth="2" />
      </svg>
      <span>8-week installs</span>
    </div>
  );
}

function DrawerSkeleton() {
  return (
    <div className="registry-drawer__skeleton">
      <span className="registry-skeleton registry-skeleton--short" />
      <span className="registry-skeleton registry-skeleton--title" />
      <span className="registry-skeleton" />
      <span className="registry-skeleton" />
      <span className="registry-skeleton registry-skeleton--command" />
      <span className="registry-skeleton" />
    </div>
  );
}

export function formatInstalls(value: number): string {
  return new Intl.NumberFormat('en-US', {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(value);
}
