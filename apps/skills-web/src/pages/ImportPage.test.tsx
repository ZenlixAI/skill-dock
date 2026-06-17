import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import * as client from '../api/client';
import * as externalImport from '../features/packaging/externalImport';
import { RegistryHttpError } from '../api/httpClient';
import { SessionProvider } from '../session/SessionProvider';
import { ImportPage } from './ImportPage';

vi.mock('../api/client', () => ({
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
    allowedActions: ['session.read', 'packaging.publish'],
    allowedPackages: [],
  }),
  importReviewPackaging: vi.fn(),
  publishSkill: vi.fn(),
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
    allowedActions: ['session.read', 'packaging.publish'],
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
  it('starts with streamlined review and publish actions without raw protocol controls', async () => {
    renderImportPage();

    expect(await screen.findByDisplayValue('https://github.com/mattpocock/skills?path=skills/engineering/zoom-out')).toBeInTheDocument();
    await waitFor(() => expect(client.getSession).toHaveBeenCalledTimes(1));
    expect(screen.getByRole('button', { name: '检查并生成发布预览' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '发布 skill' })).toBeDisabled();
    expect(screen.queryByRole('button', { name: 'Prepare' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Validate' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Pack' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '清理 workspace' })).not.toBeInTheDocument();
  });

  it('imports a GitHub skill archive and publishes it with the reviewed summary', async () => {
    vi.mocked(client.importReviewPackaging).mockResolvedValue({
      ok: true,
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      manifestDigest: 'sha256:archive',
      components: ['commands'],
      warnings: [],
      warningDetails: [],
      errors: [],
    });
    vi.mocked(client.publishSkill).mockResolvedValue({
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      manifestDigest: 'sha256:archive',
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

    fireEvent.change(await screen.findByDisplayValue('https://github.com/mattpocock/skills?path=skills/engineering/zoom-out'), {
      target: { value: 'https://github.com/example/skills?path=skills/engineering/zoom-out' },
    });
    fireEvent.change(screen.getByDisplayValue('@company.skill/zoom-out'), {
      target: { value: '@company.skill/zoom-out' },
    });
    fireEvent.change(screen.getByDisplayValue('1.2.0'), {
      target: { value: '1.2.0' },
    });

    fireEvent.click(screen.getByRole('button', { name: '检查并生成发布预览' }));

    await waitFor(() =>
      expect(externalImport.buildExternalImportArchive).toHaveBeenCalledWith(
        'https://github.com/example/skills?path=skills/engineering/zoom-out',
        undefined
      )
    );
    expect(client.importReviewPackaging).toHaveBeenCalledWith(
      expect.objectContaining({
        source: {
          type: 'upload_archive',
          archiveBase64: 'archive-base64',
        },
        target: {
          packageName: '@company.skill/zoom-out',
          version: '1.2.0',
        },
      })
    );

    fireEvent.click(screen.getByRole('button', { name: '发布 skill' }));

    await waitFor(() =>
      expect(client.publishSkill).toHaveBeenCalledWith(
        expect.objectContaining({
          source: {
            type: 'upload_archive',
            archiveBase64: 'archive-base64',
          },
          review: {
            manifestDigest: 'sha256:archive',
            components: ['commands'],
          },
        })
      )
    );
    expect((await screen.findAllByText('skill 已完成一次性发布，并写入 release record。')).length).toBeGreaterThan(0);
  });

  it('runs import-review for platform_authoring and publishes with review summary', async () => {
    vi.mocked(client.importReviewPackaging).mockResolvedValue({
      ok: true,
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      manifestDigest: 'sha256:test',
      components: ['workflows'],
      warnings: [],
      warningDetails: [],
      errors: [],
    });
    vi.mocked(client.publishSkill).mockResolvedValue({
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      manifestDigest: 'sha256:test',
      components: ['workflows'],
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

    expect(await screen.findByText('发布预览已生成，可以继续执行一次性发布。')).toBeInTheDocument();
    expect(client.importReviewPackaging).toHaveBeenCalledWith(
      expect.objectContaining({
        source: {
          type: 'platform_authoring',
          files: {
            'workflows/main.md': 'step 1\nstep 2\n',
          },
        },
      })
    );

    fireEvent.click(screen.getByRole('button', { name: '发布 skill' }));

    await waitFor(() =>
      expect(client.publishSkill).toHaveBeenCalledWith(
        expect.objectContaining({
          review: {
            manifestDigest: 'sha256:test',
            components: ['workflows'],
          },
        })
      )
    );
  });

  it('applies advanced raw json and uses it in the import-review payload', async () => {
    vi.mocked(client.importReviewPackaging).mockResolvedValue({
      ok: true,
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      manifestDigest: 'sha256:test',
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
    fireEvent.click(screen.getByRole('button', { name: '检查并生成发布预览' }));

    expect(await screen.findByText('发布预览已生成，可以继续执行一次性发布。')).toBeInTheDocument();
    expect(client.importReviewPackaging).toHaveBeenCalledWith(
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

  it('does not call importReviewPackaging when external import acquisition fails', async () => {
    vi.mocked(externalImport.buildExternalImportArchive).mockRejectedValue(new Error('archive failed'));

    renderImportPage();
    fireEvent.click(await screen.findByRole('button', { name: '检查并生成发布预览' }));

    expect(await screen.findByText('archive failed')).toBeInTheDocument();
    expect(client.importReviewPackaging).not.toHaveBeenCalled();
  });

  it('surfaces publish permission failures and keeps review metadata on screen', async () => {
    vi.mocked(client.importReviewPackaging).mockResolvedValue({
      ok: true,
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      manifestDigest: 'sha256:test',
      components: ['commands'],
      warnings: [],
      warningDetails: [],
      errors: [],
    });
    vi.mocked(client.publishSkill).mockRejectedValue(
      new RegistryHttpError('forbidden', 403, { error: { requestId: 'req_123' } })
    );

    renderImportPage();
    fireEvent.click(await screen.findByRole('button', { name: '检查并生成发布预览' }));
    expect((await screen.findAllByText('sha256:test')).length).toBeGreaterThan(0);

    fireEvent.click(screen.getByRole('button', { name: '发布 skill' }));

    expect((await screen.findAllByText('Current session is not allowed to perform this action.')).length).toBeGreaterThan(0);
    expect(screen.getAllByText('sha256:test').length).toBeGreaterThan(0);
  });

  it('surfaces stale workspace-style 404 publish errors with an actionable hint', async () => {
    vi.mocked(client.importReviewPackaging).mockResolvedValue({
      ok: true,
      packageName: '@company.skill/zoom-out',
      version: '1.2.0',
      manifestDigest: 'sha256:test',
      components: ['commands'],
      warnings: [],
      warningDetails: [],
      errors: [],
    });
    vi.mocked(client.publishSkill).mockRejectedValue(
      new RegistryHttpError('Bundle archive does not exist for the requested workspace.', 404, {
        error: {
          code: 'bundle_not_found',
          requestId: 'req_404',
          details: {
            workspaceHandle: 'ws_123',
          },
        },
      })
    );

    renderImportPage();
    fireEvent.click(await screen.findByRole('button', { name: '检查并生成发布预览' }));
    expect((await screen.findAllByText('sha256:test')).length).toBeGreaterThan(0);

    fireEvent.click(screen.getByRole('button', { name: '发布 skill' }));

    expect(
      (
        await screen.findAllByText(
          'Catalog returned a stale workspace or bundle reference. One-shot packaging should be stateless; retry after upgrading Catalog or removing any legacy workspace-based handoff.'
        )
      ).length
    ).toBeGreaterThan(0);
    expect(screen.getAllByText('sha256:test').length).toBeGreaterThan(0);
  });
});
