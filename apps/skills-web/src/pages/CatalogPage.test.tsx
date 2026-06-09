import { cleanup, render, screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import * as client from '../api/client';
import { CatalogPage } from './CatalogPage';

vi.mock('../api/client', () => ({
  listSkills: vi.fn(),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('CatalogPage', () => {
  it('hydrates the search query from the URL and shows discovery-first empty guidance', async () => {
    vi.mocked(client.listSkills).mockResolvedValue({
      items: [],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });

    render(
      <MemoryRouter initialEntries={['/catalog?q=pdf']}>
        <CatalogPage />
      </MemoryRouter>
    );

    expect(await screen.findByDisplayValue('pdf')).toBeInTheDocument();
    expect(screen.getByText('请尝试更宽泛的能力词、包范围或来源筛选。')).toBeInTheDocument();
  });

  it('renders discovery cards with description before install metadata and provenance before the action', async () => {
    vi.mocked(client.listSkills).mockResolvedValue({
      items: [
        {
          packageName: '@platform.skill/release-audit',
          displayName: '发布审计',
          description: '追踪谁改动了注册表包。',
          sourceType: 'native',
          status: 'retired',
          latestVersion: '1.3.0',
          latestVersionStatus: 'published',
          updatedAt: '2026-06-08T10:00:00Z',
        },
      ],
      pageInfo: {
        hasNextPage: false,
        nextCursor: null,
      },
    });

    render(
      <MemoryRouter initialEntries={['/catalog?q=audit']}>
        <CatalogPage />
      </MemoryRouter>
    );

    const card = (await screen.findByText('发布审计')).closest('.skill-summary-card');
    expect(card).not.toBeNull();

    if (!card) {
      throw new Error('Expected a catalog result card');
    }

    const description = card.querySelector<HTMLElement>('.skill-summary-card__description');
    const install = card.querySelector<HTMLElement>('.skill-summary-card__install');
    const footer = card.querySelector<HTMLElement>('.skill-summary-card__footer');
    const actions = card.querySelector<HTMLElement>('.skill-summary-card__actions');

    expect(description).not.toBeNull();
    expect(install).not.toBeNull();
    expect(footer).not.toBeNull();
    expect(actions).not.toBeNull();

    if (!description || !install || !footer || !actions) {
      throw new Error('Expected discovery card structure');
    }

    expect(within(install).getByText('退役')).toBeInTheDocument();
    expect(within(install).getByText('已发布')).toBeInTheDocument();
    expect(within(footer).getByText(/更新于/)).toBeInTheDocument();
    expect(within(actions).getByRole('link', { name: '查看技能' })).toBeInTheDocument();
    expect(description.compareDocumentPosition(install) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(install.compareDocumentPosition(footer) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });
});
