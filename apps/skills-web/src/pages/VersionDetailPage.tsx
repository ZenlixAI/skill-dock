import { useState } from 'react';
import { Link, useParams, useSearchParams } from 'react-router-dom';
import {
  applyVersionGovernanceChange,
  createReleaseRecord,
  getPackagingBundleDownloadPath,
  getVersionDetail,
  listGovernanceChanges,
} from '../api/client';
import { normalizeThrownRegistryError } from '../api/httpClient';
import {
  inferMutationFailureOutcome,
  MutationResultPanel,
  readMutationResultSearchParams,
} from '../components/MutationResultPanel';
import { InstallCommandPanel } from '../components/InstallCommandPanel';
import { PermissionGate } from '../components/PermissionGate';
import { SectionHeading } from '../components/SectionHeading';
import { StatusPill } from '../components/StatusPill';
import { SurfaceCard } from '../components/SurfaceCard';
import {
  createMutationState,
  type CatalogMutationState,
  persistCatalogMutation,
} from '../features/catalog-mutations/useCatalogMutation';
import { useAsyncData } from '../lib/useAsyncData';
import { formatDate } from '../lib/format';
import {
  getGovernanceActionLabel,
  getGovernanceTargetTypeLabel,
  getInstallPolicyLabel,
  getInstallRegistryLabel,
  getVersionStatusLabel,
} from '../lib/installCommand';
import { buildReleaseHandoffPath, readReleaseHandoff } from '../lib/releaseHandoff';
import { useRegistrySession } from '../session/SessionProvider';

type VersionLogEntry = {
  id: string;
  action: 'release-record' | 'governance';
  outcome: 'succeeded' | 'failed';
  summary: string;
};

function getLogActionLabel(action: VersionLogEntry['action']) {
  return action === 'release-record' ? '发布记录' : '治理变更';
}

function getLogOutcomeLabel(outcome: VersionLogEntry['outcome']) {
  return outcome === 'succeeded' ? '成功' : '失败';
}

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
  const [publishConfirmed, setPublishConfirmed] = useState(false);
  const [governanceAction, setGovernanceAction] = useState<'deprecate' | 'yank'>('deprecate');
  const [governanceReason, setGovernanceReason] = useState('');
  const [replacementRecommendation, setReplacementRecommendation] = useState('');
  const [governanceConfirmed, setGovernanceConfirmed] = useState(false);
  const [activityLog, setActivityLog] = useState<VersionLogEntry[]>([]);
  const [actionState, setActionState] = useState<CatalogMutationState<'release' | 'governance'>>(
    createMutationState()
  );

  const detail = useAsyncData(() => getVersionDetail(decodedPackage, version), [decodedPackage, version, refreshKey]);
  const { session, hasCapability, getPosture, getPostureLabel } = useRegistrySession();
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
  const bundleDownloadPath =
    hasHandoff && handoff.workspacePath
      ? getPackagingBundleDownloadPath(handoff.workspacePath, decodedPackage, version)
      : null;
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
  const releasePosture = getPosture({
    packageName: decodedPackage,
    requiredAction: 'catalog.release_record.create',
  });
  const releaseWriteEnabled = hasCapability('catalog.release_record.create', decodedPackage);
  const governanceWriteEnabled = hasCapability('catalog.governance.version.apply', decodedPackage);
  const releasePostureLabel =
    releasePosture === 'write_enabled'
      ? '允许发布登记'
      : releasePosture === 'read_only'
        ? '发布登记只读'
        : getPostureLabel({
            packageName: decodedPackage,
            requiredAction: 'catalog.release_record.create',
          });
  const mutationResult = readMutationResultSearchParams(searchParams, {
    packageName: decodedPackage,
    version,
    actor: session.data?.actor ?? null,
  });
  const mutationDefaults = {
    packageName: decodedPackage,
    version,
    actor: session.data?.actor ?? null,
  };
  const installRegistry = getInstallRegistryLabel(detail.data?.registry.npmRegistry);

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
      ...createMutationState<'release' | 'governance'>(),
      loading: 'release',
    });
    try {
      const response = await createReleaseRecord(decodedPackage, version, {
        publishRegistry: publishRegistry.trim() || undefined,
        manifestDigest: manifestDigest.trim() || undefined,
        components: releaseComponents.length ? releaseComponents : undefined,
      });
      const successMessage = response.created
        ? '已写入新的发布记录。'
        : '发布记录已存在，后端返回幂等重放结果。';
      setActionState({
        loading: false,
        error: null,
        success: successMessage,
      });
      setSearchParams(
        persistCatalogMutation(
          searchParams,
          {
            action: '发布记录',
            outcome: 'succeeded',
            message: successMessage,
          },
          mutationDefaults
        ),
        { replace: true }
      );
      appendLog(
        'release-record',
        'succeeded',
        response.created ? '已写入新的发布记录。' : '发布记录返回幂等重放结果。'
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
      setSearchParams(
        persistCatalogMutation(
          searchParams,
          {
            action: '发布记录',
            outcome,
            message: normalizedError.message,
            requestId: normalizedError.requestId,
          },
          mutationDefaults
        ),
        { replace: true }
      );
    }
  }

  async function handleVersionGovernance() {
    setActionState({
      ...createMutationState<'release' | 'governance'>(),
      loading: 'governance',
    });
    try {
      const response = await applyVersionGovernanceChange(decodedPackage, version, {
        action: governanceAction,
        reason: governanceReason.trim(),
        replacementRecommendation:
          replacementRecommendation.trim() === '' ? undefined : replacementRecommendation.trim(),
      });
      const successMessage = `已提交${getGovernanceActionLabel(response.record.action)}。当前版本状态：${
        getVersionStatusLabel(response.resultingState.versionStatus || 'unknown')
      }`;
      setActionState({
        loading: false,
        error: null,
        success: successMessage,
      });
      setSearchParams(
        persistCatalogMutation(
          searchParams,
          {
            action: getGovernanceActionLabel(response.record.action),
            outcome: 'succeeded',
            message: successMessage,
          },
          mutationDefaults
        ),
        { replace: true }
      );
      appendLog('governance', 'succeeded', `已应用${getGovernanceActionLabel(response.record.action)}治理动作。`);
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
      setSearchParams(
        persistCatalogMutation(
          searchParams,
          {
            action: getGovernanceActionLabel(governanceAction),
            outcome,
            message: normalizedError.message,
            requestId: normalizedError.requestId,
          },
          mutationDefaults
        ),
        { replace: true }
      );
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
        <StatusPill tone={versionTone}>{getVersionStatusLabel(currentVersionStatus)}</StatusPill>
        {detail.data?.installPolicy ? (
          <StatusPill tone={detail.data.installPolicy === 'resolvable' ? 'good' : detail.data.installPolicy === 'catalog_only' ? 'warn' : 'bad'}>
            {getInstallPolicyLabel(detail.data.installPolicy)}
          </StatusPill>
        ) : null}
        {detail.data ? <StatusPill>{installRegistry}</StatusPill> : null}
        <StatusPill tone={hasHandoff ? 'accent' : undefined}>{hasHandoff ? '已附带交接' : '直接打开'}</StatusPill>
        <StatusPill tone={releasePosture === 'write_enabled' ? 'accent' : releasePosture === 'package_filter_required' ? 'warn' : 'default'}>
          {releasePostureLabel}
        </StatusPill>
      </div>

      {detail.data ? (
        <InstallCommandPanel
          title="安装这个版本"
          packageName={decodedPackage}
          version={version}
          npmRegistry={installRegistry}
          installPolicy={detail.data.installPolicy || 'catalog_only'}
          tarballUrl={detail.data.registry.tarballUrl}
          integrity={detail.data.registry.integrity}
        />
      ) : (
        <SurfaceCard description="等仓库元数据返回后，这里会显示可直接复制的版本安装命令。" title="安装状态" variant="hero">
          <div className="empty-box">当前还没有可用的仓库安装信息。</div>
        </SurfaceCard>
      )}

      <div className="console-strip">
        <article className="console-signal">
          <span>发布登记</span>
          <strong>{actionState.loading === 'release' ? '执行中' : '就绪'}</strong>
          <p>用于在打包产物已经发布到 npm 仓库之后写入发布记录。</p>
        </article>
        <article className="console-signal">
          <span>治理动作</span>
          <strong>{actionState.loading === 'governance' ? '执行中' : '就绪'}</strong>
          <p>可执行弃用或撤回。</p>
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
                  <span>{detail.data?.installPolicy ? getInstallPolicyLabel(detail.data.installPolicy) : '未知'}</span>
                </div>
                <StatusPill tone={detail.data?.installPolicy === 'resolvable' ? 'good' : detail.data?.installPolicy === 'catalog_only' ? 'warn' : 'bad'}>
                    {detail.data?.installPolicy ? getInstallPolicyLabel(detail.data.installPolicy) : '未知'}
                  </StatusPill>
              </div>
              <p>
                {detail.data?.installPolicy === 'blocked'
                  ? '下游安装解析应被阻断。'
                  : detail.data?.installPolicy === 'catalog_only'
                    ? '版本保留在目录中，但不应再作为默认安装目标。'
                    : '当前版本仍可被正常解析和安装。'}
              </p>
            </article>
            <article className="list-card">
              <div className="list-card__head">
                <div>
                  <strong>当前治理影响</strong>
                  <span>{getGovernanceActionLabel(governanceAction)}</span>
                </div>
                <StatusPill tone={governanceAction === 'yank' ? 'bad' : 'warn'}>{getGovernanceActionLabel(governanceAction)}</StatusPill>
              </div>
              <p>{governanceImpact}</p>
            </article>
          </div>
        </SurfaceCard>

        <SurfaceCard description="复制命令前，先核对仓库与打包产物摘要。" title="仓库与打包产物">
          <dl className="property-list">
            <div>
              <dt>版本状态</dt>
              <dd>
                {detail.data ? (
                  <StatusPill tone={detail.data.versionStatus === 'published' ? 'good' : detail.data.versionStatus === 'deprecated' ? 'warn' : 'bad'}>
                    {getVersionStatusLabel(detail.data.versionStatus)}
                  </StatusPill>
                ) : (
                  '—'
                )}
              </dd>
            </div>
            <div>
              <dt>安装策略</dt>
              <dd>{detail.data?.installPolicy ? getInstallPolicyLabel(detail.data.installPolicy) : '—'}</dd>
            </div>
            <div>
              <dt>npm 仓库地址</dt>
              <dd>{detail.data ? installRegistry : '—'}</dd>
            </div>
            <div>
              <dt>完整性校验</dt>
              <dd>{detail.data?.registry.integrity || '—'}</dd>
            </div>
            <div>
              <dt>校验摘要</dt>
              <dd>{detail.data?.registry.shasum || '—'}</dd>
            </div>
            <div>
              <dt>清单摘要</dt>
              <dd>{detail.data?.bundle.skillManifestDigest || '—'}</dd>
            </div>
            <div>
              <dt>组件清单</dt>
              <dd>{detail.data?.bundle.components.join(', ') || '—'}</dd>
            </div>
          </dl>
        </SurfaceCard>
      </div>

      {hasHandoff ? (
        <SurfaceCard
          action={
            <Link className="button button--secondary" to={authoringPath}>
              返回作者流程
            </Link>
          }
          description="交接信息保留给发布登记流程，不再抢占安装首屏。"
          title="发布交接"
          variant="spotlight"
        >
          <dl className="property-list">
            <div>
              <dt>工作区路径</dt>
              <dd>{handoff.workspacePath || '—'}</dd>
            </div>
            <div>
              <dt>打包产物路径</dt>
              <dd>{handoff.bundlePath || '—'}</dd>
            </div>
            <div>
              <dt>清单摘要</dt>
              <dd>{handoff.manifestDigest || '—'}</dd>
            </div>
            <div>
              <dt>组件清单</dt>
              <dd>{handoff.components?.join(', ') || '—'}</dd>
            </div>
          </dl>
        </SurfaceCard>
      ) : null}

      <MutationResultPanel result={mutationResult} />

      <div className="two-column">
        <SurfaceCard title="执行日志">
          <div className="list-stack">
            {activityLog.length ? (
              activityLog.map((entry) => (
                  <article className="list-card" key={entry.id}>
                    <div className="list-card__head">
                      <div>
                        <strong>{getLogActionLabel(entry.action)}</strong>
                        <span>{entry.summary}</span>
                      </div>
                      <StatusPill tone={entry.outcome === 'succeeded' ? 'good' : 'bad'}>{getLogOutcomeLabel(entry.outcome)}</StatusPill>
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
        <SurfaceCard className="surface-card--sticky" title="发布登记" variant="console">
          <PermissionGate
            fallback={<div className="empty-box">当前会话对 `catalog.release_record.create` 只有只读权限。</div>}
            packageName={decodedPackage}
            requiredAction="catalog.release_record.create"
          >
            <div className="stack-column">
              <label className="field">
                <span>发布目标仓库</span>
                <input
                  onChange={(event) => setPublishRegistry(event.target.value)}
                  placeholder={detail.data?.registry.npmRegistry || 'https://npm.internal.example.com'}
                  value={publishRegistry}
                />
              </label>
              <label className="field">
                <span>清单摘要</span>
                <input
                  onChange={(event) => setManifestDigest(event.target.value)}
                  placeholder={detail.data?.bundle.skillManifestDigest || 'sha256:...'}
                  value={manifestDigest}
                />
              </label>
              <label className="field">
                <span>组件清单</span>
                <input
                  onChange={(event) => setComponents(event.target.value)}
                  placeholder={detail.data?.bundle.components.join(', ') || '命令, 脚本'}
                  value={components}
                />
              </label>
              <div className="action-row">
                <div className="action-row__copy">
                  <strong>登记这个打包产物</strong>
                  <p>先确认这个版本已经完成 `npm publish`，再为它写入目录发布记录。</p>
                </div>
              </div>
              <div className="list-stack">
                <article className="list-card">
                  <div className="list-card__head">
                    <div>
                      <strong>发布前检查清单</strong>
                      <span>发布记录不会替你执行 `npm publish`。</span>
                    </div>
                    <StatusPill tone={publishConfirmed ? 'good' : 'warn'}>
                      {publishConfirmed ? '已确认' : '待确认'}
                    </StatusPill>
                  </div>
                  <p>先下载打包产物，并在你自己的终端里对目标 npm 仓库执行 `npm publish`。</p>
                  {bundleDownloadPath ? (
                    <div className="authoring-actions">
                      <a className="button button--secondary" href={bundleDownloadPath}>
                        下载打包产物
                      </a>
                      <span className="support-copy">{handoff.bundlePath || '使用当前交接信息里的打包产物结果。'}</span>
                    </div>
                  ) : (
                    <p className="support-copy">如果你不是从发布交接进入这个页面，请先回到导入页重新生成打包产物。</p>
                  )}
                </article>
              </div>
              <label className="checkbox-row">
                <input
                  aria-label="我确认这个版本已经完成 npm publish，可继续写入发布记录。"
                  checked={publishConfirmed}
                  onChange={(event) => setPublishConfirmed(event.target.checked)}
                  type="checkbox"
                />
                <span>我确认这个版本已经完成 npm publish，可继续写入发布记录。</span>
              </label>
              <button
                className="button button--primary"
                disabled={actionState.loading !== false || !publishConfirmed}
                onClick={() => void handleReleaseRecord()}
                type="button"
              >
                {actionState.loading === 'release' ? '提交中…' : '写入发布记录'}
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
          >
            <div className="stack-column">
              <label className="field">
                <span>动作</span>
                <select value={governanceAction} onChange={(event) => setGovernanceAction(event.target.value as 'deprecate' | 'yank')}>
                  <option value="deprecate">弃用</option>
                  <option value="yank">撤回</option>
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
                      <strong>{getGovernanceActionLabel(item.action)}</strong>
                      <span>{formatDate(item.changedAt)}</span>
                    </div>
                    <StatusPill tone={item.action === 'yank' ? 'bad' : 'warn'}>
                      {getGovernanceTargetTypeLabel(item.targetType)}
                    </StatusPill>
                  </div>
                  <p>{item.reason}</p>
                  {item.replacementRecommendation ? (
                    <div className="pill-row">
                      <StatusPill tone="accent">替代版本 {item.replacementRecommendation}</StatusPill>
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
