import { cleanup, render, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { LocaleProvider } from '../lib/i18n';
import { DocumentMeta } from './DocumentMeta';

let initialHead = '';

beforeEach(() => {
  initialHead = document.head.innerHTML;
});

afterEach(() => {
  cleanup();
  document.head.innerHTML = initialHead;
  document.documentElement.lang = 'en';
});

describe('DocumentMeta', () => {
  it('localizes the document language, canonical URL, and hreflang alternates', async () => {
    render(
      <MemoryRouter initialEntries={['/zh/privacy']}>
        <LocaleProvider>
          <DocumentMeta
            description="Skill Dock 网站如何处理站点使用数据。"
            title="隐私 · Skill Dock"
          />
        </LocaleProvider>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(document.documentElement.lang).toBe('zh-CN');
      expect(document.title).toBe('隐私 · Skill Dock');
    });

    const canonical = document.head.querySelector<HTMLLinkElement>('link[rel="canonical"]');
    expect(canonical?.href).toBe('https://skill-dock.pages.dev/zh/privacy');

    const english = document.head.querySelector<HTMLLinkElement>(
      'link[rel="alternate"][hreflang="en"]'
    );
    const chinese = document.head.querySelector<HTMLLinkElement>(
      'link[rel="alternate"][hreflang="zh-CN"]'
    );
    expect(english?.href).toBe('https://skill-dock.pages.dev/privacy');
    expect(chinese?.href).toBe('https://skill-dock.pages.dev/zh/privacy');

    const locale = document.head.querySelector<HTMLMetaElement>('meta[property="og:locale"]');
    expect(locale?.content).toBe('zh_CN');
  });
});
