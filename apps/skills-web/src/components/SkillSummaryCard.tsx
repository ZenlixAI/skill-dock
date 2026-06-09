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

function getVersionLabel(versionStatus: SkillSummary['latestVersionStatus']) {
  if (versionStatus === 'published') {
    return '已发布';
  }

  if (versionStatus === 'deprecated') {
    return '已弃用';
  }

  return '未知';
}

function getSourceTypeLabel(sourceType: SkillSummary['sourceType']) {
  if (sourceType === 'native') {
    return '原生';
  }

  if (sourceType === 'imported') {
    return '导入';
  }

  return '平台创建';
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
      </div>

      <p className="skill-summary-card__description">{skill.description}</p>

      {curationReason ? <p className="skill-summary-card__reason">{curationReason}</p> : null}

      <div className="skill-summary-card__install">
        <StatusPill tone={getVersionTone(skill.latestVersionStatus)}>{getVersionLabel(skill.latestVersionStatus)}</StatusPill>
        <StatusPill tone="accent">{getSourceTypeLabel(skill.sourceType)}</StatusPill>
        {skill.latestVersion ? <StatusPill>最新 {skill.latestVersion}</StatusPill> : null}
      </div>

      <div className="skill-summary-card__footer">
        <span className="skill-summary-card__provenance">{namespace} · 更新于 {formatDate(skill.updatedAt)}</span>
        <div className="skill-summary-card__actions">
          <Link className="button button--primary" to={`/skills/${encodeURIComponent(skill.packageName)}`}>
            查看技能
          </Link>
        </div>
      </div>
    </article>
  );
}
