import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { SkillsDirectoryPage } from './SkillsDirectoryPage';

const browsePage = {
  items: [
    {
      id: 'heygen-com/hyperframes/music-to-video',
      source: 'heygen-com/hyperframes',
      skillId: 'music-to-video',
      name: 'music-to-video',
      installs: 419296,
      isOfficial: false,
      weeklyInstalls: [43707, 21520, 32432],
    },
  ],
  pageInfo: { hasNextPage: true, nextCursor: null },
  total: 9902,
  page: 1,
};

const detail = {
  ...browsePage.items[0],
  displayName: 'Music to Video',
  description: 'Turn a track into a beat-synced video.',
  summaryHtml: '<p><strong>Summary text</strong></p>',
  contentHtml: '<p>Content text</p>',
  topics: ['Video', 'Audio'],
  installCommand: 'skills add heygen-com/hyperframes --skill music-to-video',
  relatedSkills: [],
  url: 'https://skills.sh/heygen-com/hyperframes/music-to-video',
};

function jsonResponse(data: unknown): Response {
  return {
    ok: true,
    status: 200,
    json: async () => data,
  } as Response;
}

function installFetchMock(options: { enabled?: boolean } = {}) {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const url = String(input);
    if (url.includes('/api/skills-sh/capabilities')) {
      return jsonResponse({
        externalSearch: { enabled: options.enabled ?? true, provider: 'skills.sh' },
      });
    }
    if (url.includes('/api/skills-sh/skill?')) return jsonResponse(detail);
    if (url.includes('/api/skills-sh/browse')) return jsonResponse(browsePage);
    if (url.includes('/api/skills-sh/search')) return jsonResponse(browsePage);
    throw new Error(`Unexpected request: ${url}`);
  });
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

beforeEach(() => {
  installFetchMock();
});

describe('SkillsDirectoryPage', () => {
  it('renders the skills.sh directory with browse views', async () => {
    render(
      <MemoryRouter initialEntries={['/registry']}>
        <SkillsDirectoryPage />
      </MemoryRouter>
    );

    expect(await screen.findByRole('heading', { name: 'Skills Directory' })).toBeInTheDocument();
    expect(await screen.findByText('music-to-video')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'All-time' })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByRole('button', { name: 'Trending' })).toHaveAttribute('aria-pressed', 'false');
    expect(screen.getByRole('button', { name: 'Hot' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /View all on skills\.sh/i })).toBeInTheDocument();
  });

  it('switches the browse view', async () => {
    const fetchMock = installFetchMock();
    render(
      <MemoryRouter initialEntries={['/registry']}>
        <SkillsDirectoryPage />
      </MemoryRouter>
    );

    expect(await screen.findByText('music-to-video')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Trending' }));

    await waitFor(() => {
      const urls = fetchMock.mock.calls.map(([input]) => String(input));
      expect(urls.some((url) => url.includes('/api/skills-sh/browse') && url.includes('view=trending'))).toBe(true);
    });
  });

  it('searches skills.sh and hides the browse view tabs', async () => {
    const fetchMock = installFetchMock();
    render(
      <MemoryRouter initialEntries={['/registry']}>
        <SkillsDirectoryPage />
      </MemoryRouter>
    );

    expect(await screen.findByText('music-to-video')).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText('Search skills'), { target: { value: 'react' } });

    await waitFor(
      () => {
        const urls = fetchMock.mock.calls.map(([input]) => String(input));
        expect(urls.some((url) => url.includes('/api/skills-sh/search') && url.includes('q=react'))).toBe(true);
      },
      { timeout: 2000 }
    );
    await waitFor(() => {
      expect(screen.queryByRole('button', { name: 'Trending' })).not.toBeInTheDocument();
    });
  });

  it('opens the detail drawer', async () => {
    render(
      <MemoryRouter initialEntries={['/registry']}>
        <SkillsDirectoryPage />
      </MemoryRouter>
    );

    expect(await screen.findByText('music-to-video')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'View details' }));

    expect(await screen.findByRole('dialog')).toBeInTheDocument();
    expect(await screen.findByText('Music to Video')).toBeInTheDocument();
    expect(screen.getByText('Summary text')).toBeInTheDocument();
    expect(screen.getByText('Content text')).toBeInTheDocument();
    expect(screen.getByText('skills add heygen-com/hyperframes --skill music-to-video')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Close skill details' }));
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
  });

  it('shows a disabled state without requesting skills.sh data', async () => {
    const fetchMock = installFetchMock({ enabled: false });
    render(
      <MemoryRouter initialEntries={['/registry']}>
        <SkillsDirectoryPage />
      </MemoryRouter>
    );

    expect(await screen.findByRole('heading', { name: 'Skills directory is not enabled' })).toBeInTheDocument();
    const urls = fetchMock.mock.calls.map(([input]) => String(input));
    expect(urls.some((url) => url.includes('/api/skills-sh/browse'))).toBe(false);
    expect(urls.some((url) => url.includes('/api/skills-sh/search'))).toBe(false);
  });
});
