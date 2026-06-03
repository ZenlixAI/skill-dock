import { useDeferredValue, useState } from 'react';
import { Link } from 'react-router-dom';
import { listSkills } from '../api/client';
import type { CatalogSourceType, SkillStatus } from '../api/types';
import { SectionHeading } from '../components/SectionHeading';
import { StatusPill } from '../components/StatusPill';
import { SurfaceCard } from '../components/SurfaceCard';
import { useAsyncData } from '../lib/useAsyncData';

export function CatalogPage() {
  const [query, setQuery] = useState('');
  const [sourceType, setSourceType] = useState<CatalogSourceType | ''>('');
  const [status, setStatus] = useState<SkillStatus | ''>('');
  const deferredQuery = useDeferredValue(query);

  const skills = useAsyncData(
    () =>
      listSkills({
        query: deferredQuery,
        sourceType,
        status,
      }),
    [deferredQuery, sourceType, status]
  );
  const resultItems = skills.data || [];
  const activeCount = resultItems.filter((item) => item.status === 'active').length;
  const retiredCount = resultItems.filter((item) => item.status === 'retired').length;
  const importedCount = resultItems.filter((item) => item.sourceType === 'imported').length;

  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="Catalog"
        title="让搜索、筛选和下一步动作在同一屏里变得明确。"
        description="这页不只是列结果，而是把 package 状态、来源、最新版本和去向一起做成一个高信号 discovery surface。"
        tone="frontstage"
      />

      <SurfaceCard description="把搜索条抬到页面主节奏里，先收窄意图，再决定是否进入 skill 或 version detail。" title="Command Band" variant="hero">
        <div className="catalog-command">
          <label className="field catalog-command__search">
            <span>Search</span>
            <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="pdf / review / scrape / governance" />
          </label>
          <div className="catalog-command__status">
            <StatusPill tone="accent">query {deferredQuery || 'all'}</StatusPill>
            <StatusPill>{sourceType || 'all sources'}</StatusPill>
            <StatusPill tone={status === 'retired' ? 'bad' : status === 'active' ? 'good' : undefined}>{status || 'all states'}</StatusPill>
            <StatusPill tone="good">{resultItems.length} results</StatusPill>
          </div>
        </div>
      </SurfaceCard>

      <div className="console-strip">
        <article className="console-signal">
          <span>Discovery posture</span>
          <strong>{deferredQuery ? 'targeted' : 'broad'}</strong>
          <p>{deferredQuery ? '当前已经在按具体意图缩小结果。' : '更适合先从 curated 和 source type 开始筛。'}</p>
        </article>
        <article className="console-signal">
          <span>Active vs retired</span>
          <strong>{activeCount} / {retiredCount}</strong>
          <p>active 仍可继续演进；retired 更适合做历史追溯或迁移判断。</p>
        </article>
        <article className="console-signal">
          <span>Imported footprint</span>
          <strong>{importedCount}</strong>
          <p>结果里外部导入越多，越值得继续检查 normalization 与 governance 风险。</p>
        </article>
      </div>

      <div className="two-column two-column--wide-right">
        <SurfaceCard description="前端只组织查询条件，不替后端重建治理判断。" title="Filters" variant="rail">
          <label className="field">
            <span>Source type</span>
            <select value={sourceType} onChange={(event) => setSourceType(event.target.value as CatalogSourceType | '')}>
              <option value="">All</option>
              <option value="native">native</option>
              <option value="imported">imported</option>
              <option value="platform_created">platform_created</option>
            </select>
          </label>
          <label className="field">
            <span>Status</span>
            <select value={status} onChange={(event) => setStatus(event.target.value as SkillStatus | '')}>
              <option value="">All</option>
              <option value="active">active</option>
              <option value="retired">retired</option>
            </select>
          </label>

          <div className="action-row">
            <div className="action-row__copy">
              <strong>Reading posture</strong>
              <p>先缩小结果集，再决定是进入 skill summary 还是直接看最新 version review，不要在列表页反复横跳。</p>
            </div>
          </div>
        </SurfaceCard>

        <SurfaceCard description="映射 `GET /api/v1/skills`，但把结果组织成更接近产品面搜索体验的卡片流。" title="Results" variant="spotlight">
          {skills.loading ? <div className="empty-box">Loading catalog…</div> : null}
          {skills.error ? <div className="error-box">{skills.error}</div> : null}

          <div className="console-strip console-strip--compact">
            <article className="console-signal console-signal--compact">
              <span>Results</span>
              <strong>{resultItems.length}</strong>
            </article>
            <article className="console-signal console-signal--compact">
              <span>Active</span>
              <strong>{activeCount}</strong>
            </article>
            <article className="console-signal console-signal--compact">
              <span>Retired</span>
              <strong>{retiredCount}</strong>
            </article>
          </div>

          <div className="list-stack">
            {resultItems.map((skill) => (
              <article className="list-card list-card--feature list-card--spotlight" key={skill.packageName}>
                <div className="list-card__head">
                  <div>
                    <strong>{skill.displayName}</strong>
                    <span>{skill.packageName}</span>
                  </div>
                  <StatusPill tone={skill.latestVersionStatus === 'published' ? 'good' : skill.latestVersionStatus === 'deprecated' ? 'warn' : 'bad'}>
                    {skill.latestVersionStatus || 'unknown'}
                  </StatusPill>
                </div>
                <p>{skill.description}</p>
                <div className="pill-row">
                  <StatusPill tone="accent">{skill.sourceType}</StatusPill>
                  <StatusPill tone={skill.status === 'active' ? 'good' : 'bad'}>{skill.status}</StatusPill>
                  <StatusPill>latest {skill.latestVersion || '—'}</StatusPill>
                </div>
                <div className="list-card__actions">
                  <Link className="button button--secondary" to={`/skills/${encodeURIComponent(skill.packageName)}`}>
                    Open skill
                  </Link>
                  {skill.latestVersion ? (
                    <Link
                      className="button button--secondary"
                      to={`/skills/${encodeURIComponent(skill.packageName)}/versions/${skill.latestVersion}`}
                    >
                      Open latest version
                    </Link>
                  ) : null}
                </div>
              </article>
            ))}
            {!skills.loading && !skills.error && resultItems.length === 0 ? (
              <div className="empty-box">No skills matched the current filters.</div>
            ) : null}
          </div>
        </SurfaceCard>
      </div>
    </div>
  );
}
