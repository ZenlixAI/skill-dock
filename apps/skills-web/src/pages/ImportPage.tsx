import { useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import {
  getPackagingBundleDownloadPath,
  cleanupPackagingWorkspace,
  packSkill,
  preparePackaging,
  publishSkill,
  validatePackaging,
} from '../api/client';
import { normalizeThrownRegistryError } from '../api/httpClient';
import type {
  PackagingPackResponse,
  PackagingPublishResponse,
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
} from '../components/MutationResultPanel';
import { PermissionGate } from '../components/PermissionGate';
import { SectionHeading } from '../components/SectionHeading';
import { StatusPill } from '../components/StatusPill';
import { SurfaceCard } from '../components/SurfaceCard';
import { WarningStack } from '../components/WarningStack';
import { buildReleaseHandoffPath } from '../lib/releaseHandoff';
import { buildExternalImportArchive } from '../features/packaging/externalImport';
import {
  createAuthoringEditorState,
  getDefaultAuthoringPath,
  hasUnsavedAuthoringChanges,
  parseOptionalAuthoringJson,
  resolvePrepareAuthoringFiles,
  serializeAuthoringFiles,
  type AuthoringFile,
  type AuthoringTemplateId,
  validateAuthoringPath,
} from '../features/packaging/useAuthoringDraft';
import {
  derivePackagingViewState,
  persistPackagingMutationResult,
  toMutationResult,
} from '../features/packaging/usePackagingWorkflow';
import { useRegistrySession } from '../session/SessionProvider';

const steps = [
  {
    title: '1. 来源',
    description: 'upload_archive / platform_authoring',
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
    title: '4. 打包与交接',
    description: '高级模式 pack handoff / 默认模式一键 publish',
  },
];

type ReviewData =
  | PackagingPrepareResponse
  | PackagingValidateResponse
  | PackagingPackResponse
  | PackagingPublishResponse;
type ImportLogEntry = {
  id: string;
  step: 'prepare' | 'validate' | 'pack' | 'publish' | 'cleanup';
  outcome: 'succeeded' | 'failed';
  summary: string;
};

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
  const { session, getPosture, getPostureLabel } = useRegistrySession();
  const [sourceType, setSourceType] = useState<PackagingSourceType>('upload_archive');
  const [location, setLocation] = useState('https://github.com/mattpocock/skills?path=skills/engineering/zoom-out');
  const [sourceRef, setSourceRef] = useState('');
  const [authoringTemplate, setAuthoringTemplate] = useState<AuthoringTemplateId>('command');
  const [authoringFiles, setAuthoringFiles] = useState<AuthoringFile[]>(
    () => createAuthoringEditorState('command').draft.files
  );
  const [selectedAuthoringPath, setSelectedAuthoringPath] = useState(
    () => createAuthoringEditorState('command').selectedPath
  );
  const [newAuthoringPath, setNewAuthoringPath] = useState('');
  const [authoringError, setAuthoringError] = useState<string | null>(null);
  const [showAdvancedFiles, setShowAdvancedFiles] = useState(false);
  const [showAdvancedWorkflow, setShowAdvancedWorkflow] = useState(false);
  const [rawInlineFiles, setRawInlineFiles] = useState('');
  const [rawInlineFilesDirty, setRawInlineFilesDirty] = useState(false);
  const [packageName, setPackageName] = useState('@company.skill/zoom-out');
  const [version, setVersion] = useState('1.2.0');
  const [displayName, setDisplayName] = useState('Zoom Out');
  const [description, setDescription] = useState('Broader context');
  const [metadataSourceType, setMetadataSourceType] = useState('imported');
  const [publishRegistry, setPublishRegistry] = useState('https://npm.internal.example.com');
  const [strictMode, setStrictMode] = useState(false);
  const [includeBundle, setIncludeBundle] = useState(false);
  const [operationState, setOperationState] = useState<{
    loading: false | 'prepare' | 'validate' | 'pack' | 'publish' | 'cleanup';
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
  const [publishResult, setPublishResult] = useState<PackagingPublishResponse | null>(null);
  const [activityLog, setActivityLog] = useState<ImportLogEntry[]>([]);

  const workflow = derivePackagingViewState({
    prepareResult,
    validateResult,
    packResult,
    publishResult,
    publishRegistry,
  });
  const reviewData: ReviewData | null = publishResult || packResult || validateResult || prepareResult;
  const stageTone =
    workflow.currentStage === 'published' || workflow.currentStage === 'pack_complete'
      ? 'good'
      : workflow.currentStage === 'validated'
        ? 'accent'
        : workflow.currentStage === 'prepared'
          ? 'warn'
          : undefined;
  const releaseHandoffPath = workflow.releaseHandoffPath;
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
  const normalizedPackageName = packageName.trim();
  const cleanupPackageName =
    packResult?.packageName || validateResult?.packageName || prepareResult?.packageName || normalizedPackageName;
  const packagingPosture = getPosture({
    packageName: normalizedPackageName,
    requiredAction: 'packaging.pack',
  });
  const packagingWriteEnabled = packagingPosture === 'write_enabled';
  const packagingPostureLabel =
    packagingPosture === 'write_enabled'
      ? '允许打包'
      : packagingPosture === 'read_only'
        ? '打包只读'
        : getPostureLabel({
            packageName: normalizedPackageName,
            requiredAction: 'packaging.pack',
          });
  const mutationResult = readMutationResultSearchParams(searchParams, {
    packageName: null,
    version: null,
    actor: session.data?.actor ?? null,
  });
  const selectedAuthoringFile =
    authoringFiles.find((file) => file.path === selectedAuthoringPath) || authoringFiles[0] || null;
  const advancedInlineFilesValue = rawInlineFilesDirty ? rawInlineFiles : serializeAuthoringFiles(authoringFiles);

  useEffect(() => {
    setPrepareResult(null);
    setValidateResult(null);
    setPackResult(null);
    setPublishResult(null);
    setOperationState((current) =>
      current.loading === false && current.success === null
        ? current
        : {
            loading: false,
            error: null,
            errorDetail: null,
            success: null,
          }
    );
  }, [
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
  ]);

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
      persistPackagingMutationResult(searchParams, {
        ...result,
        actor: session.data?.actor ?? null,
      }),
      { replace: true }
    );
  }

  function resetAuthoringTemplate(nextTemplate: AuthoringTemplateId) {
    const hasUnsavedChanges = hasUnsavedAuthoringChanges({
      authoringFiles,
      rawInlineFilesDirty,
      template: authoringTemplate,
    });

    if (
      hasUnsavedChanges &&
      !window.confirm('切换模板会重建当前文件列表，未保存的作者输入会丢失。继续吗？')
    ) {
      return;
    }

    const nextAuthoringState = createAuthoringEditorState(nextTemplate);
    setAuthoringTemplate(nextTemplate);
    setAuthoringFiles(nextAuthoringState.draft.files);
    setSelectedAuthoringPath(nextAuthoringState.selectedPath);
    setAuthoringError(null);
    setRawInlineFiles('');
    setRawInlineFilesDirty(false);
    setNewAuthoringPath('');
  }

  function handleAddAuthoringFile() {
    const validationError = validateAuthoringPath(newAuthoringPath, authoringFiles);
    if (validationError) {
      setAuthoringError(validationError);
      return;
    }

    const trimmed = newAuthoringPath.trim();
    const nextFiles = [...authoringFiles, { path: trimmed, content: '', locked: false }];
    setAuthoringFiles(nextFiles);
    setSelectedAuthoringPath(trimmed);
    setAuthoringError(null);
    setNewAuthoringPath('');
    setRawInlineFilesDirty(false);
  }

  function handleDeleteAuthoringFile(path: string) {
    const target = authoringFiles.find((file) => file.path === path);
    if (!target || target.locked) {
      return;
    }

    const nextFiles = authoringFiles.filter((file) => file.path !== path);
    setAuthoringFiles(nextFiles);
    setAuthoringError(null);
    setRawInlineFilesDirty(false);

    if (selectedAuthoringPath === path) {
      setSelectedAuthoringPath(getDefaultAuthoringPath(nextFiles));
    }
  }

  function handleAuthoringContentChange(content: string) {
    const activePath = selectedAuthoringFile?.path;
    if (!activePath) {
      return;
    }

    setAuthoringFiles((current) =>
      current.map((file) => (file.path === activePath ? { ...file, content } : file))
    );
    setAuthoringError(null);
    setRawInlineFilesDirty(false);
  }

  function applyRawInlineFiles() {
    const nextAuthoringState = createAuthoringEditorState(
      authoringTemplate,
      parseOptionalAuthoringJson(rawInlineFiles) || {}
    );
    setAuthoringFiles(nextAuthoringState.draft.files);
    setSelectedAuthoringPath(nextAuthoringState.selectedPath);
    setAuthoringError(null);
    setRawInlineFilesDirty(false);
  }

  async function buildPrepareRequest(): Promise<PackagingPrepareRequest> {
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
    };
  }

  async function runPrepare(successMessage = 'prepare 完成，已经拿到 workspace handle 和 import review 数据。') {
    if (sourceType === 'platform_authoring' && authoringError) {
      setOperationState({
        loading: false,
        error: authoringError,
        errorDetail: createLocalRequestError(authoringError),
        success: null,
      });
      return null;
    }

    setOperationState({
      loading: 'prepare',
      error: null,
      errorDetail: null,
      success: null,
    });
    try {
      const response = await preparePackaging(await buildPrepareRequest());
      setPrepareResult(response);
      setValidateResult(null);
      setPackResult(null);
      setPublishResult(null);
      setOperationState({
        loading: false,
        error: null,
        errorDetail: null,
        success: successMessage,
      });
      persistMutationResult({
        action: 'prepare',
        outcome: 'succeeded',
        message: successMessage,
        packageName: response.packageName,
        version: response.version,
      });
      appendLog('prepare', 'succeeded', `已准备 ${response.packageName}@${response.version}。`);
      return response;
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
      persistMutationResult(
        toMutationResult('prepare', normalizedError, outcome, {
          packageName: normalizedPackageName || null,
          version: version.trim() || null,
          actor: session.data?.actor ?? null,
          message,
        })
      );
      return null;
    }
  }

  async function runValidate(
    workspacePath: string,
    messages: {
      success: string;
      failed: string;
    } = {
      success: 'validate 通过，可以继续 pack。',
      failed: 'validate 返回未通过，请先处理 errors。',
    }
  ) {
    setOperationState({
      loading: 'validate',
      error: null,
      errorDetail: null,
      success: null,
    });
    try {
      const response = await validatePackaging({
        workspacePath,
        target: {
          packageName: packageName.trim(),
          version: version.trim(),
        },
      });
      setValidateResult(response);
      setPackResult(null);
      setPublishResult(null);
      const message = response.ok ? messages.success : messages.failed;
      setOperationState({
        loading: false,
        error: null,
        errorDetail: null,
        success: message,
      });
      persistMutationResult({
        action: 'validate',
        outcome: response.ok ? 'succeeded' : 'failed',
        message,
        packageName: response.packageName,
        version: response.version,
      });
      appendLog('validate', response.ok ? 'succeeded' : 'failed', response.ok ? '校验通过。' : '校验返回阻断错误。');
      return response;
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
      persistMutationResult(
        toMutationResult('validate', normalizedError, outcome, {
          packageName: prepareResult?.packageName || normalizedPackageName || null,
          version: prepareResult?.version || version.trim() || null,
          actor: session.data?.actor ?? null,
          message,
        })
      );
      return null;
    }
  }

  async function runPack(workspacePath: string, successMessage = 'pack 完成，已经拿到 bundlePath / manifestDigest / components。') {
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
      setPublishResult(null);
      setOperationState({
        loading: false,
        error: null,
        errorDetail: null,
        success: successMessage,
      });
      persistMutationResult({
        action: 'pack',
        outcome: 'succeeded',
        message: successMessage,
        packageName: response.packageName,
        version: response.version,
      });
      appendLog('pack', 'succeeded', `已打包 bundle ${response.bundlePath}。`);
      return response;
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
      persistMutationResult(
        toMutationResult('pack', normalizedError, outcome, {
          packageName: validateResult?.packageName || prepareResult?.packageName || normalizedPackageName || null,
          version: validateResult?.version || prepareResult?.version || version.trim() || null,
          actor: session.data?.actor ?? null,
          message,
        })
      );
      return null;
    }
  }

  async function handlePrepare() {
    await runPrepare();
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

    await runValidate(prepareResult.workspacePath);
  }

  async function handlePreview() {
    const prepared = await runPrepare('来源检查完成，继续生成预览…');
    if (!prepared?.workspacePath) {
      return;
    }

    await runValidate(prepared.workspacePath, {
      success: '预览已生成，可以继续发布 skill。',
      failed: '预览生成完成，但存在阻断错误，请先处理 errors。',
    });
  }

  async function handlePublish() {
    if (!validateResult?.ok) {
      const message = '请先执行“检查并生成发布预览”。';
      setOperationState({
        loading: false,
        error: message,
        errorDetail: createLocalRequestError(message),
        success: null,
      });
      persistMutationResult({
        action: 'pack',
        outcome: 'failed',
        message,
        packageName: normalizedPackageName || null,
        version: version.trim() || null,
      });
      return;
    }

    setOperationState({
      loading: 'publish',
      error: null,
      errorDetail: null,
      success: null,
    });
    try {
      const prepareRequest = await buildPrepareRequest();
      const response = await publishSkill({
        ...prepareRequest,
        review: {
          manifestDigest: validateResult.manifestDigest,
          components: validateResult.components,
        },
      });
      setPublishResult(response);
      setPackResult(null);
      setOperationState({
        loading: false,
        error: null,
        errorDetail: null,
        success: 'skill 已发布到 registry，并已写入 release record。',
      });
      persistMutationResult({
        action: 'publish',
        outcome: 'succeeded',
        message: 'skill 已发布到 registry，并已写入 release record。',
        packageName: response.packageName,
        version: response.version,
      });
      appendLog(
        'publish',
        'succeeded',
        response.publishedToRegistry
          ? `已发布 ${response.packageName}@${response.version} 并写入 release record。`
          : `已复用现有 registry 版本并补齐 ${response.packageName}@${response.version} 的 release record。`
      );
    } catch (error: unknown) {
      const outcome = inferMutationFailureOutcome(error);
      const normalizedError = normalizeThrownRegistryError(error);
      const message = describeImportOperationError(normalizedError);
      appendLog('publish', 'failed', normalizedError.message);
      setOperationState({
        loading: false,
        error: normalizedError.message,
        errorDetail: normalizedError,
        success: null,
      });
      persistMutationResult(
        toMutationResult('publish', normalizedError, outcome, {
          packageName: validateResult?.packageName || prepareResult?.packageName || normalizedPackageName || null,
          version: validateResult?.version || prepareResult?.version || version.trim() || null,
          actor: session.data?.actor ?? null,
          message,
        })
      );
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

    await runPack(workspacePath);
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
      setPublishResult(null);
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
      persistMutationResult(
        toMutationResult('cleanup', normalizedError, outcome, {
          packageName: null,
          version: null,
          actor: session.data?.actor ?? null,
          message,
        })
      );
    }
  }

  return (
    <div className="page-grid">
      <SectionHeading
        eyebrow="导入与打包"
        title="先检查并生成预览，再一键发布。"
        description="默认模式只暴露“检查并生成发布预览”和“发布 skill”两个动作。底层 prepare / validate / pack 仍然保留在高级模式里给 operator 排障。"
        tone="backstage"
      />

      <div className="context-strip">
        <StatusPill tone="accent">来源 {sourceType}</StatusPill>
        <StatusPill>{packageName}</StatusPill>
        <StatusPill>{version}</StatusPill>
        <StatusPill tone={strictMode ? 'warn' : 'good'}>{strictMode ? '严格' : '宽松'}</StatusPill>
        <StatusPill tone={stageTone}>阶段 {workflow.currentStage}</StatusPill>
        <StatusPill tone={packagingPosture === 'write_enabled' ? 'accent' : packagingPosture === 'package_filter_required' ? 'warn' : 'default'}>
          {packagingPostureLabel}
        </StatusPill>
      </div>

      <div className="two-column two-column--wide-right">
        <SurfaceCard title="导入 skill" description="先填来源和目标，再用默认模式完成预览与发布。" variant="console">
          <div className="stack-column form-section">
            <label className="field">
              <span>来源类型</span>
              <select value={sourceType} onChange={(event) => setSourceType(event.target.value as PackagingSourceType)}>
                <option value="upload_archive">upload_archive</option>
                <option value="platform_authoring">platform_authoring</option>
              </select>
            </label>

            {sourceType === 'platform_authoring' ? (
              <div className="authoring-editor">
                <div className="authoring-editor__header">
                  <div>
                    <strong>作者输入</strong>
                    <p>不要手写 `files` JSON。先选模板，再用文件面板补内容。</p>
                  </div>
                  <label className="field">
                    <span>模板</span>
                    <select
                      aria-label="模板"
                      onChange={(event) => resetAuthoringTemplate(event.target.value as AuthoringTemplateId)}
                      value={authoringTemplate}
                    >
                      {Object.entries(AUTHORING_TEMPLATES).map(([key, template]) => (
                        <option key={key} value={key}>
                          {template.label} · {template.description}
                        </option>
                      ))}
                    </select>
                  </label>
                </div>

                <div className="authoring-editor__layout">
                  <div className="authoring-editor__files">
                    <label className="field">
                      <span>新文件路径</span>
                      <input
                        aria-label="新文件路径"
                        onChange={(event) => setNewAuthoringPath(event.target.value)}
                        placeholder="例如 docs/README.md"
                        value={newAuthoringPath}
                      />
                    </label>
                    <button className="button button--secondary" onClick={handleAddAuthoringFile} type="button">
                      添加文件
                    </button>

                    <div className="list-stack">
                      {authoringFiles.map((file) => (
                        <div className="authoring-file-row" key={file.path}>
                          <button
                            className={
                              file.path === selectedAuthoringFile?.path
                                ? 'button button--secondary authoring-file-row__select authoring-file-row__select--active'
                                : 'button button--ghost authoring-file-row__select'
                            }
                            onClick={() => {
                              setSelectedAuthoringPath(file.path);
                              setAuthoringError(null);
                            }}
                            type="button"
                          >
                            {file.path}
                          </button>
                          {!file.locked ? (
                            <button
                              className="button button--ghost"
                              onClick={() => handleDeleteAuthoringFile(file.path)}
                              type="button"
                            >
                              删除 {file.path}
                            </button>
                          ) : null}
                        </div>
                      ))}
                    </div>
                  </div>

                  <div className="authoring-editor__content">
                    <label className="field">
                      <span>当前文件路径</span>
                      <input aria-label="当前文件路径" readOnly value={selectedAuthoringFile?.path || ''} />
                    </label>
                    <label className="field">
                      <span>当前文件内容</span>
                      <textarea
                        aria-label="当前文件内容"
                        className="code-field"
                        onChange={(event) => handleAuthoringContentChange(event.target.value)}
                        value={selectedAuthoringFile?.content || ''}
                      />
                    </label>
                  </div>
                </div>

                <div className="authoring-editor__advanced">
                  <button
                    className="button button--ghost"
                    onClick={() => setShowAdvancedFiles((current) => !current)}
                    type="button"
                  >
                    {showAdvancedFiles ? '收起高级模式 / 原始 files JSON' : '高级模式 / 原始 files JSON'}
                  </button>
                  {showAdvancedFiles ? (
                    <div className="stack-column">
                      <label className="field">
                        <span>原始 files JSON</span>
                        <textarea
                          className="code-field"
                          onChange={(event) => {
                            setRawInlineFiles(event.target.value);
                            setRawInlineFilesDirty(true);
                            setAuthoringError(null);
                          }}
                          value={advancedInlineFilesValue}
                        />
                      </label>
                      <div className="authoring-actions">
                        <button
                          className="button button--secondary"
                          onClick={() => {
                            try {
                              applyRawInlineFiles();
                            } catch (error: unknown) {
                              setAuthoringError(error instanceof Error ? error.message : '原始 files JSON 无法解析。');
                            }
                          }}
                          type="button"
                        >
                          应用 JSON 到文件列表
                        </button>
                        <span className="support-copy">这里只给调试和粘贴现有 payload 用，默认还是走可视文件编辑。</span>
                      </div>
                    </div>
                  ) : null}
                </div>

                {authoringError ? <div className="error-box">{authoringError}</div> : null}
              </div>
            ) : (
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
                <strong>默认模式</strong>
                <p>先检查并生成发布预览，再执行后端一次性 publish。默认“发布 skill”会由 Catalog 完成 registry publish 和 release record。</p>
              </div>
            </div>
            <div className="button-row">
              <PermissionGate
                fallback={<div className="empty-box">当前会话对 `packaging.prepare` 只有只读权限。</div>}
                packageName={normalizedPackageName}
                requiredAction="packaging.prepare"
              >
                <button
                  className="button button--primary"
                  disabled={operationState.loading !== false}
                  onClick={() => void handlePreview()}
                  type="button"
                >
                  {operationState.loading === 'prepare' || operationState.loading === 'validate'
                    ? '生成预览中…'
                    : '检查并生成发布预览'}
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
              <button
                className="button button--ghost"
                onClick={() => setShowAdvancedWorkflow((current) => !current)}
                type="button"
              >
                {showAdvancedWorkflow ? '收起高级模式' : '高级模式：查看 prepare / validate / pack'}
              </button>
            </div>

            {showAdvancedWorkflow ? (
              <SurfaceCard
                title="高级模式"
                description="普通用户不需要看这层协议细节；这里只给 operator 做排障和手动控制。"
              >
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

                <div className="console-strip">
                  <article className="console-signal">
                    <span>当前阶段</span>
                    <strong>{workflow.currentStage}</strong>
                    <p>{workflow.currentStageSummary}</p>
                  </article>
                  <article className="console-signal">
                    <span>workspace handle</span>
                    <strong>{workflow.workspaceStatus}</strong>
                    <p>{prepareResult?.workspacePath || 'prepare 之后这里会出现 workspace handle。'}</p>
                  </article>
                  <article className="console-signal">
                    <span>发布交接</span>
                    <strong>{workflow.releaseStatus}</strong>
                    <p>{workflow.releaseSummary}</p>
                  </article>
                </div>

                <div className="action-row">
                  <div className="action-row__copy">
                    <strong>原始协议动作</strong>
                    <p>按 prepare → validate → pack 手动执行，适合排查响应内容或复现特定阶段问题。</p>
                  </div>
                </div>

                <div className="button-row">
                  <PermissionGate
                    fallback={<div className="empty-box">当前会话对 `packaging.prepare` 只有只读权限。</div>}
                    packageName={normalizedPackageName}
                    requiredAction="packaging.prepare"
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
                  >
                    <button
                      className="button button--secondary"
                      disabled={!workflow.canValidate || operationState.loading !== false}
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
                  >
                    <button
                      className="button button--secondary"
                      disabled={!workflow.canPack || operationState.loading !== false}
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
                  >
                    <button
                      className="button button--secondary"
                      disabled={!workflow.canCleanup || operationState.loading !== false}
                      onClick={() => void handleCleanup()}
                      type="button"
                    >
                      {operationState.loading === 'cleanup' ? '清理中…' : '清理 workspace'}
                    </button>
                  </PermissionGate>
                </div>
              </SurfaceCard>
            ) : null}
          </div>
        </SurfaceCard>

        <SurfaceCard className="surface-card--sticky" title="校验审阅" description="右侧始终展示当前预览 / 打包结果，方便确认能不能继续发版。" variant="rail">
          {!reviewData ? <div className="empty-box">先执行“检查并生成发布预览”，再决定是否继续发布。</div> : null}
          {reviewData ? (
            <div className="stack-column">
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
                  <article className="list-card">
                    <strong>默认模式会直接发布</strong>
                    <p>点击“发布 skill”后，Catalog 会完成 registry publish 和 release record；只有高级模式里的 Pack 仍然是手动交接。</p>
                  </article>
                </div>
              </SurfaceCard>

              <SurfaceCard title="发布就绪度" description="把当前阶段能不能进入下一步说清楚。">
                <div className="list-stack">
                  <article className="list-card">
                    <div className="list-card__head">
                      <div>
                        <strong>Prepare</strong>
                        <span>{workflow.prepareReadiness.detail}</span>
                      </div>
                      <StatusPill tone={workflow.prepareReadiness.tone}>{workflow.prepareReadiness.label}</StatusPill>
                    </div>
                  </article>
                  <article className="list-card">
                    <div className="list-card__head">
                      <div>
                        <strong>Validate</strong>
                        <span>{workflow.validateReadiness.detail}</span>
                      </div>
                      <StatusPill tone={workflow.validateReadiness.tone}>{workflow.validateReadiness.label}</StatusPill>
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
                title={publishResult ? '发布结果' : 'Pack 交接'}
                description={
                  publishResult
                    ? '后端已经完成 registry publish 和 release record。'
                    : '发布前最后确认的 bundle 元数据。'
                }
              >
                <dl className="property-list">
                  {'workspacePath' in reviewData ? (
                    <div>
                      <dt>workspace handle</dt>
                      <dd>{reviewData.workspacePath}</dd>
                    </div>
                  ) : null}
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
                  {publishResult ? (
                    <>
                      <div>
                        <dt>registry publish</dt>
                        <dd>{publishResult.publishedToRegistry ? '本次已执行' : '复用已有版本'}</dd>
                      </div>
                      <div>
                        <dt>release record</dt>
                        <dd>{publishResult.created ? '本次已创建' : '已存在，按幂等返回'}</dd>
                      </div>
                    </>
                  ) : null}
                  {hasBundlePath(reviewData) ? (
                    <div>
                      <dt>bundle base64</dt>
                      <dd>{reviewData.bundleBase64 ? '已包含在响应中' : '未包含'}</dd>
                    </div>
                  ) : null}
                </dl>
                {publishResult ? (
                  <div className="stack-column handoff-panel">
                    <div className="pill-row">
                      <StatusPill tone="good">发布已完成</StatusPill>
                    </div>
                    <p className="support-copy">
                      这次已经由后端完成 registry publish 和 release record，普通模式下不再暴露 bundle / handoff 细节。
                    </p>
                  </div>
                ) : null}
                {packResult && !publishResult && releaseHandoffPath && authoringHandoffPath ? (
                  <div className="stack-column handoff-panel">
                    <div className="pill-row">
                      <StatusPill tone="good">pack 已可进入发布交接</StatusPill>
                    </div>
                    <p className="support-copy">
                      这里还没有替你执行 npm publish，下载 bundle 并发布到目标 registry 后，再进入 release record。
                    </p>
                    <div className="authoring-actions">
                      <a
                        className="button button--secondary"
                        href={getPackagingBundleDownloadPath(packResult.workspacePath, packResult.packageName, packResult.version)}
                      >
                        下载 bundle
                      </a>
                      <Link className="button button--primary" to={releaseHandoffPath}>
                        发布后进入 release record
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
