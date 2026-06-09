import { cleanup, fireEvent, render, screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import * as client from '../api/client';
import { OverviewPage } from './OverviewPage';

vi.mock('../api/client', () => ({
  listAuditEvents: vi.fn(),
  listCuratedSkills: vi.fn(),
  listGovernanceChanges: vi.fn(),
  listSkills: vi.fn(),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function CatalogTarget() {
  const location = useLocation();
  return <div>{`catalog target${location.search}`}</div>;
}

describe('OverviewPage', () => {
  it('submits the hero search into the catalog query string', async () => {
    vi.mocked(client.listSkills).mockResolvedValue({ items: [], pageInfo: { hasNextPage: false, nextCursor: null } });
    vi.mocked(client.listCuratedSkills).mockResolvedValue([]);
    vi.mocked(client.listGovernanceChanges).mockResolvedValue({ items: [], pageInfo: { hasNextPage: false, nextCursor: null } });
    vi.mocked(client.listAuditEvents).mockResolvedValue({ items: [], pageInfo: { hasNextPage: false, nextCursor: null } });

    render(
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route path="/" element={<OverviewPage />} />
          <Route path="/catalog" element={<CatalogTarget />} />
        </Routes>
      </MemoryRouter>
    );

    fireEvent.change(await screen.findByLabelText('搜索技能'), { target: { value: 'review' } });
    fireEvent.click(screen.getByRole('button', { name: '搜索目录' }));

    expect(await screen.findByText('catalog target?q=review')).toBeInTheDocument();
  });

  it('renders recommended and recently updated discovery sections with curated usefulness copy first', async () => {
    vi.mocked(client.listSkills).mockResolvedValue({
      items: [
        {
          packageName: '@platform.skill/legacy-audit',
          displayName: '旧版审计',
          description: '用于回看更早的注册表活动。',
          sourceType: 'native',
          status: 'active',
          latestVersion: '1.0.0',
          latestVersionStatus: 'published',
          updatedAt: '2026-06-01T10:00:00Z',
        },
        {
          packageName: '@platform.skill/release-audit',
          displayName: '发布审计',
          description: '追踪谁改动了注册表包。',
          sourceType: 'imported',
          status: 'active',
          latestVersion: '1.3.0',
          latestVersionStatus: 'published',
          updatedAt: '2026-06-08T10:00:00Z',
        },
        {
          packageName: '@platform.skill/queue-review',
          displayName: '队列评审',
          description: '帮助整理待评审的技能变更。',
          sourceType: 'native',
          status: 'retired',
          latestVersion: '0.8.0',
          latestVersionStatus: 'deprecated',
          updatedAt: '2026-06-05T10:00:00Z',
        },
        {
          packageName: '@platform.skill/dependency-map',
          displayName: '依赖地图',
          description: '查看上游包之间的关系。',
          sourceType: 'imported',
          status: 'active',
          latestVersion: '0.9.1',
          latestVersionStatus: 'deprecated',
          updatedAt: '2026-06-07T10:00:00Z',
        },
      ],
      pageInfo: { hasNextPage: false, nextCursor: null },
    });
    vi.mocked(client.listCuratedSkills).mockResolvedValue([
      {
        packageName: '@platform.skill/review-helper',
        displayName: '评审助手',
        description: '为评审者汇总 PR 级别的技能变更。',
        sourceType: 'native',
        status: 'active',
        latestVersion: '2.4.0',
        latestVersionStatus: 'published',
        updatedAt: '2026-06-09T09:00:00Z',
        curationReason: '适合按能力优先进行代码评审分流。',
      },
    ]);
    vi.mocked(client.listGovernanceChanges).mockResolvedValue({ items: [], pageInfo: { hasNextPage: false, nextCursor: null } });
    vi.mocked(client.listAuditEvents).mockResolvedValue({ items: [], pageInfo: { hasNextPage: false, nextCursor: null } });

    render(
      <MemoryRouter initialEntries={['/']}>
        <OverviewPage />
      </MemoryRouter>
    );

    const recommendedHeading = await screen.findByRole('heading', { name: '推荐技能' });
    const recommendedSection = recommendedHeading.closest('section');
    const recentHeading = screen.getByRole('heading', { name: '最近更新' });
    const recentSection = recentHeading.closest('section');

    expect(recommendedSection).not.toBeNull();
    expect(recentSection).not.toBeNull();

    if (!recommendedSection || !recentSection) {
      throw new Error('Expected overview sections to render');
    }

    expect(await within(recommendedSection).findByText('评审助手')).toBeInTheDocument();
    expect(within(recommendedSection).getByText('适合按能力优先进行代码评审分流。')).toBeInTheDocument();
    expect(await within(recentSection).findByText('发布审计')).toBeInTheDocument();

    const recommendedCard = within(recommendedSection).getByText('评审助手').closest('.skill-summary-card');
    expect(recommendedCard).not.toBeNull();

    if (!recommendedCard) {
      throw new Error('Expected recommended card to render');
    }

    const reason = recommendedCard.querySelector('.skill-summary-card__reason');
    const install = recommendedCard.querySelector('.skill-summary-card__install');
    expect(reason).not.toBeNull();
    expect(install).not.toBeNull();

    if (!reason || !install) {
      throw new Error('Expected recommended card structure');
    }

    expect(reason.compareDocumentPosition(install) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();

    const recentCards = Array.from(recentSection.querySelectorAll('.skill-summary-card strong')).map((node) => node.textContent);
    expect(recentCards).toEqual(['发布审计', '依赖地图', '队列评审']);
  });
});
