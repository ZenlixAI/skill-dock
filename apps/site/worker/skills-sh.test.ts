import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  fullContentFromRSC,
  handleSkillsShRequest,
  parseSkillsShDetail,
  sanitizeSkillsShHTML,
} from './skills-sh';

const detailHTML = `<!DOCTYPE html>
<html>
<head>
<title>vercel-react-best-practices — vercel-labs/agent-skills</title>
<meta name="description" content="Short description." />
</head>
<body>
<main>
<nav aria-label="Breadcrumb"><a href="/">skills</a><a href="/vercel-labs">vercel-labs</a><a href="/vercel-labs/agent-skills">agent-skills</a></nav>
<h1>vercel-react-best-practices</h1>
<div><a href="/topic/react">React</a><a href="/topic/nextjs">Next.js</a></div>
<div class="uppercase"><span>Installation</span><div><button>Command</button></div></div>
<code>$ npx skills add https://github.com/vercel-labs/agent-skills --skill vercel-react-best-practices</code>
<div class="uppercase">Summary</div>
<div class="prose prose-invert"><p><strong>Summary text</strong></p><ul><li><a href="/topic/react">topic link</a></li></ul></div>
<div class="uppercase">SKILL.md</div>
<div class="prose prose-invert"><h1>Vercel React Best Practices</h1><p>The <a href="/topic/react" onclick="alert(1)">guide</a> and <a href="javascript:alert(1)">bad link</a>.</p><img src="x" onerror="alert(1)" /><script>alert(1)</script></div>
<div class="uppercase">Related skills</div>
<section><a href="/owner/repo/related-skill"><div><h3>related-skill</h3><p>Related description</p></div></a></section>
<div>Installs</div>
<svg aria-label="Weekly installs: 19,251, 18,945, 21,814"></svg>
<svg aria-label="Verified organization on GitHub"></svg>
</main>
<script type="application/ld+json">{"@context":"https://schema.org","@type":"SoftwareApplication","name":"vercel-react-best-practices","description":"Full description from JSON-LD.","publisher":{"@type":"Organization","name":"vercel-labs"},"interactionStatistic":{"@type":"InteractionCounter","interactionType":"https://schema.org/InstallAction","userInteractionCount":886530}}</script>
</body>
</html>`;

const RSC_PAYLOAD =
  '1:T60,<p>RSC rest content</p>\n2:["$","div",null,{"previewHtml":"<p>RSC preview content</p>","restHtml":"$1","proseClassName":"prose"}]\n';

function jsonResponse(data: unknown): Response {
  return new Response(JSON.stringify(data), {
    status: 200,
    headers: { 'content-type': 'application/json' },
  });
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('skills.sh Worker parser', () => {
  it('parses the detail page and sanitizes embedded HTML', () => {
    const detail = parseSkillsShDetail(
      detailHTML,
      'vercel-labs/agent-skills',
      'vercel-react-best-practices',
      'https://skills.sh'
    );

    expect(detail.name).toBe('vercel-react-best-practices');
    expect(detail.displayName).toBe('Vercel React Best Practices');
    expect(detail.description).toBe('Full description from JSON-LD.');
    expect(detail.installs).toBe(886530);
    expect(detail.topics).toEqual(['React', 'Next.js']);
    expect(detail.weeklyInstalls).toEqual([19251, 18945, 21814]);
    expect(detail.isOfficial).toBe(true);
    expect(detail.installCommand).toBe(
      'skills add vercel-labs/agent-skills --skill vercel-react-best-practices'
    );
    expect(detail.summaryHtml).toContain('<strong>Summary text</strong>');
    expect(detail.contentHtml).toContain('href="https://www.skills.sh/topic/react"');
    expect(detail.contentHtml).not.toContain('onclick');
    expect(detail.contentHtml).not.toContain('javascript:');
    expect(detail.contentHtml).not.toContain('<img');
    expect(detail.contentHtml).not.toContain('onerror');
    expect(detail.relatedSkills).toHaveLength(1);
    expect(detail.relatedSkills?.[0].skillId).toBe('related-skill');
  });

  it('removes dangerous markup from arbitrary fragments', () => {
    const sanitized = sanitizeSkillsShHTML(
      '<p>ok</p><script>alert(1)</script><a href="javascript:alert(1)" onclick="x()">bad</a><iframe src="x"></iframe>'
    );
    expect(sanitized).toContain('<p>ok</p>');
    expect(sanitized).not.toContain('<script');
    expect(sanitized).not.toContain('javascript:');
    expect(sanitized).not.toContain('onclick');
    expect(sanitized).not.toContain('<iframe');
  });

  it('extracts the full SKILL.md from the RSC payload', () => {
    const content = fullContentFromRSC(RSC_PAYLOAD);
    expect(content).toContain('<p>RSC preview content</p>');
    expect(content).toContain('<p>RSC rest content</p>');
  });
});

describe('skills.sh Worker handler', () => {
  it('proxies search and browse', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes('/api/search')) {
        return jsonResponse({
          skills: [
            {
              id: 'vercel-labs/agent-skills/vercel-react-best-practices',
              source: 'vercel-labs/agent-skills',
              skillId: 'vercel-react-best-practices',
              name: 'vercel-react-best-practices',
              installs: 885584,
            },
          ],
        });
      }
      if (url.includes('/api/skills/all-time/1')) {
        return jsonResponse({
          skills: [
            { source: 'owner/repo', skillId: 'one', name: 'one', installs: 10, isOfficial: true },
          ],
          total: 42,
          hasMore: true,
          page: 1,
        });
      }
      throw new Error(`Unexpected request: ${url}`);
    });
    vi.stubGlobal('fetch', fetchMock);
    vi.stubGlobal('caches', undefined);

    const search = await handleSkillsShRequest(
      new Request('https://site.test/api/skills-sh/search?q=react&limit=2'),
      { SKILLS_SH_ENABLED: 'true' }
    );
    const searchPayload = (await search.json()) as { items: Array<{ name: string }> };
    expect(searchPayload.items[0].name).toBe('vercel-react-best-practices');

    const browse = await handleSkillsShRequest(
      new Request('https://site.test/api/skills-sh/browse?view=all-time&limit=1'),
      { SKILLS_SH_ENABLED: 'true' }
    );
    const browsePayload = (await browse.json()) as { items: unknown[]; total: number };
    expect(browsePayload.items).toHaveLength(1);
    expect(browsePayload.total).toBe(42);
  });

  it('proxies the detail page and uses the full RSC content', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (new Headers(init?.headers).get('RSC') === '1') {
        return new Response(RSC_PAYLOAD, { status: 200 });
      }
      return new Response(detailHTML, { status: 200, headers: { 'content-type': 'text/html' } });
    });
    vi.stubGlobal('fetch', fetchMock);
    vi.stubGlobal('caches', undefined);

    const response = await handleSkillsShRequest(
      new Request(
        'https://site.test/api/skills-sh/skill?source=vercel-labs/agent-skills&skillId=vercel-react-best-practices'
      ),
      { SKILLS_SH_ENABLED: 'true' }
    );
    const payload = (await response.json()) as { contentHtml: string; name: string };
    expect(payload.name).toBe('vercel-react-best-practices');
    expect(payload.contentHtml).toContain('RSC preview content');
    expect(payload.contentHtml).toContain('RSC rest content');
  });

  it('reports capabilities and disables data routes', async () => {
    vi.stubGlobal('caches', undefined);

    const capabilities = await handleSkillsShRequest(
      new Request('https://site.test/api/skills-sh/capabilities'),
      { SKILLS_SH_ENABLED: 'false' }
    );
    const payload = (await capabilities.json()) as { externalSearch: { enabled: boolean } };
    expect(payload.externalSearch.enabled).toBe(false);

    const search = await handleSkillsShRequest(
      new Request('https://site.test/api/skills-sh/search?q=react'),
      { SKILLS_SH_ENABLED: 'false' }
    );
    expect(search.status).toBe(404);
  });
});
