import { HTMLElement, Node, NodeType, parse } from 'node-html-parser';

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

export interface SkillsShEnv {
  SKILLS_SH_ENABLED?: string;
  SKILLS_SH_BASE_URL?: string;
}

export interface SkillsShExecutionContext {
  waitUntil(promise: Promise<unknown>): void;
}

const DEFAULT_BASE_URL = 'https://skills.sh';
const SEARCH_MIN_CHARS = 2;
const MAX_SEARCH_LIMIT = 100;
const SOURCE_PATTERN = /^[A-Za-z0-9][A-Za-z0-9_.-]*\/[A-Za-z0-9][A-Za-z0-9_.-]*$/;
const SKILL_ID_PATTERN = /^[A-Za-z0-9][A-Za-z0-9_.:-]*$/;
const COMMAND_NAME_PATTERN = /^[A-Za-z0-9][A-Za-z0-9_.:@-]*$/;
const ALLOWED_TAGS = new Set([
  'a', 'p', 'br', 'hr', 'strong', 'b', 'em', 'i', 'code', 'pre', 'ul', 'ol', 'li', 'blockquote',
  'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'table', 'thead', 'tbody', 'tr', 'th', 'td', 'span', 'div',
  'del', 'sup', 'sub',
]);
const VOID_TAGS = new Set(['br', 'hr']);
const BLOCKED_TAGS = new Set(['script', 'style', 'iframe', 'object', 'embed', 'form', 'input', 'button']);

interface SearchEntry {
  id?: string;
  source?: string;
  skillId?: string;
  name?: string;
  installs?: number;
}

interface BrowseEntry extends SearchEntry {
  installsYesterday?: number;
  change?: number;
  weeklyInstalls?: number[];
  isOfficial?: boolean;
}

interface SkillsShSearchPayload {
  skills?: SearchEntry[];
}

interface SkillsShBrowsePayload {
  skills?: BrowseEntry[];
  total?: number;
  hasMore?: boolean;
  page?: number;
}

export async function handleSkillsShRequest(
  request: Request,
  env: SkillsShEnv,
  ctx?: SkillsShExecutionContext
): Promise<Response> {
  const url = new URL(request.url);
  const path = url.pathname.replace(/\/+$/, '') || '/';
  const enabled = skillsShEnabled(env);
  const baseURL = normalizeBaseURL(env.SKILLS_SH_BASE_URL);

  if (request.method === 'OPTIONS') {
    return new Response(null, { status: 204, headers: corsHeaders() });
  }

  if (path === '/api/skills-sh/capabilities') {
    return jsonResponse(
      { externalSearch: { enabled, provider: 'skills.sh' } },
      { cacheSeconds: 0, status: 200 }
    );
  }

  if (request.method !== 'GET') {
    return errorResponse(405, 'method_not_allowed', 'Only GET is supported.');
  }

  if (!enabled) {
    return errorResponse(404, 'external_search_disabled', 'The skills directory is not enabled for this deployment.');
  }

  try {
    if (path === '/api/skills-sh/search') {
      const query = (url.searchParams.get('q') || '').trim();
      if (query.length < SEARCH_MIN_CHARS) {
        return errorResponse(400, 'invalid_request', 'q must contain at least 2 characters.');
      }
      const limit = positiveInt(url.searchParams.get('limit'), 20, MAX_SEARCH_LIMIT);
      return await cachedJSON(request, 30, ctx, async () => {
        const payload = await fetchJSON<SkillsShSearchPayload>(
          `${baseURL}/api/search?q=${encodeURIComponent(query)}&limit=${limit}`
        );
        return {
          items: (payload.skills || []).map((entry) => summaryFromSearch(entry, baseURL)),
          pageInfo: { hasNextPage: false, nextCursor: null },
        } satisfies ExternalSkillsPage;
      });
    }

    if (path === '/api/skills-sh/browse') {
      const view = normalizeView(url.searchParams.get('view'));
      const pageNumber = positiveInt(url.searchParams.get('page'), 1, 1000);
      const limit = positiveInt(url.searchParams.get('limit'), 12, MAX_SEARCH_LIMIT);
      return await cachedJSON(request, 30, ctx, async () => {
        const payload = await fetchJSON<SkillsShBrowsePayload>(
          `${baseURL}/api/skills/${encodeURIComponent(view)}/${pageNumber}`
        );
        let items = (payload.skills || []).map((entry) => summaryFromBrowse(entry, baseURL));
        const hasMore = Boolean(payload.hasMore) || items.length > limit;
        items = items.slice(0, limit);
        return {
          items,
          pageInfo: { hasNextPage: hasMore, nextCursor: null },
          total: payload.total ?? 0,
          page: payload.page || pageNumber,
        } satisfies ExternalSkillsPage;
      });
    }

    if (path === '/api/skills-sh/skill') {
      const source = (url.searchParams.get('source') || '').trim();
      const skillId = (url.searchParams.get('skillId') || '').trim();
      if (!SOURCE_PATTERN.test(source) || !SKILL_ID_PATTERN.test(skillId)) {
        return errorResponse(400, 'invalid_request', 'source and skillId must identify a skill.');
      }
      return await cachedJSON(request, 600, ctx, () => fetchSkillsShDetail(baseURL, source, skillId));
    }
  } catch (error) {
    console.error('skills.sh proxy failed', error);
    return errorResponse(502, 'external_search_unavailable', 'The skills directory is temporarily unavailable.');
  }

  return errorResponse(404, 'not_found', 'Unknown skills directory route.');
}

export function skillsShEnabled(env: SkillsShEnv): boolean {
  const value = (env.SKILLS_SH_ENABLED || '').trim().toLowerCase();
  return !['0', 'false', 'no', 'off'].includes(value);
}

function normalizeBaseURL(raw: string | undefined): string {
  return (raw || DEFAULT_BASE_URL).trim().replace(/\/+$/, '') || DEFAULT_BASE_URL;
}

function normalizeView(raw: string | null): string {
  switch ((raw || '').trim().toLowerCase()) {
    case 'trending':
      return 'trending';
    case 'hot':
      return 'hot';
    default:
      return 'all-time';
  }
}

function positiveInt(raw: string | null, fallback: number, maximum: number): number {
  const value = Number.parseInt(raw || '', 10);
  if (!Number.isFinite(value) || value <= 0) return fallback;
  return Math.min(value, maximum);
}

function summaryFromSearch(entry: SearchEntry, baseURL: string): ExternalSkill {
  const source = (entry.source || '').trim();
  const skillId = (entry.skillId || '').trim();
  const id = (entry.id || (source && skillId ? `${source}/${skillId}` : '')).trim();
  return {
    id,
    source,
    skillId,
    name: (entry.name || skillId).trim(),
    installs: Number(entry.installs) || 0,
    learnMoreUrl: id ? `${baseURL}/${id}` : undefined,
    originRegistryUrl: baseURL,
  };
}

function summaryFromBrowse(entry: BrowseEntry, baseURL: string): ExternalSkill {
  return {
    ...summaryFromSearch(entry, baseURL),
    installsYesterday: Number(entry.installsYesterday) || 0,
    change: Number(entry.change) || 0,
    weeklyInstalls: Array.isArray(entry.weeklyInstalls) ? entry.weeklyInstalls.map(Number) : undefined,
    isOfficial: Boolean(entry.isOfficial),
  };
}

async function fetchJSON<T>(url: string): Promise<T> {
  const response = await fetch(url, { headers: { accept: 'application/json' } });
  if (!response.ok) {
    throw new Error(`skills.sh request failed: ${response.status}`);
  }
  return (await response.json()) as T;
}

async function fetchSkillsShDetail(baseURL: string, source: string, skillId: string): Promise<ExternalSkillDetail> {
  const detailURL = detailURLFor(baseURL, source, skillId);
  const response = await fetch(detailURL, {
    headers: { accept: 'text/html,application/xhtml+xml' },
    redirect: 'follow',
  });
  if (!response.ok) {
    throw new Error(`skills.sh detail failed: ${response.status}`);
  }

  const detail = parseSkillsShDetail(await response.text(), source, skillId, baseURL);
  if (!detail.description && !detail.summaryHtml && !detail.contentHtml) {
    throw new Error('skills.sh skill not found');
  }

  const fullContent = await fetchFullContent(detailURL);
  if (fullContent) {
    detail.contentHtml = sanitizeSkillsShHTML(fullContent);
  }
  return detail;
}

export function parseSkillsShDetail(
  rawHTML: string,
  source: string,
  skillId: string,
  baseURL: string
): ExternalSkillDetail {
  const root = parse(rawHTML);
  const detailURL = detailURLFor(baseURL, source, skillId);
  const detail: ExternalSkillDetail = {
    id: `${source}/${skillId}`,
    source,
    skillId,
    name: skillId,
    installs: 0,
    learnMoreUrl: detailURL,
    originRegistryUrl: baseURL,
    url: detailURL,
  };

  const title = root.querySelector('title')?.text.trim() || '';
  if (title) {
    detail.name = title.split(' — ')[0].trim();
  }
  detail.description =
    metaContent(root, 'name', 'description') || metaContent(root, 'property', 'og:description') || '';
  detail.displayName = detail.name;

  const h1s = root.querySelectorAll('h1').map((element) => element.text.trim()).filter(Boolean);
  if (h1s.length > 0) {
    detail.name = h1s[0];
    if (h1s.length > 1) {
      const candidate = h1s[1];
      if (candidate !== detail.name && [...candidate].length <= 80 && !candidate.includes(' — ')) {
        detail.displayName = candidate;
      }
    }
  }

  for (const script of root.querySelectorAll('script[type="application/ld+json"]')) {
    try {
      const payload = JSON.parse(script.text) as Record<string, unknown>;
      if (payload['@type'] !== 'SoftwareApplication') continue;
      if (typeof payload.description === 'string' && payload.description.trim()) {
        detail.description = payload.description.trim();
      }
      const statistic = payload.interactionStatistic as Record<string, unknown> | undefined;
      const count = statistic?.userInteractionCount;
      if (typeof count === 'number') {
        detail.installs = count;
      }
    } catch {
      // Ignore malformed JSON-LD; the page metadata is the fallback.
    }
  }

  const installationHeading = findHeading(root.querySelectorAll('.uppercase'), 'Installation');
  const installationOffset = installationHeading?.range?.[0] ?? Number.POSITIVE_INFINITY;
  detail.topics = unique(
    root
      .querySelectorAll('a[href^="/topic/"]')
      .filter((element) => (element.range?.[0] ?? 0) < installationOffset)
      .map((element) => element.text.trim())
      .filter(Boolean)
  );

  const weeklyLabel = root.querySelector('svg[aria-label^="Weekly installs:"]')?.getAttribute('aria-label') || '';
  detail.weeklyInstalls = (weeklyLabel.match(/\d[\d,]*/g) || [])
    .map((value) => Number.parseInt(value.replaceAll(',', ''), 10))
    .filter((value) => Number.isFinite(value));
  detail.isOfficial = hasOfficialBadge(root);

  const commandName = COMMAND_NAME_PATTERN.test(detail.name) ? detail.name : skillId;
  detail.installCommand = `skills add ${source} --skill ${commandName}`;

  const prose = root.querySelectorAll('.prose');
  detail.summaryHtml = sanitizeSkillsShHTML(prose[0]?.innerHTML || '');
  detail.contentHtml = sanitizeSkillsShHTML(prose[1]?.innerHTML || prose[0]?.innerHTML || '');
  detail.relatedSkills = relatedSkills(root, baseURL);
  return detail;
}

function detailURLFor(baseURL: string, source: string, skillId: string): string {
  const sourcePath = source.split('/').map((part) => encodeURIComponent(part)).join('/');
  return `${baseURL}/${sourcePath}/${encodeURIComponent(skillId)}`;
}

function metaContent(root: HTMLElement, attribute: string, value: string): string {
  return root.querySelector(`meta[${attribute}="${value}"]`)?.getAttribute('content')?.trim() || '';
}

function findHeading(headings: HTMLElement[], wanted: string): HTMLElement | undefined {
  return headings.find((element) => headingText(element) === wanted);
}

function headingText(element: HTMLElement): string {
  const direct = element.childNodes
    .filter((node) => node.nodeType === NodeType.TEXT_NODE)
    .map((node) => node.text)
    .join('')
    .trim();
  return direct || element.querySelector('span')?.text.trim() || '';
}

function hasOfficialBadge(root: HTMLElement): boolean {
  return root.querySelectorAll('[aria-label]').some((element) => {
    const label = (element.getAttribute('aria-label') || '').trim().toLowerCase();
    return label === 'official' || label.includes('verified organization');
  });
}

function relatedSkills(root: HTMLElement, baseURL: string): ExternalSkill[] {
  const seen = new Set<string>();
  const related: ExternalSkill[] = [];
  for (const anchor of root.querySelectorAll('a[href]')) {
    const href = anchor.getAttribute('href') || '';
    if (!href.startsWith('/') || href.startsWith('/topic/')) continue;
    const parts = href.replace(/^\/+|\/+$/g, '').split('/');
    if (parts.length !== 3) continue;
    const heading = anchor.querySelector('h3');
    if (!heading) continue;
    const id = parts.join('/');
    if (seen.has(id)) continue;
    seen.add(id);
    related.push({
      id,
      source: `${parts[0]}/${parts[1]}`,
      skillId: parts[2],
      name: heading.text.trim(),
      installs: 0,
      learnMoreUrl: `${baseURL}/${id}`,
      originRegistryUrl: baseURL,
    });
    if (related.length >= 6) break;
  }
  return related;
}

function unique(values: string[]): string[] {
  return [...new Set(values)];
}

async function fetchFullContent(detailURL: string): Promise<string> {
  try {
    const response = await fetch(detailURL, {
      headers: { RSC: '1', accept: 'text/x-component' },
      redirect: 'follow',
    });
    if (!response.ok) return '';
    return fullContentFromRSC(await response.text());
  } catch {
    return '';
  }
}

export function fullContentFromRSC(body: string): string {
  const preview = extractRSCJSONString(body, 'previewHtml');
  if (!preview) return '';
  const rest = extractRSCJSONString(body, 'restHtml');
  if (rest.startsWith('$')) {
    return preview + extractRSCChunk(body, rest.slice(1));
  }
  return preview + rest;
}

function extractRSCJSONString(body: string, key: string): string {
  const marker = `"${key}":"`;
  const index = body.indexOf(marker);
  if (index < 0) return '';
  let value = '';
  let escaped = false;
  for (let i = index + marker.length; i < body.length; i += 1) {
    const character = body[i];
    if (escaped) {
      value += `\\${character}`;
      escaped = false;
      continue;
    }
    if (character === '\\') {
      escaped = true;
      continue;
    }
    if (character === '"') {
      try {
        return JSON.parse(`"${value}"`) as string;
      } catch {
        return '';
      }
    }
    value += character;
  }
  return '';
}

function extractRSCChunk(body: string, id: string): string {
  const marker = `${id}:T`;
  const index = body.indexOf(marker);
  if (index < 0) return '';
  const start = index + marker.length;
  const comma = body.indexOf(',', start);
  if (comma < 0) return '';

  let content = body.slice(comma + 1);
  const previewIndex = content.indexOf('"previewHtml":"');
  if (previewIndex > 0) {
    const beforePreview = content.slice(0, previewIndex);
    const rows = [...beforePreview.matchAll(/[0-9a-f]{1,3}:\["\$",/g)];
    if (rows.length > 0) {
      return beforePreview.slice(0, rows[rows.length - 1].index);
    }
  }

  const nextRow = content.search(/\n[0-9a-f]{1,3}:/);
  if (nextRow >= 0) {
    content = content.slice(0, nextRow);
  }
  return content;
}

export function sanitizeSkillsShHTML(raw: string): string {
  const trimmed = raw.trim();
  if (!trimmed) return '';
  const root = parse(trimmed, { comment: false });
  return root.childNodes.map((node) => serializeNode(node)).join('').trim();
}

function serializeNode(node: Node): string {
  if (node.nodeType === NodeType.TEXT_NODE) {
    return escapeHTML(node.text);
  }
  if (node.nodeType !== NodeType.ELEMENT_NODE) {
    return '';
  }

  const element = node as HTMLElement;
  const tag = (element.rawTagName || '').toLowerCase();
  if (BLOCKED_TAGS.has(tag)) return '';
  if (!ALLOWED_TAGS.has(tag)) {
    return element.childNodes.map((child) => serializeNode(child)).join('');
  }

  let attributes = '';
  if (tag === 'a') {
    const href = sanitizeHref(element.getAttribute('href'));
    if (href) {
      attributes = ` href="${escapeHTML(href)}" rel="noreferrer noopener" target="_blank"`;
    }
  }
  if (tag === 'th' || tag === 'td') {
    const colspan = numericAttribute(element, 'colspan');
    const rowspan = numericAttribute(element, 'rowspan');
    if (colspan) attributes += ` colspan="${colspan}"`;
    if (rowspan) attributes += ` rowspan="${rowspan}"`;
  }

  const children = element.childNodes.map((child) => serializeNode(child)).join('');
  if (VOID_TAGS.has(tag)) {
    return `<${tag}${attributes}>`;
  }
  return `<${tag}${attributes}>${children}</${tag}>`;
}

function sanitizeHref(raw: string | null | undefined): string {
  const href = (raw || '').trim();
  if (!href || href.startsWith('#')) return '';
  if (href.startsWith('/')) {
    return `https://www.skills.sh${href}`;
  }
  try {
    const parsed = new URL(href);
    return parsed.protocol === 'http:' || parsed.protocol === 'https:' ? href : '';
  } catch {
    return '';
  }
}

function numericAttribute(element: HTMLElement, name: string): string {
  const value = (element.getAttribute(name) || '').trim();
  return /^\d+$/.test(value) ? value : '';
}

function escapeHTML(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;');
}

async function cachedJSON(
  request: Request,
  cacheSeconds: number,
  ctx: SkillsShExecutionContext | undefined,
  loader: () => Promise<unknown>
): Promise<Response> {
  let cache: Cache | undefined;
  try {
    if (typeof caches !== 'undefined') cache = await caches.open('skills-sh');
  } catch {
    // A runtime without the Cache API should still serve live data.
  }
  const cacheKey = new Request(request.url, { method: 'GET' });
  if (cache) {
    try {
      const hit = await cache.match(cacheKey);
      if (hit) return hit;
    } catch {
      // A cache miss or a local runtime without the Cache API should not fail the request.
    }
  }

  const response = jsonResponse(await loader(), { cacheSeconds, status: 200 });
  if (cache && ctx) {
    ctx.waitUntil(cache.put(cacheKey, response.clone()).catch(() => undefined));
  }
  return response;
}

function jsonResponse(
  payload: unknown,
  options: { cacheSeconds: number; status: number }
): Response {
  return new Response(JSON.stringify(payload), {
    status: options.status,
    headers: {
      ...corsHeaders(),
      'content-type': 'application/json; charset=utf-8',
      'cache-control': options.cacheSeconds > 0 ? `public, max-age=${options.cacheSeconds}` : 'no-store',
    },
  });
}

function errorResponse(status: number, code: string, message: string): Response {
  return jsonResponse({ error: { code, message } }, { cacheSeconds: 0, status });
}

function corsHeaders(): Record<string, string> {
  return {
    'access-control-allow-origin': '*',
    'access-control-allow-methods': 'GET, HEAD, OPTIONS',
    'access-control-allow-headers': 'Content-Type',
    vary: 'Origin',
  };
}
