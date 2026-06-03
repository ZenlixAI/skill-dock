import { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import {
  applySkillGovernanceChange,
  getInstallTarget,
  getSkillDetail,
  getSkillVersions,
  listGovernanceChanges,
} from '../api/client';
import { SectionHeading } from '../components/SectionHeading';
import { StatusPill } from '../components/StatusPill';
import { SurfaceCard } from '../components/SurfaceCard';
import { useAsyncData } from '../lib/useAsyncData';
import { formatDate } from '../lib/format';

type SkillLogEntry = {
  id: string;
  action: 'retire';
  outcome: 'succeeded' | 'failed';
  summary: string;
};

export function SkillDetailPage() {
  const { packageName = '' } = useParams();
  const decodedPackage = decodeURIComponent(packageName);
  const [refreshKey, setRefreshKey] = useState(0);
  const [retireReason, setRetireReason] = useState('');
  const [activityLog, setActivityLog] = useState<SkillLogEntry[]>([]);
  const [submitState, setSubmitState] = useState<{
    loading: boolean;
    error: string | null;
    success: string | null;
  }>({
    loading: false,
    error: null,
    success: null,
  });

  const detail = useAsyncData(() => getSkillDetail(decodedPackage), [decodedPackage, refreshKey]);
  const versionList = useAsyncData(() => getSkillVersions(decodedPackage), [decodedPackage, refreshKey]);
  const installTarget = useAsyncData(() => getInstallTarget(decodedPackage), [decodedPackage, refreshKey]);
  const governance = useAsyncData(
    () => listGovernanceChanges({ packageName: decodedPackage }),
    [decodedPackage, refreshKey]
  );
  const latestVersionItem = versionList.data?.[0];
  const statusTone = detail.data?.status === 'active' ? 'good' : 'bad';

  function appendLog(outcome: SkillLogEntry['outcome'], summary: string) {
    setActivityLog((current) => [
      {
        id: `retire-${Date.now()}`,
        action: 'retire',
        outcome,
        summary,
      },
      ...current,
    ]);
  }

  async function handleRetireSkill() {
    setSubmitState({
      loading: true,
      error: null,
      success: null,
    });
    try {
      const response = await applySkillGovernanceChange(decodedPackage, {
        action: 'retire',
        reason: retireReason.trim(),
      });
      setSubmitState({
        loading: false,
        error: null,
        success: `已提交 skill retire。当前状态：${response.resultingState.status || 'retired'}`,
      });
      appendLog('succeeded', `Retired ${decodedPackage}.`);
      setRefreshKey((current) => current + 1);
    } catch (error: unknown) {
      appendLog('failed', error instanceof Error ? error.message : String(error));
      setSubmitState({
        loading: false,
        error: error instanceof Error ? error.message : String(error),
        success: null,
      });
    }
  }

  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="Skill Detail"
        title={detail.data?.displayName || decodedPackage}
        description={detail.data?.description || 'Loading skill detail…'}
      />

      <div className="context-strip">
        <StatusPill tone={statusTone}>{detail.data?.status || 'unknown'}</StatusPill>
        {detail.data?.sourceType ? <StatusPill tone="accent">{detail.data.sourceType}</StatusPill> : null}
        {detail.data?.latestVersion ? <StatusPill>latest {detail.data.latestVersion}</StatusPill> : null}
        {installTarget.data?.version ? <StatusPill tone="good">install target {installTarget.data.version}</StatusPill> : null}
      </div>

      <div className="console-strip">
        <article className="console-signal">
          <span>Discovery handoff</span>
          <strong>{detail.data?.status === 'active' ? 'actionable' : 'historical'}</strong>
          <p>{detail.data?.status === 'active' ? '这个 skill 仍然适合继续深入到 version review 或 install 预览。' : '这个 skill 更像历史对象，下一步重点应该是替代或追溯。'}</p>
        </article>
        <article className="console-signal">
          <span>Version surface</span>
          <strong>{versionList.data?.length || 0}</strong>
          <p>这里有多少个可检查版本，决定了后续是去 latest review 还是回看历史治理。</p>
        </article>
        <article className="console-signal">
          <span>Governance pressure</span>
          <strong>{governance.data?.length || 0}</strong>
          <p>治理事件越多，说明这个 skill 越应该用控制台视角去看，而不是只看 README 式详情。</p>
        </article>
      </div>

      <div className="two-column">
        <SurfaceCard className="surface-card--warm" title="Identity" description="这是用户判断‘这个 skill 是什么’的第一屏。">
          <dl className="property-list">
            <div>
              <dt>package name</dt>
              <dd>{decodedPackage}</dd>
            </div>
            <div>
              <dt>source type</dt>
              <dd>{detail.data?.sourceType || '—'}</dd>
            </div>
            <div>
              <dt>skill status</dt>
              <dd>{detail.data ? <StatusPill tone={detail.data.status === 'active' ? 'good' : 'bad'}>{detail.data.status}</StatusPill> : '—'}</dd>
            </div>
            <div>
              <dt>latest version</dt>
              <dd>{detail.data?.latestVersion || '—'}</dd>
            </div>
          </dl>
        </SurfaceCard>

        <SurfaceCard className="surface-card--cool" title="Install target preview" description="Reader 最关心的不是对象理论，而是怎么装。">
          <dl className="property-list">
            <div>
              <dt>registry</dt>
              <dd>{installTarget.data?.npmRegistry || '—'}</dd>
            </div>
            <div>
              <dt>tarball</dt>
              <dd>{installTarget.data?.tarballUrl || '—'}</dd>
            </div>
            <div>
              <dt>integrity</dt>
              <dd>{installTarget.data?.integrity || '—'}</dd>
            </div>
            <div>
              <dt>manifest digest</dt>
              <dd>{installTarget.data?.skillManifestDigest || '—'}</dd>
            </div>
          </dl>
          <div className="list-card__actions">
            {latestVersionItem ? (
              <Link
                className="button button--secondary"
                to={`/skills/${encodeURIComponent(decodedPackage)}/versions/${latestVersionItem.version}`}
              >
                Inspect latest version
              </Link>
            ) : null}
            <Link className="button button--secondary" to="/catalog">
              Back to catalog
            </Link>
          </div>
        </SurfaceCard>
      </div>

      <div className="two-column">
        <SurfaceCard title="Versions" description="这里不是纯列表，而是从 skill 进入 version review 的分发台。">
          <div className="list-stack">
            {versionList.data?.map((item) => (
              <article className="list-card" key={item.version}>
                <div className="list-card__head">
                  <div>
                    <strong>{item.version}</strong>
                    <span>{formatDate(item.publishedAt)}</span>
                  </div>
                  <StatusPill tone={item.versionStatus === 'published' ? 'good' : item.versionStatus === 'deprecated' ? 'warn' : 'bad'}>
                    {item.versionStatus}
                  </StatusPill>
                </div>
                <div className="pill-row">
                  <StatusPill tone={item.installPolicy === 'resolvable' ? 'good' : item.installPolicy === 'catalog_only' ? 'warn' : 'bad'}>
                    {item.installPolicy}
                  </StatusPill>
                </div>
                <div className="list-card__actions">
                  <Link
                    className="button button--secondary"
                    to={`/skills/${encodeURIComponent(decodedPackage)}/versions/${item.version}`}
                  >
                    Open version
                  </Link>
                </div>
              </article>
            ))}
          </div>
        </SurfaceCard>

        <div className="stack-column">
          <SurfaceCard className="surface-card--cool surface-card--sticky" title="Skill Governance" description="skill 级治理写接口：当前先接 retire。">
            <label className="field">
              <span>Reason</span>
              <textarea
                value={retireReason}
                onChange={(event) => setRetireReason(event.target.value)}
                placeholder="merged into successor skill / no longer allowed in catalog"
              />
            </label>
            {submitState.error ? <div className="error-box">{submitState.error}</div> : null}
            {submitState.success ? <div className="empty-box empty-box--success">{submitState.success}</div> : null}
            <div className="action-row">
              <div className="action-row__copy">
                <strong>Retire the whole skill line</strong>
                <p>这是 package 级动作，不只是某个版本下线。它会把这个 skill 从继续演进的对象变成历史对象。</p>
              </div>
            </div>
            <button
              className="button button--secondary"
              disabled={submitState.loading || retireReason.trim() === ''}
              onClick={() => void handleRetireSkill()}
              type="button"
            >
              {submitState.loading ? 'Submitting…' : 'Retire skill'}
            </button>
          </SurfaceCard>

          <SurfaceCard title="Selection Path" description="把这个 skill 当前最合理的下一步动作说清楚。">
            <div className="list-stack">
              <article className="list-card">
                <div className="list-card__head">
                  <div>
                    <strong>Inspect latest version</strong>
                    <span>{latestVersionItem?.version || 'no version found'}</span>
                  </div>
                  <StatusPill tone={latestVersionItem?.versionStatus === 'published' ? 'good' : latestVersionItem?.versionStatus === 'deprecated' ? 'warn' : 'bad'}>
                    {latestVersionItem?.versionStatus || 'unknown'}
                  </StatusPill>
                </div>
                <p>如果你要继续走发布、release-record、governance，下一站应该是 version detail，而不是停留在 skill summary。</p>
              </article>
              <article className="list-card">
                <div className="list-card__head">
                  <div>
                    <strong>Catalog install posture</strong>
                    <span>{installTarget.data?.version || 'unresolved'}</span>
                  </div>
                  <StatusPill tone={detail.data?.status === 'active' ? 'good' : 'bad'}>
                    {detail.data?.status || 'unknown'}
                  </StatusPill>
                </div>
                <p>如果 install target 都不值得继续给下游，就应该优先治理，而不是继续宣传这个 skill。</p>
              </article>
            </div>
          </SurfaceCard>

          <SurfaceCard title="Recent Governance" description="只展示当前 skill 的治理记录，方便和写动作闭环。">
            <div className="list-stack">
              {governance.data?.length ? (
                governance.data.map((item) => (
                  <article className="list-card" key={item.id}>
                    <div className="list-card__head">
                      <div>
                        <strong>{item.action}</strong>
                        <span>{formatDate(item.changedAt)}</span>
                      </div>
                      <StatusPill tone={item.targetType === 'skill' ? 'bad' : 'warn'}>{item.targetType}</StatusPill>
                    </div>
                    <p>{item.reason}</p>
                  </article>
                ))
              ) : (
                <div className="empty-box">No governance events for this skill yet.</div>
              )}
            </div>
          </SurfaceCard>

          <SurfaceCard title="Execution Log" description="记录这次会话里对这个 skill 做过的治理动作。">
            <div className="list-stack">
              {activityLog.length ? (
                activityLog.map((entry) => (
                  <article className="list-card" key={entry.id}>
                    <div className="list-card__head">
                      <div>
                        <strong>{entry.action}</strong>
                        <span>{entry.summary}</span>
                      </div>
                      <StatusPill tone={entry.outcome === 'succeeded' ? 'good' : 'bad'}>{entry.outcome}</StatusPill>
                    </div>
                  </article>
                ))
              ) : (
                <div className="empty-box">No skill-level actions run yet in this session.</div>
              )}
            </div>
          </SurfaceCard>
        </div>
      </div>
    </div>
  );
}
