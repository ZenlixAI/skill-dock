import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import * as client from '../api/client';
import * as externalImport from '../features/packaging/externalImport';
import { RegistryHttpError } from '../api/httpClient';
import { SessionProvider } from '../session/SessionProvider';
import { ImportPage } from './ImportPage';

vi.mock('../api/client', () => ({
  cleanupPackagingWorkspace: vi.fn(),
  getRegistryClientConfig: vi.fn(() => ({
    mode: 'real',
    baseUrl: null,
    role: null,
    actor: null,
    hasToken: false,
  })),
  getSession: vi.fn().mockResolvedValue({
    role: 'operator',
    actor: 'operator',
    authMechanism: 'session',
    allowedActions: [
      'session.read',
      'packaging.prepare',
      'packaging.validate',
      'packaging.pack',
      'packaging.publish',
    ],
    allowedPackages: [],
  }),
  getPackagingBundleDownloadPath: vi.fn(
    (workspacePath: string, packageName: string, version: string) =>
      `/api/v1/skill-packaging/bundle?workspacePath=${encodeURIComponent(workspacePath)}&packageName=${encodeURIComponent(packageName)}&version=${encodeURIComponent(version)}`
  ),
  packSkill: vi.fn(),
  publishSkill: vi.fn(),
  preparePackaging: vi.fn(),
  validatePackaging: vi.fn(),
}));

vi.mock('../features/packaging/externalImport', () => ({
  buildExternalImportArchive: vi.fn(),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  vi.restoreAllMocks();
});

beforeEach(() => {
  vi.mocked(client.getSession).mockResolvedValue({
    role: 'operator',
    actor: 'operator',
    authMechanism: 'session',
    allowedActions: [
      'session.read',
      'packaging.prepare',
      'packaging.validate',
      'packaging.pack',
      'packaging.publish',
    ],
    allowedPackages: [],
  });
  vi.mocked(externalImport.buildExternalImportArchive).mockResolvedValue({
    archiveBase64: 'archive-base64',
  });
});

function renderImportPage() {
  return render(
    <SessionProvider>
      <MemoryRouter>
        <ImportPage />
      </MemoryRouter>
    </SessionProvider>
  );
}

describe('ImportPage', () => {
  it('starts with streamlined preview and publish actions instead of raw protocol controls', async () => {
    renderImportPage();

    expect(await screen.findByDisplayValue('https://github.com/mattpocock/skills?path=skills/engineering/zoom-out')).toBeInTheDocument();
    await waitFor(() => expect(client.getSession).toHaveBeenCalledTimes(1));
    expect(screen.getByRole('button', { name: '检查并生成发布预览' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '发布 skill' })).toBeDisabled();
    expect(screen.getByRole('button', { name: '高级模式：查看 prepare / validate / pack' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Prepare' })).not.toBeInTheDocument();
    expect(screen.getByPlaceholderText('留空使用默认分支；也可填 release / commit sha')).toHaveValue('');
    expect(screen.getByDisplayValue('@company.skill/zoom-out')).toBeInTheDocument();
    expect(screen.getByDisplayValue('Zoom Out')).toBeInTheDocument();
  });

  it('reveals raw protocol controls only after opening advanced mode', async () => {
    renderImportPage();

    fireEvent.click(await screen.findByRole('button', { name: '高级模式：查看 prepare / validate / pack' }));

    expect(screen.getByRole('button', { name: 'Prepare' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Validate' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Pack' })).toBeInTheDocument();
  });

  it('shows a template-driven authoring editor for platform_authoring', async () => {
    renderImportPage();

    fireEvent.change(await screen.findByLabelText('来源类型'), {
      target: { value: 'platform_authoring' },
    });

    expect(screen.getByLabelText('模板')).toHaveValue('command');
    expect(screen.getByDisplayValue('commands/run.txt')).toBeInTheDocument();
    expect(screen.getByLabelText('当前文件内容')).toHaveValue('echo authored\n');
  });

  it('surfaces package_filter_required in the session posture pill when a scoped pack action has no package focus', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'operator',
      actor: 'operator',
      authMechanism: 'session',
      allowedActions: ['session.read', 'packaging.pack'],
      allowedPackages: ['@company.skill/*'],
    });

    renderImportPage();

    fireEvent.change(await screen.findByLabelText('packageName'), {
      target: { value: '' },
    });

    const matches = await screen.findAllByText('需要先选定包名');
    expect(matches.some((element) => element.classList.contains('status-pill'))).toBe(true);
  });

  it('allows adding and deleting non-locked authoring files', async () => {
    renderImportPage();

    fireEvent.change(await screen.findByLabelText('来源类型'), {
      target: { value: 'platform_authoring' },
    });

    fireEvent.change(screen.getByLabelText('新文件路径'), {
      target: { value: 'docs/README.md' },
    });
    fireEvent.click(screen.getByRole('button', { name: '添加文件' }));

    expect(screen.getByDisplayValue('docs/README.md')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '删除 docs/README.md' }));

    expect(screen.queryByDisplayValue('docs/README.md')).not.toBeInTheDocument();
  });

  it('switches templates and generates a preview from the authoring files', async () => {
    vi.mocked(client.preparePackaging).mockResolvedValue({
      workspacePath: '/tmp/skill-packaging/workspace',
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      warnings: [],
      warningDetails: [],
      errors: [],
    });
    vi.mocked(client.validatePackaging).mockResolvedValue({
      ok: true,
      workspacePath: '/tmp/skill-packaging/workspace',
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      manifestDigest: 'sha256:manifest',
      components: ['workflows'],
      warnings: [],
      warningDetails: [],
      errors: [],
    });

    renderImportPage();

    fireEvent.change(await screen.findByLabelText('来源类型'), {
      target: { value: 'platform_authoring' },
    });

    fireEvent.change(screen.getByLabelText('模板'), {
      target: { value: 'workflow' },
    });
    fireEvent.change(screen.getByLabelText('当前文件内容'), {
      target: { value: 'step 1\nstep 2\n' },
    });
    fireEvent.click(screen.getByRole('button', { name: '检查并生成发布预览' }));

    expect(await screen.findByText('预览已生成，可以继续发布 skill。')).toBeInTheDocument();
    expect(client.preparePackaging).toHaveBeenCalledWith(
      expect.objectContaining({
        source: {
          type: 'platform_authoring',
          files: {
            'workflows/main.md': 'step 1\nstep 2\n',
          },
        },
      })
    );
    expect(client.validatePackaging).toHaveBeenCalledWith({
      workspacePath: '/tmp/skill-packaging/workspace',
      target: {
        packageName: '@company.skill/zoom-out',
        version: '1.2.0',
      },
    });
  });

  it('applies advanced raw json and uses it in the prepare payload', async () => {
    vi.mocked(client.preparePackaging).mockResolvedValue({
      workspacePath: '/tmp/skill-packaging/workspace',
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      warnings: [],
      warningDetails: [],
      errors: [],
    });
    vi.mocked(client.validatePackaging).mockResolvedValue({
      ok: true,
      workspacePath: '/tmp/skill-packaging/workspace',
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      manifestDigest: 'sha256:manifest',
      components: ['docs'],
      warnings: [],
      warningDetails: [],
      errors: [],
    });

    renderImportPage();

    fireEvent.change(await screen.findByLabelText('来源类型'), {
      target: { value: 'platform_authoring' },
    });

    fireEvent.click(screen.getByRole('button', { name: '高级模式 / 原始 files JSON' }));
    fireEvent.change(screen.getByLabelText('原始 files JSON'), {
      target: {
        value: '{\n  "docs/README.md": "hello\\n"\n}',
      },
    });
    fireEvent.click(screen.getByRole('button', { name: '应用 JSON 到文件列表' }));

    expect(screen.getByDisplayValue('docs/README.md')).toBeInTheDocument();
    expect(screen.getByLabelText('当前文件内容')).toHaveValue('hello\n');

    fireEvent.click(screen.getByRole('button', { name: '检查并生成发布预览' }));

    expect(await screen.findByText('预览已生成，可以继续发布 skill。')).toBeInTheDocument();
    expect(client.preparePackaging).toHaveBeenCalledWith(
      expect.objectContaining({
        source: {
          type: 'platform_authoring',
          files: {
            'docs/README.md': 'hello\n',
          },
        },
      })
    );
  });

  it('uses dirty advanced raw json in the prepare payload before apply', async () => {
    vi.mocked(client.preparePackaging).mockResolvedValue({
      workspacePath: '/tmp/skill-packaging/workspace',
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      warnings: [],
      warningDetails: [],
      errors: [],
    });
    vi.mocked(client.validatePackaging).mockResolvedValue({
      ok: true,
      workspacePath: '/tmp/skill-packaging/workspace',
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      manifestDigest: 'sha256:manifest',
      components: ['docs'],
      warnings: [],
      warningDetails: [],
      errors: [],
    });

    renderImportPage();

    fireEvent.change(await screen.findByLabelText('来源类型'), {
      target: { value: 'platform_authoring' },
    });

    fireEvent.click(screen.getByRole('button', { name: '高级模式 / 原始 files JSON' }));
    fireEvent.change(screen.getByLabelText('原始 files JSON'), {
      target: {
        value: '{\n  "docs/README.md": "hello\\n"\n}',
      },
    });

    fireEvent.click(screen.getByRole('button', { name: '检查并生成发布预览' }));

    expect(await screen.findByText('预览已生成，可以继续发布 skill。')).toBeInTheDocument();
    expect(client.preparePackaging).toHaveBeenCalledWith(
      expect.objectContaining({
        source: {
          type: 'platform_authoring',
          files: {
            'docs/README.md': 'hello\n',
          },
        },
      })
    );
  });

  it('builds an upload_archive payload for external import', async () => {
    vi.mocked(client.preparePackaging).mockResolvedValue({
      workspacePath: '/tmp/skill-packaging/workspace',
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      warnings: [],
      warningDetails: [],
      errors: [],
    });
    vi.mocked(client.validatePackaging).mockResolvedValue({
      ok: true,
      workspacePath: '/tmp/skill-packaging/workspace',
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      manifestDigest: 'sha256:manifest',
      components: ['commands'],
      warnings: [],
      warningDetails: [],
      errors: [],
    });
    renderImportPage();

    fireEvent.click(await screen.findByRole('button', { name: '检查并生成发布预览' }));

    await waitFor(() =>
      expect(client.preparePackaging).toHaveBeenCalledWith(
        expect.objectContaining({
          source: expect.objectContaining({
            type: 'upload_archive',
            archiveBase64: expect.any(String),
          }),
        })
      )
    );
    expect(externalImport.buildExternalImportArchive).toHaveBeenCalledWith(
      'https://github.com/mattpocock/skills?path=skills/engineering/zoom-out',
      undefined
    );
  });

  it('does not call preparePackaging when external import acquisition fails', async () => {
    vi.mocked(externalImport.buildExternalImportArchive).mockRejectedValue(
      new Error('无法获取外部 skill 源，请检查地址、仓库可见性或浏览器网络限制。')
    );

    renderImportPage();

    fireEvent.click(await screen.findByRole('button', { name: '检查并生成发布预览' }));

    expect(await screen.findByText('无法获取外部 skill 源，请检查地址、仓库可见性或浏览器网络限制。')).toBeInTheDocument();
    expect(client.preparePackaging).not.toHaveBeenCalled();
  });

  it('publishes through the backend publish api and stops showing manual handoff links', async () => {
    vi.mocked(client.preparePackaging).mockResolvedValue({
      workspacePath: '/tmp/skill-packaging/workspace',
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      warnings: [],
      warningDetails: [],
      errors: [],
    });
    vi.mocked(client.validatePackaging).mockResolvedValue({
      ok: true,
      workspacePath: '/tmp/skill-packaging/workspace',
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      manifestDigest: 'sha256:manifest',
      components: ['commands'],
      warnings: [],
      warningDetails: [],
      errors: [],
    });
    vi.mocked(client.publishSkill).mockResolvedValue({
      workspacePath: '/tmp/skill-packaging/workspace',
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      manifestDigest: 'sha256:manifest',
      components: ['commands'],
      publishedToRegistry: true,
      created: true,
      idempotentReplay: false,
      versionStatus: 'published',
      installPolicy: 'resolvable',
      warnings: [],
      warningDetails: [],
      errors: [],
    });

    renderImportPage();

    fireEvent.click(await screen.findByRole('button', { name: '检查并生成发布预览' }));
    fireEvent.click(await screen.findByRole('button', { name: '发布 skill' }));

    expect(client.publishSkill).toHaveBeenCalledWith({
      source: {
        type: 'upload_archive',
        archiveBase64: 'archive-base64',
      },
      target: {
        packageName: '@company.skill/zoom-out',
        version: '1.2.0',
      },
      metadata: {
        displayName: 'Zoom Out',
        description: 'Help me see the bigger picture',
        sourceType: 'imported',
      },
      options: {},
      review: {
        manifestDigest: 'sha256:manifest',
        components: ['commands'],
      },
    });
    expect(await screen.findByText('skill 已发布到 registry，并已写入 release record。')).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: '下载 bundle' })).not.toBeInTheDocument();
    expect(screen.queryByText('这里还没有替你执行 npm publish，下载 bundle 并发布到目标 registry 后，再进入 release record。')).not.toBeInTheDocument();
  });

  it('shows the validated stage after preview succeeds', async () => {
    vi.mocked(client.preparePackaging).mockResolvedValue({
      workspacePath: '/tmp/skill-packaging/workspace',
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      warnings: [],
      warningDetails: [],
      errors: [],
    });
    vi.mocked(client.validatePackaging).mockResolvedValue({
      ok: true,
      workspacePath: '/tmp/skill-packaging/workspace',
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      manifestDigest: 'sha256:manifest',
      components: ['commands'],
      warnings: [],
      warningDetails: [],
      errors: [],
    });

    renderImportPage();

    fireEvent.click(await screen.findByRole('button', { name: '检查并生成发布预览' }));

    expect(await screen.findByText('阶段 validated')).toBeInTheDocument();
  });

  it('preserves the enriched prepare failure message in the mutation panel', async () => {
    vi.mocked(client.preparePackaging).mockRejectedValue(
      new RegistryHttpError('forbidden', 403, {
        error: {
          requestId: 'req_123',
        },
      })
    );

    renderImportPage();

    fireEvent.click(await screen.findByRole('button', { name: '检查并生成发布预览' }));

    expect(
      await screen.findByText(
        'Current session is not allowed to perform this action. 这次导入操作需要具备 packaging 权限的会话后再重试。'
      )
    ).toBeInTheDocument();
  });
});
