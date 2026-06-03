import { useMemo, useState } from 'react';
import { listGovernanceChanges } from '../api/client';
import type { GovernanceAction, GovernanceRecord, GovernanceTargetType } from '../api/types';
import { SectionHeading } from '../components/SectionHeading';
import { StatusPill } from '../components/StatusPill';
import { SurfaceCard } from '../components/SurfaceCard';
import { useAsyncData } from '../lib/useAsyncData';
import { formatDate } from '../lib/format';

type GovernanceLayer = {
  key: string;
  title: string;
  description: string;
  items: GovernanceRecord[];
};

function buildGovernanceLayers(records: GovernanceRecord[]): GovernanceLayer[] {
  const blocking = records.filter((item) => item.action === 'yank' || item.action === 'retire');
  const migration = records.filter((item) => item.action === 'deprecate');
  const packageWide = records.filter((item) => item.targetType === 'skill');

  return [
    {
      key: 'blocking',
      title: 'Blocking / hard-stop actions',
      description: '这些动作会直接改变下游是否还能继续装、继续引用或继续宣传这个 skill。',
      items: blocking,
    },
    {
      key: 'migration',
      title: 'Migration / replacement actions',
      description: '这些动作更偏向迁移引导，告诉消费方该去哪个 replacement，而不是立即强阻断。',
      items: migration,
    },
    {
      key: 'package-wide',
      title: 'Package-wide governance',
      description: '这些动作影响的是整个 skill 线，而不是单个版本。',
      items: packageWide,
    },
  ].filter((layer) => layer.items.length > 0);
}

export function GovernancePage() {
  const [packageName, setPackageName] = useState('');
  const [targetType, setTargetType] = useState<GovernanceTargetType | ''>('');
  const [action, setAction] = useState<GovernanceAction | ''>('');

  const governance = useAsyncData(
    () =>
      listGovernanceChanges({
        packageName: packageName.trim() || undefined,
        targetType,
      }),
    [packageName, targetType]
  );

  const filteredItems = useMemo(() => {
    const items = governance.data || [];
    if (!action) {
      return items;
    }
    return items.filter((item) => item.action === action);
  }, [governance.data, action]);

  const layers = buildGovernanceLayers(filteredItems);
  const blockingCount = filteredItems.filter((item) => item.action === 'yank' || item.action === 'retire').length;
  const replacementCount = filteredItems.filter((item) => Boolean(item.replacementRecommendation)).length;
  const versionCount = filteredItems.filter((item) => item.targetType === 'version').length;

  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="Governance"
        title="把 retire / deprecate / yank 从后端状态机翻译成可操作的治理控制台。"
        description="这页现在不只是一条时间线，而是先回答影响面，再让人按 target / action / package 缩小范围。"
      />

      <div className="context-strip">
        <StatusPill tone="accent">{packageName.trim() || 'all packages'}</StatusPill>
        <StatusPill>{targetType || 'all targets'}</StatusPill>
        <StatusPill tone={action === 'yank' || action === 'retire' ? 'bad' : action === 'deprecate' ? 'warn' : undefined}>
          {action || 'all actions'}
        </StatusPill>
        <StatusPill tone="good">{filteredItems.length} records</StatusPill>
      </div>

      <div className="console-strip">
        <article className="console-signal">
          <span>Blocking actions</span>
          <strong>{blockingCount}</strong>
          <p>这些动作会显式改变 install / adoption 的可行性，属于优先关注的治理事件。</p>
        </article>
        <article className="console-signal">
          <span>Replacement guidance</span>
          <strong>{replacementCount}</strong>
          <p>这些事件已经给出了 replacement recommendation，更适合做迁移引导而不是只给 warning。</p>
        </article>
        <article className="console-signal">
          <span>Version-scoped</span>
          <strong>{versionCount}</strong>
          <p>version 级治理通常对应 release 线上的局部调整，不一定意味着整个 skill 线应该退场。</p>
        </article>
      </div>

      <div className="two-column two-column--wide-right">
        <SurfaceCard className="surface-card--warm" title="Filters" description="先缩小治理面，再读时间线。">
          <label className="field">
            <span>Package name</span>
            <input
              onChange={(event) => setPackageName(event.target.value)}
              placeholder="@platform.skill/pdf-skill"
              value={packageName}
            />
          </label>
          <label className="field">
            <span>Target type</span>
            <select value={targetType} onChange={(event) => setTargetType(event.target.value as GovernanceTargetType | '')}>
              <option value="">All</option>
              <option value="skill">skill</option>
              <option value="version">version</option>
            </select>
          </label>
          <label className="field">
            <span>Action</span>
            <select value={action} onChange={(event) => setAction(event.target.value as GovernanceAction | '')}>
              <option value="">All</option>
              <option value="retire">retire</option>
              <option value="deprecate">deprecate</option>
              <option value="yank">yank</option>
            </select>
          </label>
          <div className="action-row">
            <div className="action-row__copy">
              <strong>How to read this console</strong>
              <p>先看 blocking / replacement 影响面，再看具体 timeline；不要把所有治理事件都当成同一种严重度。</p>
            </div>
          </div>
        </SurfaceCard>

        <div className="stack-column">
          {governance.loading ? <div className="empty-box">Loading governance records…</div> : null}
          {governance.error ? <div className="error-box">{governance.error}</div> : null}
          {!governance.loading && !governance.error && filteredItems.length === 0 ? (
            <div className="empty-box">No governance records matched the current filters.</div>
          ) : null}

          {layers.map((layer) => (
            <SurfaceCard
              className={layer.key === 'blocking' ? 'surface-card--cool' : undefined}
              description={layer.description}
              key={layer.key}
              title={layer.title}
            >
              <div className="timeline-section">
                {layer.items.map((item) => (
                  <article className="timeline-row" key={item.id}>
                    <div className="timeline-row__head">
                      <div>
                        <strong>{item.action} {item.packageName}{item.version ? ` ${item.version}` : ''}</strong>
                        <span>{item.changedBy} · {formatDate(item.changedAt)}</span>
                      </div>
                      <div className="pill-row">
                        <StatusPill tone={item.action === 'yank' || item.action === 'retire' ? 'bad' : 'warn'}>{item.action}</StatusPill>
                        <StatusPill tone={item.targetType === 'version' ? 'warn' : 'bad'}>{item.targetType}</StatusPill>
                      </div>
                    </div>
                    <p>{item.reason}</p>
                    {item.replacementRecommendation ? (
                      <div className="pill-row">
                        <StatusPill tone="accent">replacement {item.replacementRecommendation}</StatusPill>
                      </div>
                    ) : null}
                  </article>
                ))}
              </div>
            </SurfaceCard>
          ))}
        </div>
      </div>
    </div>
  );
}
