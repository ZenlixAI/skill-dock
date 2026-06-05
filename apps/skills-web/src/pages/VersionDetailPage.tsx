import { useState } from 'react';
import { Link, useParams, useSearchParams } from 'react-router-dom';
import {
  applyVersionGovernanceChange,
  createReleaseRecord,
  getSession,
  getVersionDetail,
  listGovernanceChanges,
} from '../api/client';
import { normalizeThrownRegistryError } from '../api/httpClient';
import { canPerformAction } from '../api/session';
import {
  inferMutationFailureOutcome,
  MutationResultPanel,
  readMutationResultSearchParams,
  writeMutationResultSearchParams,
} from '../components/MutationResultPanel';
import { PermissionGate } from '../components/PermissionGate';
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
  const [searchParams, setSearchParams] = useSearchParams();
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
  const [governanceConfirmed, setGovernanceConfirmed] = useState(false);
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
  const session = useAsyncData(() => getSession(), []);
  const governance = useAsyncData(
    () => listGovernanceChanges({ packageName: decodedPackage, version }),
    [decodedPackage, version, refreshKey]
  );

  const releaseComponents = components
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
  const authoringPath = buildReleaseHandoffPath(handoff, '/workbench/authoring');
  const hasHandoff = Boolean(handoff.manifestDigest || handoff.bundlePath || handoff.workspacePath || handoff.components?.length);
  const currentVersionStatus = detail.data?.versionStatus || 'unknown';
  const versionTone =
    currentVersionStatus === 'published' ? 'good' : currentVersionStatus === 'deprecated' ? 'warn' : 'bad';
  const governanceImpact =
    governanceAction === 'yank'
      ? '显式安装应被阻断，下游用户需要迁移到替代版本。'
      : '当前版本仍可见，但应明确引导消费方迁移到更新版本。';
  const replacementRequired = governanceAction === 'deprecate' || governanceAction === 'yank';
  const canSubmitGovernance =
    actionState.loading === false &&
    governanceReason.trim() !== '' &&
    governanceConfirmed &&
    (!replacementRequired || replacementRecommendation.trim() !== '');
  const releaseWriteEnabled = canPerformAction(session.data, 'catalog.release_record.create', decodedPackage);
  const governanceWriteEnabled = canPerformAction(session.data, 'catalog.governance.version.apply', decodedPackage);
  const mutationResult = readMutationResultSearchParams(searchParams, {
    packageName: decodedPackage,
    version,
    actor: session.data?.actor ?? null,
  });

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

  function persistMutationResult(result: {
    action: string;
    outcome: 'succeeded' | 'failed' | 'forbidden' | 'unauthorized';
    message: string;
    requestId?: string | null;
  }) {
    setSearchParams(
      writeMutationResultSearchParams(searchParams, {
        ...result,
        actor: session.data?.actor ?? null,
        packageName: decodedPackage,
        version,
      }),
      { replace: true }
    );
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
      persistMutationResult({
        action: 'release-record',
        outcome: 'succeeded',
        message: response.created
          ? '已创建新的 release record。'
          : 'release record 已存在，后端返回幂等重放结果。',
      });
      appendLog(
        'release-record',
        'succeeded',
        response.created ? '已创建新的 release record。' : 'release record 返回幂等重放结果。'
      );
      setRefreshKey((current) => current + 1);
    } catch (error: unknown) {
      const outcome = inferMutationFailureOutcome(error);
      const normalizedError = normalizeThrownRegistryError(error);
      appendLog('release-record', 'failed', normalizedError.message);
      setActionState({
        loading: false,
        error: normalizedError.message,
        success: null,
      });
      persistMutationResult({
        action: 'release-record',
        outcome,
        message: normalizedError.message,
        requestId: normalizedError.requestId,
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
      persistMutationResult({
        action: response.record.action,
        outcome: 'succeeded',
        message: `已提交 ${response.record.action}。当前 version state：${
          response.resultingState.versionStatus || response.record.action
        }`,
      });
      appendLog('governance', 'succeeded', `已应用 ${response.record.action} 治理动作。`);
      setRefreshKey((current) => current + 1);
    } catch (error: unknown) {
      const outcome = inferMutationFailureOutcome(error);
      const normalizedError = normalizeThrownRegistryError(error);
      appendLog('governance', 'failed', normalizedError.message);
      setActionState({
        loading: false,
        error: normalizedError.message,
        success: null,
      });
      persistMutationResult({
        action: governanceAction,
        outcome,
        message: normalizedError.message,
        requestId: normalizedError.requestId,
      });
    }
  }

  return (
    <div className="page-grid">
      <SectionHeading
        actions={
          <div className="authoring-actions">
            <Link className="button button--secondary" to={`/skills/${encodeURIComponent(decodedPackage)}`}>
              返回技能
            </Link>
            {hasHandoff ? (
              <Link className="button button--primary" to={authoringPath}>
                返回作者台
              </Link>
            ) : null}
          </div>
        }
        eyebrow="版本详情"
        title={`${decodedPackage} · ${version}`}
        description={hasHandoff ? '已附带发布交接信息。' : '查看版本状态和发布数据。'}
        tone="detail"
      />

      <div className="context-strip">
        <StatusPill tone={versionTone}>{currentVersionStatus}</StatusPill>
        {detail.data?.installPolicy ? <StatusPill tone={detail.data.installPolicy === 'resolvable' ? 'good' : detail.data.installPolicy === 'catalog_only' ? 'warn' : 'bad'}>{detail.data.installPolicy}</StatusPill> : null}
        {detail.data?.registry.npmRegistry ? <StatusPill>{detail.data.registry.npmRegistry}</StatusPill> : null}
        <StatusPill tone={hasHandoff ? 'accent' : undefined}>{hasHandoff ? '已附带交接' : '直接打开'}</StatusPill>
        <StatusPill tone={releaseWriteEnabled ? 'accent' : 'default'}>
          {session.loading ? '会话加载中' : session.error ? '会话不可用' : releaseWriteEnabled ? '允许发布登记' : '发布登记只读'}
        </StatusPill>
      </div>

      {hasHandoff ? (
        <SurfaceCard
          title="发布交接"
          variant="spotlight"
          action={
            <Link className="button button--secondary" to={authoringPath}>
              返回作者流程
            </Link>
          }
        >
          <dl className="property-list">
            <div>
              <dt>工作目录</dt>
              <dd>{handoff.workspacePath || '—'}</dd>
            </div>
            <div>
              <dt>bundle 路径</dt>
              <dd>{handoff.bundlePath || '—'}</dd>
            </div>
            <div>
              <dt>manifest 摘要</dt>
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
          <span>发布登记</span>
          <strong>{actionState.loading === 'release' ? '执行中' : '就绪'}</strong>
          <p>用于在 bundle 已经发布到 npm registry 之后写入 release record。</p>
        </article>
        <article className="console-signal">
          <span>治理动作</span>
          <strong>{actionState.loading === 'governance' ? '执行中' : '就绪'}</strong>
          <p>可执行 deprecate 或 yank。</p>
        </article>
        <article className="console-signal">
          <span>校验视角</span>
          <strong>严格</strong>
          <p>直接显示后端错误。</p>
        </article>
      </div>

      <div className="two-column">
        <SurfaceCard title="安装影响">
          <div className="list-stack">
            <article className="list-card">
              <div className="list-card__head">
                <div>
                  <strong>当前解析态势</strong>
                  <span>{detail.data?.installPolicy || '未知'}</span>
                </div>
                <StatusPill tone={detail.data?.installPolicy === 'resolvable' ? 'good' : detail.data?.installPolicy === 'catalog_only' ? 'warn' : 'bad'}>
                    {detail.data?.installPolicy || '未知'}
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
                  <strong>当前治理影响</strong>
                  <span>{governanceAction}</span>
                </div>
                <StatusPill tone={governanceAction === 'yank' ? 'bad' : 'warn'}>{governanceAction}</StatusPill>
              </div>
              <p>{governanceImpact}</p>
            </article>
          </div>
        </SurfaceCard>

        <SurfaceCard title="执行日志">
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
              <div className="empty-box">这个会话里还没有执行过动作。</div>
            )}
          </div>
        </SurfaceCard>
      </div>

      <div className="two-column">
        <SurfaceCard title="仓信息与 Bundle">
          <dl className="property-list">
            <div>
              <dt>版本状态</dt>
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
              <dt>安装策略</dt>
              <dd>{detail.data?.installPolicy || '—'}</dd>
            </div>
            <div>
              <dt>npm 仓地址</dt>
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
              <dt>manifest 摘要</dt>
              <dd>{detail.data?.bundle.skillManifestDigest || '—'}</dd>
            </div>
            <div>
              <dt>components</dt>
              <dd>{detail.data?.bundle.components.join(', ') || '—'}</dd>
            </div>
          </dl>
        </SurfaceCard>

        <MutationResultPanel result={mutationResult} />
      </div>

      <div className="two-column">
        <SurfaceCard className="surface-card--sticky" title="发布登记" variant="console">
          <PermissionGate
            fallback={<div className="empty-box">当前会话对 `catalog.release_record.create` 只有只读权限。</div>}
            packageName={decodedPackage}
            requiredAction="catalog.release_record.create"
            session={session.data}
            sessionError={session.error}
            sessionLoading={session.loading}
          >
            <div className="stack-column">
              <label className="field">
                <span>发布仓地址</span>
                <input
                  onChange={(event) => setPublishRegistry(event.target.value)}
                  placeholder={detail.data?.registry.npmRegistry || 'https://npm.internal.example.com'}
                  value={publishRegistry}
                />
              </label>
              <label className="field">
                <span>manifest 摘要</span>
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
              <div className="action-row">
                <div className="action-row__copy">
                  <strong>Register this bundle</strong>
                  <p>先确认这个版本已经完成 `npm publish`，再为它写入 catalog release record。</p>
                </div>
              </div>
              <button
                className="button button--primary"
                disabled={actionState.loading !== false}
                onClick={() => void handleReleaseRecord()}
                type="button"
              >
                {actionState.loading === 'release' ? '提交中…' : '创建 release record'}
              </button>
            </div>
          </PermissionGate>
        </SurfaceCard>
      </div>

      <div className="two-column">
        <SurfaceCard title="版本治理" variant="console">
          <PermissionGate
            fallback={<div className="empty-box">当前会话对 `catalog.governance.version.apply` 只有只读权限。</div>}
            packageName={decodedPackage}
            requiredAction="catalog.governance.version.apply"
            session={session.data}
            sessionError={session.error}
            sessionLoading={session.loading}
          >
            <div className="stack-column">
              <label className="field">
                <span>动作</span>
                <select value={governanceAction} onChange={(event) => setGovernanceAction(event.target.value as 'deprecate' | 'yank')}>
                  <option value="deprecate">deprecate</option>
                  <option value="yank">yank</option>
                </select>
              </label>
              <label className="field">
                <span>原因</span>
                <textarea
                  onChange={(event) => setGovernanceReason(event.target.value)}
                  placeholder="已有替代版本 / 安全问题 / 行为不兼容"
                  value={governanceReason}
                />
              </label>
              <label className="field">
                <span>替代版本建议</span>
                <input
                  onChange={(event) => setReplacementRecommendation(event.target.value)}
                  placeholder="1.2.1"
                  value={replacementRecommendation}
                />
              </label>
              <label className="checkbox-row">
                <input
                  checked={governanceConfirmed}
                  onChange={(event) => setGovernanceConfirmed(event.target.checked)}
                  type="checkbox"
                />
                <span>我理解这会改变下游消费者的可安装性。</span>
              </label>
              <div className="action-row">
                <div className="action-row__copy">
                  <strong>变更安装行为</strong>
                  <p>更新下游解析结果。</p>
                </div>
              </div>
              <button
                className="button button--secondary"
                disabled={!canSubmitGovernance}
                onClick={() => void handleVersionGovernance()}
                type="button"
              >
                {actionState.loading === 'governance' ? '提交中…' : '提交治理变更'}
              </button>
            </div>
          </PermissionGate>
        </SurfaceCard>

        <SurfaceCard title="已应用治理">
          <div className="list-stack">
            {governance.data?.items.length ? (
              governance.data.items.map((item) => (
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
              <div className="empty-box">这个版本还没有治理记录。</div>
            )}
          </div>
          {governance.data?.pageInfo.hasNextPage ? <p className="support-copy">这里只展示第一页版本治理；完整时间线请到治理页继续翻页。</p> : null}
        </SurfaceCard>
      </div>
    </div>
  );
}
