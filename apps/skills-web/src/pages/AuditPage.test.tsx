import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import * as client from '../api/client';
import { AuditPage } from './AuditPage';

vi.mock('../api/client', () => ({
  listAuditEvents: vi.fn(),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('AuditPage', () => {
  it('在没有审计记录时，先展示摘要区，再给出面向筛选的空状态指引', async () => {
    vi.mocked(client.listAuditEvents).mockResolvedValue({
      items: [],
      pageInfo: { hasNextPage: false, nextCursor: null },
    });

    render(
      <MemoryRouter>
        <AuditPage />
      </MemoryRouter>
    );

    expect(await screen.findByRole('heading', { name: '审计追踪' })).toBeInTheDocument();
    expect(screen.getByText('拒绝请求')).toBeInTheDocument();
    expect(screen.getByText('当前视图下没有匹配的审计事件。')).toBeInTheDocument();
    expect(screen.getByText('请调整操作者、包名或结果筛选条件，以扩大追溯范围。')).toBeInTheDocument();
  });
});
