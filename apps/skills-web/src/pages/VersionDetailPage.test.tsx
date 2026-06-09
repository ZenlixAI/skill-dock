import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import * as client from '../api/client';
import { SessionProvider } from '../session/SessionProvider';
import { VersionDetailPage } from './VersionDetailPage';

vi.mock('../api/client', () => ({
  applyVersionGovernanceChange: vi.fn(),
  createReleaseRecord: vi.fn(),
  getPackagingBundleDownloadPath: vi.fn(
    (workspacePath: string, packageName: string, version: string) =>
      `/api/v1/skill-packaging/bundle?workspacePath=${encodeURIComponent(workspacePath)}&packageName=${encodeURIComponent(packageName)}&version=${encodeURIComponent(version)}`
  ),
  getRegistryClientConfig: vi.fn(() => ({
    mode: 'real',
    baseUrl: null,
    role: null,
    actor: null,
    hasToken: false,
  })),
  getSession: vi.fn(),
  getVersionDetail: vi.fn(),
  listGovernanceChanges: vi.fn(),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('VersionDetailPage', () => {
  it('consumes provider session posture without issuing a page-local session fetch', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'operator',
      actor: 'operator',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });
    vi.mocked(client.getVersionDetail).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      version: '1.0.0',
      versionStatus: 'published',
      installPolicy: 'resolvable',
      publishedAt: '2026-01-01T00:00:00Z',
      registry: {
        npmRegistry: 'https://npm.internal.example.com',
        tarballUrl: 'https://npm.internal.example.com/agent-notes-1.0.0.tgz',
        integrity: 'sha512-test',
        shasum: 'test',
        publishedAt: '2026-01-01T00:00:00Z',
        distTags: ['latest'],
      },
      bundle: {
        skillManifestDigest: 'sha256:test',
        components: ['commands'],
      },
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
        <MemoryRouter initialEntries={['/skills/%40platform.skill%2Fagent-notes/versions/1.0.0']}>
          <Routes>
            <Route path="/skills/:packageName/versions/:version" element={<VersionDetailPage />} />
          </Routes>
        </MemoryRouter>
      </SessionProvider>
    );

    expect(await screen.findByText('发布登记只读')).toBeInTheDocument();
    await waitFor(() => expect(client.getSession).toHaveBeenCalledTimes(1));
  });

  it('shows the install command panel ahead of release and governance forms', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'operator',
      actor: 'operator',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });
    vi.mocked(client.getVersionDetail).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      version: '1.0.0',
      versionStatus: 'published',
      installPolicy: 'resolvable',
      publishedAt: '2026-01-01T00:00:00Z',
      registry: {
        npmRegistry: 'https://npm.internal.example.com',
        tarballUrl: 'https://npm.internal.example.com/agent-notes-1.0.0.tgz',
        integrity: 'sha512-test',
        shasum: 'test',
        publishedAt: '2026-01-01T00:00:00Z',
        distTags: ['latest'],
      },
      bundle: {
        skillManifestDigest: 'sha256:test',
        components: ['commands'],
      },
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
        <MemoryRouter initialEntries={['/skills/%40platform.skill%2Fagent-notes/versions/1.0.0']}>
          <Routes>
            <Route path="/skills/:packageName/versions/:version" element={<VersionDetailPage />} />
          </Routes>
        </MemoryRouter>
      </SessionProvider>
    );

    const installHeading = await screen.findByRole('heading', { name: '安装这个版本' });
    const releaseHeading = screen.getByRole('heading', { name: '发布登记' });
    const governanceHeading = screen.getByRole('heading', { name: '版本治理' });

    expect(
      screen.getByText('npm install @platform.skill/agent-notes@1.0.0 --registry=https://npm.internal.example.com')
    ).toBeInTheDocument();
    expect(screen.getAllByText('可直接安装').length).toBeGreaterThan(0);
    expect(installHeading.compareDocumentPosition(releaseHeading) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(installHeading.compareDocumentPosition(governanceHeading) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it('uses Chinese fallback copy when registry install data is unavailable', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'operator',
      actor: 'operator',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });
    vi.mocked(client.getVersionDetail).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      version: '1.0.0',
      versionStatus: 'published',
      installPolicy: 'catalog_only',
      publishedAt: '2026-01-01T00:00:00Z',
      registry: {
        npmRegistry: '',
        tarballUrl: '',
        integrity: '',
        shasum: 'test',
        publishedAt: '2026-01-01T00:00:00Z',
        distTags: ['latest'],
      },
      bundle: {
        skillManifestDigest: 'sha256:test',
        components: ['commands'],
      },
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
        <MemoryRouter initialEntries={['/skills/%40platform.skill%2Fagent-notes/versions/1.0.0']}>
          <Routes>
            <Route path="/skills/:packageName/versions/:version" element={<VersionDetailPage />} />
          </Routes>
        </MemoryRouter>
      </SessionProvider>
    );

    expect(await screen.findByRole('heading', { name: '安装状态' })).toBeInTheDocument();
    expect(screen.getByText('等仓库元数据返回后，这里会显示可直接复制的版本安装命令。')).toBeInTheDocument();
    expect(screen.getByText('当前还没有可用的仓库安装信息。')).toBeInTheDocument();
  });

  it('requires manual publish confirmation before enabling release-record submission', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'operator',
      actor: 'operator',
      authMechanism: 'session',
      allowedActions: ['session.read', 'catalog.release_record.create'],
      allowedPackages: [],
    });
    vi.mocked(client.getVersionDetail).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      version: '1.0.0',
      versionStatus: 'published',
      installPolicy: 'resolvable',
      publishedAt: '2026-01-01T00:00:00Z',
      registry: {
        npmRegistry: 'https://npm.internal.example.com',
        tarballUrl: 'https://npm.internal.example.com/agent-notes-1.0.0.tgz',
        integrity: 'sha512-test',
        shasum: 'test',
        publishedAt: '2026-01-01T00:00:00Z',
        distTags: ['latest'],
      },
      bundle: {
        skillManifestDigest: 'sha256:test',
        components: ['commands'],
      },
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
        <MemoryRouter initialEntries={['/skills/%40platform.skill%2Fagent-notes/versions/1.0.0']}>
          <Routes>
            <Route path="/skills/:packageName/versions/:version" element={<VersionDetailPage />} />
          </Routes>
        </MemoryRouter>
      </SessionProvider>
    );

    const submitButton = await screen.findByRole('button', { name: '创建 release record' });
    expect(submitButton).toBeDisabled();

    fireEvent.click(screen.getByLabelText('我确认这个版本已经完成 npm publish，可继续写 release record。'));

    expect(submitButton).not.toBeDisabled();
  });

  it('shows bundle handoff instructions when release handoff data is present', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'operator',
      actor: 'operator',
      authMechanism: 'session',
      allowedActions: ['session.read', 'catalog.release_record.create'],
      allowedPackages: [],
    });
    vi.mocked(client.getVersionDetail).mockResolvedValue({
      packageName: '@platform.skill/agent-notes',
      version: '1.0.0',
      versionStatus: 'published',
      installPolicy: 'resolvable',
      publishedAt: '2026-01-01T00:00:00Z',
      registry: {
        npmRegistry: 'https://npm.internal.example.com',
        tarballUrl: 'https://npm.internal.example.com/agent-notes-1.0.0.tgz',
        integrity: 'sha512-test',
        shasum: 'test',
        publishedAt: '2026-01-01T00:00:00Z',
        distTags: ['latest'],
      },
      bundle: {
        skillManifestDigest: 'sha256:test',
        components: ['commands'],
      },
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
        <MemoryRouter
          initialEntries={[
            '/skills/%40platform.skill%2Fagent-notes/versions/1.0.0?workspacePath=%2Ftmp%2Fskill-packaging%2Fworkspace&bundlePath=%2Ftmp%2Fskill-packaging%2Fworkspace%2F.dist%2Fagent-notes-1.0.0.tgz&manifestDigest=sha256%3Atest&components=commands&publishRegistry=https%3A%2F%2Fnpm.internal.example.com',
          ]}
        >
          <Routes>
            <Route path="/skills/:packageName/versions/:version" element={<VersionDetailPage />} />
          </Routes>
        </MemoryRouter>
      </SessionProvider>
    );

    expect(await screen.findByText('发布前检查清单')).toBeInTheDocument();
    expect(
      screen.getByText('先下载 bundle，并在你自己的终端里对目标 npm registry 执行 publish。')
    ).toBeInTheDocument();
  });
});
