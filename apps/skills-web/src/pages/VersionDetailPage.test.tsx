import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import * as client from '../api/client';
import { SessionProvider } from '../session/SessionProvider';
import { VersionDetailPage } from './VersionDetailPage';

vi.mock('../api/client', () => ({
  applyVersionGovernanceChange: vi.fn(),
  createReleaseRecord: vi.fn(),
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
  const expectedCommand =
    `npx --registry ${window.location.origin}/repository/skills-npm/ skills add @platform.skill/agent-notes --version 1.0.0`;

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
        npmRegistry: 'http://192.168.10.127:32081/repository/skills-npm/',
        tarballUrl: 'https://npm.internal.example.com/@platform.skill/agent-notes/-/agent-notes-1.0.0.tgz',
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
        npmRegistry: 'http://192.168.10.127:32081/repository/skills-npm/',
        tarballUrl: 'https://npm.internal.example.com/@platform.skill/agent-notes/-/agent-notes-1.0.0.tgz',
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
      screen.getByDisplayValue(expectedCommand)
    ).toBeInTheDocument();
    expect(screen.getAllByText('可直接安装').length).toBeGreaterThan(0);
    expect(screen.getAllByText('已发布').length).toBeGreaterThan(0);
    expect(screen.getByText('仓库与打包产物')).toBeInTheDocument();
    expect(screen.getAllByText('http://192.168.10.127:32081').length).toBeGreaterThan(0);
    expect(
      screen.getAllByText('http://192.168.10.127:32081/@platform.skill/agent-notes/-/agent-notes-1.0.0.tgz').length
    ).toBeGreaterThan(0);
    expect(screen.getAllByText('完整性校验').length).toBeGreaterThan(0);
    expect(screen.getByText('校验摘要')).toBeInTheDocument();
    expect(screen.getByText('组件清单')).toBeInTheDocument();
    expect(screen.queryByText('integrity')).not.toBeInTheDocument();
    expect(screen.queryByText('shasum')).not.toBeInTheDocument();
    expect(screen.queryByText('components')).not.toBeInTheDocument();
    expect(screen.queryByText('published')).not.toBeInTheDocument();
    expect(screen.queryByText('http://192.168.10.127:32081/repository/skills-npm/')).not.toBeInTheDocument();
    expect(installHeading.compareDocumentPosition(releaseHeading) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(installHeading.compareDocumentPosition(governanceHeading) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it('shows a loading hero while version install data is still loading', () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'operator',
      actor: 'operator',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });
    vi.mocked(client.getVersionDetail).mockReturnValue(new Promise(() => {}));
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

    expect(screen.getByRole('heading', { name: '安装状态' })).toBeInTheDocument();
    expect(screen.getByText('正在等待版本安装数据返回，安装命令会在解析完成后显示在这里。')).toBeInTheDocument();
    expect(screen.getByText('正在加载这个版本的安装信息，请稍候。')).toBeInTheDocument();
  });

  it('shows an error hero when the version install request fails', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'operator',
      actor: 'operator',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });
    vi.mocked(client.getVersionDetail).mockRejectedValue(new Error('detail unavailable'));
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

    expect(
      await screen.findByText('版本安装数据请求失败，暂时无法生成可复制的安装命令。')
    ).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: '安装状态' })).toBeInTheDocument();
    expect(screen.getByText('detail unavailable')).toBeInTheDocument();
  });

  it('shows an unavailable hero when version install data is absent', async () => {
    vi.mocked(client.getSession).mockResolvedValue({
      role: 'operator',
      actor: 'operator',
      authMechanism: 'session',
      allowedActions: ['session.read'],
      allowedPackages: [],
    });
    vi.mocked(client.getVersionDetail).mockResolvedValue(null as never);
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

    expect(
      await screen.findByText('等仓库元数据返回后，这里会显示可直接复制的版本安装命令。')
    ).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: '安装状态' })).toBeInTheDocument();
    expect(screen.getByText('当前还没有可用的仓库安装信息。')).toBeInTheDocument();
  });

  it('keeps the shared install panel when registry install data is unavailable', async () => {
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

    expect(await screen.findByRole('heading', { name: '安装这个版本' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: '安装状态' })).not.toBeInTheDocument();
    expect(screen.getAllByText('待补充仓库 Host').length).toBeGreaterThan(0);
    expect(screen.getByDisplayValue(expectedCommand)).toBeInTheDocument();
    expect(screen.getByText('当前未返回制品包地址。')).toBeInTheDocument();
    expect(screen.getByText('当前未返回完整性校验值。')).toBeInTheDocument();
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

    const submitButton = await screen.findByRole('button', { name: '写入发布记录' });
    expect(submitButton).toBeDisabled();

    fireEvent.click(screen.getByLabelText('我确认这个版本已经完成 npm publish，可继续写入发布记录。'));

    expect(submitButton).not.toBeDisabled();
  });

  it('shows review handoff data without workspace or bundle paths', async () => {
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
            '/skills/%40platform.skill%2Fagent-notes/versions/1.0.0?manifestDigest=sha256%3Atest&components=commands&publishRegistry=https%3A%2F%2Fnpm.internal.example.com',
          ]}
        >
          <Routes>
            <Route path="/skills/:packageName/versions/:version" element={<VersionDetailPage />} />
          </Routes>
        </MemoryRouter>
      </SessionProvider>
    );

    expect(await screen.findByText('发布交接')).toBeInTheDocument();
    expect(screen.getAllByText('清单摘要').length).toBeGreaterThan(0);
    expect(screen.getAllByText('组件清单').length).toBeGreaterThan(0);
    expect(screen.getAllByText('sha256:test').length).toBeGreaterThan(0);
    expect(screen.queryByText('工作区路径')).not.toBeInTheDocument();
    expect(screen.queryByText('打包产物路径')).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: '下载打包产物' })).not.toBeInTheDocument();
  });
});
