import { Link } from 'react-router-dom';
import type { CuratedSkillSummary, SkillSummary } from '../api/types';
import { formatDate } from '../lib/format';
import { StatusPill } from './StatusPill';

type SkillSummaryCardProps = {
  skill: SkillSummary | CuratedSkillSummary;
  curationReason?: string;
};

function getVersionTone(versionStatus: SkillSummary['latestVersionStatus']) {
  if (versionStatus === 'published') {
    return 'good';
  }

  if (versionStatus === 'deprecated') {
    return 'warn';
  }

  return 'bad';
}

export function SkillSummaryCard({ skill, curationReason }: SkillSummaryCardProps) {
  const namespace = skill.packageName.split('/')[0] || skill.packageName;

  return (
    <article className="skill-summary-card">
      <div className="skill-summary-card__header">
        <div>
          <strong>{skill.displayName}</strong>
          <span>{skill.packageName}</span>
        </div>
        <StatusPill tone={getVersionTone(skill.latestVersionStatus)}>{skill.latestVersionStatus || 'unknown'}</StatusPill>
      </div>

      <p className="skill-summary-card__description">{skill.description}</p>

      <div className="skill-summary-card__meta">
        <StatusPill tone="accent">{skill.sourceType}</StatusPill>
        {skill.latestVersion ? <StatusPill>latest {skill.latestVersion}</StatusPill> : null}
        <span className="skill-summary-card__provenance">{namespace} · updated {formatDate(skill.updatedAt)}</span>
      </div>

      {curationReason ? <p className="skill-summary-card__reason">{curationReason}</p> : null}

      <div className="skill-summary-card__actions">
        <Link className="button button--primary" to={`/skills/${encodeURIComponent(skill.packageName)}`}>
          Open skill
        </Link>
      </div>
    </article>
  );
}
