import { useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { listAuditEvents } from '../api/client';
import type { AuditEventRecord } from '../api/types';
import { SectionHeading } from '../components/SectionHeading';
import { StatusPill } from '../components/StatusPill';
import { SurfaceCard } from '../components/SurfaceCard';
import { formatDate } from '../lib/format';
import { formatAuditEndpoint } from '../lib/formatAuditEndpoint';
import { usePaginatedCollection } from '../lib/usePaginatedCollection';

type AuditLayer = {
  key: string;
  title: string;
  description: string;
  items: AuditEventRecord[];
};

function buildAuditLayers(records: AuditEventRecord[]): AuditLayer[] {
  const failures = records.filter((item) => item.outcome !== 'succeeded');
  const mutations = records.filter((item) => item.method !== 'GET');
  const reads = records.filter((item) => item.method === 'GET');

  return [
    {
      key: 'failures',
      title: 'Failures / denied / blocked',
      description: '先看失败动作，因为这些最直接决定当前控制台是否可用、是否被策略拦住。',
      items: failures,
    },
    {
      key: 'mutations',
      title: 'Mutating operations',
      description: '这些动作真的改变了 Catalog / packaging / governance 状态，应该和只读事件区分开看。',
      items: mutations,
    },
    {
      key: 'reads',
      title: 'Read-side trace',
      description: '只读请求同样重要，但更适合放到最后看，用于追溯谁查过什么。',
      items: reads,
    },
  ].filter((layer) => layer.items.length > 0);
}

export function AuditPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const packageName = searchParams.get('package') || '';
  const version = searchParams.get('version') || '';
  const actor = searchParams.get('actor') || '';
  const outcome = searchParams.get('outcome') || '';
  const authDecision = searchParams.get('authDecision') || '';

  const audit = usePaginatedCollection({
    loader: (filters) =>
      listAuditEvents({
        ...filters,
        packageName: packageName.trim() || undefined,
        version: version.trim() || undefined,
        actor: actor.trim() || undefined,
        outcome: outcome || undefined,
        authDecision: authDecision || undefined,
        pageSize: 10,
      }),
    filters: {
      cursor: undefined,
    },
    deps: [packageName, version, actor, outcome, authDecision],
  });

  const items = useMemo(() => audit.items, [audit.items]);
  const layers = buildAuditLayers(items);
  const deniedCount = items.filter((item) => item.authDecision === 'denied').length;
  const failedCount = items.filter((item) => item.outcome !== 'succeeded').length;
  const mutatingCount = items.filter((item) => item.method !== 'GET').length;

  function updateFilter(key: 'package' | 'version' | 'actor' | 'outcome' | 'authDecision', value: string) {
    const next = new URLSearchParams(searchParams);
    const normalizedValue = value.trim();
    if (normalizedValue === '') {
      next.delete(key);
    } else {
      next.set(key, normalizedValue);
    }
    setSearchParams(next, { replace: true });
  }

  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="Audit"
        title="让人追溯‘谁在什么时候对哪个 skill/version 做了哪一步’，并先看真正重要的失败面。"
        description="审计页不再只是堆 feed，而是把 denied / failed / mutating / read-side trace 分层展开。"
        tone="backstage"
      />

      <div className="context-strip">
        <StatusPill tone="accent">{packageName.trim() || 'all packages'}</StatusPill>
        <StatusPill>{version.trim() || 'all versions'}</StatusPill>
        <StatusPill>{actor.trim() || 'all actors'}</StatusPill>
        <StatusPill tone={outcome === 'succeeded' ? 'good' : outcome ? 'bad' : undefined}>{outcome || 'all outcomes'}</StatusPill>
        <StatusPill tone={authDecision === 'denied' ? 'bad' : authDecision === 'allowed' ? 'good' : undefined}>{authDecision || 'all auth decisions'}</StatusPill>
      </div>

      <div className="console-strip">
        <article className="console-signal">
          <span>Denied requests</span>
          <strong>{deniedCount}</strong>
          <p>如果 denied 持续偏高，说明控制台当前暴露的动作和权限模型之间还有错位。</p>
        </article>
        <article className="console-signal">
          <span>Failed operations</span>
          <strong>{failedCount}</strong>
          <p>失败不是噪音，它们直接决定用户现在能不能完成发布、治理或安装相关操作。</p>
        </article>
        <article className="console-signal">
          <span>Mutations</span>
          <strong>{mutatingCount}</strong>
          <p>写操作是 audit 里最值得优先看的那部分，因为它们真的改变了系统状态。</p>
        </article>
      </div>

      <div className="two-column two-column--wide-right">
        <SurfaceCard title="筛选" description="先缩小 actor / outcome / package，再读时间轴。" variant="console">
          <label className="field">
            <span>包名</span>
            <input
              onChange={(event) => updateFilter('package', event.target.value)}
              placeholder="@company.skill/pdf-skill"
              value={packageName}
            />
          </label>
          <label className="field">
            <span>版本</span>
            <input onChange={(event) => updateFilter('version', event.target.value)} placeholder="1.2.0" value={version} />
          </label>
          <label className="field">
            <span>Actor</span>
            <input
              onChange={(event) => updateFilter('actor', event.target.value)}
              placeholder="svc-publisher / web-ui / user_214"
              value={actor}
            />
          </label>
          <label className="field">
            <span>结果</span>
            <select value={outcome} onChange={(event) => updateFilter('outcome', event.target.value)}>
              <option value="">全部</option>
              <option value="succeeded">succeeded</option>
              <option value="failed">failed</option>
              <option value="forbidden">forbidden</option>
              <option value="unauthorized">unauthorized</option>
            </select>
          </label>
          <label className="field">
            <span>鉴权结果</span>
            <select value={authDecision} onChange={(event) => updateFilter('authDecision', event.target.value)}>
              <option value="">全部</option>
              <option value="allowed">allowed</option>
              <option value="denied">denied</option>
            </select>
          </label>
          <div className="action-row">
            <div className="action-row__copy">
                  <strong>阅读建议</strong>
                  <p>先看 failures / denied，再看 mutating operations，最后再回头看 reads；不要把所有请求当同一种追溯粒度。</p>
                </div>
              </div>
        </SurfaceCard>

        <div className="stack-column">
          {audit.loading ? <div className="empty-box">正在加载审计记录…</div> : null}
          {audit.error ? <div className="error-box">{audit.error}</div> : null}
          {!audit.loading && !audit.error && items.length === 0 ? (
            <div className="empty-box">当前筛选条件下没有审计记录。</div>
          ) : null}

          {layers.map((layer) => (
            <SurfaceCard
              className={layer.key === 'failures' ? 'surface-card--cool' : undefined}
              description={layer.description}
              key={layer.key}
              title={layer.title}
            >
              <div className="timeline-section">
                {layer.items.map((item) => (
                  <article className="timeline-row" key={item.id}>
                    <div className="timeline-row__head">
                      <div>
                        <strong>{item.action}</strong>
                        <span>{item.actor} · {item.role} · {formatDate(item.createdAt)}</span>
                      </div>
                      <div className="pill-row">
                        <StatusPill tone={item.outcome === 'succeeded' ? 'good' : 'bad'}>{item.outcome}</StatusPill>
                        <StatusPill tone={item.authDecision === 'allowed' ? 'good' : 'bad'}>{item.authDecision}</StatusPill>
                        <StatusPill>{item.method}</StatusPill>
                      </div>
                    </div>
                    <p title={item.endpoint}>{formatAuditEndpoint(item.endpoint)}</p>
                    <div className="pill-row">
                      <StatusPill>{item.requestId}</StatusPill>
                      <StatusPill tone="accent">{item.statusCode.toString()}</StatusPill>
                      {item.packageName ? <StatusPill>{item.packageName}</StatusPill> : null}
                      {item.version ? <StatusPill>{item.version}</StatusPill> : null}
                    </div>
                  </article>
                ))}
              </div>
            </SurfaceCard>
          ))}

          {!audit.loading && !audit.error ? (
            <SurfaceCard
              description={
                audit.pageInfo.hasNextPage
                  ? `当前只展示前 ${items.length} 条审计记录，后面还有更多 trace。`
                  : `当前审计结果集已完整展示，共 ${items.length} 条。`
              }
              title="分页"
              variant="rail"
            >
              {audit.pageInfo.hasNextPage ? (
                <button className="button button--secondary" disabled={audit.loadingMore} onClick={audit.loadMore} type="button">
                  {audit.loadingMore ? '加载中…' : '继续加载'}
                </button>
              ) : null}
            </SurfaceCard>
          ) : null}
        </div>
      </div>
    </div>
  );
}
