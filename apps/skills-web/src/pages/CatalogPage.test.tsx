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

  it('renders discovery cards with description before install metadata and provenance before the action', async () => {
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

    const card = (await screen.findByText('Release Audit')).closest('.skill-summary-card');
    expect(card).not.toBeNull();

    if (!card) {
      throw new Error('Expected a catalog result card');
    }

    const cardText = card.textContent ?? '';
    expect(cardText.indexOf('Trace who changed a registry package.')).toBeLessThan(cardText.indexOf('published'));
    expect(cardText.indexOf('published')).toBeLessThan(cardText.indexOf('updated'));
    expect(cardText.indexOf('updated')).toBeLessThan(cardText.indexOf('Open skill'));
  });
});
