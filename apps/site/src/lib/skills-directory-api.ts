export interface SkillsShCapabilities {
  externalSearch: {
    enabled: boolean;
    provider: string;
  };
}

export interface ExternalSkill {
  id: string;
  source: string;
  skillId: string;
  name: string;
  installs: number;
  installsYesterday?: number;
  change?: number;
  weeklyInstalls?: number[];
  isOfficial?: boolean;
  installCommand?: string;
  learnMoreUrl?: string;
  originRegistryUrl?: string;
}

export interface ExternalSkillsPage {
  items: ExternalSkill[];
  pageInfo: {
    hasNextPage: boolean;
    nextCursor: string | null;
  };
  total?: number;
  page?: number;
}

export interface ExternalSkillDetail extends ExternalSkill {
  displayName?: string;
  description?: string;
  summaryHtml?: string;
  contentHtml?: string;
  topics?: string[];
  relatedSkills?: ExternalSkill[];
  url?: string;
}

export type SkillsShBrowseView = 'all-time' | 'trending' | 'hot';

const API_PREFIX = '/api/skills-sh';

function apiURL(path: string, params?: Record<string, string | number | undefined>): string {
  const origin = typeof window === 'undefined' ? 'http://localhost' : window.location.origin;
  const url = new URL(`${API_PREFIX}${path}`, origin);
  if (params) {
    for (const [key, value] of Object.entries(params)) {
      if (value === undefined || value === '') continue;
      url.searchParams.set(key, String(value));
    }
  }
  return typeof window === 'undefined' ? url.toString() : `${url.pathname}${url.search}`;
}

async function fetchJSON<T>(path: string, signal?: AbortSignal): Promise<T> {
  const response = await fetch(path, {
    headers: { accept: 'application/json' },
    signal,
  });
  if (!response.ok) {
    let message = `Request failed with ${response.status}`;
    try {
      const payload = (await response.json()) as { error?: { message?: string } };
      if (payload.error?.message) message = payload.error.message;
    } catch {
      // Keep the status-based message when the body is not JSON.
    }
    throw new Error(message);
  }
  return (await response.json()) as T;
}

export function fetchSkillsShCapabilities(signal?: AbortSignal): Promise<SkillsShCapabilities> {
  return fetchJSON<SkillsShCapabilities>(apiURL('/capabilities'), signal);
}

export function fetchSkillsShSearch(
  options: { query: string; pageSize?: number },
  signal?: AbortSignal
): Promise<ExternalSkillsPage> {
  return fetchJSON<ExternalSkillsPage>(
    apiURL('/search', {
      q: options.query,
      limit: options.pageSize ?? 12,
    }),
    signal
  );
}

export function fetchSkillsShBrowse(
  options: { view: SkillsShBrowseView; page?: number; pageSize?: number },
  signal?: AbortSignal
): Promise<ExternalSkillsPage> {
  return fetchJSON<ExternalSkillsPage>(
    apiURL('/browse', {
      view: options.view,
      page: options.page ?? 1,
      limit: options.pageSize ?? 12,
    }),
    signal
  );
}

export function fetchSkillsShDetail(
  source: string,
  skillId: string,
  signal?: AbortSignal
): Promise<ExternalSkillDetail> {
  return fetchJSON<ExternalSkillDetail>(
    apiURL('/skill', { source, skillId }),
    signal
  );
}

export const SKILLS_SH_BASE_URL = 'https://skills.sh';

export function skillsShSkillURL(source: string, skillId: string): string {
  return `${SKILLS_SH_BASE_URL}/${source}/${skillId}`;
}

export function skillsShSourceURL(source: string): string {
  return `${SKILLS_SH_BASE_URL}/${source}`;
}

export function skillsShSearchURL(query: string): string {
  return query ? `${SKILLS_SH_BASE_URL}/search?q=${encodeURIComponent(query)}` : SKILLS_SH_BASE_URL;
}

export function skillsShBrowseURL(view: SkillsShBrowseView): string {
  if (view === 'trending') return `${SKILLS_SH_BASE_URL}/trending`;
  if (view === 'hot') return `${SKILLS_SH_BASE_URL}/hot`;
  return SKILLS_SH_BASE_URL;
}
