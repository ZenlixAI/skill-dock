import type { WarningDetail } from '../api/types';
import { StatusPill } from './StatusPill';

interface WarningStackProps {
  warnings: WarningDetail[];
}

export function WarningStack({ warnings }: WarningStackProps) {
  if (warnings.length === 0) {
    return (
      <div className="empty-box">
        <strong>暂无告警</strong>
        <p>当前预览下，这条导入链路没有结构性告警。</p>
      </div>
    );
  }

  return (
    <div className="warning-stack">
      {warnings.map((warning) => (
        <article className="warning-card" key={warning.code}>
          <div className="warning-card__top">
            <StatusPill tone={warning.severity === 'high-risk' ? 'bad' : 'warn'}>
              {warning.severity}
            </StatusPill>
            <StatusPill>{warning.code}</StatusPill>
          </div>
          <h3>{warning.title}</h3>
          <p>{warning.message}</p>
          {warning.suggestedAction ? (
            <p className="warning-card__action">
              <strong>建议动作</strong> {warning.suggestedAction}
            </p>
          ) : null}
        </article>
      ))}
    </div>
  );
}
