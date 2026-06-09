import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import * as client from '../api/client';
import { WorkbenchHomePage } from './WorkbenchHomePage';

vi.mock('../api/client', () => ({
  listAuditEvents: vi.fn(),
  listGovernanceChanges: vi.fn(),
  listSkills: vi.fn(),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('WorkbenchHomePage', () => {
  it('将工作台重构为控制台首页，并展示按动作分组的入口卡片', async () => {
    vi.mocked(client.listSkills).mockResolvedValue({ items: [], pageInfo: { hasNextPage: false, nextCursor: null } });
    vi.mocked(client.listGovernanceChanges).mockResolvedValue({ items: [], pageInfo: { hasNextPage: false, nextCursor: null } });
    vi.mocked(client.listAuditEvents).mockResolvedValue({ items: [], pageInfo: { hasNextPage: false, nextCursor: null } });

    render(
      <MemoryRouter>
        <WorkbenchHomePage />
      </MemoryRouter>
    );

    expect(await screen.findByRole('heading', { name: '控制台首页' })).toBeInTheDocument();
    expect(screen.getByText('导入与打包')).toBeInTheDocument();
    expect(screen.getByText('治理与审计')).toBeInTheDocument();
  });
});
