import { useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import {
  cleanupPackagingWorkspace,
  getSession,
  packSkill,
  preparePackaging,
  validatePackaging,
} from '../api/client';
import { normalizeThrownRegistryError } from '../api/httpClient';
import { canPerformAction } from '../api/session';
import type {
  PackagingPackResponse,
  PackagingPrepareRequest,
  PackagingPrepareResponse,
  PackagingSourceType,
  PackagingValidateResponse,
  RegistryRequestError,
} from '../api/types';
import { ImportReportCard } from '../components/ImportReportCard';
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
import { WarningStack } from '../components/WarningStack';
import { buildReleaseHandoffPath } from '../lib/releaseHandoff';
import { useAsyncData } from '../lib/useAsyncData';

const steps = [
  {
    title: '1. 来源',
    description: 'git / platform_authoring',
  },
  {
    title: '2. 目标',
    description: 'packageName / version / strict / publishRegistry',
  },
  {
    title: '3. 校验',
    description: 'warningDetails + importReport + components',
  },
  {
    title: '4. 打包 / 发布',
    description: 'bundle metadata + release handoff',
  },
];

type ReviewData = PackagingPrepareResponse | PackagingValidateResponse | PackagingPackResponse;
type ImportLogEntry = {
  id: string;
  step: 'prepare' | 'validate' | 'pack' | 'cleanup';
  outcome: 'succeeded' | 'failed';
  summary: string;
};

function parseInlineFiles(raw: string) {
  const trimmed = raw.trim();
  if (trimmed === '') {
    return undefined;
  }

  const parsed = JSON.parse(trimmed) as unknown;
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('Inline files 必须是 JSON object，例如 {"commands/run.txt":"echo hi"}。');
  }

  const files: Record<string, string> = {};
  Object.entries(parsed).forEach(([path, content]) => {
    if (typeof content !== 'string') {
      throw new Error(`Inline file ${path} 的内容必须是字符串。`);
    }
    files[path] = content;
  });
  return files;
}

function hasComponents(data: ReviewData): data is PackagingValidateResponse | PackagingPackResponse {
  return 'components' in data;
}

function hasManifestDigest(data: ReviewData): data is PackagingValidateResponse | PackagingPackResponse {
  return 'manifestDigest' in data;
}

function hasBundlePath(data: ReviewData): data is PackagingPackResponse {
  return 'bundlePath' in data;
}

function createLocalRequestError(message: string): RegistryRequestError {
  return {
    kind: 'unknown',
    message,
  };
}

function describeImportOperationError(error: RegistryRequestError) {
  if (error.kind === 'auth') {
    return `${error.message} 这次导入操作需要具备 packaging 权限的会话后再重试。`;
  }

  if (error.kind === 'network') {
    return `${error.message} 当前无法连接 packaging 后端，等真实后端恢复后再重试。`;
  }

  if (error.kind === 'server') {
    return error.message;
  }

  if (error.kind === 'not_found') {
    return `${error.message} 对应的 workspace 或引用的 registry 对象可能已经被清理。`;
  }

  return error.message;
}

export function ImportPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const session = useAsyncData(() => getSession(), []);
  const [sourceType, setSourceType] = useState<PackagingSourceType>('git');
  const [location, setLocation] = useState('https://github.com/mattpocock/skills?path=skills/pdf');
  const [sourceRef, setSourceRef] = useState('main');
  const [inlineFiles, setInlineFiles] = useState('{\n  "commands/run.txt": "echo authored"\n}');
  const [packageName, setPackageName] = useState('@company.skill/pdf-skill');
  const [version, setVersion] = useState('1.2.0');
  const [displayName, setDisplayName] = useState('PDF 技能');
  const [description, setDescription] = useState('PDF 处理工作流');
  const [metadataSourceType, setMetadataSourceType] = useState('imported');
  const [publishRegistry, setPublishRegistry] = useState('https://npm.internal.example.com');
  const [strictMode, setStrictMode] = useState(false);
  const [includeBundle, setIncludeBundle] = useState(false);
  const [operationState, setOperationState] = useState<{
    loading: false | 'prepare' | 'validate' | 'pack' | 'cleanup';
    error: string | null;
    errorDetail: RegistryRequestError | null;
    success: string | null;
  }>({
    loading: false,
    error: null,
    errorDetail: null,
    success: null,
  });
  const [prepareResult, setPrepareResult] = useState<PackagingPrepareResponse | null>(null);
  const [validateResult, setValidateResult] = useState<PackagingValidateResponse | null>(null);
  const [packResult, setPackResult] = useState<PackagingPackResponse | null>(null);
  const [activityLog, setActivityLog] = useState<ImportLogEntry[]>([]);

  const reviewData: ReviewData | null = packResult || validateResult || prepareResult;
  const activeStepIndex = packResult ? 3 : validateResult ? 2 : prepareResult ? 1 : 0;
  const currentStage = packResult ? 'pack_complete' : validateResult ? 'validated' : prepareResult ? 'prepared' : 'draft';
  const stageTone =
    currentStage === 'pack_complete' ? 'good' : currentStage === 'validated' ? 'accent' : currentStage === 'prepared' ? 'warn' : undefined;
  const releaseHandoffPath = packResult
    ? buildReleaseHandoffPath({
        packageName: packResult.packageName,
        version: packResult.version,
        publishRegistry: publishRegistry.trim() || undefined,
        manifestDigest: packResult.manifestDigest,
        components: packResult.components,
        workspacePath: packResult.workspacePath,
        bundlePath: packResult.bundlePath,
      })
    : null;
  const authoringHandoffPath = packResult
    ? buildReleaseHandoffPath(
        {
          packageName: packResult.packageName,
          version: packResult.version,
          publishRegistry: publishRegistry.trim() || undefined,
          manifestDigest: packResult.manifestDigest,
          components: packResult.components,
          workspacePath: packResult.workspacePath,
          bundlePath: packResult.bundlePath,
        },
        '/workbench/authoring'
      )
    : null;
  const canValidate = Boolean(prepareResult?.workspacePath) && operationState.loading === false;
  const canPack = Boolean(validateResult?.ok) && operationState.loading === false;
  const canCleanup = Boolean(prepareResult?.workspacePath) && operationState.loading === false;
  const normalizedPackageName = packageName.trim();
  const cleanupPackageName =
    packResult?.packageName || validateResult?.packageName || prepareResult?.packageName || normalizedPackageName;
  const packagingWriteEnabled = canPerformAction(session.data, 'packaging.pack', normalizedPackageName);
  const mutationResult = readMutationResultSearchParams(searchParams, {
    packageName: null,
    version: null,
    actor: session.data?.actor ?? null,
  });

  function appendLog(step: ImportLogEntry['step'], outcome: ImportLogEntry['outcome'], summary: string) {
    setActivityLog((current) => [
      {
        id: `${step}-${Date.now()}`,
        step,
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
    packageName?: string | null;
    version?: string | null;
  }) {
    setSearchParams(
      writeMutationResultSearchParams(searchParams, {
        ...result,
        actor: session.data?.actor ?? null,
      }),
      { replace: true }
    );
  }

  function buildPrepareRequest(): PackagingPrepareRequest {
    return {
      source:
        sourceType === 'platform_authoring'
          ? {
              type: sourceType,
              files: parseInlineFiles(inlineFiles),
            }
          : {
              type: sourceType,
              location: location.trim(),
              ref: sourceType === 'git' && sourceRef.trim() !== '' ? sourceRef.trim() : undefined,
            },
      target: {
        packageName: packageName.trim(),
        version: version.trim(),
      },
      metadata: {
        displayName: displayName.trim(),
        description: description.trim(),
        sourceType: metadataSourceType,
      },
      options: {
        publishRegistry: publishRegistry.trim() || undefined,
        strict: strictMode,
      },
    };
  }

  async function handlePrepare() {
    setOperationState({
      loading: 'prepare',
      error: null,
      errorDetail: null,
      success: null,
    });
    try {
      const response = await preparePackaging(buildPrepareRequest());
      setPrepareResult(response);
      setValidateResult(null);
      setPackResult(null);
      setOperationState({
        loading: false,
        error: null,
        errorDetail: null,
        success: 'prepare 完成，已经拿到 workspacePath 和 import review 数据。',
      });
      persistMutationResult({
        action: 'prepare',
        outcome: 'succeeded',
        message: 'prepare 完成，已经拿到 workspacePath 和 import review 数据。',
        packageName: response.packageName,
        version: response.version,
      });
      appendLog('prepare', 'succeeded', `已准备 ${response.packageName}@${response.version}。`);
    } catch (error: unknown) {
      const outcome = inferMutationFailureOutcome(error);
      const normalizedError = normalizeThrownRegistryError(error);
      const message = describeImportOperationError(normalizedError);
      appendLog('prepare', 'failed', normalizedError.message);
      setOperationState({
        loading: false,
        error: normalizedError.message,
        errorDetail: normalizedError,
        success: null,
      });
      persistMutationResult({
        action: 'prepare',
        outcome,
        message,
        requestId: normalizedError.requestId,
        packageName: normalizedPackageName || null,
        version: version.trim() || null,
      });
    }
  }

  async function handleValidate() {
    if (!prepareResult?.workspacePath) {
      setOperationState({
        loading: false,
        error: '请先执行 prepare。',
        errorDetail: createLocalRequestError('请先执行 prepare。'),
        success: null,
      });
      persistMutationResult({
        action: 'validate',
        outcome: 'failed',
        message: '请先执行 prepare。',
        packageName: normalizedPackageName || null,
        version: version.trim() || null,
      });
      return;
    }
    setOperationState({
      loading: 'validate',
      error: null,
      errorDetail: null,
      success: null,
    });
    try {
      const response = await validatePackaging({
        workspacePath: prepareResult.workspacePath,
        target: {
          packageName: packageName.trim(),
          version: version.trim(),
        },
      });
      setValidateResult(response);
      setPackResult(null);
      setOperationState({
        loading: false,
        error: null,
        errorDetail: null,
        success: response.ok
          ? 'validate 通过，可以继续 pack。'
          : 'validate 返回未通过，请先处理 errors。',
      });
      persistMutationResult({
        action: 'validate',
        outcome: response.ok ? 'succeeded' : 'failed',
        message: response.ok ? 'validate 通过，可以继续 pack。' : 'validate 返回未通过，请先处理 errors。',
        packageName: response.packageName,
        version: response.version,
      });
      appendLog('validate', response.ok ? 'succeeded' : 'failed', response.ok ? '校验通过。' : '校验返回阻断错误。');
    } catch (error: unknown) {
      const outcome = inferMutationFailureOutcome(error);
      const normalizedError = normalizeThrownRegistryError(error);
      const message = describeImportOperationError(normalizedError);
      appendLog('validate', 'failed', normalizedError.message);
      setOperationState({
        loading: false,
        error: normalizedError.message,
        errorDetail: normalizedError,
        success: null,
      });
      persistMutationResult({
        action: 'validate',
        outcome,
        message,
        requestId: normalizedError.requestId,
        packageName: prepareResult.packageName || normalizedPackageName || null,
        version: prepareResult.version || version.trim() || null,
      });
    }
  }

  async function handlePack() {
    const workspacePath = validateResult?.workspacePath || prepareResult?.workspacePath;
    if (!workspacePath) {
      setOperationState({
        loading: false,
        error: '请先执行 prepare，建议再执行 validate。',
        errorDetail: createLocalRequestError('请先执行 prepare，建议再执行 validate。'),
        success: null,
      });
      persistMutationResult({
        action: 'pack',
        outcome: 'failed',
        message: '请先执行 prepare，建议再执行 validate。',
        packageName: normalizedPackageName || null,
        version: version.trim() || null,
      });
      appendLog('pack', 'failed', '在 validate 通过前就请求了 pack。');
      return;
    }
    setOperationState({
      loading: 'pack',
      error: null,
      errorDetail: null,
      success: null,
    });
    try {
      const response = await packSkill({
        workspacePath,
        target: {
          packageName: packageName.trim(),
          version: version.trim(),
        },
        includeBundle,
      });
      setPackResult(response);
      setOperationState({
        loading: false,
        error: null,
        errorDetail: null,
        success: 'pack 完成，已经拿到 bundlePath / manifestDigest / components。',
      });
      persistMutationResult({
        action: 'pack',
        outcome: 'succeeded',
        message: 'pack 完成，已经拿到 bundlePath / manifestDigest / components。',
        packageName: response.packageName,
        version: response.version,
      });
      appendLog('pack', 'succeeded', `已打包 bundle ${response.bundlePath}。`);
    } catch (error: unknown) {
      const outcome = inferMutationFailureOutcome(error);
      const normalizedError = normalizeThrownRegistryError(error);
      const message = describeImportOperationError(normalizedError);
      appendLog('pack', 'failed', normalizedError.message);
      setOperationState({
        loading: false,
        error: normalizedError.message,
        errorDetail: normalizedError,
        success: null,
      });
      persistMutationResult({
        action: 'pack',
        outcome,
        message,
        requestId: normalizedError.requestId,
        packageName: validateResult?.packageName || prepareResult?.packageName || normalizedPackageName || null,
        version: validateResult?.version || prepareResult?.version || version.trim() || null,
      });
    }
  }

  async function handleCleanup() {
    const workspacePath = packResult?.workspacePath || validateResult?.workspacePath || prepareResult?.workspacePath;
    if (!workspacePath) {
      setOperationState({
        loading: false,
        error: '没有可清理的 workspace。',
        errorDetail: createLocalRequestError('没有可清理的 workspace。'),
        success: null,
      });
      persistMutationResult({
        action: 'cleanup',
        outcome: 'failed',
        message: '没有可清理的 workspace。',
        packageName: null,
        version: null,
      });
      return;
    }

    setOperationState({
      loading: 'cleanup',
      error: null,
      errorDetail: null,
      success: null,
    });

    try {
      await cleanupPackagingWorkspace({ workspacePath });
      setPrepareResult(null);
      setValidateResult(null);
      setPackResult(null);
      setOperationState({
        loading: false,
        error: null,
        errorDetail: null,
        success: 'workspace 已清理，当前 packaging session 已重置。',
      });
      persistMutationResult({
        action: 'cleanup',
        outcome: 'succeeded',
        message: 'workspace 已清理，当前 packaging session 已重置。',
        packageName: null,
        version: null,
      });
      appendLog('cleanup', 'succeeded', `已清理 workspace ${workspacePath}。`);
    } catch (error: unknown) {
      const outcome = inferMutationFailureOutcome(error);
      const normalizedError = normalizeThrownRegistryError(error);
      const message = describeImportOperationError(normalizedError);
      appendLog('cleanup', 'failed', normalizedError.message);
      setOperationState({
        loading: false,
        error: normalizedError.message,
        errorDetail: normalizedError,
        success: null,
      });
      persistMutationResult({
        action: 'cleanup',
        outcome,
        message,
        requestId: normalizedError.requestId,
        packageName: null,
        version: null,
      });
    }
  }

  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="导入与打包"
        title="把 prepare → validate → pack 做成显式工作流，而不是一键黑盒。"
        description="这个页面已经可以直连真实 packaging API。前端不做规则判断，只把 warningDetails 和 importReport 变成用户能看懂的确认面。"
        tone="backstage"
      />

      <div className="context-strip">
        <StatusPill tone="accent">来源 {sourceType}</StatusPill>
        <StatusPill>{packageName}</StatusPill>
        <StatusPill>{version}</StatusPill>
        <StatusPill tone={strictMode ? 'warn' : 'good'}>{strictMode ? '严格' : '宽松'}</StatusPill>
        <StatusPill tone={stageTone}>阶段 {currentStage}</StatusPill>
        <StatusPill tone={packagingWriteEnabled ? 'accent' : 'default'}>
          {session.loading ? '会话加载中' : session.error ? '会话不可用' : packagingWriteEnabled ? '允许打包' : '打包只读'}
        </StatusPill>
      </div>

      <div className="two-column two-column--wide-right">
        <SurfaceCard title="工作流" description="按当前同步的 packaging API 合同拆成 4 步。" variant="console">
          <div className="step-list">
            {steps.map((step, index) => (
              <article className={index === activeStepIndex ? 'step-card step-card--active' : 'step-card'} key={step.title}>
                <strong>{step.title}</strong>
                <p>{step.description}</p>
              </article>
            ))}
          </div>

          <div className="console-strip">
            <article className="console-signal">
              <span>当前阶段</span>
              <strong>{currentStage}</strong>
              <p>{packResult ? 'bundle 已可交接到 release review。' : validateResult ? '校验完成，等待打包。' : prepareResult ? 'workspace 已准备，建议继续 validate。' : '先填写 source 和 target。'}</p>
            </article>
            <article className="console-signal">
              <span>工作目录</span>
              <strong>{prepareResult?.workspacePath ? 'ready' : 'pending'}</strong>
              <p>{prepareResult?.workspacePath || 'prepare 之后这里会出现临时工作目录。'}</p>
            </article>
            <article className="console-signal">
              <span>发布交接</span>
              <strong>{releaseHandoffPath ? 'armed' : 'idle'}</strong>
              <p>{releaseHandoffPath ? 'pack 结果已经能直接带到 release review。' : '完成 pack 后才会开放下游 handoff。'}</p>
            </article>
          </div>

          <div className="stack-column form-section">
            <label className="field">
              <span>来源类型</span>
              <select value={sourceType} onChange={(event) => setSourceType(event.target.value as PackagingSourceType)}>
                <option value="git">git</option>
                <option value="platform_authoring">platform_authoring</option>
              </select>
            </label>

            {sourceType === 'platform_authoring' ? (
              <label className="field">
                <span>内联文件 JSON</span>
                <textarea
                  className="code-field"
                  onChange={(event) => setInlineFiles(event.target.value)}
                  value={inlineFiles}
                />
              </label>
            ) : (
              <>
                <label className="field">
                  <span>位置</span>
                  <input
                    onChange={(event) => setLocation(event.target.value)}
                    placeholder="git 地址"
                    value={location}
                  />
                </label>
                {sourceType === 'git' ? (
                  <label className="field">
                    <span>Git 引用</span>
                    <input
                      onChange={(event) => setSourceRef(event.target.value)}
                      placeholder="main / release / commit sha"
                      value={sourceRef}
                    />
                  </label>
                ) : null}
              </>
            )}

            <div className="two-column">
              <label className="field">
                <span>packageName</span>
                <input onChange={(event) => setPackageName(event.target.value)} value={packageName} />
              </label>
              <label className="field">
                <span>version</span>
                <input onChange={(event) => setVersion(event.target.value)} value={version} />
              </label>
            </div>

            <label className="field">
              <span>显示名</span>
              <input onChange={(event) => setDisplayName(event.target.value)} value={displayName} />
            </label>
            <label className="field">
              <span>描述</span>
              <textarea onChange={(event) => setDescription(event.target.value)} value={description} />
            </label>
            <label className="field">
              <span>元数据来源类型</span>
              <select value={metadataSourceType} onChange={(event) => setMetadataSourceType(event.target.value)}>
                <option value="imported">imported</option>
                <option value="native">native</option>
                <option value="platform_created">platform_created</option>
              </select>
            </label>
            <label className="field">
              <span>发布仓地址</span>
              <input onChange={(event) => setPublishRegistry(event.target.value)} value={publishRegistry} />
            </label>

            <label className="checkbox-row">
              <input checked={strictMode} onChange={(event) => setStrictMode(event.target.checked)} type="checkbox" />
              <span>严格模式</span>
            </label>
            <label className="checkbox-row">
              <input checked={includeBundle} onChange={(event) => setIncludeBundle(event.target.checked)} type="checkbox" />
              <span>在 pack 响应里附带 bundle base64</span>
            </label>

            <MutationResultPanel result={mutationResult} />

            <div className="action-row">
              <div className="action-row__copy">
                <strong>执行下一步打包动作</strong>
                <p>先 prepare，再 validate，最后 pack。按钮状态会随链路自动变化，不需要你自己记步骤。</p>
              </div>
            </div>
            <div className="button-row">
              <PermissionGate
                fallback={<div className="empty-box">当前会话对 `packaging.prepare` 只有只读权限。</div>}
                packageName={normalizedPackageName}
                requiredAction="packaging.prepare"
                session={session.data}
                sessionError={session.error}
                sessionLoading={session.loading}
              >
                <button
                  className="button button--primary"
                  disabled={operationState.loading !== false}
                  onClick={() => void handlePrepare()}
                  type="button"
                >
                  {operationState.loading === 'prepare' ? '准备中…' : 'Prepare'}
                </button>
              </PermissionGate>
              <PermissionGate
                fallback={<div className="empty-box">当前会话对 `packaging.validate` 只有只读权限。</div>}
                packageName={normalizedPackageName}
                requiredAction="packaging.validate"
                session={session.data}
                sessionError={session.error}
                sessionLoading={session.loading}
              >
                <button
                  className="button button--secondary"
                  disabled={!canValidate}
                  onClick={() => void handleValidate()}
                  type="button"
                >
                  {operationState.loading === 'validate' ? '校验中…' : 'Validate'}
                </button>
              </PermissionGate>
              <PermissionGate
                fallback={<div className="empty-box">当前会话对 `packaging.pack` 只有只读权限。</div>}
                packageName={normalizedPackageName}
                requiredAction="packaging.pack"
                session={session.data}
                sessionError={session.error}
                sessionLoading={session.loading}
              >
                <button
                  className="button button--secondary"
                  disabled={!canPack}
                  onClick={() => void handlePack()}
                  type="button"
                >
                  {operationState.loading === 'pack' ? '打包中…' : 'Pack'}
                </button>
              </PermissionGate>
              <PermissionGate
                fallback={<div className="empty-box">当前会话对 `packaging.pack` 只有只读权限。</div>}
                packageName={cleanupPackageName}
                requiredAction="packaging.pack"
                session={session.data}
                sessionError={session.error}
                sessionLoading={session.loading}
              >
                <button
                  className="button button--secondary"
                  disabled={!canCleanup}
                  onClick={() => void handleCleanup()}
                  type="button"
                >
                  {operationState.loading === 'cleanup' ? '清理中…' : '丢弃工作目录'}
                </button>
              </PermissionGate>
            </div>
          </div>
        </SurfaceCard>

        <SurfaceCard className="surface-card--sticky" title="校验审阅" description="右侧始终展示当前 prepare / validate / pack 的最新结果。" variant="rail">
          {!reviewData ? <div className="empty-box">先执行 `prepare`，再启动这条打包工作流。</div> : null}
          {reviewData ? (
            <div className="stack-column">
              <div className="console-strip console-strip--compact">
                <article className="console-signal console-signal--compact">
                  <span>最近动作</span>
                  <strong>{operationState.loading || currentStage}</strong>
                </article>
                <article className="console-signal console-signal--compact">
                  <span>告警</span>
                  <strong>{reviewData.warningDetails.length}</strong>
                </article>
                <article className="console-signal console-signal--compact">
                  <span>错误</span>
                  <strong>{reviewData.errors.length}</strong>
                </article>
              </div>

              <div className="metric-grid metric-grid--compact">
                <div className="metric-card">
                  <strong>{reviewData.importReport?.normalized ? '是' : '否'}</strong>
                  <span>已归一化导入</span>
                </div>
                <div className="metric-card">
                  <strong>{reviewData.importReport?.strictMode ? '严格' : '宽松'}</strong>
                  <span>导入模式</span>
                </div>
                <div className="metric-card">
                  <strong>{hasComponents(reviewData) ? reviewData.components.length : 0}</strong>
                  <span>识别组件数</span>
                </div>
              </div>

              <div className="pill-row">
                {'ok' in reviewData ? (
                  <StatusPill tone={reviewData.ok ? 'good' : 'bad'}>validate {reviewData.ok ? 'ok' : 'failed'}</StatusPill>
                ) : null}
                {hasBundlePath(reviewData) ? <StatusPill tone="accent">pack 完成</StatusPill> : null}
                {'workspacePath' in reviewData ? <StatusPill>{reviewData.workspacePath}</StatusPill> : null}
              </div>

              <div className="two-column">
                <SurfaceCard title="导入摘要" description="来自 `importReport` 的结构化状态。">
                  {reviewData.importReport ? (
                    <ImportReportCard report={reviewData.importReport} />
                  ) : (
                    <div className="empty-box">当前还没有返回 importReport。</div>
                  )}
                </SurfaceCard>
                <SurfaceCard title="告警" description="优先展示人类可读 warning，而不是机器码。">
                  <WarningStack warnings={reviewData.warningDetails || []} />
                </SurfaceCard>
              </div>

              {reviewData.errors.length ? (
                <SurfaceCard title="错误" description="packaging API 返回的结构化 errors。">
                  <div className="list-stack">
                    {reviewData.errors.map((error) => (
                      <div className="error-box" key={error}>
                        {error}
                      </div>
                    ))}
                  </div>
                </SurfaceCard>
              ) : null}

              <SurfaceCard title="操作提示" description="当前右侧不是解释文档，而是给发布操作者的决策提示。">
                <div className="list-stack">
                  <article className="list-card">
                    <strong>Warnings 是审阅项</strong>
                    <p>warningDetails 代表可以继续，但需要人确认的导入偏差；errors 才是阻断条件。</p>
                  </article>
                  <article className="list-card">
                    <strong>Validate 是放行闸门</strong>
                    <p>只要 validate 还没有过，就不要把 bundle 当成可发布制品看待。</p>
                  </article>
                </div>
              </SurfaceCard>

              <SurfaceCard title="发布就绪度" description="把当前阶段能不能进入下一步说清楚。">
                <div className="list-stack">
                  <article className="list-card">
                    <div className="list-card__head">
                      <div>
                        <strong>Prepare</strong>
                        <span>{prepareResult?.workspacePath || '等待工作目录'}</span>
                      </div>
                      <StatusPill tone={prepareResult ? 'good' : 'warn'}>{prepareResult ? '就绪' : '等待中'}</StatusPill>
                    </div>
                  </article>
                  <article className="list-card">
                    <div className="list-card__head">
                      <div>
                        <strong>Validate</strong>
                        <span>{validateResult?.ok ? '可以进入 pack' : '必须先通过再 pack'}</span>
                      </div>
                      <StatusPill tone={validateResult?.ok ? 'good' : 'warn'}>{validateResult?.ok ? '已通过' : '等待中'}</StatusPill>
                    </div>
                  </article>
                  <article className="list-card">
                    <div className="list-card__head">
                      <div>
                        <strong>发布交接</strong>
                        <span>{releaseHandoffPath ? '下游链接已就绪' : 'pack 还没完成'}</span>
                      </div>
                      <StatusPill tone={releaseHandoffPath ? 'good' : 'warn'}>{releaseHandoffPath ? '已就绪' : '空闲'}</StatusPill>
                    </div>
                  </article>
                </div>
              </SurfaceCard>

              <SurfaceCard title="执行日志" description="记录这次会话里执行过的控制台动作。">
                <div className="list-stack">
                  {activityLog.length ? (
                    activityLog.map((entry) => (
                      <article className="list-card" key={entry.id}>
                        <div className="list-card__head">
                          <div>
                            <strong>{entry.step}</strong>
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

              <SurfaceCard title="Pack 交接" description="发布前最后确认的 bundle 元数据。">
                <dl className="property-list">
                  <div>
                    <dt>工作目录</dt>
                    <dd>{reviewData.workspacePath}</dd>
                  </div>
                  {hasBundlePath(reviewData) ? (
                    <div>
                      <dt>bundle 路径</dt>
                      <dd>{reviewData.bundlePath}</dd>
                    </div>
                  ) : null}
                  {hasManifestDigest(reviewData) && reviewData.manifestDigest ? (
                    <div>
                      <dt>manifest 摘要</dt>
                      <dd>{reviewData.manifestDigest}</dd>
                    </div>
                  ) : null}
                  {hasComponents(reviewData) ? (
                    <div>
                      <dt>components</dt>
                      <dd>{reviewData.components.join(', ') || '无'}</dd>
                    </div>
                  ) : null}
                  {hasBundlePath(reviewData) ? (
                    <div>
                      <dt>bundle base64</dt>
                      <dd>{reviewData.bundleBase64 ? '已包含在响应中' : '未包含'}</dd>
                    </div>
                  ) : null}
                </dl>
                {packResult && releaseHandoffPath && authoringHandoffPath ? (
                  <div className="stack-column handoff-panel">
                    <div className="pill-row">
                      <StatusPill tone="good">pack 已可进入发布交接</StatusPill>
                    </div>
                    <p className="support-copy">
                      这里不再让作者自己抄 digest / components。前端会把 pack 结果直接带到 release review 或更上层 authoring flow。
                    </p>
                    <div className="authoring-actions">
                      <Link className="button button--primary" to={releaseHandoffPath}>
                        继续进入 release record
                      </Link>
                      <Link className="button button--secondary" to={authoringHandoffPath}>
                        打开作者流程
                      </Link>
                    </div>
                  </div>
                ) : null}
              </SurfaceCard>
            </div>
          ) : null}
        </SurfaceCard>
      </div>
    </div>
  );
}
