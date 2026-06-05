import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import * as client from '../api/client';
import { ImportPage } from './ImportPage';

vi.mock('../api/client', () => ({
  cleanupPackagingWorkspace: vi.fn(),
  getSession: vi.fn().mockResolvedValue({
    role: 'operator',
    actor: 'operator',
    authMechanism: 'session',
    allowedActions: ['session.read', 'packaging.prepare', 'packaging.validate', 'packaging.pack'],
    allowedPackages: [],
  }),
  getPackagingBundleDownloadPath: vi.fn(
    (workspacePath: string, packageName: string, version: string) =>
      `/api/v1/skill-packaging/bundle?workspacePath=${encodeURIComponent(workspacePath)}&packageName=${encodeURIComponent(packageName)}&version=${encodeURIComponent(version)}`
  ),
  packSkill: vi.fn(),
  preparePackaging: vi.fn(),
  validatePackaging: vi.fn(),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('ImportPage', () => {
  it('starts with a working git import example instead of a broken placeholder path', async () => {
    render(
      <MemoryRouter>
        <ImportPage />
      </MemoryRouter>
    );

    expect(await screen.findByDisplayValue('https://github.com/mattpocock/skills?path=skills/engineering/zoom-out')).toBeInTheDocument();
    expect(screen.getByText('4. 打包与交接')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('留空使用默认分支；也可填 release / commit sha')).toHaveValue('');
    expect(screen.getByDisplayValue('@company.skill/zoom-out')).toBeInTheDocument();
    expect(screen.getByDisplayValue('Zoom Out')).toBeInTheDocument();
  });

  it('shows a template-driven authoring editor for platform_authoring', async () => {
    render(
      <MemoryRouter>
        <ImportPage />
      </MemoryRouter>
    );

    fireEvent.change(await screen.findByLabelText('来源类型'), {
      target: { value: 'platform_authoring' },
    });

    expect(screen.getByLabelText('模板')).toHaveValue('command');
    expect(screen.getByDisplayValue('commands/run.txt')).toBeInTheDocument();
    expect(screen.getByLabelText('当前文件内容')).toHaveValue('echo authored\n');
  });

  it('allows adding and deleting non-locked authoring files', async () => {
    render(
      <MemoryRouter>
        <ImportPage />
      </MemoryRouter>
    );

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

  it('switches templates and sends generated files in the prepare payload', async () => {
    vi.mocked(client.preparePackaging).mockResolvedValue({
      workspacePath: '/tmp/skill-packaging/workspace',
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      warnings: [],
      warningDetails: [],
      errors: [],
    });

    render(
      <MemoryRouter>
        <ImportPage />
      </MemoryRouter>
    );

    fireEvent.change(await screen.findByLabelText('来源类型'), {
      target: { value: 'platform_authoring' },
    });

    fireEvent.change(screen.getByLabelText('模板'), {
      target: { value: 'workflow' },
    });
    fireEvent.change(screen.getByLabelText('当前文件内容'), {
      target: { value: 'step 1\nstep 2\n' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Prepare' }));

    expect(await screen.findByText('prepare 完成，已经拿到 workspacePath 和 import review 数据。')).toBeInTheDocument();
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
  });

  it('offers a bundle download handoff after pack instead of implying the bundle was already published', async () => {
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
    vi.mocked(client.packSkill).mockResolvedValue({
      workspacePath: '/tmp/skill-packaging/workspace',
      bundlePath: '/tmp/skill-packaging/workspace/.dist/zoom-out-1.2.0.tgz',
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      manifestDigest: 'sha256:manifest',
      components: ['commands'],
      warnings: [],
      warningDetails: [],
      errors: [],
    });

    render(
      <MemoryRouter>
        <ImportPage />
      </MemoryRouter>
    );

    fireEvent.click(await screen.findByRole('button', { name: 'Prepare' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Validate' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Pack' }));

    expect(await screen.findByRole('link', { name: '下载 bundle' })).toHaveAttribute(
      'href',
      '/api/v1/skill-packaging/bundle?workspacePath=%2Ftmp%2Fskill-packaging%2Fworkspace&packageName=%40company.skill%2Fzoom-out&version=1.2.0'
    );
    expect(screen.getByText('这里还没有替你执行 npm publish，下载 bundle 并发布到目标 registry 后，再进入 release record。')).toBeInTheDocument();
  });
});
