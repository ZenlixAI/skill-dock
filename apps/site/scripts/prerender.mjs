import { mkdir, writeFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { chromium } from 'playwright';
import { preview } from 'vite';

const port = 4176;
const routes = [
  { path: '/', file: 'index.html' },
  { path: '/registry?prerender=1', file: 'registry.html' },
  { path: '/privacy', file: 'privacy.html' },
  { path: '/terms', file: 'terms.html' },
  { path: '/zh', file: 'zh/index.html' },
  { path: '/zh/registry?prerender=1', file: 'zh/registry.html' },
  { path: '/zh/privacy', file: 'zh/privacy.html' },
  { path: '/zh/terms', file: 'zh/terms.html' },
];

const server = await preview({
  preview: { host: '127.0.0.1', port, strictPort: true },
});
const browser = await chromium.launch();

try {
  const page = await browser.newPage({
    colorScheme: 'dark',
    reducedMotion: 'reduce',
    viewport: { width: 1440, height: 900 },
  });

  for (const route of routes) {
    const url = `http://127.0.0.1:${port}${route.path}`;
    await page.goto(url, { waitUntil: 'networkidle' });
    await page.waitForSelector('main h1');
    await page.evaluate(() => document.fonts.ready);

    const html = await page.content();
    if (!html.includes('<main') || html.length < 2_000) {
      throw new Error(`Prerender for ${route.path} produced an incomplete document.`);
    }

    const target = join('dist', route.file);
    await mkdir(dirname(target), { recursive: true });
    await writeFile(target, html, 'utf8');
    console.log(`prerendered ${route.path} -> dist/${route.file}`);
  }
} finally {
  await browser.close();
  await server.close();
}
