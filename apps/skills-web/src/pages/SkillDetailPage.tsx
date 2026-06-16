import { useState } from 'react';
import { Link, useParams, useSearchParams } from 'react-router-dom';
import {
  MutationResultPanel,
  readMutationResultSearchParams,
} from '../components/MutationResultPanel';
import { InstallCommandPanel } from '../components/InstallCommandPanel';
import { PermissionGate } from '../components/PermissionGate';
import { SectionHeading } from '../components/SectionHeading';
import { StatusPill } from '../components/StatusPill';
import { SurfaceCard } from '../components/SurfaceCard';
import {
  persistCatalogMutation,
} from '../features/catalog-mutations/useCatalogMutation';
import {
  useSkillDetailActions,
  type SkillLogEntry,
} from '../features/skill-detail/useSkillDetailActions';
import { useSkillDetailSnapshot } from '../features/skill-detail/useSkillDetailSnapshot';
import { formatDate } from '../lib/format';
import {
  getGovernanceActionLabel,
  getGovernanceTargetTypeLabel,
  getInstallHostLabel,
  getInstallPolicyLabel,
  rewriteInstallTarballUrl,
  getSkillStatusLabel,
  getSourceTypeLabel,
  getVersionStatusLabel,
} from '../lib/installCommand';
import { useRegistrySession } from '../session/SessionProvider';

function describeSkillInstallPolicy(installPolicy?: 'catalog_only' | 'resolvable' | 'blocked' | null) {
  return installPolicy ? getInstallPolicyLabel(installPolicy) : '—';
}

function getLogOutcomeLabel(outcome: SkillLogEntry['outcome']) {
  return outcome === 'succeeded' ? '成功' : '失败';
}

export function SkillDetailPage() {
  const { packageName = '' } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const decodedPackage = decodeURIComponent(packageName);
  const { session, hasCapability, getPosture, getPostureLabel } = useRegistrySession();
  const [refreshKey, setRefreshKey] = useState(0);
  const [retireReason, setRetireReason] = useState('');
  const [retireConfirmed, setRetireConfirmed] = useState(false);

  const {
    detail,
    versionList,
    installTarget,
    governance,
    latestVersionItem,
    installTargetVersionItem,
    statusTone,
    hasInstallTarget,
    installVersion,
  } = useSkillDetailSnapshot({
    packageName: decodedPackage,
    refreshKey,
  });
  const installRegistryHost = getInstallHostLabel(installTarget.data?.npmRegistry);
  const installTarballUrl = rewriteInstallTarballUrl(
    installTarget.data?.tarballUrl,
    installTarget.data?.npmRegistry
  );
  const governancePosture = getPosture({
    packageName: decodedPackage,
    requiredAction: 'catalog.governance.skill.apply',
  });
  const governanceWriteEnabled = hasCapability('catalog.governance.skill.apply', decodedPackage);
  const governancePostureLabel =
    governancePosture === 'write_enabled'
      ? '允许退役'
      : governancePosture === 'read_only'
        ? '只读'
        : getPostureLabel({
            packageName: decodedPackage,
            requiredAction: 'catalog.governance.skill.apply',
          });
  const mutationResult = readMutationResultSearchParams(searchParams, {
    packageName: decodedPackage,
    version: null,
    actor: session.data?.actor ?? null,
  });
  const mutationDefaults = {
    packageName: decodedPackage,
    version: null,
    actor: session.data?.actor ?? null,
  };
  const { submitState, activityLog, handleRetireSkill } = useSkillDetailActions({
    packageName: decodedPackage,
    onMutationResult: (result) => {
      setSearchParams(
        persistCatalogMutation(
          searchParams,
          {
            action: result.action,
            outcome: result.outcome,
            message: result.message,
            requestId: result.requestId,
          },
          mutationDefaults
        ),
        { replace: true }
      );
    },
    onRefresh: () => {
      setRefreshKey((current) => current + 1);
    },
  });
  const canRetireSkill = retireReason.trim() !== '' && retireConfirmed && !submitState.loading;

  function renderInstallSurface() {
    if (installTarget.loading) {
      return (
        <SurfaceCard
          description="正在等待安装目标返回，安装命令会在解析完成后显示在这里。"
          title="安装状态"
          variant="hero"
        >
          <div className="empty-box">正在加载安装目标，请稍候。</div>
        </SurfaceCard>
      );
    }

    if (installTarget.data && versionList.loading) {
      return (
        <SurfaceCard
          description="安装目标已解析，正在等待对应版本的安装策略返回。"
          title="安装状态"
          variant="hero"
        >
          <div className="empty-box">正在加载安装策略，请稍候。</div>
        </SurfaceCard>
      );
    }

    if (installTarget.error) {
      return (
        <SurfaceCard
          description="安装目标请求失败，暂时无法生成可复制的默认安装命令。"
          title="安装状态"
          variant="hero"
        >
          <div className="error-box">{installTarget.error}</div>
        </SurfaceCard>
      );
    }

    if (!installTarget.data) {
      return (
        <SurfaceCard description="等安装目标被仓库解析出来后，这里会显示可直接复制的命令。" title="安装状态" variant="hero">
          <div className="empty-box">当前还没有可用的安装目标。请先检查最新版本是否已经发布并可解析。</div>
        </SurfaceCard>
      );
    }

    if (versionList.error) {
      return (
        <SurfaceCard
          description="安装目标已解析，但对应版本策略请求失败，暂时无法生成可复制的默认安装命令。"
          title="安装状态"
          variant="hero"
        >
          <div className="error-box">{versionList.error}</div>
        </SurfaceCard>
      );
    }

    if (!installTargetVersionItem) {
      return (
        <SurfaceCard
          description="安装目标已经解析，但还没有拿到对应版本的安装策略。"
          title="安装状态"
          variant="hero"
        >
          <div className="empty-box">当前安装目标缺少对应的版本策略，请先检查版本列表是否完整返回。</div>
        </SurfaceCard>
      );
    }

    return (
      <InstallCommandPanel
        title="安装这个技能"
        packageName={decodedPackage}
        version={installTargetVersionItem.version}
        npmRegistry={installTarget.data.npmRegistry}
        installPolicy={installTargetVersionItem.installPolicy}
        tarballUrl={installTarget.data.tarballUrl}
        integrity={installTarget.data.integrity}
      />
    );
  }

  return (
    <div className="page-grid">
      <SectionHeading
        actions={
          <div className="authoring-actions">
            <Link className="button button--secondary" to="/catalog">
              返回目录
            </Link>
            {latestVersionItem ? (
              <Link
                className="button button--primary"
                to={`/skills/${encodeURIComponent(decodedPackage)}/versions/${latestVersionItem.version}`}
              >
                查看最新版本
              </Link>
            ) : null}
          </div>
        }
        eyebrow="技能详情"
        title={detail.data?.displayName || decodedPackage}
        description={detail.data?.description || '正在加载技能详情…'}
        tone="detail"
      />

      <div className="context-strip">
        <StatusPill tone={statusTone}>{detail.data?.status ? getSkillStatusLabel(detail.data.status) : '未知'}</StatusPill>
        {detail.data?.sourceType ? <StatusPill tone="accent">{getSourceTypeLabel(detail.data.sourceType)}</StatusPill> : null}
        {detail.data?.latestVersion ? <StatusPill>最新 {detail.data.latestVersion}</StatusPill> : null}
        {installTarget.data?.version ? <StatusPill tone="good">安装目标 {installTarget.data.version}</StatusPill> : null}
        <StatusPill tone={governancePosture === 'write_enabled' ? 'accent' : governancePosture === 'package_filter_required' ? 'warn' : 'default'}>
          {governancePostureLabel}
        </StatusPill>
      </div>

      {renderInstallSurface()}

      <div className="console-strip">
        <article className="console-signal">
              <span>状态</span>
              <strong>{detail.data?.status === 'active' ? '可继续操作' : '仅历史查看'}</strong>
              <p>{detail.data?.status === 'active' ? '当前可继续审阅和治理。' : '当前只保留历史信息。'}</p>
            </article>
            <article className="console-signal">
              <span>版本数</span>
              <strong>{versionList.data?.items.length || 0}</strong>
              <p>已记录的版本数量。</p>
            </article>
            <article className="console-signal">
              <span>治理记录</span>
              <strong>{governance.data?.items.length || 0}</strong>
              <p>已经落库的治理动作。</p>
            </article>
      </div>

      <div className="two-column">
        <SurfaceCard title="身份信息" variant="spotlight">
          <dl className="property-list">
            <div>
              <dt>包名</dt>
              <dd>{decodedPackage}</dd>
            </div>
            <div>
              <dt>来源类型</dt>
              <dd>{detail.data?.sourceType ? getSourceTypeLabel(detail.data.sourceType) : '—'}</dd>
            </div>
            <div>
              <dt>技能状态</dt>
              <dd>
                {detail.data ? (
                  <StatusPill tone={detail.data.status === 'active' ? 'good' : 'bad'}>{getSkillStatusLabel(detail.data.status)}</StatusPill>
                ) : (
                  '—'
                )}
              </dd>
            </div>
            <div>
              <dt>最新版本</dt>
              <dd>{detail.data?.latestVersion || '—'}</dd>
            </div>
          </dl>
        </SurfaceCard>

        <SurfaceCard description="安装来源和摘要数据放在首屏，方便在复制命令前先做核对。" title="安装摘要" variant="rail">
          <dl className="property-list">
            <div>
              <dt>安装版本</dt>
              <dd>{installVersion || '—'}</dd>
            </div>
            <div>
              <dt>仓库地址</dt>
              <dd>{hasInstallTarget ? installRegistryHost : '—'}</dd>
            </div>
            <div>
              <dt>制品包地址</dt>
              <dd>{installTarballUrl || '—'}</dd>
            </div>
            <div>
              <dt>完整性校验</dt>
              <dd>{installTarget.data?.integrity || '—'}</dd>
            </div>
            <div>
              <dt>清单摘要</dt>
              <dd>{installTarget.data?.skillManifestDigest || '—'}</dd>
            </div>
            <div>
              <dt>安装策略</dt>
              <dd>{describeSkillInstallPolicy(installTargetVersionItem?.installPolicy)}</dd>
            </div>
          </dl>
          <div className="list-card__actions">
            {latestVersionItem ? (
              <Link
                className="button button--secondary"
                to={`/skills/${encodeURIComponent(decodedPackage)}/versions/${latestVersionItem.version}`}
              >
                查看最新版本
              </Link>
            ) : null}
            <Link className="button button--secondary" to="/catalog">
              返回目录
            </Link>
          </div>
        </SurfaceCard>
      </div>

      <div className="two-column">
        <SurfaceCard title="版本列表">
          <div className="list-stack">
            {versionList.data?.items.map((item) => (
              <article className="list-card" key={item.version}>
                <div className="list-card__head">
                  <div>
                    <strong>{item.version}</strong>
                    <span>{formatDate(item.publishedAt)}</span>
                  </div>
                  <StatusPill tone={item.versionStatus === 'published' ? 'good' : item.versionStatus === 'deprecated' ? 'warn' : 'bad'}>
                    {getVersionStatusLabel(item.versionStatus)}
                  </StatusPill>
                </div>
                <div className="pill-row">
                  <StatusPill tone={item.installPolicy === 'resolvable' ? 'good' : item.installPolicy === 'catalog_only' ? 'warn' : 'bad'}>
                    {getInstallPolicyLabel(item.installPolicy)}
                  </StatusPill>
                </div>
                <div className="list-card__actions">
                  <Link
                    className="button button--secondary"
                    to={`/skills/${encodeURIComponent(decodedPackage)}/versions/${item.version}`}
                  >
                    打开版本
                  </Link>
                </div>
              </article>
            ))}
          </div>
        </SurfaceCard>

        <div className="stack-column">
          <SurfaceCard className="surface-card--sticky" title="技能治理" variant="console">
            <MutationResultPanel result={mutationResult} />
            <PermissionGate
              fallback={<div className="empty-box">当前会话对 `catalog.governance.skill.apply` 只有只读权限。</div>}
              packageName={decodedPackage}
              requiredAction="catalog.governance.skill.apply"
            >
              <label className="field">
                <span>原因</span>
                <textarea
                  value={retireReason}
                  onChange={(event) => setRetireReason(event.target.value)}
                  placeholder="已合并到后继技能 / 不再允许出现在目录中"
                />
              </label>
              <label className="checkbox-row">
                <input
                  checked={retireConfirmed}
                  onChange={(event) => setRetireConfirmed(event.target.checked)}
                  type="checkbox"
                />
                <span>我理解这会改变整条 skill 线的可安装性。</span>
              </label>
              <div className="action-row">
                <div className="action-row__copy">
                  <strong>退役整条技能线</strong>
                  <p>这是包级治理动作。</p>
                </div>
              </div>
              <button
                className="button button--secondary"
                disabled={!canRetireSkill}
                onClick={() => void handleRetireSkill(retireReason)}
                type="button"
              >
                {submitState.loading ? '提交中…' : '退役技能'}
              </button>
            </PermissionGate>
          </SurfaceCard>

          <SurfaceCard title="下一步">
            <div className="list-stack">
              <article className="list-card">
                <div className="list-card__head">
                  <div>
                    <strong>查看最新版本</strong>
                    <span>{latestVersionItem?.version || '未找到版本'}</span>
                  </div>
                  <StatusPill tone={latestVersionItem?.versionStatus === 'published' ? 'good' : latestVersionItem?.versionStatus === 'deprecated' ? 'warn' : 'bad'}>
                    {getVersionStatusLabel(latestVersionItem?.versionStatus || 'unknown')}
                  </StatusPill>
                </div>
                <p>继续查看最新版本的详细状态。</p>
              </article>
              <article className="list-card">
                <div className="list-card__head">
                  <div>
                    <strong>仓库安装态势</strong>
                    <span>{installTarget.data?.version || '未解析'}</span>
                  </div>
                  <StatusPill tone={detail.data?.status === 'active' ? 'good' : 'bad'}>
                    {detail.data?.status ? getSkillStatusLabel(detail.data.status) : '未知'}
                  </StatusPill>
                </div>
                <p>检查当前安装解析是否正常。</p>
              </article>
            </div>
          </SurfaceCard>

          <SurfaceCard title="最近治理">
            <div className="list-stack">
              {governance.data?.items.length ? (
                governance.data.items.map((item) => (
                  <article className="list-card" key={item.id}>
                    <div className="list-card__head">
                      <div>
                        <strong>{getGovernanceActionLabel(item.action)}</strong>
                        <span>{formatDate(item.changedAt)}</span>
                      </div>
                      <StatusPill tone={item.targetType === 'skill' ? 'bad' : 'warn'}>
                        {getGovernanceTargetTypeLabel(item.targetType)}
                      </StatusPill>
                    </div>
                    <p>{item.reason}</p>
                  </article>
                ))
              ) : (
                <div className="empty-box">这个技能还没有治理记录。</div>
              )}
            </div>
          </SurfaceCard>

          <SurfaceCard title="执行日志">
            <div className="list-stack">
              {activityLog.length ? (
                activityLog.map((entry) => (
                  <article className="list-card" key={entry.id}>
                    <div className="list-card__head">
                      <div>
                        <strong>{getGovernanceActionLabel(entry.action)}</strong>
                        <span>{entry.summary}</span>
                      </div>
                      <StatusPill tone={entry.outcome === 'succeeded' ? 'good' : 'bad'}>{getLogOutcomeLabel(entry.outcome)}</StatusPill>
                    </div>
                  </article>
                ))
              ) : (
                <div className="empty-box">这个会话里还没有执行过技能级动作。</div>
              )}
            </div>
          </SurfaceCard>
        </div>
      </div>
    </div>
  );
}
