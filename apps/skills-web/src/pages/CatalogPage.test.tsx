import { cleanup, render, screen } from '@testing-library/react';
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
    expect(screen.getByText('Try a broader capability, package scope, or source filter.')).toBeInTheDocument();
  });
});
