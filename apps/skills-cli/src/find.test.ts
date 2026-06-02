import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { runCliOutput, stripLogo } from './test-utils.ts';

vi.mock('./catalog.ts', async () => {
  const actual = await vi.importActual<typeof import('./catalog.ts')>('./catalog.ts');
  return {
    ...actual,
    searchCatalogSkills: vi.fn(),
    searchCatalogSkillsPage: vi.fn(),
  };
});

import { CatalogError } from './catalog.ts';
import { parseFindOptions, runFind, searchSkillsAPI } from './find.ts';
import { searchCatalogSkills, searchCatalogSkillsPage } from './catalog.ts';

describe('parseFindOptions', () => {
  it('parses a plain query', () => {
    expect(parseFindOptions(['pdf', 'workflow'])).toEqual({
      query: 'pdf workflow',
      registry: undefined,
      json: false,
    });
  });

  it('parses --registry alongside query text', () => {
    expect(parseFindOptions(['pdf', '--registry', 'https://catalog.example.com', 'workflow'])).toEqual({
      query: 'pdf workflow',
      registry: 'https://catalog.example.com',
      json: false,
    });
  });

  it('parses --json alongside query text', () => {
    expect(parseFindOptions(['pdf', '--json'])).toEqual({
      query: 'pdf',
      registry: undefined,
      json: true,
    });
  });
});

describe('find command error handling', () => {
  let logSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    vi.mocked(searchCatalogSkills).mockReset();
    vi.mocked(searchCatalogSkillsPage).mockReset();
  });

  afterEach(() => {
    logSpy.mockRestore();
  });

  it('does not swallow catalog search failures', async () => {
    vi.mocked(searchCatalogSkillsPage).mockRejectedValueOnce(
      new CatalogError({
        code: 'unauthorized',
        message: 'Authentication is required.',
        status: 401,
      })
    );

    await expect(searchSkillsAPI('pdf', 'https://catalog.example.com')).rejects.toMatchObject({
      code: 'unauthorized',
    });
  });

  it('follows cursor pagination when searching skills', async () => {
    vi.mocked(searchCatalogSkillsPage)
      .mockResolvedValueOnce({
        items: [
          {
            packageName: '@platform.skill/pdf-skill',
            displayName: 'PDF Skill',
            description: 'PDF processing workflow',
            latestVersion: '1.2.0',
          },
        ],
        pageInfo: {
          nextCursor: 'cursor_1',
          hasNextPage: true,
        },
      })
      .mockResolvedValueOnce({
        items: [
          {
            packageName: '@platform.skill/code-review',
            displayName: 'Code Review Accelerator',
            description: 'Structured code review prompts and checklists',
            latestVersion: '0.9.0',
          },
        ],
        pageInfo: {
          nextCursor: null,
          hasNextPage: false,
        },
      });

    await expect(searchSkillsAPI('pdf', 'https://catalog.example.com')).resolves.toEqual([
      {
        packageName: '@platform.skill/pdf-skill',
        displayName: 'PDF Skill',
        description: 'PDF processing workflow',
        latestVersion: '1.2.0',
      },
      {
        packageName: '@platform.skill/code-review',
        displayName: 'Code Review Accelerator',
        description: 'Structured code review prompts and checklists',
        latestVersion: '0.9.0',
      },
    ]);

    expect(searchCatalogSkillsPage).toHaveBeenNthCalledWith(1, 'pdf', {
      registry: 'https://catalog.example.com',
      pageSize: 10,
      cursor: undefined,
    });
    expect(searchCatalogSkillsPage).toHaveBeenNthCalledWith(2, 'pdf', {
      registry: 'https://catalog.example.com',
      pageSize: 10,
      cursor: 'cursor_1',
    });
  });

  it('prints catalog failures instead of pretending no results exist', async () => {
    vi.mocked(searchCatalogSkillsPage).mockRejectedValueOnce(
      new CatalogError({
        code: 'unauthorized',
        message: 'Authentication is required.',
        status: 401,
      })
    );

    await runFind(['pdf']);

    expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('Find failed'));
    expect(logSpy).not.toHaveBeenCalledWith(expect.stringContaining('No skills found'));
  });

  it('prints a complete empty-result sentence for query mode', async () => {
    vi.mocked(searchCatalogSkillsPage).mockResolvedValueOnce({
      items: [],
      pageInfo: {
        nextCursor: null,
        hasNextPage: false,
      },
    });

    await runFind(['pdf']);

    expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('No skills found for "pdf".'));
  });

  it('prints a truncation hint when query results exceed the display limit', async () => {
    vi.mocked(searchCatalogSkillsPage).mockResolvedValueOnce({
      items: Array.from({ length: 7 }, (_, index) => ({
        packageName: `@platform.skill/skill-${index + 1}`,
        displayName: `Skill ${index + 1}`,
        description: `Description ${index + 1}`,
        latestVersion: '1.0.0',
      })),
      pageInfo: {
        nextCursor: null,
        hasNextPage: false,
      },
    });

    await runFind(['pdf']);

    expect(logSpy).toHaveBeenCalledWith(
      expect.stringContaining('Showing first 6 of 7 results')
    );
  });

  it('prints a truncation hint when additional catalog pages still exist after the fetch limit', async () => {
    vi.mocked(searchCatalogSkillsPage)
      .mockResolvedValueOnce({
        items: Array.from({ length: 10 }, (_, index) => ({
          packageName: `@platform.skill/page-1-skill-${index + 1}`,
          displayName: `Page 1 Skill ${index + 1}`,
          description: `Description ${index + 1}`,
          latestVersion: '1.0.0',
        })),
        pageInfo: {
          nextCursor: 'cursor_1',
          hasNextPage: true,
        },
      })
      .mockResolvedValueOnce({
        items: Array.from({ length: 10 }, (_, index) => ({
          packageName: `@platform.skill/page-2-skill-${index + 1}`,
          displayName: `Page 2 Skill ${index + 1}`,
          description: `Description ${index + 1}`,
          latestVersion: '1.0.0',
        })),
        pageInfo: {
          nextCursor: 'cursor_2',
          hasNextPage: true,
        },
      })
      .mockResolvedValueOnce({
        items: Array.from({ length: 10 }, (_, index) => ({
          packageName: `@platform.skill/page-3-skill-${index + 1}`,
          displayName: `Page 3 Skill ${index + 1}`,
          description: `Description ${index + 1}`,
          latestVersion: '1.0.0',
        })),
        pageInfo: {
          nextCursor: 'cursor_3',
          hasNextPage: true,
        },
      });

    await runFind(['pdf']);

    expect(logSpy).toHaveBeenCalledWith(
      expect.stringContaining('Showing first 6 of at least 30 results')
    );
  });

  it('prints machine-readable JSON for query mode', async () => {
    vi.mocked(searchCatalogSkillsPage).mockResolvedValueOnce({
      items: [
        {
          packageName: '@platform.skill/pdf-skill',
          displayName: 'PDF Skill',
          description: 'PDF processing workflow',
          latestVersion: '1.2.0',
        },
      ],
      pageInfo: {
        nextCursor: null,
        hasNextPage: false,
      },
    });

    await runFind(['pdf', '--json']);

    expect(logSpy).toHaveBeenCalledWith(`{
  "items": [
    {
      "packageName": "@platform.skill/pdf-skill",
      "displayName": "PDF Skill",
      "description": "PDF processing workflow",
      "latestVersion": "1.2.0"
    }
  ],
  "total": 1,
  "truncated": false
}`);
  });

  it('prints machine-readable JSON errors in query mode', async () => {
    vi.mocked(searchCatalogSkillsPage).mockRejectedValueOnce(
      new CatalogError({
        code: 'unauthorized',
        message: 'Authentication is required.',
        status: 401,
      })
    );

    await runFind(['pdf', '--json']);

    expect(logSpy).toHaveBeenCalledWith(`{
  "error": {
    "message": "Authentication is required."
  }
}`);
  });

  it('shows dedicated find help output', () => {
    const output = stripLogo(runCliOutput(['find', '--help']));
    expect(output).toContain('Usage: skills find [query] [options]');
    expect(output).toContain('Search for skills interactively or by keyword');
    expect(output).toContain('--json');
    expect(output).toContain('skills find typescript');
  });
});
