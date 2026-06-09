import { cleanup, fireEvent, render, screen } from '@testing-library/react';
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

    fireEvent.change(await screen.findByLabelText('Search skills'), { target: { value: 'review' } });
    fireEvent.click(screen.getByRole('button', { name: 'Search catalog' }));

    expect(await screen.findByText('catalog target?q=review')).toBeInTheDocument();
  });
});
