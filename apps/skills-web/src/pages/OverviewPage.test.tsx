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
          packageName: '@platform.skill/release-audit',
          displayName: 'Release Audit',
          description: 'Trace who changed a registry package.',
          sourceType: 'native',
          status: 'active',
          latestVersion: '1.3.0',
          latestVersionStatus: 'published',
          updatedAt: '2026-06-08T10:00:00Z',
        },
        {
          packageName: '@platform.skill/dependency-map',
          displayName: 'Dependency Map',
          description: 'Inspect upstream package relationships.',
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
        displayName: 'Review Helper',
        description: 'Summarize PR-level skill changes for reviewers.',
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

    expect(await within(recommendedSection).findByText('Review Helper')).toBeInTheDocument();
    expect(within(recommendedSection).getByText('适合按能力优先进行代码评审分流。')).toBeInTheDocument();
    expect(await within(recentSection).findByText('Release Audit')).toBeInTheDocument();

    const recommendedCard = within(recommendedSection).getByText('Review Helper').closest('.skill-summary-card');
    expect(recommendedCard).not.toBeNull();

    if (!recommendedCard) {
      throw new Error('Expected recommended card to render');
    }

    const cardText = recommendedCard.textContent ?? '';
    expect(cardText.indexOf('适合按能力优先进行代码评审分流。')).toBeLessThan(cardText.indexOf('最新 2.4.0'));
  });
});
