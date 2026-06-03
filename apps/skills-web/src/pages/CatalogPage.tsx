import type { CatalogSourceType, SkillStatus } from '../api/types';
import { useDeferredValue, useState } from 'react';
import { Link } from 'react-router-dom';
import { listSkills } from '../api/client';
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
        title="搜索 skill，不在列表页就丢失最关键的状态信息。"
        description="首版 UI 先把 packageName、sourceType、latestVersion、version status 和 install 预判露出来。"
      />

      <div className="context-strip">
        <StatusPill tone="accent">query {deferredQuery || 'all'}</StatusPill>
        <StatusPill>{sourceType || 'all sources'}</StatusPill>
        <StatusPill tone={status === 'retired' ? 'bad' : status === 'active' ? 'good' : undefined}>{status || 'all states'}</StatusPill>
        <StatusPill tone="good">{resultItems.length} results</StatusPill>
      </div>

      <div className="console-strip">
        <article className="console-signal">
          <span>Discovery posture</span>
          <strong>{deferredQuery ? 'targeted' : 'broad'}</strong>
          <p>{deferredQuery ? '你已经在按具体意图筛技能。' : '当前更适合先浏览 catalog 面。'}</p>
        </article>
        <article className="console-signal">
          <span>Active vs retired</span>
          <strong>{activeCount} / {retiredCount}</strong>
          <p>active 代表仍可继续演进；retired 代表只能做历史追溯或替代迁移。</p>
        </article>
        <article className="console-signal">
          <span>Imported footprint</span>
          <strong>{importedCount}</strong>
          <p>这批结果里有多少来自外部导入，有助于判断后续是否还要看 normalization / governance 风险。</p>
        </article>
      </div>

      <div className="two-column two-column--wide-right">
        <SurfaceCard className="surface-card--warm" title="Filters" description="前端只拼查询条件，不在浏览器里重建治理判断。">
          <label className="field">
            <span>Search</span>
            <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="pdf / review / scrape" />
          </label>
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
              <strong>How to use this page</strong>
              <p>先缩小结果集，再选中一个 skill 进入 detail；如果已经知道 package/version，下一步应该直接进 version review，而不是留在列表页犹豫。</p>
            </div>
          </div>
        </SurfaceCard>

        <SurfaceCard className="surface-card--cool" title="Results" description="映射 `GET /api/v1/skills`，当前已经可以切 mock / real client。">
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
              <article className="list-card list-card--feature" key={skill.packageName}>
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
