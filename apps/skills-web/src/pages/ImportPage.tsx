import { useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import type {
  PackagingImportReviewRequest,
  PackagingImportReviewResponse,
  PackagingPublishResponse,
  PackagingSourceType,
} from '../api/types';
import { ImportReportCard } from '../components/ImportReportCard';
import {
  MutationResultPanel,
  readMutationResultSearchParams,
} from '../components/MutationResultPanel';
import { PermissionGate } from '../components/PermissionGate';
import { SectionHeading } from '../components/SectionHeading';
import { StatusPill } from '../components/StatusPill';
import { SurfaceCard } from '../components/SurfaceCard';
import { WarningStack } from '../components/WarningStack';
import { buildExternalImportArchive } from '../features/packaging/externalImport';
import {
  derivePackagingViewState,
  persistPackagingMutationResult,
} from '../features/packaging/packagingWorkflowState';
import { useImportWorkflow } from '../features/packaging/useImportWorkflow';
import {
  resolvePrepareAuthoringFiles,
  type AuthoringTemplateId,
} from '../features/packaging/useAuthoringDraft';
import { useAuthoringEditor } from '../features/packaging/useAuthoringEditor';
import { buildReleaseHandoffPath } from '../lib/releaseHandoff';
import { useRegistrySession } from '../session/SessionProvider';

const steps = [
  {
    title: '1. 来源',
    description: 'upload_archive / platform_authoring',
  },
  {
    title: '2. 审阅',
    description: 'warningDetails + importReport + manifestDigest + components',
  },
  {
    title: '3. 发布',
    description: 'review summary -> stateless publish',
  },
];

type ReviewData = PackagingImportReviewResponse | PackagingPublishResponse;

const AUTHORING_TEMPLATES: Record<
  AuthoringTemplateId,
  {
    label: string;
    description: string;
  }
> = {
  prompt: {
    label: 'prompt',
    description: '单 prompt skill',
  },
  command: {
    label: 'command',
    description: '单命令 skill',
  },
  workflow: {
    label: 'workflow',
    description: '步骤型 workflow skill',
  },
};

export function ImportPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { session, getPosture, getPostureLabel } = useRegistrySession();
  const [sourceType, setSourceType] = useState<PackagingSourceType>('upload_archive');
  const [location, setLocation] = useState('https://github.com/mattpocock/skills?path=skills/engineering/zoom-out');
  const [sourceRef, setSourceRef] = useState('');
  const [packageName, setPackageName] = useState('@company.skill/zoom-out');
  const [version, setVersion] = useState('1.2.0');
  const [displayName, setDisplayName] = useState('Zoom Out');
  const [description, setDescription] = useState('Broader context');
  const [metadataSourceType, setMetadataSourceType] = useState('imported');
  const [publishRegistry, setPublishRegistry] = useState('https://npm.internal.example.com');
  const [strictMode, setStrictMode] = useState(false);
  const [includeBundle, setIncludeBundle] = useState(false);

  const normalizedPackageName = packageName.trim();
  const packagingPosture = getPosture({
    packageName: normalizedPackageName,
    requiredAction: 'packaging.publish',
  });
  const packagingPostureLabel =
    packagingPosture === 'write_enabled'
      ? '允许发布'
      : packagingPosture === 'read_only'
        ? '发布只读'
        : getPostureLabel({
            packageName: normalizedPackageName,
            requiredAction: 'packaging.publish',
          });
  const mutationResult = readMutationResultSearchParams(searchParams, {
    packageName: null,
    version: null,
    actor: session.data?.actor ?? null,
  });
  const {
    authoringTemplate,
    authoringFiles,
    selectedAuthoringPath,
    setSelectedAuthoringPath,
    selectedAuthoringFile,
    newAuthoringPath,
    setNewAuthoringPath,
    authoringError,
    showAdvancedFiles,
    setShowAdvancedFiles,
    rawInlineFiles,
    setRawInlineFiles,
    rawInlineFilesDirty,
    setRawInlineFilesDirty,
    advancedInlineFilesValue,
    resetAuthoringTemplate,
    handleAddAuthoringFile,
    handleDeleteAuthoringFile,
    handleAuthoringContentChange,
    applyRawInlineFiles,
  } = useAuthoringEditor('command');

  function persistMutationResult(result: {
    action: string;
    outcome: 'succeeded' | 'failed' | 'forbidden' | 'unauthorized';
    message: string;
    requestId?: string | null;
    packageName?: string | null;
    version?: string | null;
  }) {
    setSearchParams(
      persistPackagingMutationResult(searchParams, {
        ...result,
        actor: session.data?.actor ?? null,
      }),
      { replace: true }
    );
  }

  async function buildReviewRequest(): Promise<PackagingImportReviewRequest> {
    const platformFiles = resolvePrepareAuthoringFiles({
      authoringFiles,
      rawInlineFiles,
      rawInlineFilesDirty,
    });

    if (sourceType !== 'platform_authoring') {
      const archive = await buildExternalImportArchive(
        location.trim(),
        sourceRef.trim() !== '' ? sourceRef.trim() : undefined
      );
      return {
        source: {
          type: 'upload_archive',
          archiveBase64: archive.archiveBase64,
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
        includeBundleBase64: includeBundle,
      };
    }

    return {
      source: {
        type: sourceType,
        files: platformFiles,
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
      includeBundleBase64: includeBundle,
    };
  }
  const {
    operationState,
    reviewResult,
    publishResult,
    activityLog,
    handlePreview,
    handlePublish,
  } = useImportWorkflow({
    sourceType,
    authoringError,
    normalizedPackageName,
    version,
    buildReviewRequest,
    onMutationResult: persistMutationResult,
    resetKeys: [
      sourceType,
      location,
      sourceRef,
      authoringFiles,
      rawInlineFiles,
      rawInlineFilesDirty,
      packageName,
      version,
      displayName,
      description,
      metadataSourceType,
      publishRegistry,
      strictMode,
      includeBundle,
    ],
  });
  const workflow = derivePackagingViewState({
    reviewResult,
    publishResult,
    publishRegistry,
  });
  const reviewData: ReviewData | null = publishResult || reviewResult;
  const stageTone = workflow.currentStage === 'published' ? 'good' : workflow.currentStage === 'reviewed' ? 'accent' : undefined;
  const releaseHandoffPath = workflow.releaseHandoffPath;
  const authoringHandoffPath = reviewResult
    ? buildReleaseHandoffPath(
        {
          packageName: reviewResult.packageName,
          version: reviewResult.version,
          publishRegistry: publishRegistry.trim() || undefined,
          manifestDigest: reviewResult.manifestDigest,
          components: reviewResult.components,
        },
        '/workbench/authoring'
      )
    : null;

  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="导入与打包"
        title="先检查并生成预览，再一键发布。"
        description="默认模式只保留“检查并生成发布预览”和“执行一次性发布”两个动作。"
        tone="backstage"
      />

      <div className="context-strip">
        <StatusPill tone="accent">来源 {sourceType}</StatusPill>
        <StatusPill>{normalizedPackageName}</StatusPill>
        <StatusPill>{version}</StatusPill>
        <StatusPill tone={strictMode ? 'warn' : 'good'}>{strictMode ? '严格' : '宽松'}</StatusPill>
        <StatusPill tone={stageTone}>阶段 {workflow.currentStage}</StatusPill>
        <StatusPill>{packagingPostureLabel}</StatusPill>
      </div>

      <div className="two-column two-column--wide-right">
        <SurfaceCard title="导入并发布 skill" description="先填来源和发布目标，再用默认模式完成发布预览与一次性发布。" variant="console">
          <div className="stack-column form-section">
            <label className="field">
              <span>来源类型</span>
              <select value={sourceType} onChange={(event) => setSourceType(event.target.value as PackagingSourceType)}>
                <option value="upload_archive">upload_archive</option>
                <option value="platform_authoring">platform_authoring</option>
              </select>
            </label>

            {sourceType === 'upload_archive' ? (
              <>
                <label className="field">
                  <span>位置</span>
                  <input
                    onChange={(event) => setLocation(event.target.value)}
                    placeholder="GitHub skill 地址"
                    value={location}
                  />
                </label>
                <label className="field">
                  <span>来源引用</span>
                  <input
                    onChange={(event) => setSourceRef(event.target.value)}
                    placeholder="留空使用默认分支；也可填 release / commit sha"
                    value={sourceRef}
                  />
                </label>
              </>
            ) : (
              <SurfaceCard title="平台作者输入" description="在浏览器里直接编辑要发版的文件列表。">
                <div className="stack-column">
                  <label className="field">
                    <span>模板</span>
                    <select
                      aria-label="模板"
                      value={authoringTemplate}
                      onChange={(event) => resetAuthoringTemplate(event.target.value as AuthoringTemplateId)}
                    >
                      {Object.entries(AUTHORING_TEMPLATES).map(([value, template]) => (
                        <option key={value} value={value}>
                          {template.label} · {template.description}
                        </option>
                      ))}
                    </select>
                  </label>

                  <div className="authoring-actions">
                    <input
                      aria-label="新文件路径"
                      placeholder="例如 docs/README.md"
                      value={newAuthoringPath}
                      onChange={(event) => setNewAuthoringPath(event.target.value)}
                    />
                    <button className="button button--secondary" onClick={handleAddAuthoringFile} type="button">
                      添加文件
                    </button>
                  </div>

                  <div className="list-stack">
                    {authoringFiles.map((file) => (
                      <article className="authoring-file-row" key={file.path}>
                        <button
                          className={
                            file.path === selectedAuthoringPath
                              ? 'button button--secondary authoring-file-row__select authoring-file-row__select--active'
                              : 'button button--secondary authoring-file-row__select'
                          }
                          onClick={() => setSelectedAuthoringPath(file.path)}
                          type="button"
                        >
                          {file.path}
                        </button>
                        {!file.locked ? (
                          <button
                            aria-label={`删除 ${file.path}`}
                            className="button button--ghost"
                            onClick={() => handleDeleteAuthoringFile(file.path)}
                            type="button"
                          >
                            删除 {file.path}
                          </button>
                        ) : null}
                      </article>
                    ))}
                  </div>

                  {selectedAuthoringFile ? (
                    <>
                      <label className="field">
                        <span>当前文件路径</span>
                        <input aria-label="当前文件路径" readOnly value={selectedAuthoringFile.path} />
                      </label>
                      <label className="field">
                        <span>当前文件内容</span>
                        <textarea
                          aria-label="当前文件内容"
                          className="code-field"
                          value={selectedAuthoringFile.content}
                          onChange={(event) => handleAuthoringContentChange(event.target.value)}
                        />
                      </label>
                    </>
                  ) : null}

                  <button
                    className="button button--ghost"
                    onClick={() => setShowAdvancedFiles((current) => !current)}
                    type="button"
                  >
                    高级模式 / 原始 files JSON
                  </button>

                  {showAdvancedFiles ? (
                    <div className="stack-column">
                      <label className="field">
                        <span>原始 files JSON</span>
                        <textarea
                          aria-label="原始 files JSON"
                          className="code-field"
                          value={advancedInlineFilesValue}
                          onChange={(event) => {
                            setRawInlineFiles(event.target.value);
                            setRawInlineFilesDirty(true);
                          }}
                        />
                      </label>
                      <button className="button button--secondary" onClick={applyRawInlineFiles} type="button">
                        应用 JSON 到文件列表
                      </button>
                    </div>
                  ) : null}
                </div>
              </SurfaceCard>
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
              <span>在 review 响应里附带 bundle base64</span>
            </label>

            <MutationResultPanel result={mutationResult} />

            <div className="action-row">
              <div className="action-row__copy">
                <strong>默认模式</strong>
                <p>先生成发布预览，再执行 Catalog 后端的一次性发布。发布成功后会一并写入 release record。</p>
              </div>
            </div>
            <div className="button-row">
              <PermissionGate
                fallback={<div className="empty-box">当前会话对 `packaging.publish` 只有只读权限。</div>}
                packageName={normalizedPackageName}
                requiredAction="packaging.publish"
              >
                <button
                  className="button button--primary"
                  disabled={operationState.loading !== false}
                  onClick={() => void handlePreview()}
                  type="button"
                >
                  {operationState.loading === 'review' ? '生成预览中…' : '检查并生成发布预览'}
                </button>
              </PermissionGate>
              <PermissionGate
                fallback={<div className="empty-box">当前会话对 `packaging.publish` 只有只读权限。</div>}
                packageName={normalizedPackageName}
                requiredAction="packaging.publish"
              >
                <button
                  className="button button--secondary"
                  disabled={!workflow.canPublish || operationState.loading !== false}
                  onClick={() => void handlePublish()}
                  type="button"
                >
                  {operationState.loading === 'publish' ? '发布中…' : '发布 skill'}
                </button>
              </PermissionGate>
            </div>
          </div>
        </SurfaceCard>

        <SurfaceCard className="surface-card--sticky" title="校验审阅" description="右侧始终展示当前 preview / publish 结果，方便确认能不能继续发版。" variant="rail">
          {!reviewData ? <div className="empty-box">先执行“检查并生成发布预览”，再决定是否继续执行一次性发布。</div> : null}
          {reviewData ? (
            <div className="stack-column">
              <div className="step-list">
                {steps.map((step, index) => (
                  <article
                    className={index === workflow.activeStepIndex ? 'step-card step-card--active' : 'step-card'}
                    key={step.title}
                  >
                    <strong>{step.title}</strong>
                    <p>{step.description}</p>
                  </article>
                ))}
              </div>

              <div className="console-strip console-strip--compact">
                <article className="console-signal console-signal--compact">
                  <span>最近动作</span>
                  <strong>{operationState.loading || workflow.currentStage}</strong>
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
                  <strong>{reviewData.components.length}</strong>
                  <span>识别组件数</span>
                </div>
              </div>

              <div className="pill-row">
                {'ok' in reviewData ? (
                  <StatusPill tone={reviewData.ok ? 'good' : 'bad'}>
                    review {reviewData.ok ? 'ok' : 'failed'}
                  </StatusPill>
                ) : null}
                {'publishedToRegistry' in reviewData ? <StatusPill tone="accent">发布完成</StatusPill> : null}
                {reviewData.manifestDigest ? <StatusPill>{reviewData.manifestDigest}</StatusPill> : null}
              </div>

              {operationState.error ? <div className="error-box">{operationState.error}</div> : null}
              {operationState.success ? <div className="success-box">{operationState.success}</div> : null}

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

              <SurfaceCard title="发布就绪度" description="把当前阶段能不能进入下一步说清楚。">
                <div className="list-stack">
                  <article className="list-card">
                    <div className="list-card__head">
                      <div>
                        <strong>导入审阅</strong>
                        <span>{workflow.reviewReadiness.detail}</span>
                      </div>
                      <StatusPill tone={workflow.reviewReadiness.tone}>{workflow.reviewReadiness.label}</StatusPill>
                    </div>
                  </article>
                  <article className="list-card">
                    <div className="list-card__head">
                      <div>
                        <strong>发布执行</strong>
                        <span>{workflow.publishReadiness.detail}</span>
                      </div>
                      <StatusPill tone={workflow.publishReadiness.tone}>{workflow.publishReadiness.label}</StatusPill>
                    </div>
                  </article>
                  <article className="list-card">
                    <div className="list-card__head">
                      <div>
                        <strong>发布交接</strong>
                        <span>{workflow.releaseReadiness.detail}</span>
                      </div>
                      <StatusPill tone={workflow.releaseReadiness.tone}>{workflow.releaseReadiness.label}</StatusPill>
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

              <SurfaceCard
                title={publishResult ? '发布结果' : '发布交接'}
                description={
                  publishResult
                    ? '后端已经完成 registry publish 和 release record。'
                    : '发布前最后确认的导入审阅元数据。'
                }
              >
                <dl className="property-list">
                  <div>
                    <dt>manifest 摘要</dt>
                    <dd>{reviewData.manifestDigest || '—'}</dd>
                  </div>
                  <div>
                    <dt>components</dt>
                    <dd>{reviewData.components.join(', ') || '无'}</dd>
                  </div>
                  {'publishedToRegistry' in reviewData ? (
                    <>
                      <div>
                        <dt>registry publish</dt>
                        <dd>{reviewData.publishedToRegistry ? '本次已执行' : '复用已有版本'}</dd>
                      </div>
                      <div>
                        <dt>release record</dt>
                        <dd>{reviewData.created ? '本次已创建' : '已存在，按幂等返回'}</dd>
                      </div>
                    </>
                  ) : null}
                  {'bundleBase64' in reviewData ? (
                    <div>
                      <dt>bundle base64</dt>
                      <dd>{reviewData.bundleBase64 ? '已包含在响应中' : '未包含'}</dd>
                    </div>
                  ) : null}
                </dl>
                {!publishResult && releaseHandoffPath && authoringHandoffPath ? (
                  <div className="stack-column handoff-panel">
                    <div className="pill-row">
                      <StatusPill tone="good">review 已可进入发布交接</StatusPill>
                    </div>
                    <p className="support-copy">
                      这里不会暴露 workspace 或 bundle 路径；版本页和作者台只承接 manifest digest、components、publishRegistry。
                    </p>
                    <div className="authoring-actions">
                      <Link className="button button--primary" to={releaseHandoffPath}>
                        打开版本详情
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
