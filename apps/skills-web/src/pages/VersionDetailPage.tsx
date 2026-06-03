import { useState } from 'react';
import { Link, useParams, useSearchParams } from 'react-router-dom';
import {
  applyVersionGovernanceChange,
  createReleaseRecord,
  getVersionDetail,
  listGovernanceChanges,
} from '../api/client';
import { SectionHeading } from '../components/SectionHeading';
import { StatusPill } from '../components/StatusPill';
import { SurfaceCard } from '../components/SurfaceCard';
import { useAsyncData } from '../lib/useAsyncData';
import { formatDate } from '../lib/format';
import { buildReleaseHandoffPath, readReleaseHandoff } from '../lib/releaseHandoff';

type VersionLogEntry = {
  id: string;
  action: 'release-record' | 'governance';
  outcome: 'succeeded' | 'failed';
  summary: string;
};

export function VersionDetailPage() {
  const { packageName = '', version = '' } = useParams();
  const [searchParams] = useSearchParams();
  const decodedPackage = decodeURIComponent(packageName);
  const handoff = readReleaseHandoff(searchParams, {
    packageName: decodedPackage,
    version,
  });
  const [refreshKey, setRefreshKey] = useState(0);
  const [publishRegistry, setPublishRegistry] = useState(handoff.publishRegistry || '');
  const [manifestDigest, setManifestDigest] = useState(handoff.manifestDigest || '');
  const [components, setComponents] = useState(handoff.components?.join(', ') || '');
  const [governanceAction, setGovernanceAction] = useState<'deprecate' | 'yank'>('deprecate');
  const [governanceReason, setGovernanceReason] = useState('');
  const [replacementRecommendation, setReplacementRecommendation] = useState('');
  const [activityLog, setActivityLog] = useState<VersionLogEntry[]>([]);
  const [actionState, setActionState] = useState<{
    loading: false | 'release' | 'governance';
    error: string | null;
    success: string | null;
  }>({
    loading: false,
    error: null,
    success: null,
  });

  const detail = useAsyncData(() => getVersionDetail(decodedPackage, version), [decodedPackage, version, refreshKey]);
  const governance = useAsyncData(
    () => listGovernanceChanges({ packageName: decodedPackage, version }),
    [decodedPackage, version, refreshKey]
  );

  const releaseComponents = components
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
  const authoringPath = buildReleaseHandoffPath(handoff, '/authoring');
  const hasHandoff = Boolean(handoff.manifestDigest || handoff.bundlePath || handoff.workspacePath || handoff.components?.length);
  const currentVersionStatus = detail.data?.versionStatus || 'unknown';
  const versionTone =
    currentVersionStatus === 'published' ? 'good' : currentVersionStatus === 'deprecated' ? 'warn' : 'bad';
  const governanceImpact =
    governanceAction === 'yank'
      ? 'Explicit installs should become blocked and downstream users should move to a replacement.'
      : 'The version remains visible, but consumers should be nudged toward a newer replacement.';

  function appendLog(action: VersionLogEntry['action'], outcome: VersionLogEntry['outcome'], summary: string) {
    setActivityLog((current) => [
      {
        id: `${action}-${Date.now()}`,
        action,
        outcome,
        summary,
      },
      ...current,
    ]);
  }

  async function handleReleaseRecord() {
    setActionState({
      loading: 'release',
      error: null,
      success: null,
    });
    try {
      const response = await createReleaseRecord(decodedPackage, version, {
        publishRegistry: publishRegistry.trim() || undefined,
        manifestDigest: manifestDigest.trim() || undefined,
        components: releaseComponents.length ? releaseComponents : undefined,
      });
      setActionState({
        loading: false,
        error: null,
        success: response.created
          ? '已创建新的 release record。'
          : 'release record 已存在，后端返回幂等重放结果。',
      });
      appendLog(
        'release-record',
        'succeeded',
        response.created ? 'Created a new release record.' : 'Release record returned idempotent replay.'
      );
      setRefreshKey((current) => current + 1);
    } catch (error: unknown) {
      appendLog('release-record', 'failed', error instanceof Error ? error.message : String(error));
      setActionState({
        loading: false,
        error: error instanceof Error ? error.message : String(error),
        success: null,
      });
    }
  }

  async function handleVersionGovernance() {
    setActionState({
      loading: 'governance',
      error: null,
      success: null,
    });
    try {
      const response = await applyVersionGovernanceChange(decodedPackage, version, {
        action: governanceAction,
        reason: governanceReason.trim(),
        replacementRecommendation:
          replacementRecommendation.trim() === '' ? undefined : replacementRecommendation.trim(),
      });
      setActionState({
        loading: false,
        error: null,
        success: `已提交 ${response.record.action}。当前 version state：${
          response.resultingState.versionStatus || response.record.action
        }`,
      });
      appendLog('governance', 'succeeded', `Applied ${response.record.action} governance change.`);
      setRefreshKey((current) => current + 1);
    } catch (error: unknown) {
      appendLog('governance', 'failed', error instanceof Error ? error.message : String(error));
      setActionState({
        loading: false,
        error: error instanceof Error ? error.message : String(error),
        success: null,
      });
    }
  }

  return (
    <div className="page-grid">
      <SectionHeading
        actions={
          <div className="authoring-actions">
            <Link className="button button--secondary" to={`/skills/${encodeURIComponent(decodedPackage)}`}>
              Back to skill
            </Link>
            {hasHandoff ? (
              <Link className="button button--primary" to={authoringPath}>
                Return to authoring
              </Link>
            ) : null}
          </div>
        }
        eyebrow="Version Detail"
        title={`${decodedPackage} · ${version}`}
        description={
          hasHandoff
            ? '这个页面已经承接上一步 pack 结果，release-record 表单会自动预填。'
            : '这个页面专门回答某个精确版本的 registry、bundle 和治理状态。'
        }
        tone="detail"
      />

      <div className="context-strip">
        <StatusPill tone={versionTone}>{currentVersionStatus}</StatusPill>
        {detail.data?.installPolicy ? <StatusPill tone={detail.data.installPolicy === 'resolvable' ? 'good' : detail.data.installPolicy === 'catalog_only' ? 'warn' : 'bad'}>{detail.data.installPolicy}</StatusPill> : null}
        {detail.data?.registry.npmRegistry ? <StatusPill>{detail.data.registry.npmRegistry}</StatusPill> : null}
        <StatusPill tone={hasHandoff ? 'accent' : undefined}>{hasHandoff ? 'handoff attached' : 'direct open'}</StatusPill>
      </div>

      {hasHandoff ? (
        <SurfaceCard
          title="Release Handoff"
          description="这些字段来自上一步 `pack`，用于减少作者在 release review 阶段的重复录入。"
          variant="spotlight"
          action={
            <Link className="button button--secondary" to={authoringPath}>
              Back to authoring flow
            </Link>
          }
        >
          <dl className="property-list">
            <div>
              <dt>workspace path</dt>
              <dd>{handoff.workspacePath || '—'}</dd>
            </div>
            <div>
              <dt>bundle path</dt>
              <dd>{handoff.bundlePath || '—'}</dd>
            </div>
            <div>
              <dt>manifest digest</dt>
              <dd>{handoff.manifestDigest || '—'}</dd>
            </div>
            <div>
              <dt>components</dt>
              <dd>{handoff.components?.join(', ') || '—'}</dd>
            </div>
          </dl>
        </SurfaceCard>
      ) : null}

      <div className="console-strip">
        <article className="console-signal">
          <span>Release action</span>
          <strong>{actionState.loading === 'release' ? 'running' : 'ready'}</strong>
          <p>这里负责把 registry 里已有 bundle 显式登记成 Catalog release record。</p>
        </article>
        <article className="console-signal">
          <span>Governance action</span>
          <strong>{actionState.loading === 'governance' ? 'running' : 'ready'}</strong>
          <p>如果版本需要 deprecate / yank，应该从这里触发而不是靠隐式状态。</p>
        </article>
        <article className="console-signal">
          <span>Failure semantics</span>
          <strong>strict</strong>
          <p>后端返回的 immutable conflict、digest mismatch、policy block 都应该原样暴露，不要被 UI 吃掉。</p>
        </article>
      </div>

      <div className="two-column">
        <SurfaceCard title="Install Impact Preview" description="这个版本当前的发布状态会如何影响下游消费。">
          <div className="list-stack">
            <article className="list-card">
              <div className="list-card__head">
                <div>
                  <strong>Current resolver posture</strong>
                  <span>{detail.data?.installPolicy || 'unknown'}</span>
                </div>
                <StatusPill tone={detail.data?.installPolicy === 'resolvable' ? 'good' : detail.data?.installPolicy === 'catalog_only' ? 'warn' : 'bad'}>
                  {detail.data?.installPolicy || 'unknown'}
                </StatusPill>
              </div>
              <p>
                {detail.data?.installPolicy === 'blocked'
                  ? '下游安装解析应被阻断。'
                  : detail.data?.installPolicy === 'catalog_only'
                    ? '版本存在于 catalog，但不应该被当成普通 install target。'
                    : '当前版本仍可被正常解析和安装。'}
              </p>
            </article>
            <article className="list-card">
              <div className="list-card__head">
                <div>
                  <strong>Selected governance impact</strong>
                  <span>{governanceAction}</span>
                </div>
                <StatusPill tone={governanceAction === 'yank' ? 'bad' : 'warn'}>{governanceAction}</StatusPill>
              </div>
              <p>{governanceImpact}</p>
            </article>
          </div>
        </SurfaceCard>

        <SurfaceCard title="Execution Log" description="记录这次页面会话中触发过的 release / governance 动作。">
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
              <div className="empty-box">No actions run yet in this session.</div>
            )}
          </div>
        </SurfaceCard>
      </div>

      <div className="two-column">
        <SurfaceCard title="Registry & Bundle" description="映射 `GET /api/v1/skills/{packageName}/versions/{version}`。">
          <dl className="property-list">
            <div>
              <dt>version status</dt>
              <dd>
                {detail.data ? (
                  <StatusPill tone={detail.data.versionStatus === 'published' ? 'good' : detail.data.versionStatus === 'deprecated' ? 'warn' : 'bad'}>
                    {detail.data.versionStatus}
                  </StatusPill>
                ) : (
                  '—'
                )}
              </dd>
            </div>
            <div>
              <dt>install policy</dt>
              <dd>{detail.data?.installPolicy || '—'}</dd>
            </div>
            <div>
              <dt>npm registry</dt>
              <dd>{detail.data?.registry.npmRegistry || '—'}</dd>
            </div>
            <div>
              <dt>integrity</dt>
              <dd>{detail.data?.registry.integrity || '—'}</dd>
            </div>
            <div>
              <dt>shasum</dt>
              <dd>{detail.data?.registry.shasum || '—'}</dd>
            </div>
            <div>
              <dt>manifest digest</dt>
              <dd>{detail.data?.bundle.skillManifestDigest || '—'}</dd>
            </div>
            <div>
              <dt>components</dt>
              <dd>{detail.data?.bundle.components.join(', ') || '—'}</dd>
            </div>
          </dl>
        </SurfaceCard>

        <SurfaceCard className="surface-card--sticky" title="Release Action Center" description="把 release-record 做成独立控制面动作，而不是附带按钮。" variant="console">
          <div className="stack-column">
            <label className="field">
              <span>publish registry</span>
              <input
                onChange={(event) => setPublishRegistry(event.target.value)}
                placeholder={detail.data?.registry.npmRegistry || 'https://npm.internal.example.com'}
                value={publishRegistry}
              />
            </label>
            <label className="field">
              <span>manifest digest</span>
              <input
                onChange={(event) => setManifestDigest(event.target.value)}
                placeholder={detail.data?.bundle.skillManifestDigest || 'sha256:...'}
                value={manifestDigest}
              />
            </label>
            <label className="field">
              <span>components</span>
              <input
                onChange={(event) => setComponents(event.target.value)}
                placeholder={detail.data?.bundle.components.join(', ') || 'commands, scripts'}
                value={components}
              />
            </label>
            {actionState.error ? <div className="error-box">{actionState.error}</div> : null}
            {actionState.success ? <div className="empty-box empty-box--success">{actionState.success}</div> : null}
            <div className="action-row">
              <div className="action-row__copy">
                <strong>Register this bundle into Catalog</strong>
                <p>只有这里提交成功，版本才正式进入 Catalog 的 release-tracked 状态。</p>
              </div>
            </div>
            <button
              className="button button--primary"
              disabled={actionState.loading !== false}
              onClick={() => void handleReleaseRecord()}
              type="button"
            >
              {actionState.loading === 'release' ? 'Submitting…' : 'Create release record'}
            </button>
          </div>
        </SurfaceCard>
      </div>

      <div className="two-column">
        <SurfaceCard title="Version Governance" description="当前支持 deprecate / yank，并允许带 replacement recommendation。" variant="console">
          <div className="stack-column">
            <label className="field">
              <span>Action</span>
              <select value={governanceAction} onChange={(event) => setGovernanceAction(event.target.value as 'deprecate' | 'yank')}>
                <option value="deprecate">deprecate</option>
                <option value="yank">yank</option>
              </select>
            </label>
            <label className="field">
              <span>Reason</span>
              <textarea
                onChange={(event) => setGovernanceReason(event.target.value)}
                placeholder="replacement available / security issue / incompatible behavior"
                value={governanceReason}
              />
            </label>
            <label className="field">
              <span>Replacement recommendation</span>
              <input
                onChange={(event) => setReplacementRecommendation(event.target.value)}
                placeholder="1.2.1"
                value={replacementRecommendation}
              />
            </label>
            <div className="action-row">
              <div className="action-row__copy">
                <strong>Change downstream install behavior</strong>
                <p>这里不是备注区，而是真正会影响 install target 解析和消费侧行为的治理入口。</p>
              </div>
            </div>
            <button
              className="button button--secondary"
              disabled={actionState.loading !== false || governanceReason.trim() === ''}
              onClick={() => void handleVersionGovernance()}
              type="button"
            >
              {actionState.loading === 'governance' ? 'Submitting…' : 'Submit governance change'}
            </button>
          </div>
        </SurfaceCard>

        <SurfaceCard title="Applied Governance" description="写入后直接在右侧回看当前版本治理记录。">
          <div className="list-stack">
            {governance.data?.length ? (
              governance.data.map((item) => (
                <article className="list-card" key={item.id}>
                  <div className="list-card__head">
                    <div>
                      <strong>{item.action}</strong>
                      <span>{formatDate(item.changedAt)}</span>
                    </div>
                    <StatusPill tone={item.action === 'yank' ? 'bad' : 'warn'}>{item.targetType}</StatusPill>
                  </div>
                  <p>{item.reason}</p>
                  {item.replacementRecommendation ? (
                    <div className="pill-row">
                      <StatusPill tone="accent">replacement {item.replacementRecommendation}</StatusPill>
                    </div>
                  ) : null}
                </article>
              ))
            ) : (
              <div className="empty-box">No governance events for this version yet.</div>
            )}
          </div>
        </SurfaceCard>
      </div>
    </div>
  );
}
