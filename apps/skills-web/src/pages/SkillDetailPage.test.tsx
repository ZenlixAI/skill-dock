import { cleanup, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import * as client from '../api/client';
import { SessionProvider } from '../session/SessionProvider';
import { SkillDetailPage } from './SkillDetailPage';

vi.mock('../api/client', () => ({
  applySkillGovernanceChange: vi.fn(),
  getRegistryClientConfig: vi.fn(() => ({
    mode: 'real',
    baseUrl: null,
    role: null,
    actor: null,
    hasToken: false,
  })),
  getSession: vi.fn(),
  getInstallTarget: vi.fn(),
  getSkillDetail: vi.fn(),
  getSkillVersions: vi.fn(),
  listGovernanceChanges: vi.fn(),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('SkillDetailPage', () => {
  it('consumes provider session posture without issuing a page-local session fetch', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'operator',
      actor: 'operator',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });
    vi.mocked(client.getSkillDetail).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      displayName: 'Agent Notes',
      description: 'Detail view',
      sourceType: 'native',
      status: 'active',
      latestVersion: '1.0.0',
      latestVersionStatus: 'published',
      updatedAt: '2026-01-01T00:00:00Z',
      createdAt: '2026-01-01T00:00:00Z',
    });
    vi.mocked(client.getSkillVersions).mockResolvedValue({
      items: [],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });
    vi.mocked(client.getInstallTarget).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      version: '1.0.0',
      npmRegistry: 'https://npm.internal.example.com',
      tarballUrl: 'https://npm.internal.example.com/agent-notes.tgz',
      integrity: 'sha512-test',
      skillManifestDigest: 'sha256:test',
      versionStatus: 'published',
    });
    vi.mocked(client.listGovernanceChanges).mockResolvedValue({
      items: [],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });

    render(
      <SessionProvider>
        <MemoryRouter initialEntries={['/skills/%40platform.skill%2Fagent-notes']}>
          <Routes>
            <Route path="/skills/:packageName" element={<SkillDetailPage />} />
          </Routes>
        </MemoryRouter>
      </SessionProvider>
    );

    expect(await screen.findByText('只读')).toBeInTheDocument();
    await waitFor(() => expect(client.getSession).toHaveBeenCalledTimes(1));
  });

  it('shows the install command panel above the version list when install target data is present', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'operator',
      actor: 'operator',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });
    vi.mocked(client.getSkillDetail).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      displayName: 'Agent Notes',
      description: 'Detail view',
      sourceType: 'native',
      status: 'active',
      latestVersion: '1.0.0',
      latestVersionStatus: 'published',
      updatedAt: '2026-01-01T00:00:00Z',
      createdAt: '2026-01-01T00:00:00Z',
    });
    vi.mocked(client.getSkillVersions).mockResolvedValue({
      items: [
        {
          packageName: '@platform.skill/agent-notes',
          version: '1.0.0',
          versionStatus: 'published',
          installPolicy: 'resolvable',
          publishedAt: '2026-01-01T00:00:00Z',
        },
      ],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });
    vi.mocked(client.getInstallTarget).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      version: '1.0.0',
      npmRegistry: 'https://npm.internal.example.com',
      tarballUrl: 'https://npm.internal.example.com/agent-notes.tgz',
      integrity: 'sha512-test',
      skillManifestDigest: 'sha256:test',
      versionStatus: 'published',
    });
    vi.mocked(client.listGovernanceChanges).mockResolvedValue({
      items: [],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });

    render(
      <SessionProvider>
        <MemoryRouter initialEntries={['/skills/%40platform.skill%2Fagent-notes']}>
          <Routes>
            <Route path="/skills/:packageName" element={<SkillDetailPage />} />
          </Routes>
        </MemoryRouter>
      </SessionProvider>
    );

    const installHeading = await screen.findByRole('heading', { name: '安装这个技能' });
    const versionsHeading = screen.getByRole('heading', { name: '版本列表' });

    expect(
      screen.getByText('npm install @platform.skill/agent-notes@1.0.0 --registry=https://npm.internal.example.com')
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '复制安装命令' })).toBeInTheDocument();
    expect(screen.getAllByText('可直接安装').length).toBeGreaterThan(0);
    expect(screen.getAllByText('已发布').length).toBeGreaterThan(0);
    expect(screen.getAllByText('原生').length).toBeGreaterThan(0);
    expect(screen.getAllByText('启用').length).toBeGreaterThan(0);
    expect(screen.getByText('仓库安装态势')).toBeInTheDocument();
    expect(screen.queryByText('published')).not.toBeInTheDocument();
    expect(screen.queryByText('resolvable')).not.toBeInTheDocument();
    expect(screen.queryByText('native')).not.toBeInTheDocument();
    expect(installHeading.compareDocumentPosition(versionsHeading) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it('shows a loading hero while install target data is still loading', () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'operator',
      actor: 'operator',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });
    vi.mocked(client.getSkillDetail).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      displayName: 'Agent Notes',
      description: 'Detail view',
      sourceType: 'native',
      status: 'active',
      latestVersion: '1.0.0',
      latestVersionStatus: 'published',
      updatedAt: '2026-01-01T00:00:00Z',
      createdAt: '2026-01-01T00:00:00Z',
    });
    vi.mocked(client.getSkillVersions).mockResolvedValue({
      items: [
        {
          packageName: '@platform.skill/agent-notes',
          version: '1.0.0',
          versionStatus: 'published',
          installPolicy: 'resolvable',
          publishedAt: '2026-01-01T00:00:00Z',
        },
      ],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });
    vi.mocked(client.getInstallTarget).mockReturnValue(new Promise(() => {}));
    vi.mocked(client.listGovernanceChanges).mockResolvedValue({
      items: [],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });

    render(
      <SessionProvider>
        <MemoryRouter initialEntries={['/skills/%40platform.skill%2Fagent-notes']}>
          <Routes>
            <Route path="/skills/:packageName" element={<SkillDetailPage />} />
          </Routes>
        </MemoryRouter>
      </SessionProvider>
    );

    expect(screen.getByRole('heading', { name: '安装状态' })).toBeInTheDocument();
    expect(screen.getByText('正在等待安装目标返回，安装命令会在解析完成后显示在这里。')).toBeInTheDocument();
    expect(screen.getByText('正在加载安装目标，请稍候。')).toBeInTheDocument();
  });

  it('shows a policy-loading hero after the install target is resolved but version policy is still loading', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'operator',
      actor: 'operator',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });
    vi.mocked(client.getSkillDetail).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      displayName: 'Agent Notes',
      description: 'Detail view',
      sourceType: 'native',
      status: 'active',
      latestVersion: '1.0.0',
      latestVersionStatus: 'published',
      updatedAt: '2026-01-01T00:00:00Z',
      createdAt: '2026-01-01T00:00:00Z',
    });
    vi.mocked(client.getSkillVersions).mockReturnValue(new Promise(() => {}));
    vi.mocked(client.getInstallTarget).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      version: '1.0.0',
      npmRegistry: 'https://npm.internal.example.com',
      tarballUrl: 'https://npm.internal.example.com/agent-notes.tgz',
      integrity: 'sha512-test',
      skillManifestDigest: 'sha256:test',
      versionStatus: 'published',
    });
    vi.mocked(client.listGovernanceChanges).mockResolvedValue({
      items: [],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });

    render(
      <SessionProvider>
        <MemoryRouter initialEntries={['/skills/%40platform.skill%2Fagent-notes']}>
          <Routes>
            <Route path="/skills/:packageName" element={<SkillDetailPage />} />
          </Routes>
        </MemoryRouter>
      </SessionProvider>
    );

    expect(await screen.findByRole('heading', { name: '安装状态' })).toBeInTheDocument();
    expect(screen.getByText('安装目标已解析，正在等待对应版本的安装策略返回。')).toBeInTheDocument();
    expect(screen.getByText('正在加载安装策略，请稍候。')).toBeInTheDocument();
  });

  it('shows an error hero when the install target request fails', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'operator',
      actor: 'operator',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });
    vi.mocked(client.getSkillDetail).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      displayName: 'Agent Notes',
      description: 'Detail view',
      sourceType: 'native',
      status: 'active',
      latestVersion: null,
      latestVersionStatus: null,
      updatedAt: '2026-01-01T00:00:00Z',
      createdAt: '2026-01-01T00:00:00Z',
    });
    vi.mocked(client.getSkillVersions).mockResolvedValue({
      items: [],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });
    vi.mocked(client.getInstallTarget).mockRejectedValue(new Error('not available'));
    vi.mocked(client.listGovernanceChanges).mockResolvedValue({
      items: [],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });

    render(
      <SessionProvider>
        <MemoryRouter initialEntries={['/skills/%40platform.skill%2Fagent-notes']}>
          <Routes>
            <Route path="/skills/:packageName" element={<SkillDetailPage />} />
          </Routes>
        </MemoryRouter>
      </SessionProvider>
    );

    expect(await screen.findByRole('heading', { name: '安装状态' })).toBeInTheDocument();
    expect(screen.getByText('安装目标请求失败，暂时无法生成可复制的默认安装命令。')).toBeInTheDocument();
    expect(screen.getByText('not available')).toBeInTheDocument();
  });

  it('shows an unavailable hero when install target data is absent', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'operator',
      actor: 'operator',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });
    vi.mocked(client.getSkillDetail).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      displayName: 'Agent Notes',
      description: 'Detail view',
      sourceType: 'native',
      status: 'active',
      latestVersion: null,
      latestVersionStatus: null,
      updatedAt: '2026-01-01T00:00:00Z',
      createdAt: '2026-01-01T00:00:00Z',
    });
    vi.mocked(client.getSkillVersions).mockResolvedValue({
      items: [],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });
    vi.mocked(client.getInstallTarget).mockResolvedValue(null as never);
    vi.mocked(client.listGovernanceChanges).mockResolvedValue({
      items: [],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });

    render(
      <SessionProvider>
        <MemoryRouter initialEntries={['/skills/%40platform.skill%2Fagent-notes']}>
          <Routes>
            <Route path="/skills/:packageName" element={<SkillDetailPage />} />
          </Routes>
        </MemoryRouter>
      </SessionProvider>
    );

    expect(await screen.findByRole('heading', { name: '安装状态' })).toBeInTheDocument();
    expect(screen.getByText('等安装目标被仓库解析出来后，这里会显示可直接复制的命令。')).toBeInTheDocument();
    expect(screen.getByText('当前还没有可用的安装目标。请先检查最新版本是否已经发布并可解析。')).toBeInTheDocument();
  });

  it('keeps the shared install panel when install target data is present but registry metadata is missing', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'operator',
      actor: 'operator',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });
    vi.mocked(client.getSkillDetail).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      displayName: 'Agent Notes',
      description: 'Detail view',
      sourceType: 'native',
      status: 'active',
      latestVersion: '1.0.0',
      latestVersionStatus: 'published',
      updatedAt: '2026-01-01T00:00:00Z',
      createdAt: '2026-01-01T00:00:00Z',
    });
    vi.mocked(client.getSkillVersions).mockResolvedValue({
      items: [
        {
          packageName: '@platform.skill/agent-notes',
          version: '1.0.0',
          versionStatus: 'published',
          installPolicy: 'catalog_only',
          publishedAt: '2026-01-01T00:00:00Z',
        },
      ],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });
    vi.mocked(client.getInstallTarget).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      version: '1.0.0',
      npmRegistry: '',
      tarballUrl: '',
      integrity: '',
      skillManifestDigest: 'sha256:test',
      versionStatus: 'published',
    });
    vi.mocked(client.listGovernanceChanges).mockResolvedValue({
      items: [],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });

    render(
      <SessionProvider>
        <MemoryRouter initialEntries={['/skills/%40platform.skill%2Fagent-notes']}>
          <Routes>
            <Route path="/skills/:packageName" element={<SkillDetailPage />} />
          </Routes>
        </MemoryRouter>
      </SessionProvider>
    );

    expect(await screen.findByRole('heading', { name: '安装这个技能' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: '安装状态' })).not.toBeInTheDocument();
    expect(screen.getAllByText('待补充 npm 仓库地址').length).toBeGreaterThan(0);
    expect(
      screen.getByText('npm install @platform.skill/agent-notes@1.0.0 --registry=<待补充 npm 仓库地址>')
    ).toBeInTheDocument();
  });

  it('uses the install policy that matches the surfaced install target version', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'operator',
      actor: 'operator',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });
    vi.mocked(client.getSkillDetail).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      displayName: 'Agent Notes',
      description: 'Detail view',
      sourceType: 'native',
      status: 'active',
      latestVersion: '2.0.0',
      latestVersionStatus: 'published',
      updatedAt: '2026-01-01T00:00:00Z',
      createdAt: '2026-01-01T00:00:00Z',
    });
    vi.mocked(client.getSkillVersions).mockResolvedValue({
      items: [
        {
          packageName: '@platform.skill/agent-notes',
          version: '2.0.0',
          versionStatus: 'published',
          installPolicy: 'blocked',
          publishedAt: '2026-02-01T00:00:00Z',
        },
        {
          packageName: '@platform.skill/agent-notes',
          version: '1.9.0',
          versionStatus: 'published',
          installPolicy: 'resolvable',
          publishedAt: '2026-01-15T00:00:00Z',
        },
      ],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });
    vi.mocked(client.getInstallTarget).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      version: '1.9.0',
      npmRegistry: 'https://npm.internal.example.com',
      tarballUrl: 'https://npm.internal.example.com/agent-notes-1.9.0.tgz',
      integrity: 'sha512-test',
      skillManifestDigest: 'sha256:test',
      versionStatus: 'published',
    });
    vi.mocked(client.listGovernanceChanges).mockResolvedValue({
      items: [],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });

    render(
      <SessionProvider>
        <MemoryRouter initialEntries={['/skills/%40platform.skill%2Fagent-notes']}>
          <Routes>
            <Route path="/skills/:packageName" element={<SkillDetailPage />} />
          </Routes>
        </MemoryRouter>
      </SessionProvider>
    );

    const installHeading = await screen.findByRole('heading', { name: '安装这个技能' });
    const installPanel = installHeading.closest('section');

    expect(installPanel).not.toBeNull();
    expect(within(installPanel as HTMLElement).getByText('npm install @platform.skill/agent-notes@1.9.0 --registry=https://npm.internal.example.com')).toBeInTheDocument();
    expect(within(installPanel as HTMLElement).getByText('可直接安装')).toBeInTheDocument();
    expect(within(installPanel as HTMLElement).queryByText('已阻断安装')).not.toBeInTheDocument();
  });
});
